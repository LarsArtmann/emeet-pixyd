//go:build linux

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Tests for the V2-head command families in pixySimulator: byte validation,
// round-trips, failure-injection parity, and the commit-shape collision
// regression for motor SET heads.

func v2MotorSpeedReport(motor pixy.MotorType, speed float32, iface byte) []byte {
	head := pixy.V2SetMotorSpeed.WithIface(iface)

	return append(head.Bytes(), pixy.MotorSpeedPayload(motor, speed)...)
}

func TestPixySimulatorV2_MotorSpeedRoundTrip(t *testing.T) {
	t.Parallel()

	for _, iface := range []byte{pixy.V2DevMotor, pixy.MotorMCUIface} {
		sim := newPixySimulator()

		report := v2MotorSpeedReport(pixy.MotorPan, 33.5, iface)
		if err := sim.Send(report); err != nil {
			t.Fatalf("iface %#02x: SetMotorSpeed rejected: %v", iface, err)
		}

		if got := sim.MotorSpeed(pixy.MotorPan); got != 33.5 {
			t.Fatalf("iface %#02x: motor speed = %v, want 33.5", iface, got)
		}

		query := append(pixy.V2GetMotorSpeed.WithIface(iface).Bytes(), byte(pixy.MotorPan))

		resp, err := sim.SendRecv(t.Context(), query)
		if err != nil {
			t.Fatalf("iface %#02x: GetMotorSpeed failed: %v", iface, err)
		}

		reading, err := pixy.ParseMotorSpeedResponse(pixy.V2GetMotorSpeed.WithIface(iface), resp)
		if err != nil {
			t.Fatalf("iface %#02x: parse response: %v", iface, err)
		}

		if reading.Motor != pixy.MotorPan || reading.Speed != 33.5 || reading.Limit != 100.0 {
			t.Fatalf("iface %#02x: round-trip = %+v, want (pan, 33.5, 100)", iface, reading)
		}
	}
}

func TestPixySimulatorV2_MotorSpeedAllAxes(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	speeds := map[pixy.MotorType]float32{
		pixy.MotorPan:  10.25,
		pixy.MotorTilt: -20.5,
		pixy.MotorZoom: 30.75,
	}

	for motor, speed := range speeds {
		if err := sim.Send(v2MotorSpeedReport(motor, speed, pixy.MotorMCUIface)); err != nil {
			t.Fatalf("%s: %v", motor, err)
		}
	}

	for motor, want := range speeds {
		if got := sim.MotorSpeed(motor); got != want {
			t.Errorf("%s speed = %v, want %v", motor, got, want)
		}
	}
}

// TestPixySimulatorV2_SpeedQueryDuality pins the two evidenced speed-query
// forms (TODO #168, for #166 comparisons): the Beta.25 x64 build rides the
// SET_MOTOR_SPEED head routed to the motor-MCU iface (09 63 01 03 + motor
// byte), while the dedicated GET head (09 03 01 13) is the 2.0.3 insertion.
// Both must answer the same [motorType][speed][limit] payload, each echoing
// the head as sent.
func TestPixySimulatorV2_SpeedQueryDuality(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	if err := sim.Send(v2MotorSpeedReport(pixy.MotorPan, 42.5, pixy.MotorMCUIface)); err != nil {
		t.Fatalf("SetMotorSpeed: %v", err)
	}

	forms := map[string]pixy.V2Head{
		"beta.25 set-head form": pixy.V2SetMotorSpeed.WithIface(pixy.MotorMCUIface),
		"2.0.3 get-head form":   pixy.V2GetMotorSpeed.WithIface(pixy.MotorMCUIface),
	}

	for name, head := range forms {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			query := append(head.Bytes(), byte(pixy.MotorPan))

			resp, err := sim.SendRecv(t.Context(), query)
			if err != nil {
				t.Fatalf("%s: SendRecv: %v", name, err)
			}

			reading, err := pixy.ParseMotorSpeedResponse(head, resp)
			if err != nil {
				t.Fatalf("%s: parse: %v", name, err)
			}

			if reading.Motor != pixy.MotorPan || reading.Speed != 42.5 || reading.Limit != 100.0 {
				t.Fatalf("%s: reading = %+v, want (pan, 42.5, 100)", name, reading)
			}
		})
	}
}

func TestPixySimulatorV2_TargetTrackRoundTrip(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	report := append(pixy.V2SetTargetTrack.Bytes(), byte(1))
	report = appendF32s(report, 1.5, 2.5, 3.5)

	if err := sim.Send(report); err != nil {
		t.Fatalf("SetTargetTrack rejected: %v", err)
	}

	mode, args := sim.TargetTrack()
	if mode != 1 || args != [3]float32{1.5, 2.5, 3.5} {
		t.Fatalf("target track = (%d, %v), want (1, [1.5 2.5 3.5])", mode, args)
	}

	resp, err := sim.SendRecv(t.Context(), pixy.V2GetTargetTrack.Bytes())
	if err != nil {
		t.Fatalf("GetTargetTrack failed: %v", err)
	}

	offset := pixy.V2ResponsePayloadOffset

	argsMatch := f32LE(resp[offset+1:]) == 1.5 &&
		f32LE(resp[offset+5:]) == 2.5 &&
		f32LE(resp[offset+9:]) == 3.5

	if resp[offset] != 1 || !argsMatch {
		t.Fatalf("target track response mismatch: %x", resp[:offset+13])
	}
}

func TestPixySimulatorV2_PayloadValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		report  []byte
		wantErr string
	}{
		{
			name:    "motor speed too short",
			report:  v2MotorSpeedReport(pixy.MotorPan, 1, pixy.V2DevMotor)[:7],
			wantErr: "payload too short",
		},
		{
			name:    "motor speed invalid motor type",
			report:  v2MotorSpeedReport(pixy.MotorType(9), 1, pixy.V2DevMotor),
			wantErr: "invalid motor type 9",
		},
		{
			name:    "preset pos slot 0",
			report:  append(pixy.V2SetMotorPresetPos.Bytes(), 0),
			wantErr: "slot 0 is invalid",
		},
		{
			name:    "preset pos too short",
			report:  pixy.V2SetMotorPresetPos.Bytes(),
			wantErr: "payload too short",
		},
		{
			name:    "target track mode out of range",
			report:  append(append(pixy.V2SetTargetTrack.Bytes(), 4), make([]byte, 12)...),
			wantErr: "out of range",
		},
		{
			name:    "target track too short",
			report:  append(pixy.V2SetTargetTrack.Bytes(), 1, 0, 0, 0),
			wantErr: "payload too short",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sim := newPixySimulator()

			err := sim.Send(tc.report)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}

			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want containing %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestPixySimulatorV2_FailureInjectionParity(t *testing.T) {
	t.Parallel()

	t.Run("sendErr applies to V2 SET", func(t *testing.T) {
		t.Parallel()

		sim := newPixySimulator()
		sim.sendErr = errors.New("injected")

		err := sim.Send(v2MotorSpeedReport(pixy.MotorPan, 5, pixy.V2DevMotor))
		if err == nil || err.Error() != "injected" {
			t.Fatalf("err = %v, want injected", err)
		}

		if got := sim.MotorSpeed(pixy.MotorPan); got != 0 {
			t.Fatalf("state mutated despite sendErr: %v", got)
		}
	})

	t.Run("commitErr does not fire on V2 SET", func(t *testing.T) {
		t.Parallel()

		sim := newPixySimulator()
		sim.commitErr = errors.New("commit-only")

		if err := sim.Send(v2MotorSpeedReport(pixy.MotorPan, 5, pixy.V2DevMotor)); err != nil {
			t.Fatalf("V2 SET failed with commitErr set: %v", err)
		}

		if got := sim.MotorSpeed(pixy.MotorPan); got != 5 {
			t.Fatalf("speed = %v, want 5", got)
		}
	})

	t.Run("sendRecvErr applies to V2 GET", func(t *testing.T) {
		t.Parallel()

		sim := newPixySimulator()
		sim.sendRecvErr = errors.New("recv-injected")

		_, err := sim.SendRecv(t.Context(), pixy.V2GetBatteryLevel.Bytes())
		if err == nil || err.Error() != "recv-injected" {
			t.Fatalf("err = %v, want recv-injected", err)
		}
	})

	t.Run("nilResponse applies to V2 GET", func(t *testing.T) {
		t.Parallel()

		sim := newPixySimulator()
		sim.nilResponse = true

		resp, err := sim.SendRecv(t.Context(), pixy.V2GetBatteryLevel.Bytes())
		if err != nil || resp != nil {
			t.Fatalf("(resp, err) = (%x, %v), want (nil, nil)", resp, err)
		}
	})

	t.Run("corruptResp applies to V2 GET", func(t *testing.T) {
		t.Parallel()

		sim := newPixySimulator()
		sim.corruptResp = true

		resp, err := sim.SendRecv(t.Context(), pixy.V2GetBatteryLevel.Bytes())
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		for i, b := range resp {
			if b != 0xFF {
				t.Fatalf("corrupt resp byte %d = %#02x, want 0xFF", i, b)
			}
		}
	})
}

func TestPixySimulatorV2_CommitCollisionRegression(t *testing.T) {
	t.Parallel()

	// SET_MOTOR_SPEED's head [09 03 01 03] matches the commit-report shape
	// ([2]=0x01, [3]==[1]). The 9-byte report must be routed as a V2 SET and
	// must NOT consume a pending legacy config for interface 0x03.
	sim := newPixySimulator()

	if err := sim.Send(v2MotorSpeedReport(pixy.MotorPan, 7.5, pixy.V2DevMotor)); err != nil {
		t.Fatalf("motor speed SET rejected: %v", err)
	}

	sim.state.mu.Lock()
	pending := len(sim.state.pending)
	sim.state.mu.Unlock()

	if pending != 0 {
		t.Fatalf("V2 SET touched legacy pending configs: %d pending", pending)
	}

	// The legacy dialect still works after V2 traffic.
	if err := sim.Send(pixyConfig(hidInterfaceTracking, hidByteTracking)); err != nil {
		t.Fatalf("legacy config rejected after V2 traffic: %v", err)
	}

	if err := sim.Send(pixyCommit(hidInterfaceTracking)); err != nil {
		t.Fatalf("legacy commit rejected after V2 traffic: %v", err)
	}

	if got := sim.Tracking(); got != pixy.StateTracking {
		t.Fatalf("tracking = %s, want tracking", got)
	}
}

func TestPixySimulatorV2_FixtureQueries(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	tests := []struct {
		name string
		head pixy.V2Head
		want []byte
	}{
		{"battery", pixy.V2GetBatteryLevel, []byte{87}},
		{"charge", pixy.V2GetChargeSta, []byte{0}},
		{"func sta", pixy.V2GetFuncSta, []byte{0, 0, 0, 0}},
		{"sn", pixy.V2GetSN, []byte("PIXY-SIM-0001")},
		{"ver", pixy.V2GetVer, []byte{0x03, 0x02}},
		{"device ver", pixy.V2GetDeviceVer, []byte{0x03, 0x02}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp, err := sim.SendRecv(t.Context(), tc.head.Bytes())
			if err != nil {
				t.Fatalf("%s query failed: %v", tc.name, err)
			}

			got := resp[pixy.V2ResponsePayloadOffset : pixy.V2ResponsePayloadOffset+len(tc.want)]

			if string(got) != string(tc.want) {
				t.Fatalf("%s payload = %x, want %x", tc.name, got, tc.want)
			}
		})
	}
}

func TestPixySimulatorV2_UnknownV2HeadRejected(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	// A valid-prefix head that is neither a known GET nor a valid legacy
	// query: [09 06 00 00]. The simulator rejects unrouted heads loudly —
	// strict validation catches wrong byte usage in tests instead of
	// silently mimicking a hardware timeout.
	_, err := sim.SendRecv(t.Context(), []byte{0x09, 0x06, 0x00, 0x00})
	if err == nil {
		t.Fatal("unknown V2 head accepted, want strict rejection")
	}
}

func TestPixySimulatorV2_ConcurrentRoundTrip(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	// One goroutine per motor axis: 10 goroutines on 3 axes would race on
	// last-writer-wins per axis and make the value assert nondeterministic.
	var waitGroup sync.WaitGroup

	errCh := make(chan error, 3)

	for i := range 3 {
		waitGroup.Add(1)

		go func(n int) {
			defer waitGroup.Done()

			motor := pixy.MotorType(n)
			speed := float32(n)*10 + 0.5

			if err := sim.Send(v2MotorSpeedReport(motor, speed, pixy.MotorMCUIface)); err != nil {
				errCh <- err

				return
			}

			query := append(pixy.V2GetMotorSpeed.Bytes(), byte(motor))

			resp, err := sim.SendRecv(context.Background(), query)
			if err != nil {
				errCh <- err

				return
			}

			reading, err := pixy.ParseMotorSpeedResponse(pixy.V2GetMotorSpeed, resp)
			if err != nil {
				errCh <- err

				return
			}

			if reading.Motor != motor || reading.Speed != speed {
				errCh <- errors.New("round-trip mismatch")
			}
		}(i)
	}

	waitGroup.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent round-trip: %v", err)
	}
}

func appendF32s(b []byte, fs ...float32) []byte {
	for _, f := range fs {
		b = binary.LittleEndian.AppendUint32(b, math.Float32bits(f))
	}

	return b
}
