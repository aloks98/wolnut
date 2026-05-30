package main

import (
	"bytes"
	"net"
	"testing"
)

func TestValidateMAC(t *testing.T) {
	cases := map[string]bool{
		"AA:BB:CC:DD:EE:FF":    true,
		"aa:bb:cc:dd:ee:ff":    true,
		"AA-BB-CC-DD-EE-FF":    true,
		"01:23:45:67:89:ab":    true,
		"":                     false,
		"AA:BB:CC:DD:EE":       false, // too short
		"AA:BB:CC:DD:EE:FF:11": false, // too long
		"GG:BB:CC:DD:EE:FF":    false, // non-hex
		"AABBCCDDEEFF":         false, // no separators
		"AA:BB:CC:DD:EE:FZ":    false,
	}
	for in, want := range cases {
		if got := ValidateMAC(in); got != want {
			t.Errorf("ValidateMAC(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNormalizeMAC(t *testing.T) {
	cases := map[string]string{
		"aa-bb-cc-dd-ee-ff": "AA:BB:CC:DD:EE:FF",
		"aa:bb:cc:dd:ee:ff": "AA:BB:CC:DD:EE:FF",
		"AA:BB:CC:DD:EE:FF": "AA:BB:CC:DD:EE:FF",
	}
	for in, want := range cases {
		if got := NormalizeMAC(in); got != want {
			t.Errorf("NormalizeMAC(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseMAC(t *testing.T) {
	got, err := ParseMAC("01-23-45-67-89-AB")
	if err != nil {
		t.Fatalf("ParseMAC returned error: %v", err)
	}
	want := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB}
	if !bytes.Equal(got, want) {
		t.Errorf("ParseMAC = %v, want %v", got, want)
	}

	if _, err := ParseMAC("not-a-mac"); err == nil {
		t.Error("ParseMAC(invalid) = nil error, want error")
	}
}

func TestBuildMagicPacket(t *testing.T) {
	packet, err := BuildMagicPacket("01:23:45:67:89:AB")
	if err != nil {
		t.Fatalf("BuildMagicPacket returned error: %v", err)
	}
	if len(packet) != 102 {
		t.Fatalf("packet length = %d, want 102", len(packet))
	}
	// First 6 bytes must be 0xFF.
	for i := 0; i < 6; i++ {
		if packet[i] != 0xFF {
			t.Errorf("packet[%d] = %#x, want 0xFF", i, packet[i])
		}
	}
	// MAC repeated 16 times.
	mac := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB}
	for i := 0; i < 16; i++ {
		off := 6 + i*6
		if !bytes.Equal(packet[off:off+6], mac) {
			t.Errorf("repetition %d = %v, want %v", i, packet[off:off+6], mac)
		}
	}

	if _, err := BuildMagicPacket("bad"); err == nil {
		t.Error("BuildMagicPacket(invalid) = nil error, want error")
	}
}

func TestBroadcastAddr(t *testing.T) {
	cases := []struct {
		cidr string
		want string
	}{
		{"192.168.1.10/24", "192.168.1.255"},
		{"10.0.0.5/8", "10.255.255.255"},
		{"172.16.4.1/23", "172.16.5.255"},
		{"192.168.1.64/26", "192.168.1.127"},
	}
	for _, c := range cases {
		_, ipnet, err := net.ParseCIDR(c.cidr)
		if err != nil {
			t.Fatalf("ParseCIDR(%q): %v", c.cidr, err)
		}
		if got := broadcastAddr(ipnet); got == nil || got.String() != c.want {
			t.Errorf("broadcastAddr(%q) = %v, want %s", c.cidr, got, c.want)
		}
	}

	// IPv6 networks have no IPv4 directed broadcast.
	_, v6, _ := net.ParseCIDR("2001:db8::/64")
	if got := broadcastAddr(v6); got != nil {
		t.Errorf("broadcastAddr(ipv6) = %v, want nil", got)
	}
}

func TestIsLikelyBroadcast(t *testing.T) {
	cases := map[string]bool{
		"192.168.1.255": true,
		"10.0.0.255":    true,
		"192.168.1.42":  false,
		"255.255.255.0": false,
	}
	for in, want := range cases {
		if got := isLikelyBroadcast(net.ParseIP(in)); got != want {
			t.Errorf("isLikelyBroadcast(%q) = %v, want %v", in, got, want)
		}
	}
	// IPv6 is never treated as a /24 directed broadcast.
	if isLikelyBroadcast(net.ParseIP("::1")) {
		t.Error("isLikelyBroadcast(::1) = true, want false")
	}
}
