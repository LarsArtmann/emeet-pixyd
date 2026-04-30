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
	audio pixy.AudioMode,
) {
	d.mu.Lock()
	d.state.InCall = true
	d.mu.Unlock()

	if camera == pixy.StatePrivacy || camera == pixy.StateIdle {
		trackErr := d.setTracking(ctx, pixy.StateTracking)
		if trackErr != nil {
			slog.Error("failed to activate tracking", "error", trackErr)
		}
	}

	if audio != pixy.AudioNC {
		audioErr := d.setAudio(ctx, pixy.AudioNC)
		if audioErr != nil {
			slog.Error("failed to set audio mode", "error", audioErr)
		}
	}

	src, srcErr := findPixySource(ctx)
	if srcErr == nil {
		setDefaultSource(ctx, src)
		slog.Info("set PipeWire default source to PIXY", "id", src)
	}

	notify(ctx, "EMEET PIXY", "Camera activated — tracking enabled")
}

func (d *Daemon) handleCallEnd(ctx context.Context) {
	d.mu.Lock()
	d.state.InCall = false
	d.mu.Unlock()

	privacyErr := d.setTracking(ctx, pixy.StatePrivacy)
	if privacyErr != nil {
		slog.Error("failed to enter privacy mode", "error", privacyErr)
	}

	notify(ctx, "EMEET PIXY", "Camera privacy mode — physically disabled")
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
		d.probeDevices()
		videoDev = d.videoDev
		d.mu.Unlock()

		if videoDev == "" {
			return
		}
	}

	if !autoMode {
		return
	}

	inUse := isCameraInUse(videoDev)

	d.mu.Lock()
	if inUse {
		d.debounceIdle = 0
		d.debounceInUse++
	} else {
		d.debounceInUse = 0
		d.debounceIdle++
	}

	debounceInUse := d.debounceInUse
	debounceIdle := d.debounceIdle
	inCall := d.state.InCall
	camera := d.state.Camera
	audio := d.state.Audio
	debounceCount := d.config.DebounceCount
	d.mu.Unlock()

	if inUse && !inCall && debounceInUse >= debounceCount {
		slog.Info("camera in use, activating tracking and noise cancellation")
		d.handleCallStart(ctx, camera, audio)
	}

	if !inUse && inCall && debounceIdle >= debounceCount {
		slog.Info("camera released, entering privacy mode")
		d.handleCallEnd(ctx)
	}

	d.mu.Lock()
	saveErr := d.saveState()
	d.mu.Unlock()

	if saveErr != nil {
		slog.Error("failed to save state", "error", saveErr)
	}

	d.mu.RLock()
	updateMetrics(d.state)
	d.mu.RUnlock()
}
