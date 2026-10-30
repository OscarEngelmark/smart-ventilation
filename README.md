# Smart Ventilation

A containerized system that keeps the CO2 level of a simulated office room
under 1000 ppm while running the ventilation as low as possible. It runs as
a closed loop: people in the room raise its CO2, a sensor reads it, a
decision service sets the ventilation damper, and the room's next CO2
reading shows the effect. The damper level comes from a room model learned
from the stored readings: on every reading, the decision service picks the
lowest level that the model predicts will keep the people counted now under
the target.

The shared building state (rooms, people, sensor values, the damper
position) is held by [BuildSim](https://github.com/eislab-cps/D7065E/tree/main/buildingsim).

## Containers

Every part runs as its own Docker container, started together by
[docker-compose.yml](docker-compose.yml). The names in parentheses are the
Compose service names.

| Container | What it does |
|---|---|
| BuildSim (`buildsim`) | Holds the shared building state in memory |
| Message broker (`mosquitto`) | MQTT broker that passes readings and commands between services |
| Occupancy simulator (`occupancy`) | Moves people in and out of the room by their weekday schedule |
| Physical model (`physical-model`) | Steps the room's CO2 mass balance from the head count and the damper position |
| CO2 sensor (`co2-sensor`) | Reads the room's CO2 from BuildSim and publishes it |
| Occupancy sensor (`occupancy-sensor`) | Counts the people in the room and publishes the count |
| Damper actuator (`actuator`) | Applies the decision service's commands to the damper |
| Storage service (`storage-service`) | Stores every reading and command in SQLite and serves their history |
| Room-model service (`room-model`) | Fits the room model to the last week of history, once per simulated day |
| Decision service (`decision`) | Chooses the damper level |
| Dashboard (`dashboard`) | Web page showing CO2, head count and damper level over time |

[containers.md](containers.md) lists each container's inputs, outputs and
dependencies, and [diagrams/](diagrams/) holds the C4 context and container
diagrams, the deployment diagram and the behavior diagrams.

## Running it

You need Docker with Compose. Start a session with:

```sh
./start.sh 10
```

The argument is the speed: how many times faster than real time the
simulated room's clock runs. Each session continues the room's clock where
the stored history ends. The first run builds every image, BuildSim
included.

Once it runs, open:

- http://127.0.0.1:8082 for the dashboard
- http://127.0.0.1:9090 for BuildSim's 3D view of the building

Stop it with `docker compose down`. The stored history is kept in the
`storage-data` volume and survives a restart.

## Tests

```sh
go test ./...
```

runs the unit tests. The end-to-end test needs the stack running at speed 1
(`./start.sh 1`):

```sh
go test -tags e2e -v -count=1 ./test/e2e
```

The fault-injection, crash-and-restart and stress tests are shell scripts
in [test/fault/](test/fault/), [test/crash/](test/crash/) and
[test/stress/](test/stress/); each one's header says how to run it. Their
saved outputs, with the data stored during each run, are in
[test/results/](test/results/).

## Report and notes

The final report's source is in [latex/report/](latex/report/). Run `make`
there to build `build/report.pdf`; it needs TeX Live and
[D2](https://d2lang.com) for the diagrams.
[project_notes.md](project_notes.md) records the reasoning behind each
design decision, with the alternatives that were rejected.

Built as a course project for D7065E at Luleå University of Technology,
2026. BuildSim comes from the course's public repository,
[eislab-cps/D7065E](https://github.com/eislab-cps/D7065E).
