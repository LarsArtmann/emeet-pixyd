package pixy

import (
	id "github.com/larsartmann/go-branded-id"
)

type pidBrand struct{}

// PID is a branded process identifier.
type PID = id.ID[pidBrand, int]

// NewPID creates a new branded PID.
func NewPID(v int) PID { return id.NewID[pidBrand](v) }

type sourceIDBrand struct{}

// SourceID is a branded PipeWire audio source identifier.
type SourceID = id.ID[sourceIDBrand, string]

// NewSourceID creates a new branded SourceID.
func NewSourceID(v string) SourceID { return id.NewID[sourceIDBrand](v) }
