package cli

import (
	"fmt"
	"os"
	"planb/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// openCmd represents the "open" command
var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the interactive Plan-B Dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize the Bubbletea program
		p := tea.NewProgram(ui.InitialModel(), tea.WithAltScreen()) // AltScreen clears the terminal beautifully

		// Run the UI. When the user presses 'q', it closes and returns here.
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Add the "open" command to our root "planb" command
	rootCmd.AddCommand(openCmd)
}
