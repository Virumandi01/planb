package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"planb/internal/config"
	"planb/internal/engine"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [name] [command]",
	Short: "Launch a detached local website instance",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		runScript := args[1]

		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error resolving path context: %v\n", err)
			return
		}

		homeDir, _ := os.UserHomeDir()
		logsDir := filepath.Join(homeDir, ".planb", "logs")
		_ = os.MkdirAll(logsDir, 0755)

		// 1. Load the actual config matrix using our new function
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error accessing config matrix: %v\n", err)
			return
		}

		nextNum := len(cfg.Instances) + 1
		nextID := fmt.Sprintf("LH%02d", nextNum)

		stdoutLog := filepath.Join(logsDir, fmt.Sprintf("%s_stdout.log", nextID))
		stderrLog := filepath.Join(logsDir, fmt.Sprintf("%s_stderr.log", nextID))

		fmt.Printf("Planet VE: Launching '%s' via detached thread...\n", name)

		pid, err := engine.StartDetached(runScript, currentDir, stdoutLog, stderrLog)
		if err != nil {
			fmt.Printf("Execution engine failed to launch instance: %v\n", err)
			return
		}

		newInstance := config.Instance{
			ID:           nextID,
			Name:         name,
			Path:         currentDir,
			StartCommand: runScript,
			TargetPort:   0,
			Status:       "LIVE",
			LastPID:      pid,
		}

		cfg.Instances[nextID] = newInstance

		// 2. Save the modifications cleanly using our new function
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving updated state mapping: %v\n", err)
			return
		}

		fmt.Printf("Successfully registered instance! [ID: %s] [PID: %d]\n", nextID, pid)
		fmt.Printf("Output logs routed cleanly to: ~/.planb/logs/%s_*.log\n", nextID)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
