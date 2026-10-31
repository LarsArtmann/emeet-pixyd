//go:build linux

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// flakyListener fails its first failCount accepts, then recovers and serves
// the inner listener — the production shape of a transient accept storm.
type flakyListener struct {
	inner     net.Listener
	failures  atomic.Int32
	failCount int32
}

func (f *flakyListener) Accept() (net.Conn, error) {
	if f.failures.Add(1) <= f.failCount {
		return nil, errors.New("boom")
	}

	return f.inner.Accept()
}

func (f *flakyListener) Close() error   { return f.inner.Close() }
func (f *flakyListener) Addr() net.Addr { return f.inner.Addr() }

func getJSON(t *testing.T, url string, target any) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp := doGet(t, ctx, url)
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

// TestDegraded_HealthAndStatusCarryDimension pins the #191b contract: both
// machine surfaces carry `degraded` + `reasons` (absent when healthy), and
// degradation does not flip /api/health's status code — it is a truth field,
// not a liveness flip.
func TestDegraded_HealthAndStatusCarryDimension(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev)
	webSrv := &webServer{daemon: d}
	server := httptest.NewServer(newWebMux(webSrv))
	t.Cleanup(server.Close)

	var healthy struct {
		Degraded bool     `json:"degraded"`
		Reasons  []string `json:"reasons"`
	}
	getJSON(t, server.URL+"/api/health", &healthy)

	if healthy.Degraded || len(healthy.Reasons) > 0 {
		t.Errorf("healthy daemon must not be degraded: %+v", healthy)
	}

	var healthyStatus struct {
		Degraded bool     `json:"degraded"`
		Reasons  []string `json:"reasons"`
	}
	getJSON(t, server.URL+"/api/status", &healthyStatus)

	if healthyStatus.Degraded || len(healthyStatus.Reasons) > 0 {
		t.Errorf("healthy daemon must not be degraded: %+v", healthyStatus)
	}

	d.markDegraded(degradedWebServer)
	d.markDegraded(degradedSocketAccept)

	var degradedHealth struct {
		Status   string   `json:"status"`
		Degraded bool     `json:"degraded"`
		Reasons  []string `json:"reasons"`
	}
	getJSON(t, server.URL+"/api/health", &degradedHealth)

	if !degradedHealth.Degraded {
		t.Error("expected degraded=true after listener losses")
	}

	if !slices.Equal(degradedHealth.Reasons, []string{degradedSocketAccept, degradedWebServer}) {
		t.Errorf("expected sorted reasons, got %v", degradedHealth.Reasons)
	}

	var degradedStatus struct {
		Degraded bool     `json:"degraded"`
		Reasons  []string `json:"reasons"`
	}
	getJSON(t, server.URL+"/api/status", &degradedStatus)

	if !degradedStatus.Degraded || !slices.Equal(degradedStatus.Reasons, degradedHealth.Reasons) {
		t.Errorf("expected /api/status to mirror the degraded dimension: %+v", degradedStatus)
	}
}

// TestDegraded_PanelBannerRendered pins the morph-safe banner placement
// (outside #status-panel, so SSE morphs never reset it).
func TestDegraded_PanelBannerRendered(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev)
	webSrv := &webServer{daemon: d}
	server := httptest.NewServer(newWebMux(webSrv))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	healthy := doGet(t, ctx, server.URL+"/")
	defer healthy.Body.Close()

	healthyBody, err := io.ReadAll(healthy.Body)
	if err != nil {
		t.Fatalf("read healthy body: %v", err)
	}

	if strings.Contains(string(healthyBody), "degraded-banner") {
		t.Error("healthy panel must not render the degraded banner")
	}

	d.markDegraded(degradedWebServer)

	degraded := doGet(t, ctx, server.URL+"/")
	defer degraded.Body.Close()

	body, err := io.ReadAll(degraded.Body)
	if err != nil {
		t.Fatalf("read degraded body: %v", err)
	}

	if !strings.Contains(string(body), "degraded-banner") || !strings.Contains(string(body), degradedWebServer) {
		t.Errorf("degraded panel must render the banner with the reason, got: %.300s", body)
	}
}

// TestAcceptLoop_CeilingMarksDegradedHealthyAcceptClears pins #191b's
// accept-loop behavior: the ceiling marks the daemon degraded (loop keeps
// running), and the first healthy accept after the storm clears the mark.
func TestAcceptLoop_CeilingMarksDegradedHealthyAcceptClears(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, testHIDDev)
	d.config.StateDir = shortSocketDir(t)

	inner, listenErr := d.newUnixListener()
	if listenErr != nil {
		t.Fatalf("newUnixListener: %v", listenErr)
	}

	// Fail past the ceiling, then recover: marks degraded, then serves.
	listener := &flakyListener{inner: inner, failCount: int32(acceptFailureCeiling + 2)}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go d.acceptUnixLoop(ctx, listener)

	marked := waitFor(t, 5*time.Second, func() bool {
		snap := d.snapshot()

		return slices.Equal(snap.DegradedReasons, []string{degradedSocketAccept})
	})
	if !marked {
		t.Fatal("accept ceiling never marked the daemon degraded")
	}

	cmdCtx, cmdCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cmdCancel()

	if _, err := pixy.SendCommand(cmdCtx, d.config.SocketPath(), "status"); err != nil {
		t.Fatalf("send command over recovered listener: %v", err)
	}

	cleared := waitFor(t, 3*time.Second, func() bool {
		return !slices.Contains(d.snapshot().DegradedReasons, degradedSocketAccept)
	})
	if !cleared {
		t.Fatal("healthy accept never cleared the socket-accept degradation")
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}

		time.Sleep(10 * time.Millisecond)
	}

	return cond()
}
