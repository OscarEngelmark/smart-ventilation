# Working plan

What's left to build, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Target: code and experiments done by
2026-10-02, report written 2026-10-02 to 2026-10-12.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 1 | Forecast service (Python): reads `GET /readings`, publishes `co2_forecast` | which simple model; the 30-min horizon | 3 |
| 2 | Decision service (Go): forecast vs. 1000 ppm, commands the actuator | behavior when the forecast is missing or stale; one instance per room or one overall | 3 |
| 3 | First full loop running end to end | none | 1 |
| 4 | Room time faster than real time? | one decision | 1 |
| 5 | Dashboard | how it reads from storage | 2 |
| 6 | Storage-service review list (`project_notes.md` §7) | fix or document each as a known limitation | 1–2 |
| 7 | Tests: one end-to-end test; fault injection (crashed service, frozen sensor, broker down); forecast vs. plain threshold; stress test to a breaking point | small ones per test | 6–8 |
