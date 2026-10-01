//go:build linux

package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	errorfamily "github.com/larsartmann/go-error-family"
)

func (d *Daemon) setDeviceState(
	ctx context.Context,
	configBytes, commitBytes []byte,
	mutator stateMutator,
) error { //nolint:erraudit // family-inheriting errorfamily.Wrap; a per-function concrete error type would add no errors.AsType consumer
	hidDev, circuitOpen := d.hidSendGuard()

	if hidDev == nil {
		return errorfamily.WrapInfrastructuref(
			pixy.ErrPIXYNotConnected,
			"device.set_state_nodevice",
			"setDeviceState (no device)",
		)
	}

	if circuitOpen {
		return errorfamily.WrapInfrastructuref(pixy.ErrPIXYNotConnected, "device.set_state_circuit", "setDeviceState")
	}

	err := hidDev.Send(configBytes)
	if err != nil {
		d.recordHIDSendFailure(ctx)

		return errorfamily.Wrap(err, errorfamily.Classify(err), "device.set_state_config", "setDeviceState send config")
	}

	select {
	case <-ctx.Done():
		return errorfamily.Wrap(ctx.Err(), errorfamily.Classify(ctx.Err()), "device.set_state_ctx", "setDeviceState")
	case <-time.After(hidCommandSleepMs * time.Millisecond):
	}

	err = hidDev.Send(commitBytes)
	if err != nil {
		// Deliberately NOT recordHIDSendFailure: commit failures must not
		// re-probe mid config+commit sequence — they accrue to the breaker
		// instead (the realistic breaker trigger; see simulator tests).
		d.mu.Lock()
		d.hidFailCount++

		recordHIDFailure(ctx)
		d.mu.Unlock()
		d.broadcastStateChanged()

		return errorfamily.Wrap(err, errorfamily.Classify(err), "device.set_state_commit", "setDeviceState send commit")
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
	err := d.writeTracking(ctx, mode)
	if err != nil {
		return err
	}

	// A track/idle write racing the firmware's privacy-trap transition is
	// ACKed but silently dropped (hardware-evidenced, privacy_trap.go).
	// When the head just arrived in the trap zone, re-assert the mode once
	// past the transition window so the user's click always lands.
	if mode != pixy.StatePrivacy && d.privacyTrapArmed() {
		d.schedulePrivacyTrapReassert(ctx, mode)
	}

	return nil
}

func (d *Daemon) writeTracking(ctx context.Context, mode pixy.CameraState) error {
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

func (d *Daemon) centerCamera(
	ctx context.Context,
) error { //nolint:erraudit // family-inheriting errorfamily.Wrap; a per-function concrete error type would add no errors.AsType consumer
	videoDev := d.videoDevice()

	if videoDev == "" {
		return errorfamily.WrapInfrastructuref(pixy.ErrPIXYNotConnected, "camera.center_nodevice", "centerCamera")
	}

	controls := map[string]string{
		ptzAxes[pixy.AxisPan].V4L2Ctrl:  "0",
		ptzAxes[pixy.AxisTilt].V4L2Ctrl: "0",
		ptzAxes[pixy.AxisZoom].V4L2Ctrl: strconv.Itoa(pixy.ZoomDefault),
	}
	for ctrl, val := range controls {
		err := d.deps.v4l2Set(ctx, videoDev, ctrl, val)
		if err != nil {
			return errorfamily.Wrapf(
				err,
				errorfamily.Classify(err),
				"camera.center_v4l2",
				"centerCamera %s=%s",
				ctrl,
				val,
			)
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

// devicePresence is a runtime connectivity snapshot. It is observed, never
// persisted: Online means the video node exists (frames can be produced),
// Controllable means the HID node exists (vendor commands can be delivered).
// Keeping this separate from pixy.CameraState is the whole point — the
// camera mode is user intent, presence is a fact about the hardware.
type devicePresence struct {
	Online       bool
	Controllable bool
	VideoDev     string
	HidrawDev    string
}

// presence snapshots connectivity under d.mu (acquire → copy → release).
func (d *Daemon) presence() devicePresence {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.presenceLocked()
}

// presenceLocked is presence for callers already holding d.mu.
func (d *Daemon) presenceLocked() devicePresence {
	return devicePresence{
		Online:       d.videoDev != "",
		Controllable: d.hidrawDev != "",
		VideoDev:     d.videoDev,
		HidrawDev:    d.hidrawDev,
	}
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

// syncState reads camera/audio/gesture from hardware and adopts them into
// daemon belief (the explicit `sync` command). HID access is serialized under
// d.hidMu: a multi-step open/write/read on the shared hidraw node racing
// another HID path could pair a response with the wrong request and be
// silently accepted (parseHIDResponse routes by the response's own byte).
func (d *Daemon) syncState(ctx context.Context) CommandResult {
	d.hidMu.Lock()
	defer d.hidMu.Unlock()

	return d.syncStateLocked(ctx)
}

// syncStateLocked is syncState for callers that already hold d.hidMu (the
// device-reconcile path).
//
// LOCK CONTRACT: caller holds d.hidMu.
func (d *Daemon) syncStateLocked(ctx context.Context) CommandResult {
	hidDev, circuitOpen := d.hidSendGuard()

	if hidDev == nil {
		return errResult(cmdSync, pixy.ErrHIDDeviceNotAvailable)
	}

	if circuitOpen {
		return errResult(cmdSync, pixy.ErrPIXYNotConnected)
	}

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
// becomes reachable (daemon startup, hotplug re-appear, or the auto-manager
// noticing a device it had been missing).
//
// Fresh install (no persisted state has ever been written): hardware is the
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
	believed := d.state.Camera
	d.mu.RUnlock()

	if !d.persistedIntent.Load() {
		_ = d.syncStateLocked(ctx)

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

	// Persisted motor speeds are user intent too (TODO #138 wiring): a power
	// cycle resets the firmware, so re-assert them on every re-appear. With
	// no configured speeds this is a no-op (fresh installs never get here —
	// they return early above). Best-effort: a failure logs and moves are
	// simply slower, never blocked.
	if speedErr := d.reassertSpeedsLocked(ctx, pixy.AxisPan, pixy.AxisTilt, pixy.AxisZoom); speedErr != nil {
		slog.Warn("reconcile: motor-speed re-assert failed", "error", speedErr)
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
			camera,
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
