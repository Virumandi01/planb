package engine

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// StartDetached launched a backend script completely severed from the parent terminal.
func StartDetached(command string, dir string, stdoutLogPath string, stderrLogPath string) (int, error) {
	// Create individual log files for STDOUT and STDERR on demand
	outLog, err := os.OpenFile(stdoutLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to create stdout log: %w", err)
	}
	errLog, err := os.OpenFile(stderrLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		outLog.Close()
		return 0, fmt.Errorf("failed to create stderr log: %w", err)
	}

	// Spin up the process via standard Unix shell execution
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir

	// Bind outputs directly to the respective log files in ~/.planb/logs/
	cmd.Stdout = outLog
	cmd.Stderr = errLog

	// The Unix Magic: Setsid detaches the child context from the active terminal session.
	// This prevents closing the terminal from killing the running web server.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	// Start running immediately without waiting for it to finish
	if err := cmd.Start(); err != nil {
		outLog.Close()
		errLog.Close()
		return 0, err
	}

	// Clean up open file pointers safely in the main tool thread;
	// the detached process keeps its own copies of the descriptors.
	outLog.Close()
	errLog.Close()

	return cmd.Process.Pid, nil
}

// StopInstance kills a detached process instantly using its PID.
func StopInstance(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	// Send SIGTERM (Graceful Kill Signal)
	return proc.Signal(syscall.SIGTERM)
}
