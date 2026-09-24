# Working plan

What's left to build, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Target: code and experiments done by
2026-10-02, report written 2026-10-02 to 2026-10-12.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 2 | Decision service (Go): keeps CO2 under 1000 ppm, ventilates ahead of predicted occupancy, commands the actuator (`project_notes.md` §7, *the system pursues three goals*) | how occupancy is sensed and forecast; how a predicted meeting becomes a damper level; the fan's safety factor; behavior when the forecast is missing or stale; one instance per room or one overall | to re-plan |
| 3 | First full loop running end to end | none | 1 |
| 4 | Room time faster than real time? | one decision | 1 |
| 5 | Dashboard | how it reads from storage | 2 |
| 6 | Storage-service review list (`project_notes.md` §7) | fix or document each as a known limitation | 1–2 |
| 7 | Tests: one end-to-end test; fault injection (crashed service, frozen sensor, broker down); forecast vs. plain threshold; stress test to a breaking point | small ones per test | 6–8 |
