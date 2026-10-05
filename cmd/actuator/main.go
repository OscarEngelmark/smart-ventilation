// actuator is the ventilation actuator process for one room. It registers its
// damper with BuildSim on startup, then serves the decision service's
// ventilation commands over REST, writing each commanded level to the damper.
// Reasoning: project_notes.md §4 and §5.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/message"
	"github.com/OscarEngelmark/smart-ventilation/internal/shutdown"
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

func main() {
	shutdown.ExitZeroOnStop()

	baseURL := env.String("BUILDSIM_URL")
	level := env.String("ROOM_LEVEL")
	roomName := env.String("ROOM_NAME")
	damperID := env.String("DAMPER_ID")
	addr := ":8080" // the port this program listens on inside its container

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
// answers 200 OK only after BuildSim has stored the new position, so a caller
// that gets any other status can send the command again.
func (c *commands) serve(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		c.reject(w, http.StatusBadRequest, "read command: %v", err)
		return
	}
	if err := schemas.Validate(c.schema, payload); err != nil {
		c.reject(w, http.StatusBadRequest, "invalid command: %v", err)
		return
	}
	var cmd message.VentilationCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		c.reject(w, http.StatusBadRequest, "bad command: %v", err)
		return
	}
	if cmd.RoomID != c.roomID {
		c.reject(w, http.StatusNotFound, "command for %s, this actuator serves %s",
			cmd.RoomID, c.roomID)
		return
	}
	// The command's level is the damper position unchanged: both run 0 to 1.
	if err := c.client.SetActuatorState(r.Context(), c.damperID, cmd.Level); err != nil {
		c.reject(w, http.StatusBadGateway, "write %s: %v", c.damperID, err)
		return
	}
	log.Printf("%s: damper %.2f", c.roomID, cmd.Level)
	w.WriteHeader(http.StatusOK)
}

// reject logs why a command was refused and answers with that reason as plain
// text, so the caller sees the same message the log holds.
func (c *commands) reject(w http.ResponseWriter, status int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Print(msg)
	http.Error(w, msg, status)
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
