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

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"github.com/a-h/templ"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/starfederation/datastar-go/datastar"
)

const (
	maxStreamBufferSize = 10 * 1024 * 1024
	maxBodyBytes        = 1 << 10

	staticCacheMaxAge = 7 * 24 * time.Hour

	sseEventRefresh = "refresh"

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
		status.PTZValues = pixy.PTZValues{Pan: 0, Tilt: 0, Zoom: pixy.ZoomDefault}
	}

	status.PresetNames = s.daemon.state.Presets.SortedNames()

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

// handleStatusPanel renders the panel as plain HTML for testing and direct access.
// The DataStar UI receives panel updates via SSE patches from handleEvents and
// action handlers.
func (s *webServer) handleStatusPanel(responseWriter http.ResponseWriter, request *http.Request) {
	status := s.getWebStatusWithPTZ(request.Context())
	templ.Handler(statusPanel(status)).ServeHTTP(responseWriter, request) //nolint:contextcheck
}

// handleEvents is the persistent SSE connection. DataStar establishes this via
// data-init="@get('/api/events', {openWhenHidden: true})". It sends the current
// panel on connect, then patches on every state-change broadcast.
func (s *webServer) handleEvents(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)

	// Send initial panel state
	status := s.getWebStatusWithPTZ(request.Context())
	if err := sse.PatchElementTempl(statusPanel(status)); err != nil { //nolint:contextcheck
		return
	}

	ch := s.daemon.broadcaster.Subscribe()
	defer s.daemon.broadcaster.Unsubscribe(ch)

	for {
		select {
		case <-sse.Context().Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}

			refreshed := s.getWebStatusWithPTZ(request.Context())
			if err := sse.PatchElementTempl(statusPanel(refreshed)); err != nil { //nolint:contextcheck
				return
			}
		}
	}
}

// patchPanel renders the status panel as a DataStar SSE element patch.
func (s *webServer) patchPanel(sse *datastar.ServerSentEventGenerator, status webStatus) {
	_ = sse.PatchElementTempl(statusPanel(status))
	sendToastScript(sse, &status)
}

// sendToastScript dispatches a toast via ExecuteScript so the client-side
// auto-dismiss logic handles fading. strconv.Quote produces a safe JS string literal.
func sendToastScript(sse *datastar.ServerSentEventGenerator, status *webStatus) {
	msg := status.Toast

	tt := status.ToastType
	if status.Error != "" {
		msg = status.Error
		tt = toastTypeError
	}

	if msg == "" {
		return
	}

	script := fmt.Sprintf("window.__showToast(%s, %s)", strconv.Quote(msg), strconv.Quote(string(tt)))
	_ = sse.ExecuteScript(script)
}

func (s *webServer) action(command string) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)

		result := s.daemon.handleCommand(request.Context(), command)

		slog.Debug("web action", "cmd", command, "response", result.String())

		status := s.getWebStatusWithPTZ(request.Context())
		toast, toastType := actionToast(command)
		applyResultToStatus(result, &status, toast, toastType)

		sse := datastar.NewSSE(responseWriter, request)
		s.patchPanel(sse, status) //nolint:contextcheck // templ rendering handles context internally
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
	mode := request.PathValue("mode")

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

	sse := datastar.NewSSE(responseWriter, request)
	s.patchPanel(sse, status) //nolint:contextcheck // templ rendering handles context internally
}

func (s *webServer) handlePTZ(responseWriter http.ResponseWriter, request *http.Request) {
	axis := pixy.Axis(request.PathValue("axis"))

	if string(axis) == "" || !ptzAxisValid(axis) {
		http.Error(responseWriter, "invalid axis", http.StatusBadRequest)

		return
	}

	var signals pixy.PTZValues
	if err := datastar.ReadSignals(request, &signals); err != nil {
		http.Error(responseWriter, "invalid signals", http.StatusBadRequest)

		return
	}

	val, _ := signals.Get(axis)

	info := ptzAxes[axis]
	intVal := info.Range.Clamp(val)
	result := s.daemon.handleCommand(request.Context(), string(axis)+" "+strconv.Itoa(intVal))
	slog.Debug("web ptz", "axis", axis, "val", intVal, "response", result.String())

	status := s.getWebStatusWithPTZ(request.Context())
	applyResultToStatus(result, &status, "", "")

	sse := datastar.NewSSE(responseWriter, request)
	s.patchPanel(sse, status) //nolint:contextcheck // templ rendering handles context internally
}

func (s *webServer) handlePresetSave(responseWriter http.ResponseWriter, request *http.Request) {
	name := request.PathValue("name")

	err := pixy.ValidatePresetName(name)
	if err != nil {
		http.Error(responseWriter, err.Error(), errorfamily.HTTPStatus(err))

		return
	}

	result := s.daemon.handleCommand(request.Context(), cmdPreset+" save "+name)

	status := s.getWebStatusWithPTZ(request.Context())
	applyResultToStatus(result, &status, result.String(), toastTypeSuccess)

	sse := datastar.NewSSE(responseWriter, request)
	s.patchPanel(sse, status) //nolint:contextcheck // templ rendering handles context internally
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

	sse := datastar.NewSSE(responseWriter, request)
	s.patchPanel(sse, status) //nolint:contextcheck // templ rendering handles context internally
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

	sse := datastar.NewSSE(responseWriter, request)
	s.patchPanel(sse, status) //nolint:contextcheck // templ rendering handles context internally
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
	mux.HandleFunc("POST /api/audio/{mode}", server.handleAudio)
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
