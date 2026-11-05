# Status Archive Manifest

Bulk-archive manifests (docs-health convention: filename → classification + deciding reason).

## 2026-11-05 sweep (docs-health AUDIT)

Forty reports annotated (inline per-item verdicts: `done — evidence` / `NOT-DO — routed owner` /
`Superseded — pointer`), both archive gates pass (`check-rows --strict`, `bare-rows` — 40/40
complete, 40/40 clean), and every genuinely open item harvested into `TODO_LIST.md` (#196–#201)
or `ROADMAP.md` before the move. `2026-11-01_00-57` stays live as the current report. Cross-repo
fleet work (vision-review-agent ops trackers) is marked NOT-DO-routed rather than duplicated here.

| File | Classification | Deciding reason |
| --- | --- | --- |
| `2026-08-17_18-53_website-recovery-pitch-screenshots-video.md` | ARCHIVE | Relaunch shipped ([0.4.0]); residues = TODO #129 / ROADMAP reshoot triggers |
| `2026-08-18_01-18_website-deploy-cert-active-copy-fixes.md` | ARCHIVE | Deploy + cert done; deploy CI later activated ([0.5.0]); CI hardening \u2192 TODO #199 |
| `2026-09-17_13-46_issue-6-pixy-2k-proc-monitor-toggle.md` | ARCHIVE | 2K support shipped in v0.4.0; issue #6 closure owned by TODO #154 |
| `2026-09-17_14-26_git-sync-lockfile-conflict-resolution.md` | ARCHIVE | Conflict fixed + frozen-lockfile guard shipped; policy questions live in ROADMAP |
| `2026-09-17_17-34_issue-6-plan-execution-release-v0.4.0.md` | ARCHIVE | v0.4.0 executed; follow-ups resolved or owned (TODO #148/#154/#156/#166) |
| `2026-09-17_18-43_emeet-studio-research-completion-self-review.md` | ARCHIVE | Research deliverables committed ([0.5.0]); specimens Wayback-durable; PR send = TODO #148 |
| `2026-09-18_05-43_pareto-t1-t3-hid-command-table-breakthrough.md` | ARCHIVE | Breakthrough converted ([0.5.0]); pins \u2192 TODO #150/#166; offshoots \u2192 ROADMAP |
| `2026-09-18_20-43_docs-health-full-sweep-annotate-archive.md` | ARCHIVE | The 09-18 sweep itself; its open threads closed or routed since |
| `2026-09-19_06-44_pareto-plan-execution-full-session.md` | ARCHIVE | Executed ([0.5.0]); hardware \u2192 #166; offshoots \u2192 ROADMAP |
| `2026-09-19_08-33_docs-health-sweep-annotate-archive-status.md` | ARCHIVE | Sweep executed; TS-pin direction superseded (tsc gate, TODO #185) |
| `2026-09-19_09-40_fix-it-all-sweep-specimen-breakthrough.md` | ARCHIVE | Corrections applied; #152 closed; g1/g2 already struck |
| `2026-09-19_10-04_x64-parser-decode-framing-breakthrough.md` | ARCHIVE | Framing implemented; version-shift retracted (superseded marker) |
| `2026-09-19_10-51_protocol-corrections-applied-x64-pipeline-committed.md` | ARCHIVE | All corrections verified in-repo; release superseded (v0.5.0) |
| `2026-09-19_10-56_self-review-gaps-found-after-protocol-corrections.md` | ARCHIVE | Gaps closed; MOTOR-iface worry superseded (0x63 confirmed Beta.25) |
| `2026-09-19_12-05_preset-pull-implementation-and-x64-protocol-decodes.md` | ARCHIVE | Shipped + hardened (#167/#168/#169/#171); hardware pins \u2192 #166 |
| `2026-09-19_13-45_website-deploy-ci-activation-todo-133.md` | ARCHIVE | Deploy CI activated ([0.5.0]); its CI/SA hardening list harvested \u2192 TODO #199 |
| `2026-09-19_14-00_harvest-and-fix-on-sight-sweep.md` | ARCHIVE | Ranked backlog fully harvested (TODO rows + ROADMAP); #44 verdict corrected inline |
| `2026-09-19_15-09_website-vision-review-sweep-17-sites.md` | ARCHIVE | Cross-repo fleet work routed to vision-review-agent trackers; emeet residue = TODO #129 |
| `2026-09-19_16-36_preset-pull-hardening-todo-sweep.md` | ARCHIVE | Hardening shipped; its doc gaps fixed 2026-11-05 (FEATURES/presets.mdx/DOMAIN_LANGUAGE) |
| `2026-09-19_20-14_website-fleet-followup-execution.md` | ARCHIVE | Cross-repo; items executed or owned by vision-review-agent ops trackers |
| `2026-09-20_00-20_website-fleet-followup-execution-2.md` | ARCHIVE | Cross-repo; same routing |
| `2026-09-20_01-23_website-fleet-followup-execution-3.md` | ARCHIVE | Cross-repo; F82 + Lars decisions outstanding in ops trackers |
| `2026-09-28_23-47_art-dupl-deduplication-sweep-status.md` | ARCHIVE | Sweep shipped ([0.5.0]); benchmark/baseline \u2192 TODO #178/ROADMAP |
| `2026-09-28_23-48_multi-device-support-and-marketing-sweep.md` | ARCHIVE | Feature shipped ([0.5.0]); hint surfacing DONE (#173); ADRs pending Lars |
| `2026-09-29_00-34_full-list-execution-security-ci-and-adrs.md` | ARCHIVE | Executed; erraudit thread superseded by the construction migration |
| `2026-09-29_01-36_erraudit-extended-triage-status.md` | ARCHIVE | Entire thread superseded (STRICT 0; ROADMAP L91); Compose pin \u2192 TODO #198 |
| `2026-09-29_05-10_error-construction-migration-status.md` | ARCHIVE | End-state matches ground truth verbatim; open b-items harvested (#198) |
| `2026-10-01_04-21_state-bug-split-brain-audit.md` | ARCHIVE | All findings fixed same day ([0.5.0]); saveState guard \u2192 TODO #196 |
| `2026-10-01_05-00_quickshell-support.md` | ARCHIVE | Endpoint + guide shipped ([0.5.0]); contract pins \u2192 TODO #197 |
| `2026-10-01_05-05_privacy-trap-race-and-webui-fix-status.html` | ARCHIVE | HTML snapshot; all four red flags resolved same day (see 05-23 + [0.5.0]) |
| `2026-10-01_05-23_privacy-trap-fixed-hardware-session.md` | ARCHIVE | Fix shipped + hardware-verified ([0.5.0]); remainders \u2192 #166 checklist |
| `2026-10-01_05-23_state-split-brain-remediation-progress.md` | ARCHIVE | Superseded by the 11-29 complete report (all gates green same day) |
| `2026-10-01_11-02_dedup-sweep-2-session-status.md` | ARCHIVE | Sweep shipped ([0.5.0]); red-tree findings resolved same day |
| `2026-10-01_11-29_state-split-brain-remediation-complete.md` | ARCHIVE | Claims match ground truth; residue = TODO #182 |
| `2026-10-01_11-37_state-split-brain-remediation-retrospective.md` | ARCHIVE | Executed; saveState guard + controllable pins harvested (#196/#197) |
| `2026-10-19_04-19_systemnix-extraction-port-option-monitoring-guide.md` | ARCHIVE | Extraction shipped ([0.5.0]); SystemNix wiring owned by TODO #183 |
| `2026-10-20_09-37_plan-execution-ci-unblock-pma-build-gate-backlog.md` | ARCHIVE | Executed (CI unblocked; P-slices done); dependabot \u2192 TODO #195 |
| `2026-10-31_14-30_architecture-review-self-health.md` | ARCHIVE | Review delivered; findings #190\u2013#194 all DONE 2026-10-31 |
| `2026-10-31_18-15_full-plan-execution-status.md` | ARCHIVE | T01\u2013T05 shipped; remainder covered by the resumed-session + final reports |
| `2026-10-31_20-58_plan-execution-resumed-session-status.md` | ARCHIVE | Executed; T11 close-out landed (see 21-05 + 11-01) |
| `2026-10-31_21-05_pareto-plan-execution-final.md` | ARCHIVE | Final report: every unblocked task DONE and verified; blocked items owned by TODO rows |
