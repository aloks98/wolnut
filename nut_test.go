package main

import (
	"strings"
	"testing"
)

func TestValidateUPSHost(t *testing.T) {
	cases := map[string]bool{
		"localhost":              true,
		"localhost:3493":         true,
		"192.168.1.10":           true,
		"192.168.1.10:3493":      true,
		"nut.example.com":        true,
		"":                       false,
		"host:0":                 false, // port out of range
		"host:99999":             false, // port out of range
		"http://host":            false, // scheme
		"host/path":              false, // path
		"host name":              false, // whitespace
		"host:port":              false, // non-numeric port
		strings.Repeat("a", 254): false, // too long
	}
	for in, want := range cases {
		if got := ValidateUPSHost(in); got != want {
			t.Errorf("ValidateUPSHost(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestValidateUPSName(t *testing.T) {
	cases := map[string]bool{
		"ups":                   true,
		"ups-1":                 true,
		"my.ups_name":           true,
		"":                      false,
		"ups name":              false, // whitespace — NUT command injection guard
		"ups\nLOGIN":            false, // newline injection
		"ups;rm":                false,
		strings.Repeat("u", 65): false, // too long
	}
	for in, want := range cases {
		if got := ValidateUPSName(in); got != want {
			t.Errorf("ValidateUPSName(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestGetStatusLabel(t *testing.T) {
	cases := map[string]string{
		"OL":         "Online",
		"OB":         "On Battery",
		"OL CHRG":    "Online, Charging",
		"OB DISCHRG": "On Battery, Discharging",
		// DISCHRG contains the substring "CHRG"; must not be read as Charging.
		"OL DISCHRG":  "Online, Discharging",
		"OB LB":       "On Battery, Low Battery",
		"OL BYPASS":   "Online, Bypass",
		"":            "Unknown",
		"WEIRDSTATUS": "WEIRDSTATUS", // unknown tokens pass through verbatim
	}
	for in, want := range cases {
		if got := GetStatusLabel(in); got != want {
			t.Errorf("GetStatusLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
