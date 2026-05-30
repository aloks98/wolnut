package main

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// upsHostRegex matches "host" or "host:port".  Reject URL schemes, paths, whitespace —
// the value is fed directly to net.DialTimeout, so a malformed value would let a
// caller probe arbitrary internal services (SSRF).
var upsHostRegex = regexp.MustCompile(`^([A-Za-z0-9._\-]+)(?::(\d{1,5}))?$`)

// upsNameRegex limits UPS names to NUT-safe identifiers.  A newline or space
// here would let a caller inject a second NUT protocol command into the same TCP session.
var upsNameRegex = regexp.MustCompile(`^[A-Za-z0-9._\-]+$`)

// ValidateUPSHost reports whether host is acceptable for net.DialTimeout.
// Accepts "hostname" or "hostname:port"; port must be 1-65535 if provided.
func ValidateUPSHost(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	m := upsHostRegex.FindStringSubmatch(host)
	if m == nil {
		return false
	}
	if m[2] != "" {
		port, err := strconv.Atoi(m[2])
		if err != nil || port < 1 || port > 65535 {
			return false
		}
	}
	return true
}

// ValidateUPSName reports whether name is safe to interpolate into a NUT command.
func ValidateUPSName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	return upsNameRegex.MatchString(name)
}

type UPSStatus struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Host string `json:"host"`

	// Connection status
	Online bool   `json:"online"`
	Error  string `json:"error,omitempty"`

	// UPS status
	Status        string `json:"status"`         // Raw status (OL, OB, LB, CHRG, etc.)
	StatusLabel   string `json:"status_label"`   // Human-readable status
	IsOnline      bool   `json:"is_online"`      // On line power
	IsOnBattery   bool   `json:"is_on_battery"`  // Running on battery
	IsLowBattery  bool   `json:"is_low_battery"`
	IsCharging    bool   `json:"is_charging"`
	IsDischarging bool   `json:"is_discharging"`

	// Battery
	BatteryCharge  int     `json:"battery_charge"`  // Percentage
	BatteryVoltage float64 `json:"battery_voltage"` // Volts
	BatteryRuntime int     `json:"battery_runtime"` // Seconds

	// Load
	Load    int     `json:"load"`     // Percentage
	Power   float64 `json:"power"`    // Watts (current draw)
	Nominal float64 `json:"nominal"`  // Nominal power in watts
	Current float64 `json:"current"`  // Output current in amps

	// Input
	InputVoltage   float64 `json:"input_voltage"`   // Volts
	InputFrequency float64 `json:"input_frequency"` // Hz

	// Output
	OutputVoltage   float64 `json:"output_voltage"`   // Volts
	OutputFrequency float64 `json:"output_frequency"` // Hz

	// UPS Info
	Model        string `json:"model"`
	Manufacturer string `json:"manufacturer"`
	Serial       string `json:"serial"`
	Firmware     string `json:"firmware"`

	// Temperature
	Temperature float64 `json:"temperature"` // Celsius

	// Calculated
	EstimatedWattage float64 `json:"estimated_wattage"` // Calculated from load * nominal
}

const (
	connectTimeout = 3 * time.Second
	readTimeout    = 5 * time.Second
)

// QueryUPS connects to a NUT server and queries UPS status
func QueryUPS(host, upsName string) (UPSStatus, error) {
	status := UPSStatus{
		Host: host,
	}

	// Add default port if not specified
	if !strings.Contains(host, ":") {
		host = host + ":3493"
	}

	// Connect with timeout
	conn, err := net.DialTimeout("tcp", host, connectTimeout)
	if err != nil {
		status.Error = fmt.Sprintf("Connection failed: %v", err)
		return status, err
	}
	defer conn.Close()

	// Bound both reads and writes. SetDeadline covers Write too — without
	// this, a half-open NUT daemon (TCP handshake completed, upsd hung)
	// could block conn.Write forever.
	conn.SetDeadline(time.Now().Add(readTimeout))

	// Send LIST VAR command
	cmd := fmt.Sprintf("LIST VAR %s\n", upsName)
	_, err = conn.Write([]byte(cmd))
	if err != nil {
		status.Error = fmt.Sprintf("Failed to send command: %v", err)
		return status, err
	}

	// Read response
	vars := make(map[string]string)
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 1<<20) // up to 1 MiB per token

	for scanner.Scan() {
		line := scanner.Text()

		// Check for end of list
		if strings.HasPrefix(line, "END LIST VAR") {
			break
		}

		// Check for error
		if strings.HasPrefix(line, "ERR") {
			status.Error = line
			return status, fmt.Errorf("NUT error: %s", line)
		}

		// Parse VAR lines: VAR <upsname> <varname> "<value>"
		if strings.HasPrefix(line, "VAR ") {
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 4 {
				varName := parts[2]
				// Remove quotes from value
				value := strings.Trim(parts[3], "\"")
				vars[varName] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		status.Error = fmt.Sprintf("Read error: %v", err)
		return status, err
	}

	// Mark as online since we got a response
	status.Online = true

	// Parse UPS status
	if s, ok := vars["ups.status"]; ok {
		status.Status = s
		status.StatusLabel = GetStatusLabel(s)
		status.IsOnline = strings.Contains(s, "OL")
		status.IsOnBattery = strings.Contains(s, "OB")
		status.IsLowBattery = strings.Contains(s, "LB")
		status.IsDischarging = strings.Contains(s, "DISCHRG")
		// CHRG but not DISCHRG
		status.IsCharging = strings.Contains(s, "CHRG") && !status.IsDischarging
	}

	// Battery charge
	if s, ok := vars["battery.charge"]; ok {
		if v, err := strconv.Atoi(s); err == nil {
			status.BatteryCharge = v
		}
	}

	// Battery voltage
	if s, ok := vars["battery.voltage"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.BatteryVoltage = v
		}
	}

	// Battery runtime (in seconds)
	if s, ok := vars["battery.runtime"]; ok {
		if v, err := strconv.Atoi(s); err == nil {
			status.BatteryRuntime = v
		}
	}

	// Load percentage
	if s, ok := vars["ups.load"]; ok {
		if v, err := strconv.Atoi(s); err == nil {
			status.Load = v
		}
	}

	// Real power (watts)
	if s, ok := vars["ups.realpower"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.Power = v
		}
	} else if s, ok := vars["ups.power"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.Power = v
		}
	}

	// Nominal power
	if s, ok := vars["ups.realpower.nominal"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.Nominal = v
		}
	} else if s, ok := vars["ups.power.nominal"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.Nominal = v
		}
	}

	// Output current
	if s, ok := vars["output.current"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.Current = v
		}
	}

	// Input voltage
	if s, ok := vars["input.voltage"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.InputVoltage = v
		}
	}

	// Input frequency
	if s, ok := vars["input.frequency"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			// Some UPS report frequency in tenths of Hz (e.g., 498 instead of 49.8)
			if v > 100 {
				v = v / 10
			}
			status.InputFrequency = v
		}
	}

	// Output voltage
	if s, ok := vars["output.voltage"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.OutputVoltage = v
		}
	}

	// Output frequency
	if s, ok := vars["output.frequency"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			// Some UPS report frequency in tenths of Hz (e.g., 498 instead of 49.8)
			if v > 100 {
				v = v / 10
			}
			status.OutputFrequency = v
		}
	}

	// UPS info
	if s, ok := vars["ups.model"]; ok {
		status.Model = s
	} else if s, ok := vars["device.model"]; ok {
		status.Model = s
	}

	if s, ok := vars["ups.mfr"]; ok {
		status.Manufacturer = s
	} else if s, ok := vars["device.mfr"]; ok {
		status.Manufacturer = s
	}

	if s, ok := vars["ups.serial"]; ok {
		status.Serial = s
	} else if s, ok := vars["device.serial"]; ok {
		status.Serial = s
	}

	if s, ok := vars["ups.firmware"]; ok {
		status.Firmware = s
	}

	// Temperature
	if s, ok := vars["ups.temperature"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.Temperature = v
		}
	}

	// Calculate estimated wattage if we have load and nominal
	if status.Nominal > 0 && status.Load > 0 {
		status.EstimatedWattage = status.Nominal * float64(status.Load) / 100
	} else if status.Power > 0 {
		status.EstimatedWattage = status.Power
	}

	return status, nil
}

// upsRefreshInterval is how often the cache re-queries every UPS.  At 15s the
// dashboard feels live without hammering NUT or making /api/ups/status block
// on a fan-out of TCP dials per request.
const upsRefreshInterval = 15 * time.Second

// UPSStatusCache caches UPS status with periodic background refresh, so HTTP
// handlers don't dial NUT (potentially blocking up to connectTimeout+readTimeout
// per UPS) on every request.
type UPSStatusCache struct {
	mu       sync.RWMutex
	statuses []UPSStatus
	state    *AppState
	stopCh   chan struct{}
}

// NewUPSStatusCache starts a background goroutine that refreshes UPS status
// every upsRefreshInterval.  Cancel via Stop().
func NewUPSStatusCache(state *AppState) *UPSStatusCache {
	c := &UPSStatusCache{
		state:  state,
		stopCh: make(chan struct{}),
	}
	go c.run()
	return c
}

func (c *UPSStatusCache) run() {
	c.refresh()
	ticker := time.NewTicker(upsRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.refresh()
		case <-c.stopCh:
			return
		}
	}
}

func (c *UPSStatusCache) refresh() {
	statuses := QueryAllUPS(c.state.GetUPSList())
	c.mu.Lock()
	c.statuses = statuses
	c.mu.Unlock()
}

// Get returns a snapshot of the most recent UPS statuses.  Safe to call from
// any goroutine; returns an empty slice until the first refresh completes.
func (c *UPSStatusCache) Get() []UPSStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]UPSStatus, len(c.statuses))
	copy(out, c.statuses)
	return out
}

// Refresh forces an immediate refresh.  Useful after a UPS list mutation so the
// dashboard reflects the new entry without waiting for the next tick.
func (c *UPSStatusCache) Refresh() {
	c.refresh()
}

// Stop terminates the background refresh goroutine.
func (c *UPSStatusCache) Stop() {
	close(c.stopCh)
}

// QueryAllUPS queries all configured UPS units in parallel.
// Total wall time is bounded by the slowest single UPS (connectTimeout+readTimeout)
// instead of the sum across all UPSes.
func QueryAllUPS(upsList []UPSEntry) []UPSStatus {
	statuses := make([]UPSStatus, len(upsList))
	var wg sync.WaitGroup

	for i, ups := range upsList {
		wg.Add(1)
		go func(idx int, u UPSEntry) {
			defer wg.Done()
			status, _ := QueryUPS(u.Host, u.UPSName)
			status.ID = u.ID
			status.Name = u.Name
			statuses[idx] = status
		}(i, ups)
	}

	wg.Wait()
	return statuses
}

// GetStatusLabel returns a human-readable status label
func GetStatusLabel(status string) string {
	var parts []string

	if strings.Contains(status, "OL") {
		parts = append(parts, "Online")
	}
	if strings.Contains(status, "OB") {
		parts = append(parts, "On Battery")
	}
	if strings.Contains(status, "LB") {
		parts = append(parts, "Low Battery")
	}
	// Check DISCHRG before CHRG since DISCHRG contains CHRG
	if strings.Contains(status, "DISCHRG") {
		parts = append(parts, "Discharging")
	} else if strings.Contains(status, "CHRG") {
		parts = append(parts, "Charging")
	}
	if strings.Contains(status, "BYPASS") {
		parts = append(parts, "Bypass")
	}
	if strings.Contains(status, "CAL") {
		parts = append(parts, "Calibrating")
	}
	if strings.Contains(status, "OFF") {
		parts = append(parts, "Off")
	}
	if strings.Contains(status, "OVER") {
		parts = append(parts, "Overloaded")
	}
	if strings.Contains(status, "TRIM") {
		parts = append(parts, "Trimming")
	}
	if strings.Contains(status, "BOOST") {
		parts = append(parts, "Boosting")
	}

	if len(parts) == 0 {
		if status == "" {
			return "Unknown"
		}
		return status
	}

	return strings.Join(parts, ", ")
}
