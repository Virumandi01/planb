package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [ID]",
	Short: "View the output logs of a specific instance",
	Long:  `Reads the stdout and stderr logs for a detached background process.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetID := strings.ToUpper(args[0])

		homeDir, _ := os.UserHomeDir()
		logPath := filepath.Join(homeDir, ".planb", "logs", fmt.Sprintf("%s_stdout.log", targetID))
		errLogPath := filepath.Join(homeDir, ".planb", "logs", fmt.Sprintf("%s_stderr.log", targetID))

		// Read standard output
		content, err := os.ReadFile(logPath)
		if err != nil {
			fmt.Printf("Planet VE: No standard logs found for '%s'. It may not have generated any output yet.\n", targetID)
		} else {
			fmt.Printf("=== STDOUT LOGS: %s ===\n", targetID)
			fmt.Println(string(content))
		}

		// Read error output
		errContent, err := os.ReadFile(errLogPath)
		if err == nil && len(errContent) > 0 {
			fmt.Printf("\n=== STDERR (ERRORS): %s ===\n", targetID)
			fmt.Println(string(errContent))
		}
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
