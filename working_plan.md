# Working plan

What's left to build, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Target: code and experiments done by
2026-10-02, report written 2026-10-02 to 2026-10-12.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 5 | Occupancy forecast: average head count per time of day over the last 20 weekdays (`project_notes.md` §7) | how far ahead it looks; what happens to the straight-line CO2 forecast | 2 |
| 6 | Decision service (Go): ventilates ahead of predicted occupancy, keeps CO2 under 1000 ppm, commands the actuator (`project_notes.md` §7, *the system pursues three goals*) | how a predicted meeting becomes a damper level; behavior when the forecast is missing or stale; one instance per room or one overall | 3 |
| 7 | First full loop running end to end | none | 1 |
| 8 | Dashboard | how it reads from storage | 2 |
| 9 | Storage-service review list (`project_notes.md` §7) | fix or document each as a known limitation | 1–2 |
| 10 | Tests: one end-to-end test; fault injection (crashed service, frozen sensor, broker down); forecast-led ventilation vs. reacting alone; stress test to a breaking point | small ones per test | 6–8 |
