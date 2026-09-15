# Project notes

Decisions, ideas, findings, and open questions for the project, organized by
final-report section — anything not obvious from the code, or that took real
digging to establish. Code and diagrams link here for rationale, and this is
the source material for the final report: each entry's "Useful for" line maps
it to report sections (§ numbers follow the final report template in the
course repository, https://github.com/eislab-cps/D7065E,
`lab-assignment/final_report_template/`). Open items are marked
**Revisit:**, **Deferred, not yet decided**, or **Idea, not yet evaluated**.

"The proposal" means `latex/proposal/proposal.tex`. Its own section numbers
are written out ("proposal section 3") to keep them apart from report §
numbers.

## 1. Summary

_(nothing yet)_

## 2. Use case and building context

- **Decided: one sensed variable (CO2) and one control loop — no second
  variable such as heating.** Stated reason: effort goes into testing,
  evaluation, and justification rather than component count. Rejected: adding
  a second controlled variable. Proposal section 3, 2026-09-04.
  - Useful for: §2, §4.4, §13.

## 3. Requirements

_(nothing yet)_

## 4. Architecture

- **Decided: every component runs as its own container, with one sensor
  process and one actuator process per room.** Components per proposal
  section 3: physical-model process, sensor, forecast service, decision
  service, actuator, with BuildSim as the shared building state. One container
  per component is a course requirement rather than a choice weighed against
  alternatives — §4.4 still needs a reason of its own for it. Proposal,
  2026-09-04.
  - Useful for: §4.2, §9.

- **Decided: Go for the networked services (sensors, actuators, decision
  service); Python only for the forecast service.** Go for goroutine-based
  concurrency and small, fast-starting binaries, since the whole system has to
  run on one laptop at once. Python for the forecasting ecosystem
  (statsmodels, scikit-learn), where Go has no mature equivalent. Rejected:
  all-Go (no mature forecasting libraries) and all-Python (loses the
  small-binary/concurrency argument for the service layer). Decided by
  2026-09-02.
  - Useful for: §4.4, §9.

- **Decided: BuildSim is the only store of physical state; the physical-model
  process keeps no values of its own between cycles (e.g. the previous CO2
  level).** Each cycle reads the current state from BuildSim, computes the
  next step, and writes it back. Noted by 2026-09-05.
  - **Revisit:** the reasoning and the rejected alternative weren't written
    down at the time — fill them in before this goes into §4.4.
  - Useful for: §4.4, §6.

- **Known caveat — services aren't synchronized: each runs on its own
  schedule.** Any component can therefore see a reading that is stale,
  repeated, or skips a step, and has to tolerate that. Noted by 2026-09-05;
  no handling designed yet.
  - Useful for: §5, §8.1, §10 (timing/delay failure tests), §13.

- **Deferred, not yet decided: how components communicate — request/response
  (REST) or publish/subscribe messaging.** Proposal section 3 describes
  readings and forecasts as "published", while the plan for the Python
  forecast service had the decision service calling it over REST. Neither has
  been chosen or justified against the other. Related to the proposal
  feedback on data transmission (see *Deferred, not yet decided: the data
  pipeline*, §7).
  - Useful for: §4.4, §5, §7.2.

- **Deferred, not yet decided: one decision-service instance per room, or one
  instance handling all rooms.** Also open: data storage, and any user-facing
  view beyond BuildSim's own. Proposal sections 3 and 6 leave these until the
  first one-room loop works end to end.
  - Useful for: §4.2, §9, §12, §13 (scaling).

## 5. Interfaces and data contracts

_(nothing yet)_

## 6. Simulating the sensor values (the physical model)

- **Decided: CO2 comes from a simple mass-balance model, not a replayed
  dataset.** CO2 rises roughly 40 ppm per occupant per hour and decays toward
  an outdoor baseline at a rate set by the ventilation level. Rejected:
  replaying a recorded CO2 dataset — a fixed trace can't respond to the
  actuator, so the control loop wouldn't close. Proposal section 4,
  2026-09-04.
  - **Revisit:** the 40 ppm/occupant/hour figure isn't sourced yet, and it
    implicitly assumes a room size (the same person raises CO2 faster in a
    smaller room).
  - Useful for: §6.

- **Decided: occupancy comes from a time-of-day schedule (arrivals, a meeting
  block, departures) plus some randomness.** Rejected: replaying a public
  occupancy dataset (e.g. UCI Occupancy Detection), and moving individual
  people along BuildSim's navigation graph. A schedule keeps scenarios such as
  a fast-filling meeting easy to shape directly. Unlike CO2, occupancy isn't
  affected by the actuator, so replaying a dataset would be possible here.
  Proposal section 4, 2026-09-04.
  - Reconsidered 2026-09-15, after finding that the course repository
    (https://github.com/eislab-cps/D7065E, `occupancysim/`, added
    2026-09-02) now includes an occupancy simulator. It moves people along
    BuildSim's walkable graph (staff, lecturers, students with lecture
    timetables, night guards) and publishes them to BuildSim's entities and
    occupancy endpoints. Kept the schedule plan for now; no new reason beyond
    the proposal's was given for preferring it.
  - **Revisit:** adopt a different occupancy source later if the schedule
    turns out too simple to be realistic (a risk in proposal section 6) —
    either the course's `occupancysim/` or a public occupancy dataset.
  - Useful for: §6, §13.

## 7. Autonomous services and data pipeline

- **Decided: the decision service compares a short-horizon CO2 forecast to a
  fixed threshold and turns ventilation on before the threshold is crossed.**
  Ventilation only affects CO2 with a delay, so a plain reactive threshold is
  either too late for a fast-filling room or too cautious for a slowly
  filling one. The forecast is load-bearing in every decision, not a fallback
  behind a plain threshold. Rejected: a plain reactive threshold (which also
  serves as the comparison in evaluation). Proposal sections 1 and 5,
  2026-09-04.
  - **Revisit:** the forecast horizon (30 minutes is the working figure,
    not yet checked) and the threshold value are not chosen or justified yet.
  - Useful for: §7.1, §11.

- **Decided: start with the simplest forecasting model that produces a usable
  forecast** (e.g. exponential smoothing or a small regression over recent
  readings), escalating only if it doesn't hold up. Chosen for simplicity,
  not by comparing models. Proposal section 5, 2026-09-04.
  - **Revisit:** whether it forecasts well enough to beat a plain reactive
    threshold — which is what justifies having it in the loop at all (a risk
    in proposal section 6).
  - Useful for: §7.1, §11, §13.

- **Deferred, not yet decided: the data pipeline** — how readings are
  collected, transmitted, stored, and served to the decision service (and any
  dashboard), and what is kept for how long. The proposal left this out; the
  feedback on accepting it (around 2026-09-15) was "Do not forget the data
  pipeline and how sensor data is transmitted", so the report must cover it
  explicitly. See also *Deferred, not yet decided: how components
  communicate*, §4.
  - Useful for: §5, §7.2.

## 8. Behaviour

_(nothing yet)_

## 9. Deployment and component view

_(nothing yet)_

## 10. Test plan

- **Decided: failure testing covers a sensor giving bad readings, a component
  going down, and delayed or dropped communication.** These are the failure
  modes named in proposal sections 6 and 7. No concrete tests designed yet.
  Proposal, 2026-09-04.
  - Useful for: §10, §11, §13.

## 11. Evaluation and results

_(nothing yet)_

## 12. Dashboard

_(nothing yet)_

## 13. Risks and critical reflection

_(nothing yet)_
