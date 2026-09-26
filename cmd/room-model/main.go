// room-model learns how one room's CO2 responds to its people and its damper.
// At startup and at each room midnight it fetches the room's recent CO2
// readings, head counts and damper commands from the storage-service, fits
// the room model with internal/roommodel, and publishes the three rates as
// one retained message. Reasoning: project_notes.md §7.
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
	"github.com/OscarEngelmark/smart-ventilation/internal/roommodel"
	"github.com/OscarEngelmark/smart-ventilation/internal/roomtime"
)

// retryWait is the real-time pause between tries at the storage-service.
const retryWait = 5 * time.Second

// learner is what one day's fit needs: where the history comes from, how much
// of it to use, and where the rates are published.
type learner struct {
	storageURL string
	http       *http.Client
	broker     mqtt.Client
	roomID     string
	topic      string
	days       int
	Cout       float64 // outdoor CO2, ppm
}

// co2Reading is the co2_reading message the storage-service returns, defined
// by schemas/co2_reading.schema.json.
type co2Reading struct {
	PPM float64   `json:"ppm"`
	Ts  time.Time `json:"ts"`
}

// occupancyReading is the occupancy_reading message the storage-service
// returns, defined by schemas/occupancy_reading.schema.json.
type occupancyReading struct {
	Count int       `json:"count"`
	Ts    time.Time `json:"ts"`
}

// ventilationCommand is the ventilation_command message the storage-service
// returns, defined by schemas/ventilation_command.schema.json.
type ventilationCommand struct {
	Level float64   `json:"level"`
	Ts    time.Time `json:"ts"`
}

// roomModel is the room_model message, defined by
// schemas/room_model.schema.json.
type roomModel struct {
	RoomID string  `json:"room_id"`
	Date   string  `json:"date"`
	A      float64 `json:"a"`
	B0     float64 `json:"b0"`
	B1     float64 `json:"b1"`
}

func main() {
	storageURL := env.String("STORAGE_URL", "http://localhost:8081")
	brokerURL := env.String("MQTT_BROKER_URL", "tcp://localhost:1883")
	level := env.String("ROOM_LEVEL", "level0")
	roomName := env.String("ROOM_NAME", "A125")
	days := int(env.Float("FIT_DAYS", 7))
	Cout := env.Float("OUTDOOR_CO2_PPM", 420)

	clock := roomtime.FromEnv()
	broker := connect(brokerURL, roomName)
	defer broker.Disconnect(250) // run at exit, giving queued messages 250 ms

	l := &learner{
		storageURL: storageURL,
		http:       &http.Client{Timeout: 30 * time.Second},
		broker:     broker,
		roomID:     buildsim.RoomKey(level, roomName),
		topic:      "room/" + roomName + "/model",
		days:       days,
		Cout:       Cout,
	}
	log.Printf("fitting %s from the last %d days, publishing to %s",
		l.roomID, l.days, l.topic)

	for {
		for {
			err := l.publish(context.Background(), clock.Now())
			if err == nil {
				break
			}
			log.Printf("retry in %s: %v", retryWait, err)
			time.Sleep(retryWait)
		}
		time.Sleep(clock.Wall(untilMidnight(clock.Now())))
	}
}

// publish fits the model to the last l.days days before now and publishes it
// as one retained message. When the history can't tell the rates apart it
// logs why and publishes nothing, so the last published model stays in force;
// only a failure to fetch or publish is returned, to be retried.
func (l *learner) publish(ctx context.Context, now time.Time) error {
	since := startOfDay(now).AddDate(0, 0, -l.days)
	co2, people, damper, err := l.history(ctx, since)
	if err != nil {
		return err
	}

	fit, err := roommodel.Fit(co2, people, damper, l.Cout)
	if err != nil {
		log.Printf("%s: no model from %d readings, %d counts, %d commands: %v",
			l.roomID, len(co2), len(people), len(damper), err)
		return nil
	}

	msg := roomModel{
		RoomID: l.roomID,
		Date:   now.Format(time.DateOnly),
		A:      fit.A,
		B0:     fit.B0,
		B1:     fit.B1,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	// Retained: the broker keeps this message and hands it to any later subscriber.
	token := l.broker.Publish(l.topic, 1, true, payload)
	token.Wait() // block until the broker has taken the message
	if err := token.Error(); err != nil {
		return err
	}
	log.Printf("%s: model for %s from %d readings: a=%.4g b0=%.4g b1=%.4g",
		l.roomID, msg.Date, len(co2), fit.A, fit.B0, fit.B1)
	return nil
}

// history returns the room's CO2 readings, head counts and damper levels
// stored from since onwards, each in time order. The damper levels are led by
// the command in force at since.
func (l *learner) history(ctx context.Context, since time.Time) (co2, people, damper []roommodel.Sample, err error) {
	var readings []co2Reading
	if err := l.get(ctx, "/co2", since, &readings); err != nil {
		return nil, nil, nil, err
	}
	var counts []occupancyReading
	if err := l.get(ctx, "/occupancy", since, &counts); err != nil {
		return nil, nil, nil, err
	}
	var commands []ventilationCommand
	if err := l.get(ctx, "/commands", since, &commands); err != nil {
		return nil, nil, nil, err
	}

	for _, r := range readings {
		co2 = append(co2, roommodel.Sample{Value: r.PPM, Time: r.Ts})
	}
	for _, c := range counts {
		people = append(people, roommodel.Sample{Value: float64(c.Count), Time: c.Ts})
	}
	for _, c := range commands {
		damper = append(damper, roommodel.Sample{Value: c.Level, Time: c.Ts})
	}
	return co2, people, damper, nil
}

// get asks the storage-service at path for the room's messages from since
// onwards and decodes the JSON array it answers into out.
func (l *learner) get(ctx context.Context, path string, since time.Time, out any) error {
	query := url.Values{}
	query.Set("room", l.roomID)
	query.Set("since", since.UTC().Format(time.RFC3339))
	address := l.storageURL + path + "?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	resp, err := l.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // run when get returns
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("storage-service answered %s to %s", resp.Status, path)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("read storage-service answer to %s: %w", path, err)
	}
	return nil
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
		SetClientID("room-model-" + roomName).
		SetConnectionLostHandler(onConnectionLost)
	broker := mqtt.NewClient(opts)
	if token := broker.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect to broker: %v", token.Error())
	}
	return broker
}
