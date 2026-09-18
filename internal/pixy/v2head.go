package pixy

import (
	"encoding/binary"
	"fmt"
	"math"
)

// V2Head is the official EMEET STUDIO HID command header, decoded from the
// binary's 162 EMHidCmdV2Head static initializers (tools/emhid/cmdtable.json,
// docs/hid-protocol-official-map.md §3.5).
//
// Wire layout: [Report, Dev, Cat, ID].
//
//   - Report: 0x09 for the V2 command surface.
//   - Dev: logical sub-device (0x00 power … 0x0a SD card).
//   - Cat: category. Large legacy values follow mergeType = (Dev<<5)|Func.
//   - ID: command ID within (Dev, Cat). SET and GET are distinct IDs, not a
//     flag bit (SET_MOTOR_SPEED=3 vs GET_MOTOR_SPEED=19).
type V2Head [4]byte

// Report prefix shared by the config/commit dialect and the V2 command surface.
const V2ReportPrefix byte = 0x09

// V2 device (Dev byte) values, from the extracted table.
const (
	V2DevPower  byte = 0x00 // battery, charge, power management
	V2DevDevice byte = 0x01 // identity, mode, factory reset
	V2DevPrivacy byte = 0x02 // privacy timing, light, GET_DEVICE_MODE
	V2DevMotor  byte = 0x03 // motor / PTZ
	V2DevOptics byte = 0x04 // focus/ev/wb, target+object track, gesture
	V2DevAudio  byte = 0x05 // audio DSP
)

// MotorMCUIface is the on-wire iface byte the official app substitutes for
// motor commands: mergeType(3,3) = (3<<5)|3. Motor SET/GET heads exist in both
// the logical (Dev=0x03) and the motor-MCU (0x63) routing; which one the wired
// PIXY firmware answers is a pending hardware verification (plan M27).
const MotorMCUIface byte = 0x63

// Known V2 command heads used by emeet-pixyd. Bytes are from
// tools/emhid/cmdtable.json; payload layouts from official controller call
// sites (map doc §3.5). Response framing is NOT yet pinned — read paths treat
// responses as best-effort parse + raw fallback until hardware verification.
var (
	// V2SetMotorSpeed: payload [motorType:u8][speed:f32 LE].
	V2SetMotorSpeed = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x03}
	// V2GetMotorSpeed: response carries value + limit (two f32).
	V2GetMotorSpeed = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x13}
	// V2SetMotorPos: payload [motorType:u8][pos:f32 LE].
	V2SetMotorPos = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x01}
	// V2GetMotorPos: response carries pan/tilt/zoom (f32×3).
	V2GetMotorPos = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x02}
	// V2SetMotorPresetPos: payload [slot:u8].
	V2SetMotorPresetPos = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x19}
	// V2SetMotorPresetPosMode: payload [slot:u8][mode:u8].
	V2SetMotorPresetPosMode = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x16}
	// V2GetMotorPresetPosMode: response [slot][mode][pan f32][tilt f32][zoom f32].
	V2GetMotorPresetPosMode = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x17}
	// V2SetTargetTrack: payload [mode:u8][f32×3] (13 bytes).
	V2SetTargetTrack = V2Head{V2ReportPrefix, V2DevOptics, 0x01, 0x01}
	// V2GetTargetTrack: response [mode:u8][f32×3].
	V2GetTargetTrack = V2Head{V2ReportPrefix, V2DevOptics, 0x01, 0x02}
	// V2GetBatteryLevel: response [level:u8] percent.
	V2GetBatteryLevel = V2Head{V2ReportPrefix, V2DevPower, 0x00, 0x02}
	// V2GetChargeSta: response [sta:u8] ChargeSta enum.
	V2GetChargeSta = V2Head{V2ReportPrefix, V2DevPower, 0x00, 0x06}
	// V2GetFuncSta: response [u32 LE] capability bitfield.
	V2GetFuncSta = V2Head{V2ReportPrefix, V2DevDevice, 0x00, 0x0d}
	// V2GetSN: response carries the serial number (string encoding unpinned).
	V2GetSN = V2Head{V2ReportPrefix, V2DevDevice, 0x00, 0x04}
	// V2GetVer: response [u16 LE] firmware version.
	V2GetVer = V2Head{V2ReportPrefix, V2DevDevice, 0x00, 0x05}
	// V2GetDeviceVer: response [u16 LE] device version.
	V2GetDeviceVer = V2Head{V2ReportPrefix, V2DevDevice, 0x00, 0x0f}
	// V2GetDeviceMode is the authoritative mode query on Dev 0x02 (our
	// empirical query reuses the SET head on Dev 0x01).
	V2GetDeviceMode = V2Head{V2ReportPrefix, V2DevPrivacy, 0x01, 0x00}
)

// MotorType selects a PTZ axis in the official motor commands.
//
// VALUE ASSIGNMENT IS AN ASSUMPTION pending hardware verification (plan M27):
// pan/tilt/zoom → 0/1/2 follows the axis order of SetMotorRunning(f32,f32,f32)
// and every other pan-first signature in the official surface. If M27 shows a
// different mapping, only these constants change.
type MotorType byte

const (
	MotorPan  MotorType = 0
	MotorTilt MotorType = 1
	MotorZoom MotorType = 2
)

func (m MotorType) Valid() bool { return m <= MotorZoom }

func (m MotorType) String() string {
	switch m {
	case MotorPan:
		return "pan"
	case MotorTilt:
		return "tilt"
	case MotorZoom:
		return "zoom"
	default:
		return fmt.Sprintf("motor(%d)", byte(m))
	}
}

// MotorTypeFromAxis maps a pixy.Axis to its motor type.
func MotorTypeFromAxis(axis Axis) (MotorType, bool) {
	switch axis {
	case AxisPan:
		return MotorPan, true
	case AxisTilt:
		return MotorTilt, true
	case AxisZoom:
		return MotorZoom, true
	default:
		return 0, false
	}
}

// Bytes returns a copy of the head suitable for passing to HID Send/SendRecv.
func (h V2Head) Bytes() []byte { return append([]byte(nil), h[:]...) }

// WithIface returns the head with the Dev byte replaced — used to route motor
// commands through the motor-MCU iface (0x63) as the official app does.
func (h V2Head) WithIface(iface byte) V2Head {
	out := h
	out[1] = iface
	return out
}

// MotorSpeedPayload builds the [motorType:u8][speed:f32 LE] payload for
// V2SetMotorSpeed / V2SetMotorPos.
func MotorSpeedPayload(motor MotorType, speed float32) []byte {
	out := make([]byte, 5)
	out[0] = byte(motor)
	binary.LittleEndian.PutUint32(out[1:], motorF32Bits(speed))
	return out
}

// ParseMotorSpeedResponse reads a [motorType:u8][speed:f32][limit:f32]
// GetMotorSpeed response payload (framing assumption: the head is echoed in
// front of the payload — pinned at hardware verification).
func ParseMotorSpeedResponse(resp []byte) (motor MotorType, speed, limit float32, err error) {
	if len(resp) < 4+9 {
		return 0, 0, 0, fmt.Errorf("motor speed response: too short (%d bytes)", len(resp))
	}

	motor = MotorType(resp[4])
	if !motor.Valid() {
		return 0, 0, 0, fmt.Errorf("motor speed response: invalid motor type %d", resp[4])
	}

	speed = f32FromBits(binary.LittleEndian.Uint32(resp[5:]))
	limit = f32FromBits(binary.LittleEndian.Uint32(resp[9:]))
	return motor, speed, limit, nil
}

func motorF32Bits(f float32) uint32 { return math.Float32bits(f) }
func f32FromBits(b uint32) float32   { return math.Float32frombits(b) }
