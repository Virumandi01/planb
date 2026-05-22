package cli

import (
	"fmt"
	"planb/internal/config"
	"planb/internal/engine"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [ID]",
	Short: "Stop a detached local website instance",
	Long:  `Provide the instance ID (e.g., LH01) to terminate the background process gracefully.`,
	Args:  cobra.ExactArgs(1), // Requires exactly one argument: the ID
	Run: func(cmd *cobra.Command, args []string) {
		targetID := args[0]

		// 1. Load the database to find the PID
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error accessing config matrix: %v\n", err)
			return
		}

		// 2. Check if the ID exists in the database
		instance, exists := cfg.Instances[targetID]
		if !exists {
			fmt.Printf("Planet VE: Error - No instance found with ID '%s'\n", targetID)
			return
		}

		// Check if it is already stopped
		if instance.Status == "STOP" {
			fmt.Printf("Planet VE: Instance '%s' is already stopped.\n", targetID)
			return
		}

		fmt.Printf("Planet VE: Terminating process [PID: %d] for instance '%s'...\n", instance.LastPID, targetID)

		// 3. Fire the engine killer
		err = engine.StopInstance(instance.LastPID)
		if err != nil {
			fmt.Printf("Warning: Could not terminate process smoothly (It may have already crashed): %v\n", err)
			// We continue anyway to update the database state
		}

		// 4. Update the database state to reflect the shutdown
		instance.Status = "STOP"
		cfg.Instances[targetID] = instance

		if err := config.SaveConfig(cfg); err != nil {
			fmt.Printf("Error updating state mapping: %v\n", err)
			return
		}

		fmt.Printf("Instance '%s' successfully stopped.\n", targetID)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
