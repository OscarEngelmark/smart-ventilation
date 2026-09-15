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

- **Decided: the target room for the first end-to-end loop is A125**
  (BuildSim id 155, level0, 29.8 m²) — office-sized, and confirmed as a
  real, current LTU room via LTU's own room locator (map.ltu.se). Decided
  2026-09-15.
  - Useful for: §2, §6.

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
  service, physical-model process); Python only for the forecast service.**
  Go for goroutine-based concurrency and small, fast-starting binaries, since
  the whole system has to run on one laptop at once. Python for the
  forecasting ecosystem (statsmodels, scikit-learn), where Go has no mature
  equivalent. Rejected: all-Go (no mature forecasting libraries) and
  all-Python (loses the small-binary/concurrency argument for the service
  layer). Decided by 2026-09-02.
  - The physical-model process wasn't covered by this decision at the time;
    added 2026-09-15 on the same reasoning — it talks to BuildSim over REST
    like sensor/actuator, and its mass-balance calculation needs nothing from
    Python's forecasting ecosystem.
  - Useful for: §4.4, §9.

- **Decided: the three Go services (sensor, actuator, decision) share one Go
  module, built as separate binaries under `cmd/`.** One `go.mod`/`go.sum`
  keeps their dependency versions in sync and lets them share small internal
  packages (a BuildSim REST client, an MQTT wrapper) without duplication;
  each still builds and deploys as its own container, so this doesn't affect
  process independence. Rejected: a separate Go module per service —
  stronger isolation (a change to shared code can't silently affect a
  service you didn't touch), but for three small services built solo, that
  wasn't worth duplicating shared code three times or the extra
  private-module/replace-directive setup. Decided 2026-09-15.
  - Useful for: §4.4, §4.2, §9.

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

- **Decided: two different communication patterns, chosen per link rather
  than one pattern system-wide.** Sensor → forecast → decision runs over
  publish/subscribe through a message broker; decision → actuator is a direct
  REST call. Reasoning: the sensor→forecast→decision stream may gain future
  consumers (dashboard, evaluation) that shouldn't require changing the
  producer, and the system already tolerates a late or dropped reading (see
  *Known caveat — services aren't synchronized*, above) — a fit for pub/sub's
  weaker, decoupled delivery. Decision→actuator is one producer to one fixed
  consumer, low frequency, and a lost command has a real effect with no
  self-correcting next reading — a fit for an explicit REST call retried
  until the actuator acknowledges it, rather than a broadcast that can
  silently drop when the subscriber is offline. Rejected: REST throughout
  (loses producer/consumer decoupling; requires hand-rolling delivery/retry
  logic per link) and broker throughout (a 1:1 low-frequency command doesn't
  get the multi-consumer benefit that justifies broker overhead, and plain
  pub/sub delivery can silently drop it). Replaces the earlier undecided
  single-pattern framing. Decided 2026-09-15.
  - **Decided: MQTT as the broker implementation**, over Kafka (built for
    high-throughput distributed streaming at a scale this system doesn't
    reach — a handful of topics on one laptop) and Redis pub/sub
    (fire-and-forget only, nothing delivered to a subscriber offline at
    publish time). MQTT is built for small, constrained-device messaging,
    matching this system's shape, and is the course's own example
    justification (Canvas Introduction page: "we used MQTT because the
    publish/subscribe model decouples sensors from agents, allowing the
    agent to restart without disrupting sensor data publishing"). Decided
    2026-09-15.
  - Useful for: §4.4, §5, §7.2.

- **Deferred, not yet decided: one decision-service instance per room, or one
  instance handling all rooms.** Also open: data storage, and any user-facing
  view beyond BuildSim's own. Proposal sections 3 and 6 leave these until the
  first one-room loop works end to end.
  - Useful for: §4.2, §9, §12, §13 (scaling).

## 5. Interfaces and data contracts

- **Decided: data contracts are written as JSON Schema documents, one per
  message type, in `schemas/`.** Payloads are JSON, matching BuildSim's own
  API and needing no extra tooling at this project's scale (rejected:
  Protobuf/MessagePack — smaller and faster, but add a schema-compiler step
  not justified here). JSON Schema was chosen over writing plain example
  payloads because a schema can be loaded by a validation library and used to
  reject a malformed message at runtime, which the invalid-sensor-data test
  (see *Decided: failure testing covers a sensor giving bad readings, a
  component going down, and delayed or dropped communication*, §10) needs
  anyway; a bare example payload is only documentation, nothing checks
  against it. Four schemas exist so far: `co2_reading` and `co2_forecast`
  (the two MQTT payloads) and `ventilation_command` /
  `ventilation_command_response` (the REST request/response for the
  decision→actuator link, see *Decided: two different communication
  patterns...*, §4). Decided and implemented 2026-09-15.
  - **Revisit:** `ventilation_command`'s `level` field (normalized 0–1) is a
    placeholder, not a decision — the actual control range depends on the
    actuator/physical model, which isn't designed yet.
  - Useful for: §5, §7.2, §10.

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

- **Decided: readings are persisted in SQLite, accessed only through a small
  storage interface** ("save a reading," "get readings for room X since time
  Y") — no component writes SQL directly. Chosen because nothing in the
  actual requirements needs more at this project's current scale (one room),
  and it adds no new infrastructure on top of Go, Docker, and MQTT, all new
  to this project at once. Rejected: InfluxDB — the better fit for
  time-series data and for multi-room scale (a course-named storage option,
  Canvas Introduction page), but that scale isn't planned before the
  deadline, and running it is a real cost (a separate service, a new query
  interface to learn) with no current payoff. The interface is what keeps
  this reversible: swapping to InfluxDB later means one new implementation
  of it, not a rewrite. Decided 2026-09-15.
  - **Replaces** an earlier same-day decision to use InfluxDB directly,
    reversed once the storage interface made a later swap cheap enough that
    committing to InfluxDB now wasn't buying anything.
  - **Deferred, not yet decided:** which component writes readings (a new
    process, or the forecast service as it consumes each MQTT reading), the
    retention policy, and how the decision service and any dashboard read
    from it. The proposal left the data pipeline out entirely; the feedback
    on accepting it (2026-09-15) was "Do not forget the data pipeline and how
    sensor data is transmitted", so the report must cover it explicitly.
  - Useful for: §4.4, §5, §7.2.

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
