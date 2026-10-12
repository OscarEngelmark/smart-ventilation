# Working plan

What's left, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Final submission: 2026-10-12, 23:59.

Phase 1 reaches every pass requirement and a complete report. Phase 2 adds
the evaluation that lifts the grade, and starts only once phase 1 is done.

## Before submission: one-command reruns

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 22 | Run `eval/run_comparison.sh` (about 1 hour; it rebuilds the images), the speed-1 tests (end-to-end, crash; run `docker compose build` with the stack stopped first, since `start.sh` doesn't build) and the three fault tests (frozen sensor, stopped broker, reconnect) on the final code; replace §5's example CO2 message with one from the rerun; check that no stored CO2 value repeats for 5 room minutes with people present and that the Decision service never opened the damper on a frozen reading; bring the new figures into `project_notes.md` and §11, the "code of 2026-10-09" mentions in `project_notes.md` §10 up to date, and §11's "run on the final code" true | — | 1 |

## Report fixes found in review

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 25 | §3 says §7.1 covers the choice of threshold, but §7.1 only states 1000 ppm and points back to §3; neither says why advice stated for schools applies to an office | Where the justification goes, and what it says | 1 |
| 26 | §9, *Edge and external compute*: the pointer to §8.2 for "keeps its last room model" belongs to §7.1's failure list | — | 1 |
| 27 | §9, *Resources*: the Room-model service used 19 MiB, not "about 10"; the Storage service's "about 40 MiB" (full system) next to "at most 36 MiB" (stress test) reads as a contradiction, since they come from different runs | — | 1 |
| 28 | `diagrams/state.tex` and §8.2: add the full opening on a frozen CO2 value; the 5-minute timeout is drawn only under "Waiting for the broker", though the code applies it in every running state | — | 1 |
| 29 | §2, §3 and `requirements.tex` name containers and actors in lowercase ("decision service", "history store", "operator", "the room's occupants"); use the box names of `diagrams/container.d2` and `diagrams/context.d2` | — | 1 |
| 30 | American spelling: "modelled" in §2, and the "Behaviour" heading of §8 | — | 1 |
| 31 | §8.2 and §9 write "(Equation~\ref{…})", printing "(Equation 9)", where the rest of the report prints "(9)" | — | 1 |

## 2026-10-12

Decide whether the page header ("D7065E – Architecture Document") and the
subtitle, which name the course, stay. Check the report against the
template's own "Checklist before submission", then delete that section
from `report.tex`, build, and submit.
