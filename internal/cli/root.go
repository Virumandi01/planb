package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "planb",
	Short: "Planet VE | Plan-B Local Hosting Manager",
	Long:  `A lightweight, 0MB-idle hosting control system and observability terminal.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("Planet VE: Plan-B Engine initialized. Awaiting commands.")
	},
}

// Execute adds all child commands to the root command.
func Execute() error {
	return rootCmd.Execute()
}
