// decision chooses the damper level for one room. It keeps the latest CO2
// reading, head count and room model from the broker,
// chooses a level on every CO2 reading, and commands the actuator whenever
// the level changes, publishing a copy of each command for storage.
// Reasoning: project_notes.md §4 and §7.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	// The image carries no timezone database, so TZ is otherwise ignored.
	_ "time/tzdata"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/message"
	"github.com/OscarEngelmark/smart-ventilation/internal/mqttclient"
	"github.com/OscarEngelmark/smart-ventilation/internal/planner"
	"github.com/OscarEngelmark/smart-ventilation/internal/roommodel"
	"github.com/OscarEngelmark/smart-ventilation/internal/roomtime"
	"github.com/OscarEngelmark/smart-ventilation/internal/shutdown"
)

// inputs is the latest of each message the decision is made from. A nil
// field has not arrived yet.
type inputs struct {
	co2    *message.CO2Reading
	people *message.OccupancyReading
	model  *message.RoomModel
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
	shutdown.ExitZeroOnStop()

	brokerURL := env.String("MQTT_BROKER_URL")
	actuatorURL := env.String("ACTUATOR_URL")
	level := env.String("ROOM_LEVEL")
	roomName := env.String("ROOM_NAME")
	settings := planSettings()
	ignoreModel := env.Bool("IGNORE_ROOM_MODEL") // true runs the CO2-only switch all the time

	clock := roomtime.FromEnv()
	state := &latest{}
	readings := make(chan message.CO2Reading, 10) // CO2 readings waiting for a decision

	// Runs on every connect, since the broker forgets subscriptions on disconnect.
	onConnect := func(client mqtt.Client) {
		log.Printf("connected to broker, subscribing")
		subscribeAll(client, roomName, ignoreModel, state, readings)
	}
	broker, err := mqttclient.Connect(brokerURL, "decision-"+roomName, onConnect)
	if err != nil {
		log.Fatalf("connect to broker: %v", err)
	}
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
	if ignoreModel {
		log.Printf("IGNORE_ROOM_MODEL is set: no room model is used, the damper switches on CO2 alone")
	}

	decide(readings, state, settings, c)
}

// planSettings reads the planner's settings from the environment.
func planSettings() planner.Settings {
	return planner.Settings{
		Target:      env.Float("PLAN_TARGET_PPM"),
		LowerMargin: env.Float("PLAN_LOWER_MARGIN_PPM"),
		Horizon:     env.Duration("PLAN_HORIZON"),
		Cout:        env.Float("OUTDOOR_CO2_PPM"),
		CloseBelow:  env.Float("FALLBACK_CLOSE_PPM"),
	}
}

// decide chooses a level on every CO2 reading and sends it to the actuator
// whenever it differs from the last level the actuator accepted. A level the
// actuator doesn't accept is tried again on the next reading.
func decide(readings <-chan message.CO2Reading, state *latest, s planner.Settings, c *commander) {
	var sent bool        // whether a level has been accepted by the actuator yet
	var last float64     // the last level the actuator accepted
	for range readings { // wait for each CO2 reading in turn
		in := state.snapshot()
		chosen := choose(in, s, last)
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
// room's current inputs, where current is the level the damper is at now.
// Without a room model it switches on CO2 alone; before the first head count
// it plans for an empty room.
func choose(in inputs, s planner.Settings, current float64) float64 {
	if in.model == nil {
		return planner.Switch(in.co2.PPM, current, s)
	}
	counted := 0
	if in.people != nil {
		counted = in.people.Count
	}
	m := roommodel.Model{A: in.model.A, B0: in.model.B0, B1: in.model.B1}
	return planner.Level(in.co2.PPM, current, counted, m, s)
}

// send commands the actuator to the level and, once the actuator has accepted
// it, publishes a copy on the command topic.
func (c *commander) send(level float64) error {
	cmd := message.VentilationCommand{
		RoomID: c.roomID,
		Level:  level,
		Ts:     c.clock.Now().UTC(),
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	if err := c.post(payload); err != nil {
		return err
	}
	c.publishCopy(cmd)
	return nil
}

// post sends a ventilation_command payload to the actuator and returns an
// error unless the actuator answers 200 OK, its sign of a stored command.
func (c *commander) post(payload []byte) error {
	resp, err := c.http.Post(c.actuatorURL+"/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close() // run when post returns
	if resp.StatusCode != http.StatusOK {
		reason, _ := io.ReadAll(io.LimitReader(resp.Body, 512)) // the actuator's plain-text reason
		return fmt.Errorf("actuator refused the command: %s: %s", resp.Status, bytes.TrimSpace(reason))
	}
	return nil
}

// publishCopy publishes an applied command on the command topic for storage,
// logging if the broker doesn't take it.
func (c *commander) publishCopy(cmd message.VentilationCommand) {
	if err := mqttclient.PublishJSON(c.broker, c.topic, false, cmd); err != nil {
		log.Printf("command applied but its copy was not published: %v", err)
	}
}

// subscribeAll subscribes to the room's three inputs, leaving out the room
// model when ignoreModel is true. Each message is stored as the latest of its
// kind; a CO2 reading is also queued for a decision.
func subscribeAll(
	client mqtt.Client, roomName string, ignoreModel bool,
	state *latest, readings chan<- message.CO2Reading,
) {
	subscribe(client, "co2/"+roomName+"/reading", func(payload []byte) {
		if r, ok := state.keepCO2(payload); ok {
			readings <- r
		}
	})
	subscribe(client, "occupancy/"+roomName+"/reading", state.keepHeadCount)
	if !ignoreModel {
		subscribe(client, "room/"+roomName+"/model", state.keepModel)
	}
}

// keepCO2 stores a CO2 reading as the latest and returns it, or reports false
// if the payload can't be read.
func (l *latest) keepCO2(payload []byte) (message.CO2Reading, bool) {
	var r message.CO2Reading
	if !decode("CO2 reading", payload, &r) {
		return r, false
	}
	l.mu.Lock()
	l.in.co2 = &r
	l.mu.Unlock()
	return r, true
}

// keepHeadCount stores a head count as the latest, logging it when it differs
// from the one before.
func (l *latest) keepHeadCount(payload []byte) {
	var r message.OccupancyReading
	if !decode("head count", payload, &r) {
		return
	}
	l.mu.Lock()
	changed := l.in.people == nil || l.in.people.Count != r.Count
	l.in.people = &r
	l.mu.Unlock()
	if changed {
		log.Printf("head count %d", r.Count)
	}
}

// keepModel stores a room model as the latest and logs its rates.
func (l *latest) keepModel(payload []byte) {
	var m message.RoomModel
	if !decode("room model", payload, &m) {
		return
	}
	l.mu.Lock()
	l.in.model = &m
	l.mu.Unlock()
	log.Printf("room model from %s: a=%.4g b0=%.4g b1=%.4g", m.Date, m.A, m.B0, m.B1)
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
func headCount(r *message.OccupancyReading) string {
	if r == nil {
		return "unknown"
	}
	return fmt.Sprint(r.Count)
}
