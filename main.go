package main

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Everything under web/static (htmx, fonts, CSS, favicon) is compiled into the binary.
//
//go:embed web/static
var webFS embed.FS

var startedAt = time.Now()

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	static, err := fs.Sub(webFS, "web/static")
	if err != nil {
		log.Fatal(err)
	}

	// Rendered once at startup, served from memory.
	launchLight := renderLaunch(launchPaletteLight)
	launchDark := renderLaunch(launchPaletteDark)

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheFor(time.Hour, http.FileServerFS(static))))
	mux.Handle("GET /favicon.ico", http.FileServerFS(static))
	mux.HandleFunc("GET /launch-light.gif", serveGIF(launchLight))
	mux.HandleFunc("GET /launch-dark.gif", serveGIF(launchDark))
	mux.HandleFunc("GET /fragments/facts", func(w http.ResponseWriter, r *http.Request) {
		render(w, serverFacts())
	})
	mux.HandleFunc("/api/cron", cron)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		render(w, homePage())
	})

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func cacheFor(d time.Duration, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(d.Seconds())))
		h.ServeHTTP(w, r)
	})
}

func serveGIF(b []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(b)
	}
}

// cron is called by DollarDeploy cron jobs: https://docs.dollardeploy.com/cron/
// DollarDeploy generates CRON_SECRET on the first HTTP cron job; redeploy once to load it.
// Runs are capped at 60s, so queue long work and return quickly.
func cron(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	secret := os.Getenv("CRON_SECRET")
	if secret == "" {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	given := []byte(r.Header.Get("Authorization"))
	if subtle.ConstantTimeCompare(given, []byte("Bearer "+secret)) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// ... do the scheduled work here ...

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"cronId": r.Header.Get("X-Dollardeploy-Cron-Id"),
		"ranAt":  time.Now().UTC().Format(time.RFC3339),
	})
}
