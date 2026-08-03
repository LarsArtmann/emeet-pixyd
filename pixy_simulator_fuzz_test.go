//go:build linux

package main

import (
	"testing"
)

// FuzzHandleConfigAndCommit feeds random byte slices through the PIXY protocol
// state machine's handleConfig and handleCommit handlers. The invariant is
// simple: no input should ever cause a panic. Valid configs followed by valid
// commits must successfully apply state; everything else must return an error.
func FuzzHandleConfigAndCommit(f *testing.F) {
	seeds := [][]byte{
		nil,
		{},
		{0x00},
		{0x09},
		{0x09, 0x01},
		pixyConfig(hidInterfaceTracking, hidByteTracking),
		pixyConfig(hidInterfaceAudio, hidByteLive),
		pixyConfig(hidInterfaceGesture, gestureEnabledByte),
		pixyCommit(hidInterfaceTracking),
		pixyCommit(hidInterfaceAudio),
		pixyCommit(hidInterfaceGesture),
		make([]byte, 32),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		t.Parallel()

		state := newPixyProtocolState()

		// The simulator's Send method distinguishes config from commit via
		// isCommitReport. We replicate that routing here so the fuzz exercises
		// the same code paths the simulator uses.
		var err error

		if isCommitReport(data) {
			err = state.handleCommit(data)
		} else {
			err = state.handleConfig(data)
		}

		if err != nil {
			return
		}

		// If handleConfig succeeded, the pending map must have an entry for
		// the interface byte at data[1].
		if !isCommitReport(data) && len(data) >= 2 {
			iface := data[1]

			state.mu.Lock()
			_, exists := state.pending[iface]
			state.mu.Unlock()

			if !exists {
				t.Errorf("handleConfig succeeded but pending[%02x] is empty", iface)
			}
		}

		// If handleCommit succeeded, the pending entry must have been deleted.
		if isCommitReport(data) && len(data) >= 2 {
			iface := data[1]

			state.mu.Lock()
			_, exists := state.pending[iface]
			state.mu.Unlock()

			if exists {
				t.Errorf("handleCommit succeeded but pending[%02x] was not deleted", iface)
			}
		}
	})
}
