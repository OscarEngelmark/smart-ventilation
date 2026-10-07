# Working plan

What's left, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Final submission: 2026-10-12, 23:59.

Phase 1 reaches every pass requirement and a complete report. Phase 2 adds
the evaluation that lifts the grade, and starts only once phase 1 is done.

## Phase 1: pass (target 2026-10-10)

Row 15 is the report, one subsection per step, in this order; §1 comes
last because it summarizes the rest. Each step settles its own small
decisions.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 15c | §5 interfaces and data contracts, second half: the REST endpoints (the Damper actuator, the Storage service, the calls to BuildSim) | | 1 |
| 15d | §6 simulating the sensor values | | 1 |
| 15e | §7.1 the autonomous service | | 1 |
| 15f | §7.2 the data pipeline | | 1 |
| 15g | `project_notes.md`: cut the two largest entries (*Decided: the system pursues three goals…* and *Deferred, not yet decided: issues and implicit choices found reviewing…*) down to what the oral exam needs | | 1 |
| 15h | §8.1 dynamic diagram, walking one reading through the loop from the end-to-end run's stored data (`test/results/e2e_2026-10-06*`) | | 1 |
| 15i | §8.2 state diagram | | 1 |
| 15j | §9 deployment diagram and component view | | 1 |
| 15k | §10 test plan | | 1 |
| 15l | §11 evaluation and results | | 1 |
| 15m | §12 dashboard, saving the dashboard screenshot (FR-7) | | 1 |
| 15n | §13 risks and critical reflection | | 1 |
| 15o | §1 summary, including the changes since the proposal | | 1 |

## Phase 2: higher grade (2026-10-11, and any time left)

Each result also updates report §10 and §11.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 16 | Fault tests: a frozen CO2 sensor; the broker down | what counts as recovered | 2 |
| 17 | Stress test to a breaking point | what to scale up, and what counts as broken | 1–2 |
| 18 | Pipeline delay, measured from stored data; the history store keeps whole seconds (`internal/store/sqlite.go`), too coarse for sub-second delays | whether to store each message's receive time | 1 |

## 2026-10-12

Check the report against the template's own "Checklist before submission",
build, and submit.
