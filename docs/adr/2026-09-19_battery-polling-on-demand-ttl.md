# ADR: Battery Polling Stays On-Demand TTL — No Background Refresh Loop

**Date:** 2026-09-19
**Status:** Accepted (engineering decision; closes the "polling design" remainder of TODO #139)
**Context:** The battery/charge surface (TODO #139, shipped `1a6cb3e`) reads the official GET heads (`09 00 00 02` battery, `09 00 00 06` charge) through a 60-second TTL cache: every surface (`status`, `battery`, Waybar, web panel) triggers at most one HID round-trip per window, and each surface degrades by omitting the line when the device does not answer. The open design question was whether to keep that lazy design or add a background refresh loop.

## Options

### Option A — Background refresh loop (ticker-driven)

Rejected. Whether the wired PIXY answers the battery heads AT ALL is still unverified (hardware session, TODO #166). A background loop on unverified heads means a permanently doomed HID query every interval plus log noise, with zero information gain. Even when verified: battery data has no consumer that needs push-like freshness (no automation keys off it), Waybar and browser views already poll `status`/`waybar` on their own cadence — the TTL deduplicates those — and the 2-second poll ticker exists for call detection; coupling battery into it mixes concerns with different natural rates.

### Option B — On-demand with TTL (keep as shipped) — RECOMMENDED

An unanswered query costs at most one timed-out request per 60-second window, and only in windows where something actually looked (a human opened Waybar/the panel, or a script ran `status`). Cost scales with interest. No goroutine, no interval knob, no log spam when the hardware ignores the heads.

### Option C — Hybrid (background only while a SSE client is connected)

Deferred. Attractive once charge-state transitions should raise toasts ("charging/discharging"), because then freshness has a consumer. It still presupposes a verified-answering device and a decoded `ChargeSta` enum (both #166). Revisit after the hardware session if toast-on-charge-transition becomes a feature.

## Decision

Keep Option B. The 60-second TTL stays the single knob. Revisit triggers, in order: (1) the #166 hardware verdict that the battery heads answer, and (2) a demand for charge-transition notifications — then Option C becomes a small, additive change (a ticker that refreshes the cache only while `Broadcaster` has subscribers).
