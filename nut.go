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
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Host          string  `json:"host"`
	Online        bool    `json:"online"`
	Status        string  `json:"status"`
	BatteryCharge int     `json:"battery_charge"`
	Runtime       int     `json:"runtime"`
	Load          int     `json:"load"`
	InputVoltage  float64 `json:"input_voltage"`
	Error         string  `json:"error,omitempty"`
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

	// Parse status
	if s, ok := vars["ups.status"]; ok {
		status.Status = s
	}

	// Parse battery charge
	if s, ok := vars["battery.charge"]; ok {
		if v, err := strconv.Atoi(s); err == nil {
			status.BatteryCharge = v
		}
	}

	// Parse runtime (in seconds)
	if s, ok := vars["battery.runtime"]; ok {
		if v, err := strconv.Atoi(s); err == nil {
			status.Runtime = v
		}
	}

	// Parse load
	if s, ok := vars["ups.load"]; ok {
		if v, err := strconv.Atoi(s); err == nil {
			status.Load = v
		}
	}

	// Parse input voltage
	if s, ok := vars["input.voltage"]; ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			status.InputVoltage = v
		}
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
	switch {
	case strings.Contains(status, "OL"):
		return "Online"
	case strings.Contains(status, "OB"):
		return "On Battery"
	case strings.Contains(status, "LB"):
		return "Low Battery"
	case strings.Contains(status, "CHRG"):
		return "Charging"
	default:
		if status == "" {
			return "Unknown"
		}
		return status
	}
}

// GetStatusClass returns CSS class for status badge
func GetStatusClass(status string) string {
	switch {
	case strings.Contains(status, "OL"):
		return "bg-emerald-900 text-emerald-400"
	case strings.Contains(status, "OB"):
		return "bg-amber-900 text-amber-400"
	case strings.Contains(status, "LB"):
		return "bg-red-900 text-red-400"
	default:
		return "bg-gray-700 text-gray-400"
	}
}

// GetBatteryColor returns the color for battery charge visualization
func GetBatteryColor(charge int) string {
	if charge > 50 {
		return "#10b981" // emerald
	} else if charge > 20 {
		return "#f59e0b" // amber
	}
	return "#ef4444" // red
}
