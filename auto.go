//go:build linux

package main

import (
	"context"
	"log/slog"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	errorfamily "github.com/larsartmann/go-error-family"
)

func (d *Daemon) handleCallStart(
	ctx context.Context,
	camera pixy.CameraState,
	autoMode pixy.AutoMode,
) {
	d.mu.Lock()
	d.state.InCall = true
	d.mu.Unlock()

	var errs []error

	log := slog.With("auto_mode", autoMode)

	if autoMode.ActivatesTracking() && (camera == pixy.StatePrivacy || camera == pixy.StateIdle) {
		trackErr := d.deps.setTracking(ctx, pixy.StateTracking)
		if trackErr != nil {
			log.Error("failed to activate tracking", "error", trackErr)
			errs = append(errs, pixy.Wrap(trackErr, "auto.track", "tracking"))
		}
	}

	if autoMode.ActivatesAudio() {
		audioErr := d.deps.setAudio(ctx, pixy.AudioNC)
		if audioErr != nil {
			log.Error("failed to set audio mode", "error", audioErr)
			errs = append(errs, pixy.Wrap(audioErr, "auto.audio", "audio"))
		}
	}

	if autoMode.SwitchesSource() {
		src, srcErr := d.deps.findSource(ctx)
		if srcErr == nil {
			d.deps.setSource(ctx, src)
			log.Info("set PipeWire default source to PIXY", "id", src.Get())
		} else {
			log.Error("failed to find PIXY audio source", "error", srcErr)
			errs = append(errs, pixy.Wrap(srcErr, "auto.source", "source"))
		}
	}

	d.mu.Lock()
	d.autoError = errorfamily.Compose(errs...)
	d.mu.Unlock()

	d.broadcastStateChanged()
	d.deps.notify(ctx, "EMEET PIXY", "Camera activated — "+autoMode.String()+" mode")
}

func (d *Daemon) handleCallEnd(ctx context.Context, autoMode pixy.AutoMode) {
	d.mu.Lock()
	d.state.InCall = false
	d.mu.Unlock()

	var autoErr error

	log := slog.With("auto_mode", autoMode)

	if autoMode.ActivatesPrivacy() {
		privacyErr := d.deps.setTracking(ctx, pixy.StatePrivacy)
		if privacyErr != nil {
			log.Error("failed to enter privacy mode", "error", privacyErr)
			autoErr = pixy.Wrap(privacyErr, "auto.privacy", "privacy")
		}

		d.deps.notify(ctx, "EMEET PIXY", "Camera privacy mode — physically disabled")
	} else {
		d.deps.notify(ctx, "EMEET PIXY", "Call ended")
	}

	d.mu.Lock()
	d.autoError = autoErr
	d.mu.Unlock()

	d.broadcastStateChanged()
}

func (d *Daemon) autoManage(ctx context.Context) {
	d.hidMu.Lock()
	defer d.hidMu.Unlock()

	d.mu.RLock()
	videoDev := d.videoDev
	autoMode := d.state.AutoMode
	d.mu.RUnlock()

	if videoDev == "" {
		d.mu.Lock()
		d.applyProbeResultLocked(probeDevices(d.config.ExtraProductIDs)) //nolint:contextcheck
		videoDev = d.videoDev
		d.mu.Unlock()
		d.broadcastStateChanged()

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

	log := slog.With("auto_mode", autoMode)

	if inUse && !inCall && debounceInUse >= debounceCount {
		log.Info("camera in use, activating")
		d.handleCallStart(ctx, camera, autoMode)

		changed = true
	}

	if !inUse && inCall && debounceIdle >= debounceCount {
		log.Info("camera released")
		d.handleCallEnd(ctx, autoMode)

		changed = true
	}

	if changed {
		d.mu.Lock()
		d.saveStateOrLog("failed to save state after auto-manage")
		d.mu.Unlock()
	} else {
		d.mu.Lock()
		d.autoError = nil
		d.mu.Unlock()
	}

	d.mu.RLock()
	updateMetrics(d.state) //nolint:contextcheck
	d.mu.RUnlock()
}
