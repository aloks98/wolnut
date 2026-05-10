package main

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"syscall"
)

var macRegex = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)

// ValidateMAC checks if a MAC address is valid
func ValidateMAC(mac string) bool {
	return macRegex.MatchString(mac)
}

// NormalizeMAC converts a MAC address to standard colon format
func NormalizeMAC(mac string) string {
	return strings.ToUpper(strings.ReplaceAll(mac, "-", ":"))
}

// ParseMAC parses a MAC address string into bytes
func ParseMAC(mac string) ([]byte, error) {
	mac = NormalizeMAC(mac)
	if !ValidateMAC(mac) {
		return nil, errors.New("invalid MAC address format")
	}

	parts := strings.Split(mac, ":")
	bytes := make([]byte, 6)
	for i, part := range parts {
		var b byte
		_, err := fmt.Sscanf(part, "%02X", &b)
		if err != nil {
			return nil, err
		}
		bytes[i] = b
	}
	return bytes, nil
}

// BuildMagicPacket creates a Wake-on-LAN magic packet
// The packet consists of 6 bytes of 0xFF followed by the MAC address repeated 16 times
func BuildMagicPacket(mac string) ([]byte, error) {
	macBytes, err := ParseMAC(mac)
	if err != nil {
		return nil, err
	}

	// Magic packet: 6x 0xFF + 16x MAC
	packet := make([]byte, 102)

	// First 6 bytes are 0xFF
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}

	// Next 96 bytes are MAC repeated 16 times
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], macBytes)
	}

	return packet, nil
}

// SendWakeOnLAN sends a Wake-on-LAN magic packet.
//
// If targetIP is a unicast or directed-broadcast address (e.g. 192.168.1.42 or
// 192.168.1.255), the packet is sent there — directed broadcasts can traverse
// L3 boundaries when intermediate switches keep ARP entries for powered-off
// hosts. If targetIP is empty or unparseable, falls back to limited broadcast
// (255.255.255.255), which only reaches the local L2 segment. SO_BROADCAST is
// set explicitly when broadcasting; without it, Linux returns EACCES on send.
func SendWakeOnLAN(mac, targetIP string) error {
	packet, err := BuildMagicPacket(mac)
	if err != nil {
		return fmt.Errorf("failed to build magic packet: %w", err)
	}

	ip := net.ParseIP(strings.TrimSpace(targetIP))
	if ip == nil {
		ip = net.IPv4bcast
	}
	isBroadcast := ip.Equal(net.IPv4bcast) || isLikelyBroadcast(ip)

	pc, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return fmt.Errorf("failed to open UDP socket: %w", err)
	}
	defer pc.Close()

	if isBroadcast {
		sc, err := pc.(*net.UDPConn).SyscallConn()
		if err != nil {
			return fmt.Errorf("syscall conn: %w", err)
		}
		var sockErr error
		if err := sc.Control(func(fd uintptr) {
			sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		}); err != nil {
			return fmt.Errorf("set SO_BROADCAST: %w", err)
		}
		if sockErr != nil {
			return fmt.Errorf("set SO_BROADCAST: %w", sockErr)
		}
	}

	if _, err := pc.WriteTo(packet, &net.UDPAddr{IP: ip, Port: 9}); err != nil {
		return fmt.Errorf("failed to send magic packet: %w", err)
	}
	return nil
}

// isLikelyBroadcast returns true for an IPv4 address whose host portion
// looks like all-ones in a /24 (e.g., 192.168.1.255).  We don't know the
// real netmask, so this is a heuristic for the common /24 case.
func isLikelyBroadcast(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return v4[3] == 255
}
