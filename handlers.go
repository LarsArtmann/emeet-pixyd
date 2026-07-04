//go:build linux

package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"slices"
	"strconv"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"github.com/a-h/templ"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	maxStreamBufferSize = 10 * 1024 * 1024
	maxBodyBytes        = 1 << 10

	staticCacheMaxAge = 7 * 24 * time.Hour

	sseEventConnected = "connected"
	sseEventRefresh   = "refresh"

	toastTrackingEnabled = "Tracking enabled"
	toastCameraIdle      = "Camera idle"
	toastPrivacyOn       = "Privacy mode on"
	toastCameraCentered  = "Camera centered"
	toastStateSynced     = "State synced"
	toastProbedDevices   = "Probed devices"
	toastAudioChanged    = "Audio mode changed"
	toastGestureToggled  = "Gesture toggled"
	toastAutoToggled     = "Auto mode toggled"
)

// toastType is the CSS class suffix for toast notifications.
type actionToastInfo struct {
	msg  string
	kind toastType
}

//nolint:gochecknoglobals
var actionToasts = map[string]actionToastInfo{
	cmdTrack:         {toastTrackingEnabled, toastTypeSuccess},
	cmdIdle:          {toastCameraIdle, toastTypeSuccess},
	cmdPrivacy:       {toastPrivacyOn, toastTypeSuccess},
	cmdCenter:        {toastCameraCentered, toastTypeSuccess},
	cmdSync:          {toastStateSynced, toastTypeSuccess},
	cmdProbe:         {toastProbedDevices, toastTypeSuccess},
	cmdToggleGesture: {toastGestureToggled, toastTypeInfo},
	cmdToggleAuto:    {toastAutoToggled, toastTypeInfo},
}

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
	//nolint:exhaustruct
	status := webStatus{
		PTZValues:  pixy.PTZValues{},
		Camera:     s.daemon.state.Camera,
		Audio:      s.daemon.state.Audio,
		Gesture:    s.daemon.state.Gesture,
		InCall:     s.daemon.state.InCall,
		Auto:       s.daemon.state.AutoMode,
		Online:     s.daemon.videoDev != "",
		Device:     s.daemon.videoDev,
		Error:      errStr(s.daemon.autoError),
		LastSynced: formatLastSynced(s.daemon.lastSyncedAt),
		Version:    buildVersion,
	}
	if status.Online {
		status.Zoom = pixy.ZoomDefault
	}

	presetNames := make([]string, 0, len(s.daemon.state.Presets))
	for name := range s.daemon.state.Presets {
		presetNames = append(presetNames, name)
	}

	slices.Sort(presetNames)
	status.PresetNames = presetNames

	return status
}

func (s *webServer) getWebStatusWithPTZ(ctx context.Context) webStatus {
	status := s.getWebStatus()
	if !status.Online {
		return status
	}

	dev := status.Device

	if values, valid := s.daemon.ptzCache.Get(); valid {
		status.Pan = values.Pan
		status.Tilt = values.Tilt
		status.Zoom = values.Zoom

		return status
	}

	ptz := s.daemon.deps.parsePTZ(ctx, dev)
	s.daemon.ptzCache.Set(ptz, ptzCacheTTL)

	status.Pan = ptz.Pan
	status.Tilt = ptz.Tilt
	status.Zoom = ptz.Zoom

	return status
}

func (s *webServer) handleIndex(responseWriter http.ResponseWriter, request *http.Request) {
	status := s.getWebStatusWithPTZ(request.Context())
	templ.Handler(page(status)).ServeHTTP(responseWriter, request) //nolint:contextcheck
}

func (s *webServer) handleHealth(responseWriter http.ResponseWriter, _ *http.Request) {
	s.daemon.mu.RLock()
	online := s.daemon.videoDev != ""
	camera := s.daemon.state.Camera
	s.daemon.mu.RUnlock()

	status := http.StatusOK
	if !online {
		status = http.StatusServiceUnavailable
	}

	_ = writeJSON(responseWriter, status, healthResponse{
		Status:  boolStr(online, "ok", "offline"),
		Camera:  camera,
		Version: buildVersion,
	})
}

type healthResponse struct {
	Status  string           `json:"status"`
	Camera  pixy.CameraState `json:"camera"`
	Version string           `json:"version"`
}

func (s *webServer) handleStatusPanel(responseWriter http.ResponseWriter, request *http.Request) {
	status := s.getWebStatusWithPTZ(request.Context())
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request) //nolint:contextcheck
}

func (s *webServer) handleEvents(responseWriter http.ResponseWriter, request *http.Request) {
	stream := newSSEStream(responseWriter, request)
	defer stream.Close()

	_ = stream.Send(SSEEvent{ //nolint:exhaustruct
		Event: sseEventConnected,
		Data:  "{}",
	})

	ch := s.daemon.broadcaster.Subscribe()
	defer s.daemon.broadcaster.Unsubscribe(ch)

	for {
		select {
		case <-stream.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}

			sendErr := stream.Send(event)
			if sendErr != nil {
				return
			}
		}
	}
}

func (s *webServer) action(command string) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)

		result := s.daemon.handleCommand(request.Context(), command)

		slog.Debug("web action", "cmd", command, "response", result.String())

		status := s.getWebStatusWithPTZ(request.Context())
		toast, toastType := actionToast(command)
		applyResultToStatus(result, &status, toast, toastType)

		templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request) //nolint:contextcheck
	}
}

func actionToast(command string) (string, toastType) {
	info, ok := actionToasts[command]
	if !ok {
		return "", ""
	}

	return info.msg, info.kind
}

func applyResultToStatus(result CommandResult, status *webStatus, toast string, tt toastType) {
	if result.IsError() {
		status.Error = result.String()
	} else {
		status.Toast = toast
		status.ToastType = tt
	}
}

func (s *webServer) handleAudio(responseWriter http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)
	mode := request.FormValue("mode")

	cmd := cmdAudio
	if mode != "" {
		cmd = cmdAudio + " " + mode
	}

	result := s.daemon.handleCommand(request.Context(), cmd)
	slog.Debug("web audio", "cmd", cmd, "response", result.String())

	status := s.getWebStatusWithPTZ(request.Context())

	toast := toastAudioChanged
	if !result.IsError() {
		toast = "Audio: " + string(status.Audio)
	}

	applyResultToStatus(result, &status, toast, toastTypeSuccess)
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request) //nolint:contextcheck
}

func (s *webServer) handlePTZ(responseWriter http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)
	axis := pixy.Axis(request.PathValue("axis"))

	val := request.FormValue("value")
	if string(axis) == "" || val == "" {
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

	info := ptzAxes[axis]
	intVal = clampInt(intVal, info.Min, info.Max)
	result := s.daemon.handleCommand(request.Context(), string(axis)+" "+strconv.Itoa(intVal))
	slog.Debug("web ptz", "axis", axis, "val", intVal, "response", result.String())

	if result.IsError() {
		status := s.getWebStatusWithPTZ(request.Context())
		sliderVal, _ := ptzAxisValue(axis, status)
		templ.Handler(ptzSliderWithToast( //nolint:contextcheck
			info.Label, string(axis), info.Min, info.Max, sliderVal, info.Unit,
			result.String(), string(toastTypeError),
		)).ServeHTTP(responseWriter, request)

		return
	}

	templ.Handler(ptzSliderWithToast( //nolint:contextcheck
		info.Label, string(axis), info.Min, info.Max, intVal, info.Unit,
		"", "",
	)).ServeHTTP(responseWriter, request)
}

func (s *webServer) handlePresetSave(responseWriter http.ResponseWriter, request *http.Request) {
	name := request.PathValue("name")
	err := pixy.ValidatePresetName(name)
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusBadRequest)

		return
	}

	result := s.daemon.handleCommand(request.Context(), cmdPreset+" save "+name)

	status := s.getWebStatusWithPTZ(request.Context())
	applyResultToStatus(result, &status, result.String(), toastTypeSuccess)
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request) //nolint:contextcheck
}

func (s *webServer) handlePresetLoad(responseWriter http.ResponseWriter, request *http.Request) {
	name := request.PathValue("name")
	if name == "" {
		http.Error(responseWriter, "missing preset name", http.StatusBadRequest)

		return
	}

	result := s.daemon.handleCommand(request.Context(), cmdPreset+" load "+name)

	status := s.getWebStatusWithPTZ(request.Context())
	applyResultToStatus(result, &status, result.String(), toastTypeSuccess)
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request) //nolint:contextcheck
}

func (s *webServer) handlePresetDelete(responseWriter http.ResponseWriter, request *http.Request) {
	name := request.PathValue("name")
	if name == "" {
		http.Error(responseWriter, "missing preset name", http.StatusBadRequest)

		return
	}

	result := s.daemon.handleCommand(request.Context(), cmdPreset+" delete "+name)

	status := s.getWebStatusWithPTZ(request.Context())
	applyResultToStatus(result, &status, result.String(), toastTypeSuccess)
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request) //nolint:contextcheck
}

func newWebMux(server *webServer) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", cachingFS{handler: http.FileServer(http.FS(staticFS))})
	mux.HandleFunc("GET /{$}", server.handleIndex)
	mux.HandleFunc("GET /panel", server.handleStatusPanel)
	mux.HandleFunc("GET /api/events", server.handleEvents)
	mux.HandleFunc("GET /api/health", server.handleHealth)
	mux.HandleFunc("POST /api/track", server.action(cmdTrack))
	mux.HandleFunc("POST /api/"+cmdIdle, server.action(cmdIdle))
	mux.HandleFunc("POST /api/privacy", server.action(cmdPrivacy))
	mux.HandleFunc("POST /api/toggle-privacy", server.action(cmdTogglePrivacy))
	mux.HandleFunc("POST /api/audio", server.handleAudio)
	mux.HandleFunc("POST /api/gesture", server.action(cmdToggleGesture))
	mux.HandleFunc("POST /api/auto", server.action(cmdToggleAuto))
	mux.HandleFunc("POST /api/center", server.action(cmdCenter))
	mux.HandleFunc("POST /api/sync", server.action(cmdSync))
	mux.HandleFunc("POST /api/probe", server.action(cmdProbe))
	mux.HandleFunc("POST /api/preset/save/{name}", server.handlePresetSave)
	mux.HandleFunc("POST /api/preset/load/{name}", server.handlePresetLoad)
	mux.HandleFunc("POST /api/preset/delete/{name}", server.handlePresetDelete)
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
