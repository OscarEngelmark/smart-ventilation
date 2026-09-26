# Working plan

What's left to build, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Target: code and experiments done by
2026-10-02, report written 2026-10-02 to 2026-10-12.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 7 | Decision service (Go): plans the damper level from the occupancy forecast and the room model, keeps CO2 under 1000 ppm, commands the actuator (`project_notes.md` §7, *the system pursues three goals*) | how a predicted meeting becomes a damper level; behavior when the forecast or room model is missing or stale, including before enough history exists | 2 |
| 8 | First full loop running end to end | none | 1 |
| 9 | Dashboard | how it reads from storage | 2 |
| 10 | Storage-service review list (`project_notes.md` §7) | fix or document each as a known limitation | 1–2 |
| 11 | Tests: one end-to-end test; fault injection (crashed service, frozen sensor, broker down); forecast-led ventilation vs. reacting alone; stress test to a breaking point | small ones per test | 6–8 |
