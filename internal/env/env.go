// Package env reads a service's settings from environment variables, which is
// how Docker Compose passes them in. Every setting has a fallback, so a
// service also runs outside Compose. See project_notes.md §6.
package env

import (
	"log"
	"os"
	"strconv"
	"time"
)

// String is the value of key, or fallback when key is unset or empty.
func String(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Float is key read as a number. A value that isn't one stops the service:
// starting with the fallback instead would hide the mistake.
func Float(key string, fallback float64) float64 {
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

// Duration is key read as a length of time, written as Go does it, e.g. "10s".
func Duration(key string, fallback time.Duration) time.Duration {
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
