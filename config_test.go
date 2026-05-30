package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig(\"\") error: %v", err)
	}
	if cfg.Server.Port != 8080 || cfg.Server.Host != "0.0.0.0" {
		t.Errorf("defaults = %+v, want port 8080 host 0.0.0.0", cfg.Server)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("server:\n  port: 9090\n  host: 127.0.0.1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Server.Port != 9090 || cfg.Server.Host != "127.0.0.1" {
		t.Errorf("got %+v, want port 9090 host 127.0.0.1", cfg.Server)
	}
}

// A missing explicit config path is tolerated (first-run before the file is
// created) and falls back to defaults.
func TestLoadConfigMissingFileTolerated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(missing) error: %v, want nil", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("port = %d, want default 8080", cfg.Server.Port)
	}
}

// An explicit path that exists but can't be read (here: a directory) must be a
// hard error rather than silently falling back to defaults.
func TestLoadConfigUnreadablePathErrors(t *testing.T) {
	dir := t.TempDir() // a directory is not a readable regular file
	if _, err := LoadConfig(dir); err == nil {
		t.Error("LoadConfig(directory) = nil error, want error")
	}
}

func TestLoadConfigParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yml")
	if err := os.WriteFile(path, []byte("server: [this is not valid: yaml"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Error("LoadConfig(bad yaml) = nil error, want error")
	}
}

func TestLoadConfigEnvOverride(t *testing.T) {
	t.Setenv("WOLNUT_SERVER_PORT", "3000")
	t.Setenv("WOLNUT_SERVER_HOST", "10.0.0.1")
	t.Setenv("WOLNUT_DATA_PATH", "/tmp/wolnut-data")
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Server.Port != 3000 || cfg.Server.Host != "10.0.0.1" || cfg.Data.Path != "/tmp/wolnut-data" {
		t.Errorf("env override not applied: %+v", cfg)
	}
}

func TestLoadConfigInvalidEnvPort(t *testing.T) {
	t.Setenv("WOLNUT_SERVER_PORT", "not-a-number")
	if _, err := LoadConfig(""); err == nil {
		t.Error("LoadConfig(non-numeric port) = nil error, want error")
	}
}

func TestLoadConfigPortOutOfRange(t *testing.T) {
	t.Setenv("WOLNUT_SERVER_PORT", "70000")
	if _, err := LoadConfig(""); err == nil {
		t.Error("LoadConfig(port 70000) = nil error, want error")
	}
}

func TestValidateAppData(t *testing.T) {
	// Reserved/empty/duplicate IDs are rewritten; valid entries pass.
	data := &AppData{
		Devices: []Device{
			{ID: "status", Name: "Reserved ID", MAC: "aa:bb:cc:dd:ee:ff"},
			{ID: "", Name: "Empty ID", MAC: "AA-BB-CC-DD-EE-00"},
		},
		UPSList: []UPSEntry{
			{ID: "dup", Name: "A", Host: "localhost", UPSName: "ups"},
			{ID: "dup", Name: "B", Host: "localhost:3493", UPSName: "ups"},
		},
	}
	if err := validateAppData(data); err != nil {
		t.Fatalf("validateAppData error: %v", err)
	}
	if data.Devices[0].ID == "status" || data.Devices[0].ID == "" {
		t.Error("reserved device ID was not rewritten")
	}
	if data.UPSList[0].ID == data.UPSList[1].ID {
		t.Error("duplicate UPS IDs were not made unique")
	}

	// Invalid MAC is rejected.
	bad := &AppData{Devices: []Device{{ID: "x", Name: "Bad", MAC: "nope"}}}
	if err := validateAppData(bad); err == nil {
		t.Error("validateAppData(bad MAC) = nil error, want error")
	}
}
