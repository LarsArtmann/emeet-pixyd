# ADR: Declarative NixOS Option for Extra Product IDs

**Date:** 2026-09-28
**Status:** Proposed — awaiting Lars's decision
**Context:** `EMEET_PIXYD_EXTRA_PRODUCT_IDS` (comma-separated hex, `internal/pixy/config.go`) is the runtime on-ramp for unregistered PIXY-family USB product IDs. Config is otherwise fully declarative in the NixOS module: `hardware.emeet-pixy.{enable,user,auto,defaultAudio,debug}` all become `Environment=` entries in the systemd user unit. This new knob would be the only one a NixOS user must set through a hand-written systemd drop-in. The question: keep it env-only, or add `hardware.emeet-pixy.extraProductIds`?

## Options

### Option A — Env-only forever

Rejected. The NixOS user story becomes: look up the env var in a README, write a `systemd.user.services.emeet-pixyd.serviceConfig.Environment` drop-in, hope the unit name survives module updates. Every other knob bypasses that; one escape hatch does not justify breaking the module's abstraction for a ~10-line option.

### Option B — Declarative `extraProductIds` option — RECOMMENDED

`hardware.emeet-pixy.extraProductIds = [ "0x00ef" "0x0123" ];` → the module joins them into the existing env var for the unit. Properties:

- **Env stays the wire format.** Non-NixOS users keep the documented env path; the option is pure Nix-side sugar over the same variable, so there is exactly one daemon-side parser and one source of truth for semantics (validation, warning behavior, extra-outranks-fixed ordering).
- **Type-safe surface:** `types.listOf types.str`, default `[]`, empty list emits nothing (byte-identical unit for everyone who does not use it — vmTest stays green with no changes).
- **vmTest hook:** the existing VM test can add a case asserting the joined env lands in the unit; optional, small.
- **Docs:** the NixOS module README row and `docs/hid-protocol.md`/website onboarding text gain one line each.

### Option C — Full declarative registry (module writes a config file)

Rejected for now. A file-based config would bypass `pixy.ConfigFromEnv` and create a second config pipeline to keep in sync — real cost, zero benefit over Option B while env-only parsing exists. Revisit only if the env var grows semantics that stop fitting a single string (e.g. per-ID labels).

## Decision

Pending Lars's sign-off on Option B. Implementation is ~10 lines of module code + 1 vmTest case + doc rows (plan M11, ~60m).
