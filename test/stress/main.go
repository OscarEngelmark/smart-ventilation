// Load program of the stress test (project_notes.md §10): publishes CO2
// readings for one made-up room at rising rates, spread evenly over each second,
// and counts how many the storage-service stored after each step. Run by
// test/stress/stress.sh.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/OscarEngelmark/smart-ventilation/internal/message"
	"github.com/OscarEngelmark/smart-ventilation/internal/mqttclient"
)

// room is the made-up room every reading is for.
const room = "stress/R000"

// load publishes the room's readings, each 10 s apart in time.
type load struct {
	client mqtt.Client
	next   time.Time // the next reading's time
}

func main() {
	rates := flag.String("rates", "10,50,100,125,150,200,300,500,1000", "readings per second, one step each")
	step := flag.Duration("step", 30*time.Second, "real time each rate is held")
	wait := flag.Duration("wait", 60*time.Second, "real time waited after a step before counting")
	broker := flag.String("broker", "tcp://localhost:1883", "the broker's address")
	storage := flag.String("storage", "http://localhost:8081", "the storage-service's address")
	start := flag.String("start", "", "the first reading's time, RFC 3339")
	flag.Parse()

	t0, err := time.Parse(time.RFC3339, *start)
	if err != nil {
		log.Fatalf("start: %v", err)
	}
	client, err := mqttclient.Connect(*broker, "stress-load", nil)
	if err != nil {
		log.Fatalf("connect to broker: %v", err)
	}
	l := &load{client: client, next: t0}

	fmt.Println("rate_per_s sent stored lost")
	storedBefore := 0
	for _, field := range strings.Split(*rates, ",") {
		rate, err := strconv.Atoi(field)
		if err != nil {
			log.Fatalf("rate %q: %v", field, err)
		}
		sent := l.publishFor(rate, *step)
		time.Sleep(*wait)
		stored := countStored(*storage, t0)
		inStep := stored - storedBefore
		fmt.Printf("%d %d %d %d\n", rate, sent, inStep, sent-inStep)
		storedBefore = stored
	}
}

// publishFor publishes rate readings per second for duration d of real time,
// spread evenly, waits until the broker has taken them all, and returns how many it sent.
func (l *load) publishFor(rate int, d time.Duration) int {
	began := time.Now()
	sent := 0
	var tokens []mqtt.Token // one per publish, done once the broker has taken it
	for time.Since(began) < d {
		due := int(time.Since(began).Seconds() * float64(rate))
		for ; sent < due; sent++ {
			tokens = append(tokens, l.publishNext())
		}
		time.Sleep(time.Millisecond)
	}
	for _, token := range tokens {
		token.Wait()
		if token.Error() != nil {
			log.Fatalf("publish: %v", token.Error())
		}
	}
	return sent
}

// publishNext publishes the room's next reading at QoS 1 without waiting for the broker.
func (l *load) publishNext() mqtt.Token {
	reading := message.CO2Reading{RoomID: room, PPM: 600, Ts: l.next}
	l.next = l.next.Add(10 * time.Second)
	payload, err := json.Marshal(reading)
	if err != nil {
		log.Fatalf("encode: %v", err)
	}
	return l.client.Publish(room+"/co2/reading", 1, false, payload)
}

// countStored returns how many readings the storage-service holds for the room from t0 on.
func countStored(storage string, t0 time.Time) int {
	query := url.Values{"room": {room}, "since": {t0.Format(time.RFC3339)}}
	resp, err := http.Get(storage + "/co2?" + query.Encode())
	if err != nil {
		log.Fatalf("count stored: %v", err)
	}
	defer resp.Body.Close()    // closes the response once countStored returns
	var rows []json.RawMessage // each reading left undecoded, since only the count matters
	err = json.NewDecoder(resp.Body).Decode(&rows)
	if err != nil {
		log.Fatalf("count stored: %v", err)
	}
	return len(rows)
}
