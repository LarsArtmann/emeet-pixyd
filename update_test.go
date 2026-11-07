//go:build linux

package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestIsNewerRelease(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		current string
		latest  string
		want    bool
	}{
		{current: "v0.5.0", latest: "v0.6.0", want: true},
		{current: "0.5.0", latest: "v0.6.0", want: true},    // no v prefix on current
		{current: "v0.5", latest: "v0.5.1", want: true},     // 2-component core pads
		{current: "v0.10.0", latest: "v0.9.0", want: false}, // numeric, not lexicographic
		{current: "v0.6.0", latest: "v0.6.0", want: false},  // equal
		{current: "v0.7.0", latest: "v0.6.0", want: false},  // running newer
		{current: "v0.6.0", latest: "v0.6.0-rc1", want: false},
		{current: "v0.6.0-rc1", latest: "v0.6.0", want: true}, // pre-release < final
		{current: "v0.6.0-rc1", latest: "v0.6.0-rc2", want: false},
		{current: "dev", latest: "v0.6.0", want: false}, // unparseable current
		{current: "v0.6.0", latest: "dev", want: false}, // unparseable latest
		{current: "v0.6.0", latest: "", want: false},
	} {
		if got := isNewerRelease(tc.current, tc.latest); got != tc.want {
			t.Errorf("isNewerRelease(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestCommitSHA(t *testing.T) {
	t.Parallel()

	full := strings.Repeat("ab12", 10) // 40 hex
	if len(full) != 40 {
		t.Fatalf("test fixture length = %d, want 40", len(full))
	}

	for _, tc := range []struct {
		in     string
		want   string
		wantOK bool
	}{
		{in: full, want: full, wantOK: true},
		{in: full + "-dirty", want: full, wantOK: true},
		{in: "0c3a1b2", want: "0c3a1b2", wantOK: true},
		{in: "v0.5.0", wantOK: false},
		{in: "dev", wantOK: false},
		{in: "0c3a1b20", wantOK: false}, // 8 chars: neither short nor full
		{in: strings.Repeat("g", 40), wantOK: false},
	} {
		got, ok := commitSHA(tc.in)

		if ok != tc.wantOK {
			t.Errorf("commitSHA(%q) ok = %v, want %v", tc.in, ok, tc.wantOK)

			continue
		}

		if got != tc.want {
			t.Errorf("commitSHA(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestReleaseFromFeed(t *testing.T) {
	t.Parallel()

	t.Run("parses tag and url", func(t *testing.T) {
		t.Parallel()

		server := newJSONServer(t, http.StatusOK, `{"tag_name":"v0.6.0","html_url":"https://example.com/v0.6.0"}`)
		release, err := releaseFromFeed(context.Background(), server.URL)
		if err != nil {
			t.Fatalf("releaseFromFeed: unexpected error: %v", err)
		}

		if release.TagName != "v0.6.0" || release.HTMLURL != "https://example.com/v0.6.0" {
			t.Errorf("release = %+v, want v0.6.0 + example URL", release)
		}
	})

	t.Run("non-200 is transient", func(t *testing.T) {
		t.Parallel()

		server := newJSONServer(t, http.StatusForbidden, `{"message":"rate limited"}`)
		_, err := releaseFromFeed(context.Background(), server.URL)

		if err == nil {
			t.Fatal("releaseFromFeed: expected error for 403, got nil")
		}

		if errorfamily.Classify(err) != errorfamily.Transient {
			t.Errorf("classify = %v, want Transient", errorfamily.Classify(err))
		}
	})

	t.Run("incomplete feed is an error", func(t *testing.T) {
		t.Parallel()

		server := newJSONServer(t, http.StatusOK, `{"tag_name":"v0.6.0"}`)
		_, err := releaseFromFeed(context.Background(), server.URL)

		if err == nil {
			t.Fatal("releaseFromFeed: expected error for missing html_url, got nil")
		}
	})

	t.Run("garbage body is an error", func(t *testing.T) {
		t.Parallel()

		server := newJSONServer(t, http.StatusOK, `not-json`)
		_, err := releaseFromFeed(context.Background(), server.URL)

		if err == nil {
			t.Fatal("releaseFromFeed: expected decode error, got nil")
		}
	})
}

func TestCompareFromAPI(t *testing.T) {
	t.Parallel()

	t.Run("builds the compare URL", func(t *testing.T) {
		t.Parallel()

		var gotPath string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"behind","behind_by":3}`))
		}))
		t.Cleanup(server.Close)

		compare, err := compareFromAPI(context.Background(), server.URL+"/compare", "v0.6.0", "0c3a1b2")
		if err != nil {
			t.Fatalf("compareFromAPI: unexpected error: %v", err)
		}

		if want := "/compare/v0.6.0...0c3a1b2"; gotPath != want {
			t.Errorf("request path = %q, want %q", gotPath, want)
		}

		if compare.Status != "behind" || compare.BehindBy != 3 {
			t.Errorf("compare = %+v, want behind by 3", compare)
		}
	})

	t.Run("http error propagates", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		t.Cleanup(server.Close)

		if _, err := compareFromAPI(context.Background(), server.URL, "v0.6.0", "0c3a1b2"); err == nil {
			t.Fatal("compareFromAPI: expected error for 500, got nil")
		}
	})
}

func TestUpdateState_SetReportsChange(t *testing.T) {
	t.Parallel()

	var state updateState

	if !state.set(&availableUpdate{Version: "v0.6.0", URL: "u"}) {
		t.Error("first set should report a change")
	}

	if state.set(&availableUpdate{Version: "v0.6.0", URL: "u"}) {
		t.Error("identical set should not report a change")
	}

	if !state.set(nil) {
		t.Error("clearing should report a change")
	}

	if state.get() != nil {
		t.Error("get after clear = non-nil, want nil")
	}
}

// updateCheckFixture wires a daemon whose feed functions are fully fake: no
// test ever touches the network. current is the buildVersion the check runs
// as (buildVersion itself is "dev" in tests, so it is passed explicitly).
type updateCheckFixture struct {
	*Daemon

	current string

	release githubRelease
	err     error

	compareResult githubCompare
	compareErr    error

	releaseCalls int
	compareCalls int
}

func newUpdateCheckFixture(t *testing.T, current string) *updateCheckFixture {
	t.Helper()

	fixture := &updateCheckFixture{}
	fixture.Daemon = newTestDaemon(t, "privacy", testVideoDev, "")
	fixture.deps.latestRelease = func(context.Context) (githubRelease, error) {
		fixture.releaseCalls++

		return fixture.release, fixture.err
	}
	fixture.deps.compareRelease = func(_ context.Context, _, _ string) (githubCompare, error) {
		fixture.compareCalls++

		return fixture.compareResult, fixture.compareErr
	}

	fixture.current = current

	return fixture
}

func (f *updateCheckFixture) runCheck() {
	f.checkForUpdate(context.Background(), f.current)
}

func TestCheckForUpdate_TagChannelDetectsNewer(t *testing.T) {
	t.Parallel()

	fixture := newUpdateCheckFixture(t, "v0.5.0")
	fixture.release = githubRelease{TagName: "v0.6.0", HTMLURL: "https://example.com/rel"}

	fixture.runCheck()

	update := fixture.snapshot().Update
	if update == nil {
		t.Fatal("snapshot().Update = nil, want v0.6.0")
	}

	if update.Version != "v0.6.0" || update.URL != "https://example.com/rel" {
		t.Errorf("Update = %+v, want v0.6.0 + example URL", update)
	}

	if fixture.compareCalls != 0 {
		t.Errorf("compare calls = %d, want 0 for the tag channel", fixture.compareCalls)
	}
}

func TestCheckForUpdate_TagChannelClearsWhenNewest(t *testing.T) {
	t.Parallel()

	fixture := newUpdateCheckFixture(t, "v0.6.0")
	fixture.release = githubRelease{TagName: "v0.6.0", HTMLURL: "https://example.com/rel"}
	fixture.updates.set(&availableUpdate{Version: "v0.5.9", URL: "stale"})

	fixture.runCheck()

	if got := fixture.snapshot().Update; got != nil {
		t.Errorf("Update = %+v, want cleared when running the newest release", got)
	}
}

func TestCheckForUpdate_KeepsLastKnownOnFeedFailure(t *testing.T) {
	t.Parallel()

	fixture := newUpdateCheckFixture(t, "v0.5.0")
	fixture.updates.set(&availableUpdate{Version: "v0.5.9", URL: "known"})
	fixture.err = errorfamily.NewTransient("update.fetch", "network down")

	fixture.runCheck()

	if got := fixture.snapshot().Update; got == nil || got.Version != "v0.5.9" {
		t.Errorf("Update = %+v, want the last known v0.5.9 kept on failure", got)
	}
}

func TestCheckForUpdate_SHAChannelBehindTagDetects(t *testing.T) {
	t.Parallel()

	sha := strings.Repeat("ab12", 10) // 40 hex

	fixture := newUpdateCheckFixture(t, sha)
	fixture.release = githubRelease{TagName: "v0.6.0", HTMLURL: "https://example.com/rel"}
	fixture.compareResult = githubCompare{Status: "behind", BehindBy: 4}

	fixture.runCheck()

	update := fixture.snapshot().Update
	if update == nil || update.Version != "v0.6.0" {
		t.Fatalf("Update = %+v, want v0.6.0 for a SHA strictly behind the tag", update)
	}

	if fixture.compareCalls != 1 {
		t.Errorf("compare calls = %d, want 1", fixture.compareCalls)
	}
}

func TestCheckForUpdate_SHAChannelAheadOrDivergedDoesNotNotify(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"ahead", "identical", "diverged"} {
		sha := strings.Repeat("cd34", 10)

		fixture := newUpdateCheckFixture(t, sha)
		fixture.release = githubRelease{TagName: "v0.6.0", HTMLURL: "https://example.com/rel"}
		fixture.compareResult = githubCompare{Status: status, BehindBy: 2}
		fixture.updates.set(&availableUpdate{Version: "v0.5.9", URL: "stale"})

		fixture.runCheck()

		if got := fixture.snapshot().Update; got != nil {
			t.Errorf("status %q: Update = %+v, want cleared", status, got)
		}
	}
}

func TestCheckForUpdate_SHAChannelFailureKeepsLastKnown(t *testing.T) {
	t.Parallel()

	fixture := newUpdateCheckFixture(t, strings.Repeat("ab12", 10))
	fixture.release = githubRelease{TagName: "v0.6.0", HTMLURL: "https://example.com/rel"}
	fixture.compareErr = errors.New("compare endpoint down")
	fixture.updates.set(&availableUpdate{Version: "v0.5.9", URL: "known"})

	fixture.runCheck()

	if got := fixture.snapshot().Update; got == nil || got.Version != "v0.5.9" {
		t.Errorf("Update = %+v, want the last known v0.5.9 kept", got)
	}
}

func TestCheckForUpdate_DevBuildNeverFetches(t *testing.T) {
	t.Parallel()

	fixture := newUpdateCheckFixture(t, "dev")
	fixture.release = githubRelease{TagName: "v0.6.0", HTMLURL: "https://example.com/rel"}

	fixture.runCheck()

	if fixture.releaseCalls != 0 {
		t.Errorf("release feed calls = %d, want 0 for an uncomparable version", fixture.releaseCalls)
	}

	if fixture.snapshot().Update != nil {
		t.Error("Update = non-nil, want nil for an uncomparable version")
	}
}

// TestRunUpdateChecker_DisabledReturnsWithoutFetching pins the opt-out: with
// EMEET_PIXYD_UPDATE_CHECK=off the loop must not reach the feed at all. It
// runs synchronously — a regression that ignores the flag blocks here until
// the test timeout instead of silently fetching.
func TestRunUpdateChecker_DisabledReturnsWithoutFetching(t *testing.T) {
	t.Parallel()

	fixture := newUpdateCheckFixture(t, "v0.5.0")
	fixture.config.UpdateCheck = false
	fixture.release = githubRelease{TagName: "v0.6.0", HTMLURL: "https://example.com/rel"}

	fixture.runUpdateChecker(context.Background())

	if fixture.releaseCalls != 0 {
		t.Errorf("release feed calls = %d, want 0 while disabled", fixture.releaseCalls)
	}
}

func TestVersionCommandCarriesUpdateHint(t *testing.T) {
	t.Parallel()

	daemon := newTestDaemon(t, "privacy", "", "")
	resp := daemon.handleQueryCommand(context.Background(), []string{cmdVersion})

	if !strings.HasPrefix(resp.String(), "emeet-pixyd "+buildVersion) {
		t.Errorf("version output = %q, want %q prefix", resp.String(), "emeet-pixyd "+buildVersion)
	}

	if strings.Contains(resp.String(), "update available") {
		t.Errorf("version output = %q, want no hint before any check ran", resp.String())
	}

	daemon.updates.set(&availableUpdate{Version: "v0.6.0", URL: "https://example.com/rel"})
	resp = daemon.handleQueryCommand(context.Background(), []string{cmdVersion})

	if !strings.Contains(resp.String(), "update available: v0.6.0") {
		t.Errorf("version output = %q, want the update hint", resp.String())
	}

	if !strings.Contains(resp.String(), "https://example.com/rel") {
		t.Errorf("version output = %q, want the release URL", resp.String())
	}
}

func TestWaybarTooltipMentionsUpdate(t *testing.T) {
	t.Parallel()

	daemon := newTestDaemon(t, "privacy", testVideoDev, "")

	if out := daemon.waybarOutput(context.Background()); strings.Contains(out, "Update available") {
		t.Errorf("waybar output = %s, want no update line before any check ran", out)
	}

	daemon.updates.set(&availableUpdate{Version: "v0.6.0", URL: "https://example.com/rel"})

	if out := daemon.waybarOutput(context.Background()); !strings.Contains(out, "Update available: v0.6.0") {
		t.Errorf("waybar output = %s, want an update tooltip line", out)
	}
}

func TestWebPanel_UpdateNotice(t *testing.T) {
	t.Parallel()

	daemon := newTestDaemon(t, "privacy", testVideoDev, "")
	server := newTestWebServer(t, daemon)

	if body := getPanelBody(t, server); strings.Contains(body, "update-available") {
		t.Error("panel should not render an update notice before any check ran")
	}

	daemon.updates.set(&availableUpdate{Version: "v0.6.0", URL: "https://example.com/rel"})

	body := getPanelBody(t, server)

	assertContainsAll(t, body, []string{
		`class="last-synced update-available"`,
		`href="https://example.com/rel"`,
		`target="_blank"`,
		`rel="noopener"`,
		"v0.6.0 available",
	})
}

func TestWebStatusJSON_UpdateFields(t *testing.T) {
	t.Parallel()

	daemon := newTestDaemon(t, "privacy", testVideoDev, "")
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+"/api/status")
	body := getBody(t, resp)
	resp.Body.Close()

	assertNotContains(t, body, `"update"`, "update object omitted before any check ran")

	daemon.updates.set(&availableUpdate{Version: "v0.6.0", URL: "https://example.com/rel"})

	resp = get(t, server.URL+"/api/status")
	got := decodeStatusResponse(t, resp)
	resp.Body.Close()

	if got.Update == nil {
		t.Fatal("update = nil, want the observed release")
	}

	if got.Update.Version != "v0.6.0" {
		t.Errorf("update.version = %q, want v0.6.0", got.Update.Version)
	}

	if got.Update.URL != "https://example.com/rel" {
		t.Errorf("update.url = %q, want the release URL", got.Update.URL)
	}
}

// newJSONServer answers every request with the given status and body.
func newJSONServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// guards against accidental real-network use: the fixture must never wait on
// the production client timeout.
func TestUpdateFetchTimeoutIsBounded(t *testing.T) {
	t.Parallel()

	if updateFetchTimeout > 15*time.Second {
		t.Errorf("updateFetchTimeout = %v, want a tight bound", updateFetchTimeout)
	}
}
