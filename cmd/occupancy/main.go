// occupancy simulates the people in the target room and writes them to
// BuildSim, where every other process reads them from. It stands in for the
// physical world alongside the physical-model process, and is kept separate
// from it so the CO2 model never has to know how people are generated (see
// project_notes.md §4).
//
// It is the only process allowed to write occupancy: BuildSim replaces the
// occupancy of the whole building on every write, so a second writer would
// erase this one's rooms.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	// The container image carries no timezone database, so without this the
	// TZ setting would be ignored and the schedule would silently run on UTC.
	_ "time/tzdata"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/occupancy"
)

func main() {
	baseURL := getEnv("BUILDSIM_URL", "http://localhost:9090")
	level := getEnv("ROOM_LEVEL", "level0")
	room := getEnv("ROOM_NAME", "A125")
	areaPerPerson := getEnvFloat("AREA_PER_PERSON_M2", 5)
	interval := getEnvDuration("OCCUPANCY_INTERVAL", 10*time.Second)

	client := buildsim.New(baseURL)
	ctx := context.Background()

	// BuildSim holds the building, so the room's area — and with it how many
	// people the room holds — is read from it rather than configured here.
	// The process exits if BuildSim isn't up yet and Docker starts it again,
	// the same way the storage service waits for the broker.
	area, err := client.RoomArea(ctx, level, room)
	if err != nil {
		log.Fatalf("read area of %s: %v", buildsim.RoomKey(level, room), err)
	}
	capacity := int(math.Round(area / areaPerPerson))
	log.Printf("room %s is %.1f m², holding up to %d people",
		buildsim.RoomKey(level, room), area, capacity)

	people := namePeople(room, capacity)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		// Room time is the wall clock: the simulation runs at real speed for
		// now, so the schedule's times are the times of day it is run at.
		present := occupancy.PeopleAt(time.Now(), capacity)
		if err := publish(ctx, client, level, room, people[:present]); err != nil {
			// A failed write leaves BuildSim showing the previous occupancy
			// until the next cycle, which is a better outcome than stopping.
			log.Printf("write occupancy: %v", err)
		}
		<-ticker.C
	}
}

// publish writes the room's people to BuildSim. Every other room is left out
// of the payload, so this also asserts that they are empty — which holds
// while the project simulates one room.
func publish(
	ctx context.Context,
	client *buildsim.Client,
	level, room string,
	present []buildsim.Person,
) error {
	return client.SetOccupancy(ctx, map[string]buildsim.RoomOccupancy{
		buildsim.RoomKey(level, room): {
			Persons: present,
			Aliens:  []buildsim.Alien{},
		},
	})
}

// namePeople builds the room's full set of occupants once, so that a person
// keeps the same identity between cycles instead of being renamed whenever
// the count changes.
func namePeople(room string, capacity int) []buildsim.Person {
	people := make([]buildsim.Person, capacity)
	for i := range people {
		people[i] = buildsim.Person{
			ID:   fmt.Sprintf("%s-person-%d", room, i+1),
			Name: fmt.Sprintf("Person %d", i+1),
		}
	}
	return people
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil {
		log.Fatalf("%s: %v", key, err)
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		log.Fatalf("%s: %v", key, err)
	}
	return parsed
}
