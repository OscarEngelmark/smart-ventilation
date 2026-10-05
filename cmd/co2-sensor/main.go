// co2-sensor is the CO2 sensor process for one room. It registers its device
// with BuildSim on startup, then reads the level the physical model wrote and
// publishes each reading to the broker, where the rest of the system picks it
// up. Reasoning: project_notes.md §4 and §5.
package main

import (
	"context"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/message"
	"github.com/OscarEngelmark/smart-ventilation/internal/mqttclient"
	"github.com/OscarEngelmark/smart-ventilation/internal/roomtime"
	"github.com/OscarEngelmark/smart-ventilation/internal/shutdown"
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

func main() {
	shutdown.ExitZeroOnStop()

	baseURL := env.String("BUILDSIM_URL")
	brokerURL := env.String("MQTT_BROKER_URL")
	level := env.String("ROOM_LEVEL")
	roomName := env.String("ROOM_NAME")
	sensorID := env.String("CO2_SENSOR_ID")
	interval := env.Duration("SENSOR_INTERVAL")

	clock := roomtime.FromEnv()
	client := buildsim.New(baseURL) // this program's link to BuildSim
	ctx := context.Background()     // empty context, no cancellation or timeout

	if err := client.Register(ctx, equipment(sensorID, level, roomName)); err != nil {
		log.Fatalf("register %s: %v", sensorID, err)
	}

	broker, err := mqttclient.Connect(brokerURL, "co2-sensor-"+roomName, nil)
	if err != nil {
		log.Fatalf("connect to broker: %v", err)
	}
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
	reading := message.CO2Reading{
		RoomID: p.roomID,
		PPM:    ppm,
		Ts:     p.clock.Now().UTC(),
	}
	// Not retained, so a subscriber that connects later waits for the next reading.
	if err := mqttclient.PublishJSON(p.broker, p.topic, false, reading); err != nil {
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
