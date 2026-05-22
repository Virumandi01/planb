package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define our clean, professional styles (No emojis)
var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")). // Planet VE Blueish
			PaddingLeft(1).
			PaddingRight(1)

	textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// model represents the state of our UI
type model struct {
	// We will load real config data here later
}

func InitialModel() model {
	return model{}
}

// Init runs when the UI starts
func (m model) Init() tea.Cmd {
	return nil
}

// Update listens for keyboard input
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If user presses 'q' or 'ctrl+c', exit the dashboard instantly (0MB RAM state)
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

// View draws the actual terminal UI
func (m model) View() string {
	// 1. Header
	header := titleStyle.Render("PLAN-B MANAGEMENT SYSTEM | Planet VE v0.1")

	// 2. System State (Hardcoded for this step, we will connect psutil next)
	systemState := textStyle.Render(
		"[SYSTEM STATE]\n" +
			"Core Engine: Idle (0MB RAM)  | Active Instances: 0     | Ports Bound: 0\n" +
			"Host CPU   : Scanning...     | Host Memory     : Scanning...\n",
	)
	systemBox := baseStyle.Render(systemState)

	// 3. Instance Repository Table
	tableHeader := textStyle.Bold(true).Render(
		fmt.Sprintf("%-6s | %-18s | %-8s | %-6s | %-8s | %-30s", "ID", "Name", "State", "Port", "Runtime", "Deployment Path"),
	)

	// Mock data for the layout test
	row1 := textStyle.Render(fmt.Sprintf("%-6s | %-18s | %-8s | %-6s | %-8s | %-30s", "LH01", "AmazonClone", "[LIVE]", "3000", "Node", "/Users/server/apps/amazon"))
	row2 := textStyle.Render(fmt.Sprintf("%-6s | %-18s | %-8s | %-6s | %-8s | %-30s", "PO02", "FastAPI_Engine", "[STOP]", "5000", "Python", "/Users/server/api/v1"))

	tableArea := textStyle.Render("[INSTANCE REPOSITORY]\n" + tableHeader + "\n" + strings.Repeat("-", 85) + "\n" + row1 + "\n" + row2)
	tableBox := baseStyle.Render(tableArea)

	// 4. Footer Directives
	footer := textStyle.Render(
		"Operational Directives:\n" +
			"planb start [ID]  |  planb stop [ID]  |  planb settings  |  Press 'q' to exit",
	)
	footerBox := baseStyle.Render(footer)

	// Combine everything into the final screen
	return lipgloss.JoinVertical(lipgloss.Left, header, systemBox, tableBox, footerBox)
}
