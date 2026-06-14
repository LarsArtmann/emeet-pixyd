//go:build linux

package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockConn struct {
	setDeadlineErr error
}

func (m *mockConn) Read([]byte) (int, error) { return 0, nil }

func (m *mockConn) Write([]byte) (int, error) { return 0, nil }

func (m *mockConn) Close() error { return nil }

func (m *mockConn) LocalAddr() net.Addr { return nil }

func (m *mockConn) RemoteAddr() net.Addr { return nil }

func (m *mockConn) SetDeadline(time.Time) error { return m.setDeadlineErr }

func (m *mockConn) SetReadDeadline(time.Time) error { return nil }

func (m *mockConn) SetWriteDeadline(time.Time) error { return nil }

func writeFakeFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)

	dirErr := os.MkdirAll(dir, 0o755)
	if dirErr != nil {
		t.Fatalf("mkdir %s: %v", dir, dirErr)
	}

	writeErr := os.WriteFile(path, []byte(content), 0o644)
	if writeErr != nil {
		t.Fatalf("write %s: %v", path, writeErr)
	}
}

type fakeVideoDev struct {
	name    string
	product string // uevent PRODUCT=vendor/product/version
	index   string
}

type fakeHidrawDev struct {
	name    string
	hidID   string
	hidName string
}

func createFakeVideo4linux(t *testing.T, root string, devices []fakeVideoDev) {
	t.Helper()

	for _, dev := range devices {
		base := filepath.Join(root, dev.name)

		content := "DEVTYPE=usb_interface\n"
		content += "DRIVER=uvcvideo\n"
		content += "PRODUCT=" + dev.product + "\n"
		writeFakeFile(t, filepath.Join(base, "device/uevent"), content)

		if dev.index != "" {
			writeFakeFile(t, filepath.Join(base, "index"), dev.index)
		}
	}
}

func createFakeHidraw(t *testing.T, root string, devices []fakeHidrawDev) {
	t.Helper()

	for _, dev := range devices {
		ueventPath := filepath.Join(root, dev.name, "device/uevent")
		content := "DRIVER=hid-generic\n"
		content += "HID_ID=" + dev.hidID + "\n"
		content += "HID_NAME=" + dev.hidName + "\n"

		writeFakeFile(t, ueventPath, content)
	}
}

func testV4L2ProbesNothing(t *testing.T, devices []fakeVideoDev) {
	t.Helper()

	root := t.TempDir()
	if len(devices) > 0 {
		createFakeVideo4linux(t, root, devices)
	}

	result := probeVideo4linux(root)
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}
