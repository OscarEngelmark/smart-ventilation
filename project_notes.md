# Project notes

Decisions, ideas, findings, and open questions for the project, organized by
final-report section — anything not obvious from the code, or that took real
digging to establish. Code and diagrams link here for rationale, and this is
the source material for the final report: each entry's "Useful for" line maps
it to report sections (§ numbers follow the final report template in the
course repository, https://github.com/eislab-cps/D7065E,
`lab-assignment/final_report_template/`). Open items are marked
**Revisit:**, **Deferred, not yet decided**, or **Idea, not yet evaluated**.
A final section, after §13, records where AI advice was wrong.

"The proposal" means `latex/proposal/proposal.tex`. Its own section numbers
are written out ("proposal section 3") to keep them apart from report §
numbers.

## 1. Summary

_(nothing yet)_

## 2. Use case and building context

- **Decided: the use case is indoor air quality (CO2) management, chosen over
  fire and gas detection.** Fire/gas detection pulls in roughly four times the
  component count — multi-sensor fusion, spatial diffusion physics, evacuation
  routing — without reaching a higher grade ceiling. CO2 gives a loop tight
  enough to trace end to end (occupancy → buildup → sensor → decision →
  ventilation command → actuator → next reading), built on physics that is
  already familiar. Decided 2026-09-02.
  - Useful for: §2, §4.4, §13.

- **Decided: one sensed variable (CO2) and one control loop — no second
  variable such as heating.** Stated reason: effort goes into testing,
  evaluation, and justification rather than component count. Rejected: adding
  a second controlled variable. Proposal section 3, 2026-09-04.
  - Useful for: §2, §4.4, §13.

- **Decided: the target room for the first end-to-end loop is A125**
  (BuildSim id 155, level0, 29.8 m², in the LTU A-building) — office-sized, and confirmed as a
  real, current LTU room via LTU's own room locator (map.ltu.se). Decided
  2026-09-15.
  - Useful for: §2, §6.

- **Why the use case matters: no fixed ventilation level meets both CO2
  and fan goals through a weekday.** The ventilation is sized for a full
  room, since a room can fill without warning, but for most of the day
  A125 holds three people or none. The full-room setting all day keeps CO2
  low at a cost in noise and energy; a fixed low setting lets CO2 cross
  1000 ppm when the 13:00 meeting fills the room. The level that meets
  both depends on the head count now and on how the room responds, which
  is what the decision service plans from (see *Decided: the system
  pursues three goals…*, §7). Reasoned from the project's own goals; no
  external source on building energy use was added. Noted 2026-10-06.
  - Useful for: §2.

## 3. Requirements

- **Decided: fifteen requirements, each traced to a test that exists or is
  planned.** Rejected: a requirement per container — more rows than tests,
  so the extra rows would be untested gaps. Requirements whose test is
  planned but not yet run stay in the table and are reported as unverified
  if the test isn't done. Decided 2026-10-05.

  | ID | Type | Priority | Requirement and acceptance criterion | Verified by |
  |---|---|---|---|---|
  | FR-1 | Functional | Must | CO2 readings and head counts are saved in the history store; a full simulated weekday has a stored CO2 reading for every 10 s and a head count for every room minute | Stored data of a room day |
  | FR-2 | Functional | Must | The decision service sets the damper in BuildSim from the CO2 readings, in time to hold the threshold; after a reading above the target, the damper is fully open within 2 min of room time | End-to-end test |
  | FR-3 | Functional | Must | A damper change shows in the CO2 readings that follow | End-to-end test |
  | FR-4 | Functional | Must | Other components can fetch a room's stored readings and commands from a given time on; a request returns exactly that room's rows from that time on, oldest first; commands are led by the one in force at that time | Unit tests, `internal/store` |
  | FR-5 | Functional | Must | The room-model service fits the room model to the stored history; the fit gives no room model when the history can't separate the rates | Unit tests, `internal/roommodel` |
  | FR-6 | Functional | Must | Without a room model, the damper opens and closes on CO2 alone | Unit tests, `internal/planner` |
  | FR-7 | Functional | Should | The dashboard shows CO2, head count and damper level over recent room time | Screenshot |
  | FR-8 | Functional | Must | The storage service keeps bad messages out of the history store; a message that fails its schema, a reading for another room than its topic names, and a second copy of a stored reading are not stored | Unit tests, `cmd/storage-service` and `internal/store` |
  | FR-9 | Functional | Must | The occupancy simulator fills the room on a weekday schedule; nobody at night or at weekends, 3 people mid-morning and mid-afternoon, empty at 12:00, full (6) at 13:30, never over capacity | Unit tests, `internal/occupancy` |
  | FR-10 | Functional | Must | The physical model's CO2 follows the people and the damper; with the damper closed, a full room settles at about 3640 ppm and one person at about 957 ppm; fully open, a full room settles at about 903 ppm; an empty room returns to the outdoor 420 ppm | Unit tests, `internal/co2` |
  | REG-1 | Regulatory | Must | CO2 stays under 1000 ppm through a full simulated weekday | Peak CO2 over a room day, `eval/day_summary.py` |
  | NFR-1 | Non-functional | Must | A killed container, once restarted, registers with BuildSim again and the loop resumes within 2 minutes | Crash-and-restart test, at speed 1 |
  | NFR-2 | Non-functional | Should | The time-weighted mean damper level over a room day is lower than under the CO2-only fallback | Decision-quality comparison |
  | NFR-3 | Non-functional | Should | A frozen CO2 sensor or a downed broker doesn't push CO2 over 1000 ppm | Fault tests |
  | NFR-4 | Non-functional | Could | Time from a reading to its command stays under 2 min of room time | Delay measurement |

  - REG-1's threshold is general advice (FoHMFS 2014:18), not a binding
    rule; see *Threshold `C_threshold` = 1000 ppm*, §7.
  - FR-2's, NFR-1's and NFR-4's 2 minutes is derived, not from a source:
    with the damper closed, a full room raises CO2 about 23 ppm/min (§7,
    *Decided: the system pursues three goals…*), so the 50 ppm between the
    950 ppm target and the threshold lasts about 2 minutes.
  - **Decided: NFR-4's limit comes from that margin, not from a
    measurement.** Rejected: measuring the delay first and setting the
    limit from it, a test that can't fail and so verifies nothing.
    Decided 2026-10-06.
  - Grown from twelve requirements with FR-8 to FR-10, so that the
    pipeline's handling of bad data and the simulated room each have a
    requirement; all three are verified by existing unit tests, plus one
    added for a message that fails its schema. Rejected: also covering the
    damper actuator's refusal of a malformed command, which has no unit
    test. Storage retention (90 room days) has no requirement. Decided
    2026-10-06.
  - Useful for: §3, §10, §11.

## 4. Architecture

- **Decided: every component runs as its own container, with one sensor
  process and one actuator process per room.** Components per proposal
  section 3: physical-model process, sensor, forecast service, decision
  service, actuator, with BuildSim as the shared building state. One container
  per component is a course requirement; its reason is that a crash in one
  container leaves the others running. Rejected: one single program, where
  any crash stops every part. Trade-off: more containers to build and
  run, and messages between them. Proposal, 2026-09-04; reason added
  2026-10-07.
  - An occupancy process was added to this list after the proposal, see
    *Decided: occupancy is generated by its own process, separate from the
    physical model* — a change to record in §1.
  - The proposal's forecast service is gone: its CO2 forecast was replaced
    by an occupancy forecast, which was later removed as well (see
    *Decided: the occupancy forecast is removed…*, §7). The occupancy sensor
    and the room-model service were added instead.
  - Useful for: §4.2, §9.

- **Decided: Go for every service, including the learning service; Python
  stays open for it if a later model needs its libraries.** Go for
  goroutine-based concurrency and small, fast-starting binaries, since the
  whole system has to run on one laptop at once. The first forecast model
  was a straight-line fit (see *Decided: the CO2 forecast service is
  removed…*, §7) and the room model is a least-squares fit of three numbers;
  neither needs a library, and the rest of such a service is MQTT, REST, and
  schema-checking code the Go services already have. Rejected: all-Python
  (loses the small-binary/concurrency argument for the service layer). The
  learning service is its own container, so moving it to Python later
  changes one folder. Decided 2026-09-23 for the forecast service; carried
  over to the room-model service when the forecast was removed.
  - Replaces Python for the forecast service (decided by 2026-09-02, for
    statsmodels and scikit-learn). Dropped once the model turned out to be
    a straight line: Python would have added a second toolchain and a second
    copy of the plumbing for libraries the model doesn't use.
  - The physical-model process wasn't covered by this decision at the time;
    added 2026-09-15 on the same reasoning — it talks to BuildSim over REST
    like sensor/actuator, and its mass-balance calculation needs nothing from
    Python's forecasting ecosystem.
  - Useful for: §4.4, §9.

- **Decided: the project's Go services share one Go module, built as
  separate binaries under `cmd/`.** One `go.mod`/`go.sum`
  keeps their dependency versions in sync and lets them share small internal
  packages (a BuildSim REST client, an MQTT wrapper) without duplication;
  each still builds and deploys as its own container, so this doesn't affect
  process independence. Rejected: a separate Go module per service —
  stronger isolation (a change to shared code can't silently affect a
  service you didn't touch), but for small services built solo, that
  wasn't worth duplicating shared code in each or the extra
  private-module/replace-directive setup. Decided 2026-09-15.
  - Useful for: §4.4, §4.2, §9.

- **Decided: BuildSim is the only store of physical state; the physical-model
  process keeps no values of its own between cycles (e.g. the previous CO2
  level).** Each cycle reads the current state from BuildSim, computes the
  next step, and writes it back. The physical model stands in for the
  physical world, outside the system: in a real building it wouldn't
  exist, and the sensor, decision service, and actuator only ever see
  BuildSim. Rejected: the model keeping the true CO2 internally and
  publishing it to BuildSim at a fixed interval. It matches "the model is
  reality" more literally, but a crash loses the value and needs restart
  logic, while here nothing is lost because the value always lives in
  BuildSim. It was first proposed to keep the model accurate when the room
  runs faster than real time; the exact solution in the mass-balance entry
  (§6) does that in this design too. Noted by 2026-09-05; reasoning and
  alternative added 2026-09-18; implemented 2026-09-22 as
  `cmd/physical-model/`.
  - **Decided: the C4 diagrams draw the physical model and the occupancy
    simulator outside the system boundary, in their own box standing in for
    the real room, connected only to BuildSim.** BuildSim is the one link
    between the simulated parts and the containers that would have a
    real-world counterpart, and the diagram shows that; in a real building
    the box would be replaced by the room and its devices. They are still
    drawn as containers, so the container diagram matches
    `docker-compose.yml` one to one. The context diagram follows the same
    rule: the room's occupants act on BuildSim, not on the system. Rejected:
    drawing them inside the system boundary, which suggests they would ship
    with a real deployment; leaving them off, which breaks the match with
    `docker-compose.yml`. Decided and drawn 2026-09-26
    (`diagrams/container.d2`, `diagrams/context.d2`).
    - Replaces *Revisit: where the physical model sits in the C4
      diagrams*.
    - REST arrows point from the caller to the service it calls, so every
      link to BuildSim is one-way: nothing uses BuildSim's WebSocket. MQTT
      arrows point the way messages travel, two-headed where they travel
      both ways. Rejected: drawing MQTT arrows from the side that opens
      the connection, as for REST. Subscribers open their own connection
      to the broker too, so every MQTT arrow would point at the broker and
      the diagram would no longer show who receives which messages.
      Decided and drawn 2026-10-07.
  - Useful for: §4.1, §4.4, §6.

- **Decided: the container that fits the room model is the "Room-model
  service".** "Room model" means only the learned rates it publishes on
  `<room_id>/model`, as in the code (`internal/roommodel`), and the box
  name matches "Storage service" and "Decision service". Rejected: keeping
  the box "Room model" and naming the rates something new, a term nothing
  else in the project uses. Decided and drawn 2026-10-06
  (`diagrams/container.d2`).
  - Useful for: §4.2.

- **Decided: occupancy is generated by its own process, separate from the
  physical model.** It writes the people in each room to BuildSim
  (`PUT /api/occupancy`); the physical-model process reads them back
  (`GET /api/occupancy`) and uses the count in the mass balance. Generating
  people from a time-of-day schedule and integrating the CO2 mass balance are
  unrelated jobs, and routing occupancy through BuildSim means the occupancy
  source can be replaced (see *Decided: occupancy comes from a time-of-day
  schedule*, §6) without touching the physical model. Rejected: computing
  occupancy inside the physical-model process — one container fewer, and the
  count is then always available where it is used, rather than fetched over
  the network where it can be missing or stale; but it ties the two together,
  so swapping the occupancy source becomes an edit to the model. Both
  processes stand in for the physical world and sit outside the system
  boundary, and the C4 diagrams draw them together in a box for the
  simulated room (see *Decided: the C4 diagrams draw the physical model and
  the occupancy simulator outside the system boundary…*, above). Decided and
  implemented 2026-09-20,
  `cmd/occupancy/`; the BuildSim calls it shares with the other Go services
  live in `internal/buildsim`.
  - The occupancy process covers the whole building rather than one room:
    BuildSim has no per-room occupancy endpoint, and a write replaces every
    room, erasing any left out — so exactly one process can own occupancy.
    This constraint applies to occupancy only; sensor values are written one
    device at a time and don't have it.
  - A cycle where the physical model can't read the occupancy — or the
    damper, or the current CO2 — is skipped and logged, leaving the CO2 in
    BuildSim as it was; the next cycle advances it by one `Δt` as usual, so
    the room's clock loses that step. This also covers the ordinary startup
    case, where the sensor and actuator processes haven't registered their
    devices yet. Rejected: advancing by the real time since the last
    successful cycle, which keeps room time honest but makes the process
    keep a timestamp between cycles, against *Decided: BuildSim is the only
    store of physical state*, above. Decided and implemented 2026-09-22,
    `cmd/physical-model/main.go`.
  - Useful for: §4.2, §4.4, §6.

- **Decided: robustness effort goes to the components inside the system
  boundary; the physical model and occupancy get only enough to keep a run
  going.** Both stand in for the physical world (see *Decided: BuildSim is
  the only store of physical state*, above), and a real building's CO2 has no
  failure modes to handle, so how well these two survive their own bugs isn't
  a property of the system. What they need instead is to be controllable,
  because the fault-injection tests work by making the simulated side
  misbehave on purpose — a CO2 value frozen at a plausible level, a room
  filling faster than the ventilation can clear it. Today that budget is
  `restart: on-failure` plus a skipped cycle on a failed read, and it is
  meant to stay there. Rejected: treating every process alike, which spends
  the same effort on the two components whose failures a real building
  doesn't have. Decided 2026-09-22.
  - Useful for: §4.4, §8.2, §10, §13.

- **Decided: the people counter (`cmd/occupancy-sensor`) takes the head
  count from BuildSim's occupancy (`GET /api/occupancy`), counting the
  people listed in its room.** The occupancy process already writes the
  true count there, so for this sensor that list is the physical world it
  observes. It reports the count as its own device value in BuildSim, the
  way BuildSim expects a device to report, and publishes it to MQTT.
  Rejected: the occupancy process also writing the count into the
  counter's device value, as the physical model does for the CO2 sensor —
  symmetric with the CO2 side, but it gives the occupancy process a second
  job and makes it depend on the counter being registered, while the count
  is already in BuildSim. Decided and implemented 2026-09-24; verified
  against the running stack (the count reaches BuildSim and the topic).
  - Useful for: §4.2, §4.4, §5.

- **Decided: each device process registers its own device with BuildSim on
  startup** — the sensor process registers the CO2 sensor, the actuator
  process registers the damper, the people counter registers itself. BuildSim keeps devices only in memory and
  starts with none, so they must exist before any value can be written or
  read. Rejected: the physical-model process registering all devices — it
  would make a restarted sensor or actuator depend on another process, where
  the course expects a crashed process to re-register itself. Trade-off: the
  physical model must wait and retry until the sensor exists. Decided
  2026-09-17; the call is `buildsim.Client.Register`, implemented 2026-09-22.
  Registration goes to `POST /api/equipment/bulk`, which skips IDs that
  already exist, rather than `POST /api/equipment`, which answers 409 for
  them. Re-registering is therefore a no-op: verified against a running
  BuildSim that a second registration returns `{"created":0,"skipped":1}` and
  leaves the value the device already holds untouched, which is what makes a
  restart safe mid-run.
  - **Known caveat —** a device process registers only on its own startup,
    so when BuildSim restarts mid-run and loses its devices, the sensors keep
    running but every read fails with 404 "sensor not found", and no
    readings reach storage until they are restarted by hand. Seen on the
    running stack 2026-09-27, when BuildSim was recreated while the sensors
    kept running; restarting `co2-sensor`, `occupancy-sensor` and
    `physical-model` restored the loop. The actuator came back only because
    it was recreated along with BuildSim. Noted 2026-09-27; not handled.
  - Useful for: §4.4, §5, §8.2, §10.

- **Known caveat — services aren't synchronized: each runs on its own
  schedule.** Any component can therefore see a reading that is stale,
  repeated, or skips a step, and has to tolerate that. Noted by 2026-09-05;
  no handling designed yet.
  - Useful for: §5, §8.1, §10 (timing/delay failure tests), §13.

- **Decided: two different communication patterns, chosen per link rather
  than one pattern system-wide.** Sensor → decision runs over
  publish/subscribe through a message broker; decision → actuator is a direct
  REST call. Reasoning: the sensor→decision stream may gain future
  consumers (dashboard, evaluation) that shouldn't require changing the
  producer, and the system already tolerates a late or dropped reading (see
  *Known caveat — services aren't synchronized*, above) — a fit for pub/sub's
  weaker, decoupled delivery. Decision→actuator is one producer to one fixed
  consumer, low frequency, and the decision service needs to know whether
  each command was applied — a fit for a REST call, whose answer says so,
  rather than a broadcast that can silently drop when the subscriber is
  offline. A command that wasn't applied is decided again on the next
  reading (`decide` in `cmd/decision/main.go`). Rejected: REST throughout
  (loses producer/consumer decoupling; requires hand-rolling delivery/retry
  logic per link) and broker throughout (a 1:1 low-frequency command doesn't
  get the multi-consumer benefit that justifies broker overhead, and plain
  pub/sub delivery can silently drop it). Replaces the earlier undecided
  single-pattern framing. Decided 2026-09-15.
  - The reason first given was a lost command with no next reading to
    correct it, so REST would retry until the actuator acknowledged. As
    built, the next reading does correct it, and the REST answer is what
    tells the decision service a command needs sending again.
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
  - **Decided: Eclipse Mosquitto as the MQTT broker**, run from its official
    Docker image. Chosen as the common default for small setups; no other
    broker was compared. Low stakes: every client speaks plain MQTT, so a
    different broker would need no code changes. Decided 2026-09-17.
  - **Decided: the decision service also publishes each command to MQTT** (on
    `<room_id>/ventilation/command`), alongside the REST call to the actuator.
    REST stays the actual delivery path — it's the link that needs confirmed
    delivery and confirmed execution, which MQTT would need extra machinery (a
    persistent session, a second confirmation topic) to match. The MQTT copy
    is for consumers that only need visibility (the storage-service, whose
    history the dashboard reads), which can tolerate a missed message the same
    way readings can. Rejected: replacing REST with MQTT for decision→actuator
    directly — even with guaranteed eventual delivery, a command queued while
    the actuator is offline can arrive stale (the situation it was decided for
    may no longer hold), and delivery to the actuator isn't the same as
    confirmation the command was executed. Decided 2026-09-16.
  - Useful for: §4.4, §5, §7.2.

- **Decided: everything derived from a room's floor area lives in one
  package, `internal/room`** — capacity `N_max`, volume `V`, and the airflow
  limits `Q_min` and `Q_max`. The occupancy process needs the capacity and
  the physical model needs all four, so the arithmetic is written once and
  both import it, while neither depends on the other. Rejected: keeping the
  capacity in `internal/occupancy`, which made the physical model import the
  weekday schedule and its random jitter to find out how many people fit; and
  putting it in `internal/co2`, which made the occupancy process depend on a
  CO2 package for a number unrelated to CO2. `internal/co2` is left holding
  only the mass balance. Decided 2026-09-21.
  - Replaces, the same day, `internal/occupancy` owning the capacity. The
    argument for that was that one five-line function is not a subject and a
    package without one collects whatever fits nowhere else. It stopped
    holding once the volume and both airflow limits turned out to be the same
    kind of value, derived the same way from the same input.
  - Useful for: §4.3, §4.4.
- **Decided: the exact solution of the CO2 balance is written once, in
  `internal/massbalance`, and both models call it.** The physical model
  (`co2.Step`) and the learned room model (`roommodel.Model.Predict`) are the
  same equation, `dC/dt = S − k·(C − C_out)`, with their own parameters:
  `S = G·N/V`, `k = Q/V` for one, `S = a·N`, `k = b0 + b1·d` for the other.
  The package holds only the solution, so the decision side shares
  mathematics with the simulator, not the simulator's room. Rejected: a copy
  in each model, about 20 lines each that could drift apart; and one model
  importing the other, which would make the decision service depend on
  simulator code, or the reverse. Trade-off: one more package for a single
  function. Decided and implemented 2026-10-08.
  - Useful for: §4.3, §7.1.

- **Decided: one decision-service instance per room.** It matches the other
  per-room services (sensors, actuator, room model), each
  room's inputs are independent, and a crash stops decisions for one room
  only. Rejected: one instance for all rooms — fewer containers, and the
  natural place for anything rooms must share, but one crash stops every
  room's decisions, and nothing is shared between rooms here. Trade-off:
  containers grow with the room count. Each small Go service uses about
  12–15 MB of memory (measured with `docker stats` on 2026-10-07), so 1000
  rooms at five per-room services is about 5000 containers and 65 GB
  (derived) — possible on a server, but too many to configure by hand in
  `docker-compose.yml`. The path to that scale is grouping rooms, e.g. one decision process per floor
  subscribing to `level0/+/co2/reading`, which fails per floor instead of
  per room.
  Decided and implemented 2026-09-26 (`cmd/decision`).
  - Replaces *Deferred, not yet decided: one decision-service instance per
    room, or one instance handling all rooms* (proposal sections 3 and 6).
    Storage and the dashboard have since been decided (§7, §12).
  - Useful for: §4.2, §4.4, §9, §13 (scaling).

- **Decided: scaling to more rooms works differently on each side of the
  system boundary — the physical model reads the floor's rooms from BuildSim
  and covers them all, while the sensors and actuators get one process per
  room.** The physical model stands in for the physical world, which is one
  thing, and `GET /api/building/floors/{level}` already lists every room with
  its area, so a second room needs no configuration and `ROOM_ID` leaves
  `sim.env`. The sensors and actuators are inside the system, where each
  device is to be independently restartable, so each gets its own service
  block in `docker-compose.yml` with `ROOM_ID` in that block's
  `environment:`, overriding the shared `sim.env`. Rejected: one process per
  room for the physical model too — a stopped room would then affect only
  itself, but every room has to be listed in compose, duplicating what
  BuildSim already knows; and one process holding a list of rooms for the
  sensors and actuators — fewer containers, but one crash removes every
  room's device at once. Occupancy already covers the whole building for a
  reason of its own (see *Decided: occupancy is generated by its own process,
  separate from the physical model*, above). Planned 2026-09-22; not
  implemented, the system runs one room.
  - Useful for: §4.2, §4.4, §9, §13.

- **Decided: report §4.3 is a table of every container's one job and its
  reason for separation, not a C4 Level 3 component diagram.** The course's
  report guide requires that list (section 3.4) and calls a Level 3
  diagram optional (section 2.2); the course's example report puts the
  table in §4.3. Rejected: a Level 3 diagram of the Decision service, whose
  internals §7.1 covers anyway. Decided 2026-10-07.
  - The CO2 sensor is separate from the Physical model because it is the
    only part a real sensor replaces, so moving to real hardware changes
    one container; and separate from the Occupancy sensor so that a crash
    of either leaves the other's readings flowing. Decided 2026-10-07.
  - The Damper actuator is the interface between the Decision service and
    the damper, simulated or real: the Decision service only sends it a
    level and never calls BuildSim, so a real damper changes the actuator
    alone. Decided 2026-10-07.
  - The Dashboard is separate from the Storage service because showing data
    is a different job from storing it: it reads history through the
    Storage service's REST endpoints, so a crashed Dashboard loses nothing.
    Rejected: the Storage service serving the page, one container fewer.
    Decided 2026-10-07.
  - The Decision service is separate from the sensors and the Damper
    actuator so that a crash in the decision logic leaves the sensors
    publishing and the damper at its last level; after a restart it
    decides again from the next CO2 reading, with the retained room model.
    Decided 2026-10-07.
  - The Room-model service's row gives the reason in *Decided: the fitting
    runs in its own service*, §7: a slow or failing fit cannot delay a
    damper command. Rejected for the row: that it is not needed for basic
    operation, which says why its failure is cheap but not why it runs
    apart from the Decision service. Decided 2026-10-07.
  - Useful for: §4.3.

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
  against it. Decided and implemented 2026-09-15; the current schemas are
  the files in `schemas/`.
  - `ventilation_command`'s `level` field (0–1) was a placeholder; now
    settled to match the damper state in BuildSim (see *Decided: room A125's
    devices in BuildSim...*, below).
  - Runtime validation against the schemas is implemented in the
    storage-service (2026-09-17, `schemas/schemas.go`) and in the actuator,
    which checks every command before writing the damper (2026-09-23).
    - **Revisit:** the decision service was written without the check: it
      reads CO2 readings, head counts and room models with `json.Unmarshal`
      alone (`decode` in `cmd/decision/main.go`), so a reading missing its
      `ppm` field reads as 0 ppm. The room-model service goes without it:
      its history comes from the storage-service, which checked each
      message before saving.
  - Only received messages are validated, not published ones: a publisher
    fills a Go struct whose fields are the schema's fields, so its own output
    can't fail the check, while a receiver is handed bytes another process
    wrote and would otherwise let `json.Unmarshal` fill a missing field with
    a zero — a command with no `level` reads as a valid "close the damper".
    - Each message is therefore defined twice over, as a schema and as a Go
      struct (one shared struct per message, see *Decided: each message has
      one Go type...*, below). A test in `internal/message/message_test.go`
      marshals a filled struct of each type and validates it against its
      schema, so a field renamed in one but not the other fails `go test`
      rather than every receiver rejecting the message at runtime. Every
      service that uses the types runs this test in its Dockerfile before
      building, so a mismatched image can't be built. Rejected: running it
      only in CI on push, which leaves a local build unchecked. Also rejected:
      generating the structs from the schemas, which removes the second
      definition altogether but adds a build step. Noted 2026-09-23;
      decided and implemented 2026-09-28.
  - Useful for: §5, §7.2, §10.

- **Decided: each message has one Go type, in `internal/message`, shared by
  every service that sends or receives it.** A field is then changed in one
  place, and the compiler flags every service that no longer agrees with it.
  Before, the CO2 reading, head count and command were each written out
  separately in up to four services. Rejected: a copy per service, which
  keeps each service's source self-contained but lets the copies drift apart
  unnoticed. The cost is a build-time link: changing a type changes every
  service that uses it at its next image build. Running containers are
  unaffected, since each program holds its own compiled copy and still
  restarts on its own; what forces services to be updated together is a
  change to the JSON itself, which holds either way. Decided and implemented
  2026-09-28.
  - The store (`internal/store`) saves and returns these same types, so a
    reading keeps one type from its sensor through storage to the room
    model. It replaces a second set of row types the store had defined for
    itself, which the storage-service converted to and from. Implemented
    2026-10-04.
  - Useful for: §4.4, §5.

- **Decided: room A125's devices in BuildSim are a CO2 sensor (value in
  ppm) and a ventilation damper (state 0–1, 0 closed, 1 fully open)**, with
  the IDs `CO2_SENSOR_ID` and `DAMPER_ID` in `sim.env`. The physical model
  writes the sensor value and reads the
  damper state; BuildSim stores both as text, so each side converts to and
  from a number. The 0–1 range matches `ventilation_command`'s `level`, so a
  command passes to the damper unchanged. Simple defaults, not weighed
  against alternatives; settled early so the physical model can be built
  before the sensor and actuator processes exist. Decided 2026-09-17.
  - Useful for: §5, §6.

- **Decided: every MQTT topic starts with the room as BuildSim names it,
  floor included, then the kind of message: `<room_id>/co2/reading`,
  `<room_id>/occupancy/reading`, `<room_id>/ventilation/command` and
  `<room_id>/model`, e.g. `level0/A125/co2/reading`.** The floor and the room
  are two topic levels, so the start of every topic is exactly the message's
  `room_id`, and the room has one name throughout the system. Location
  first, broadest to narrowest, also gives the useful wildcards:
  `+/+/co2/reading` for every room's CO2 readings, `level0/+/co2/reading`
  for one floor's, `level0/A125/#` for everything about one room. Rejected: the room's name
  alone (`A125/co2/reading`), which leaves two names for one room. Decided
  and implemented 2026-10-07 (each service in `cmd/`; the storage-service
  checks that a topic starts with its message's `room_id`).
  - The room ID is one setting, `ROOM_ID=level0/A125` in `sim.env`, which
    every service reads. Only the `internal/buildsim` client splits it into
    floor and room, since BuildSim's device records and floor data keep the
    two apart. Each service's MQTT client ID is `<room_id>/<service>`, e.g.
    `level0/A125/decision`, unique per room. Rejected: separate floor and
    room settings joined by each service, which made services that never
    call BuildSim import its client to join them. Decided and implemented
    2026-10-07.
  - Replaces `co2/<room>/reading` with the room's name alone in the middle
    level (2026-09-22). That choice weighed only the room's name against the
    full key squeezed into one level, where its slash would break the `+`
    wildcard, and missed giving the floor and the room a level each.
  - Useful for: §5, §7.2.

- **Decided: the actuator serves one endpoint, `PUT /command` on port 8080,
  taking a `ventilation_command` and answering 200 OK once it is applied.**
  The answer is sent only after BuildSim has stored the new damper position,
  so a caller that gets any other status knows the command did not land and can send it again — the retry the REST
  link was chosen for (see *Decided: two different communication patterns...*,
  §4). A command for another room is refused with 404 rather than obeyed: one
  actuator process serves one room, so a foreign `room_id` means something is
  misrouted. Malformed or schema-invalid commands are refused with 400 and a
  failed BuildSim write with 502, each with the reason as plain text.
  Rejected: accepting on receipt and writing the damper afterwards, which is
  cheaper to serve but leaves the decision service unable to tell a stored
  command from a lost one. Decided and implemented 2026-09-23.
  - Replaces, 2026-10-05, a JSON answer (`ventilation_command_response`)
    whose `accepted` field was true exactly when the status was 200, and
    whose timestamp nothing read. Rejected: keeping it, which documents the
    answer as a schema but states the outcome twice.
  - Replaces, 2026-10-07, `POST /command`, which was never weighed against
    PUT. A command sets the damper's level rather than creating something,
    and HTTP defines PUT as the method that is safe to repeat, as a resent
    command is. Nothing behaves differently; the method only tells a reader
    of the contract what to expect.
  - Useful for: §5, §8.1, §10.

- **Decided: the BuildSim client keeps BuildSim's nested
  `Equipment`/`Sensor`/`Actuator` shape rather than a flat `Device` with one
  register method per kind.** BuildSim's shape is general — one equipment
  record can hold several sensors and actuators — but every device here is one
  record holding one device, so a caller writes `ID`, `Name`, and `Type` twice
  in two literals that have to agree. BuildSim accepts a record whose
  equipment ID and sensor ID differ, so a mismatch fails silently: values are
  written to an ID the 3D view is not reading. Rejected: a flat `Device` plus
  `RegisterSensor`/`RegisterActuator`, which makes that mismatch
  unrepresentable and drops two fields that are always the same constant
  (`Category`, `DataType`), at the cost of the client's exported types no
  longer mirroring BuildSim's API and of a `Unit` field meaning nothing for
  actuators. Kept because the duplication is only written once per device
  type, the call sites existed and were correct, and the device list is
  fixed per room — so the flat shape would
  guard against a mistake that is already past, while making the client
  harder to check against BuildSim's own API. Noted 2026-09-22 as open;
  decided 2026-09-23 with the sensor and actuator processes written.
  - **Revisit:** the people counter (2026-09-24) is a third device type, so
    the duplication has returned with it (`equipment` in each device
    process); the flat shape has not been weighed again.
  - Useful for: §5.

## 6. Simulating the sensor values (the physical model)

- **Decided: CO2 comes from a simple mass-balance model, not a replayed
  dataset.** CO2 rises with the people in the room and decays toward an
  outdoor baseline at a rate set by the ventilation level. Rejected:
  replaying a recorded CO2 dataset — a fixed trace can't respond to the
  actuator, so the control loop wouldn't close. Proposal section 4,
  2026-09-04.
  - Replaces the proposal's rise of roughly 40 ppm per person per hour, the
    course's guidance for a typical room. Before ventilation removes any,
    one person raises CO2 by `G/V`, so the rate depends on the room's air
    volume: ≈ 282 ppm/h in A125's 71.5 m³, and 40 ppm/h only in a room of
    about 500 m³ (both computed from `G` in the entry below). Settled
    2026-10-08.
  - Useful for: §6.

- **Decided: the mass balance is `V·dC/dt = G·N − Q·(C − C_out)`, advanced
  each cycle with its exact solution `C_next = C_steady + (C − C_steady) ·
  exp(−Q·Δt/V)`, where `C_steady = C_out + G·N/Q`.** `C` is room CO2 (ppm),
  `N` the number of people, `C_out` outdoor CO2 (about 420 ppm), `G` the CO2
  one person breathes out per unit time, `Q` the airflow through the room,
  and `Δt` the room time between cycles. `N` and the damper are read once
  per cycle and held fixed within it, so the solution is exact for any `Δt`:
  accuracy doesn't depend on the cycle length or on running the room faster
  than real time. What `Δt` still sets is how late the model notices a
  change in `N` or the damper. Decided 2026-09-17, exact solution 2026-09-18,
  implemented 2026-09-21 as `co2.Step` in `internal/co2/step.go`, which
  since 2026-10-08 hands the solving to `massbalance.After` (see *Decided:
  the exact solution of the CO2 balance is written once…*, §4).
  - Replaces forward-Euler stepping, `C_next = C + Δt·(G·N/V − (Q/V)·(C −
    C_out))`. That is only accurate while `Δt` is much smaller than the time
    constant `τ = V/Q` (≈ 10 min in A125 with the damper open), so a faster
    room would have needed more cycles, and more requests to BuildSim, to
    stay accurate.
  - The claim that `Δt` is free is tested: in `internal/co2/step_test.go` and
    `internal/massbalance/massbalance_test.go`,
    one 10-minute step and sixty 10-second steps agree to within 0.01 ppm.
    Under forward Euler those two would differ, so this is the test that
    distinguishes the two methods rather than just exercising the code.
  - **`V` and `N_max` are computed from the room's floor area in BuildSim**
    (`V = area · 2.4 m`, `N_max = round(area / 5 m²)`; for A125: 71.5 m³ and
    6 people), so a second room needs no new configuration. Rejected:
    writing both values per room into `docker-compose.yml` or a `.env` file.
  - **Values not read from BuildSim are environment variables, loaded by
    Docker Compose from one shared file**, so each value is written once and
    every service that uses it reads the same number. It holds the
    simulation's assumed values, such as ceiling height and `G`, and settings
    of the system that several places read, such as the threshold. Only values
    from BuildSim count as facts about the building; the rest are assumptions,
    so they should be changeable between runs without a rebuild. Rejected: a
    shared config file (YAML/JSON) mounted into each container — allows
    grouping and comments, but needed parsing code in both Go and Python, the
    forecast service's planned language at the time. Decided 2026-09-18.
    - Replaces fixing ceiling height and area per person as constants in a
      shared Go package, which treated them as facts about the building
      rather than assumptions.
    - Implemented 2026-09-20 as `sim.env`, read by Docker Compose through
      `env_file`. It holds only what more than one service needs; a service's
      own wiring, such as the BuildSim URL and its update interval, stays in
      its `environment` block in `docker-compose.yml`.
    - The two device IDs joined it 2026-09-23. They had matched only because
      the writer and the reader of each device built the same fallback string
      independently, which fails silently when one changes: BuildSim answers
      both sides normally and the value simply never meets.
    - The physical model's parameters joined it 2026-09-22, which settled the
      naming: a name carries the unit where the value has one
      (`CEILING_HEIGHT_M`, `CO2_PER_PERSON_LPS`), and each entry has a
      comment saying what the value is and where it comes from.
      `MAX_AIRFLOW_FACTOR` was named `AIRFLOW_MARGIN` first; "margin" reads
      as a distance in ppm below the threshold, which is how the same word
      had already produced a wrong comment on `room.MaxAirflow`, while the
      value multiplies an airflow.
    - **Decided: every setting is required, with no fallback value in the
      code** (`internal/env`). A setting that is missing stops the service at
      startup with `<NAME>: not set`, so each value lives only in `sim.env`
      or `docker-compose.yml`. Rejected: a fallback per setting in the code,
      which lets a service start without Compose but keeps a second copy of
      each value that drifts silently, as the occupancy sensor's interval
      had (10 s in the code, 1 minute in Compose). Nothing in the project
      starts a service without Compose; the unit tests pass their values in
      directly. The listen ports are fixed in the code instead of being
      settings: each is the container's own port, which Compose's port
      mappings and URLs already name, and a different port on the host is
      set by the mapping alone. The session clock written by `start.sh` is
      the exception: unset means no session, so room time is the wall clock
      (`internal/roomtime`). Decided and implemented 2026-10-05.
  - **Ceiling height 2.4 m:** BuildSim gives no height. This is the minimum
    the Swedish Work Environment Authority advises for workplaces (general
    advice to chapter 3, section 5 of AFS 2023:12, p. 17,
    https://www.av.se/globalassets/filer/publikationer/foreskrifter/utformning-av-arbetsplatser-afs2023-12.pdf),
    so it gives the fastest plausible CO2 rise. Rejected: an unsourced 3 m
    guess.
  - **5 m² per person** is an estimate, not sourced — roughly what a meeting
    room allows; those workplace rules give no figure per person.
  - **Airflow `Q = Q_min + damper·(Q_max − Q_min)`, with `Q_max = 1.2 ·
    G·N_max / (C_threshold − C_out)`** (≈ 70 L/s in A125), so `Q_max` is computed
    from floor area like `V`, `N_max` and `Q_min` and a second room needs no
    new configuration. The lower bound `Q_max > G·N_max /
    (C_threshold − C_out)` follows from the steady state `C_steady = C_out +
    G·N/Q`: fully open at `N_max`, CO2 must settle below the threshold, or
    the damper reaches its limit with the room still above it and the
    decision service has nothing left to command. The factor 1.2 is a safety
    factor on that bound, a judgment rather than a sourced figure: the fan
    holds a full room under the threshold with some reserve and is no larger,
    per *Decided: the system pursues three goals…*, §7. A full A125 is then
    held at 1000 ppm with the damper at about 0.8. Sizing `Q_max` this way
    does not keep the room below the threshold — with the damper closed it
    still settles far above it (≈
    3650 ppm at `N_max` in A125) — it only makes opening the damper able to
    bring it back under, which is what leaves the decision service something
    to decide. `Q_min` is the airflow with the damper closed; without it
    `Q = 0` and CO2 rises without limit. Rejected: taking `Q_max` from a real
    ventilation device; and choosing a value per room, which doesn't carry
    over to more rooms. Decided 2026-09-21; factor lowered to 1.2 and
    implemented 2026-09-24 (`MAX_AIRFLOW_FACTOR` in `sim.env`).
    - Replaces a factor of 2, an unsourced margin that made the fan larger
      than any goal needed.
    - `V` and `Q_max` both scale with floor area, so they cancel in
      `τ = V/Q_max`: every room clears at the same rate, ≈ 17 min with
      factor 1.2, so how early a room must be ventilated ahead of a meeting
      is the same for every room.
  - **`Q_min` = 0.35 L/s per m² of floor area** (≈ 10 L/s in A125), so it
    is computed from floor area like `V` and `N_max`. AFS 2023:12, chapter
    5, section 4 (p. 39), gives this as the minimum outdoor airflow per m²
    of floor at workplaces, on top of the airflow per person; FoHMFS 2014:18
    (https://www.folkhalsomyndigheten.se/contentassets/641784832543443ea4eebe9b300c244e/fohmfs-2014-18.pdf,
    p. 1) gives the same for schools and childcare premises. It covers
    pollution from building materials. A closed damper therefore means
    ventilation at a low background level, not fully off. With it, CO2 in
    A125 settles around 3650 ppm for 6 people and around 960 ppm for one
    person. Rejected: a fixed share of `Q_max` (e.g. 10%),
    which is simpler but has no source; and leakage through the building
    shell, which matches "closed" literally but has no source checked yet.
    Decided 2026-09-18.
  - **CO2 per person `G` = 0.0056 L/s** (≈ 282 ppm/h per person in A125 with
    no ventilation), computed from Persily & de Jonge, "Carbon Dioxide
    Generation Rates from Building Occupants", Healthy Buildings 2017 Europe
    (https://tsapps.nist.gov/publication/get_pdf.cfm?pub_id=922955; journal
    version in Indoor Air 27(5), https://doi.org/10.1111/ina.12383). The
    paper puts seated office work at 1.5 met (section 3, p. 3); its Table 2
    (p. 4) lists rates by age and sex at 1.4 and 1.6 met but not 1.5.
    Averaging those two columns over men and women aged 21–60 gives
    ≈ 0.0052 L/s, the ASHRAE 62.1 value the paper cites (p. 3). The paper
    states its rates at 273 K; at 20 °C the same CO2 fills 0.0056 L/s. Both
    figures are computed here, not stated in the paper.
  - **`Δt` = 10 s of room time**, not real time, so the model notices a
    change in `N` or the damper within the same room time at any speed;
    only the real wait between cycles (and so the traffic to BuildSim)
    shrinks as the room runs faster. 10 s is a default chosen to be short
    compared with how often people arrive or leave (minutes), not tuned.
    Rejected: `Δt` in real time, where a faster room would notice changes
    later in room time. Decided 2026-09-18.
  - `C_threshold` is set in *Decided: the system pursues three goals…*, §7.
  - Useful for: §4.4, §6, §10, §13.

- **Decided: room time runs at a speed chosen per session, and each session
  continues the room timeline where the previous one stopped.** A session
  is started with `./start.sh <speed>`: the script asks the storage-service
  for the latest stored timestamp, and the session begins at that room time,
  or at the fixed origin 2026-09-25 09:00 (Stockholm time) if nothing later
  is stored. It records three values for every container: the wall-clock
  moment the session started, the room time it started at, and the speed.
  Every process computes room time = room start + speed × (wall clock now −
  session start), so all processes agree without talking to each other, and
  a restarted container rejoins the same clock. Room time stands still
  between sessions, so the stored history stays one continuous timeline
  across shutdowns and speed changes. Decided 2026-09-24; the clock
  (`internal/roomtime`), `start.sh`, and the storage-service's
  `GET /latest` implemented the same day, and every service switched to the
  clock 2026-09-25. Verified: `./start.sh 10` wrote the session to `run.env`
  and recreated the project's services while BuildSim and the broker kept
  running.
  - **Decided: intervals that belong to the simulated room run on room time;
    waits that belong to the software run on real time.** Room time covers
    the sensors' sampling interval, the physical model's step, the occupancy
    update, and the room model's daily fit; every timestamp in a message is
    room time. Real time covers retry waits, such as the room model's pause
    between tries at the storage-service. So speed doesn't change how often the room
    is sampled per room hour, and the stored history is equally dense at
    any speed. Rejected: sensor intervals in real time, where a faster room
    gets fewer readings per room hour, so the chosen speed would change the
    system's behavior. Trade-off: message traffic grows with speed (at 60×,
    a 10 s sensor interval is one message every 0.17 real seconds). Decided
    and implemented 2026-09-25.
  - Why faster at all: testing and evaluation cover many room days, and the
    room model is fitted on 7 days of history (see *Decided: each fit uses
    the last 7 days of history*, §7), a real week at normal speed. Trade-off:
    the system's real delays stretch in room time (a 1 s delay becomes a
    minute at 60×), so a fast run misrepresents how the system would perform
    in a real building.
  - Rejected: normal speed only, with the forecast's history written
    directly from the occupancy schedule — much simpler, but every
    evaluation run would take real days. Rejected: one fixed instant where
    room time equals wall-clock time, with room time = instant + speed ×
    elapsed — no start script, but a run can't start at a chosen room time,
    and changing speed makes room time jump past or behind the stored
    history. Rejected: each process counting room time from its own start —
    a restarted process would fall behind the others. Rejected: a clock
    service the other processes ask — one more container that every process
    depends on.
  - **Decided: every service restarts by itself after a crash but not when
    the PC boots (`restart: on-failure` in `docker-compose.yml`).** Only
    `./start.sh` starts a session, so shutting the PC down never causes a
    jump. With `restart: unless-stopped`, Docker started the services again
    at boot with the previous session's values, and room time leapt ahead by
    the speed times the hours the PC was off. Rejected: keeping
    `unless-stopped` and stopping the system by hand before every shutdown —
    one forgotten stop gives a silent jump; with `on-failure`, forgetting
    `./start.sh` just leaves the simulation off. Decided and implemented
    2026-09-24.
    - Fixed 2026-09-28: every service now exits with code 0 when Docker
      stops it (`internal/shutdown`). Before, a Go program that is its
      container's main process exited with code 2, which Docker counts as a
      failure, so at boot it started every project service again. Without
      the broker and BuildSim, which exit with code 0 and stayed off, those
      services exited and restarted in a loop until `./start.sh`. Docker's
      wait between restarts grows to about a minute, longer than the 30 s
      `./start.sh` waits for the storage-service, so `./start.sh` failed
      once. Rejected: a longer wait in `./start.sh`, which hides the loop
      instead of removing it. Checked: the dashboard and storage-service
      exit with code 0 when stopped. A crash still exits with another code
      and is restarted; at that boot the storage-service restarted 12 times.
    - `docker kill` counts as a manual stop too: Docker doesn't restart the
      container whatever its exit code. A crash test has to kill the process
      from the host instead (`sudo kill -9` on the PID from `docker inspect`).
  - Changing speed means running `./start.sh` again. That recreates only the
    project's own services, which read the three values; BuildSim and the
    broker keep running, so the room's CO2 and damper carry on.
  - **Decided: a session can start at a chosen room time, later than the
    stored history and at most `FIT_DAYS` (7 days) after it**
    (`./start.sh <speed> <room time>`), so the demo can begin just before a
    weekday's 13:00 meeting. The skipped room time is a gap in the history.
    The room model's fit pairs only readings under a minute apart, so the
    gap adds nothing to it, and the limit keeps stored history inside the
    fit's 7-day window. The room's CO2 in BuildSim carries over the gap
    unchanged, as between any two sessions. Rejected: an earlier start,
    which would write readings at room times already stored; and running
    fast up to the meeting and stopping by hand, which needs no code but a
    stop timed by eye, and every rehearsal uses up that day's meeting.
    `FIT_DAYS` moved from the Room-model service's block in
    `docker-compose.yml` to `sim.env`, since `start.sh` reads it too.
    Decided and implemented 2026-10-08.
  - **Known caveat —** a full shutdown restarts BuildSim, which keeps state
    only in memory, so the room's CO2 resumes from outdoor air while room
    time continues. Ending sessions outside office hours hides this, since
    an empty room's CO2 decays toward outdoor air overnight anyway. Seen
    after the 2026-09-27 reboot: the empty room's CO2 stepped from 454.3 to
    420 ppm between two readings.
    - **Revisit:** the room model's fit treats that step as a real change.
      The two readings were 24 room seconds apart, inside the fit's 1-minute
      limit, with the damper and head count unchanged, so the step enters
      the next seven fits as a drop of 86 ppm per minute. At night the
      effect is small; a reboot with people present would give a far larger
      false drop, pulling the learned clearing rates up. Size of the effect
      not measured.
  - Useful for: §6, §9, §11.

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
  - **The schedule for A125** (weekdays; empty at weekends; the times are
    constants in `internal/occupancy/schedule.go`): half the room's capacity,
    rounded up, are regulars, who arrive one at a time in the morning, leave
    together for lunch, so the room empties rather than thins out, and leave
    one at a time in the afternoon. The rest of the capacity join a meeting
    13:00–14:00, filling the room to its 6-person maximum. Each of a person's
    times is shifted by up to ±20 minutes, drawn from a random generator
    seeded with the date. `PeopleAt` keeps no state and draws the shifts again
    on every call, every 10 room seconds; the date seed makes every call on
    one day draw the same shifts, so each person keeps one arrival time all
    day, and a restarted Occupancy simulator carries on with the same day. The
    13:00 meeting is the scenario the demo turns on: six people need the
    damper at about 0.9 to stay under the 950 ppm target, against about 0.4
    for three (the stored commands of 2026-10-22), so it is the hardest case
    the decision service has to hold. Times are room time, so they follow the
    simulation speed. Plausible office hours, not taken from a source or a
    measured building. Decided and implemented 2026-09-20,
    `internal/occupancy/schedule.go`, with unit tests covering the counts at
    known times, the weekend, the lunch dip, and that the room fills to
    capacity exactly once a day.
  - **Revisit:** adopt a different occupancy source later if the schedule
    turns out too simple to be realistic (a risk in proposal section 6) —
    either the course's `occupancysim/` or a public occupancy dataset.
  - Useful for: §6, §13.

## 7. Autonomous services and data pipeline

- **Decided: the system pursues three goals — CO2 stays under 1000 ppm at
  all times, the fan runs as low as it can (noise and energy), and the fan
  is no larger than the first goal needs (cost) — and the decision service's
  plan serves the second.** The fan is sized by the first goal alone: a room
  can fill without warning, so the fan must hold a full room (`N_max`) under
  the threshold by itself, with a safety factor on top. On every reading,
  the decision service picks the lowest damper level that its learned room
  model predicts will hold the people counted now under the threshold (see
  *Decided: on every CO2 reading, the decision service picks the lowest
  damper level…*, below). Rejected: a fan smaller than a full room needs,
  relying on a forecast to clean the room beforehand — a room that fills
  unexpectedly then goes over the threshold. Decided 2026-09-24; implemented
  2026-09-26.
  - Replaces, 2026-09-27, an occupancy forecast serving the second goal:
    ventilating moderately ahead of a predicted meeting, so the fan could
    run lower during it, with a reactive mechanism for unexpected occupancy
    (see *Decided: the occupancy forecast is removed…*, below).
  - Replaces the proposal's plan (proposal sections 1 and 5): a CO2
    forecast 30 minutes ahead, compared to the threshold so ventilation
    starts before it is crossed. In this simulation that gains nothing over
    reacting at the threshold. A new damper position takes effect at the
    physical model's next step, and a fully open damper reverses a full
    room's rise at once: in A125 with 6 people at 1000 ppm, CO2 rises about
    23 ppm/min with the damper closed and falls about 28 ppm/min fully open
    (with the fan factor then at 2, see §6), so opening when a reading reaches 1000 overshoots by about 10 ppm. A
    delay between command and effect, or a weaker fan, would give that
    forecast work, but only by creating the problem it then solves. A
    forecast from recent CO2 also can't see a rise before people arrive;
    the daily occupancy pattern can.
  - **Threshold `C_threshold` = 1000 ppm.** The Public Health Agency of
    Sweden's general advice on ventilation, FoHMFS 2014:18
    (https://www.folkhalsomyndigheten.se/contentassets/641784832543443ea4eebe9b300c244e/fohmfs-2014-18.pdf),
    treats CO2 regularly above 1000 ppm in normal use as a sign of
    inadequate ventilation (stated for schools and childcare premises). The
    workplace rules cited for the ceiling height (AFS 2023:12) give no CO2
    limit, only a minimum outdoor airflow per person and per m². No
    alternative value was weighed. Decided 2026-09-18.
  - **Decided: occupancy is sensed by its own sensor process**, reporting
    the room's head count the way the CO2 sensor reports CO2. The room
    model's fit and the plan both need the number of people, which CO2
    can't give: past CO2 also records what the damper did, so the same
    reading can mean six people with the damper open or two with it closed;
    the head count doesn't depend on the damper. No other way of obtaining
    occupancy was weighed. Decided 2026-09-24.
    - **Decided: the occupancy sensor counts once a minute of room time**,
      against 10 s for the CO2 sensor. People arrive one at a time, so the
      count changes a few dozen times a day. The physical model reads the
      head count from BuildSim every step, so the physics doesn't depend on
      this interval. Rejected: 10 s like the CO2 sensor — six times the
      stored history, for no gain in the occupancy forecast the interval was
      first chosen for. Trade-off: an arrival reaches the decision service
      up to a minute late, and with the forecast removed the plan learns of
      every arrival this way. Decided and implemented 2026-09-25
      (`SENSOR_INTERVAL` in `docker-compose.yml`).
  - **Decided: the decision service plans with a room model learned from
    stored data, never with the simulator's own values.** The model has the
    shape of any well-mixed ventilated room, `dC/dt = a·N − (b0 + b1·d)·(C −
    C_out)`, with `d` the damper level. Its three numbers are fitted by least
    squares from stored CO2 readings, head counts and damper levels: `a`, how
    fast CO2 rises per person; `b0`, how fast the room clears with the damper
    closed; `b1`, how much faster per unit of damper opening. The volume,
    CO2 per person and airflows the physical model uses (§6) are never given
    to it, so the decision logic would carry over to a real building, which
    provides the same three data streams, and how well the model is learned
    can be measured. Rejected: the physical model's equation and values,
    which make the decision work by construction and show nothing. Rejected:
    a model with no physics, such as a table of the CO2 level each head count
    and damper level has led to — it assumes nothing about the room, but
    needs far more history and can't plan for a head count or starting level
    it hasn't seen. Decided 2026-09-25; the fit implemented 2026-09-25
    (`internal/roommodel`, with unit tests), run by `cmd/room-model`.
    - **Decided: each step, from one CO2 reading to the next, gives one
      equation for the fit**, using the head count and damper level last
      stored at or before the step's start (each holds until the next is
      stored), and the average of its two readings as the step's CO2 level.
      A step is left out when it is longer than a minute, when the head
      count or damper level changed during it, or when either is not yet
      known. On 8 simulated hours the fit recovers the physical model's
      values to within 0.01%. Rejected: averaging all three streams onto a
      1-minute grid first — it discards CO2 detail and blends the minutes
      where the damper or head count changed into rows that match neither.
      Trade-off: the head count is stored once a minute, so an arrival shows
      up to a minute late, and the steps in that minute carry the old count.
      The fit refuses to answer when the history can't separate the three
      numbers, e.g. with the damper never moved. Decided and implemented
      2026-09-25.
    - **Known caveat —** the model's shape matches the simulated room
      exactly, so the fit will be close to exact here. A real room departs
      from it (uneven mixing, open windows, a slow sensor). The shape is
      standard ventilation physics, not taken from the simulator; the
      numbers are what the simulator sets.
    - **Decided: the fitting runs in its own service, `room-model`**: at
      its start and at each room midnight it fetches history from
      the storage-service, fits the model, and publishes the three numbers
      as a retained MQTT message. A slow or failing fit then can't delay a
      damper command, and the decision service keeps planning with the last
      published numbers while the learner is down. Rejected: fitting inside
      the decision service — one container fewer to build, test and draw,
      but it mixes a slow batch job with the per-minute loop. Trade-off: one
      more container and message format, and the decision service must
      handle a missing or outdated model. The rates are published on `<room_id>/model`
      (`schemas/room_model.schema.json`). When the history can't separate
      them, nothing is published and the last model stays retained. Decided
      2026-09-25; implemented 2026-09-26 (`cmd/room-model`). Verified against
      the running stack with one person present and the damper set by hand to
      closed, fully open and half open: `a` = 4.701 ppm/min per person
      (physical model 4.698), `b0` = 0.00908 per min (0.00875, 4% high),
      `b1` = 0.0486 per min (0.0496, 2% low).
      - **Revisit:** the unit tests recover all three to within 0.01%, the
        running stack only to within 4%. The cause is not yet checked.
    - **Decided: each fit uses the last 7 days of history.** Every usable
      step between CO2 readings adds to the same few sums, so a longer window
      costs one larger fetch (about 60,000 readings and 10,000 head counts),
      not a harder fit. With a noise-free simulated sensor, far fewer steps
      would give the same rates; the window is sized so it always holds
      weekdays with people present and the damper at more than one level.
      Rejected: only the last day — after a weekend it holds an empty room
      and an unmoved damper, which the fit refuses. Rejected: all stored
      history — the model would never forget a room that has changed.
      Trade-off: after a change, such as the fan losing capacity, the fit
      blends old and new behavior for several days. Decided 2026-09-26.
      - **Revisit:** a model that adapts over hours rather than days. With
        one fit per room day on 7 days of history, a change to the room
        takes about 4 room days to mostly show. Two ways to shorten it:
        refit more often on a shorter window fetched from storage, or
        subscribe to the readings and update the sums at every step, with
        old steps fading out. Either way the shorter memory must still hold
        the damper at more than one level, or the fit can't separate the
        rates.
  - **Decided: on every CO2 reading, the decision service picks the lowest
    damper level that keeps predicted CO2 under 950 ppm for the next 60
    minutes, for the people counted now, holding that level the whole
    time.** The prediction starts from the current reading and expects the
    latest head count the whole hour (nobody before the first count); levels are tried
    from closed upward in steps of 0.05, and fully open is chosen when none
    lower is enough. The plan is made again on every reading, so an arrival
    or departure changes the level as soon as the occupancy sensor counts
    it, before the arrivals' CO2 has built up. The target sits below the
    1000 ppm threshold so that small errors in the head count or the fit
    don't push a room held at the target over it. A reading at or above 950
    ppm opens the damper fully, since no lower level keeps CO2 under it,
    which covers a wrong model or a failed counter. Both numbers are
    untuned defaults (`PLAN_TARGET_PPM` in `sim.env`, shared with the
    dashboard, and `PLAN_HORIZON` in `docker-compose.yml`); 60 minutes is longer than the 20–40 minutes a
    moderately open damper takes to clear the room. Rejected: a level for
    every minute ahead, chosen to use the least fan in total — it needs an
    optimization solver and is harder to test and explain. Rejected:
    opening fully when a reading reaches the threshold — simpler, but it
    runs the whole fan for one extra person, against the second goal, and
    waits until CO2 is at the limit. Trade-off: the plan assumes the people
    counted now stay the whole hour and nobody else arrives, and it relies
    on the head count and the room model being right; if either is wrong,
    only the full opening at 950 ppm is left. Decided and implemented
    2026-09-26 (`internal/planner`, with unit tests; used by `cmd/decision`);
    planning from the head count alone since 2026-09-27. Seen on the running
    stack on 2026-10-06, a day that already planned this way: the damper
    stayed closed over lunch, rose to 0.90 once six people were counted for
    the meeting, and CO2 peaked at 935 ppm.
    - A plan makes at most 21 predictions and takes about 0.29 µs when it
      makes all 21 (a full room at 920 ppm), against FR-2's 2 minutes; one
      prediction takes about 12 ns. Measured with Go benchmarks in
      `internal/planner/planner_test.go` on 2026-10-08; output in
      `test/results/bench_planner_2026-10-08.txt`.
    - **Decided: the prediction is the exact solution of the room model,
      checked only now and at the end of the hour** (`Model.Predict` in
      `internal/roommodel`, `staysUnder` in `internal/planner/planner.go`). With `N` and `d` held
      fixed, CO2 moves steadily toward its settling level, so if both ends
      are under the limit, the whole hour is. Replaces forward-Euler steps
      of 10 seconds over the hour: within 0.87 ppm of the exact solution
      (derived), but an approximation needing its own justification, inconsistent with
      the physical model's exact step (see *Decided: the mass balance is
      `V·dC/dt = G·N − Q·(C − C_out)`…*, §6), and 360 steps where one
      calculation does; one plan took about 2.4 µs. Trade-off: the closed
      form holds only while the plan assumes `N` fixed for the hour; a plan
      expecting the head count to change would need to step again. Decided
      and implemented 2026-10-08.
    - Replaces *Deferred, not yet decided: how the decision service turns a
      predicted meeting into a damper level*.
    - Until 2026-09-27 the plan expected the occupancy forecast's head count
      at each step, and the counted people only when more were counted than
      forecast (see *Decided: the occupancy forecast is removed…*, below).
    - **Decided: the level is raised as soon as the current one no longer
      keeps CO2 under the target, but lowered only to a level that keeps it
      20 ppm under the target.** Without this, the chosen level flipped
      back and forth, each flip a new command: 32 in 22 room minutes of the
      meeting of 2026-10-01. Replaying that meeting's stored readings, head
      counts, forecast and room model through the planner reproduced all 32
      commands exactly and showed two causes:
      - Between 0.80 and 0.85, once a minute: the level needed sat right at
        the boundary between the two, and the prediction then stepped in
        whole minutes from the reading, so the meeting time left in it
        dropped by a whole minute at the first reading of each minute and
        then stayed fixed while measured CO2 rose.
      - Between 0.85 and fully open, at the target: a reading at or above
        950 ppm opened the damper fully, and the next reading just under 950
        returned the ordinary level.
      - With the 20 ppm margin, a level that only just keeps CO2 under the
        target no longer replaces the current one. A full room at the
        target opens fully until CO2 is under 930 ppm, then settles at 0.95
        (about 925 ppm) instead of 0.90 (about 948 ppm). Rejected: only
        shortening the prediction's step to 10 seconds, which removes the
        first cause in that meeting but not the second, nor flips from other
        small changes at a boundary. Rejected: a limit on how often a
        command may be sent, which slows the alternation without ending it
        and would hold back a needed rise. Rejected: a gap at the target
        like the fallback switch's, staying fully open until CO2 is under
        900 ppm — fully open, a full room settles at 903 ppm, so the damper
        would stay fully open for the rest of the meeting. Trade-off: the
        fan runs a little higher than strictly needed, and at full for a
        few minutes after CO2 reaches the target. The 20 ppm is an untuned
        default (`PLAN_LOWER_MARGIN_PPM` in `docker-compose.yml`). The
        prediction's step was shortened from 1 minute to 10 seconds in the
        same change.
      - Evidence: replayed on the same stored readings, the planner holds
        0.85 and changes once, to fully open at 950 ppm; a unit test running
        the planner against the room model for an hour, starting a full
        room at the target, changes the level twice (fully open, then
        0.95). Decided and implemented 2026-09-26.
  - **Decided: without a room model the damper switches on CO2 alone; an
    old room model is used however old.**
    - No room model, as before the first fit succeeds, or when the decision
      service restarts after a broker restart (the broker keeps retained
      messages in memory only, while a running decision service keeps the
      model it holds across a reconnect): no level can be
      predicted, so the damper opens fully when a reading reaches 950 ppm
      and closes once CO2 is back below 800 ppm. The gap between the two
      keeps the damper from flipping on every reading. This also moves the
      damper between closed and fully open, which the room-model fit needs
      before it can publish a first model. 800 ppm is an untuned default
      (`FALLBACK_CLOSE_PPM` in `docker-compose.yml`).
    - An old room model: kept, since a room's rates change slowly. When a
      fit fails, the room-model service publishes nothing and the last
      model stays retained.
    - Rejected for the missing model: keeping the damper closed — with 3
      people CO2 settles near 2100 ppm, and the damper never moves, so the
      room model can never be fitted. Rejected: fully open whenever anyone
      is counted — safe, since the fan holds a full room, but it runs the
      whole fan all day and relies on the counter. Trade-off: until the
      first model, CO2 swings between 800 and 950 ppm, with the fan off or
      at full.
    - Decided and implemented 2026-09-26 (`planner.Switch` in
      `internal/planner`, with unit tests; used by `cmd/decision`).
  - Useful for: §1, §4.4, §6, §7.1, §11, §13.

- **Decided: the occupancy forecast is removed; the decision service plans
  from the head count and the room model alone.** The forecast gained
  little. A room brought down to outdoor air holds only about 20 minutes of
  a full meeting's CO2 (in A125, `V·(1000 − 420 ppm)` against `G·N_max`), so
  over a meeting the level is set by what a full room needs, and when the
  empty room already sits at outdoor level, ventilating ahead removes
  nothing. Measured, a day planned with the forecast used more fan than a
  day planned without it (below). Against that, the forecast cost a
  container and a rule in the decision service weighing the head count
  against the forecast. The simulated schedule repeats every weekday with
  little randomness, so the forecast here was close to the best any
  forecast could be, and its gain close to the most a forecast could give in
  this room. The room model, learned from stored data and used in every
  decision, is the data-driven component. Rejected: keeping the forecast
  for display only — a service no decision reads, which still has to be
  tested and defended. Trade-off: the room is no longer ventilated ahead of
  meetings, and the plan learns of each arrival only through the head
  count, up to a minute late. Noted 2026-09-26; decided and implemented
  2026-09-27. The forecast is last present in commit `07954c2`
  (`cmd/occupancy-forecast`, `internal/occupancyforecast`).
  - What it was: the average head count in each 5-minute slot of the day
    over the last 5 weekdays (weekends empty), made once at room midnight
    and published as one retained MQTT message on
    `occupancy/<room>/forecast`. The plan expected the forecast's head count
    at each step of its horizon, and the counted people only when the count
    was above the forecast for now rounded up. Computed with A125's fitted
    rates, a meeting of 6 from 13:00 opened the damper to 0.55 at 12:30 with
    the room at outdoor level, and needed 0.85 at 13:00 against 0.9 for a
    room starting at 900 ppm.
  - **Decided: the comparison runs on the running stack, one weekday with
    the forecast and the next without it.** The `occupancy-forecast`
    container is stopped on the evening of the first day, so the second
    day's plan has no forecast and plans from the head count alone. For
    each day, `eval/day_summary.py` reads the storage-service and gives the
    time-weighted mean damper level over the day, the hour before the
    meeting, and the meeting hour, and the peak CO2. Both days run at 10×,
    the speed of every earlier run. Rejected: a program that steps the
    physical model, the schedule and the planner together, replaying the
    same date with and without the forecast. It pairs the days exactly and
    runs many in seconds, but skips the sensors and the broker, among them
    the head count reaching the decision service up to a minute late, a
    disadvantage of planning without the forecast. Trade-off: the two days'
    arrivals differ by up to ±20 minutes per person, so a very small
    difference can't be told apart from that spread with one day each; a
    second pair of days is added if the result is borderline. Decided
    2026-09-27.
  - **Result: the day with the forecast used more fan, and its one gain
    was 0.05 of damper level while the room was full.** Monday 2026-10-05
    ran with the forecast, Tuesday 2026-10-06 without, both at 10× with
    nearly the same room model:

    | | With forecast | Without |
    |---|---|---|
    | Mean damper level, whole day | 0.160 | 0.132 |
    | Mean damper level, 12:00–13:00 | 0.644 | 0.111 |
    | Mean damper level, 13:00–14:00 | 0.858 | 0.813 |
    | Peak CO2 | 949 ppm | 935 ppm |
    | Person-minutes over the day | 1656 | 1567 |
    | Minutes with the room full | 48 | 34 |

    With the forecast, the damper opened from 12:00 and CO2 fell from 892
    to about 650 ppm before the meeting; the full room was then held at
    0.85. Without it, the damper stayed closed over lunch, CO2 fell to 750
    ppm on the background airflow alone, and the level rose to 0.90 once
    six people were counted: the 0.85 against 0.9 computed beforehand. The
    hour before the meeting accounts for 0.022 of the 0.028 difference over
    the day; Tuesday's fewer people for the rest. Both days stayed under the
    950 ppm target; Tuesday's lower peak follows its shorter meeting. The
    difference is not borderline, so no second pair of days was run.
    Evidence: `python3 eval/day_summary.py 2026-10-05` and `2026-10-06`;
    averaging instead the level in force at each CO2 reading gives the same
    means to within 0.001. Measured 2026-09-27.
  - Useful for: §1, §4.4, §7.1, §10, §11.

- **Decided: the CO2 forecast service is removed.** The CO2 forecast was a least-squares straight
  line through the last 10 minutes of CO2 readings, extended 30 minutes
  ahead and published on `co2/<room>/forecast` (`cmd/forecast` and
  `internal/forecast`, last present in commit `9ba69fc`). It gains nothing in
  this simulation: a forecast pays off only when the system has to act in
  advance, and here a damper change takes effect at the physical model's
  next step and a fully open damper turns a full room's rise around at once,
  so reacting when a reading reaches the threshold overshoots by only about
  10 ppm (see *Decided: the system pursues three goals…*, above). The line
  also used CO2 readings alone, so it saw people only once their CO2 had
  raised the readings, not when the occupancy sensor counted them. Rejected:
  keeping it running beside the occupancy forecast — a service no decision
  reads, which still has to be tested and defended. Rejected: a CO2 forecast
  service that puts the current head count into the mass balance — it would
  see arrivals at once, but a head count already in the room gives nothing
  to act on in advance. The same mass balance, with learned numbers, is how
  the decision service plans instead: there it predicts CO2 for a damper
  level it is considering (see *Decided: the decision service plans with a
  room model learned from stored data*, above). Decided and implemented
  2026-09-25; the occupancy forecast was then the system's only forecast,
  until it was removed too (see *Decided: the occupancy forecast is
  removed…*, above).
  - What it showed while it ran: with the damper closed the line overshot
    the real 30-minute rise in A125 by about 12%; with the damper open it fit
    worse, since the real curve flattens within about 10 minutes, and could
    fall below outdoor air, so it was raised to outdoor CO2 before
    publishing. It took no damper position as input, so it predicted where
    CO2 ends up if nothing changes.
  - Replaces *Decided: start with the simplest forecasting model that
    produces a usable forecast* (proposal section 5, 2026-09-04), of which
    the straight line (2026-09-23) was the first and only model.
  - Useful for: §1, §4.4, §7.1, §11, §13.

- **Decided: CO2 readings, occupancy counts, and commands are persisted in
  SQLite, accessed only through a small storage interface**
  (`store.Store` in `internal/store` — a typed `Save` method per message
  kind, and a read method for each kind that is read back) — no component
  writes SQL directly. Chosen because nothing in the
  actual requirements needs more at this project's current scale (one room),
  and it adds no new infrastructure on top of Go, Docker, and MQTT, all new
  to this project at once. Rejected: InfluxDB — the better fit for
  time-series data and for multi-room scale (a course-named storage option,
  Canvas Introduction page), but that scale isn't planned before the
  deadline, and running it is a real cost (a separate container, where
  SQLite runs inside the storage-service) with no current payoff. The
  interface is what keeps this reversible: swapping to InfluxDB later means one new implementation
  of it, not a rewrite. Decided 2026-09-15, implemented 2026-09-16.
  - **Replaces** an earlier same-day decision to use InfluxDB directly,
    reversed once the storage interface made a later swap cheap enough that
    committing to InfluxDB now wasn't buying anything.
  - **Decided: a dedicated storage-service process owns storage** — it
    subscribes to the CO2 reading, head count and command topics and
    writes each through the storage
    interface, and answers read requests from other components over REST.
    Rejected: folding writing into the CO2 forecast service (since
    removed), which already subscribed to readings — it would mix forecasting logic with
    persistence, and couple their failures (a forecast-service outage would
    also stop storage, instead of the two failing independently, which the
    fault-injection tests need to tell apart). Rejected: a separate
    storage-reader container for reads — writing and reading would fail
    independently, but it adds a container to build, test, and draw. With
    both in one process, an outage stops both, which only matters to a
    service reading stored history at that moment. Writing decided and
    implemented 2026-09-16 (`cmd/storage-service`); reads decided 2026-09-17
    and implemented 2026-09-23 (`GET /co2`, `serveCO2`), and verified
    against the running stack.
    - Timestamps are converted to UTC before they are stored (2026-09-24).
      They are compared as text, so one sent with another offset would
      otherwise sort out of time order.
    - Occupancy counts are stored and served the same way
      (`GET /occupancy`), for the room model's history. Implemented and
      verified against the running stack 2026-09-24.
    - **Decided: commands are served at
      `GET /commands?room=<id>&since=<RFC3339>`, led by the last command
      stored before `since`**, for the room model's history. A damper level
      holds until the next command, which may come days later, so without
      that command the fit would treat the damper as unknown until the first
      command in the window, and skip the readings before it. Rejected: the
      room model asking for extra days before its window — it can't know how
      many are enough. Rejected: the decision service repeating its command
      every minute so every window holds one — it would shape the decision
      service around storage. Decided and implemented 2026-09-25
      (`CommandsInForceSince` in `internal/store`, with unit tests); not yet
      checked against the running stack.
    - **Known caveat —** a room id contains a slash (`level0/A125`), so
      callers must percent-encode it in the query string. Go's `net/http`
      decodes it back, so the service is unaffected, but a hand-written URL
      with a bare slash silently reads a different room.
    - **Replaces** the write-only storage-writer (2026-09-16), renamed when
      it took on reads.
  - **Decided: the storage-service keeps the last 90 room days, measured
    back from the newest stored timestamp, and deletes older rows once every
    real hour, keeping each room's command in force at the cutoff.** A
    service that runs unattended shouldn't be able to fill the disk, however
    long it runs. The window is counted in room time because the stored
    timestamps are room time and the speed changes between sessions.
    Rejected: a real-time window, which would keep a different number of
    room days after every session; downsampling old readings to hourly
    means, which keeps a longer history in less space but needs a second
    table and code path. 90 days caps the file at about 120 MB (about 1.3 MB
    per room day, measured over 16.5 room days) and keeps experiment days
    long enough to be analyzed for the report, since room time advances
    about 60 days per real day at speed 60. SQLite reuses the space of
    deleted rows rather than returning it to the disk, so the file stops
    growing at its largest size. `RETENTION_DAYS` sets the window. Decided
    and implemented 2026-09-28 (`DeleteBefore` in `internal/store`, with unit
    tests; run by `cmd/storage-service`).
    - Replaces *no retention policy*, decided 2026-09-16 on the grounds that
      one room's data over a few weeks stays small. That holds for the
      size, but it left growth unbounded for as long as the stack runs.
    - **Revisit:** if scale changes (see the rejected InfluxDB alternative
      above), InfluxDB has retention built in.
  - **Decided: the CO2 forecast service read recent readings from the
    storage-service over REST** to rebuild its window after a restart,
    retrying three times, 2 s apart, before filling it from the
    `co2_reading` topic instead. Rejected: keeping the window only in memory
    — after a restart there's no forecast until it refills. Rejected:
    request/reply over MQTT — MQTT has no built-in reply, so matching
    replies to requests and timing out would be hand-built, while a REST
    call returns the readings or an immediate error. Rejected: querying
    SQLite directly — it breaks the storage-interface rule. Decided
    2026-09-17, implemented 2026-09-23. The calling side was removed with
    the service 2026-09-25 (see
    *Decided: the CO2 forecast service is removed…*, above); the storage
    side, `GET /co2`, stays.
  - The dashboard reads history through the storage-service's REST
    endpoints (see *Decided: the dashboard is a small service of its own…*,
    §12). The proposal left the data pipeline out entirely; the feedback on
    accepting it (2026-09-15) was "Do not forget the data pipeline and how
    sensor data is transmitted", so the report must cover it explicitly.
  - **Decided: the diagrams and report call the SQLite file the History
    store.** It holds the readings and damper commands over time, which the
    Room-model service and the Dashboard read back as history. Replaces
    "Readings store", which left out the commands. Rejected: "Database",
    which repeats the box's `[SQLite]` tag and says nothing about what is
    stored. Decided 2026-10-07.
  - Useful for: §4.4, §5, §7.2.

- **Deferred, not yet decided: issues and implicit choices found reviewing
  `cmd/storage-service` and `internal/store`, to fix or decide one by one.**
  Noted 2026-09-17.
  - **Fixed 2026-09-17 (bug):** payloads weren't validated against the JSON
    schemas, so a message with missing fields was saved with zero values
    (e.g. 0 ppm). Each message is now checked against its schema before
    saving; an invalid one is logged with the reason and dropped. The
    schemas are built into the program (`schemas/schemas.go`), so the
    `.json` files stay the only definition of a valid message. Rejected:
    hand-written checks in Go — the same rules would live in two places and
    drift apart. Verified with Mosquitto: a valid reading was saved; a
    missing `ppm`, a negative `ppm`, an invalid `ts`, and a `level` of 7
    were each rejected with a clear log line.
  - **Decided: a message whose `room_id` names another room than its topic
    is dropped** — `level0/B200` arriving on `level0/A125/co2/reading` is logged
    and not saved. Before, only `room_id` was used, so a publisher bug would
    have filed readings under the wrong room without any sign. Today the two
    can't disagree, since each publisher builds both from the same level and
    room name settings. Rejected: documenting it as a known limitation —
    simpler, but a wrong-room save would stay silent. Tested by
    `cmd/storage-service/main_test.go` and by publishing a mismatched message
    to the running stack. Decided and implemented 2026-09-28.
    - **Known caveat —** topics hold the room name but not the level, so only
      the name is compared, and two rooms with the same name on different
      floors would share a topic.
  - **Fixed 2026-09-17 (bug):** after a broker restart, the MQTT client
    reconnected but didn't resubscribe (the default clean session makes the
    broker forget subscriptions), so the service kept running and silently
    saved nothing. Now subscribes in the on-connect handler, which runs on
    every reconnect. Verified by restarting a Mosquitto container mid-run: a
    reading published after the restart was stored. Rejected: a persistent
    session instead — Mosquitto keeps sessions in memory by default, so a
    broker restart would still lose the subscriptions.
  - **Decided: the storage-service connects with a persistent session, and
    each table refuses a second row with the same room and timestamp.**
    With a clean session, the broker forgets it on disconnect and every
    reading published while it is down is lost. With
    clean session off, the broker keeps its subscriptions and saves QoS 1
    messages for it until it reconnects under the same client ID. Mosquitto
    saves at most 1000 per absent client by default, about 14 real minutes
    at 10× speed; a longer outage still loses the rest, and a broker restart
    loses the saved messages (they are kept in memory), which is why
    resubscribing on every connect stays. The broker resends any QoS 1
    message it isn't sure was received, so duplicates become more likely:
    `(room_id, ts)` is made unique in each table, and inserts skip a row
    that is already stored. Timestamps are whole seconds, so this assumes
    no source sends twice in one second. Rejected: QoS 2 (exactly once) —
    the MQTT library confirms a message to the broker only after the
    handler has saved it, and keeps its record of received messages in
    memory, so a restart between saving and confirming brings the message
    again under QoS 2 too. Only the store can refuse that copy; with it,
    QoS 2 adds two packets per message and gains nothing. Decided 2026-09-26,
    implemented 2026-09-27 (`SetCleanSession(false)` in
    `cmd/storage-service`; unique indexes and `ON CONFLICT DO NOTHING` in
    `internal/store/sqlite.go`).
    - SQLite can't add a unique index to a table that already holds
      duplicates. Opening the file first deleted them, keeping the earliest
      saved row, until 2026-10-05; the live database held none (105,025 CO2
      readings checked) and has had the indexes since 2026-09-27, so the
      cleanup was removed. Rejected: keeping it, so an older copy of the
      database still opens. Trade-off: an older copy that holds duplicates
      now fails to open, with SQLite's error; no such copy exists.
    - The message handlers are attached to the MQTT client before it
      connects (`AddRoute`), not when subscribing. The broker sends the
      messages it held as soon as the connection opens, before the
      on-connect handler has subscribed; the library leaves a message with
      no handler unsaved and unacknowledged, so it waits for the next
      connection. Found in a 60 s outage that lost the first 5 readings;
      the broker delivered them once the fix was running.
    - Evidence: a unit test for a reading saved twice (stored once). On
      the running stack at 10×, three 60 s stops of the storage-service
      left no hole in the stored history: 248 CO2 readings, each 10 room
      seconds after the last, and head counts every 60.
  - **Fixed 2026-09-27 (bug): a save or read that met another request's
    lock on the SQLite file failed at once with "database is locked".**
    In SQLite's default mode a write must wait for reads in progress and
    reads for a write, and the default wait allowed is zero. Over about 8
    real hours of running the log held 86 such errors, 29 of them saves (27 CO2
    readings, 2 head counts) that were lost; the others were reads, which
    showed on the dashboard as a failed refresh. The file is now opened in
    WAL mode, where new rows go to a side file first so reads and a write
    don't block each other, with a 5 s wait on any lock left
    (`OpenSQLite` in `internal/store/sqlite.go`). Rejected: the wait alone —
    collisions would become short waits instead of errors, enough at
    today's load, but reads would still hold up saves as readers are
    added. Rejected: one database connection shared by every request, so
    the program queues them itself — no lock ever meets another, but each
    save waits behind whole reads, including the room model's 7-day fetch.
    Trade-off: WAL keeps two extra files beside the database, and works
    only while every program using the file runs on the same machine.
    Evidence: a unit test saving readings while four goroutines read in a
    loop failed on every run before the change (4 read errors per run) and
    passes after it; a second test checks the file is in WAL mode. On the
    running stack at 10×, 5 minutes of `GET /latest` and a 6-hour
    `GET /co2` every real second gave no failed request, lock error or
    failed save.
    - Indexes on `(room_id, ts)` would shorten each read, but not stop the
      failures. Each table has one since 2026-09-27, as the uniqueness rule
      above; before that every query scanned its whole table, and
      `GET /latest` took about 17 ms at two weeks of history.
  - **Known caveat — a bad payload or a failed save is logged and dropped,
    never retried.** Accepted: a bad payload fails the same way on every
    try, and the failed saves seen so far were lock errors, fixed at their
    cause by WAL mode (above). Rejected: retrying — it would add a queue
    and a retry limit for a failure that no longer occurs. Decided
    2026-09-27.
  - **Revisit:** only the sender's timestamp is stored (as text, whole
    seconds), not the receive time — pipeline latency can't be measured from
    stored data.
  - Fixed 2026-09-17: the SQLite file was lost whenever the container was
    recreated. `docker-compose.yml` now stores it in a Docker volume (storage
    kept outside the container), so readings survive a restart or rebuild.
  - **Decided: the pure-Go SQLite library (`modernc.org/sqlite`).** The
    service is built with no C compiler (`CGO_ENABLED=0`) into an image
    holding only the program. Rejected: `mattn/go-sqlite3`, which wraps
    SQLite's C code and would need a C compiler in the build and C libraries
    in the image. Trade-off: the pure-Go library is slower, by an amount not
    yet measured; the stress test shows whether it matters. Decided
    2026-09-27.
  - **Decided for convention: settings come from environment variables**
    (broker URL, database path, retention), set in `docker-compose.yml` and
    required (see *Decided: every setting is required…*, §6). No
    alternative to environment variables was weighed. Decided 2026-09-27.
  - **Decided: if the broker is unreachable at startup, the process exits
    and Docker starts it again** (`restart: on-failure`). Rejected: retrying
    inside the program — it would repeat what the restart policy already
    does, and the same policy covers a crash later on. Decided 2026-09-27.
  - **Known caveat — no graceful shutdown: on container stop the database
    isn't closed and the client doesn't disconnect.** Accepted: each save is
    committed as it happens, and the library confirms a message to the
    broker only after its handler has returned, so a message cut off
    mid-save is sent again after the restart and the duplicate check skips
    it if it was saved after all. Decided 2026-09-27.
  - Useful for: §5, §7.2, §9, §10.

- **Decided: the decision service connects to the broker with a clean
  session, so after a restart it starts from the next reading instead of
  the ones it missed.** It decides on each reading as it arrives, so a
  backlog handed over after a longer outage would make it decide, and
  possibly command the damper, on CO2 from minutes before. With a reading
  every 10 s, the next fresh one is never far off. Rejected: a persistent
  session, as the storage-service has (see *Decided: the storage-service
  connects with a persistent session…*, above), which needs every reading,
  while the decision service needs only the latest. Trade-off: up to one
  reading interval of extra delay after a restart. The crash test measured
  10.8 s twice, the decision service having subscribed just after the
  restarted sensor's first reading, within NFR-1's 2 minutes (see *Decided: the crash test kills the CO2 sensor and
  the decision service together…*, §10). The clean session was the MQTT
  library's default until the crash test showed its cost. Decided
  2026-10-06; no code change.
  - Useful for: §4.4, §7.1, §8.2, §11.

## 8. Behaviour

- **One CO2 reading traced through the full loop: a damper command it
  triggered reached the physical model within one step.** Traced on the
  running stack for A125 on 2026-10-01, room time (UTC), from the stored
  readings and commands (`GET /co2`, `GET /commands`) and each container's
  log, with the topics and the command's HTTP method as they were then:
  1. `physical-model`, with 2 people and the damper at 0.80, steps CO2 to
     555.6 ppm and writes it to BuildSim as the sensor's value.
  2. `co2-sensor` reads it from BuildSim and publishes it on
     `co2/A125/reading`, stamped 10:45:09; `storage-service` stores it.
  3. `decision` receives it; the plan now needs 0.85 instead of 0.80, so it
     sends `POST /command` with 0.85 to `actuator`, stamped 10:45:09.
  4. `actuator` writes 0.85 to BuildSim as the damper's position and
     answers that it was accepted; `decision` then publishes the copy on
     `ventilation/A125/command`, and `storage-service` stores it.
  5. `physical-model`'s next step reads 0.85 from BuildSim.
  6. The next reading, stamped 10:45:19, is 556.0 ppm: the rise per
     reading slowed from 0.5 to 0.4 ppm.
  - Every hop logged in the same real second (at speed 10, one step of 10
    room seconds); the logs are stamped to the whole second, so the time
    per hop is not measured. Noted 2026-09-26.
  - Useful for: §8.1, §10, §11.

## 9. Deployment and component view

- **Decided: one `docker-compose.yml` starts the whole system, BuildSim
  included.** BuildSim is built from the course repository at a pinned
  commit (its build context in `docker-compose.yml`), so no local copy is
  needed and its behavior can't change
  mid-project. Rejected: running BuildSim by hand outside Docker — one more
  manual step before every run and demo. Pinning the commit is an unchecked
  default. Decided and implemented 2026-09-17; every service has been
  added to it since.
  - Useful for: §4.2, §9.

## 10. Test plan

- **Decided: each requirement has one piece of evidence, of the kind that
  fits what the requirement is about.** The loop's answer to one reading:
  the end-to-end test (FR-2, FR-3). What the system does over a whole day:
  the stored data of a room day (FR-1, REG-1, NFR-2, NFR-4). One piece of
  logic on its own: unit tests (FR-4 to FR-6, FR-8 to FR-10). Behavior when something
  breaks: fault tests (NFR-1, NFR-3). What the dashboard shows: a
  screenshot (FR-7). The report's requirements table names each
  requirement's evidence, and the test-plan table lists the requirements
  each check covers. Rejected: splitting one requirement across two
  checks, as FR-1 was between the end-to-end test and a stored day, which
  leaves two places to keep in step for one claim. Decided 2026-10-06.
  - Useful for: §3, §10.

- **Decided: tests run at speed 1; the day-long evaluation runs at any
  faster speed.** A test checks how fast something happens (the end-to-end
  test, the crash test, the fault tests, the delay measurement), and its
  limits are in room time, so room time and real time must run on one
  clock. The evaluation runs (REG-1's peak CO2, NFR-2's damper comparison)
  only read the CO2 and damper levels stamped in room time, which the
  physical model computes exactly at any speed (§6); a faster speed only
  stretches millisecond processing delays in room time, and a room day
  takes 2.4 real hours at speed 10 and 24 minutes at speed 60. Each test
  refuses to run at any speed but 1. Rejected: speed 1 for everything,
  which costs about three of the days left before submission on three room
  days. Decided 2026-10-06.
  - Useful for: §10, §11.

- **Decided: the end-to-end test writes 980 ppm into BuildSim and checks the
  loop's answer: the reading stored, a fully open command stored within 2
  minutes of it, the damper open in BuildSim, and CO2 falling.**
  `test/e2e/loop_test.go` verifies FR-2 and FR-3 (§3) against the running
  stack, run with `go test -tags e2e -v -count=1 ./test/e2e`, and refuses
  to run unless the session runs at speed 1. The physical model continues
  from the CO2 that BuildSim holds, so the injected value runs the whole
  loop like a room that has just filled. 980 ppm is above the 950 ppm
  target, which opens the damper fully with or without a room model, and
  below the threshold. Rejected: waiting for the occupancy schedule to change the
  damper — it only works in occupied room hours, and a normal level change
  moves CO2 by under 1 ppm per reading, too little to check. Trade-offs: the
  test covers the full opening, not the plan choosing a level in between
  (covered by the unit tests in `internal/planner`); and the injected jump
  enters the stored history the next room-model fit uses, an effect not
  measured. Decided and implemented 2026-10-05.
  - FR-2's 2 minutes are checked from the stored timestamps of the reading
    and the command. Two limits: the decision service stamps the command
    just before the actuator writes it to BuildSim, so the write itself is
    not timed; and the history store keeps whole seconds, so the gap is
    known to 1 s. Added 2026-10-06.
  - Result, 2026-10-07, at speed 1 with the room at 900 ppm, one person
    and the damper closed: passed in 31 real seconds. CO2 went to 980 ppm
    at the injection; a 1.00 command was stored in the same room second as
    that reading; CO2 then fell about 4.6 ppm per reading, 975 → 966 ppm
    over three readings. It replaces a pass on 2026-10-06 with the same
    outcome, rerun after the topics and room settings were renamed.
    Evidence in `test/results/`: the test's output (`e2e_2026-10-07.txt`)
    and the stored CO2 readings, commands and head counts of the run's
    window (`e2e_2026-10-07_*.json`).
  - Useful for: §10, §11.

- **All 64 unit tests and subtests passed on 2026-10-08**, among them
  those that verify FR-4 (`internal/store`), FR-5 (`internal/roommodel`),
  FR-6 (`internal/planner`), FR-8 (`cmd/storage-service`,
  `internal/store`), FR-9 (`internal/occupancy`) and FR-10
  (`internal/co2`), §3. Evidence: `test/results/unit_2026-10-08.txt`, the
  output of `go test -v -count=1 ./...`. It replaces a pass on 2026-10-07,
  rerun after `internal/co2`'s tests took the fully open airflow from
  `MAX_AIRFLOW_FACTOR` 1.2 instead of the earlier 2.
  - Useful for: §10, §11.

- **Decided: the crash test kills the CO2 sensor and the decision service
  together, writes 980 ppm into BuildSim while both are down, and measures
  the real time until BuildSim holds the damper fully open.**
  `test/crash/crash.sh` verifies NFR-1 (§3) against the running stack, and
  refuses to run unless the session runs at speed 1. A restart takes real
  time whatever the speed, while the 2-minute limit comes from how fast a
  real room fills; only at speed 1 do both run on the same clock, including
  the sensor's 10 s reading interval. The test fails unless Docker restarted
  each process exactly once, and prints the first stored reading after the
  kill to show the injection took effect. The decision service is the
  one whose absence the 2-minute limit is about; the CO2 sensor is a device
  process, so the damper can only open once it has registered with BuildSim
  again and published a reading. One run thus covers both halves of NFR-1.
  Each process is killed from the host with `sudo kill -9`, since Docker
  doesn't restart a container it was told to stop (see *Decided: every
  service restarts by itself after a crash…*, §6). Rejected: killing the CO2
  sensor alone — its outage is easy to see, but it says nothing about the decision
  service; killing the decision service alone — it doesn't register with
  BuildSim. The other containers aren't killed: every one restarts through
  the same `restart: on-failure`, and every device process registers through
  the same `buildsim.Client.Register`. Decided and implemented 2026-10-06.
  - Result, 2026-10-07, at speed 1: two runs, each passed in 10.8 real
    seconds, with Docker restarting each container once per run. In both,
    the restarted sensor published 980 ppm within a second of the kill,
    13–17 ms before the restarted decision service subscribed, so the
    decision service missed that reading and opened the damper on the next
    one, 10 s later. Recovery is thus the restart time plus up to one
    reading interval: the decision service connects without a persistent
    session, so the broker keeps no readings for it while it is down; had
    it subscribed first, it would have acted on the first reading. Replaces
    two runs on 2026-10-06, rerun after the topics and room settings were
    renamed. Evidence in `test/results/`: each run's output
    (`crash_2026-10-07_run1.txt`, `crash_2026-10-07.txt`), and its stored
    CO2 readings, commands, head counts and the two services' logs
    (`crash_2026-10-07_run1_*`, `crash_2026-10-07_*`).
  - Useful for: §8.2, §10, §11.

- **REG-1 held on Thursday 2026-10-22: the peak CO2 was 937 ppm, at 13:48
  during the meeting.** The day ran at speed 60 on the current code, the
  first full weekday of that session; Wednesday 2026-10-21 was left out
  because its peak, 980 ppm at 04:13, was written into BuildSim by a test.
  The mean damper level was 0.132 over the day and 0.772 in the meeting
  hour. Evidence in `test/results/`: `day_model_2026-10-22.txt` (the
  output of `eval/day_summary.py`) and the day's stored CO2 readings,
  commands and head counts (`day_model_2026-10-22_*.json`). Measured
  2026-10-06.
  - The same day verifies FR-1: 8640 CO2 readings are stored, one for
    every 10 s, and 1440 head counts, one for every room minute
    (`day_model_2026-10-22_co2.json` and `_occupancy.json`).
    Rejected: the end-to-end test, which shows one reading stored where
    the day shows all of them, and would wait up to a minute at speed 1
    for the occupancy sensor's next count. Decided 2026-10-06.
  - Useful for: §10, §11.

- **Decided: failure testing covers a sensor giving bad readings, a component
  going down, and delayed or dropped communication.** These are the failure
  modes named in proposal sections 6 and 7. The crash test covers a
  component going down (see *Decided: the crash test kills the CO2 sensor
  and the decision service together…*, above); a frozen CO2 sensor and a
  stopped broker are not yet tested, and delay only as a measurement
  (NFR-4). Proposal, 2026-09-04.
  - Useful for: §10, §11, §13.

- **Idea, not yet evaluated: change `Q_max` in the physical model mid-run — a
  fan losing capacity — and measure how long the room model stays wrong.**
  The decision side never sees `Q_max`, `V` or `G`; the room model learns
  how fast the room clears from stored history rather than from the
  parameters (see *Decided: the decision service plans with a room model
  learned from stored data…*, §7). Changing `Q_max` invalidates what it
  learned, and the recovery time is a measure of how long the pipeline takes
  to catch up with a building that has changed. Noted 2026-09-21.
  - Useful for: §10, §11, §13.

## 11. Evaluation and results

- **Decided: decision quality is scored by the time-weighted mean damper
  level over a room day, the learned-model plan against the CO2-only
  switch.** Each level counts for as long as it was in force.
  `eval/day_summary.py` already computes it, and the forecast comparison
  used it (see *Decided: the comparison runs on the running stack…*, §7).
  No other metric was weighed; chosen because the tool and a first result
  already exist. Decided 2026-10-06.
  - Useful for: §3 (NFR-2), §10, §11.

- **Decided: the CO2-only day runs on the normal stack, with a setting that
  makes the decision service ignore room models.** With
  `IGNORE_ROOM_MODEL: "true"` in `docker-compose.yml`, the decision service
  never subscribes to the room-model topic, so it has no model and runs the
  CO2-only switch (FR-6), the same code a new system runs before its first
  fit. Thursday 2026-10-22 (the REG-1 day) is the room-model day, and
  Tuesday 2026-10-27 the switch day, both at speed 60. Rejected: stopping
  the room-model service, deleting its retained model from the broker and
  restarting the decision service by hand, which needs no code but is
  three manual steps to get right and undo. Trade-off: as in the forecast
  comparison (§7), arrivals differ by up to ±20 minutes per person between
  the days, so a second pair of days is added if the result is borderline.
  Decided and implemented 2026-10-06.
  - **Result: NFR-2 holds; the room model used 0.132 of damper level over
    the day against the switch's 0.152.**

    | | Room model, Thu 2026-10-22 | CO2-only switch, Tue 2026-10-27 |
    |---|---|---|
    | Mean damper level, whole day | 0.132 | 0.152 |
    | Mean damper level, 12:00–13:00 | 0.107 | 0.143 |
    | Mean damper level, 13:00–14:00 (meeting) | 0.772 | 1.000 |
    | Peak CO2 | 937 ppm at 13:48 | 951 ppm at 09:28 |
    | Person-minutes over the day | 1558 | 1642 |
    | Minutes with the room full (6) | 35 | 48 |
    | Distinct damper levels commanded | 11 | 2 (0 and 1) |

    The meeting hour shows the difference most directly: with six people
    in both rooms, the switch held the damper fully open while the room
    model held it at about 0.77. Tuesday had 5% more person-minutes, which
    raises its damper use; scaling Tuesday's mean down by that share gives
    0.144 (derived, assumes damper use grows in step with people), still
    above Thursday's, so the result was judged not borderline and no
    second pair was run. The switch also crossed the 950 ppm target, as it
    only opens once a reading reaches it; both days stayed under 1000 ppm.
    Evidence in `test/results/`: `day_model_2026-10-22*` and `day_switch_2026-10-27*`
    (the day summaries and each day's stored readings, commands and head
    counts). Measured 2026-10-06.
  - Useful for: §7.1, §10, §11.

## 12. Dashboard

- **Decided: the dashboard is a small service of its own, serving one web
  page that reads the room's history from the storage-service's REST
  endpoints.** The page asks the storage-service for the newest stored room
  time and draws the history before it (`cmd/dashboard/static/index.html`),
  since room time runs ahead of the wall clock. The service passes the
  page's `/api/...` requests on to the storage-service, since a browser
  refuses to read from another address
  than the page's own unless that address allows it. Rejected: Grafana
  reading the same endpoints through a plugin. Grafana draws every chart
  over a window of wall-clock time and hides data outside it, while stored
  timestamps are room time, days ahead of the wall clock and, sped up,
  drawing further ahead every second; a fixed shift of the window can't
  follow that. Rejected: any tool reading the SQLite file directly — it
  bypasses the storage interface (see *Decided: CO2 readings, occupancy
  counts, and commands are persisted in…*, §7). Rejected: a page fed live
  from MQTT — it shows only what arrived while it was open and needs the
  broker to accept WebSocket connections; reading storage also shows the
  pipeline serving monitoring. Trade-off: the page is code to write and
  test, and it polls rather than being pushed each change. Charts use uPlot,
  kept in the repository so the page needs no internet; chosen as a small
  library with time axes built in, a low-stakes default. Decided 2026-09-27.
  - Useful for: §4.4, §7.2, §12.

## 13. Risks and critical reflection

_(nothing yet)_

## Where AI advice was wrong

Cases where advice from the AI coding assistant turned out wrong, and how it
was caught. Kept for the report's reflection and the oral exam.

- **The 1000 ppm CO2 threshold was attributed to the wrong source.** The
  assistant suggested it came from the workplace rules already cited for the
  ceiling height (AFS 2023:12). Reading that document showed it has no CO2
  limit, only a minimum outdoor airflow. The figure actually comes from
  FoHMFS 2014:18 (see the threshold in *Decided: the system pursues three
  goals…*, §7). Caught because the value was checked against the source
  before it was logged. Noted 2026-09-18.
  - Useful for: §13.

- **Every course rule was treated as needing its own row in the
  requirements table.** The assistant proposed adding rows for separate
  processes and for planning with the room model, so that each pass
  requirement of the course appeared in §3. The course's grade-5 example
  has eight rows, all about what its use case must achieve, and shows the
  course rules in their own sections (container diagram, decision logic,
  dashboard). Caught when the table kept growing and was compared with the
  example. Correct: a row states what the system must do or achieve, with a
  criterion its test checks; course rules are shown where the report
  describes them. Noted 2026-10-06.
  - Useful for: §13.

- **The assistant proposed changing the simulation so the forecast would
  have something to do.** Faced with the finding that reacting at the
  threshold already works (see *Decided: the system pursues three goals…*,
  §7), it first suggested a delay between a damper command and its effect,
  then a fan too weak for a full room, with the forecast cleaning the room
  ahead of meetings. The user rejected the first as introducing a problem
  only to solve it, and the second because a room that fills unexpectedly
  would then go over the threshold. Its first analysis had also silently
  assumed the damper is either closed or fully open, which the user caught
  by asking. Noted 2026-09-24.
  - Useful for: §13.

- **The assistant claimed faster room time would shrink the stored
  occupancy history.** It reasoned that the counter's 10-second interval is
  real time, so a faster room would hold fewer counts per room day. That
  holds for the current code, but only because running faster isn't built
  yet: if the sensors kept a real-time interval, speed would change how
  often the system samples, which the design keeps independent of speed
  (see *Decided: room time runs at a speed chosen per session…*, §6). The user
  caught it by asking whether speed changes the system's behavior. The
  history holds about 240,000 counts per 20 weekdays at any speed. Noted
  2026-09-24.
  - Useful for: §13.

- **The assistant argued against its own earlier refactor using an obstacle
  that does not exist.** Asked to reassess merging the BuildSim client's
  `Equipment`, `Sensor`, and `Actuator` structs into one `Device`, it claimed
  the merge was unworkable because `type` means different things on the
  equipment record (picks the 3D view's icon) and on the sensor record
  (decides CO2 room shading), so one field could not serve both. Reading
  BuildSim's viewer source showed the shading matches the substring `co2` or
  `carbon` against `sensor.type || sensor.name || sensor.id`
  (`web/modern/src/live.js`), so a single type string of `co2_sensor`
  satisfies the icon and the shading at once. Caught by pushing back on the
  reversal and checking the source. The only real cost of the merge is that
  `Unit` is meaningless on actuators. Noted 2026-09-22.
  - Useful for: §13.

- **The assistant said `docker compose down` would remove the container of a
  service just deleted from `docker-compose.yml`.** Compose acts only on the
  services named in the current file, so the old container kept running,
  kept its network in use, and was reported as an "orphan" on the next
  start. The user caught it from the warnings in the terminal output.
  Removing it takes `docker compose down --remove-orphans`, or removing the
  container by name. Noted 2026-09-25.
  - Useful for: §13.

- **The assistant said `./start.sh` would resume room time where the stored
  history stopped, leaving no gap, after the machine was shut down hard.**
  The broker had stayed down while the other services came back and kept
  running on the old session's clock. `start.sh` first starts the
  storage-service, which also starts the broker; the CO2 sensor, still on
  the old clock, then got a reading stored before `start.sh` asked for the
  latest one, so the session resumed at the old clock's time. The history
  has no CO2 readings from 2026-09-29 13:41 to 2026-09-30 23:12 UTC. Caught
  by comparing the room start in `run.env` with the time the history should
  have stopped, then listing the gaps in the stored readings. Noted
  2026-09-26.
  - Useful for: §13.

- **The assistant reported the same CO2 level, 848.5 ppm, at 13:00 on both
  days of the forecast comparison.** Its one-off check fetched every reading
  from 12:55 onwards and looked them up by clock time, so the second day's
  readings overwrote the first day's. Caught by the assistant itself, since an
  identical level after very different damper levels in the hour before was
  implausible; a trace of each day showed 724 ppm with the forecast and 845
  ppm without. Noted 2026-09-27.
  - Useful for: §13.

- **The assistant ran `docker compose up -d --build decision` to rebuild
  only the decision service.** `--build` also rebuilds the services a
  service depends on, here the actuator and, through it, BuildSim, whose
  image is built from the course repository. Both were recreated, and
  BuildSim lost the room's state and devices. `start.sh` already warned
  against rebuilding for this reason. Caught from the command's output
  listing BuildSim as started. Rebuilding one service alone takes
  `docker compose build decision`, then `docker compose up -d --no-deps
  decision`. Noted 2026-09-27.
  - Useful for: §13.

- **The assistant said a persistent session alone would keep the messages
  published while the storage-service was down.** The broker did keep them,
  but the service attached its message handlers only when subscribing after
  connecting, and the first held messages arrived before that, so they
  weren't saved. Caught by stopping the service for 60 s on the running
  stack and listing the holes in the stored readings. The handlers are now
  attached before connecting (see *Decided: the storage-service connects
  with a persistent session…*, §7). Noted 2026-09-27.
  - Useful for: §13.

- **The assistant concluded from one reboot that no container started until
  `./start.sh` ran.** It checked only the containers' start times, which
  `./start.sh` resets when it replaces them. The storage-service's own log
  showed it trying to reach the broker from 7 s after boot. Caught at the
  next boot, when `./start.sh` gave up waiting for the storage-service (see
  *Decided: every service restarts by itself after a crash but not when the
  PC boots*, §6). Noted 2026-09-28.
  - Useful for: §13.

- **The assistant said a crash test could use `docker kill`, since the
  process then exits with code 137 and `restart: on-failure` restarts it.**
  Docker treats `docker kill` as a manual stop and never restarts after it.
  Caught by killing the dashboard: it exited with 137 and stayed off, and
  Docker's log said "restart canceled". Noted 2026-09-28.
  - Useful for: §13.

- **The assistant said `docker compose build` followed by `./start.sh` would
  leave BuildSim running with the room's state, since its image hadn't
  changed.** The image indeed kept the same ID, but `./start.sh` recreated
  the BuildSim container anyway, so the room's CO2 restarted from outdoor
  air. Why it was recreated is not known. Caught from `./start.sh`'s output
  listing BuildSim as started. Noted 2026-09-28.
  - Useful for: §13.

- **The assistant gave the end-to-end test's run command without
  `-count=1`.** Without it, `go test` may report a remembered earlier pass,
  marked `(cached)`, instead of running the test, so a repeat run against
  the stack could show a pass that never happened. Caught by the assistant
  while explaining the command's flags; the flag is now in the command in
  `test/e2e/loop_test.go`. Noted 2026-10-05.
  - Useful for: §10, §13.

- **The assistant planned the crash test to measure recovery in room time at
  speed 10, against NFR-1's 2 room minutes.** Restarting a container takes
  real time whatever the speed, so at speed 10 each real second counted as
  10 room seconds and the result depended on the speed. Caught by the user
  before the test was run. Correct: run the test at speed 1, where room time
  is real time, and measure real seconds (see *Decided: the crash test kills
  the CO2 sensor and the decision service together…*, §10). Noted
  2026-10-06.
  - Useful for: §10, §13.

- **The assistant advised against describing the room model as not needed
  for basic operation, saying it would undercut the course's requirement
  for a data-driven component.** The requirement asks that the system has
  and uses one, which it does: in normal operation every damper level comes
  from the room model's plan. That the system keeps running on the CO2-only
  switch when the component fails is graceful degradation, a separate
  property and a strength. Caught by the user. Noted 2026-10-07.
  - Useful for: §4.3, §13.

- **The assistant held the design-decisions table to a stricter rule than
  the course: that a decision may cite only a requirement the rejected
  alternative would not meet.** It stripped requirement links from four
  rows and proposed new requirements to restore them. Caught by the user,
  who questioned whether requirements were being made up to fit decisions;
  the course's example report links its one-container-per-role row to a
  crash-recovery requirement a restarted single program could also meet.
  Correct: a row cites the requirement its decision helps the system meet.
  Noted 2026-10-07.
  - Useful for: §4.4.

- **The assistant framed the MQTT topic as a choice between the room's name
  alone and BuildSim's full room key in one topic level.** The full key
  (`level0/A125`) contains a slash, so in one level it would break the `+`
  wildcard; the room's name alone was chosen, leaving the room with two
  names, one in the topic and one in the payload. Caught by the user, who
  asked why the floor wasn't simply a topic level of its own. Correct: the
  floor and the room as two levels, `level0/A125/co2/reading` (see
  *Decided: every MQTT topic starts with the room as BuildSim names it…*,
  §5). Noted 2026-10-07.
  - Useful for: §5, §13.

- **The CO2 per person was attributed to a table column that does not
  exist.** The notes said Table 2 of Persily & de Jonge gives ≈ 0.0052 L/s
  at 1.5 met, the paper's level for office work. The table has columns for
  1.4 and 1.6 met only, so the figure was the assistant's own average of
  the two. Caught when the value was checked against the paper itself
  before citing it in the report. Correct: `G` is computed from the paper,
  and marked as such (see *Decided: the mass balance is…*, §6). Noted
  2026-10-08.
  - Useful for: §6, §13.

- **The notes described a different lunch break from the one the code
  runs.** They gave lunch as 11:30–13:00 with 0–1 people in the room; the
  occupancy schedule has always emptied the room 11:30–12:30. Caught when
  the code was read to describe the schedule in the report. Correct: the
  schedule in `internal/occupancy/schedule.go`.
  Noted 2026-10-08.
  - Useful for: §6.

- **The notes claimed that seeding the schedule's random shifts from the
  date makes a demo or test repeatable.** The system never replays a date:
  every session resumes from the latest stored room time, so the seed only
  fixes each date's schedule. Caught by the user while reviewing report §6,
  which also raised how the demo could show the 13:00 meeting at all; see
  *Decided: a session can start at a chosen room time…*, §6. Noted
  2026-10-08.
  - Useful for: §6, §13.

- **The notes and report said the 13:00 meeting is the case where CO2
  climbs fast enough to cross 1000 ppm.** With the damper closed, any two
  or more people cross it eventually: three settle near 2030 ppm. What sets
  the meeting apart is the damper it needs, about 0.9 against about 0.4 for
  three people, as the stored commands of 2026-10-22 show. Caught when the
  report was checked against the code and stored data. Noted 2026-10-08.
  - Useful for: §2, §13.
