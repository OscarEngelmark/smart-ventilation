# Working plan

What's left, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Final submission: 2026-10-12, 23:59.

Phase 1 reaches every pass requirement and a complete report. Phase 2 adds
the evaluation that lifts the grade, and starts only once phase 1 is done.

## Phase 1: pass (target 2026-10-10)

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 20 | REG-1 evidence, before report §10: a current room day through `eval/day_summary.py`, its output and that day's stored data saved in `test/results/` | which room day | 1 |
| 15 | Report §2–13, then §1, which summarizes the rest; one subsection per step; the dynamic (§8.1), state (§8.2) and deployment (§9) diagrams are drawn in their section's step | small ones per section | ~20 |

## Phase 2: higher grade (2026-10-11, and any time left)

Each result also updates report §10 and §11.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 16 | Fault tests: a frozen CO2 sensor; the broker down | what counts as recovered | 2 |
| 17 | Stress test to a breaking point | what to scale up, and what counts as broken | 1–2 |
| 18 | Pipeline delay, measured from stored data | whether to store each message's receive time | 1 |
| 19 | Decision quality: the learned-model plan against the CO2-only fallback | which metric | 1 |

## 2026-10-12

Check the report against the template's own "Checklist before submission",
build, and submit.
