// occupancy-sensor is the people counter for one room. It registers its device
// with BuildSim on startup, then counts the people BuildSim lists in the room,
// reports the count as its device value, and publishes it to the broker.
// Reasoning: project_notes.md §4 and §5.
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
)

// publisher is what one reading needs: the room it counts, the device it
// reports to, and where it is published.
type publisher struct {
	client   *buildsim.Client
	broker   mqtt.Client
	sensorID string
	roomID   string
	topic    string
}

// occupancyReading is the occupancy_reading message, defined by
// schemas/occupancy_reading.schema.json.
type occupancyReading struct {
	RoomID string    `json:"room_id"`
	Count  int       `json:"count"`
	Ts     time.Time `json:"ts"`
}

func main() {
	baseURL := env.String("BUILDSIM_URL", buildsim.DefaultURL)
	brokerURL := env.String("MQTT_BROKER_URL", "tcp://localhost:1883")
	level := env.String("ROOM_LEVEL", "level0")
	roomName := env.String("ROOM_NAME", "A125")
	sensorID := env.String("OCCUPANCY_SENSOR_ID", roomName+"-occupancy")
	interval := env.Duration("SENSOR_INTERVAL", 10*time.Second)

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
		sensorID: sensorID,
		roomID:   buildsim.RoomKey(level, roomName),
		topic:    "occupancy/" + roomName + "/reading",
	}
	log.Printf("counting people in %s every %s, publishing to %s",
		p.roomID, interval, p.topic)

	ticker := time.NewTicker(interval)
	for {
		if err := p.publish(ctx); err != nil {
			log.Printf("skip this reading: %v", err)
		}
		<-ticker.C // wait for the next tick
	}
}

// publish counts the people in the room, writes the count to the device, and
// sends it on as one reading.
func (p *publisher) publish(ctx context.Context) error {
	building, err := p.client.Occupancy(ctx)
	if err != nil {
		return err
	}
	count := len(building[p.roomID].Persons) // a room BuildSim omits is empty

	if err := p.client.SetSensorValue(ctx, p.sensorID, float64(count)); err != nil {
		return err
	}

	reading := occupancyReading{
		RoomID: p.roomID,
		Count:  count,
		Ts:     time.Now().UTC(),
	}
	payload, err := json.Marshal(reading)
	if err != nil {
		return err
	}
	token := p.broker.Publish(p.topic, 1, false, payload)
	token.Wait() // block until the broker has taken the message
	if err := token.Error(); err != nil {
		return err
	}
	log.Printf("%s: %d people", p.roomID, count)
	return nil
}

// equipment is the device record this process registers: one equipment record
// in the room, holding the one people counter.
func equipment(sensorID, level, roomName string) buildsim.Equipment {
	device := buildsim.Sensor{
		ID:       sensorID,
		Name:     "Occupancy",
		Type:     "occupancy_sensor",
		DataType: "text",
		Unit:     "people",
	}
	return buildsim.Equipment{
		ID:       sensorID,
		Name:     "People counter",
		Type:     "occupancy_sensor",
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
		SetClientID("occupancy-sensor-" + roomName).
		SetConnectionLostHandler(onConnectionLost)
	broker := mqtt.NewClient(opts)
	if token := broker.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect to broker: %v", token.Error())
	}
	return broker
}
