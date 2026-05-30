package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// reservedIDs are path segments under /api/devices/ and /api/ups/ that would
// shadow CRUD routes if used as an entity ID.  Server-generated UUIDs never
// hit these, but an imported data.json could.
var reservedIDs = map[string]bool{
	"status": true,
	"wake":   true,
	"":       true,
}

// ErrNotFound is returned by Update*/Delete* when no entry matches the given ID.
var ErrNotFound = errors.New("not found")

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type DataConfig struct {
	Path string `yaml:"path"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	Data   DataConfig   `yaml:"data"`
	Log    LogConfig    `yaml:"log"`
}

type Device struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MAC  string `json:"mac"`
	IP   string `json:"ip,omitempty"`
}

type UPSEntry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	UPSName string `json:"ups_name"`
}

type AppData struct {
	Devices []Device   `json:"devices"`
	UPSList []UPSEntry `json:"ups_list"`
}

type AppState struct {
	Config Config
	Data   AppData
	mu     sync.RWMutex
}

func LoadConfig(path string) (Config, error) {
	// Defaults
	cfg := Config{
		Server: ServerConfig{Port: 8080, Host: "0.0.0.0"},
		Data:   DataConfig{Path: "./data"},
		Log:    LogConfig{Level: "info"},
	}

	// Load from file if exists. An explicitly-passed path that can't be read
	// (typo, bad permissions) is fatal — silently falling back to defaults
	// would run against the wrong data directory without any signal. A missing
	// file is tolerated so first-run with a not-yet-created config works.
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return cfg, fmt.Errorf("read %s: %w", path, err)
			}
		} else if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse %s: %w", path, err)
		}
	}

	// Environment overrides
	if v := os.Getenv("WOLNUT_SERVER_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return cfg, fmt.Errorf("WOLNUT_SERVER_PORT: not a number: %q", v)
		}
		cfg.Server.Port = port
	}
	if v := os.Getenv("WOLNUT_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("WOLNUT_DATA_PATH"); v != "" {
		cfg.Data.Path = v
	}
	if v := os.Getenv("WOLNUT_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}

	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return cfg, fmt.Errorf("server.port %d out of range (1-65535)", cfg.Server.Port)
	}

	return cfg, nil
}

func (s *AppState) dataFilePath() string {
	return filepath.Join(s.Config.Data.Path, "data.json")
}

func (s *AppState) LoadData() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure data directory exists
	if err := os.MkdirAll(s.Config.Data.Path, 0755); err != nil {
		return err
	}

	// Initialize with empty data
	s.Data = AppData{
		Devices: []Device{},
		UPSList: []UPSEntry{},
	}

	// Load from file if exists
	data, err := os.ReadFile(s.dataFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty data file
			return s.saveDataLocked()
		}
		return err
	}

	return json.Unmarshal(data, &s.Data)
}

func (s *AppState) SaveData() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveDataLocked()
}

func (s *AppState) saveDataLocked() error {
	data, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}

	target := s.dataFilePath()
	tmp, err := os.CreateTemp(filepath.Dir(target), ".data.json.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, target)
}

func (s *AppState) AddDevice(device Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Data.Devices = append(s.Data.Devices, device)
	return s.saveDataLocked()
}

func (s *AppState) UpdateDevice(device Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, d := range s.Data.Devices {
		if d.ID == device.ID {
			s.Data.Devices[i] = device
			return s.saveDataLocked()
		}
	}
	return ErrNotFound
}

func (s *AppState) DeleteDevice(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, d := range s.Data.Devices {
		if d.ID == id {
			s.Data.Devices = append(s.Data.Devices[:i], s.Data.Devices[i+1:]...)
			return s.saveDataLocked()
		}
	}
	return ErrNotFound
}

func (s *AppState) GetDevice(id string) (Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, d := range s.Data.Devices {
		if d.ID == id {
			return d, true
		}
	}
	return Device{}, false
}

func (s *AppState) GetDevices() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]Device, len(s.Data.Devices))
	copy(devices, s.Data.Devices)
	return devices
}

func (s *AppState) AddUPS(ups UPSEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Data.UPSList = append(s.Data.UPSList, ups)
	return s.saveDataLocked()
}

func (s *AppState) UpdateUPS(ups UPSEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, u := range s.Data.UPSList {
		if u.ID == ups.ID {
			s.Data.UPSList[i] = ups
			return s.saveDataLocked()
		}
	}
	return ErrNotFound
}

func (s *AppState) DeleteUPS(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, u := range s.Data.UPSList {
		if u.ID == id {
			s.Data.UPSList = append(s.Data.UPSList[:i], s.Data.UPSList[i+1:]...)
			return s.saveDataLocked()
		}
	}
	return ErrNotFound
}

func (s *AppState) GetUPSList() []UPSEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	upsList := make([]UPSEntry, len(s.Data.UPSList))
	copy(upsList, s.Data.UPSList)
	return upsList
}

func (s *AppState) GetRawData() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.MarshalIndent(s.Data, "", "  ")
}

func (s *AppState) ImportData(data []byte) error {
	var newData AppData
	if err := json.Unmarshal(data, &newData); err != nil {
		return err
	}

	if err := validateAppData(&newData); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data = newData
	return s.saveDataLocked()
}

// validateAppData enforces the same constraints as the HTTP handlers and
// rewrites IDs that are missing, duplicated, or reserved so an imported
// data.json can't shadow CRUD routes (e.g. id == "status") or create
// unaddressable entries.
func validateAppData(d *AppData) error {
	seen := make(map[string]bool)
	for i := range d.Devices {
		dev := &d.Devices[i]
		if dev.Name == "" || len(dev.Name) > 128 {
			return fmt.Errorf("device %d: name is required and must be <=128 chars", i)
		}
		dev.MAC = NormalizeMAC(dev.MAC)
		if !ValidateMAC(dev.MAC) {
			return fmt.Errorf("device %q: invalid MAC address", dev.Name)
		}
		if dev.IP != "" && net.ParseIP(dev.IP) == nil {
			return fmt.Errorf("device %q: invalid IP address %q", dev.Name, dev.IP)
		}
		if dev.ID == "" || reservedIDs[dev.ID] || seen[dev.ID] {
			dev.ID = uuid.New().String()
		}
		seen[dev.ID] = true
	}

	seenUPS := make(map[string]bool)
	for i := range d.UPSList {
		u := &d.UPSList[i]
		if u.Name == "" || len(u.Name) > 128 {
			return fmt.Errorf("ups %d: name is required and must be <=128 chars", i)
		}
		if !ValidateUPSHost(u.Host) {
			return fmt.Errorf("ups %q: invalid host %q", u.Name, u.Host)
		}
		if !ValidateUPSName(u.UPSName) {
			return fmt.Errorf("ups %q: invalid ups_name %q", u.Name, u.UPSName)
		}
		if u.ID == "" || reservedIDs[u.ID] || seenUPS[u.ID] {
			u.ID = uuid.New().String()
		}
		seenUPS[u.ID] = true
	}
	return nil
}
