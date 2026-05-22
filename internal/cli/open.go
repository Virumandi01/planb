package cli

import (
	"fmt"
	"os"
	"planb/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the interactive Plan-B Dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		// Pass tea.WithAltScreen() to completely clear the screen and take over the terminal window
		p := tea.NewProgram(
			ui.InitialModel(),
			tea.WithAltScreen(),       // Takes over the entire window
			tea.WithMouseCellMotion(), // Prepares it for mouse interactions later
		)

		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
