// occupancy writes the people in the target room to BuildSim, following a
// time-of-day schedule. It must stay the only process that writes occupancy:
// BuildSim replaces the whole building's occupancy on every write.
// Reasoning: project_notes.md §4.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	// The image carries no timezone database, so TZ is otherwise ignored.
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

	client := buildsim.New(baseURL) // this program's link to BuildSim
	ctx := context.Background()     // empty context, no cancellation or timeout

	// Read the room's area from BuildSim; it sets how many people fit.
	area, err := client.RoomArea(ctx, level, room)
	if err != nil {
		log.Fatalf("read area of %s: %v", buildsim.RoomKey(level, room), err)
	}
	capacity := int(math.Round(area / areaPerPerson))
	log.Printf("room %s is %.1f m², holding up to %d people",
		buildsim.RoomKey(level, room), area, capacity)

	people := namePeople(room, capacity)

	ticker := time.NewTicker(interval)
	for {
		// Room time is the wall clock: the simulation runs at real speed.
		present := occupancy.PeopleAt(time.Now(), capacity)
		if err := publish(ctx, client, level, room, people[:present]); err != nil {
			// Skip this cycle; BuildSim keeps the occupancy it already has.
			log.Printf("write occupancy: %v", err)
		}
		<-ticker.C // wait for the next tick
	}
}

// publish sets who is in the room. Rooms left out of the payload are emptied
// by BuildSim, which is correct while the project simulates one room.
func publish(
	ctx context.Context,
	client *buildsim.Client,
	level, room string,
	present []buildsim.Person,
) error {
	occupants := buildsim.RoomOccupancy{
		Persons: present,
		Aliens:  []buildsim.Alien{},
	}
	building := map[string]buildsim.RoomOccupancy{
		buildsim.RoomKey(level, room): occupants,
	}
	return client.SetOccupancy(ctx, building)
}

// namePeople builds the room's full set of occupants once, so a person keeps
// the same identity as the count changes.
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
