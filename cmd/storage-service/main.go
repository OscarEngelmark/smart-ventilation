// storage-service subscribes to the co2 reading, occupancy reading, and
// ventilation command MQTT topics, saves each one through
// internal/store, and serves stored readings, occupancy counts and commands
// back to other components over REST.
// Reasoning: project_notes.md §7.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/message"
	"github.com/OscarEngelmark/smart-ventilation/internal/shutdown"
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

func main() {
	shutdown.ExitZeroOnStop()

	brokerURL := env.String("MQTT_BROKER_URL", "tcp://localhost:1883")
	dbPath := env.String("SQLITE_PATH", "storage-service.db")
	addr := env.String("STORAGE_ADDR", ":8081")
	retentionDays := int(env.Float("RETENTION_DAYS", 90))

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

	handlers := storeHandlers(db, sch) // topic -> the function that saves a message from it
	client := connectBroker(brokerURL, handlers)
	defer client.Disconnect(250)

	go deleteOldEveryHour(db, retentionDays) // runs alongside the server below
	serve(db, addr)
}

// deleteOldEveryHour runs deleteOld at startup and then once every real hour.
func deleteOldEveryHour(db store.Store, days int) {
	ticker := time.NewTicker(time.Hour)
	for {
		if err := deleteOld(context.Background(), db, days); err != nil {
			log.Printf("retention: %v", err)
		}
		<-ticker.C // wait for the next tick
	}
}

// deleteOld deletes what is stored from more than days room days before the
// newest stored timestamp, keeping each room's command in force at the cutoff.
func deleteOld(ctx context.Context, db store.Store, days int) error {
	latest, ok, err := db.Latest(ctx)
	if err != nil || !ok {
		return err
	}
	cutoff := latest.AddDate(0, 0, -days)
	deleted, err := db.DeleteBefore(ctx, cutoff)
	if err != nil {
		return err
	}
	if deleted > 0 {
		log.Printf("retention: deleted %d rows from before %s", deleted, cutoff.Format(time.RFC3339))
	}
	return nil
}

// connectBroker opens the link to the broker, with each topic's messages
// routed to its handler. The client reconnects on its own.
func connectBroker(brokerURL string, handlers map[string]func(topic string, payload []byte)) mqtt.Client {
	// Subscribing here rather than once after Connect: the client reconnects
	// automatically, but a broker restart forgets every session's
	// subscriptions, so they must be renewed on every connect.
	onConnect := func(client mqtt.Client) {
		log.Printf("connected to broker, subscribing")
		for topic := range handlers {
			subscribe(client, topic)
		}
	}
	onConnectionLost := func(_ mqtt.Client, err error) {
		log.Printf("connection to broker lost: %v", err)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("storage-service").
		SetCleanSession(false). // the broker keeps QoS 1 messages for this client ID while it is away
		SetOnConnectHandler(onConnect).
		SetConnectionLostHandler(onConnectionLost)
	client := mqtt.NewClient(opts)
	addRoutes(client, handlers)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect to broker: %v", token.Error())
	}
	return client
}

// addRoutes attaches each topic's handler to the client. It must run before
// connecting, because the broker sends the messages it held as soon as the
// connection opens, before onConnect runs.
func addRoutes(client mqtt.Client, handlers map[string]func(topic string, payload []byte)) {
	for topic, handle := range handlers {
		client.AddRoute(topic, func(_ mqtt.Client, msg mqtt.Message) {
			handle(msg.Topic(), msg.Payload())
		})
	}
}

// serve answers the read endpoints on addr until the process stops.
func serve(db store.Store, addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /co2", serveCO2(db))
	mux.HandleFunc("GET /occupancy", serveOccupancy(db))
	mux.HandleFunc("GET /commands", serveCommands(db))
	mux.HandleFunc("GET /latest", serveLatest(db))
	log.Printf("serving stored CO2 readings, occupancy and commands on %s", addr)
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

		body := make([]message.CO2Reading, 0, len(stored)) // empty rather than nil, so no readings encodes as [] and not null
		for _, s := range stored {
			body = append(body, message.CO2Reading{RoomID: s.RoomID, PPM: s.PPM, Ts: s.Time})
		}
		writeJSON(w, "co2", body)
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

		body := make([]message.OccupancyReading, 0, len(stored)) // encodes as [] when empty
		for _, s := range stored {
			body = append(body, message.OccupancyReading{RoomID: s.RoomID, Count: s.Count, Ts: s.Time})
		}
		writeJSON(w, "occupancy", body)
	}
}

// serveCommands answers GET /commands?room=<id>&since=<RFC3339> with that
// room's stored commands from that time onwards, led by the last one before
// it, oldest first, as a JSON array of ventilation_command messages. Both
// parameters are required.
func serveCommands(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID, since, err := roomAndSince(r)
		if err != nil {
			fail(w, http.StatusBadRequest, "commands: %v", err)
			return
		}
		stored, err := db.CommandsInForceSince(r.Context(), roomID, since)
		if err != nil {
			fail(w, http.StatusInternalServerError, "commands: read %s: %v", roomID, err)
			return
		}

		body := make([]message.VentilationCommand, 0, len(stored)) // encodes as [] when empty
		for _, s := range stored {
			body = append(body, message.VentilationCommand{RoomID: s.RoomID, Level: s.Level, Ts: s.Time})
		}
		writeJSON(w, "commands", body)
	}
}

// writeJSON answers with body encoded as JSON, logging under kind if the
// answer can't be written.
func writeJSON(w http.ResponseWriter, kind string, body any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("%s: write response: %v", kind, err)
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

// storeHandlers gives, for every topic this service stores, the function that
// saves one message from it, given the topic it arrived on. Each message is
// checked against its schema first, so an invalid one is logged and dropped
// instead of being saved with zero values for missing fields. A message whose
// room_id names another room than its topic is dropped too.
func storeHandlers(db store.Store, sch payloadSchemas) map[string]func(topic string, payload []byte) {
	handlers := make(map[string]func(topic string, payload []byte))
	handlers["co2/+/reading"] = func(topic string, payload []byte) {
		saveCO2Reading(db, sch.reading, topic, payload)
	}
	handlers["occupancy/+/reading"] = func(topic string, payload []byte) {
		saveOccupancy(db, sch.occupancy, topic, payload)
	}
	handlers["ventilation/+/command"] = func(topic string, payload []byte) {
		saveCommand(db, sch.command, topic, payload)
	}
	return handlers
}

// saveCO2Reading saves one message from co2/+/reading, or logs why it was
// dropped.
func saveCO2Reading(db store.Store, sch *jsonschema.Schema, topic string, payload []byte) {
	var p message.CO2Reading
	if !accept("reading", sch, topic, payload, &p, &p.RoomID) {
		return
	}
	r := store.CO2Reading{RoomID: p.RoomID, PPM: p.PPM, Time: p.Ts}
	if err := db.SaveCO2Reading(context.Background(), r); err != nil {
		log.Printf("reading: save failed: %v", err)
	}
}

// saveOccupancy saves one message from occupancy/+/reading, or logs why it
// was dropped.
func saveOccupancy(db store.Store, sch *jsonschema.Schema, topic string, payload []byte) {
	var p message.OccupancyReading
	if !accept("occupancy", sch, topic, payload, &p, &p.RoomID) {
		return
	}
	o := store.Occupancy{RoomID: p.RoomID, Count: p.Count, Time: p.Ts}
	if err := db.SaveOccupancy(context.Background(), o); err != nil {
		log.Printf("occupancy: save failed: %v", err)
	}
}

// saveCommand saves one message from ventilation/+/command, or logs why it
// was dropped.
func saveCommand(db store.Store, sch *jsonschema.Schema, topic string, payload []byte) {
	var p message.VentilationCommand
	if !accept("command", sch, topic, payload, &p, &p.RoomID) {
		return
	}
	c := store.Command{RoomID: p.RoomID, Level: p.Level, Time: p.Ts}
	if err := db.SaveCommand(context.Background(), c); err != nil {
		log.Printf("command: save failed: %v", err)
	}
}

// accept checks payload against sch, decodes it into out, and checks that the
// room it names, read from roomID once decoded, is the one in topic. It logs
// under kind and returns false for a message that fails any check.
func accept(kind string, sch *jsonschema.Schema, topic string, payload []byte, out any, roomID *string) bool {
	if err := schemas.Validate(sch, payload); err != nil {
		log.Printf("%s: invalid payload: %v", kind, err)
		return false
	}
	if err := json.Unmarshal(payload, out); err != nil {
		log.Printf("%s: bad payload: %v", kind, err)
		return false
	}
	if !roomMatchesTopic(*roomID, topic) {
		log.Printf("%s: room %q doesn't match topic %s", kind, *roomID, topic)
		return false
	}
	return true
}

// subscribe asks the broker for topic's messages at QoS 1; they are delivered
// to the handler already attached with AddRoute.
func subscribe(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil) // nil: no handler here, the route handles it
	token.Wait()
	if err := token.Error(); err != nil {
		log.Fatalf("subscribe %s: %v", topic, err)
	}
}

// roomMatchesTopic reports whether roomID ("level0/A125") names the same room
// as topic ("co2/A125/reading"). The topic holds no level, so only the room
// name is compared.
func roomMatchesTopic(roomID, topic string) bool {
	parts := strings.Split(topic, "/") // e.g. ["co2", "A125", "reading"]
	if len(parts) != 3 {
		return false
	}
	return path.Base(roomID) == parts[1] // path.Base gives the part after the last "/"
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
