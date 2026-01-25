package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// API response types
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// RegisterAPIHandlers sets up all API routes
func RegisterAPIHandlers(mux *http.ServeMux, state *AppState) {
	// Device endpoints
	mux.HandleFunc("GET /api/devices", handleAPIGetDevices(state))
	mux.HandleFunc("POST /api/devices", handleAPIAddDevice(state))
	mux.HandleFunc("PUT /api/devices/{id}", handleAPIUpdateDevice(state))
	mux.HandleFunc("DELETE /api/devices/{id}", handleAPIDeleteDevice(state))
	mux.HandleFunc("POST /api/devices/{id}/wake", handleAPIWakeDevice(state))
	mux.HandleFunc("GET /api/devices/status", handleAPIDevicesStatus(state))

	// UPS endpoints
	mux.HandleFunc("GET /api/ups", handleAPIGetUPS(state))
	mux.HandleFunc("POST /api/ups", handleAPIAddUPS(state))
	mux.HandleFunc("PUT /api/ups/{id}", handleAPIUpdateUPS(state))
	mux.HandleFunc("DELETE /api/ups/{id}", handleAPIDeleteUPS(state))
	mux.HandleFunc("GET /api/ups/status", handleAPIUPSStatus(state))

	// Config endpoints
	mux.HandleFunc("GET /api/config/export", handleAPIExport(state))
	mux.HandleFunc("POST /api/config/import", handleAPIImport(state))

	// Health check
	mux.HandleFunc("GET /api/health", handleAPIHealth())
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
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

		if err := SendWakeOnLAN(device.MAC); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to send wake packet: "+err.Error())
			return
		}

		writeSuccess(w, map[string]string{"message": "Wake packet sent to " + device.Name})
	}
}

func handleAPIDevicesStatus(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devices := state.GetDevices()
		statuses := GetDevicesStatus(devices)
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

func handleAPIAddUPS(state *AppState) http.HandlerFunc {
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

		ups.ID = uuid.New().String()

		if err := state.AddUPS(ups); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save UPS")
			return
		}

		writeSuccess(w, ups)
	}
}

func handleAPIUpdateUPS(state *AppState) http.HandlerFunc {
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

		ups.ID = id

		if err := state.UpdateUPS(ups); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update UPS")
			return
		}

		writeSuccess(w, ups)
	}
}

func handleAPIDeleteUPS(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "UPS ID required")
			return
		}

		if err := state.DeleteUPS(id); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to delete UPS")
			return
		}

		writeSuccess(w, nil)
	}
}

func handleAPIUPSStatus(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upsList := state.GetUPSList()
		statuses := QueryAllUPS(upsList)
		writeSuccess(w, statuses)
	}
}

// Config handlers

func handleAPIExport(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := state.GetRawData()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=wol-nut-backup.json")
		w.Write(data)
	}
}

func handleAPIImport(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "No file uploaded")
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Failed to read file")
			return
		}

		if err := state.ImportData(data); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid backup file")
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
