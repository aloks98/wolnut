package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

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

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(readTimeout))

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

// QueryAllUPS queries all configured UPS units
func QueryAllUPS(upsList []UPSEntry) []UPSStatus {
	statuses := make([]UPSStatus, len(upsList))

	for i, ups := range upsList {
		status, _ := QueryUPS(ups.Host, ups.UPSName)
		status.ID = ups.ID
		status.Name = ups.Name
		statuses[i] = status
	}

	return statuses
}

// FormatRuntime formats runtime seconds into human-readable format
func FormatRuntime(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
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
