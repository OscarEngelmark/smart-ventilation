// physical-model stands in for the physical world. Each cycle it reads the
// room's occupancy and damper position from BuildSim, advances the CO2 mass
// balance by one step, and writes the new level back to the CO2 sensor. It
// keeps nothing between cycles: BuildSim holds the state.
// Reasoning: project_notes.md §4 and §6.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/co2"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/room"
	"github.com/OscarEngelmark/smart-ventilation/internal/roomtime"
	"github.com/OscarEngelmark/smart-ventilation/internal/shutdown"
)

// model is what one cycle needs: the room's parameters, and the devices its
// state is read from and written to.
type model struct {
	client   *buildsim.Client
	roomID   string
	sensorID string
	damperID string
	V        float64 // air volume, m³
	Qmin     float64 // airflow with the damper closed, L/s
	Qmax     float64 // airflow with the damper fully open, L/s
	G        float64 // CO2 one person breathes out, L/s
	Cout     float64 // outdoor CO2, ppm
	dt       time.Duration
}

func main() {
	shutdown.ExitZeroOnStop()

	baseURL := env.String("BUILDSIM_URL")
	roomID := env.String("ROOM_ID")

	clock := roomtime.FromEnv()
	client := buildsim.New(baseURL) // this program's link to BuildSim
	ctx := context.Background()     // empty context, no cancellation or timeout

	m, err := newModel(ctx, client, roomID)
	if err != nil {
		log.Fatal(err)
	}

	ticker := clock.NewTicker(m.dt)
	for {
		// A cycle fails while a device is missing, which is the normal state
		// until the sensor and actuator processes have registered theirs.
		if err := m.step(ctx); err != nil {
			log.Printf("skip this cycle: %v", err)
		}
		<-ticker.C // wait for the next tick
	}
}

// newModel builds the room's model from its settings and from its floor area,
// which it reads from BuildSim.
func newModel(ctx context.Context, client *buildsim.Client, roomID string) (*model, error) {
	areaPerPerson := env.Float("AREA_PER_PERSON_M2")
	ceilingHeight := env.Float("CEILING_HEIGHT_M")
	G := env.Float("CO2_PER_PERSON_LPS")
	Cout := env.Float("OUTDOOR_CO2_PPM")
	Cthres := env.Float("CO2_THRESHOLD_PPM")
	minPerArea := env.Float("MIN_AIRFLOW_LPS_PER_M2")
	factor := env.Float("MAX_AIRFLOW_FACTOR")

	// Read the room's area from BuildSim; every parameter below follows from it.
	area, err := client.RoomArea(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("read area of %s: %w", roomID, err)
	}
	m := &model{
		client:   client,
		roomID:   roomID,
		sensorID: env.String("CO2_SENSOR_ID"),
		damperID: env.String("DAMPER_ID"),
		V:        room.Volume(area, ceilingHeight),
		Qmin:     room.MinAirflow(area, minPerArea),
		Qmax:     room.MaxAirflow(area, areaPerPerson, G, Cthres, Cout, factor),
		G:        G,
		Cout:     Cout,
		dt:       env.Duration("SIM_STEP"),
	}
	log.Printf("room %s is %.1f m², holding %.1f m³ of air, ventilated at %.1f to %.1f L/s",
		roomID, area, m.V, m.Qmin, m.Qmax)
	return m, nil
}

// step advances the room by dt: read the state BuildSim holds, compute the
// CO2 level at the end of the step, and write it back.
func (m *model) step(ctx context.Context) error {
	building, err := m.client.Occupancy(ctx)
	if err != nil {
		return err
	}
	N := len(building[m.roomID].Persons) // a room BuildSim omits is empty

	damper, err := m.client.ActuatorState(ctx, m.damperID)
	if errors.Is(err, buildsim.ErrNoValue) {
		damper = 0 // no command yet, so the damper is still closed
	} else if err != nil {
		return err
	}

	C, err := m.client.SensorValue(ctx, m.sensorID)
	if errors.Is(err, buildsim.ErrNoValue) {
		C = m.Cout // nothing written yet, so the room starts at outdoor level
	} else if err != nil {
		return err
	}

	Q := room.Airflow(damper, m.Qmin, m.Qmax)
	Cnext := co2.Step(C, N, m.V, Q, m.G, m.Cout, m.dt)
	log.Printf("%d people, damper %.2f, %.0f -> %.0f ppm", N, damper, C, Cnext)
	return m.client.SetSensorValue(ctx, m.sensorID, Cnext)
}
