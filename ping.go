package main

import (
	"context"
	"net"
	"sync"
	"time"
)

// DeviceStatus represents a device with its online status
type DeviceStatus struct {
	Device
	Online *bool `json:"online"` // nil = no IP configured, true = online, false = offline
}

const pingTimeout = 1 * time.Second

// StatusCache caches device online status with background refresh
type StatusCache struct {
	mu       sync.RWMutex
	statuses map[string]*bool // device ID -> online status
	state    *AppState
	stopCh   chan struct{}
}

// NewStatusCache creates a new status cache with background refresh
func NewStatusCache(state *AppState) *StatusCache {
	sc := &StatusCache{
		statuses: make(map[string]*bool),
		state:    state,
		stopCh:   make(chan struct{}),
	}
	go sc.backgroundRefresh()
	return sc
}

// backgroundRefresh periodically updates device status
func (sc *StatusCache) backgroundRefresh() {
	// Initial refresh
	sc.refresh()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sc.refresh()
		case <-sc.stopCh:
			return
		}
	}
}

// refresh updates all device statuses
func (sc *StatusCache) refresh() {
	devices := sc.state.GetDevices()
	newStatuses := make(map[string]*bool)
	var wg sync.WaitGroup

	var mu sync.Mutex
	for _, device := range devices {
		if device.IP != "" {
			wg.Add(1)
			go func(d Device) {
				defer wg.Done()
				online := CheckDeviceOnline(d.IP)
				mu.Lock()
				newStatuses[d.ID] = &online
				mu.Unlock()
			}(device)
		}
	}

	wg.Wait()

	sc.mu.Lock()
	sc.statuses = newStatuses
	sc.mu.Unlock()
}

// GetStatus returns cached status for a device
func (sc *StatusCache) GetStatus(deviceID string) *bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.statuses[deviceID]
}

// GetDevicesStatus returns all devices with their cached online status
func (sc *StatusCache) GetDevicesStatus() []DeviceStatus {
	devices := sc.state.GetDevices()
	statuses := make([]DeviceStatus, len(devices))

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for i, device := range devices {
		statuses[i] = DeviceStatus{Device: device}
		if device.IP != "" {
			statuses[i].Online = sc.statuses[device.ID]
		}
	}

	return statuses
}

// Stop stops the background refresh
func (sc *StatusCache) Stop() {
	close(sc.stopCh)
}

// CheckDeviceOnline checks if a device is reachable via TCP ping
// All ports are checked in parallel for faster response
func CheckDeviceOnline(ip string) bool {
	if ip == "" {
		return false
	}

	// Try common ports in parallel
	ports := []string{"80", "443", "22", "3389", "445"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCh := make(chan bool, len(ports))

	for _, port := range ports {
		go func(p string) {
			dialer := net.Dialer{Timeout: pingTimeout}
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, p))
			if err == nil {
				conn.Close()
				resultCh <- true
				cancel() // Cancel other goroutines
				return
			}
			resultCh <- false
		}(port)
	}

	// Wait for all results or first success
	for i := 0; i < len(ports); i++ {
		if <-resultCh {
			return true
		}
	}

	return false
}

