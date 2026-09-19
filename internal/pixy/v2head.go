package pixy

import (
	"encoding/binary"
	"errors"
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

// V2ReportPrefix is the report head shared by the config/commit dialect and
// the V2 command surface.
const V2ReportPrefix byte = 0x09

// V2 device (Dev byte) values, from the extracted table.
const (
	V2DevPower   byte = 0x00 // battery, charge, power management
	V2DevDevice  byte = 0x01 // identity, mode, factory reset
	V2DevPrivacy byte = 0x02 // privacy timing, light, GET_DEVICE_MODE
	V2DevMotor   byte = 0x03 // motor / PTZ
	V2DevOptics  byte = 0x04 // focus/ev/wb, target+object track, gesture
	V2DevAudio   byte = 0x05 // audio DSP
)

// MotorMCUIface is the on-wire iface byte the official app substitutes for
// motor commands: mergeType(3,3) = (3<<5)|3. Motor SET/GET heads exist in both
// the logical (Dev=0x03) and the motor-MCU (0x63) routing; which one the wired
// PIXY firmware answers is a pending hardware verification (plan M27).
const MotorMCUIface byte = 0x63

var (
	// ErrV2ResponseShort is returned when a V2 read response is too short to
	// carry the documented payload.
	ErrV2ResponseShort = errors.New("v2 response too short")
	// ErrV2ResponseHeadMismatch is returned when a V2 response does not echo
	// the request head (official parsers reject those; so do we).
	ErrV2ResponseHeadMismatch = errors.New("v2 response head mismatch")
	// ErrInvalidMotorType is returned when a motor byte is not a valid
	// pixy.MotorType.
	ErrInvalidMotorType = errors.New("invalid motor type")
	// ErrInvalidChargeStatus is returned when a charge-status byte is not a
	// known ChargeStatus value.
	ErrInvalidChargeStatus = errors.New("invalid charge status")
)

// Known V2 command heads used by emeet-pixyd.
//
// EVIDENCE GRADES (see docs/hid-protocol-official-map.md §3.5):
//   - Head bytes: statically evidenced, 2.0.3 Mac cmdtable
//     (tools/emhid/cmdtable.json); 108/162 of those heads independently
//     confirmed byte-for-byte in the 2.0.0-Beta.25 x64 build under the
//     version-shift model (2.0.3 = Beta.25 IDs + 1 after two insertions).
//   - Response framing: statically evidenced, Beta.25 x64 parser disassembly
//     (see V2ResponsePayloadOffset) — hardware confirmation pending (#166).
//   - Per-command payload layouts: map doc §3.5 call sites; graded per
//     constant below where stronger evidence exists.
//
//nolint:gochecknoglobals // protocol constants; [4]byte cannot be Go consts
var (
	// V2SetMotorSpeed: payload [motorType:u8][speed:f32 LE].
	V2SetMotorSpeed = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x03}
	// V2GetMotorSpeed: response [motorType:u8][speed:f32][limit:f32]. Note
	// (static, Beta.25 x64 send site @0x14017ecad): Beta.25 queries speed by
	// riding the SET head (9,3,1,3) with the dev byte replaced by
	// mergeType(3,3)=0x63 and the motorType byte appended; the dedicated GET
	// head below is the 2.0.3 insertion (version-shift model) — which head the
	// wired firmware answers is a #166 pin.
	V2GetMotorSpeed = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x13}
	// V2SetMotorPos: payload [motorType:u8][pos:f32 LE].
	V2SetMotorPos = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x01}
	// V2GetMotorPos: response carries pan/tilt/zoom (f32×3).
	V2GetMotorPos = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x02}
	// V2SetMotorPresetPos: payload [slot:u8].
	V2SetMotorPresetPos = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x19}
	// V2SetMotorPresetPosMode: payload [slot:u8][mode:u8].
	V2SetMotorPresetPosMode = V2Head{V2ReportPrefix, V2DevMotor, 0x01, 0x16}
	// V2GetMotorPresetPosMode: queried with head+[slot]; response is
	// mode-only ([mode:u8], min 9) per the Beta.25 x64 GET parser, while the
	// SET_MOTOR_PRESET_POS_MODE echo response carries the full
	// [slot:u8][mode:u8][pan f32][tilt f32][zoom f32] shape — both are
	// accepted by ParseMotorPresetPosResponse (#166 pins which the wired
	// firmware answers).
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

// V2PayloadSize is the byte count of a [motorType:u8][speed:f32 LE] payload.
const V2PayloadSize = 5

// V2ResponsePayloadOffset is where the payload starts in a V2 GET response.
// EVIDENCE (static, Beta.25 x64 parser disassembly, TODO #166 for hardware):
// every official CMD_*_VAL parser validates that the response's first four
// bytes ECHO the request head, requires len >= 9, and reads the first
// payload byte at offset 8 — bytes 4..7 are a reserved dword of unknown
// meaning. Our previous offset-4 assumption is provably wrong against this;
// the offset, the head-echo check, and the simulator's response builder all
// use this constant so they evolve together.
const V2ResponsePayloadOffset = 8

// v2Payload validates the response framing and returns the payload slice:
// a 4-byte request-head echo, the reserved dword (bytes 4..7), then the
// payload. need is the required payload length in bytes.
func v2Payload(want V2Head, resp []byte, need int) ([]byte, error) {
	if len(resp) < len(want) {
		return nil, fmt.Errorf("v2 response %d bytes: %w", len(resp), ErrV2ResponseShort)
	}

	var echo V2Head
	copy(echo[:], resp[:len(want)])

	if echo != want {
		return nil, fmt.Errorf("v2 response head %x, want %x: %w", echo, want, ErrV2ResponseHeadMismatch)
	}

	if len(resp) < V2ResponsePayloadOffset+need {
		return nil, fmt.Errorf(
			"v2 payload %d bytes (need %d): %w",
			len(resp)-V2ResponsePayloadOffset, need, ErrV2ResponseShort,
		)
	}

	return resp[V2ResponsePayloadOffset:], nil
}

// v2MotorSpeedPayloadLen is the byte count of a GetMotorSpeed response
// payload: [motorType:u8][speed:f32][limit:f32].
const v2MotorSpeedPayloadLen = 9

// MotorSpeedPayload builds the [motorType:u8][speed:f32 LE] payload for
// V2SetMotorSpeed / V2SetMotorPos.
func MotorSpeedPayload(motor MotorType, speed float32) []byte {
	out := make([]byte, V2PayloadSize)
	out[0] = byte(motor)
	binary.LittleEndian.PutUint32(out[1:], motorF32Bits(speed))

	return out
}

// MotorSpeedReading is a parsed CMD_GET_MOTOR_SPEED response: the queried
// axis, its configured speed, and the hardware-reported speed limit.
type MotorSpeedReading struct {
	Motor MotorType
	Speed float32
	Limit float32
}

// ParseMotorSpeedResponse reads a [motorType:u8][speed:f32][limit:f32]
// GetMotorSpeed response. head is the head AS SENT (including any
// MotorMCUIface routing): official parsers validate the echo against the
// request head, and so does v2Payload.
func ParseMotorSpeedResponse(head V2Head, resp []byte) (MotorSpeedReading, error) {
	payload, err := v2Payload(head, resp, v2MotorSpeedPayloadLen)
	if err != nil {
		return MotorSpeedReading{}, err
	}

	motor := MotorType(payload[0])
	if !motor.Valid() {
		return MotorSpeedReading{}, fmt.Errorf("motor speed payload byte %d: %w", payload[0], ErrInvalidMotorType)
	}

	return MotorSpeedReading{
		Motor: motor,
		Speed: f32FromBits(binary.LittleEndian.Uint32(payload[1:])),
		Limit: f32FromBits(binary.LittleEndian.Uint32(payload[5:])),
	}, nil
}

// MotorPresetPositioned is the position-mode byte value that marks an
// occupied motor preset slot: the full response shape carries the
// pan/tilt/zoom position only when this byte is 1. EVIDENCE (static,
// Beta.25 x64 parser disasm @0x14017e330): the official parser gates the
// three floats on this byte; every other value marks an empty/invalid slot
// and their semantics are undecoded.
const MotorPresetPositioned byte = 1

// MotorPresetReading is a parsed CMD_GET_MOTOR_PRESET_POS_MODE response.
//
// Two response shapes are statically evidenced in the Beta.25 x64 build and
// both are accepted here (#166 pins which the wired firmware answers):
//
//   - Mode-only (the GET's own parser, thunk @0x14017e5d0 → shared
//     single-byte parser @0x140179640): [mode:u8], min length 9. HasPosition
//     is false — the slot's mode is known, its position is not exposed.
//   - Full shape (the SET_MOTOR_PRESET_POS_MODE echo parser @0x14017e330):
//     [slot:u8][mode:u8][pan:f32][tilt:f32][zoom:f32], min length 0x16 when
//     occupied; the floats are present only when mode==1.
type MotorPresetReading struct {
	Mode   byte
	Slot   byte
	Pan    float32
	Tilt   float32
	Zoom   float32
	hasPos bool
}

// Occupied reports whether the slot carries a stored position (mode==1).
// Empty, invalid, and undecoded mode values all read as not occupied, so the
// preset pull skips them.
func (r MotorPresetReading) Occupied() bool { return r.Mode == MotorPresetPositioned }

// HasPosition reports whether the response carried the pan/tilt/zoom floats.
// A reading can be occupied (mode==1) without a position when the device
// answered with the mode-only GET shape.
func (r MotorPresetReading) HasPosition() bool { return r.hasPos }

// PTZValues converts the slot position to software preset values, rounding
// the hardware floats to the integer user-facing units and clamping to the
// V4L2 limits. Only meaningful when HasPosition is true.
func (r MotorPresetReading) PTZValues() PTZValues {
	return PTZValues{
		Pan:  PanRange.Clamp(int(math.Round(float64(r.Pan)))),
		Tilt: TiltRange.Clamp(int(math.Round(float64(r.Tilt)))),
		Zoom: ZoomRange.Clamp(int(math.Round(float64(r.Zoom)))),
	}
}

// v2MotorPresetFullLen is the byte count of the full (SET-echo) preset
// response payload: [slot:u8][mode:u8][pan:f32][tilt:f32][zoom:f32].
const v2MotorPresetFullLen = 14

// ParseMotorPresetPosResponse reads a GetMotorPresetPosMode response for one
// queried slot, accepting both statically evidenced shapes (see
// MotorPresetReading): a mode-only byte at offset 8, or the full
// slot+mode+position shape where the floats exist only when mode==1 — the
// same gating the official SET-echo parser applies.
func ParseMotorPresetPosResponse(head V2Head, resp []byte) (MotorPresetReading, error) {
	payload, err := v2Payload(head, resp, 1)
	if err != nil {
		return MotorPresetReading{}, err
	}

	if len(payload) < 2 {
		// Mode-only GET answer (Beta.25 GET parser: single byte at offset 8).
		return MotorPresetReading{Mode: payload[0]}, nil
	}

	reading := MotorPresetReading{
		Slot:   payload[0],
		Mode:   payload[1],
		hasPos: payload[1] == MotorPresetPositioned && len(payload) >= v2MotorPresetFullLen,
	}

	if !reading.hasPos {
		return reading, nil
	}

	reading.Pan = f32FromBits(binary.LittleEndian.Uint32(payload[2:]))
	reading.Tilt = f32FromBits(binary.LittleEndian.Uint32(payload[6:]))
	reading.Zoom = f32FromBits(binary.LittleEndian.Uint32(payload[10:]))

	return reading, nil
}

func motorF32Bits(f float32) uint32 { return math.Float32bits(f) }
func f32FromBits(b uint32) float32  { return math.Float32frombits(b) }

// ParseBatteryLevel reads a GetBatteryLevel response: [level:u8] percent.
// EVIDENCE (static, Beta.25 x64 parser disasm @0x14017a710): level is the raw
// byte at offset 8; payload layout verified instruction-by-instruction.
func ParseBatteryLevel(head V2Head, resp []byte) (int, error) {
	payload, err := v2Payload(head, resp, 1)
	if err != nil {
		return 0, err
	}

	return int(payload[0]), nil
}

// ChargeStatus is the CMD_GET_CHARGE_STA response enum.
//
// EVIDENCE (static, Beta.25 x64 consumer code @0x1403ccbf8): the official app
// computes isCharging as (sta-1) <= 1, i.e. {1,2} = charging, 0 = discharging.
// What distinguishes 1 from 2 is undecoded (no UI enum registration exists
// for ChargeSta); both constants below therefore mean "charging" until the
// hardware session (#166) separates them.
type ChargeStatus byte

const (
	ChargeDischarging ChargeStatus = 0
	ChargeCharging    ChargeStatus = 1
	ChargeChargingAlt ChargeStatus = 2
)

func (c ChargeStatus) Valid() bool { return c <= ChargeChargingAlt }

// Charging reports whether the status is any charging state ({1,2}), the
// official consumer code's exact predicate.
func (c ChargeStatus) Charging() bool { return c == ChargeCharging || c == ChargeChargingAlt }

func (c ChargeStatus) String() string {
	switch c {
	case ChargeDischarging:
		return "discharging"
	case ChargeCharging:
		return "charging"
	case ChargeChargingAlt:
		return "charging-alt"
	default:
		return fmt.Sprintf("charge(%d)", byte(c))
	}
}

// ParseChargeStatus reads a GetChargeSta response: [sta:u8] after the
// head-echo framing. EVIDENCE (static, Beta.25 x64 parser disasm
// @0x14017a7d0): the payload byte is the ChargeSta enum; values above 2 are
// not in the official consumer predicate and are rejected here.
func ParseChargeStatus(head V2Head, resp []byte) (ChargeStatus, error) {
	payload, err := v2Payload(head, resp, 1)
	if err != nil {
		return 0, err
	}

	sta := ChargeStatus(payload[0])
	if !sta.Valid() {
		return 0, fmt.Errorf("charge status %d: %w", payload[0], ErrInvalidChargeStatus)
	}

	return sta, nil
}

// TargetTrackMode selects the tracking variant of the official
// SetTargetTrack command (TODO #140).
//
// EVIDENCE (static, Beta.25 x64 UI enum registration decode @0x14027c820):
// 0 = None ("No Smart Composition"), 1 = Face, 2 = HalfBody, 3 = FullBody —
// every official UI enum in the app is 1-based with 0 = None. Our previous
// 0/1/2 face-first assignment was off by one and would have sent "none"
// where the user asked for "face". Hardware confirmation pending (#166);
// until then the mode byte is transmitted verbatim per this table.
// The command payload is [mode:u8][f32x3]; the three floats' meaning
// (sensitivity or target box) is unknown, so emeet-pixyd transmits zeros.
type TargetTrackMode byte

const (
	TrackNone     TargetTrackMode = 0
	TrackFace     TargetTrackMode = 1
	TrackHalfBody TargetTrackMode = 2
	TrackFullBody TargetTrackMode = 3
)

func (m TargetTrackMode) Valid() bool { return m <= TrackFullBody }

func (m TargetTrackMode) String() string {
	switch m {
	case TrackNone:
		return "none"
	case TrackFace:
		return "face"
	case TrackHalfBody:
		return "halfbody"
	case TrackFullBody:
		return "fullbody"
	default:
		return fmt.Sprintf("track(%d)", byte(m))
	}
}

// ParseTargetTrackMode maps CLI aliases (including the short forms
// "half"/"full") to a variant. Numeric aliases are deliberately absent:
// they encoded the disproved 0-based mapping, and silently remapping them
// would change their meaning.
func ParseTargetTrackMode(input string) (TargetTrackMode, bool) {
	switch input {
	case "none", "off":
		return TrackNone, true
	case "face":
		return TrackFace, true
	case "halfbody", "half":
		return TrackHalfBody, true
	case "fullbody", "full":
		return TrackFullBody, true
	default:
		return 0, false
	}
}

// TargetTrackPayload builds the [mode:u8][f32x3] payload for
// V2SetTargetTrack. The three floats are transmitted as zeros: their
// semantics are not decoded yet (M27-verify).
// v2TargetTrackPayloadLen is the byte count of the SetTargetTrack payload:
// [mode:u8][f32x3].
const v2TargetTrackPayloadLen = 13

func TargetTrackPayload(mode TargetTrackMode) []byte {
	out := make([]byte, v2TargetTrackPayloadLen)
	out[0] = byte(mode)

	return out
}

// ParseU16 reads a GetVer/GetDeviceVer response: [u16 LE] after the
// head-echo framing (payload shape itself still assumed, map doc §3.5).
func ParseU16(head V2Head, resp []byte) (uint16, error) {
	payload, err := v2Payload(head, resp, 2)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint16(payload), nil
}

// v2U32PayloadLen is the byte count of a u32 response payload.
const v2U32PayloadLen = 4

// ParseU32 reads a GetFuncSta response: [u32 LE] after the head-echo
// framing (payload shape itself still assumed, map doc §3.5). The
// bitfield's individual capability bits are not decoded yet.
func ParseU32(head V2Head, resp []byte) (uint32, error) {
	payload, err := v2Payload(head, resp, v2U32PayloadLen)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(payload), nil
}

// ParseString reads a GetSN-style response: printable bytes after the
// head-echo framing, terminated by NUL or end of buffer. Non-printable
// bytes truncate the string (defensive against the undecoded string
// encoding, map doc §3.5).
func ParseString(head V2Head, resp []byte) (string, error) {
	payload, err := v2Payload(head, resp, 1)
	if err != nil {
		return "", err
	}

	end := 0

	for end < len(payload) && payload[end] >= 0x20 && payload[end] != 0x7f {
		end++
	}

	if end == 0 {
		return "", fmt.Errorf("string payload empty: %w", ErrV2ResponseShort)
	}

	return string(payload[:end]), nil
}
