package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"planb/internal/config"
	"planb/internal/engine"
	"planb/internal/monitor"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")).
			PaddingLeft(1).
			PaddingRight(1)

	textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	echoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // Dimmer color for your echoed commands
)

// logLine represents a single line in the embedded terminal history
type logLine struct {
	text  string
	isErr bool
	isCmd bool // True if it's the command you typed, false if it's a system response
}

type model struct {
	cpuLoad   string
	ramUsed   string
	instances map[string]config.Instance

	input   textinput.Model
	history []logLine // Stores the terminal history
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func InitialModel() model {
	cpu, ram := monitor.GetSystemStats()
	cfg, _ := config.LoadConfig()

	ti := textinput.New()
	ti.Placeholder = "Type 'clear', 'stop LH01', etc..."
	ti.Prompt = "PLAN-B > "
	ti.PromptStyle = promptStyle
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 80

	return model{
		cpuLoad:   cpu,
		ramUsed:   ram,
		instances: cfg.Instances,
		input:     ti,
		history:   []logLine{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tick())
}

// addLog appends a line to the terminal history and keeps only the last 4 lines
func (m *model) addLog(text string, isErr bool, isCmd bool) {
	m.history = append(m.history, logLine{text: text, isErr: isErr, isCmd: isCmd})
	if len(m.history) > 4 {
		m.history = m.history[len(m.history)-4:] // Drop the oldest line
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			val := m.input.Value()
			m.input.SetValue("")
			m.processCommand(val)
			return m, nil
		}

	case tickMsg:
		// 1. Refresh System Stats
		m.cpuLoad, m.ramUsed = monitor.GetSystemStats()

		// 2. Refresh Database
		cfg, _ := config.LoadConfig()

		// 3. AUTO-PORT SCANNER LOGIC
		needsSave := false
		for k, inst := range cfg.Instances {
			// If it's LIVE but we don't know the port yet...
			if inst.Status == "LIVE" && inst.TargetPort == 0 {
				detectedPort := monitor.DetectPort(inst.LastPID)
				if detectedPort > 0 {
					// We found the port! Update the struct and save it forever.
					inst.TargetPort = detectedPort
					cfg.Instances[k] = inst
					needsSave = true

					// Announce it in the embedded terminal
					m.addLog(fmt.Sprintf("System detected %s bound to Port %d", k, detectedPort), false, false)
				}
			}
		}

		if needsSave {
			_ = config.SaveConfig(cfg)
		}

		m.instances = cfg.Instances
		return m, tick()
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *model) processCommand(inputStr string) {
	inputStr = strings.TrimSpace(inputStr)
	if inputStr == "" {
		return
	}

	// Echo the command you just typed into the history
	m.addLog(inputStr, false, true)

	parts := strings.SplitN(inputStr, " ", 3)
	action := strings.ToLower(parts[0])

	switch action {
	case "clear":
		m.history = []logLine{} // Instantly wipe the terminal history
		return

	case "stop":
		if len(parts) < 2 {
			m.addLog("Usage: stop [ID]", true, false)
			return
		}
		targetID := strings.ToUpper(parts[1])
		cfg, _ := config.LoadConfig()
		inst, exists := cfg.Instances[targetID]

		if !exists {
			m.addLog(fmt.Sprintf("Error: No instance found with ID '%s'", targetID), true, false)
			return
		}
		if inst.Status == "STOP" {
			m.addLog(fmt.Sprintf("Instance '%s' is already stopped.", targetID), false, false)
			return
		}

		_ = engine.StopInstance(inst.LastPID)
		inst.Status = "STOP"
		cfg.Instances[targetID] = inst
		_ = config.SaveConfig(cfg)

		m.addLog(fmt.Sprintf("Successfully stopped %s.", targetID), false, false)

	case "start":
		if len(parts) < 3 {
			m.addLog("Usage: start [Name] [Command]", true, false)
			return
		}
		name := parts[1]
		runScript := parts[2]

		currentDir, _ := os.Getwd()
		homeDir, _ := os.UserHomeDir()
		logsDir := filepath.Join(homeDir, ".planb", "logs")
		cfg, _ := config.LoadConfig()

		nextNum := len(cfg.Instances) + 1
		nextID := fmt.Sprintf("LH%02d", nextNum)

		stdoutLog := filepath.Join(logsDir, fmt.Sprintf("%s_stdout.log", nextID))
		stderrLog := filepath.Join(logsDir, fmt.Sprintf("%s_stderr.log", nextID))

		pid, err := engine.StartDetached(runScript, currentDir, stdoutLog, stderrLog)
		if err != nil {
			m.addLog(fmt.Sprintf("Failed to launch: %v", err), true, false)
			return
		}

		cfg.Instances[nextID] = config.Instance{
			ID:           nextID,
			Name:         name,
			Path:         currentDir,
			StartCommand: runScript,
			TargetPort:   0,
			Status:       "LIVE",
			LastPID:      pid,
		}
		_ = config.SaveConfig(cfg)

		m.addLog(fmt.Sprintf("Started %s [%s] on PID: %d", name, nextID, pid), false, false)

	default:
		m.addLog(fmt.Sprintf("Unknown command: %s", action), true, false)
	}
}

func (m model) View() string {
	header := titleStyle.Render("PLAN-B MANAGEMENT SYSTEM | Planet VE v0.1")

	activeCount := 0
	for _, inst := range m.instances {
		if inst.Status == "LIVE" {
			activeCount++
		}
	}

	systemState := textStyle.Render(
		fmt.Sprintf("[SYSTEM STATE]\n"+
			"Core Engine: Active (Live)   | Active Instances: %-5d | Ports Bound: 0\n"+
			"Host CPU   : %-14s  | Host Memory     : %-14s\n", activeCount, m.cpuLoad, m.ramUsed),
	)
	systemBox := baseStyle.Render(systemState)

	tableHeader := textStyle.Bold(true).Render(
		fmt.Sprintf("%-6s | %-18s | %-8s | %-6s | %-8s | %-30s", "ID", "Name", "State", "Port", "Runtime", "Deployment Path"),
	)

	var rows []string
	if len(m.instances) == 0 {
		rows = append(rows, textStyle.Render("No instances registered. Use the terminal below to start one."))
	} else {
		var keys []string
		for k := range m.instances {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, id := range keys {
			inst := m.instances[id]
			stateStr := fmt.Sprintf("[%s]", inst.Status)
			portStr := fmt.Sprintf("%d", inst.TargetPort)
			if inst.TargetPort == 0 {
				portStr = "N/A"
			}
			path := inst.Path
			if len(path) > 28 {
				path = "..." + path[len(path)-25:]
			}
			runtime := "Custom"
			cmdLower := strings.ToLower(inst.StartCommand)
			if strings.Contains(cmdLower, "python") {
				runtime = "Python"
			} else if strings.Contains(cmdLower, "node") {
				runtime = "Node.js"
			}
			row := textStyle.Render(fmt.Sprintf("%-6s | %-18s | %-8s | %-6s | %-8s | %-30s",
				id, inst.Name, stateStr, portStr, runtime, path))
			rows = append(rows, row)
		}
	}

	tableArea := textStyle.Render("[INSTANCE REPOSITORY]\n" + tableHeader + "\n" + strings.Repeat("-", 85) + "\n" + strings.Join(rows, "\n"))
	tableBox := baseStyle.Render(tableArea)

	// Build the Fixed-Height Embedded Terminal History
	historyLines := ""

	// Pad with empty lines if we have fewer than 4 items in history
	emptyLinesNeeded := 4 - len(m.history)
	for i := 0; i < emptyLinesNeeded; i++ {
		historyLines += "\n"
	}

	// Render the actual history
	for _, msg := range m.history {
		if msg.isCmd {
			historyLines += echoStyle.Render("PLAN-B > "+msg.text) + "\n"
		} else if msg.isErr {
			historyLines += errorStyle.Render(">> "+msg.text) + "\n"
		} else {
			historyLines += successStyle.Render(">> "+msg.text) + "\n"
		}
	}

	inputArea := baseStyle.Render("[EMBEDDED TERMINAL]\n" + historyLines + m.input.View())

	footer := textStyle.Render("Press 'Esc' or 'Ctrl+C' to exit back to OS terminal.")
	footerBox := baseStyle.Render(footer)

	return lipgloss.JoinVertical(lipgloss.Left, header, systemBox, tableBox, inputArea, footerBox)
}
