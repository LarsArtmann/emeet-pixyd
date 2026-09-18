# ADR: Structured Command Types — Recommend Incremental Typed Registry (Status Quo Plus)

**Date:** 2026-09-18
**Status:** Proposed (decision requested from Lars)
**Context:** ROADMAP design-pender (former TODO #116, HIGH impact / HIGH effort). `handleCommand(ctx, string)` + `strings.Fields` dispatch has served every feature so far, but three pressure points have accumulated: multi-word arguments truncate silently (the `preset save "my home"` bug, #123), every new command re-implements argument validation by hand, and the string-typed surface means the compiler cannot catch a misspelled command constant.

## Survey of the current surface

`commands.go` dispatches ~25 commands in a three-layer switch (transport lock → axis route → subcommand). Patterns present today:

- Named constants for every command name and response string (no raw literals anywhere).
- Hand-rolled validation per command: `speed <axis> <val>` validates axis via `pixy.MotorTypeFromAxis` and range via a named bound; `tracking <variant>` via `pixy.ParseTargetTrackMode`; `audio <mode>` via `pixy.ParseAudioMode`.
- Two lock classes chosen per command (`hidMu` vs `v4l2Mu` vs state-only), encoded in the dispatcher.
- The web UI routes through the SAME `handleCommand` string path — one validation definition for both transports.

The V2 protocol work (2026-09-18) already produced a typed, tabular sub-registry: `pixy.V2Head` constants, payload builders, and the simulator's `v2SetSpecs` validation table. That is the shape this ADR argues for — extended to the CLI surface.

## Options

### Option A — Command-parser library (e.g. cobra-style command tree)

Rejected. The daemon is not a CLI app with subcommand help trees; `os.Args` is intentionally NOT flag-parsed (it doubles as the socket command transport). A command tree brings dependency weight, a second help system beside `helpText`, and an import-graph inversion for the socket path. Also violates the project's dependency minimalism (see kardianos ADR for the precedent of rejecting generic lifecycle/framework glue).

### Option B — Full hand-rolled typed registry (one pass)

Every command becomes a struct: `{Name, MinArgs, MaxArgs, ArgsSpec, LockClass, Handler(ctx, TypedArgs) Result}` registered in a map; dispatch becomes lookup + generic arg decode. Rejected as a big-bang for now: it touches all ~25 commands at once, the win is mostly type-safety aesthetics (validation is already centralized and tested), and the risk lands exactly where the hardware-touching code is.

### Option C — Status quo plus incremental typed registry (RECOMMENDED)

Keep `handleCommand(ctx, string)` as the transport boundary. Extend the existing pattern command-by-command, following the v2 protocol precedent:

1. **Typed vocabularies per domain** (already happening): `pixy.MotorType`, `pixy.TargetTrackMode`, `pixy.AudioMode`, `pixy.AutoMode` — each with `Parse`/`Valid`/`String`. New commands add one of these instead of hand-rolled validation.
2. **Arg specs only where a family repeats**: if a fourth `X <axis> <val>`-style command appears, extract a shared `axisValueArg` helper; do not pre-build a generic framework (YAGNI).
3. **Multi-word args**: fixed under the same increment (see the companion ADR for #123) — `joinRemaining(parts, n)` for name-taking commands, no framework needed.

**Effort:** the parts that matter are already merged (v2 vocabulary). Remaining increment ≈ one small PR when the next command family lands.

**Migration risk:** near zero — no transport, registry, or dispatch rewrite; existing tests pin behavior.

**What we consciously give up:** compile-time exhaustiveness of the command set. The named-constant convention plus the fuzz/behavior tests cover most of that risk; `go vet`-style exhaustiveness would require the Option B registry.

## Recommendation

Option C. The 2026-09-18 protocol work already validated the incremental approach in the hardest subsystem (HID), and the full rewrite (Option B) buys structure the codebase does not yet need at 25 commands. Revisit Option B only if a fourth transport (beyond CLI/socket/web) or ~40+ commands materialize.

## Decision requested

Approve Option C (and the companion #123 approach) so both ROADMAP penders can close. No code ships under this ADR beyond what is already merged.
