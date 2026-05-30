package main

import (
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// ianaProtocolICMP is the IP protocol number for ICMPv4 (used by icmp.ParseMessage).
const ianaProtocolICMP = 1

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

// CheckDeviceOnline reports whether a device is reachable.
//
// It prefers an ICMP echo, which detects a host that is up regardless of which
// ports it exposes, and falls back to a TCP-connect probe when ICMP isn't
// permitted (no CAP_NET_RAW and no unprivileged ping socket) or when the host
// doesn't answer ICMP but may still be serving TCP (ICMP-filtered hosts).
// ICMP is attempted for IPv4 only; IPv6 targets use the TCP probe.
func CheckDeviceOnline(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}

	if parsed.To4() != nil {
		switch alive, usable := icmpPingv4(parsed); {
		case usable && alive:
			return true
		case usable && !alive:
			// ICMP worked but no reply — host may filter ICMP; try TCP.
		default:
			// ICMP unavailable (no privilege) — TCP is the only option.
		}
	}

	return tcpProbe(parsed)
}

// tcpProbe returns true if a TCP connection to any common service port succeeds
// within pingTimeout. Ports are dialed in parallel; the first success cancels
// the rest.
func tcpProbe(ip net.IP) bool {
	ports := []string{"80", "443", "22", "3389", "445"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCh := make(chan bool, len(ports))
	host := ip.String()

	for _, port := range ports {
		go func(p string) {
			dialer := net.Dialer{Timeout: pingTimeout}
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, p))
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

// icmpPingv4 sends one ICMP echo request to ip and waits for a reply.
//
//	alive  — an echo reply came back from ip within pingTimeout
//	usable — an ICMP socket could be opened (i.e. ICMP is permitted here)
//
// When usable is false the caller should fall back to another probe.
func icmpPingv4(ip net.IP) (alive, usable bool) {
	// Prefer the unprivileged datagram socket (Linux net.ipv4.ping_group_range);
	// fall back to a raw socket (needs CAP_NET_RAW). The two use different
	// destination address types when sending.
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	udp := true
	if err != nil {
		conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		udp = false
		if err != nil {
			return false, false
		}
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(pingTimeout)); err != nil {
		return false, true
	}

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1, Data: []byte("wolnut-ping")},
	}
	wb, err := msg.Marshal(nil)
	if err != nil {
		return false, true
	}

	var dst net.Addr = &net.IPAddr{IP: ip}
	if udp {
		dst = &net.UDPAddr{IP: ip}
	}
	if _, err := conn.WriteTo(wb, dst); err != nil {
		return false, true
	}

	// Match replies by source address rather than ICMP ID: the unprivileged
	// udp4 socket has the kernel rewrite the ID, so an ID check would never
	// match there. Keep reading until a reply from ip arrives or the deadline.
	rb := make([]byte, 1500)
	for {
		n, peer, err := conn.ReadFrom(rb)
		if err != nil {
			return false, true // deadline reached or read error — no reply
		}
		if p := peerIP(peer); p == nil || !p.Equal(ip) {
			continue // a reply meant for some other ping
		}
		rm, err := icmp.ParseMessage(ianaProtocolICMP, rb[:n])
		if err != nil {
			continue
		}
		if rm.Type == ipv4.ICMPTypeEchoReply {
			return true, true
		}
	}
}

// peerIP extracts the IP from the address returned by icmp.PacketConn.ReadFrom,
// which is a *net.UDPAddr for udp4 sockets and a *net.IPAddr for raw sockets.
func peerIP(addr net.Addr) net.IP {
	switch a := addr.(type) {
	case *net.UDPAddr:
		return a.IP
	case *net.IPAddr:
		return a.IP
	}
	return nil
}
