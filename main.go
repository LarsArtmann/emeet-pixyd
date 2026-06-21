//go:build linux

// Package main implements the emeet-pixyd daemon for the EMEET PIXY webcam.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"github.com/coreos/go-systemd/v22/daemon"
)

// Build info, overridden via -ldflags.
//
//nolint:gochecknoglobals
var buildVersion = "dev"

type Daemon struct {
	mu        sync.RWMutex
	cmdMu     sync.Mutex
	state     pixy.State
	config    pixy.Config
	videoDev  string
	hidrawDev string
	hidDev    HIDDevice

	// Debounce counters: number of consecutive polls observing a stable
	// in-use or idle state. Both clamp to config.DebounceCount so the
	// >= check below is the trigger boundary, not unbounded growth.
	debounceInUse int
	debounceIdle  int
	hidFailCount  int
	autoError     error
	lastSyncedAt  time.Time

	lastFrame lastFrameCache

	ptzCache ptzCache

	streamSema chan struct{}

	// broadcaster distributes state-change events to all connected SSE
	// clients via a thread-safe fan-out hub.
	broadcaster *Broadcaster

	deps Dependencies
}

const hidCircuitBreakerThreshold = 3

const (
	httpReadHeaderTimeout = 5 * time.Second
	httpReadTimeout       = 10 * time.Second
	httpWriteTimeout      = 30 * time.Second
	httpIdleTimeout       = 60 * time.Second
	httpMaxHeaderBytes    = 1 << 20
	ueventChBufSize       = 8
	shutdownTimeout       = 5 * time.Second
)

func NewDaemon(cfg pixy.Config) (*Daemon, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	//nolint:exhaustruct // remaining fields set below or zero-valued
	d := &Daemon{
		config:      cfg,
		state:       pixy.DefaultState(),
		streamSema:  make(chan struct{}, 1),
		broadcaster: NewBroadcaster(),
	}
	//nolint:exhaustruct // remaining deps set below (circular ref on d.setTracking etc)
	d.deps = Dependencies{
		isCameraInUse: isCameraInUse,
		findSource:    findPixySource,
		setSource:     setDefaultSource,
		notify:        notify,
	}
	d.deps.setTracking = d.setTracking
	d.deps.setAudio = d.setAudio
	d.deps.setGesture = d.setGesture
	d.deps.centerCamera = d.centerCamera
	d.deps.v4l2Set = v4l2Set
	d.deps.parsePTZ = parsePTZValues
	// Persisted state wins on subsequent restarts; env-configured defaults apply
	// only on first run (no valid state file present). This way EMEET_PIXYD_AUTO
	// and EMEET_PIXYD_DEFAULT_AUDIO seed initial state, then the daemon takes over.
	if !d.loadState() {
		d.state.AutoMode = cfg.AutoMode
		d.state.Audio = cfg.DefaultAudio
	}

	registerMetrics()
	// NewDaemon runs before any goroutines exist, so we can call the
	// _Locked variant directly without taking d.mu.
	d.applyProbeResultLocked(probeDevices())
	checkExternalDeps()

	return d, nil
}

func checkExternalDeps() {
	for _, dep := range []struct {
		binary string
		impact string
	}{
		{ffmpegBin, "MJPEG streaming unavailable"},
		{v4l2ctl, "PTZ control unavailable"},
		{wpctl, "PipeWire source switching unavailable"},
		{notifySend, "desktop notifications unavailable"},
	} {
		_, err := exec.LookPath(dep.binary)
		if err != nil {
			slog.Warn("optional dependency not found", "binary", dep.binary, "impact", dep.impact)
		}
	}
}

func sdNotify(state string) {
	sent, err := daemon.SdNotify(false, state)
	if err != nil {
		slog.Warn("sd_notify failed", "error", err)
	} else if !sent {
		slog.Debug("sd_notify not sent (no NOTIFY_SOCKET)")
	}
}

func (d *Daemon) newHTTPServer() *http.Server {
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)
	//nolint:exhaustruct
	return &http.Server{
		Addr: d.config.WebAddr,
		Handler: chain(
			securityHeaderMiddleware, loggingMiddleware, requestIDMW,
		)(mux),
		ReadHeaderTimeout: httpReadHeaderTimeout,
		ReadTimeout:       httpReadTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
		MaxHeaderBytes:    httpMaxHeaderBytes,
	}
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

	httpSrv := d.startHTTPServer()

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

	d.eventLoop(ctx, cancel, sigs, httpSrv)
}

// broadcastStateChanged notifies all active SSE clients that they should
// refresh the UI. Uses a thread-safe fan-out hub for race-safe, non-blocking
// fan-out — slow clients drop events without stalling the daemon.
func (d *Daemon) broadcastStateChanged() {
	d.broadcaster.Broadcast(SSEEvent{ //nolint:exhaustruct
		Event: sseEventRefresh,
		Data:  "{}",
	})
}

func (d *Daemon) startHTTPServer() *http.Server {
	if d.config.WebAddr == "" {
		return nil
	}

	httpSrv := d.newHTTPServer()

	go func() {
		slog.Info("web UI starting", "addr", d.config.WebAddr)

		listenErr := httpSrv.ListenAndServe()
		if listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			slog.Error("web server error", "error", listenErr)
		}
	}()

	return httpSrv
}

func (d *Daemon) handleShutdown(cancel context.CancelFunc, httpSrv *http.Server) {
	sdNotify("STOPPING=1")
	slog.Info("shutting down")
	d.mu.Lock()
	d.saveStateOrLog("failed to save state on shutdown")
	d.mu.Unlock()
	cancel()

	_ = os.Remove(d.config.SocketPath())

	if httpSrv != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		_ = httpSrv.Shutdown(shutdownCtx)

		shutdownCancel()
	}
}

func (d *Daemon) eventLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	sigs <-chan os.Signal,
	httpSrv *http.Server,
) {
	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	ueventCh := make(chan struct{}, ueventChBufSize)
	go d.listenUevents(ctx, ueventCh)

	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGHUP {
				slog.Info("received SIGHUP, saving state")
				d.mu.Lock()
				d.saveStateOrLog("failed to save state on SIGHUP")
				d.mu.Unlock()

				continue
			}

			d.handleShutdown(cancel, httpSrv) //nolint:contextcheck // shutdown cancels the parent context

			return
		case <-ueventCh:
			slog.Info("device event detected, re-probing")
			d.cmdMu.Lock()
			d.mu.Lock()
			oldVideo := d.videoDev
			d.applyProbeResultLocked(probeDevices()) //nolint:contextcheck
			newVideo := d.videoDev
			d.mu.Unlock()
			d.broadcastStateChanged()

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

func handleFlag() bool {
	if len(os.Args) < 2 {
		return false
	}

	switch os.Args[1] {
	case "--version", "-v":
		_, printErr := fmt.Fprintln(os.Stdout, "emeet-pixyd", buildVersion)
		if printErr != nil {
			slog.Debug("failed to print version", "error", printErr)
		}

		return true
	case "--help", "-h":
		_, _ = fmt.Fprintln(
			os.Stdout,
			"Usage: emeet-pixyd [command]\n\nRun without arguments to start the daemon.\nRun with a command to send it to a running daemon via Unix socket.\n\nCommands:\n  status            Show current camera status\n  waybar            Output waybar-compatible JSON\n  version           Print version\n  sync              Sync state from hardware\n  probe             Re-probe for PIXY\n  device            Show device paths\n  track             Enable tracking mode\n  idle              Set idle mode\n  privacy           Enable privacy mode\n  toggle-privacy    Toggle privacy on/off\n  center            Center camera (pan/tilt/zoom reset)\n  audio [mode]      Cycle or set audio mode (nc, live, org/original)\n  gesture-on        Enable gesture control\n  gesture-off       Disable gesture control\n  toggle-gesture    Toggle gesture control\n  auto              Show current auto mode\n  auto-on           Enable auto mode (full)\n  auto-off          Disable auto mode\n  toggle-auto       Toggle auto mode\n  pan <degrees>     Set pan position (absolute; rel+/- for relative)\n  tilt <degrees>    Set tilt position (absolute; rel+/- for relative)\n  zoom <value>      Set zoom level (absolute; rel+/- for relative)",
		)

		return true
	default:
		return false
	}
}

func main() {
	cfg := pixy.ConfigFromEnv()

	if cfg.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if handleFlag() {
		return
	}

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
