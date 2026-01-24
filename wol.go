package main

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
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

// SendWakeOnLAN sends a Wake-on-LAN magic packet to wake a device
func SendWakeOnLAN(mac string) error {
	packet, err := BuildMagicPacket(mac)
	if err != nil {
		return fmt.Errorf("failed to build magic packet: %w", err)
	}

	// Broadcast address
	addr, err := net.ResolveUDPAddr("udp", "255.255.255.255:9")
	if err != nil {
		return fmt.Errorf("failed to resolve broadcast address: %w", err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	defer conn.Close()

	// Send the magic packet
	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to send magic packet: %w", err)
	}

	return nil
}
