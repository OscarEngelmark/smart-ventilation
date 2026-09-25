// co2-sensor is the CO2 sensor process for one room. It registers its device
// with BuildSim on startup, then reads the level the physical model wrote and
// publishes each reading to the broker, where the rest of the system picks it
// up. Reasoning: project_notes.md §4 and §5.
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/roomtime"
)

// publisher is what one reading needs: the device it is read from, and where
// it is published.
type publisher struct {
	client   *buildsim.Client
	broker   mqtt.Client
	clock    *roomtime.Clock
	sensorID string
	roomID   string
	topic    string
}

// co2Reading is the co2_reading message, defined by
// schemas/co2_reading.schema.json.
type co2Reading struct {
	RoomID string    `json:"room_id"`
	PPM    float64   `json:"ppm"`
	Ts     time.Time `json:"ts"`
}

func main() {
	baseURL := env.String("BUILDSIM_URL", buildsim.DefaultURL)
	brokerURL := env.String("MQTT_BROKER_URL", "tcp://localhost:1883")
	level := env.String("ROOM_LEVEL", "level0")
	roomName := env.String("ROOM_NAME", "A125")
	sensorID := env.String("CO2_SENSOR_ID", roomName+"-co2")
	interval := env.Duration("SENSOR_INTERVAL", 10*time.Second)

	clock := roomtime.FromEnv()
	client := buildsim.New(baseURL) // this program's link to BuildSim
	ctx := context.Background()     // empty context, no cancellation or timeout

	if err := client.Register(ctx, equipment(sensorID, level, roomName)); err != nil {
		log.Fatalf("register %s: %v", sensorID, err)
	}

	broker := connect(brokerURL, roomName)
	defer broker.Disconnect(250) // run at exit, giving queued messages 250 ms

	p := &publisher{
		client:   client,
		broker:   broker,
		clock:    clock,
		sensorID: sensorID,
		roomID:   buildsim.RoomKey(level, roomName),
		topic:    "co2/" + roomName + "/reading",
	}
	log.Printf("reading %s every %s, publishing to %s", sensorID, interval, p.topic)

	ticker := clock.NewTicker(interval)
	for {
		// A reading fails while the physical model has written no value yet,
		// which is the normal state until it has run once.
		if err := p.publish(ctx); err != nil {
			log.Printf("skip this reading: %v", err)
		}
		<-ticker.C // wait for the next tick
	}
}

// publish reads the sensor's current value from BuildSim and sends it on as
// one reading.
func (p *publisher) publish(ctx context.Context) error {
	ppm, err := p.client.SensorValue(ctx, p.sensorID)
	if err != nil {
		return err
	}
	reading := co2Reading{
		RoomID: p.roomID,
		PPM:    ppm,
		Ts:     p.clock.Now().UTC(),
	}
	payload, err := json.Marshal(reading)
	if err != nil {
		return err
	}
	// QoS 1 delivers the message at least once; not retained, so a subscriber
	// that connects later waits for the next reading instead of seeing an old one.
	token := p.broker.Publish(p.topic, 1, false, payload)
	token.Wait() // block until the broker has taken the message
	if err := token.Error(); err != nil {
		return err
	}
	log.Printf("%s: %.0f ppm", p.roomID, ppm)
	return nil
}

// equipment is the device record this process registers: one equipment record
// in the room, holding the one CO2 sensor.
func equipment(sensorID, level, roomName string) buildsim.Equipment {
	device := buildsim.Sensor{
		ID:       sensorID,
		Name:     "CO2",
		Type:     "co2_sensor", // holds "co2", so the 3D view shades the room by this value
		DataType: "text",
		Unit:     "ppm",
	}
	return buildsim.Equipment{
		ID:       sensorID,
		Name:     "CO2 sensor",
		Type:     "co2_sensor",
		Category: "sensor",
		Level:    level,
		Room:     roomName,
		Sensors:  []buildsim.Sensor{device},
	}
}

// connect opens the link to the broker. The client reconnects on its own, so
// a broker restart doesn't end this process.
func connect(brokerURL, roomName string) mqtt.Client {
	onConnectionLost := func(_ mqtt.Client, err error) {
		log.Printf("connection to broker lost: %v", err)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("co2-sensor-" + roomName).
		SetConnectionLostHandler(onConnectionLost)
	broker := mqtt.NewClient(opts)
	if token := broker.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect to broker: %v", token.Error())
	}
	return broker
}
