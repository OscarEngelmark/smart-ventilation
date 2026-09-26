# Containers

Every container in the system, what it is responsible for, what it takes in,
what it puts out, and what it needs running. Update a row whenever a
container's inputs or outputs change; the reasoning behind each choice is in
`project_notes.md`.

The dividing rule between the learning services and the decision service:
a learning service answers a question about the room that doesn't depend on
what the damper does (when people come, how the room responds). Anything that
depends on the damper choice, including predicting CO2, belongs to the
decision service.

`<room>` is the room's name, e.g. `A125`. Every interval is in room time.

| Container | Role | Inputs | Outputs | Depends on | Status |
|---|---|---|---|---|---|
| `buildsim` | Holds the shared building state in memory: rooms, devices, who is where, sensor values, the damper position | REST calls from the other processes | REST answers; a 3D view on port 9090 | none | Running; the course's own program, pinned to one commit, never modified |
| `mosquitto` | MQTT broker: passes each published message to its subscribers, and keeps the last retained message on a topic for later subscribers | Published messages | Messages to subscribers | none | Running |
| `occupancy` | The simulated people: moves each person in or out of the room by their weekday schedule | Room time; the room's area, from BuildSim | Every 10 s, the building's occupancy, written to BuildSim | `buildsim` | Running |
| `physical-model` | The simulated room: steps the CO2 mass balance. The only container that knows the room's volume, CO2 per person and airflows | Every 10 s, from BuildSim: the head count, the damper position, the current CO2 level | The next CO2 level, written to BuildSim as the CO2 sensor's value | `buildsim` | Running |
| `co2-sensor` | Registers the room's CO2 sensor with BuildSim and reads it | Every 10 s, the sensor's value from BuildSim | `co2/<room>/reading` | `buildsim`, `mosquitto` | Running |
| `occupancy-sensor` | Registers the room's people counter with BuildSim and counts the people in the room | Every minute, the occupancy from BuildSim | The count, written to BuildSim as the counter's value; `occupancy/<room>/reading` | `buildsim`, `mosquitto` | Running |
| `actuator` | Registers the room's damper with BuildSim and applies commands to it | `POST /command` (port 8080), a `ventilation_command` | The level, written to BuildSim as the damper position; a `ventilation_command_response` once BuildSim has stored it | `buildsim` | Running |
| `storage-service` | Stores every reading and command, and serves their history | `co2/+/reading`, `occupancy/+/reading`, `ventilation/+/command` | SQLite file on the `storage-data` volume; `GET /co2`, `GET /occupancy`, `GET /commands` (led by the last command before the window), `GET /latest` (port 8081) | `mosquitto` | Running |
| `occupancy-forecast` | Learns when people come to the room | At room midnight, the last 7 days of head counts, from `GET /occupancy` | `occupancy/<room>/forecast` (retained): the expected head count for each 5-minute slot of the day | `storage-service`, `mosquitto` | Running |
| `room-model` | Learns how the room's CO2 responds to people and to the damper: the three rates of the room model | At room midnight, the last 7 days of CO2 readings, head counts and damper commands, from `GET /co2`, `GET /occupancy` and `GET /commands` | `room/<room>/model` (retained): the three rates; nothing when the history can't separate them | `storage-service`, `mosquitto` | Running |
| `decision` | The only container that chooses: sets the damper level that keeps CO2 under 1000 ppm with the fan as low as possible | `co2/<room>/reading`, `occupancy/<room>/reading`, `occupancy/<room>/forecast`, `room/<room>/model` | On every CO2 reading, a level; when it changes, `POST /command` to the actuator and, once accepted, a copy on `ventilation/<room>/command` | `mosquitto`, `actuator` | Running; one per room; plans the level from the forecast and the room model, or from the head count when it is above the forecast or there is no forecast; without a room model, opens and closes the damper on CO2 alone |
| dashboard | Shows the room's readings, damper level and decisions | Not yet decided | A web page | `storage-service` | Planned (working plan row 9) |
