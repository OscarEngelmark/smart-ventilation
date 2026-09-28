// Package shutdown makes a service end with exit code 0 when Docker stops it.
// Reasoning: project_notes.md §6.
package shutdown

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// ExitZeroOnStop ends the program with exit code 0 when it receives SIGTERM,
// the signal Docker sends to stop a container. Without it, a Go program that is
// its container's main process exits with code 2, which Docker counts as a
// crash. Nothing is cleaned up first. Call it once, at the start of main.
func ExitZeroOnStop() {
	stop := make(chan os.Signal, 1) // receives SIGTERM instead of the program dying
	signal.Notify(stop, syscall.SIGTERM)
	go func() {
		<-stop // wait until the signal arrives
		log.Printf("stopped, exiting")
		os.Exit(0)
	}()
}
