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
| 15o | §1 summary, including the changes since the proposal | | 1 |

## Phase 2: higher grade (2026-10-11, and any time left)

Each result also updates report §10 and §11, and its row in §13's risk
table.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 16 | Fault tests: a frozen CO2 sensor; the broker down. Readings that stop reaching the Decision service leave the damper at its last level, so people arriving meanwhile can take CO2 over 1000 ppm (about 1400 ppm for a full room at 0.4, derived); include that case; a broker down long enough also delays recovery after it returns, since the MQTT library doubles its wait between reconnect attempts up to 10 minutes (`MaxReconnectInterval`), so measure that too; and if NFR-3 fails, decide on a fail-safe such as the Damper actuator opening fully when no command arrives for a set time (the Decision service sends a command only when the level changes, so a steady level must not trigger it) | what counts as recovered; whether to add the fail-safe | 2 |
| 17 | Stress test to a breaking point; sample each container's memory and CPU repeatedly over the run, since §9's figures rest on one `docker stats` sample | what to scale up, and what counts as broken | 1–2 |
| 20 | The Decision service checks each message it receives against its schema, as the Storage service and the Damper actuator do; today a CO2 reading missing its `ppm` field reads as 0 ppm (Revisit in the notes' JSON Schema entry, §5); then update the report where it says the Decision service does not check: §7.1 and §7.2 | what it does with a refused message | 1 |
| 18 | Pipeline delay, measured from stored data and set against the sampling intervals (10 s for CO2, 1 min for the head count), which likely dominate how fast the damper reacts to a change; the history store keeps whole seconds (`internal/store/sqlite.go`), too coarse for sub-second delays | whether to store each message's receive time | 1 |

## Before submission: one-command reruns

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 22 | Put the whole comparison in one script that runs unattended from a snapshot to the comparison table, saving into one run folder, so the final rerun on the submitted code is one command with no decisions on the way; then rerun it and the speed-1 tests on the final code | which settings the final rerun uses | 1 |

## 2026-10-12

Check the report against the template's own "Checklist before submission",
build, and submit.
