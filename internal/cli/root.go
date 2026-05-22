package cli

import (
	"fmt"
	"os"
	"planb/internal/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "planb",
	Short: "Planet VE | Plan-B Local Hosting Manager",
	Long:  `A lightweight, 0MB-idle hosting control system and observability terminal.`,
	// PersistentPreRun runs before ANY command is executed
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := config.Init(); err != nil {
			fmt.Println("Error initializing Plan-B config:", err)
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("Planet VE: Plan-B Engine initialized. System is ready.")
	},
}

func Execute() error {
	return rootCmd.Execute()
}
