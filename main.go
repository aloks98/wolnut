package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

//go:embed templates/*
var templateFS embed.FS

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

	// Parse templates with custom functions
	funcMap := template.FuncMap{
		"formatRuntime": FormatRuntime,
	}

	tmpl := template.Must(
		template.New("").Funcs(funcMap).ParseFS(templateFS,
			"templates/*.html",
			"templates/partials/*.html",
		),
	)

	// Setup routes
	mux := http.NewServeMux()
	RegisterHandlers(mux, state, tmpl)

	// Server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{Addr: addr, Handler: mux}

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
