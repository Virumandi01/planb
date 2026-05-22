package config

// Instance represents a single hosted website/process
type Instance struct {
	ID           string `json:"id"`            // e.g., "LH01"
	Name         string `json:"name"`          // e.g., "AmazonClone"
	Path         string `json:"path"`          // Absolute path on Mac
	StartCommand string `json:"start_command"` // e.g., "flutter run -d web-server"
	TargetPort   int    `json:"target_port"`   // e.g., 3000
	Status       string `json:"status"`        // "LIVE" or "STOPPED"
	LastPID      int    `json:"last_pid"`      // The OS Process ID
}

// PlanBConfig is the master blueprint for the entire config.json file
type PlanBConfig struct {
	Version   string              `json:"version"`
	Instances map[string]Instance `json:"instances"`
}
