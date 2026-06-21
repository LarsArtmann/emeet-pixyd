//go:build linux

package main

import (
	"context"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Dependencies holds all external function dependencies for the daemon.
// Tests override individual fields; production wiring happens in NewDaemon.
type Dependencies struct {
	commander     CommandRunner
	isCameraInUse func(videoDev string) bool
	findSource    func(ctx context.Context) (pixy.SourceID, error)
	setSource     func(ctx context.Context, sourceID pixy.SourceID)
	notify        func(ctx context.Context, title, body string)
	setTracking   func(ctx context.Context, state pixy.CameraState) error
	setAudio      func(ctx context.Context, mode pixy.AudioMode) error
	setGesture    func(ctx context.Context, enabled bool) error
	centerCamera  func(ctx context.Context) error
	v4l2Set       func(ctx context.Context, dev, ctrl, val string) error
	parsePTZ      func(ctx context.Context, dev string) pixy.PTZValues
}
