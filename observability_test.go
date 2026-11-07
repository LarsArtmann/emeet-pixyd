//go:build linux

package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// counterValue reads an int64 counter from the shared test registry, 0 when
// absent (OTel only exports a metric after its first write, so a never-touched
// counter legitimately reads as 0). All tests touching the global metrics
// registry run serially (nolint:paralleltest) like TestUpdateMetrics.
func counterValue(t *testing.T, name string, attrs ...attribute.KeyValue) int64 {
	t.Helper()
	registerMetrics()

	var rm metricdata.ResourceMetrics

	err := collectMetrics(context.Background(), &rm)
	if err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("%s: not an int64 counter", name)
			}

			for _, dp := range sum.DataPoints {
				if matchAttrs(dp.Attributes, attrs) {
					return dp.Value
				}
			}
		}
	}

	return 0
}

// tickBreakerGauge reproduces the poll loop's metrics tick (autoManage): the
// breaker gauge derives from hidFailCount >= threshold on every tick — the
// strike path itself writes only at the trip site. Tests must tick before
// asserting instead of trusting registry residue from earlier tests.
func tickBreakerGauge(d *Daemon) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	updateMetrics(d.state, d.videoDev != "", d.hidFailCount >= hidCircuitBreakerThreshold)
}

//nolint:paralleltest // global metrics registry
func TestBreakerStateMetric_MirrorsAccounting(t *testing.T) {
	registerMetrics()

	d := newTestDaemon(t, pixy.DesiredIdle, "", "", withConfig(t.TempDir()))

	// Strikes below the threshold keep the breaker closed.
	d.recordHIDSendFailure(context.Background())
	d.recordHIDSendFailure(context.Background())
	tickBreakerGauge(d)
	requireGaugeValue(t, "emeet_pixyd_breaker_state", 0)

	// The third strike trips it — recorded immediately.
	d.recordHIDSendFailure(context.Background())
	requireGaugeValue(t, "emeet_pixyd_breaker_state", 1)

	// A probe that finds the device present resets the strikes; the gauge
	// re-derives from the strike count on the next updateMetrics tick
	// (the poll loop's shape — see autoManage).
	d.mu.Lock()
	d.applyProbeResultLocked(probeResult{VideoDev: testVideoDev, HidrawDev: testHIDDev})
	d.mu.Unlock()

	tickBreakerGauge(d)

	requireGaugeValue(t, "emeet_pixyd_breaker_state", 0)
}

// failingDepsCommander fails LookPath for every binary, standing in for a
// host without the optional helpers.
type failingDepsCommander struct {
	noopCommandRunner
}

func (failingDepsCommander) LookPath(string) (string, error) {
	return "", errors.New("not found")
}

// failingTrackingFn is a setTracking dep that always fails, for driving the
// auto-error counter through the real handleCallStart path.
func failingTrackingFn(context.Context, pixy.CameraState) error {
	return errors.New("tracking failed")
}

// withFailingTracking wires the always-failing setTracking dep.
func withFailingTracking() testDaemonOption {
	return func(d *Daemon) {
		d.deps.setTracking = failingTrackingFn
	}
}

//nolint:paralleltest // global metrics registry
func TestCheckExternalDeps_MarksDegradedPerMissingBinary(t *testing.T) {
	d := newTestDaemon(t, pixy.DesiredIdle, "", "", withConfig(t.TempDir()))
	d.deps.commander = failingDepsCommander{}

	d.checkExternalDeps()

	d.mu.RLock()
	reasons := d.degradedReasonsSortedLocked()
	d.mu.RUnlock()

	want := []string{
		degradedExternalDepPrefix + ffmpegBin,
		degradedExternalDepPrefix + notifySend,
		degradedExternalDepPrefix + v4l2ctl,
		degradedExternalDepPrefix + wpctl,
	}

	if len(reasons) != len(want) {
		t.Fatalf("degraded reasons = %v, want exactly %v", reasons, want)
	}

	for i, reason := range reasons {
		if reason != want[i] {
			t.Errorf("degraded reason[%d] = %q, want %q", i, reason, want[i])
		}
	}
}

//nolint:paralleltest // global metrics registry
func TestAutoErrorsMetric_CountsFailedOperations(t *testing.T) {
	registerMetrics()

	before := counterValue(t, "emeet_pixyd_auto_errors_total", attribute.String("op", "track"))

	// setAudio + findSource deps stay noop (no error); only tracking fails,
	// so exactly one op=track error must be counted.
	d := newTestDaemon(
		t,
		pixy.DesiredPrivacy,
		"",
		"",
		withConfig(t.TempDir()),
		withFailingTracking(),
		withNoopAudio(),
		withFindSource("test-source"),
	)

	d.handleCallStart(context.Background(), pixy.DesiredPrivacy, pixy.AutoFull)

	after := counterValue(t, "emeet_pixyd_auto_errors_total", attribute.String("op", "track"))
	if after != before+1 {
		t.Errorf("auto_errors_total{op=track} = %d after one failed activation, want %d", after, before+1)
	}
}

//nolint:paralleltest // global metrics registry
func TestAcceptLoopFailureMetric_CountsEachFailure(t *testing.T) {
	registerMetrics()

	before := counterValue(t, "emeet_pixyd_socket_accept_failures_total")

	d := newTestDaemon(t, pixy.DesiredIdle, testVideoDev, testHIDDev, withConfig(t.TempDir()))
	d.config.StateDir = shortSocketDir(t)

	inner, listenErr := d.newUnixListener()
	if listenErr != nil {
		t.Fatalf("newUnixListener: %v", listenErr)
	}

	flaky := &flakyListener{inner: inner, failCount: 3}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	done := make(chan struct{})

	go func() {
		d.acceptUnixLoop(ctx, flaky)
		close(done)
	}()

	waited := waitFor(t, 5*time.Second, func() bool {
		return flaky.failures.Load() >= 3
	})
	if !waited {
		t.Fatal("flaky listener never exhausted its failures")
	}

	cancel()
	<-done

	after := counterValue(t, "emeet_pixyd_socket_accept_failures_total")
	if after < before+3 {
		t.Errorf("socket_accept_failures_total = %d after 3 failures, want >= %d", after, before+3)
	}
}
