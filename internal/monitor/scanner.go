package monitor

import (
	"sort"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// DiscoveredApp holds data for processes found in the wild
type DiscoveredApp struct {
	PID        int
	Name       string
	Path       string
	Port       int
	IsLocal    bool
	CreateTime int64
}

// ScanGlobalNetwork sweeps the Mac for all listening TCP ports
func ScanGlobalNetwork() []DiscoveredApp {
	conns, err := net.Connections("tcp")
	if err != nil {
		return nil
	}

	var found []DiscoveredApp

	for _, c := range conns {
		if c.Status == "LISTEN" {
			pid := int(c.Pid)
			if pid == 0 {
				continue // Skip kernel/system level ghosts
			}

			p, err := process.NewProcess(int32(pid))
			if err != nil {
				continue
			}

			name, _ := p.Name()
			if name == "" {
				name = "Unknown"
			}

			// Try to get the executable path (macOS may block this for system apps)
			exePath, _ := p.Exe()
			if exePath == "" {
				exePath = "System/Protected"
			}

			createTime, _ := p.CreateTime()

			// Determine if it's open to the internet (0.0.0.0 / ::) or just local (127.0.0.1)
			isLocal := true
			ip := c.Laddr.IP
			if ip == "0.0.0.0" || ip == "::" || ip == "" {
				isLocal = false
			}

			found = append(found, DiscoveredApp{
				PID:        pid,
				Name:       name,
				Path:       exePath,
				Port:       int(c.Laddr.Port),
				IsLocal:    isLocal,
				CreateTime: createTime,
			})
		}
	}

	// Sort chronologically (oldest running process first)
	sort.Slice(found, func(i, j int) bool {
		return found[i].CreateTime < found[j].CreateTime
	})

	return found
}
