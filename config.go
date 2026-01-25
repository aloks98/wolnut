package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"gopkg.in/yaml.v3"
)

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

	// Load from file if exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			yaml.Unmarshal(data, &cfg)
		}
	}

	// Environment overrides
	if v := os.Getenv("WOLNUT_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
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
	return os.WriteFile(s.dataFilePath(), data, 0644)
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
	return nil
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
	return nil
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
	return nil
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
	return nil
}

func (s *AppState) GetUPSList() []UPSEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	upsList := make([]UPSEntry, len(s.Data.UPSList))
	copy(upsList, s.Data.UPSList)
	return upsList
}

func (s *AppState) GetRawData() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, _ := json.MarshalIndent(s.Data, "", "  ")
	return data
}

func (s *AppState) ImportData(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var newData AppData
	if err := json.Unmarshal(data, &newData); err != nil {
		return err
	}

	s.Data = newData
	return s.saveDataLocked()
}
