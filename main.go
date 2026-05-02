package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

//go:embed web/build/*
var frontendFS embed.FS

var (
	version = "dev"
	commit  = "none"
)

func main() {
	configPath := flag.String("config", "", "Path to config.yml")
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("wol-nut %s (%s)\n", version, commit)
		os.Exit(0)
	}

	// Load config
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize app state
	state := &AppState{Config: cfg}
	if err := state.LoadData(); err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}

	// Initialize status cache for device online status
	statusCache := NewStatusCache(state)
	defer statusCache.Stop()

	// Setup routes
	mux := http.NewServeMux()

	// Register API handlers
	RegisterAPIHandlers(mux, state, statusCache)

	// Serve frontend
	frontendContent, err := fs.Sub(frontendFS, "web/build")
	if err != nil {
		log.Fatalf("Failed to load frontend: %v", err)
	}
	fileServer := http.FileServer(http.FS(frontendContent))

	// SPA fallback handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Don't serve frontend for API routes
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		if _, err := fs.Stat(frontendContent, strings.TrimPrefix(path, "/")); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for all other routes
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	// Server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{Addr: addr, Handler: corsMiddleware(mux)}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		server.Close()
	}()

	log.Printf("WoL-NUT %s starting on http://%s", version, addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// corsMiddleware adds CORS headers when WOLNUT_CORS_ORIGIN is set.
// Default (unset) sends no CORS header — same-origin only, which is correct
// for production when the SPA and API ship from the same binary.
// Set WOLNUT_CORS_ORIGIN=* during local frontend dev (vite on a different port).
func corsMiddleware(next http.Handler) http.Handler {
	origin := os.Getenv("WOLNUT_CORS_ORIGIN")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
