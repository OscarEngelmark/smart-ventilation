// decision chooses the damper level for one room. It keeps the latest CO2
// reading, head count, occupancy forecast and room model from the broker,
// chooses a level on every CO2 reading, and commands the actuator whenever
// the level changes, publishing a copy of each command for storage.
// Reasoning: project_notes.md §4 and §7.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	// The image carries no timezone database, so TZ is otherwise ignored.
	_ "time/tzdata"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/planner"
	"github.com/OscarEngelmark/smart-ventilation/internal/roommodel"
	"github.com/OscarEngelmark/smart-ventilation/internal/roomtime"
)

// co2Reading is the co2_reading message, defined by
// schemas/co2_reading.schema.json.
type co2Reading struct {
	RoomID string    `json:"room_id"`
	PPM    float64   `json:"ppm"`
	Ts     time.Time `json:"ts"`
}

// occupancyReading is the occupancy_reading message, defined by
// schemas/occupancy_reading.schema.json.
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

// roomModel is the room_model message, defined by
// schemas/room_model.schema.json.
type roomModel struct {
	RoomID string  `json:"room_id"`
	Date   string  `json:"date"`
	A      float64 `json:"a"`
	B0     float64 `json:"b0"`
	B1     float64 `json:"b1"`
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

// inputs is the latest of each message the decision is made from. A nil
// field has not arrived yet.
type inputs struct {
	co2      *co2Reading
	people   *occupancyReading
	forecast *occupancyForecast
	model    *roomModel
}

// latest holds the inputs as the broker delivers them. The broker's handlers
// write it and the decision loop reads it, so every access goes through mu.
type latest struct {
	mu sync.Mutex
	in inputs
}

// snapshot returns a copy of the inputs as they are now.
func (l *latest) snapshot() inputs {
	l.mu.Lock()
	defer l.mu.Unlock() // run when snapshot returns
	return l.in
}

// commander sends the chosen level to the actuator and publishes a copy.
type commander struct {
	actuatorURL string
	http        *http.Client
	broker      mqtt.Client
	clock       *roomtime.Clock
	roomID      string
	topic       string
}

func main() {
	brokerURL := env.String("MQTT_BROKER_URL", "tcp://localhost:1883")
	actuatorURL := env.String("ACTUATOR_URL", "http://localhost:8080")
	level := env.String("ROOM_LEVEL", "level0")
	roomName := env.String("ROOM_NAME", "A125")
	settings := planner.Settings{
		Target:  env.Float("PLAN_TARGET_PPM", 950),
		Horizon: env.Duration("PLAN_HORIZON", time.Hour),
		Cout:    env.Float("OUTDOOR_CO2_PPM", 420),
	}

	clock := roomtime.FromEnv()
	state := &latest{}
	readings := make(chan co2Reading, 10) // CO2 readings waiting for a decision

	onConnect := func(client mqtt.Client) {
		log.Printf("connected to broker, subscribing")
		subscribeAll(client, roomName, state, readings)
	}
	broker := connect(brokerURL, roomName, onConnect)
	defer broker.Disconnect(250) // run at exit, giving queued messages 250 ms

	c := &commander{
		actuatorURL: actuatorURL,
		http:        &http.Client{Timeout: 5 * time.Second},
		broker:      broker,
		clock:       clock,
		roomID:      buildsim.RoomKey(level, roomName),
		topic:       "ventilation/" + roomName + "/command",
	}
	log.Printf("deciding for %s, commanding %s, keeping CO2 under %.0f ppm over the next %v",
		c.roomID, c.actuatorURL, settings.Target, settings.Horizon)

	var sent bool        // whether a level has been accepted by the actuator yet
	var last float64     // the last level the actuator accepted
	for range readings { // wait for each CO2 reading in turn
		in := state.snapshot()
		chosen := choose(in, settings)
		if sent && chosen == last {
			continue
		}
		if err := c.send(chosen); err != nil {
			log.Printf("command %.2f not applied, retrying on the next reading: %v", chosen, err)
			continue
		}
		sent, last = true, chosen
		log.Printf("%s: damper %.2f at %.0f ppm, %s people", c.roomID, chosen,
			in.co2.PPM, headCount(in.people))
	}
}

// choose returns the damper level, 0 (closed) to 1 (fully open), for the
// room's current inputs: the planner's level once a forecast and a room model
// have arrived, and closed until then.
func choose(in inputs, s planner.Settings) float64 {
	if in.forecast == nil || in.model == nil {
		return 0
	}
	f := plannerForecast(in.forecast)
	m := roommodel.Model{A: in.model.A, B0: in.model.B0, B1: in.model.B1}
	return planner.Level(in.co2.PPM, in.co2.Ts, f, m, s)
}

// plannerForecast turns the forecast message into the planner's form.
func plannerForecast(msg *occupancyForecast) planner.Forecast {
	f := planner.Forecast{SlotLength: time.Duration(msg.SlotMinutes) * time.Minute}
	for _, slot := range msg.Slots {
		f.Slots = append(f.Slots, planner.Slot{Start: slot.Start, N: slot.People})
	}
	return f
}

// send commands the actuator to the level and, once the actuator has accepted
// it, publishes a copy on the command topic.
func (c *commander) send(level float64) error {
	cmd := ventilationCommand{
		RoomID: c.roomID,
		Level:  level,
		Ts:     c.clock.Now().UTC(),
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	resp, err := c.http.Post(c.actuatorURL+"/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close() // run when send returns
	var answer commandResponse
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return fmt.Errorf("read actuator answer: %w", err)
	}
	if !answer.Accepted {
		return fmt.Errorf("actuator refused the command: %s", resp.Status)
	}

	token := c.broker.Publish(c.topic, 1, false, payload)
	token.Wait() // block until the broker has taken the message
	if err := token.Error(); err != nil {
		log.Printf("command applied but its copy was not published: %v", err)
	}
	return nil
}

// subscribeAll subscribes to the room's four inputs. Each message is stored as
// the latest of its kind; a CO2 reading is also queued for a decision.
func subscribeAll(client mqtt.Client, roomName string, state *latest, readings chan<- co2Reading) {
	subscribe(client, "co2/"+roomName+"/reading", func(payload []byte) {
		var r co2Reading
		if !decode("CO2 reading", payload, &r) {
			return
		}
		state.mu.Lock()
		state.in.co2 = &r
		state.mu.Unlock()
		readings <- r
	})

	subscribe(client, "occupancy/"+roomName+"/reading", func(payload []byte) {
		var r occupancyReading
		if !decode("head count", payload, &r) {
			return
		}
		state.mu.Lock()
		changed := state.in.people == nil || state.in.people.Count != r.Count
		state.in.people = &r
		state.mu.Unlock()
		if changed {
			log.Printf("head count %d", r.Count)
		}
	})

	subscribe(client, "occupancy/"+roomName+"/forecast", func(payload []byte) {
		var f occupancyForecast
		if !decode("forecast", payload, &f) {
			return
		}
		state.mu.Lock()
		state.in.forecast = &f
		state.mu.Unlock()
		log.Printf("forecast for %s, %d slots", f.Date, len(f.Slots))
	})

	subscribe(client, "room/"+roomName+"/model", func(payload []byte) {
		var m roomModel
		if !decode("room model", payload, &m) {
			return
		}
		state.mu.Lock()
		state.in.model = &m
		state.mu.Unlock()
		log.Printf("room model from %s: a=%.4g b0=%.4g b1=%.4g", m.Date, m.A, m.B0, m.B1)
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

// decode reads a JSON message into out, logging and reporting false when it
// can't.
func decode(kind string, payload []byte, out any) bool {
	if err := json.Unmarshal(payload, out); err != nil {
		log.Printf("%s: bad payload: %v", kind, err)
		return false
	}
	return true
}

// headCount is the head count as text, or "unknown" before the first one.
func headCount(r *occupancyReading) string {
	if r == nil {
		return "unknown"
	}
	return fmt.Sprint(r.Count)
}

// connect opens the link to the broker. The client reconnects on its own, and
// onConnect runs on every connect, since with a clean session the broker
// forgets subscriptions on disconnect.
func connect(brokerURL, roomName string, onConnect mqtt.OnConnectHandler) mqtt.Client {
	onConnectionLost := func(_ mqtt.Client, err error) {
		log.Printf("connection to broker lost: %v", err)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("decision-" + roomName).
		SetOnConnectHandler(onConnect).
		SetConnectionLostHandler(onConnectionLost)
	broker := mqtt.NewClient(opts)
	if token := broker.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect to broker: %v", token.Error())
	}
	return broker
}
