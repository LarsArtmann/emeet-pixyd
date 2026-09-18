//go:build linux

package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func (d *Daemon) setDeviceState(
	ctx context.Context,
	configBytes, commitBytes []byte,
	mutator stateMutator,
) error {
	d.mu.RLock()
	hidDev := d.hidDev
	circuitOpen := d.hidFailCount >= hidCircuitBreakerThreshold
	d.mu.RUnlock()

	if hidDev == nil {
		return fmt.Errorf("setDeviceState (no device): %w", pixy.ErrPIXYNotConnected)
	}

	if circuitOpen {
		return fmt.Errorf("setDeviceState: %w", pixy.ErrPIXYNotConnected)
	}

	err := hidDev.Send(configBytes)
	if err != nil {
		d.mu.Lock()
		d.hidFailCount++

		recordHIDFailure(ctx)

		if d.hidFailCount < hidCircuitBreakerThreshold {
			d.applyProbeResultLocked(probeDevices()) //nolint:contextcheck
		}
		d.mu.Unlock()
		d.broadcastStateChanged()

		return fmt.Errorf("setDeviceState send config: %w", err)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("setDeviceState: %w", ctx.Err())
	case <-time.After(hidCommandSleepMs * time.Millisecond):
	}

	err = hidDev.Send(commitBytes)
	if err != nil {
		d.mu.Lock()
		d.hidFailCount++

		recordHIDFailure(ctx)
		d.mu.Unlock()
		d.broadcastStateChanged()

		return fmt.Errorf("setDeviceState send commit: %w", err)
	}

	d.mu.Lock()
	d.hidFailCount = 0
	mutator(d)
	d.saveStateOrLog("failed to save state")
	d.mu.Unlock()
	d.broadcastStateChanged()

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
	videoDev := d.videoDevice()

	if videoDev == "" {
		return fmt.Errorf("centerCamera: %w", pixy.ErrPIXYNotConnected)
	}

	controls := map[string]string{
		ptzAxes[pixy.AxisPan].V4L2Ctrl:  "0",
		ptzAxes[pixy.AxisTilt].V4L2Ctrl: "0",
		ptzAxes[pixy.AxisZoom].V4L2Ctrl: strconv.Itoa(pixy.ZoomDefault),
	}
	for ctrl, val := range controls {
		err := d.deps.v4l2Set(ctx, videoDev, ctrl, val)
		if err != nil {
			return fmt.Errorf("centerCamera %s=%s: %w", ctrl, val, err)
		}
	}

	return nil
}

func (d *Daemon) videoDevice() string {
	d.mu.RLock()
	dev := d.videoDev
	d.mu.RUnlock()

	return dev
}

func (d *Daemon) queryTracking(ctx context.Context) (pixy.CameraState, error) {
	return queryHIDState(
		ctx, d.hidDevice(),
		[]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01},
		func(p hidResponse) pixy.CameraState { return p.Tracking },
	)
}

func (d *Daemon) queryAudio(ctx context.Context) (pixy.AudioMode, error) {
	return queryHIDState(
		ctx, d.hidDevice(),
		[]byte{cameraConfigPrefix, hidInterfaceAudio, audioConfigMarker, 0x04},
		func(p hidResponse) pixy.AudioMode { return p.Audio },
	)
}

func (d *Daemon) queryGesture(ctx context.Context) (bool, error) {
	return queryHIDState(
		ctx, d.hidDevice(),
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

func (d *Daemon) hidDevice() HIDDevice {
	d.mu.RLock()
	dev := d.hidDev
	d.mu.RUnlock()

	return dev
}

func (d *Daemon) syncState(ctx context.Context) CommandResult {
	videoDev := d.videoDevice()

	if videoDev == "" {
		return errResult(cmdSync, pixy.ErrPIXYNotConnected)
	}

	tracking, trackingErr := d.queryTracking(ctx)
	audio, audioErr := d.queryAudio(ctx)
	gesture, gestureErr := d.queryGesture(ctx)

	d.mu.Lock()
	changed := false

	log := slog.With("device", d.hidrawDev)

	if trackingErr == nil && tracking.Valid() && tracking != pixy.StateOffline {
		if d.state.Camera != tracking {
			log.Info("state sync: camera changed", "believed", d.state.Camera, "actual", tracking)
			d.state.Camera = tracking
			changed = true
		}
	} else if trackingErr != nil {
		log.Warn("tracking query failed", "error", trackingErr)
	}

	changed = d.adoptSecondaryStateLocked(audio, audioErr, gesture, gestureErr) || changed

	d.lastSyncedAt = time.Now()

	if changed {
		d.saveStateOrLog("failed to save synced state")
		d.mu.Unlock()
		d.broadcastStateChanged()

		return okResult("synced (state updated from camera)")
	}

	d.mu.Unlock()

	return okResult("synced (no changes)")
}

// adoptSecondaryStateLocked applies queried audio and gesture values to
// daemon state, logging (but not propagating) query failures. It reports
// whether any value changed. Caller must hold d.mu.
func (d *Daemon) adoptSecondaryStateLocked(
	audio pixy.AudioMode,
	audioErr error,
	gesture bool,
	gestureErr error,
) bool {
	log := slog.With("device", d.hidrawDev)
	changed := false

	if audioErr == nil && audio.Valid() {
		if d.state.Audio != audio {
			log.Info("state sync: audio changed", "believed", d.state.Audio, "actual", audio)
			d.state.Audio = audio
			changed = true
		}
	} else if audioErr != nil {
		log.Warn("audio query failed", "error", audioErr)
	}

	if gestureErr == nil {
		if d.state.Gesture != gesture {
			log.Info("state sync: gesture changed", "believed", d.state.Gesture, "actual", gesture)
			d.state.Gesture = gesture
			changed = true
		}
	} else {
		log.Warn("gesture query failed", "error", gestureErr)
	}

	return changed
}

// reconcileOnDeviceAppear aligns daemon belief and hardware when the device
// becomes reachable (daemon startup or hotplug re-appear).
//
// Fresh install (no persisted state existed at startup): hardware is the
// source of truth, belief is adopted from it, so a fresh daemon tells the
// truth about the lens instead of assuming privacy while the camera is on.
//
// Otherwise the persisted camera mode is the user's intent: a hardware mode
// that differs (power cycles and replugs reset the camera to its boot
// default) is re-asserted, so privacy and manual choices survive reboots.
// Audio and gesture always adopt from hardware; the boot defaults are
// acceptable and carry no privacy dimension.
//
// Every failure path logs and keeps the current belief; nothing here is
// fatal. Callers hold d.hidMu (HID access is serialized).
func (d *Daemon) reconcileOnDeviceAppear(ctx context.Context) {
	if d.videoDevice() == "" {
		return
	}

	d.mu.RLock()
	hadPersistedState := d.hadPersistedState
	believed := d.state.Camera
	d.mu.RUnlock()

	if !hadPersistedState {
		_ = d.syncState(ctx)

		return
	}

	actual, queryErr := d.queryTracking(ctx)
	if queryErr != nil {
		slog.Warn("reconcile: camera query failed, keeping persisted mode", "error", queryErr)

		return
	}

	if !actual.Valid() || actual == pixy.StateOffline {
		slog.Warn("reconcile: hardware reported unusable camera mode, keeping persisted mode", "actual", actual)

		return
	}

	if actual != believed {
		slog.Info(
			"reconcile: hardware differs from persisted camera mode, re-asserting",
			"persisted", believed,
			"hardware", actual,
		)

		if setErr := d.setTracking(ctx, believed); setErr != nil {
			slog.Error("reconcile: failed to re-assert persisted camera mode", "mode", believed, "error", setErr)
		}
	}

	audio, audioErr := d.queryAudio(ctx)
	gesture, gestureErr := d.queryGesture(ctx)

	d.mu.Lock()
	if d.adoptSecondaryStateLocked(audio, audioErr, gesture, gestureErr) {
		d.saveStateOrLog("failed to save reconciled state")
	}
	d.mu.Unlock()

	d.broadcastStateChanged()
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
			autoMode,
		)
	}

	ptz := d.deps.parsePTZ(ctx, videoDev)

	base := fmt.Sprintf(
		"camera=%s audio=%s gesture=%v pan=%d tilt=%d zoom=%d in_call=%s auto=%s device=%s",
		camera,
		audio,
		gesture,
		ptz.Pan,
		ptz.Tilt,
		ptz.Zoom,
		boolStr(inCall, "yes", "no"),
		autoMode,
		videoDev,
	)

	// Battery is best-effort (TODO #139): the line appears only when the
	// wired device answers the official HID battery queries.
	reading, ok := d.powerStatus(ctx)

	return appendPowerLine(base, reading, ok)
}

func boolStr(b bool, ifTrue, ifFalse string) string {
	if b {
		return ifTrue
	}

	return ifFalse
}
