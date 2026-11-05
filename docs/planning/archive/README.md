# Planning Archive Manifest

Bulk-archive manifests (docs-health convention: filename → classification + deciding reason).

## 2026-11-05 sweep (docs-health AUDIT)

All seven plans were executed to completion before archiving; every task row carries an inline verdict
(`done at <hash>` / `done — evidence` / `NOT-DO — routed owner` / `Superseded`), both archive gates
(`check-rows --strict`, `bare-rows`) pass, and open residues are owned by `TODO_LIST.md` rows.

| File | Classification | Deciding reason |
| --- | --- | --- |
| `2026-09-17_13-52_ship-2k-and-restore-repo-trust.md` | ARCHIVE | Executed (v0.4.0 shipped 2026-09-17); fine tables struck; sole open thread (issue #6 closure) owned by TODO #154 |
| `2026-09-18_20-51_SUPERB-pareto-execution-plan.md` | ARCHIVE | Executed 2026-09-18/19 (research breakthrough converted; M27 hardware → TODO #166, M30 superseded by v0.5.0); every M-row + F-row verdicted |
| `2026-09-19_15-33_website-vision-review-followup-pareto-plan.md` | ARCHIVE | Executed 2026-09-19 (reports 15-09/20-14/00-20/01-23); repo-relevant residue = TODO #129; cross-repo items owned by the vision-review-agent ops trackers |
| `2026-09-28_23-53_SUPERB-pareto-execution-plan.md` | ARCHIVE | Executed 2026-09-28/29 + completions 2026-10-01/10-31; hardware M-rows → TODO #166; decision rows → ADRs awaiting Lars (ROADMAP); every M/F-row verdicted |
| `2026-10-01_04-24_SUPERB-state-split-brain-remediation-plan.md` | ARCHIVE | Executed 2026-10-01 (CHANGELOG [0.5.0] Fixed; ADR `2026-10-01_camera-intent-vs-connectivity`); §6 checklist ticked; sole residue = TODO #182 |
| `2026-10-19_04-29_SUPERB-systemnix-closeout-and-backlog-pareto-plan.md` | ARCHIVE | Executed 2026-10-20 + 2026-10-31 completions; inert SystemNix items owned by TODO #176/#183; rest DONE |
| `2026-10-31_14-33_SUPERB-v05-release-first-pareto-plan.md` | ARCHIVE | Executed same day (T01–T27 minus hardware/Lars gates; final report `docs/status/2026-10-31_21-05`); gates table ledger-marked (open Lars gates route to TODO_LIST) |
