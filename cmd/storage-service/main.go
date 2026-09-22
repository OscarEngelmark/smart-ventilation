// storage-service subscribes to the co2 reading, co2 forecast, and
// ventilation command MQTT topics, and saves each one through
// internal/store. Serving reads over REST is not implemented yet.
package main

import (
	"context"
	"encoding/json"
	"log"
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
	reading  *jsonschema.Schema
	forecast *jsonschema.Schema
	command  *jsonschema.Schema
}

type co2ReadingPayload struct {
	RoomID string    `json:"room_id"`
	PPM    float64   `json:"ppm"`
	Ts     time.Time `json:"ts"`
}

type co2ForecastPayload struct {
	RoomID      string    `json:"room_id"`
	PPMForecast float64   `json:"ppm_forecast"`
	HorizonMin  float64   `json:"horizon_min"`
	Ts          time.Time `json:"ts"`
}

type ventilationCommandPayload struct {
	RoomID string    `json:"room_id"`
	Level  float64   `json:"level"`
	Ts     time.Time `json:"ts"`
}

func main() {
	brokerURL := env.String("MQTT_BROKER_URL", "tcp://localhost:1883")
	dbPath := env.String("SQLITE_PATH", "storage-service.db")

	var db store.Store
	var err error
	db, err = store.OpenSQLite(dbPath)
	if err != nil {
		log.Fatalf("open storage: %v", err)
	}
	defer db.Close()

	sch := payloadSchemas{
		reading:  mustLoadSchema("co2_reading.schema.json"),
		forecast: mustLoadSchema("co2_forecast.schema.json"),
		command:  mustLoadSchema("ventilation_command.schema.json"),
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

	select {}
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
		if err := db.SaveReading(context.Background(), store.Reading{
			RoomID: p.RoomID, PPM: p.PPM, Time: p.Ts,
		}); err != nil {
			log.Printf("reading: save failed: %v", err)
		}
	})

	subscribe(client, "co2/+/forecast", func(payload []byte) {
		if err := schemas.Validate(sch.forecast, payload); err != nil {
			log.Printf("forecast: invalid payload: %v", err)
			return
		}
		var p co2ForecastPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			log.Printf("forecast: bad payload: %v", err)
			return
		}
		if err := db.SaveForecast(context.Background(), store.Forecast{
			RoomID: p.RoomID, PPMForecast: p.PPMForecast, HorizonMin: p.HorizonMin, Time: p.Ts,
		}); err != nil {
			log.Printf("forecast: save failed: %v", err)
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
