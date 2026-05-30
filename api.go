package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// API response types
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// RegisterAPIHandlers sets up all API routes
func RegisterAPIHandlers(mux *http.ServeMux, state *AppState, statusCache *StatusCache, upsCache *UPSStatusCache) {
	// Device endpoints
	mux.HandleFunc("GET /api/devices", handleAPIGetDevices(state))
	mux.HandleFunc("POST /api/devices", handleAPIAddDevice(state))
	mux.HandleFunc("PUT /api/devices/{id}", handleAPIUpdateDevice(state))
	mux.HandleFunc("DELETE /api/devices/{id}", handleAPIDeleteDevice(state))
	mux.HandleFunc("POST /api/devices/{id}/wake", handleAPIWakeDevice(state))
	mux.HandleFunc("GET /api/devices/status", handleAPIDevicesStatus(statusCache))

	// UPS endpoints
	mux.HandleFunc("GET /api/ups", handleAPIGetUPS(state))
	mux.HandleFunc("POST /api/ups", handleAPIAddUPS(state, upsCache))
	mux.HandleFunc("PUT /api/ups/{id}", handleAPIUpdateUPS(state, upsCache))
	mux.HandleFunc("DELETE /api/ups/{id}", handleAPIDeleteUPS(state, upsCache))
	mux.HandleFunc("GET /api/ups/status", handleAPIUPSStatus(upsCache))

	// Config endpoints
	mux.HandleFunc("GET /api/config/export", handleAPIExport(state))
	mux.HandleFunc("POST /api/config/import", handleAPIImport(state))

	// Health check
	mux.HandleFunc("GET /api/health", handleAPIHealth())

	// Version
	mux.HandleFunc("GET /api/version", handleAPIVersion())
	mux.HandleFunc("GET /api/version/latest", handleAPILatestRelease(newLatestReleaseCache(time.Hour)))
}

// latestReleaseCache memoizes the GitHub /releases/latest response so that
// browser-side polls don't burn the unauthenticated GitHub rate limit
// (60/h/IP) — multiple tabs or NAT'd users would otherwise silently exhaust it.
//
// On a failed fetch it backs off for errTTL before contacting GitHub again and
// keeps serving the last good value (stale) in the meantime. Without this, a
// GitHub outage or a 403 rate-limit response would turn every poll into yet
// another GitHub call — amplifying the very rate-limiting the cache prevents.
type latestReleaseCache struct {
	mu        sync.RWMutex
	version   string
	url       string
	ok        bool          // a successful fetch has happened at least once
	fetchedAt time.Time     // time of last successful fetch
	lastErrAt time.Time     // time of last failed fetch
	ttl       time.Duration // freshness window for a successful value
	errTTL    time.Duration // backoff window after a failed fetch
}

func newLatestReleaseCache(ttl time.Duration) *latestReleaseCache {
	return &latestReleaseCache{ttl: ttl, errTTL: 5 * time.Minute}
}

type releaseSnapshot struct {
	version string
	url     string
	fresh   bool // a successful value within ttl — serve it directly
	stale   bool // an older value exists, worth serving while GitHub is down
	backoff bool // a fetch failed recently — don't contact GitHub yet
}

func (c *latestReleaseCache) snapshot() releaseSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s := releaseSnapshot{version: c.version, url: c.url}
	if c.ok {
		s.stale = true
		s.fresh = time.Since(c.fetchedAt) <= c.ttl
	}
	if !c.lastErrAt.IsZero() && time.Since(c.lastErrAt) < c.errTTL {
		s.backoff = true
	}
	return s
}

func (c *latestReleaseCache) setSuccess(version, url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.version, c.url, c.ok, c.fetchedAt = version, url, true, time.Now()
	c.lastErrAt = time.Time{} // a fresh success clears the backoff window
}

func (c *latestReleaseCache) setError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastErrAt = time.Now()
}

func handleAPILatestRelease(cache *latestReleaseCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap := cache.snapshot()
		if snap.fresh {
			writeSuccess(w, map[string]string{"version": snap.version, "url": snap.url})
			return
		}
		// A recent fetch failed: serve the stale value if we have one, otherwise
		// fail fast — either way, don't contact GitHub again until errTTL passes.
		if snap.backoff {
			if snap.stale {
				writeSuccess(w, map[string]string{"version": snap.version, "url": snap.url})
				return
			}
			writeError(w, http.StatusServiceUnavailable, "Latest release temporarily unavailable")
			return
		}

		// serveFailure records the failure (starting the backoff window) and
		// serves the stale value if available, else the given error.
		serveFailure := func(status int, msg string) {
			cache.setError()
			if snap.stale {
				writeSuccess(w, map[string]string{"version": snap.version, "url": snap.url})
				return
			}
			writeError(w, status, msg)
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		const url = "https://api.github.com/repos/aloks98/wolnut/releases/latest"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			// Building the request never touches GitHub, so this isn't a reason
			// to start the backoff window.
			writeError(w, http.StatusInternalServerError, "Failed to build request")
			return
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "wol-nut/"+version)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			serveFailure(http.StatusBadGateway, "Failed to reach GitHub: "+err.Error())
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			serveFailure(http.StatusBadGateway, fmt.Sprintf("GitHub responded with %d", resp.StatusCode))
			return
		}

		var rel struct {
			TagName string `json:"tag_name"`
			HTMLURL string `json:"html_url"`
			Name    string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			serveFailure(http.StatusBadGateway, "Failed to parse GitHub response")
			return
		}

		ver := strings.TrimPrefix(rel.TagName, "v")
		if ver == "" {
			ver = rel.Name
		}
		cache.setSuccess(ver, rel.HTMLURL)
		writeSuccess(w, map[string]string{"version": ver, "url": rel.HTMLURL})
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("writeJSON: encode error: %v", err)
	}
}

func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, APIResponse{Success: false, Error: message})
}

// Device handlers

func handleAPIGetDevices(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devices := state.GetDevices()
		writeSuccess(w, devices)
	}
}

func handleAPIAddDevice(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var device Device
		if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if device.Name == "" || device.MAC == "" {
			writeError(w, http.StatusBadRequest, "Name and MAC are required")
			return
		}

		device.MAC = NormalizeMAC(device.MAC)
		if !ValidateMAC(device.MAC) {
			writeError(w, http.StatusBadRequest, "Invalid MAC address format")
			return
		}

		device.ID = uuid.New().String()

		if err := state.AddDevice(device); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save device")
			return
		}

		writeSuccess(w, device)
	}
}

func handleAPIUpdateDevice(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "Device ID required")
			return
		}

		var device Device
		if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if device.Name == "" || device.MAC == "" {
			writeError(w, http.StatusBadRequest, "Name and MAC are required")
			return
		}

		device.ID = id
		device.MAC = NormalizeMAC(device.MAC)
		if !ValidateMAC(device.MAC) {
			writeError(w, http.StatusBadRequest, "Invalid MAC address format")
			return
		}

		if err := state.UpdateDevice(device); err != nil {
			if errors.Is(err, ErrNotFound) {
				writeError(w, http.StatusNotFound, "Device not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Failed to update device")
			return
		}

		writeSuccess(w, device)
	}
}

func handleAPIDeleteDevice(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "Device ID required")
			return
		}

		if err := state.DeleteDevice(id); err != nil {
			if errors.Is(err, ErrNotFound) {
				writeError(w, http.StatusNotFound, "Device not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Failed to delete device")
			return
		}

		writeSuccess(w, nil)
	}
}

func handleAPIWakeDevice(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "Device ID required")
			return
		}

		device, ok := state.GetDevice(id)
		if !ok {
			writeError(w, http.StatusNotFound, "Device not found")
			return
		}

		if err := SendWakeOnLAN(device.MAC, device.IP); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to send wake packet: "+err.Error())
			return
		}

		writeSuccess(w, map[string]string{"message": "Wake packet sent to " + device.Name})
	}
}

func handleAPIDevicesStatus(statusCache *StatusCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statuses := statusCache.GetDevicesStatus()
		writeSuccess(w, statuses)
	}
}

// UPS handlers

func handleAPIGetUPS(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upsList := state.GetUPSList()
		writeSuccess(w, upsList)
	}
}

func handleAPIAddUPS(state *AppState, cache *UPSStatusCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ups UPSEntry
		if err := json.NewDecoder(r.Body).Decode(&ups); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if ups.Name == "" || ups.Host == "" || ups.UPSName == "" {
			writeError(w, http.StatusBadRequest, "Name, Host, and UPSName are required")
			return
		}
		if !ValidateUPSHost(ups.Host) {
			writeError(w, http.StatusBadRequest, "Invalid Host: expected 'hostname' or 'hostname:port'")
			return
		}
		if !ValidateUPSName(ups.UPSName) {
			writeError(w, http.StatusBadRequest, "Invalid UPSName: alphanumerics, dot, dash, underscore only")
			return
		}

		ups.ID = uuid.New().String()

		if err := state.AddUPS(ups); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save UPS")
			return
		}

		go cache.Refresh()
		writeSuccess(w, ups)
	}
}

func handleAPIUpdateUPS(state *AppState, cache *UPSStatusCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "UPS ID required")
			return
		}

		var ups UPSEntry
		if err := json.NewDecoder(r.Body).Decode(&ups); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if ups.Name == "" || ups.Host == "" || ups.UPSName == "" {
			writeError(w, http.StatusBadRequest, "Name, Host, and UPSName are required")
			return
		}
		if !ValidateUPSHost(ups.Host) {
			writeError(w, http.StatusBadRequest, "Invalid Host: expected 'hostname' or 'hostname:port'")
			return
		}
		if !ValidateUPSName(ups.UPSName) {
			writeError(w, http.StatusBadRequest, "Invalid UPSName: alphanumerics, dot, dash, underscore only")
			return
		}

		ups.ID = id

		if err := state.UpdateUPS(ups); err != nil {
			if errors.Is(err, ErrNotFound) {
				writeError(w, http.StatusNotFound, "UPS not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Failed to update UPS")
			return
		}

		go cache.Refresh()
		writeSuccess(w, ups)
	}
}

func handleAPIDeleteUPS(state *AppState, cache *UPSStatusCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "UPS ID required")
			return
		}

		if err := state.DeleteUPS(id); err != nil {
			if errors.Is(err, ErrNotFound) {
				writeError(w, http.StatusNotFound, "UPS not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Failed to delete UPS")
			return
		}

		go cache.Refresh()
		writeSuccess(w, nil)
	}
}

func handleAPIUPSStatus(cache *UPSStatusCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, cache.Get())
	}
}

// Config handlers

func handleAPIExport(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := state.GetRawData()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to serialize data: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=wol-nut-backup.json")
		if _, err := w.Write(data); err != nil {
			log.Printf("export: write failed: %v", err)
		}
	}
}

const maxImportSize = 1 << 20 // 1 MiB — plenty for a JSON config

func handleAPIImport(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxImportSize)

		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "No file uploaded or file too large")
			return
		}
		defer file.Close()

		data, err := io.ReadAll(io.LimitReader(file, maxImportSize+1))
		if err != nil {
			writeError(w, http.StatusBadRequest, "Failed to read file")
			return
		}
		if int64(len(data)) > maxImportSize {
			writeError(w, http.StatusRequestEntityTooLarge, "File exceeds 1 MiB limit")
			return
		}

		if err := state.ImportData(data); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid backup file: "+err.Error())
			return
		}

		writeSuccess(w, map[string]string{"message": "Configuration imported successfully"})
	}
}

// Health check

func handleAPIHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// Version

func handleAPIVersion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, map[string]string{
			"version": version,
			"commit":  commit,
		})
	}
}
