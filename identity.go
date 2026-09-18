//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Identity read surface (TODO #151): serial number, versions, and the
// capability bitfield, via the official GET heads extracted from the EMEET
// STUDIO binary. Every query is best-effort - an unanswered head simply
// omits its line, because response framing is still an assumption pinned at
// the hardware session (plan M27).

type identityInfo struct {
	Serial     string
	Version    uint16
	DeviceVer  uint16
	FuncStatus uint32
}

// identityQueryCount is the number of identity heads identityStatus probes.
const identityQueryCount = 4

// identityStatus queries all identity heads and reports which answered.
func (d *Daemon) identityStatus(ctx context.Context) (identityInfo, []bool) {
	info := identityInfo{
		Serial:     "",
		Version:    0,
		DeviceVer:  0,
		FuncStatus: 0,
	}

	ok := make([]bool, identityQueryCount)

	if sn, err := d.queryIdentityString(ctx, pixy.V2GetSN); err == nil {
		info.Serial = sn
		ok[0] = true
	}

	if ver, err := d.queryIdentityU16(ctx, pixy.V2GetVer); err == nil {
		info.Version = ver
		ok[1] = true
	}

	if ver, err := d.queryIdentityU16(ctx, pixy.V2GetDeviceVer); err == nil {
		info.DeviceVer = ver
		ok[2] = true
	}

	if sta, err := d.queryIdentityU32(ctx, pixy.V2GetFuncSta); err == nil {
		info.FuncStatus = sta
		ok[3] = true
	}

	return info, ok
}

//nolint:wrapcheck // pixy parse errors are already domain-wrapped
func (d *Daemon) queryIdentityString(ctx context.Context, head pixy.V2Head) (string, error) {
	resp, err := d.v2Read(ctx, head)
	if err != nil {
		return "", fmt.Errorf("identity string %x: %w", head, err)
	}

	return pixy.ParseString(resp)
}

//nolint:wrapcheck // pixy parse errors are already domain-wrapped
func (d *Daemon) queryIdentityU16(ctx context.Context, head pixy.V2Head) (uint16, error) {
	resp, err := d.v2Read(ctx, head)
	if err != nil {
		return 0, fmt.Errorf("identity u16 %x: %w", head, err)
	}

	return pixy.ParseU16(resp)
}

//nolint:wrapcheck // pixy parse errors are already domain-wrapped
func (d *Daemon) queryIdentityU32(ctx context.Context, head pixy.V2Head) (uint32, error) {
	resp, err := d.v2Read(ctx, head)
	if err != nil {
		return 0, fmt.Errorf("identity u32 %x: %w", head, err)
	}

	return pixy.ParseU32(resp)
}

// formatIdentity renders the answered identity fields as additional
// key=value tokens for the `device` command output. Fields that did not
// answer are omitted entirely - output stays stable on devices that ignore
// these heads.
func formatIdentity(info identityInfo, ok []bool) []string {
	tokens := make([]string, 0, identityQueryCount)

	if ok[0] {
		tokens = append(tokens, "sn="+info.Serial)
	}

	if ok[1] {
		tokens = append(tokens, fmt.Sprintf("ver=0x%04x", info.Version))
	}

	if ok[2] {
		tokens = append(tokens, fmt.Sprintf("devver=0x%04x", info.DeviceVer))
	}

	if ok[3] {
		tokens = append(tokens, fmt.Sprintf("func=0x%08x", info.FuncStatus))
	}

	return tokens
}
