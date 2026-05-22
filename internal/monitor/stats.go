package monitor

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// GetSystemStats returns the current CPU percentage and used RAM in GB
func GetSystemStats() (string, string) {
	// Grab RAM usage
	v, err := mem.VirtualMemory()
	ramStr := "ERR"
	if err == nil {
		// Convert bytes to Gigabytes
		ramGB := float64(v.Used) / 1024 / 1024 / 1024
		ramStr = fmt.Sprintf("%.1f GB", ramGB)
	}

	// Grab CPU usage (measuring over a tiny 100ms window for accuracy)
	cpuPercents, err := cpu.Percent(100*time.Millisecond, false)
	cpuStr := "ERR"
	if err == nil && len(cpuPercents) > 0 {
		cpuStr = fmt.Sprintf("%.1f%%", cpuPercents[0])
	}

	return cpuStr, ramStr
}
