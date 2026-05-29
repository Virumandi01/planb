package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PlanBSettings struct {
	AutoHealthCheck    bool   `json:"auto_health_check"`
	CloudflareTunnelID string `json:"cloudflare_tunnel_id"`
	LogsLocation       string `json:"logs_location"`
	ErrorLogsLocation  string `json:"error_logs_location"`
	HostBookLocation   string `json:"host_book_location"`
}

func LoadSettings() (PlanBSettings, error) {
	homeDir, _ := os.UserHomeDir()
	settingsPath := filepath.Join(homeDir, ".planb", "settings.json")

	// Default Settings
	defaults := PlanBSettings{
		AutoHealthCheck:    false, // Default off to save RAM!
		CloudflareTunnelID: "Not Configured",
		LogsLocation:       filepath.Join(homeDir, ".planb", "logs"),
		ErrorLogsLocation:  filepath.Join(homeDir, ".planb", "logs", "errors.log"),
		HostBookLocation:   filepath.Join(homeDir, ".planb", "hostbook.json"),
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		SaveSettings(defaults)
		return defaults, nil
	}

	var s PlanBSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return defaults, err
	}
	return s, nil
}

func SaveSettings(s PlanBSettings) error {
	homeDir, _ := os.UserHomeDir()
	settingsPath := filepath.Join(homeDir, ".planb", "settings.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, data, 0644)
}
