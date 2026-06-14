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
	setter stateSetter,
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

		return fmt.Errorf("setDeviceState send commit: %w", err)
	}

	d.mu.Lock()
	d.hidFailCount = 0
	setter(d)
	d.saveStateOrLog("failed to save state")
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
		log.Debug("tracking query failed", "error", trackingErr)
	}

	if audioErr == nil && audio.Valid() {
		if d.state.Audio != audio {
			log.Info("state sync: audio changed", "believed", d.state.Audio, "actual", audio)
			d.state.Audio = audio
			changed = true
		}
	} else if audioErr != nil {
		log.Debug("audio query failed", "error", audioErr)
	}

	if gestureErr == nil {
		if d.state.Gesture != gesture {
			log.Info("state sync: gesture changed", "believed", d.state.Gesture, "actual", gesture)
			d.state.Gesture = gesture
			changed = true
		}
	} else {
		log.Debug("gesture query failed", "error", gestureErr)
	}

	d.lastSyncedAt = time.Now()

	if changed {
		d.saveStateOrLog("failed to save synced state")
		d.mu.Unlock()

		return okResult("synced (state updated from camera)")
	}

	d.mu.Unlock()

	return okResult("synced (no changes)")
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

	return fmt.Sprintf(
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
}

func boolStr(b bool, ifTrue, ifFalse string) string {
	if b {
		return ifTrue
	}

	return ifFalse
}
