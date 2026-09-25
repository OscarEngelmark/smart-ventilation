// storage-service subscribes to the co2 reading, occupancy reading, and
// ventilation command MQTT topics, saves each one through
// internal/store, and serves stored readings and occupancy counts back to
// other components over REST.
// Reasoning: project_notes.md §7.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/store"
	"github.com/OscarEngelmark/smart-ventilation/schemas"
)

// payloadSchemas holds the schema each incoming message is checked against
// before it is saved.
type payloadSchemas struct {
	reading   *jsonschema.Schema
	occupancy *jsonschema.Schema
	command   *jsonschema.Schema
}

type co2ReadingPayload struct {
	RoomID string    `json:"room_id"`
	PPM    float64   `json:"ppm"`
	Ts     time.Time `json:"ts"`
}

type occupancyReadingPayload struct {
	RoomID string    `json:"room_id"`
	Count  int       `json:"count"`
	Ts     time.Time `json:"ts"`
}

type ventilationCommandPayload struct {
	RoomID string    `json:"room_id"`
	Level  float64   `json:"level"`
	Ts     time.Time `json:"ts"`
}

func main() {
	brokerURL := env.String("MQTT_BROKER_URL", "tcp://localhost:1883")
	dbPath := env.String("SQLITE_PATH", "storage-service.db")
	addr := env.String("STORAGE_ADDR", ":8081")

	var db store.Store
	var err error
	db, err = store.OpenSQLite(dbPath)
	if err != nil {
		log.Fatalf("open storage: %v", err)
	}
	defer db.Close()

	sch := payloadSchemas{
		reading:   mustLoadSchema("co2_reading.schema.json"),
		occupancy: mustLoadSchema("occupancy_reading.schema.json"),
		command:   mustLoadSchema("ventilation_command.schema.json"),
	}

	// Subscribing here rather than once after Connect: the client reconnects
	// automatically, but with a clean session the broker forgets
	// subscriptions on disconnect, so they must be renewed on every connect.
	onConnect := func(client mqtt.Client) {
		log.Printf("connected to broker, subscribing")
		subscribeAll(client, db, sch)
	}
	onConnectionLost := func(_ mqtt.Client, err error) {
		log.Printf("connection to broker lost: %v", err)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("storage-service").
		SetOnConnectHandler(onConnect).
		SetConnectionLostHandler(onConnectionLost)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect to broker: %v", token.Error())
	}
	defer client.Disconnect(250)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /co2", serveCO2(db))
	mux.HandleFunc("GET /occupancy", serveOccupancy(db))
	mux.HandleFunc("GET /latest", serveLatest(db))
	log.Printf("serving stored CO2 readings and occupancy on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// serveCO2 answers GET /co2?room=<id>&since=<RFC3339> with that room's stored
// CO2 readings from that time onwards, oldest first, as a JSON array of
// co2_reading messages. Both parameters are required.
func serveCO2(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID, since, err := roomAndSince(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "co2: %v", err)
			return
		}
		stored, err := db.CO2ReadingsSince(r.Context(), roomID, since)
		if err != nil {
			fail(w, http.StatusInternalServerError, "co2: read %s: %v", roomID, err)
			return
		}

		body := make([]co2ReadingPayload, 0, len(stored)) // empty rather than nil, so no readings encodes as [] and not null
		for _, s := range stored {
			body = append(body, co2ReadingPayload{RoomID: s.RoomID, PPM: s.PPM, Ts: s.Time})
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			log.Printf("co2: write response: %v", err)
		}
	}
}

// serveOccupancy answers GET /occupancy?room=<id>&since=<RFC3339> with that
// room's stored occupancy counts from that time onwards, oldest first, as a
// JSON array of occupancy_reading messages. Both parameters are required.
func serveOccupancy(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID, since, err := roomAndSince(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "occupancy: %v", err)
			return
		}
		stored, err := db.OccupancySince(r.Context(), roomID, since)
		if err != nil {
			fail(w, http.StatusInternalServerError, "occupancy: read %s: %v", roomID, err)
			return
		}

		body := make([]occupancyReadingPayload, 0, len(stored)) // encodes as [] when empty
		for _, s := range stored {
			body = append(body, occupancyReadingPayload{RoomID: s.RoomID, Count: s.Count, Ts: s.Time})
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			log.Printf("occupancy: write response: %v", err)
		}
	}
}

// serveLatest answers GET /latest with the newest timestamp stored in any
// table, as plain RFC 3339 text for start.sh to read, or 204 No Content when
// nothing is stored yet.
func serveLatest(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latest, ok, err := db.Latest(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, "latest: %v", err)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, latest.UTC().Format(time.RFC3339))
	}
}

// roomAndSince reads the two query parameters both read endpoints require.
func roomAndSince(r *http.Request) (string, time.Time, error) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		return "", time.Time{}, fmt.Errorf("no room given")
	}
	sinceText := r.URL.Query().Get("since")
	since, err := time.Parse(time.RFC3339, sinceText)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("bad since %q: %v", sinceText, err)
	}
	return roomID, since, nil
}

// fail logs why a request was refused and answers with that reason as plain
// text, so a caller sees the same message the log holds.
func fail(w http.ResponseWriter, status int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Print(msg)
	http.Error(w, msg, status)
}

// subscribeAll subscribes to every topic this service stores. Each message is
// checked against its schema first, so an invalid one is logged and dropped
// instead of being saved with zero values for missing fields.
func subscribeAll(client mqtt.Client, db store.Store, sch payloadSchemas) {
	subscribe(client, "co2/+/reading", func(payload []byte) {
		if err := schemas.Validate(sch.reading, payload); err != nil {
			log.Printf("reading: invalid payload: %v", err)
			return
		}
		var p co2ReadingPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			log.Printf("reading: bad payload: %v", err)
			return
		}
		if err := db.SaveCO2Reading(context.Background(), store.CO2Reading{
			RoomID: p.RoomID, PPM: p.PPM, Time: p.Ts,
		}); err != nil {
			log.Printf("reading: save failed: %v", err)
		}
	})

	subscribe(client, "occupancy/+/reading", func(payload []byte) {
		if err := schemas.Validate(sch.occupancy, payload); err != nil {
			log.Printf("occupancy: invalid payload: %v", err)
			return
		}
		var p occupancyReadingPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			log.Printf("occupancy: bad payload: %v", err)
			return
		}
		if err := db.SaveOccupancy(context.Background(), store.Occupancy{
			RoomID: p.RoomID, Count: p.Count, Time: p.Ts,
		}); err != nil {
			log.Printf("occupancy: save failed: %v", err)
		}
	})

	subscribe(client, "ventilation/+/command", func(payload []byte) {
		if err := schemas.Validate(sch.command, payload); err != nil {
			log.Printf("command: invalid payload: %v", err)
			return
		}
		var p ventilationCommandPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			log.Printf("command: bad payload: %v", err)
			return
		}
		if err := db.SaveCommand(context.Background(), store.Command{
			RoomID: p.RoomID, Level: p.Level, Time: p.Ts,
		}); err != nil {
			log.Printf("command: save failed: %v", err)
		}
	})
}

func subscribe(client mqtt.Client, topic string, handle func(payload []byte)) {
	token := client.Subscribe(topic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		handle(msg.Payload())
	})
	token.Wait()
	if err := token.Error(); err != nil {
		log.Fatalf("subscribe %s: %v", topic, err)
	}
}

// mustLoadSchema exits the process if a schema can't be loaded, so the
// service never runs without checking messages.
func mustLoadSchema(name string) *jsonschema.Schema {
	sch, err := schemas.Load(name)
	if err != nil {
		log.Fatalf("load schema: %v", err)
	}
	return sch
}
