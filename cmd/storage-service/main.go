// storage-service subscribes to the co2 reading, co2 forecast, and
// ventilation command MQTT topics, and saves each one through
// internal/store. Serving reads over REST is not implemented yet.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/OscarEngelmark/smart-ventilation/internal/store"
)

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
	brokerURL := getEnv("MQTT_BROKER_URL", "tcp://localhost:1883")
	dbPath := getEnv("SQLITE_PATH", "storage-service.db")

	var db store.Store
	var err error
	db, err = store.OpenSQLite(dbPath)
	if err != nil {
		log.Fatalf("open storage: %v", err)
	}
	defer db.Close()

	opts := mqtt.NewClientOptions().AddBroker(brokerURL).SetClientID("storage-service")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect to broker: %v", token.Error())
	}
	defer client.Disconnect(250)

	subscribe(client, "co2/+/reading", func(payload []byte) {
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

	select {}
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
