//go:build linux

package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// The update checker asks GitHub for the newest release on a slow cadence
// and records it so every status surface can show a small "update available"
// notice (web panel footer, /api/status, CLI version, Waybar tooltip). Two
// version channels must both work:
//
//   - GoReleaser builds inject the release tag ("v0.6.0") — compared
//     numerically against the latest tag.
//   - Nix flake builds inject a git commit SHA (flake.nix uses
//     self.rev / self.dirtyRev) — compared via the GitHub compare API
//     (only a commit that is strictly BEHIND the release tag counts).
//
// Anything else ("dev" from a bare `go build`) cannot be compared and never
// notifies. All failures are debug-logged only: the check is best-effort
// observability and must never degrade the daemon.
const (
	updateFeedURL        = "https://api.github.com/repos/LarsArtmann/emeet-pixyd/releases/latest"
	updateCompareBaseURL = "https://api.github.com/repos/LarsArtmann/emeet-pixyd/compare"

	updateCheckInitialDelay = 30 * time.Second
	updateCheckInterval     = 24 * time.Hour
	updateFetchTimeout      = 10 * time.Second
	updateMaxBodyBytes      = 1 << 20
)

// availableUpdate describes a newer upstream release observed by the update
// checker: the release tag as published (e.g. "v0.6.0") and the human-facing
// release page URL. Immutable once constructed. JSON tags serve the
// additive /api/status `update` object (omitted entirely when nil).
type availableUpdate struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// updateState holds the update checker's last verdict. Guarded by its own
// lock (never d.mu — availability changes on a slow network cadence fully
// independent of camera state, and snapshot() only reads it as a leaf lock).
type updateState struct {
	mu     sync.RWMutex
	update *availableUpdate
}

// get returns the currently known update, nil when none is known.
func (u *updateState) get() *availableUpdate {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.update
}

// set records a verdict (nil = no update) and reports whether the observation
// changed, so the caller can broadcast only on transitions.
func (u *updateState) set(update *availableUpdate) bool {
	u.mu.Lock()
	defer u.mu.Unlock()

	changed := !sameUpdate(u.update, update)
	u.update = update

	return changed
}

func sameUpdate(a, b *availableUpdate) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Version == b.Version && a.URL == b.URL
}

// githubRelease is the subset of the GitHub releases/latest payload the
// checker consumes (snake_case is the upstream API contract).
type githubRelease struct {
	TagName string `json:"tag_name"` //nolint:tagliatelle
	HTMLURL string `json:"html_url"` //nolint:tagliatelle
}

// githubCompare is the subset of the GitHub compare payload the checker
// consumes: status is one of identical/behind/ahead/diverged (head vs base).
type githubCompare struct {
	Status   string `json:"status"`
	BehindBy int    `json:"behind_by"` //nolint:tagliatelle
}

//nolint:gochecknoglobals // shared HTTP client, safe for concurrent use
var updateHTTPClient = &http.Client{Timeout: updateFetchTimeout}

// fetchJSON GETs a GitHub API endpoint and decodes its JSON body into
// target. Non-2xx answers (rate limits, missing releases) are Transient —
// the checker just retries on its next cadence tick.
func fetchJSON(ctx context.Context, url string, target any) error {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return errorfamily.WrapTransientf(reqErr, "update.build_request", "build request for %s", url)
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	resp, getErr := updateHTTPClient.Do(req)
	if getErr != nil {
		return errorfamily.WrapTransientf(getErr, "update.fetch", "fetch %s", url)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Debug("update feed body close failed", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return errorfamily.NewTransient("update.http_status",
			fmt.Sprintf("GET %s answered %s", url, resp.Status))
	}

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, updateMaxBodyBytes))
	if readErr != nil {
		return errorfamily.WrapTransientf(readErr, "update.read_body", "read %s body", url)
	}

	if decodeErr := json.Unmarshal(body, target); decodeErr != nil {
		return errorfamily.WrapTransientf(decodeErr, "update.decode", "decode %s body", url)
	}

	return nil
}

// fetchLatestReleaseFeed reads the newest published release. A release
// without a tag or page URL is treated as an error so a malformed feed can
// never surface an unusable notice.
func fetchLatestReleaseFeed(ctx context.Context) (githubRelease, error) {
	return releaseFromFeed(ctx, updateFeedURL)
}

func releaseFromFeed(ctx context.Context, feedURL string) (githubRelease, error) {
	var release githubRelease
	if err := fetchJSON(ctx, feedURL, &release); err != nil {
		return githubRelease{}, err
	}

	if release.TagName == "" || release.HTMLURL == "" {
		return githubRelease{}, errorfamily.NewTransient("update.feed_incomplete",
			fmt.Sprintf("release feed lacks tag_name or html_url (tag %q)", release.TagName))
	}

	return release, nil
}

// fetchReleaseCompare asks whether headSHA sits strictly behind baseTag on
// the default branch history (the flake-build channel: buildVersion is a
// commit SHA, not a tag).
func fetchReleaseCompare(ctx context.Context, baseTag, headSHA string) (githubCompare, error) {
	return compareFromAPI(ctx, updateCompareBaseURL, baseTag, headSHA)
}

func compareFromAPI(ctx context.Context, baseURL, baseTag, headSHA string) (githubCompare, error) {
	var result githubCompare

	url := fmt.Sprintf("%s/%s...%s", baseURL, baseTag, headSHA)
	if err := fetchJSON(ctx, url, &result); err != nil {
		return githubCompare{}, err
	}

	return result, nil
}

// isNewerRelease compares two release tags numerically (major.minor.patch).
// It reports whether latest is strictly newer than current; versions that do
// not parse (including "dev") never notify. Equal cores rank the pre-release
// form as older than the final (0.6.0-rc1 < 0.6.0); two pre-releases of the
// same core are treated as equal (no notice — noise beats precision here).
func isNewerRelease(current, latest string) bool {
	curCore, curPre, curOK := parseReleaseTag(current)

	latCore, latPre, latOK := parseReleaseTag(latest)
	if !curOK || !latOK {
		return false
	}

	switch compareVersionCores(curCore, latCore) {
	case -1:
		return true
	case 1:
		return false
	default:
		// Equal cores: the pre-release form is older than the final, so the
		// latest is newer only when current carries a pre-suffix and the
		// latest does not.
		return curPre != "" && latPre == ""
	}
}

// compareVersionCores compares two 3-component numeric cores: -1 (a older),
// 0 (equal), 1 (a newer). Cores are pre-padded by parseReleaseTag.
func compareVersionCores(a, b [3]int) int {
	for i := range a {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}

	return 0
}

// parseReleaseTag splits a tag into its numeric core (padded to 3
// components) and pre-release suffix; ok is false for anything that does not
// start major[.minor[.patch]] numerically ("dev", commit SHAs, garbage).
func parseReleaseTag(tag string) ([3]int, string, bool) {
	version := strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V")

	if corePart, pre, found := strings.Cut(version, "-"); found {
		core, ok := parseCore(corePart)
		if !ok {
			return [3]int{}, "", false
		}

		return core, pre, true
	}

	core, ok := parseCore(version)
	if !ok {
		return [3]int{}, "", false
	}

	return core, "", true
}

func parseCore(version string) ([3]int, bool) {
	var core [3]int

	parts := strings.Split(version, ".")
	if len(parts) > 3 {
		return core, false
	}

	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return core, false
		}

		core[i] = n
	}

	return core, true
}

// commitSHA extracts a git commit SHA from a buildVersion: the full 40-hex
// form (flake self.rev), optionally "-dirty"-suffixed (self.dirtyRev), or a
// 7-hex short SHA. ok is false for release tags and unparseable versions.
func commitSHA(buildVersion string) (string, bool) {
	sha := strings.TrimSuffix(buildVersion, "-dirty")
	if len(sha) != 40 && len(sha) != 7 {
		return "", false
	}

	for _, ch := range sha {
		if !isHexDigit(ch) {
			return "", false
		}
	}

	return sha, true
}

func isHexDigit(ch rune) bool {
	return ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F'
}

// checkForUpdate runs one upstream release check against the injected feed
// functions and records the verdict. current is the running buildVersion. A
// version that cannot be compared ("dev") returns before any network use;
// a transient failure keeps the last known observation (no flapping);
// verdict changes broadcast to SSE clients so open panels morph the notice
// in live.
func (d *Daemon) checkForUpdate(ctx context.Context, current string) {
	if _, byCommit := commitSHA(current); !byCommit {
		if _, _, byTag := parseReleaseTag(current); !byTag {
			slog.Debug("skipping update check for uncomparable version", "version", current)

			return
		}
	}

	release, err := d.deps.latestRelease(ctx)
	if err != nil {
		slog.Debug("update check failed", "error", err, "current", current)

		return
	}

	newer, err := d.newerThanCurrent(ctx, current, release)
	if err != nil {
		return // already debug-logged; keep the last known observation
	}

	update := (*availableUpdate)(nil)
	if newer {
		update = &availableUpdate{Version: release.TagName, URL: release.HTMLURL}
		slog.Info("update available", "current", current, "latest", release.TagName, "url", release.HTMLURL)
	} else {
		slog.Debug("running the newest release", "current", current, "latest", release.TagName)
	}

	if d.updates.set(update) {
		d.broadcastStateChanged()
	}
}

// commitBehindRelease resolves the flake-build channel: the running build
// is a commit SHA, so it is newer than the release only when the commit
// sits strictly BEHIND the release tag in history.
func (d *Daemon) commitBehindRelease(ctx context.Context, release githubRelease, sha string) (bool, error) {
	compare, err := d.deps.compareRelease(ctx, release.TagName, sha)
	if err != nil {
		slog.Debug("commit compare failed", "error", err, "tag", release.TagName, "sha", sha)

		return false, err
	}

	return compare.Status == "behind" && compare.BehindBy > 0, nil
}

// newerThanCurrent resolves both version channels against a fetched release:
// the commit channel via the GitHub compare API, the tag channel numerically.
func (d *Daemon) newerThanCurrent(ctx context.Context, current string, release githubRelease) (bool, error) {
	if sha, byCommit := commitSHA(current); byCommit {
		return d.commitBehindRelease(ctx, release, sha)
	}

	return isNewerRelease(current, release.TagName), nil
}

// runUpdateChecker polls the upstream release feed until ctx is done: first
// after updateCheckInitialDelay (let startup settle behind READY=1), then
// once per updateCheckInterval. Disabled by EMEET_PIXYD_UPDATE_CHECK=off.
func (d *Daemon) runUpdateChecker(ctx context.Context) {
	if !d.config.UpdateCheck {
		slog.Debug("update check disabled (EMEET_PIXYD_UPDATE_CHECK)")

		return
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(updateCheckInitialDelay):
	}

	for {
		d.checkForUpdate(ctx, buildVersion)

		select {
		case <-ctx.Done():
			return
		case <-time.After(updateCheckInterval):
		}
	}
}
