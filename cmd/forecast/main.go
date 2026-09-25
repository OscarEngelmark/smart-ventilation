// forecast is the CO2 forecast service for one room. It keeps a window of
// recent readings, and after each new reading publishes where CO2 is heading
// if nothing in the room changes. Reasoning: project_notes.md §7.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/forecast"
	"github.com/OscarEngelmark/smart-ventilation/internal/roomtime"
	"github.com/OscarEngelmark/smart-ventilation/schemas"
)

// co2Reading is the co2_reading message, defined by
// schemas/co2_reading.schema.json.
type co2Reading struct {
	RoomID string    `json:"room_id"`
	PPM    float64   `json:"ppm"`
	Ts     time.Time `json:"ts"`
}

// co2Forecast is the co2_forecast message, defined by
// schemas/co2_forecast.schema.json.
type co2Forecast struct {
	RoomID      string    `json:"room_id"`
	PPMForecast float64   `json:"ppm_forecast"`
	HorizonMin  float64   `json:"horizon_min"`
	Ts          time.Time `json:"ts"`
}

// forecaster holds one room's window of readings and where its forecasts go.
type forecaster struct {
	broker   mqtt.Client
	schema   *jsonschema.Schema
	clock    *roomtime.Clock
	roomID   string
	topic    string
	window   time.Duration
	horizon  time.Duration
	Cout     float64 // outdoor CO2, in ppm
	readings []forecast.Reading
}

func main() {
	brokerURL := env.String("MQTT_BROKER_URL", "tcp://localhost:1883")
	storageURL := env.String("STORAGE_URL", "http://localhost:8081")
	level := env.String("ROOM_LEVEL", "level0")
	roomName := env.String("ROOM_NAME", "A125")

	schema, err := schemas.Load("co2_reading.schema.json")
	if err != nil {
		log.Fatalf("load schema: %v", err)
	}

	f := &forecaster{
		schema:  schema,
		clock:   roomtime.FromEnv(),
		roomID:  buildsim.RoomKey(level, roomName),
		topic:   "co2/" + roomName + "/forecast",
		window:  env.Duration("FORECAST_WINDOW", 10*time.Minute),
		horizon: env.Duration("FORECAST_HORIZON", 30*time.Minute),
		Cout:    env.Float("OUTDOOR_CO2_PPM", 420),
	}
	f.readings = fillWindow(storageURL, f.roomID, f.window, f.clock)

	readingTopic := "co2/" + roomName + "/reading"
	// Subscribing on every connect, since the broker forgets subscriptions
	// when the connection drops.
	onConnect := func(client mqtt.Client) {
		token := client.Subscribe(readingTopic, 1, func(_ mqtt.Client, msg mqtt.Message) {
			f.handle(msg.Payload())
		})
		token.Wait()
		if err := token.Error(); err != nil {
			log.Fatalf("subscribe %s: %v", readingTopic, err)
		}
	}
	onConnectionLost := func(_ mqtt.Client, err error) {
		log.Printf("connection to broker lost: %v", err)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("co2-forecast-" + roomName).
		SetOnConnectHandler(onConnect).
		SetConnectionLostHandler(onConnectionLost)
	f.broker = mqtt.NewClient(opts)
	if token := f.broker.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect to broker: %v", token.Error())
	}
	defer f.broker.Disconnect(250)

	log.Printf("forecasting %s %s ahead from %s, publishing to %s",
		f.roomID, f.horizon, readingTopic, f.topic)
	select {} // wait forever; incoming readings are handled by f.handle
}

// fillWindow asks the storage-service for the room's readings from the last
// window, so a restarted service can forecast without waiting for the window
// to refill. After three failed tries it starts with an empty window, which
// the readings topic then fills.
func fillWindow(
	storageURL, roomID string,
	window time.Duration,
	clock *roomtime.Clock,
) []forecast.Reading {
	const tries = 3
	for try := 1; try <= tries; try++ {
		readings, err := storedCO2Readings(storageURL, roomID, clock.Now().Add(-window))
		if err == nil {
			log.Printf("filled window with %d stored readings", len(readings))
			return readings
		}
		log.Printf("read stored readings, try %d of %d: %v", try, tries, err)
		if try < tries {
			time.Sleep(2 * time.Second)
		}
	}
	log.Printf("starting with an empty window")
	return nil
}

// storedCO2Readings calls GET /co2 on the storage-service for one room's
// readings from since onwards.
func storedCO2Readings(storageURL, roomID string, since time.Time) ([]forecast.Reading, error) {
	query := url.Values{
		"room":  {roomID},
		"since": {since.UTC().Format(time.RFC3339)},
	}
	resp, err := http.Get(storageURL + "/co2?" + query.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // run at return, freeing the connection
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("storage-service answered %s", resp.Status)
	}

	var body []co2Reading
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	readings := make([]forecast.Reading, 0, len(body))
	for _, r := range body {
		readings = append(readings, forecast.Reading{PPM: r.PPM, Time: r.Ts})
	}
	return readings, nil
}

// handle takes one reading from the broker, adds it to the window, and
// publishes a forecast once the window holds enough readings.
func (f *forecaster) handle(payload []byte) {
	if err := schemas.Validate(f.schema, payload); err != nil {
		log.Printf("reading: invalid payload: %v", err)
		return
	}
	var r co2Reading
	if err := json.Unmarshal(payload, &r); err != nil {
		log.Printf("reading: bad payload: %v", err)
		return
	}
	f.add(forecast.Reading{PPM: r.PPM, Time: r.Ts})

	ppm, ok := forecast.Predict(f.readings, f.window, f.horizon)
	if !ok {
		log.Printf("no forecast yet: window holds too few readings")
		return
	}
	// Ventilation can only bring the room down to outdoor air.
	ppm = max(ppm, f.Cout)

	if err := f.publish(ppm); err != nil {
		log.Printf("publish forecast: %v", err)
	}
}

// add appends r to the window and drops readings older than the window.
func (f *forecaster) add(r forecast.Reading) {
	start := r.Time.Add(-f.window)
	kept := f.readings[:0] // reuses the same memory, overwritten as it goes
	for _, old := range f.readings {
		if !old.Time.Before(start) {
			kept = append(kept, old)
		}
	}
	f.readings = append(kept, r)
}

// publish sends one forecast, in ppm, to the room's forecast topic.
func (f *forecaster) publish(ppm float64) error {
	msg := co2Forecast{
		RoomID:      f.roomID,
		PPMForecast: ppm,
		HorizonMin:  f.horizon.Minutes(),
		Ts:          f.clock.Now().UTC(),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	token := f.broker.Publish(f.topic, 1, false, payload)
	token.Wait() // block until the broker has taken the message
	if err := token.Error(); err != nil {
		return err
	}
	log.Printf("%s: %.0f ppm in %s", f.roomID, ppm, f.horizon)
	return nil
}
