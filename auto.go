//go:build linux

// Package main implements the emeet-pixyd daemon for the EMEET PIXY webcam.
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

	if autoMode.ActivatesTracking() && (camera == pixy.StatePrivacy || camera == pixy.StateIdle) {
		trackErr := d.setTracking(ctx, pixy.StateTracking)
		if trackErr != nil {
			slog.Error("failed to activate tracking", "error", trackErr)
		}
	}

	if autoMode.ActivatesAudio() {
		audioErr := d.setAudio(ctx, pixy.AudioNC)
		if audioErr != nil {
			slog.Error("failed to set audio mode", "error", audioErr)
		}
	}

	if autoMode.SwitchesSource() {
		src, srcErr := d.findSourceFn(ctx)
		if srcErr == nil {
			d.setSourceFn(ctx, src)
			slog.Info("set PipeWire default source to PIXY", "id", src)
		}
	}

	d.notifyFn(ctx, "EMEET PIXY", "Camera activated — "+autoMode.String()+" mode")
}

func (d *Daemon) handleCallEnd(ctx context.Context, autoMode pixy.AutoMode) {
	d.mu.Lock()
	d.state.InCall = false
	d.mu.Unlock()

	if autoMode.ActivatesPrivacy() {
		privacyErr := d.setTracking(ctx, pixy.StatePrivacy)
		if privacyErr != nil {
			slog.Error("failed to enter privacy mode", "error", privacyErr)
		}

		d.notifyFn(ctx, "EMEET PIXY", "Camera privacy mode — physically disabled")
	} else {
		d.notifyFn(ctx, "EMEET PIXY", "Call ended")
	}
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

	if autoMode.IsOff() {
		return
	}

	inUse := d.isCameraInUseFn(videoDev)

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
	autoMode = d.state.AutoMode
	debounceCount := d.config.DebounceCount
	d.mu.Unlock()

	if inUse && !inCall && debounceInUse >= debounceCount {
		slog.Info("camera in use, activating", "auto_mode", autoMode)
		d.handleCallStart(ctx, camera, autoMode)
	}

	if !inUse && inCall && debounceIdle >= debounceCount {
		slog.Info("camera released", "auto_mode", autoMode)
		d.handleCallEnd(ctx, autoMode)
	}

	d.mu.Lock()
	d.saveStateOrLog("failed to save state after auto-manage")
	d.mu.Unlock()

	d.mu.RLock()
	updateMetrics(d.state)
	d.mu.RUnlock()
}
