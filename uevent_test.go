//go:build linux

package main

import (
	"testing"
)

func TestParseUevent(t *testing.T) {
	t.Parallel()
	// ueventOf returns a complete uevent, suppressing exhaustruct linter
	// warnings since test cases intentionally omit fields irrelevant to parseUevent.
	ueventOf := func(action, subsys, devpath string) uevent {
		return uevent{Action: action, Subsys: subsys, DevPath: devpath}
	}
	tests := []struct {
		name  string
		input string
		want  uevent
	}{
		{
			name:  "video add",
			input: "ACTION=add\nSUBSYSTEM=video4linux\nDEVPATH=/devices/pci0000:00/0000:00:14.0/usb1/1-1/1-1:1.0/video4linux/video0",
			want: ueventOf(
				"add",
				"video4linux",
				"/devices/pci0000:00/0000:00:14.0/usb1/1-1/1-1:1.0/video4linux/video0",
			),
		},
		{
			name:  "hidraw remove",
			input: "ACTION=remove\nSUBSYSTEM=hidraw\nDEVPATH=/devices/pci0000:00/hidraw/hidraw0",
			want:  ueventOf("remove", "hidraw", "/devices/pci0000:00/hidraw/hidraw0"),
		},
		{
			name:  "empty input",
			input: "",

			want: uevent{},
		},
		{
			name:  "no equals sign",
			input: "GARBAGE\nANOTHER",

			want: uevent{},
		},
		{
			name:  "partial keys only",
			input: "ACTION=add\nMAJOR=81",
			want:  uevent{Action: ueventAdd},
		},
		{
			name:  "extra newlines",
			input: "\nACTION=add\n\nSUBSYSTEM=hidraw\n\n",
			want:  uevent{Action: ueventAdd, Subsys: "hidraw"},
		},
		{
			name:  "value contains equals",
			input: "ACTION=add\nDEVPATH=/path/with=equals",
			want:  uevent{Action: "add", DevPath: "/path/with=equals"},
		},
		{
			name:  "change action ignored",
			input: "ACTION=change\nSUBSYSTEM=video4linux\nDEVPATH=/dev/video0",
			want:  ueventOf("change", "video4linux", "/dev/video0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseUevent(tt.input)
			if got != tt.want {
				t.Errorf("parseUevent(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsRelevantUevent(t *testing.T) {
	t.Parallel()
	// ueventCase returns a complete uevent for isRelevantUevent tests.
	// The nolint suppresses exhaustruct warnings: test cases intentionally
	// omit DevPath since isRelevantUevent only inspects Action and Subsys.
	ueventCase := func(action, subsys string) uevent {
		return uevent{Action: action, Subsys: subsys}
	}
	// ueventCaseEmpty returns an empty uevent, suppressing the exhaustruct warning
	// for intentionally partial struct literals in tests.
	ueventCaseEmpty := func() uevent {
		return uevent{}
	}
	tests := []struct {
		name string
		evt  uevent
		want bool
	}{
		{name: "add video4linux", evt: ueventCase("add", "video4linux"), want: true},
		{name: "remove video4linux", evt: ueventCase("remove", "video4linux"), want: true},
		{name: "add hidraw", evt: ueventCase("add", "hidraw"), want: true},
		{name: "remove hidraw", evt: ueventCase("remove", "hidraw"), want: true},
		{name: "change action", evt: ueventCase("change", "video4linux"), want: false},
		{name: "wrong subsystem", evt: ueventCase("add", "net"), want: false},
		{name: "empty", evt: ueventCaseEmpty(), want: false},
		{name: "bind action", evt: ueventCase("bind", "hidraw"), want: false},
		{name: "add usb subsystem", evt: ueventCase("add", "usb"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isRelevantUevent(tt.evt); got != tt.want {
				t.Errorf("isRelevantUevent(%+v) = %v, want %v", tt.evt, got, tt.want)
			}
		})
	}
}
