# ADR: Surface the Unsupported-Device Hint in the Web UI and Waybar

**Date:** 2026-09-28
**Status:** Proposed — awaiting Lars's decision
**Context:** The multi-device probe (`internal/pixy/model.go`, `probe.go`) classifies what it finds on the bus. Recognized-but-not-controllable devices (fixed-lens EMEET webcams) and unknown EMEET products produce an actionable hint (`pixy.UnsupportedDeviceHint`) that currently reaches exactly two surfaces: the `device`/`probe` CLI commands and a rate-limited daemon log line (`unsupportedWarnLimiter`, 1h). A C960 owner who opens the web UI sees "offline" with no explanation — the message that would save them a support ticket exists but does not reach them. The question: which surfaces should carry the hint? (TODO_LIST #173 is gated on this ADR.)

## Options

### Option A — CLI + logs only (status quo)

Rejected as the end state. The hint's entire purpose is to reach someone who does not know the CLI exists; the web UI's offline panel is precisely where that person looks. Keeping the answer out of the UI optimizes for the one user who already found the answer another way.

### Option B — Short label + full text in web UI and Waybar — RECOMMENDED

Add a typed `webStatus.UnsupportedHint string` (empty = nothing to show; the daemon already stores the hint under `d.mu`, so this is a field copy, no new locking). Surfaces:

- **Web UI offline panel**: one short line ("Recognized: EMEET C960 — not controllable by this daemon") plus the full hint in the panel's existing tooltip/`title` treatment. The offline banner and SSE indicator live outside `#status-panel` precisely so morphs do not reset them — the hint line goes inside the panel where it belongs.
- **Waybar**: one line appended to the tooltip + an additive `unsupportedHint` key in the JSON output, exactly mirroring how `model` and the battery fields shipped (additive, absent when unset, so no consumer breaks).

Copy stays short in the UI; the long-form "why + how to fix" text remains the CLI/log version. Nothing flaps: the hint is stable per device presence and already rate-limited in the log path.

### Option C — Full long-form text everywhere

Rejected. The long-form hint (report-this-PID guidance, env-var onboarding) is CLI-length copy; in a panel or tooltip it becomes wall-of-text noise for the common case ("I know what a C960 is, I just wanted tracking").

## Decision

Pending Lars's sign-off on Option B. Implementation is `TODO_LIST.md` #173 (~90m): one typed field, two template additions, two Waybar branches, tests pinned on the additive-JSON and empty-when-unset contracts.
