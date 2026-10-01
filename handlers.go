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
	"github.com/dustin/go-humanize"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/starfederation/datastar-go/datastar"
)

const (
	maxStreamBufferSize = 10 * 1024 * 1024
	maxBodyBytes        = 1 << 10

	staticCacheMaxAge = 7 * 24 * time.Hour

	toastTrackingEnabled = "Tracking enabled"
	toastCameraIdle      = "Camera idle"
	toastPrivacyOn       = "Privacy mode on"
	toastCameraCentered  = "Camera centered"
	toastStateSynced     = "State synced"
	toastProbedDevices   = "Probed devices"
	toastAudioChanged    = "Audio mode changed"
	toastSpeedChanged    = "Motor speed updated"
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

	return humanize.Time(t)
}

type webServer struct {
	daemon *Daemon
}

func (s *webServer) getWebStatus(ctx context.Context) webStatus {
	s.daemon.mu.RLock()
	//nolint:exhaustruct
	status := webStatus{
		Camera:      s.daemon.state.Camera,
		Audio:       s.daemon.state.Audio,
		Gesture:     s.daemon.state.Gesture,
		InCall:      s.daemon.state.InCall,
		Auto:        s.daemon.state.AutoMode,
		Online:      s.daemon.videoDev != "",
		Device:      s.daemon.videoDev,
		Model:       string(s.daemon.model),
		Error:       errStr(s.daemon.autoError),
		LastSynced:  formatLastSynced(s.daemon.lastSyncedAt),
		Version:     buildVersion,
		PresetNames: s.daemon.state.Presets.SortedNames(),
		TrackMode:   s.daemon.state.EffectiveTrackMode().String(),
		Speeds:      s.daemon.state.Speeds,
	}
	s.daemon.mu.RUnlock()

	if status.Online {
		status.PTZValues = pixy.PTZValues{Pan: 0, Tilt: 0, Zoom: pixy.ZoomDefault}
	}

	// Battery reads take d.mu themselves (v2Read guards), so they happen
	// AFTER the outer lock is released — never nest the RLocks.
	if reading, ok := s.daemon.powerStatus(ctx); ok {
		status.Battery = reading.String()
	}

	return status
}

func (s *webServer) getWebStatusWithPTZ(ctx context.Context) webStatus {
	status := s.getWebStatus(ctx)
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

	if err := writeJSON(responseWriter, status, healthResponse{
		Status:  boolStr(online, "ok", "offline"),
		Camera:  camera,
		Version: buildVersion,
	}); err != nil {
		slog.Debug("health response write failed", "err", err)
	}
}

type healthResponse struct {
	Status  string           `json:"status"`
	Camera  pixy.CameraState `json:"camera"`
	Version string           `json:"version"`
}

// statusResponse is the JSON contract served at GET /api/status for
// status-bar and desktop-shell widgets (Quickshell / DankMaterialShell, or any
// HTTP client). The field names are a stable public interface chosen by widget
// authors — rename with care. Unlike /api/health it always answers 200 while
// the daemon runs, even with the camera unplugged, so a widget can tell
// "daemon down" (connection refused) apart from "camera offline" (`online`
// false) without parsing error bodies.
type statusResponse struct {
	Camera  pixy.CameraState `json:"camera"`
	Device  string           `json:"device"`
	Model   string           `json:"model"`
	Online  bool             `json:"online"`
	InCall  bool             `json:"inCall"`
	Auto    pixy.AutoMode    `json:"auto"`
	Audio   pixy.AudioMode   `json:"audio"`
	Gesture bool             `json:"gesture"`
	Battery string           `json:"battery,omitzero"`
	Version string           `json:"version"`
}

// handleStatusJSON serves the widget-facing status contract. It reuses
// getWebStatus so the payload carries the same state (and the TTL-cached
// battery reading) the web panel and Waybar already show.
func (s *webServer) handleStatusJSON(responseWriter http.ResponseWriter, request *http.Request) {
	status := s.getWebStatus(request.Context())

	resp := statusResponse{
		Camera:  status.Camera,
		Device:  status.Device,
		Model:   status.Model,
		Online:  status.Online,
		InCall:  status.InCall,
		Auto:    status.Auto,
		Audio:   status.Audio,
		Gesture: status.Gesture,
		Battery: status.Battery,
		Version: status.Version,
	}

	if err := writeJSON(responseWriter, http.StatusOK, resp); err != nil {
		slog.Debug("status response write failed", "err", err)
	}
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
	// The server's global WriteTimeout would cut this persistent stream at 30s;
	// clear it so the connection lives until the client disconnects (mirrors the
	// MJPEG path in stream.go). An abandoned client still ends the handler via
	// sse.Context().Done().
	rc := http.NewResponseController(responseWriter)
	if dlErr := rc.SetWriteDeadline(time.Time{}); dlErr != nil {
		slog.Warn("could not clear write deadline; SSE may be cut off by server timeout", "error", dlErr)
	}

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
	if err := sse.PatchElementTempl(statusPanel(status)); err != nil {
		slog.Debug("status panel patch failed", "err", err)
	}

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
	if err := sse.ExecuteScript(script); err != nil {
		slog.Debug("toast script dispatch failed", "err", err)
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

	sse := datastar.NewSSE(responseWriter, request)

	if result.IsError() {
		status := s.getWebStatusWithPTZ(request.Context())
		applyResultToStatus(result, &status, "", "")
		s.patchPanel(sse, status) //nolint:contextcheck // templ rendering handles context internally

		return
	}

	status := s.getWebStatusWithPTZ(request.Context())
	if err := sse.MarshalAndPatchSignals(status.PTZValues); err != nil {
		slog.Debug("PTZ signal patch failed", "err", err)
	}
}

// presetAction builds a handler for the preset commands that address a
// preset by name path segment (save additionally validates the name, see
// handlePresetSave): it rejects an empty name, dispatches
// `preset <verb> <name>` through the command path, and patches the panel
// with the outcome.
func (s *webServer) presetAction(verb string) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		name := request.PathValue("name")
		if name == "" {
			http.Error(responseWriter, "missing preset name", http.StatusBadRequest)

			return
		}

		s.patchPresetResult(responseWriter, request, cmdPreset+" "+verb+" "+name)
	}
}

// patchPresetResult is the shared tail of every preset web handler: run the
// command, refresh the panel status, and morph the panel with the outcome.
func (s *webServer) patchPresetResult(responseWriter http.ResponseWriter, request *http.Request, command string) {
	result := s.daemon.handleCommand(request.Context(), command)

	status := s.getWebStatusWithPTZ(request.Context())
	applyResultToStatus(result, &status, result.String(), toastTypeSuccess)

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

	s.patchPresetResult(responseWriter, request, cmdPreset+" "+presetSave+" "+name)
}

// handlePresetPull implements POST /api/preset/pull — the web surface for
// sweeping hardware motor slots into software presets (TODO #141). The sweep
// is read-only on the hardware side (nothing moves), so unlike push it needs
// no browser confirm() gate.
func (s *webServer) handlePresetPull(responseWriter http.ResponseWriter, request *http.Request) {
	s.patchPresetResult(responseWriter, request, cmdPreset+" "+presetPull)
}

func newWebMux(server *webServer) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", cachingFS{handler: http.FileServer(http.FS(staticFS))})
	mux.HandleFunc("GET /{$}", server.handleIndex)
	mux.HandleFunc("GET /panel", server.handleStatusPanel)
	mux.HandleFunc("GET /api/events", server.handleEvents)
	mux.HandleFunc("GET /api/health", server.handleHealth)
	mux.HandleFunc("GET /api/status", server.handleStatusJSON)
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
	mux.HandleFunc("POST /api/preset/load/{name}", server.presetAction(presetLoad))
	mux.HandleFunc("POST /api/preset/delete/{name}", server.presetAction(presetDelete))
	// push moves the physical camera — the UI gates it behind a browser confirm().
	mux.HandleFunc("POST /api/preset/push/{name}", server.presetAction(presetPush))
	mux.HandleFunc("POST /api/preset/pull", server.handlePresetPull)
	mux.HandleFunc("POST /api/ptz/{axis}", server.handlePTZ)
	mux.HandleFunc("POST /api/speed/{axis}", server.handleSpeed)
	mux.HandleFunc("POST /api/tracking/{variant}", server.handleTrackingVariant)
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

// speedSignals mirrors the client-side speed slider signals
// ($speedPan/$speedTilt/$speedZoom) sent by DataStar in the POST body.
type speedSignals struct {
	SpeedPan  float64 `json:"speedPan"`
	SpeedTilt float64 `json:"speedTilt"`
	SpeedZoom float64 `json:"speedZoom"`
}

func (s speedSignals) get(axis pixy.Axis) float64 {
	switch axis {
	case pixy.AxisPan:
		return s.SpeedPan
	case pixy.AxisTilt:
		return s.SpeedTilt
	case pixy.AxisZoom:
		return s.SpeedZoom
	default:
		return 0
	}
}

// handleSpeed implements POST /api/speed/{axis} — the web surface for the
// HID motor-speed command. Values arrive as DataStar signals and are
// dispatched through the same `speed` command path as the CLI, so validation
// and device behavior have a single definition.
func (s *webServer) handleSpeed(responseWriter http.ResponseWriter, request *http.Request) {
	axis := pixy.Axis(request.PathValue("axis"))

	if string(axis) == "" || !ptzAxisValid(axis) {
		http.Error(responseWriter, "invalid axis", http.StatusBadRequest)

		return
	}

	var signals speedSignals
	if err := datastar.ReadSignals(request, &signals); err != nil {
		http.Error(responseWriter, "invalid signals", http.StatusBadRequest)

		return
	}

	speed := signals.get(axis)
	speedCmd := cmdSpeed + " " + string(axis) + " " + strconv.FormatFloat(speed, 'f', -1, 64)
	result := s.daemon.handleCommand(request.Context(), speedCmd)

	slog.Debug("web speed", "axis", axis, "speed", speed, "response", result.String())

	status := s.getWebStatusWithPTZ(request.Context())

	applyResultToStatus(result, &status, toastSpeedChanged, toastTypeSuccess)

	sse := datastar.NewSSE(responseWriter, request)
	s.patchPanel(sse, status) //nolint:contextcheck // templ rendering handles context internally
}

// handleTrackingVariant implements POST /api/tracking/{variant} — the web
// picker for the tracking variant, routed through the same command path as
// the CLI.
func (s *webServer) handleTrackingVariant(responseWriter http.ResponseWriter, request *http.Request) {
	variant := request.PathValue("variant")

	result := s.daemon.handleCommand(request.Context(), cmdTracking+" "+variant)

	slog.Debug("web tracking variant", "variant", variant, "response", result.String())

	status := s.getWebStatus(request.Context())

	toast, tt := actionToast(cmdTracking)
	applyResultToStatus(result, &status, toast, tt)

	sse := datastar.NewSSE(responseWriter, request)
	s.patchPanel(sse, status) //nolint:contextcheck // templ rendering handles context internally
}
