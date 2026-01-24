package main

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"

	"github.com/google/uuid"
)

func RegisterHandlers(mux *http.ServeMux, state *AppState, tmpl *template.Template) {
	// Pages
	mux.HandleFunc("GET /", handleIndex(state, tmpl))
	mux.HandleFunc("GET /devices", handleDevices(state, tmpl))
	mux.HandleFunc("GET /ups", handleUPS(state, tmpl))

	// Device actions
	mux.HandleFunc("POST /devices/add", handleAddDevice(state, tmpl))
	mux.HandleFunc("DELETE /devices/{id}", handleDeleteDevice(state, tmpl))

	// UPS actions
	mux.HandleFunc("POST /ups/add", handleAddUPS(state, tmpl))
	mux.HandleFunc("DELETE /ups/{id}", handleDeleteUPS(state, tmpl))

	// WoL
	mux.HandleFunc("POST /wake/{id}", handleWake(state, tmpl))

	// API
	mux.HandleFunc("GET /api/ups/status", handleUPSStatusAPI(state))

	// Partials
	mux.HandleFunc("GET /partials/ups-cards", handleUPSCards(state, tmpl))
	mux.HandleFunc("GET /partials/device-list", handleDeviceList(state, tmpl))

	// Config export/import
	mux.HandleFunc("GET /config/export", handleExport(state))
	mux.HandleFunc("POST /config/import", handleImport(state, tmpl))

	// Health
	mux.HandleFunc("GET /health", handleHealth())
}

type PageData struct {
	Title       string
	Page        string
	Devices     []Device
	UPSList     []UPSEntry
	UPSStatuses []UPSStatus
}

type ToastData struct {
	Type    string
	Message string
}

func handleIndex(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		devices := state.GetDevices()
		upsList := state.GetUPSList()
		statuses := QueryAllUPS(upsList)

		data := PageData{
			Title:       "Dashboard",
			Page:        "dashboard",
			Devices:     devices,
			UPSStatuses: statuses,
		}

		tmpl.ExecuteTemplate(w, "index", data)
	}
}

func handleDevices(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:   "Devices",
			Page:    "devices",
			Devices: state.GetDevices(),
		}
		tmpl.ExecuteTemplate(w, "devices", data)
	}
}

func handleUPS(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:   "UPS",
			Page:    "ups",
			UPSList: state.GetUPSList(),
		}
		tmpl.ExecuteTemplate(w, "ups", data)
	}
}

func handleAddDevice(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")
		mac := r.FormValue("mac")

		if name == "" || mac == "" {
			http.Error(w, "Name and MAC are required", http.StatusBadRequest)
			return
		}

		mac = NormalizeMAC(mac)
		if !ValidateMAC(mac) {
			http.Error(w, "Invalid MAC address format", http.StatusBadRequest)
			return
		}

		device := Device{
			ID:   uuid.New().String(),
			Name: name,
			MAC:  mac,
		}

		if err := state.AddDevice(device); err != nil {
			http.Error(w, "Failed to save device", http.StatusInternalServerError)
			return
		}

		// Return updated device list partial
		tmpl.ExecuteTemplate(w, "device_list_manage", state.GetDevices())
	}
}

func handleDeleteDevice(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "Device ID required", http.StatusBadRequest)
			return
		}

		if err := state.DeleteDevice(id); err != nil {
			http.Error(w, "Failed to delete device", http.StatusInternalServerError)
			return
		}

		// Return updated device list partial
		tmpl.ExecuteTemplate(w, "device_list_manage", state.GetDevices())
	}
}

func handleAddUPS(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")
		host := r.FormValue("host")
		upsName := r.FormValue("ups_name")

		if name == "" || host == "" || upsName == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		ups := UPSEntry{
			ID:      uuid.New().String(),
			Name:    name,
			Host:    host,
			UPSName: upsName,
		}

		if err := state.AddUPS(ups); err != nil {
			http.Error(w, "Failed to save UPS", http.StatusInternalServerError)
			return
		}

		// Return updated UPS list partial
		tmpl.ExecuteTemplate(w, "ups_list_manage", state.GetUPSList())
	}
}

func handleDeleteUPS(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "UPS ID required", http.StatusBadRequest)
			return
		}

		if err := state.DeleteUPS(id); err != nil {
			http.Error(w, "Failed to delete UPS", http.StatusInternalServerError)
			return
		}

		// Return updated UPS list partial
		tmpl.ExecuteTemplate(w, "ups_list_manage", state.GetUPSList())
	}
}

func handleWake(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "Device ID required", http.StatusBadRequest)
			return
		}

		device, ok := state.GetDevice(id)
		if !ok {
			tmpl.ExecuteTemplate(w, "toast", ToastData{
				Type:    "error",
				Message: "Device not found",
			})
			return
		}

		if err := SendWakeOnLAN(device.MAC); err != nil {
			tmpl.ExecuteTemplate(w, "toast", ToastData{
				Type:    "error",
				Message: "Failed to send wake packet: " + err.Error(),
			})
			return
		}

		tmpl.ExecuteTemplate(w, "toast", ToastData{
			Type:    "success",
			Message: "Wake packet sent to " + device.Name,
		})
	}
}

func handleUPSStatusAPI(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upsList := state.GetUPSList()
		statuses := QueryAllUPS(upsList)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statuses)
	}
}

func handleUPSCards(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upsList := state.GetUPSList()
		statuses := QueryAllUPS(upsList)
		tmpl.ExecuteTemplate(w, "ups_cards", statuses)
	}
}

func handleDeviceList(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "device_list", state.GetDevices())
	}
}

func handleExport(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := state.GetRawData()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=wol-nut-backup.json")
		w.Write(data)
	}
}

func handleImport(state *AppState, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file uploaded", http.StatusBadRequest)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusBadRequest)
			return
		}

		if err := state.ImportData(data); err != nil {
			http.Error(w, "Invalid backup file", http.StatusBadRequest)
			return
		}

		// Redirect to home
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}
