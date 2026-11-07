//go:build linux

package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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
// release page URL. Immutable once constructed.
type availableUpdate struct {
	Version string
	URL     string
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
// checker consumes.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// githubCompare is the subset of the GitHub compare payload the checker
// consumes: status is one of identical/behind/ahead/diverged (head vs base).
type githubCompare struct {
	Status   string `json:"status"`
	BehindBy int    `json:"behind_by"`
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
	var release githubRelease
	if err := fetchJSON(ctx, updateFeedURL, &release); err != nil {
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
	var result githubCompare

	url := fmt.Sprintf("%s/%s...%s", updateCompareBaseURL, baseTag, headSHA)
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
		return curPre == "" && latPre != ""
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
func parseReleaseTag(tag string) (core [3]int, preRelease string, ok bool) {
	version := strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V")

	if corePart, pre, found := strings.Cut(version, "-"); found {
		if !parseCore(corePart, &core) {
			return [3]int{}, "", false
		}

		return core, pre, true
	}

	if !parseCore(version, &core) {
		return [3]int{}, "", false
	}

	return core, "", true
}

func parseCore(s string, core *[3]int) bool {
	parts := strings.Split(s, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return false
	}

	for i, part := range parts {
		if part == "" {
			return false
		}

		n := 0
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}

			n = n*10 + int(ch-'0')
		}

		core[i] = n
	}

	return true
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
// transient failure keeps the last known observation (no flapping); verdict
// changes broadcast to SSE clients so open panels morph the notice in live.
func (d *Daemon) checkForUpdate(ctx context.Context, current string) {
	release, err := d.deps.latestRelease(ctx)
	if err != nil {
		slog.Debug("update check failed", "error", err, "current", current)

		return
	}

	newer, ok := d.releaseIsNewer(ctx, current, release)
	if !ok {
		return
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

// releaseIsNewer decides whether the running version predates the release.
// The second return is false when the running version cannot be compared
// ("dev") or the SHA comparison failed — both skip the verdict entirely.
func (d *Daemon) releaseIsNewer(ctx context.Context, current string, release githubRelease) (newer, comparable bool) {
	if sha, isSHA := commitSHA(current); isSHA {
		compare, err := d.deps.compareRelease(ctx, release.TagName, sha)
		if err != nil {
			slog.Debug("commit compare failed", "error", err, "tag", release.TagName, "sha", sha)

			return false, false
		}

		return compare.Status == "behind" && compare.BehindBy > 0, true
	}

	if _, _, isTag := parseReleaseTag(current); isTag {
		return isNewerRelease(current, release.TagName), true
	}

	slog.Debug("skipping update check for uncomparable version", "version", current)

	return false, false
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
