//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"github.com/coreos/go-systemd/v22/daemon"
)

type Daemon struct {
	mu        sync.RWMutex
	cmdMu     sync.Mutex
	state     pixy.State
	config    pixy.Config
	videoDev  string
	hidrawDev string

	debounceInUse int
	debounceIdle  int
	lastSyncedAt  time.Time

	lastFrame struct {
		sync.RWMutex
		data []byte
	}

	ptzCache struct {
		mu        sync.RWMutex
		values    ptzValues
		expiresAt time.Time
	}

	streamSema chan struct{}
}

func NewDaemon(cfg pixy.Config) (*Daemon, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	d := &Daemon{
		mu:         sync.RWMutex{},
		config:     cfg,
		state:      pixy.DefaultState(),
		videoDev:   "",
		hidrawDev:  "",
		streamSema: make(chan struct{}, 1),
	}
	registerMetrics()
	d.loadState()
	d.probeDevices()

	return d, nil
}

func (d *Daemon) setDeviceState(
	ctx context.Context,
	configBytes, commitBytes []byte,
	setter stateSetter,
) error {
	d.mu.RLock()
	hidrawDev := d.hidrawDev
	d.mu.RUnlock()

	if hidrawDev == "" {
		return fmt.Errorf("setDeviceState: %w", pixy.ErrPIXYNotConnected)
	}

	err := hidSend(hidrawDev, configBytes)
	if err != nil {
		d.mu.Lock()
		d.probeDevices()
		d.mu.Unlock()

		return fmt.Errorf("setDeviceState send config: %w", err)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("setDeviceState sleep: %w", ctx.Err())
	case <-time.After(hidCommandSleepMs * time.Millisecond):
	}

	err = hidSend(hidrawDev, commitBytes)
	if err != nil {
		return fmt.Errorf("setDeviceState send commit: %w", err)
	}

	d.mu.Lock()
	setter(d)

	if saveErr := d.saveState(); saveErr != nil {
		slog.Error("failed to save state", "error", saveErr)
	}
	d.mu.Unlock()

	return nil
}

func (d *Daemon) setTracking(ctx context.Context, mode pixy.CameraState) error {
	return d.setDeviceState(
		ctx,
		pixyConfig(hidInterfaceTracking, cameraHIDByte(mode)),
		pixyCommit(hidInterfaceTracking),
		func(d *Daemon) { d.state.Camera = mode },
	)
}

func (d *Daemon) setAudio(ctx context.Context, mode pixy.AudioMode) error {
	return d.setDeviceState(
		ctx,
		pixyConfig(hidInterfaceAudio, audioHIDByte(mode)),
		pixyCommit(hidInterfaceAudio),
		func(d *Daemon) { d.state.Audio = mode },
	)
}

func (d *Daemon) setGesture(ctx context.Context, enabled bool) error {
	var mark byte = hidByteIdle
	if enabled {
		mark = gestureEnabledByte
	}

	return d.setDeviceState(
		ctx,
		pixyConfig(hidInterfaceGesture, mark),
		pixyCommit(hidInterfaceGesture),
		func(d *Daemon) { d.state.Gesture = enabled },
	)
}

func (d *Daemon) centerCamera(ctx context.Context) error {
	d.mu.RLock()
	videoDev := d.videoDev
	d.mu.RUnlock()

	if videoDev == "" {
		return fmt.Errorf("centerCamera: %w", pixy.ErrPIXYNotConnected)
	}

	err := v4l2SetMultiple(ctx, videoDev, map[string]string{
		"pan_absolute":  "0",
		"tilt_absolute": "0",
		"zoom_absolute": "100",
	})
	if err != nil {
		return fmt.Errorf("centerCamera: %w", err)
	}

	return nil
}

func (d *Daemon) queryTracking(ctx context.Context) (pixy.CameraState, error) {
	d.mu.RLock()
	hidrawDev := d.hidrawDev
	d.mu.RUnlock()

	return queryHIDState(
		ctx,
		hidrawDev,
		[]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01},
		func(p hidResponse) pixy.CameraState { return p.Tracking },
	)
}

func (d *Daemon) queryAudio(ctx context.Context) (pixy.AudioMode, error) {
	d.mu.RLock()
	hidrawDev := d.hidrawDev
	d.mu.RUnlock()

	return queryHIDState(
		ctx,
		hidrawDev,
		[]byte{cameraConfigPrefix, hidInterfaceAudio, audioConfigMarker, 0x04},
		func(p hidResponse) pixy.AudioMode { return p.Audio },
	)
}

func (d *Daemon) queryGesture(ctx context.Context) (bool, error) {
	d.mu.RLock()
	hidrawDev := d.hidrawDev
	d.mu.RUnlock()

	return queryHIDState(
		ctx,
		hidrawDev,
		[]byte{
			cameraConfigPrefix, hidInterfaceGesture,
			gestureConfigMark1, gestureConfigMark2,
			0x00, cameraConfigMarker,
			0x00, cameraConfigMarker,
			gestureConfigMark3,
		},
		func(p hidResponse) bool { return p.Gesture },
	)
}

func (d *Daemon) syncState(ctx context.Context) string {
	d.mu.RLock()
	videoDev := d.videoDev
	d.mu.RUnlock()

	if videoDev == "" {
		return "error: PIXY not connected"
	}

	tracking, trackingErr := d.queryTracking(ctx)
	audio, audioErr := d.queryAudio(ctx)
	gesture, gestureErr := d.queryGesture(ctx)

	d.mu.Lock()
	changed := false

	if trackingErr == nil && tracking.Valid() && tracking != pixy.StateOffline {
		if d.state.Camera != tracking {
			slog.Info("state sync: camera changed", "believed", d.state.Camera, "actual", tracking)
			d.state.Camera = tracking
			changed = true
		}
	} else if trackingErr != nil {
		slog.Debug("tracking query failed", "error", trackingErr)
	}

	if audioErr == nil && audio.Valid() {
		if d.state.Audio != audio {
			slog.Info("state sync: audio changed", "believed", d.state.Audio, "actual", audio)
			d.state.Audio = audio
			changed = true
		}
	} else if audioErr != nil {
		slog.Debug("audio query failed", "error", audioErr)
	}

	if gestureErr == nil {
		if d.state.Gesture != gesture {
			slog.Info("state sync: gesture changed", "believed", d.state.Gesture, "actual", gesture)
			d.state.Gesture = gesture
			changed = true
		}
	} else {
		slog.Debug("gesture query failed", "error", gestureErr)
	}

	d.lastSyncedAt = time.Now()

	if changed {
		if saveErr := d.saveState(); saveErr != nil {
			slog.Error("failed to save synced state", "error", saveErr)
		}

		d.mu.Unlock()

		return "synced (state updated from camera)"
	}

	d.mu.Unlock()

	return "synced (no changes)"
}

func boolStr(b bool, ifTrue, ifFalse string) string {
	if b {
		return ifTrue
	}
	return ifFalse
}

func sdNotify(state string) {
	sent, err := daemon.SdNotify(false, state)
	if err != nil {
		slog.Debug("sd_notify failed", "error", err)
	} else if !sent {
		slog.Debug("sd_notify not sent (no NOTIFY_SOCKET)")
	}
}

func (d *Daemon) getStatus(ctx context.Context) string {
	d.mu.RLock()
	videoDev := d.videoDev
	camera := d.state.Camera
	audio := d.state.Audio
	gesture := d.state.Gesture
	inCall := d.state.InCall
	autoMode := d.state.AutoMode
	d.mu.RUnlock()

	if videoDev == "" {
		return fmt.Sprintf(
			"camera=%s audio=%s gesture=%v pan=%d tilt=%d zoom=%d in_call=%s auto=%s device=",
			pixy.StateOffline,
			audio,
			gesture,
			0,
			0,
			0,
			boolStr(inCall, "yes", "no"),
			boolStr(autoMode, "on", "off"),
		)
	}

	ptz := parsePTZValues(ctx, videoDev)

	return fmt.Sprintf(
		"camera=%s audio=%s gesture=%v pan=%d tilt=%d zoom=%d in_call=%s auto=%s device=%s",
		camera,
		audio,
		gesture,
		ptz.Pan,
		ptz.Tilt,
		ptz.Zoom,
		boolStr(inCall, "yes", "no"),
		boolStr(autoMode, "on", "off"),
		videoDev,
	)
}

func (d *Daemon) waybarOutput() string {
	d.mu.RLock()
	camera := d.state.Camera
	audio := d.state.Audio
	inCall := d.state.InCall
	autoMode := d.state.AutoMode
	d.mu.RUnlock()

	icon := ""
	class := ""
	text := ""

	switch camera {
	case pixy.StateTracking:
		icon = "\uf030"
		class = "tracking"
		text = "CAM"
	case pixy.StatePrivacy:
		icon = "\uf011"
		class = string(pixy.StatePrivacy)
		text = "OFF"
	case pixy.StateIdle:
		icon = "\uf03d"
		class = "idle"
		text = "IDLE"
	case pixy.StateOffline:
		icon = "\uf00d"
		class = "offline"
		text = "---"
	}

	if inCall {
		class += " in-call"
	}

	tooltip := fmt.Sprintf("EMEET PIXY: %s", camera)
	tooltip += fmt.Sprintf("\nAudio: %s", audio)

	tooltip += fmt.Sprintf("\nAuto: %t", autoMode)
	if inCall {
		tooltip += "\nIn call: yes"
	}

	out := map[string]string{
		"text":    icon + " " + text,
		"tooltip": tooltip,
		"class":   "custom-camera " + class,
	}

	data, err := json.Marshal(out)
	if err != nil {
		return `{"text":"?","tooltip":"json marshal error","class":"custom-camera offline"}`
	}

	return string(data)
}

const socketIOTimeout = 5 * time.Second

func (d *Daemon) listenUnix(ctx context.Context) error {
	socketPath := d.config.SocketPath()
	_ = os.Remove(socketPath)

	createErr := os.MkdirAll(d.config.StateDir, pixy.PermissionStateDir)
	if createErr != nil {
		return fmt.Errorf("create state dir: %w", createErr)
	}

	lc := net.ListenConfig{}

	listener, err := lc.Listen(ctx, "unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	defer func() {
		closeErr := listener.Close()
		if closeErr != nil {
			slog.Debug("listener close error", "error", closeErr)
		}
	}()

	chmodErr := os.Chmod(socketPath, pixy.PermissionSocket)
	if chmodErr != nil {
		slog.Error("failed to set socket permissions", "error", chmodErr)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			slog.Error("socket accept error", "error", err)

			continue
		}

		buf := make([]byte, pixy.SocketBufSize)

		_ = conn.SetReadDeadline(time.Now().Add(socketIOTimeout))
		n, readErr := conn.Read(buf)
		if readErr == nil && n > 0 {
			cmd := strings.TrimSpace(string(buf[:n]))

			response := d.handleCommand(ctx, cmd) + "\n"

			_ = conn.SetWriteDeadline(time.Now().Add(socketIOTimeout))
			_, writeErr := conn.Write([]byte(response))
			if writeErr != nil {
				slog.Debug("socket write error", "error", writeErr)
			}
		}

		closeErr := conn.Close()
		if closeErr != nil {
			slog.Debug("conn close error", "error", closeErr)
		}
	}
}

func sendCommand(cfg pixy.Config, cmd string) (string, error) {
	resp, err := pixy.SendCommand(context.Background(), cfg.SocketPath(), cmd)
	if err != nil {
		return "", fmt.Errorf("sendCommand %q: %w", cmd, err)
	}

	return resp, nil
}

func (d *Daemon) Run() {
	createErr := os.MkdirAll(d.config.StateDir, pixy.PermissionStateDir)
	if createErr != nil {
		slog.Error("failed to create state dir", "error", createErr)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		listenErr := d.listenUnix(ctx)
		if listenErr != nil {
			slog.Error("unix socket error", "error", listenErr)
		}
	}()

	var httpSrv *http.Server
	if d.config.WebAddr != "" {
		webSrv := &webServer{daemon: d}
		mux := newWebMux(webSrv)
		httpSrv = &http.Server{
			Addr:              d.config.WebAddr,
			Handler:           requestIDMiddleware(securityMiddleware(mux)),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
		}

		go func() {
			slog.Info("web UI starting", "addr", d.config.WebAddr)
			listenErr := httpSrv.ListenAndServe()
			if listenErr != nil && listenErr != http.ErrServerClosed {
				slog.Error("web server error", "error", listenErr)
			}
		}()
	}

	slog.Info("EMEET PIXY daemon started")
	sdNotify("READY=1")
	d.mu.Lock()
	slog.Info(
		"initial state",
		"camera",
		d.state.Camera,
		"audio",
		d.state.Audio,
		"auto",
		d.state.AutoMode,
	)
	d.mu.Unlock()

	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	ueventCh := make(chan struct{}, 8)
	go d.listenUevents(ueventCh)

	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGHUP {
				slog.Info("received SIGHUP, saving state")
				d.mu.Lock()
				if saveErr := d.saveState(); saveErr != nil {
					slog.Error("failed to save state on SIGHUP", "error", saveErr)
				}
				d.mu.Unlock()
				continue
			}
			sdNotify("STOPPING=1")
			slog.Info("shutting down")
			d.mu.Lock()
			if saveErr := d.saveState(); saveErr != nil {
				slog.Error("failed to save state on shutdown", "error", saveErr)
			}
			d.mu.Unlock()
			cancel()
			_ = os.Remove(d.config.SocketPath())
			if httpSrv != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = httpSrv.Shutdown(shutdownCtx)
				cancel()
			}
			return
		case <-ueventCh:
			slog.Info("device event detected, re-probing")
			d.cmdMu.Lock()
			d.mu.Lock()
			oldVideo := d.videoDev
			d.probeDevices()
			newVideo := d.videoDev
			d.mu.Unlock()
			if oldVideo == "" && newVideo != "" {
				slog.Info("device appeared, syncing state")
				_ = d.syncState(ctx)
			}
			d.cmdMu.Unlock()
		case <-ticker.C:
			d.autoManage(ctx)
			sdNotify("WATCHDOG=1")
		}
	}
}

func exitWithDaemonError(err error) {
	if err != nil {
		_, dieErr := fmt.Fprintf(os.Stderr, "Error: %v\nIs emeet-pixyd running?\n", err)
		_ = dieErr
		os.Exit(1)
	}
}

func main() {
	cfg := pixy.DefaultConfig()

	if len(os.Args) > 1 {
		cmd := strings.Join(os.Args[1:], " ")

		resp, err := sendCommand(cfg, cmd)
		exitWithDaemonError(err)

		_, printErr := fmt.Fprintln(os.Stdout, resp)
		if printErr != nil {
			slog.Debug("failed to print response", "error", printErr)
		}

		return
	}

	d, err := NewDaemon(cfg)
	if err != nil {
		slog.Error("daemon init failed", "error", err)
		os.Exit(1)
	}
	d.Run()
}
