# Working plan

What's left, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Final submission: 2026-10-12, 23:59.

Phase 1 reaches every pass requirement and a complete report. Phase 2 adds
the evaluation that lifts the grade, and starts only once phase 1 is done.

## Before submission: one-command reruns

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 22 | Run `eval/run_comparison.sh` (about 1 hour; it rebuilds the images), the speed-1 tests (end-to-end, crash; run `docker compose build` with the stack stopped first, since `start.sh` doesn't build) and the three fault tests (frozen sensor, stopped broker, reconnect) on the final code; replace §5's example CO2 message with one from the rerun; check that no stored CO2 value repeats for 5 room minutes with people present and that the Decision service never opened the damper on a frozen reading; bring the new figures into `project_notes.md` and §11, the "code of 2026-10-09" mentions in `project_notes.md` §10 up to date, and §11's "run on the final code" true | — | 1 |

## 2026-10-12

Decide whether the page header ("D7065E – Architecture Document") and the
subtitle, which name the course, stay. Check the report against the
template's own "Checklist before submission", then delete that section
from `report.tex`, build, and submit.
