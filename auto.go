//go:build linux

package main

import (
	"context"
	"log/slog"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func (d *Daemon) handleCallStart(
	ctx context.Context,
	camera pixy.CameraState,
	autoMode pixy.AutoMode,
) {
	d.mu.Lock()
	d.state.InCall = true
	d.mu.Unlock()

	var autoErr string

	if autoMode.ActivatesTracking() && (camera == pixy.StatePrivacy || camera == pixy.StateIdle) {
		trackErr := d.deps.setTracking(ctx, pixy.StateTracking)
		if trackErr != nil {
			slog.Error("failed to activate tracking", "error", trackErr)
			autoErr = "tracking: " + trackErr.Error()
		}
	}

	if autoMode.ActivatesAudio() {
		audioErr := d.deps.setAudio(ctx, pixy.AudioNC)
		if audioErr != nil {
			slog.Error("failed to set audio mode", "error", audioErr)

			if autoErr != "" {
				autoErr += "; "
			}

			autoErr += "audio: " + audioErr.Error()
		}
	}

	if autoMode.SwitchesSource() {
		src, srcErr := d.deps.findSource(ctx)
		if srcErr == nil {
			d.deps.setSource(ctx, src)
			slog.Info("set PipeWire default source to PIXY", "id", src.Get())
		}
	}

	d.mu.Lock()
	d.autoError = autoErr
	d.mu.Unlock()

	d.deps.notify(ctx, "EMEET PIXY", "Camera activated — "+autoMode.String()+" mode")
}

func (d *Daemon) handleCallEnd(ctx context.Context, autoMode pixy.AutoMode) {
	d.mu.Lock()
	d.state.InCall = false
	d.mu.Unlock()

	var autoErr string

	if autoMode.ActivatesPrivacy() {
		privacyErr := d.deps.setTracking(ctx, pixy.StatePrivacy)
		if privacyErr != nil {
			slog.Error("failed to enter privacy mode", "error", privacyErr)
			autoErr = "privacy: " + privacyErr.Error()
		}

		d.deps.notify(ctx, "EMEET PIXY", "Camera privacy mode — physically disabled")
	} else {
		d.deps.notify(ctx, "EMEET PIXY", "Call ended")
	}

	d.mu.Lock()
	d.autoError = autoErr
	d.mu.Unlock()
}

func (d *Daemon) autoManage(ctx context.Context) {
	d.cmdMu.Lock()
	defer d.cmdMu.Unlock()

	d.mu.RLock()
	videoDev := d.videoDev
	autoMode := d.state.AutoMode
	d.mu.RUnlock()

	if videoDev == "" {
		d.mu.Lock()
		d.applyProbeResult(probeDevices()) //nolint:contextcheck
		videoDev = d.videoDev
		d.mu.Unlock()

		if videoDev == "" {
			return
		}
	}

	if autoMode.IsOff() {
		return
	}

	inUse := d.deps.isCameraInUse(videoDev)

	d.mu.Lock()

	debounceCount := d.config.DebounceCount
	if inUse {
		d.debounceIdle = 0

		d.debounceInUse++
		if d.debounceInUse > debounceCount {
			d.debounceInUse = debounceCount
		}
	} else {
		d.debounceInUse = 0

		d.debounceIdle++
		if d.debounceIdle > debounceCount {
			d.debounceIdle = debounceCount
		}
	}

	debounceInUse := d.debounceInUse
	debounceIdle := d.debounceIdle
	inCall := d.state.InCall
	camera := d.state.Camera
	autoMode = d.state.AutoMode
	d.mu.Unlock()

	changed := false

	if inUse && !inCall && debounceInUse >= debounceCount {
		slog.Info("camera in use, activating", "auto_mode", autoMode)
		d.handleCallStart(ctx, camera, autoMode)

		changed = true
	}

	if !inUse && inCall && debounceIdle >= debounceCount {
		slog.Info("camera released", "auto_mode", autoMode)
		d.handleCallEnd(ctx, autoMode)

		changed = true
	}

	if changed {
		d.mu.Lock()
		d.saveStateOrLog("failed to save state after auto-manage")
		d.mu.Unlock()
	} else {
		d.mu.Lock()
		d.autoError = ""
		d.mu.Unlock()
	}

	d.mu.RLock()
	updateMetrics(d.state) //nolint:contextcheck
	d.mu.RUnlock()
}
