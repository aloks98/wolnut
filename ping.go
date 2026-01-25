package main

import (
	"net"
	"sync"
	"time"
)

// DeviceStatus represents a device with its online status
type DeviceStatus struct {
	Device
	Online *bool `json:"online"` // nil = no IP configured, true = online, false = offline
}

const pingTimeout = 2 * time.Second

// CheckDeviceOnline checks if a device is reachable via TCP ping
// We use TCP port 80, 443, or 22 as fallback since ICMP requires root
func CheckDeviceOnline(ip string) bool {
	if ip == "" {
		return false
	}

	// Try common ports
	ports := []string{"80", "443", "22", "3389", "445"}

	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), pingTimeout)
		if err == nil {
			conn.Close()
			return true
		}
	}

	// Also try ICMP-like check via UDP (will fail fast if unreachable)
	conn, err := net.DialTimeout("udp", net.JoinHostPort(ip, "33434"), pingTimeout)
	if err == nil {
		conn.Close()
		// UDP "connected" doesn't mean host is up, but we tried
	}

	return false
}

// GetDevicesStatus returns all devices with their online status
func GetDevicesStatus(devices []Device) []DeviceStatus {
	statuses := make([]DeviceStatus, len(devices))
	var wg sync.WaitGroup

	for i, device := range devices {
		statuses[i] = DeviceStatus{Device: device}

		if device.IP != "" {
			wg.Add(1)
			go func(idx int, ip string) {
				defer wg.Done()
				online := CheckDeviceOnline(ip)
				statuses[idx].Online = &online
			}(i, device.IP)
		}
	}

	wg.Wait()
	return statuses
}
