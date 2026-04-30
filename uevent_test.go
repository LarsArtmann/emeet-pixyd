//go:build linux

package main

import (
	"testing"
)

func TestParseUevent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  uevent
	}{
		{
			name:  "video add",
			input: "ACTION=add\nSUBSYSTEM=video4linux\nDEVPATH=/devices/pci0000:00/0000:00:14.0/usb1/1-1/1-1:1.0/video4linux/video0",
			want: uevent{
				Action:  "add",
				Subsys:  "video4linux",
				DevPath: "/devices/pci0000:00/0000:00:14.0/usb1/1-1/1-1:1.0/video4linux/video0",
			},
		},
		{
			name:  "hidraw remove",
			input: "ACTION=remove\nSUBSYSTEM=hidraw\nDEVPATH=/devices/pci0000:00/hidraw/hidraw0",
			want: uevent{
				Action:  "remove",
				Subsys:  "hidraw",
				DevPath: "/devices/pci0000:00/hidraw/hidraw0",
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  uevent{},
		},
		{
			name:  "no equals sign",
			input: "GARBAGE\nANOTHER",
			want:  uevent{},
		},
		{
			name:  "partial keys only",
			input: "ACTION=add\nMAJOR=81",
			want: uevent{
				Action: "add",
			},
		},
		{
			name:  "extra newlines",
			input: "\nACTION=add\n\nSUBSYSTEM=hidraw\n\n",
			want: uevent{
				Action: "add",
				Subsys: "hidraw",
			},
		},
		{
			name:  "value contains equals",
			input: "ACTION=add\nDEVPATH=/path/with=equals",
			want: uevent{
				Action:  "add",
				DevPath: "/path/with=equals",
			},
		},
		{
			name:  "change action ignored",
			input: "ACTION=change\nSUBSYSTEM=video4linux\nDEVPATH=/dev/video0",
			want: uevent{
				Action:  "change",
				Subsys:  "video4linux",
				DevPath: "/dev/video0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseUevent(tt.input)
			if got != tt.want {
				t.Errorf("parseUevent(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsRelevantUevent(t *testing.T) {
	tests := []struct {
		name string
		evt  uevent
		want bool
	}{
		{
			name: "add video4linux",
			evt:  uevent{Action: "add", Subsys: "video4linux"},
			want: true,
		},
		{
			name: "remove video4linux",
			evt:  uevent{Action: "remove", Subsys: "video4linux"},
			want: true,
		},
		{
			name: "add hidraw",
			evt:  uevent{Action: "add", Subsys: "hidraw"},
			want: true,
		},
		{
			name: "remove hidraw",
			evt:  uevent{Action: "remove", Subsys: "hidraw"},
			want: true,
		},
		{
			name: "change action",
			evt:  uevent{Action: "change", Subsys: "video4linux"},
			want: false,
		},
		{
			name: "wrong subsystem",
			evt:  uevent{Action: "add", Subsys: "net"},
			want: false,
		},
		{
			name: "empty",
			evt:  uevent{},
			want: false,
		},
		{
			name: "bind action",
			evt:  uevent{Action: "bind", Subsys: "hidraw"},
			want: false,
		},
		{
			name: "add usb subsystem",
			evt:  uevent{Action: "add", Subsys: "usb"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRelevantUevent(tt.evt); got != tt.want {
				t.Errorf("isRelevantUevent(%+v) = %v, want %v", tt.evt, got, tt.want)
			}
		})
	}
}
