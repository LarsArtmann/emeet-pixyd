//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Identity read surface (TODO #151): serial number, versions, and the
// capability bitfield, via the official GET heads extracted from the EMEET
// STUDIO binary. Every query is best-effort — an unanswered head simply
// omits its line, because response framing is still an assumption pinned at
// the hardware session (plan M27).

type identityInfo struct {
	Serial     string
	Version    uint16
	DeviceVer  uint16
	FuncStatus uint32
}

// identityStatus queries all identity heads and reports which answered.
func (d *Daemon) identityStatus(ctx context.Context) (identityInfo, []bool) {
	info := identityInfo{}
	ok := make([]bool, 4)

	if sn, err := d.queryIdentityString(pixy.V2GetSN); err == nil {
		info.Serial = sn
		ok[0] = true
	}

	if ver, err := d.queryIdentityU16(pixy.V2GetVer); err == nil {
		info.Version = ver
		ok[1] = true
	}

	if ver, err := d.queryIdentityU16(pixy.V2GetDeviceVer); err == nil {
		info.DeviceVer = ver
		ok[2] = true
	}

	if sta, err := d.queryIdentityU32(pixy.V2GetFuncSta); err == nil {
		info.FuncStatus = sta
		ok[3] = true
	}

	return info, ok
}

func (d *Daemon) queryIdentityString(head pixy.V2Head) (string, error) {
	resp, err := d.v2Read(ctxOrBackground(nil), head)
	if err != nil {
		return "", fmt.Errorf("identity string %x: %w", head, err)
	}

	return pixy.ParseString(resp)
}

func (d *Daemon) queryIdentityU16(head pixy.V2Head) (uint16, error) {
	resp, err := d.v2Read(ctxOrBackground(nil), head)
	if err != nil {
		return 0, fmt.Errorf("identity u16 %x: %w", head, err)
	}

	return pixy.ParseU16(resp)
}

func (d *Daemon) queryIdentityU32(head pixy.V2Head) (uint32, error) {
	resp, err := d.v2Read(ctxOrBackground(nil), head)
	if err != nil {
		return 0, fmt.Errorf("identity u32 %x: %w", head, err)
	}

	return pixy.ParseU32(resp)
}

// ctxOrBackground is a compile-time guard used by identity queries that
// receive an explicit context from identityStatus.
func ctxOrBackground(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}

	return context.Background()
}
