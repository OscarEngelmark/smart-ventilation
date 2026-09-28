# Working plan

What's left to build, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Target: code and experiments done by
2026-10-02, report written 2026-10-02 to 2026-10-12.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 10 | Readability refactor: split multi-job functions in `decision`, `room-model`, `physical-model`; then find and merge duplicated code across services | how to share duplicated code (e.g. `connect`, `equipment`) between services | 3–4 |
| 11 | Tests: one end-to-end test; fault injection (crashed service, frozen sensor, broker down); stress test to a breaking point | small ones per test; whether to store each message's receive time, so pipeline delay can be measured from stored data | 6–8 |
