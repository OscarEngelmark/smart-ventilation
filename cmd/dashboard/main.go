// dashboard serves a web page showing one room's CO2, head count and damper
// level over recent room time, read from the storage-service. It passes the
// page's /api/ requests on to the storage-service unchanged.
// Reasoning: project_notes.md §12.
package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/env"
	"github.com/OscarEngelmark/smart-ventilation/internal/roomtime"
	"github.com/OscarEngelmark/smart-ventilation/internal/shutdown"
)

//go:embed static
var static embed.FS // the page and its chart library, built into the program

// pageConfig is what the page needs to know before it asks for any data.
type pageConfig struct {
	Room          string  `json:"room"`
	Speed         float64 `json:"speed"` // room time per real time, as start.sh set it
	WindowSeconds float64 `json:"window_seconds"`
	TargetPPM     float64 `json:"target_ppm"`
	ThresholdPPM  float64 `json:"threshold_ppm"`
}

func main() {
	shutdown.ExitZeroOnStop()

	storageURL := env.String("STORAGE_URL")
	addr := ":8082" // the port this program listens on inside its container
	level := env.String("ROOM_LEVEL")
	roomName := env.String("ROOM_NAME")
	config := pageConfig{
		Room:          buildsim.RoomKey(level, roomName),
		Speed:         roomtime.FromEnv().Speed(),
		WindowSeconds: env.Duration("DASHBOARD_WINDOW").Seconds(),
		TargetPPM:     env.Float("PLAN_TARGET_PPM"),
		ThresholdPPM:  env.Float("CO2_THRESHOLD_PPM"),
	}

	storage, err := url.Parse(storageURL)
	if err != nil {
		log.Fatalf("STORAGE_URL: %v", err)
	}
	page, err := fs.Sub(static, "static") // the static folder's files, served from /
	if err != nil {
		log.Fatalf("page files: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/", http.StripPrefix("/api", httputil.NewSingleHostReverseProxy(storage)))
	mux.HandleFunc("GET /config", serveConfig(config))
	mux.Handle("GET /", http.FileServerFS(page))
	log.Printf("serving the dashboard for %s on %s, reading from %s", config.Room, addr, storageURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// serveConfig answers GET /config with the page's settings as JSON.
func serveConfig(config pageConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(config); err != nil {
			log.Printf("config: write response: %v", err)
		}
	}
}
