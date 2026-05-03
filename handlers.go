//go:build linux

package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
		Camera:     s.daemon.state.Camera,
		Audio:      s.daemon.state.Audio,
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
		toast, _ := actionToast(command)
		applyResponseToStatus(resp, &status, toast)

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
	if IsCommandErrorResponse(resp) {
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
