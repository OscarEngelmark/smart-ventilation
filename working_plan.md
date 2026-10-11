# Working plan

What's left, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Final submission: 2026-10-12, 23:59.

Phase 1 reaches every pass requirement and a complete report. Phase 2 adds
the evaluation that lifts the grade, and starts only once phase 1 is done.

## Phase 2: higher grade (2026-10-11, and any time left)

Each result also updates report §10 and §11, and its row in §13's risk
table.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 24 | Cap the MQTT library's wait between reconnect attempts, now its default of up to 10 minutes (`MaxReconnectInterval` in `internal/mqttclient`), so a service notices a returning broker within seconds; then update the report where it says 10 minutes: §8.2 and the stopped-broker row of §13 | the cap | 1 |
| 16 | Fixes for the two faults that failed NFR-3 (`test/fault/frozen.sh`, `test/fault/broker.sh`), then rerun both and update §11 and §13: a fail-safe such as the Damper actuator opening fully when no command arrives for a set time (the Decision service sends a command only when the level changes, so a steady level must not trigger it), and a way to catch a frozen CO2 sensor; if there is no time, §13 names them as next steps | whether to add each fix, and how | 1–2 |
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
