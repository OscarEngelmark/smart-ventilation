// occupancy-forecast predicts how many people one room will hold through the
// day. At startup and at each room midnight it fetches the room's recent head
// counts from the storage-service, forecasts every slot of the day with
// internal/occupancyforecast, and publishes the whole day as one retained
// message. Reasoning: project_notes.md §7.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	// The image carries no timezone database, so TZ is otherwise ignored.
	_ "time/tzdata"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/occupancyforecast"
	"github.com/OscarEngelmark/smart-ventilation/internal/roomtime"
)

// retryWait is the real-time pause between tries at the storage-service.
const retryWait = 5 * time.Second

// forecaster is what one day's forecast needs: where the history comes from,
// how much of it to use, and where the forecast is published.
type forecaster struct {
	storageURL string
	http       *http.Client
	broker     mqtt.Client
	roomID     string
	topic      string
	days       int
}

// occupancyReading is the occupancy_reading message the storage-service
// returns, defined by schemas/occupancy_reading.schema.json.
type occupancyReading struct {
	RoomID string    `json:"room_id"`
	Count  int       `json:"count"`
	Ts     time.Time `json:"ts"`
}

// occupancyForecast is the occupancy_forecast message, defined by
// schemas/occupancy_forecast.schema.json.
type occupancyForecast struct {
	RoomID      string         `json:"room_id"`
	Date        string         `json:"date"`
	SlotMinutes int            `json:"slot_minutes"`
	Slots       []slotForecast `json:"slots"`
}

type slotForecast struct {
	Start  time.Time `json:"start"`
	People float64   `json:"people"`
}

func main() {
	storageURL := env.String("STORAGE_URL", "http://localhost:8081")
	brokerURL := env.String("MQTT_BROKER_URL", "tcp://localhost:1883")
	level := env.String("ROOM_LEVEL", "level0")
	roomName := env.String("ROOM_NAME", "A125")
	days := int(env.Float("FORECAST_DAYS", 5))

	clock := roomtime.FromEnv()
	broker := connect(brokerURL, roomName)
	defer broker.Disconnect(250) // run at exit, giving queued messages 250 ms

	f := &forecaster{
		storageURL: storageURL,
		http:       &http.Client{Timeout: 10 * time.Second},
		broker:     broker,
		roomID:     buildsim.RoomKey(level, roomName),
		topic:      "occupancy/" + roomName + "/forecast",
		days:       days,
	}
	log.Printf("forecasting %s from the last %d weekdays, publishing to %s",
		f.roomID, f.days, f.topic)

	for {
		for {
			err := f.publish(context.Background(), clock.Now())
			if err == nil {
				break
			}
			log.Printf("retry in %s: %v", retryWait, err)
			time.Sleep(retryWait)
		}
		time.Sleep(clock.Wall(untilMidnight(clock.Now())))
	}
}

// publish forecasts every slot of the day now falls in and publishes the day
// as one retained message.
func (f *forecaster) publish(ctx context.Context, now time.Time) error {
	start := startOfDay(now)
	end := start.AddDate(0, 0, 1)
	weeks := (f.days + 4) / 5 // whole weeks holding that many weekdays
	history, err := f.fetch(ctx, start.AddDate(0, 0, -7*weeks))
	if err != nil {
		return err
	}

	forecast := occupancyForecast{
		RoomID:      f.roomID,
		Date:        start.Format(time.DateOnly),
		SlotMinutes: int(occupancyforecast.Slot / time.Minute),
		Slots:       []slotForecast{}, // encodes as [] rather than null when empty
	}
	for at := start; at.Before(end); at = at.Add(occupancyforecast.Slot) {
		people, ok := occupancyforecast.Expected(history, at, f.days)
		if !ok {
			continue
		}
		slot := slotForecast{Start: at.UTC(), People: people}
		forecast.Slots = append(forecast.Slots, slot)
	}

	payload, err := json.Marshal(forecast)
	if err != nil {
		return err
	}
	// Retained: the broker keeps this message and hands it to any later subscriber.
	token := f.broker.Publish(f.topic, 1, true, payload)
	token.Wait() // block until the broker has taken the message
	if err := token.Error(); err != nil {
		return err
	}
	log.Printf("%s: forecast for %s from %d stored counts, %d slots",
		f.roomID, forecast.Date, len(history), len(forecast.Slots))
	return nil
}

// fetch returns the room's head counts stored from since onwards.
func (f *forecaster) fetch(ctx context.Context, since time.Time) ([]occupancyforecast.Count, error) {
	query := url.Values{}
	query.Set("room", f.roomID)
	query.Set("since", since.UTC().Format(time.RFC3339))
	address := f.storageURL + "/occupancy?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // run when fetch returns
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("storage-service answered %s", resp.Status)
	}

	var readings []occupancyReading
	if err := json.NewDecoder(resp.Body).Decode(&readings); err != nil {
		return nil, fmt.Errorf("read storage-service answer: %w", err)
	}
	history := make([]occupancyforecast.Count, 0, len(readings))
	for _, r := range readings {
		history = append(history, occupancyforecast.Count{People: r.Count, Time: r.Ts})
	}
	return history, nil
}

// startOfDay is midnight at the start of t's date, in t's time zone.
func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// untilMidnight is how much room time is left of t's date.
func untilMidnight(t time.Time) time.Duration {
	return startOfDay(t).AddDate(0, 0, 1).Sub(t)
}

// connect opens the link to the broker. The client reconnects on its own, so
// a broker restart doesn't end this process.
func connect(brokerURL, roomName string) mqtt.Client {
	onConnectionLost := func(_ mqtt.Client, err error) {
		log.Printf("connection to broker lost: %v", err)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("occupancy-forecast-" + roomName).
		SetConnectionLostHandler(onConnectionLost)
	broker := mqtt.NewClient(opts)
	if token := broker.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect to broker: %v", token.Error())
	}
	return broker
}
