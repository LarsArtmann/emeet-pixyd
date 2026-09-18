# ADR: Multi-Word Preset Names via CLI — Recommend Join-Remaining-Parts

**Date:** 2026-09-18
**Status:** Proposed (decision requested from Lars)
**Context:** ROADMAP design-pender (former TODO #123). The CLI dispatches on `strings.Fields`, so `preset save my home` silently saves the preset `"my"` and drops `"home"`. The web UI is unaffected (its name input travels as a bound signal, not through field splitting). The truncation is now pinned by a regression test (`TestPresetSave_MultiWordNameTruncates`) which fails — loudly — once this ADR's recommendation lands and must then be inverted.

## Repro (pinned by test)

```console
$ emeet-pixy preset save "my home"   # shell quoting is irrelevant: the
                                     # daemon receives the whole string...
preset 'my home' saved               # ...but dispatch splits on spaces
$ emeet-pixy preset list
presets:
  my: pan=30 tilt=-10 zoom=120       # "home" silently gone
```

## Options

### Option A — Shell-style quoting (`preset save "my home"` after field split, re-join quoted spans)

Rejected. Re-introduces a quoting dialect inside the socket protocol; every transport (CLI, socket, web) would need the same quoting semantics, error messages get subtle ("unbalanced quote"), and the daemon has no business teaching users a mini-shell.

### Option B — Structured commands (the full #116 registry)

Rejected for now — gated on the registry decision, which the companion ADR recommends deferring. Using a big-bang rewrite to fix a two-line argument bug is the tail wagging the dog.

### Option C — Join remaining parts for name-taking subcommands (RECOMMENDED)

`preset save|load|delete|push` take exactly one argument: the name. So `parts[2:]` joined with a single space IS the name — no ambiguity exists because no name-taking subcommand takes a second argument. A ~6-line `joinRemaining(parts, 2)` helper, applied at each name-taking site, preserves 100% of current single-word behavior and fixes multi-word names in every transport at once.

Validation already guards the result: `pixy.ValidatePresetName` rejects empty, >32-rune, and control-character names, so pathological whitespace input degrades to a validation error, not a corrupt preset.

**Effort:** one small PR (helper + four call sites + test inversion). **Risk:** near zero; existing preset tests pin single-word behavior.

### Option D — Accept the limitation (document only)

Rejected: the failure mode is silent data loss (a preset that did not save where the user thinks it did), the fix is cheap, and the web UI already proves multi-word names are a real use case.

## Recommendation

Option C, under the incremental-registry umbrella of the companion structured-command-types ADR (this IS its step 3). After it lands: invert the pinning test to assert `my home` is saved, close #123, and record the fix in CHANGELOG.

## Decision requested

Approve Option C for implementation in the next PR batch.
