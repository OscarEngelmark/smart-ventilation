# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Course context

D7065E — Embedded Intelligence at the Edge (LTU, HT2026). The deliverable is a
distributed, containerized cyber-physical system built around BuildSim as the
shared building-state backend.

- Course reference repo (read-only — do not edit): `~/projects/D7065E`. Contains
  course notes (`course-notes/`), hands-on tutorials (`tutorials/`), the
  proposal/final-report LaTeX+D2 toolchain and worked examples
  (`lab-assignment/`), and the BuildSim source itself (`buildingsim/`).
- Working solo. The course default is pairs of 2.

## Use case

Indoor air quality (CO2) management. Closed loop: occupancy drives CO2 buildup
(first-order dynamics, roughly +40 ppm/person/hour, decaying toward outdoor
baseline at a rate set by ventilation) → CO2 sensor reads the room → a
short-horizon forecast model predicts CO2 30 minutes ahead → a decision
service compares the forecast to a threshold and commands the ventilation
actuator → BuildSim updates room/actuator state → this feeds back into the
next simulated CO2 reading.

## Constraints

- Every component (sensors, actuators, the autonomous decision service) runs
  as its own independently deployable and restartable process (Docker
  container) — never collapse the system into a monolith.
- The CO2 forecast model is this project's data-driven component; keep it in
  the decision loop rather than quietly falling back to a plain threshold.

## Toolchain

Language choice is split deliberately by component role, not a default —
don't add a component in Python "for convenience" or in Go "for consistency"
without checking whether it fits this split first:

- **Go** for the networked service layer — sensor processes, actuator
  processes, and the decision-service shell/orchestration. Chosen for
  goroutine-based concurrency and small, fast-starting static binaries, which
  matter because the whole system has to run on one laptop at once (see
  `tutorials/docker-containers.md`, "laptop budget"). The user is new to
  Go — favor idiomatic, simple code (`gofmt`/`go vet` clean) over clever
  code, and lean on C/Python analogies in explanations.
- **Python** for the CO2 forecasting model specifically, packaged as its own
  small service that the decision service calls over REST like everything
  else. Chosen for the forecasting/ML ecosystem (e.g. statsmodels, Prophet,
  scikit-learn), where Go has no mature equivalent.
- **Docker / Docker Compose** for every service regardless of language,
  except BuildSim's own binary.
- **BuildSim** (`~/projects/D7065E/buildingsim`) is the shared building-state
  backend: REST + WebSocket API, in-memory state, no auth, loopback-only by
  default. Treat it as a fixed external dependency — consume it via
  `pkg/client` or raw REST/WS, never modify it.
- **D2** for architecture diagrams (C4 model), and **LaTeX** (pdflatex) for
  the proposal/final report, following the Makefile pattern already set up in
  `~/projects/D7065E/lab-assignment/`.

## Status

Pre-implementation. Use case is selected; the architecture/design document has
not been written yet. Don't invent a service layout or directory structure
ahead of that — it gets added once the C4 container diagram is settled at the
week-3 design checkpoint.
