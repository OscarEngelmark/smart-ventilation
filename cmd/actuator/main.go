// actuator is the ventilation actuator process for one room. It registers its
// damper with BuildSim on startup, then serves the decision service's
// ventilation commands over REST, writing each commanded level to the damper.
// Reasoning: project_notes.md §4 and §5.
package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/schemas"
)

// commands is what serving one command needs: the damper it is written to,
// and the schema it is checked against.
type commands struct {
	client   *buildsim.Client
	schema   *jsonschema.Schema
	damperID string
	roomID   string
}

// ventilationCommand is the ventilation_command message, defined by
// schemas/ventilation_command.schema.json.
type ventilationCommand struct {
	RoomID string    `json:"room_id"`
	Level  float64   `json:"level"`
	Ts     time.Time `json:"ts"`
}

// commandResponse is the ventilation_command_response message, defined by
// schemas/ventilation_command_response.schema.json.
type commandResponse struct {
	Accepted bool      `json:"accepted"`
	Ts       time.Time `json:"ts"`
}

func main() {
	baseURL := env.String("BUILDSIM_URL", buildsim.DefaultURL)
	level := env.String("ROOM_LEVEL", "level0")
	roomName := env.String("ROOM_NAME", "A125")
	damperID := env.String("DAMPER_ID", roomName+"-damper")
	addr := env.String("ACTUATOR_ADDR", ":8080")

	client := buildsim.New(baseURL) // this program's link to BuildSim
	ctx := context.Background()     // empty context, no cancellation or timeout

	if err := client.Register(ctx, equipment(damperID, level, roomName)); err != nil {
		log.Fatalf("register %s: %v", damperID, err)
	}

	schema, err := schemas.Load("ventilation_command.schema.json")
	if err != nil {
		log.Fatalf("load schema: %v", err)
	}

	c := &commands{
		client:   client,
		schema:   schema,
		damperID: damperID,
		roomID:   buildsim.RoomKey(level, roomName),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /command", c.serve)
	log.Printf("serving commands for %s on %s, writing %s", c.roomID, addr, damperID)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// serve takes one ventilation command and writes its level to the damper. It
// answers only after BuildSim has stored the new position, so a caller that
// gets no acceptance can send the command again.
func (c *commands) serve(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		reject(w, http.StatusBadRequest, "read command: %v", err)
		return
	}
	if err := schemas.Validate(c.schema, payload); err != nil {
		reject(w, http.StatusBadRequest, "invalid command: %v", err)
		return
	}
	var cmd ventilationCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		reject(w, http.StatusBadRequest, "bad command: %v", err)
		return
	}
	if cmd.RoomID != c.roomID {
		reject(w, http.StatusNotFound, "command for %s, this actuator serves %s",
			cmd.RoomID, c.roomID)
		return
	}
	// The command's level is the damper position unchanged: both run 0 to 1.
	if err := c.client.SetActuatorState(r.Context(), c.damperID, cmd.Level); err != nil {
		reject(w, http.StatusBadGateway, "write %s: %v", c.damperID, err)
		return
	}
	log.Printf("%s: damper %.2f", c.roomID, cmd.Level)
	respond(w, http.StatusOK, true)
}

// reject logs why a command was refused and answers that it was not accepted.
func reject(w http.ResponseWriter, status int, format string, args ...any) {
	log.Printf(format, args...)
	respond(w, status, false)
}

// respond answers with one ventilation_command_response.
func respond(w http.ResponseWriter, status int, accepted bool) {
	body := commandResponse{
		Accepted: accepted,
		Ts:       time.Now().UTC(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

// equipment is the device record this process registers: one equipment record
// in the room, holding the one damper.
func equipment(damperID, level, roomName string) buildsim.Equipment {
	device := buildsim.Actuator{
		ID:   damperID,
		Name: "Damper",
		Type: "damper",
	}
	return buildsim.Equipment{
		ID:        damperID,
		Name:      "Ventilation damper",
		Type:      "damper",
		Category:  "actuator",
		Level:     level,
		Room:      roomName,
		Actuators: []buildsim.Actuator{device},
	}
}
