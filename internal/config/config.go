package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetConfigPath finds the ~/.planb/config.json file on your Mac
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".planb", "config.json"), nil
}

// Init ensures the folder and file exist
func Init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	planbDir := filepath.Join(homeDir, ".planb")

	// Create ~/.planb/ directory if it doesn't exist
	if err := os.MkdirAll(planbDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(planbDir, "config.json")

	// If config.json doesn't exist, create it with a default empty layout
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := PlanBConfig{
			Version:   "0.1.0",
			Instances: make(map[string]Instance),
		}

		bytes, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			return err
		}

		err = os.WriteFile(configPath, bytes, 0644)
		if err != nil {
			return err
		}
		fmt.Println("Planet VE: Initialized new config at", configPath)
	}

	return nil
}

// LoadConfig safely opens and parses the main data matrix file
func LoadConfig() (PlanBConfig, error) {
	var cfg PlanBConfig
	path, err := GetConfigPath()
	if err != nil {
		return cfg, err
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(bytes, &cfg)
	return cfg, err
}

// SaveConfig commits the memory config variables to physical state storage
func SaveConfig(cfg PlanBConfig) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}
