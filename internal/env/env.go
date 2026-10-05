// Package env reads a service's settings from environment variables, which is
// how Docker Compose passes them in from sim.env and docker-compose.yml. Every
// setting is required: one that is missing or unreadable stops the service at
// startup. See project_notes.md §6.
package env

import (
	"log"
	"os"
	"strconv"
	"time"
)

// String is the value of key. An unset or empty key stops the service.
func String(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s: not set", key)
	}
	return v
}

// Float is key read as a number. A value that isn't one stops the service.
func Float(key string) float64 {
	parsed, err := strconv.ParseFloat(String(key), 64)
	if err != nil {
		log.Fatalf("%s: %v", key, err)
	}
	return parsed
}

// Duration is key read as a length of time, written as Go does it, e.g. "10s".
func Duration(key string) time.Duration {
	parsed, err := time.ParseDuration(String(key))
	if err != nil {
		log.Fatalf("%s: %v", key, err)
	}
	return parsed
}
