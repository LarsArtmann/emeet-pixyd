//go:build linux

package main

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"github.com/a-h/templ"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	audioCommand        = "audio"
	zoomDefault         = 100
	maxStreamBufferSize = 10 * 1024 * 1024
	maxBodyBytes        = 1 << 10

	panMin  = -170
	panMax  = 170
	tiltMin = -30
	tiltMax = 30
	zoomMin = 100
	zoomMax = 400

	staticCacheMaxAge = 7 * 24 * time.Hour
	ffmpegShutdown    = 2 * time.Second
	streamBufSize     = 64 * 1024

	toastTypeSuccess = "success"
	toastTypeInfo    = "info"

	ptzCacheTTL = 2 * time.Second
)

//go:embed static
var staticFS embed.FS

var (
	promExporter      *prometheus.Exporter
	metricInCall      metric.Float64Gauge
	metricAutoMode    metric.Float64Gauge
	metricCameraState metric.Float64Gauge
)

var metricsOnce sync.Once

func init() {
	registerMetrics()
}

func registerMetrics() {
	metricsOnce.Do(func() {
		var err error
		promExporter, err = prometheus.New(
			prometheus.WithoutScopeInfo(),
			prometheus.WithoutTargetInfo(),
		)
		if err != nil {
			slog.Error("failed to create OTel Prometheus exporter", "error", err)
			return
		}
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(promExporter))
		meter := mp.Meter("emeet-pixyd")
		if metricInCall, err = meter.Float64Gauge("emeet_pixyd_in_call",
			metric.WithDescription("Whether the camera is currently in a call (1=yes, 0=no)"),
		); err != nil {
			slog.Error("failed to create in_call gauge", "error", err)
		}
		if metricAutoMode, err = meter.Float64Gauge("emeet_pixyd_auto_mode",
			metric.WithDescription("Whether auto-management mode is enabled (1=yes, 0=no)"),
		); err != nil {
			slog.Error("failed to create auto_mode gauge", "error", err)
		}
		if metricCameraState, err = meter.Float64Gauge("emeet_pixyd_camera_state",
			metric.WithDescription("Current camera state as a gauge per state label (1=active)"),
		); err != nil {
			slog.Error("failed to create camera_state gauge", "error", err)
		}
	})
}

func updateMetrics(state pixy.State) {
	ctx := context.Background()
	if state.InCall {
		metricInCall.Record(ctx, 1)
	} else {
		metricInCall.Record(ctx, 0)
	}
	if state.AutoMode {
		metricAutoMode.Record(ctx, 1)
	} else {
		metricAutoMode.Record(ctx, 0)
	}
	for _, s := range []pixy.CameraState{pixy.StatePrivacy, pixy.StateTracking, pixy.StateIdle} {
		stateAttr := metric.WithAttributes(attribute.String("state", string(s)))
		if state.Camera == s {
			metricCameraState.Record(ctx, 1, stateAttr)
		} else {
			metricCameraState.Record(ctx, 0, stateAttr)
		}
	}
}

func formatLastSynced(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	elapsed := time.Since(t)
	if elapsed < time.Minute {
		return "just now"
	}
	if elapsed < time.Hour {
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	}

	return t.Format("15:04")
}

type webServer struct {
	daemon *Daemon
}

func (s *webServer) getWebStatus() webStatus {
	s.daemon.mu.RLock()
	defer s.daemon.mu.RUnlock()
	status := webStatus{
		Camera:     string(s.daemon.state.Camera),
		Audio:      string(s.daemon.state.Audio),
		Gesture:    s.daemon.state.Gesture,
		Pan:        0,
		Tilt:       0,
		Zoom:       0,
		InCall:     s.daemon.state.InCall,
		Auto:       s.daemon.state.AutoMode,
		Online:     s.daemon.videoDev != "",
		Device:     s.daemon.videoDev,
		LastSynced: formatLastSynced(s.daemon.lastSyncedAt),
	}
	if status.Online {
		status.Zoom = zoomDefault
	}

	return status
}

func (s *webServer) getWebStatusWithPTZ(ctx context.Context) webStatus {
	status := s.getWebStatus()
	if !status.Online {
		return status
	}
	dev := status.Device

	now := time.Now()
	s.daemon.ptzCache.mu.RLock()
	if now.Before(s.daemon.ptzCache.expiresAt) {
		status.Pan = s.daemon.ptzCache.values.Pan
		status.Tilt = s.daemon.ptzCache.values.Tilt
		status.Zoom = s.daemon.ptzCache.values.Zoom
		s.daemon.ptzCache.mu.RUnlock()

		return status
	}
	s.daemon.ptzCache.mu.RUnlock()

	ptz := parsePTZValues(ctx, dev)
	s.daemon.ptzCache.mu.Lock()
	s.daemon.ptzCache.values = ptz
	s.daemon.ptzCache.expiresAt = now.Add(ptzCacheTTL)
	s.daemon.ptzCache.mu.Unlock()

	status.Pan = ptz.Pan
	status.Tilt = ptz.Tilt
	status.Zoom = ptz.Zoom

	return status
}

func (s *webServer) handleIndex(responseWriter http.ResponseWriter, request *http.Request) {
	status := s.getWebStatusWithPTZ(request.Context())
	templ.Handler(page(status)).ServeHTTP(responseWriter, request)
}

func (s *webServer) handleStatusPanel(responseWriter http.ResponseWriter, request *http.Request) {
	status := s.getWebStatusWithPTZ(request.Context())
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request)
}

func (s *webServer) action(command string) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)

		resp := s.daemon.handleCommand(request.Context(), command)

		slog.Debug("web action", "cmd", command, "response", resp)

		status := s.getWebStatusWithPTZ(request.Context())
		if strings.HasPrefix(resp, "error:") {
			status.Error = resp
		} else {
			status.Toast, status.ToastType = actionToast(command)
		}

		templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request)
	}
}

func actionToast(command string) (string, string) {
	switch command {
	case "track":
		return "Tracking enabled", toastTypeSuccess
	case cmdIdle:
		return "Camera idle", toastTypeSuccess
	case cmdPrivacy:
		return "Privacy mode on", toastTypeSuccess
	case "center":
		return "Camera centered", toastTypeSuccess
	case "sync":
		return "State synced", toastTypeSuccess
	case "probe":
		return "Probed devices", toastTypeSuccess
	case cmdToggleGesture:
		return "Gesture toggled", toastTypeInfo
	case cmdToggleAuto:
		return "Auto mode toggled", toastTypeInfo
	default:
		return "", ""
	}
}

func applyResponseToStatus(resp string, status *webStatus, toast string) {
	if strings.HasPrefix(resp, "error:") {
		status.Error = resp
	} else {
		status.Toast = toast
		status.ToastType = toastTypeInfo
	}
}

func (s *webServer) handleAudio(responseWriter http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)
	mode := request.FormValue("mode")
	cmd := audioCommand
	if mode != "" {
		cmd = audioCommand + " " + mode
	}
	resp := s.daemon.handleCommand(request.Context(), cmd)
	slog.Debug("web audio", "cmd", cmd, "response", resp)
	status := s.getWebStatusWithPTZ(request.Context())
	applyResponseToStatus(resp, &status, "Audio mode changed")
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request)
}

func clampInt(v, lo, hi int) int {
	return max(lo, min(hi, v))
}

func ptzLimits(axis string) (int, int) {
	switch axis {
	case axisPan:
		return panMin, panMax
	case axisTilt:
		return tiltMin, tiltMax
	case axisZoom:
		return zoomMin, zoomMax
	default:
		return 0, 0
	}
}

func (s *webServer) handlePTZ(responseWriter http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)
	axis := request.PathValue("axis")
	val := request.FormValue("value")
	if axis == "" || val == "" {

		http.Error(responseWriter, "missing axis or value", http.StatusBadRequest)

		return
	}
	if !ptzAxisValid(axis) {
		http.Error(responseWriter, "invalid axis", http.StatusBadRequest)
		return
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		http.Error(responseWriter, "invalid value", http.StatusBadRequest)
		return
	}
	lo, hi := ptzLimits(axis)
	intVal = clampInt(intVal, lo, hi)
	resp := s.daemon.handleCommand(request.Context(), axis+" "+strconv.Itoa(intVal))
	slog.Debug("web ptz", "axis", axis, "val", intVal, "response", resp)
	status := s.getWebStatusWithPTZ(request.Context())
	switch axis {
	case axisPan:

		templ.Handler(ptzSlider("Pan", axisPan, panMin, panMax, status.Pan, "\u00b0")).
			ServeHTTP(responseWriter, request)
	case axisTilt:

		templ.Handler(ptzSlider("Tilt", axisTilt, tiltMin, tiltMax, status.Tilt, "\u00b0")).
			ServeHTTP(responseWriter, request)
	case axisZoom:

		templ.Handler(ptzSlider("Zoom", axisZoom, zoomMin, zoomMax, status.Zoom, "x")).
			ServeHTTP(responseWriter, request)
	default:

		templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request)
	}
}

func (s *webServer) checkDevice(responseWriter http.ResponseWriter) (webStatus, bool) {
	status := s.getWebStatus()
	if status.Device == "" {

		http.Error(responseWriter, "no camera device", http.StatusServiceUnavailable)

		return status, false
	}
	return status, true
}

func (s *webServer) handleSnapshot(responseWriter http.ResponseWriter, _ *http.Request) {
	s.daemon.lastFrame.RLock()
	frame := s.daemon.lastFrame.data
	s.daemon.lastFrame.RUnlock()

	if len(frame) == 0 {
		http.Error(responseWriter, "no frame available", http.StatusServiceUnavailable)

		return
	}

	responseWriter.Header().Set("Content-Type", "image/jpeg")
	responseWriter.Header().Set("Cache-Control", "no-store")
	_, _ = responseWriter.Write(frame)
}

func ffmpegStreamCmd(ctx context.Context, device string) *exec.Cmd {
	return exec.CommandContext(ctx,
		"ffmpeg",
		"-f", "v4l2",
		"-input_format", "mjpeg",
		"-i", device,
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-q:v", "5",
		"-vf", "scale=640:-1",
		"pipe:1",
	)
}

func cleanupFFmpeg(cmd *exec.Cmd) {
	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(ffmpegShutdown):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func (s *webServer) handleStream(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	select {
	case s.daemon.streamSema <- struct{}{}:
	default:
		http.Error(responseWriter, "stream already in use", http.StatusServiceUnavailable)

		return
	}
	defer func() { <-s.daemon.streamSema }()

	status, ok := s.checkDevice(responseWriter)
	if !ok {
		return
	}
	if _, lookErr := exec.LookPath("ffmpeg"); lookErr != nil {
		http.Error(responseWriter, "ffmpeg not available", http.StatusServiceUnavailable)

		return
	}
	flusher, flushOk := responseWriter.(http.Flusher)
	if !flushOk {

		http.Error(responseWriter, "streaming not supported", http.StatusInternalServerError)

		return
	}
	ctx := request.Context()
	cmd := ffmpegStreamCmd(ctx, status.Device)
	stdOut, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		http.Error(responseWriter, "stream pipe error", http.StatusInternalServerError)

		return
	}
	startErr := cmd.Start()
	if startErr != nil {
		http.Error(responseWriter, "stream start error", http.StatusInternalServerError)

		return
	}
	responseWriter.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	responseWriter.Header().Set("Cache-Control", "no-store")
	defer cleanupFFmpeg(cmd)
	br := bufio.NewReaderSize(stdOut, streamBufSize)
	var buf bytes.Buffer
	for {

		select {

		case <-ctx.Done():

			return

		default:

		}

		frame, frameErr := extractJPEGFrame(br, &buf)

		if frameErr != nil {

			slog.Debug("frame extract error", "error", frameErr)

			return

		}

		s.daemon.lastFrame.Lock()
		s.daemon.lastFrame.data = frame
		s.daemon.lastFrame.Unlock()

		_, headerErr := fmt.Fprintf(

			responseWriter,

			"--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n",

			len(frame),
		)

		if headerErr != nil {
			return
		}

		_, writeErr := responseWriter.Write(frame)

		if writeErr != nil {
			return
		}

		_, sepErr := fmt.Fprint(responseWriter, "\r\n")

		if sepErr != nil {
			return
		}

		flusher.Flush()
	}
}

func extractJPEGFrame(br *bufio.Reader, buf *bytes.Buffer) ([]byte, error) {
	var soiFound bool
	for {
		if buf.Len() > maxStreamBufferSize {
			buf.Reset()
		}

		b, err := br.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("read byte: %w", err)
		}

		if !soiFound {
			if b == 0xFF {
				next, nextErr := br.ReadByte()
				if nextErr != nil {
					return nil, fmt.Errorf("read soi next: %w", nextErr)
				}
				switch next {
				case 0xD8:
					buf.Reset()
					buf.Write([]byte{0xFF, 0xD8})
					soiFound = true
				case 0xFF:
					_ = br.UnreadByte()
				}
			}
			continue
		}

		buf.WriteByte(b)

		if b == 0xFF {
			next, nextErr := br.ReadByte()
			if nextErr != nil {
				return nil, fmt.Errorf("read eoi next: %w", nextErr)
			}
			buf.WriteByte(next)
			if next == 0xD9 {
				frame := make([]byte, buf.Len())
				copy(frame, buf.Bytes())
				return frame, nil
			}
		}
	}
}

func (s *webServer) handleGestureToggle(responseWriter http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)
	resp := s.daemon.handleCommand(request.Context(), cmdToggleGesture)
	slog.Debug("web gesture toggle", "response", resp)
	status := s.getWebStatusWithPTZ(request.Context())
	applyResponseToStatus(resp, &status, "Gesture toggled")
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request)
}

func (s *webServer) handleAutoToggle(responseWriter http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)
	resp := s.daemon.handleCommand(request.Context(), cmdToggleAuto)
	slog.Debug("web auto toggle", "response", resp)
	status := s.getWebStatusWithPTZ(request.Context())
	applyResponseToStatus(resp, &status, "Auto mode toggled")
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request)
}

type cachingFS struct {
	handler http.Handler
}

func (c cachingFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().
		Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int64(staticCacheMaxAge.Seconds())))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	c.handler.ServeHTTP(w, r)
}

func ptzAxisValid(axis string) bool {
	switch axis {
	case axisPan, axisTilt, axisZoom:
		return true
	default:
		return false
	}
}

func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().
			Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

func newWebMux(server *webServer) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", cachingFS{handler: http.FileServer(http.FS(staticFS))})
	mux.HandleFunc("GET /{$}", server.handleIndex)
	mux.HandleFunc("GET /panel", server.handleStatusPanel)
	mux.HandleFunc("POST /api/track", server.action("track"))
	mux.HandleFunc("POST /api/"+cmdIdle, server.action(cmdIdle))
	mux.HandleFunc("POST /api/privacy", server.action(cmdPrivacy))
	mux.HandleFunc("POST /api/toggle-privacy", server.action("toggle-privacy"))
	mux.HandleFunc("POST /api/audio", server.handleAudio)
	mux.HandleFunc("POST /api/gesture", server.handleGestureToggle)
	mux.HandleFunc("POST /api/auto", server.handleAutoToggle)
	mux.HandleFunc("POST /api/center", server.action("center"))
	mux.HandleFunc("POST /api/sync", server.action("sync"))
	mux.HandleFunc("POST /api/probe", server.action("probe"))
	mux.HandleFunc("POST /api/ptz/{axis}", server.handlePTZ)
	mux.HandleFunc("POST /api/ptz/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing axis", http.StatusBadRequest)
	})
	mux.HandleFunc("GET /api/snapshot", server.handleSnapshot)
	mux.HandleFunc("GET /api/stream", server.handleStream)
	mux.Handle("GET /metrics", promhttp.Handler())
	if server.daemon.config.Debug {
		mux.HandleFunc("GET /debug/pprof/", pprof.Index)
		mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	}
	return mux
}
