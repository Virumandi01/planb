package monitor

import (
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// DetectPort scans the macOS kernel to find which TCP port a background process is using.
func DetectPort(parentPID int) int {
	// 1. Grab all active TCP connections on the entire Mac
	conns, err := net.Connections("tcp")
	if err != nil {
		return 0
	}

	// 2. Build a map of valid PIDs (the parent shell + the actual child script)
	validPIDs := make(map[int32]bool)
	validPIDs[int32(parentPID)] = true

	p, err := process.NewProcess(int32(parentPID))
	if err == nil {
		children, _ := p.Children()
		for _, child := range children {
			validPIDs[child.Pid] = true
		}
	}

	// 3. Search the network stack for our specific PIDs in a "LISTEN" state
	for _, c := range conns {
		if c.Status == "LISTEN" && validPIDs[c.Pid] {
			return int(c.Laddr.Port)
		}
	}

	return 0
}
