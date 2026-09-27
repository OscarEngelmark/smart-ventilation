# Working plan

What's left to build, in order. Delete a row when it's done; the decisions
themselves go in `project_notes.md`. Target: code and experiments done by
2026-10-02, report written 2026-10-02 to 2026-10-12.

| # | Part | Open decisions | Steps |
|---|---|---|---|
| 10 | Storage-service review list (`project_notes.md` §7): the room in the topic isn't compared to the room in the message | check it, or document it as a known limitation | 1 |
| 11 | Tests: one end-to-end test; fault injection (crashed service, frozen sensor, broker down); stress test to a breaking point | small ones per test; whether to store each message's receive time, so pipeline delay can be measured from stored data | 6–8 |
