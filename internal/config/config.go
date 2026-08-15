package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	Port           string `json:"port"`
	DefaultPrinter string `json:"default_printer"`
	PaperWidth     string `json:"paper_width"` // "58mm" or "80mm"
	DefaultCopies  int    `json:"default_copies"` // 1, 2, 3...
	AutoCut        bool   `json:"auto_cut"`
	OpenDrawer     bool   `json:"open_drawer"`
	LogLevel       string `json:"log_level"`
}

var (
	currentConfig *Config
	configMutex   sync.RWMutex
	configFile    = "config.json"
)

func DefaultConfig() *Config {
	return &Config{
		Port:           "18181",
		DefaultPrinter: "Predefinida",
		PaperWidth:     "80mm",
		DefaultCopies:  1,
		AutoCut:        true,
		OpenDrawer:     true,
		LogLevel:       "info",
	}
}

func LoadConfig() (*Config, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	exePath, err := os.Executable()
	var configPath string
	if err == nil {
		configPath = filepath.Join(filepath.Dir(exePath), configFile)
	} else {
		configPath = configFile
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = configFile
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg := DefaultConfig()
		_ = saveConfigUnlocked(cfg, configPath)
		currentConfig = cfg
		return cfg, nil
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		currentConfig = cfg
		return cfg, nil
	}

	if cfg.DefaultCopies < 1 {
		cfg.DefaultCopies = 1
	}

	currentConfig = cfg
	return cfg, nil
}

func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if currentConfig == nil {
		return DefaultConfig()
	}
	return currentConfig
}

func SaveConfig(cfg *Config) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	if cfg.DefaultCopies < 1 {
		cfg.DefaultCopies = 1
	}

	currentConfig = cfg
	exePath, err := os.Executable()
	var configPath string
	if err == nil {
		configPath = filepath.Join(filepath.Dir(exePath), configFile)
	} else {
		configPath = configFile
	}

	return saveConfigUnlocked(cfg, configPath)
}

func saveConfigUnlocked(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
