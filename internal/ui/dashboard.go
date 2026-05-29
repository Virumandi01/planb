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
	// Default border style
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	// High-contrast border style for the focused panel
	focusedStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("69")) // Planet VE Active Blue

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")).
			PaddingLeft(1).
			PaddingRight(1)

	textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	echoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

type logLine struct {
	text  string
	isErr bool
	isCmd bool
}

type model struct {
	cpuLoad   string
	ramUsed   string
	instances map[string]config.Instance

	input   textinput.Model
	history []logLine

	// New Navigation States
	focusedPanel string // "terminal" or "table"
	cursor       int    // Which row is highlighted in the table
	scrollOffset int    // Viewport window tracking pointer
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func InitialModel() model {
	cpu, ram := monitor.GetSystemStats()
	cfg, _ := config.LoadConfig()

	// Global Radar Boot Sequence
	discovered := monitor.ScanGlobalNetwork()
	needsSave := false

	opCount, lhCount := 0, 0
	for k := range cfg.Instances {
		if strings.HasPrefix(k, "OP") {
			opCount++
		}
		if strings.HasPrefix(k, "LH") {
			lhCount++
		}
	}

	for _, app := range discovered {
		alreadyTracked := false
		for _, inst := range cfg.Instances {
			if inst.LastPID == app.PID && inst.Status == "LIVE" {
				alreadyTracked = true
				break
			}
		}

		if !alreadyTracked {
			var newID string
			if app.IsLocal {
				lhCount++
				newID = fmt.Sprintf("LH%02d", lhCount)
			} else {
				opCount++
				newID = fmt.Sprintf("OP%02d", opCount)
			}

			defaultName := fmt.Sprintf("%s_%d", app.Name, app.Port)

			cfg.Instances[newID] = config.Instance{
				ID:           newID,
				Name:         defaultName,
				Path:         app.Path,
				StartCommand: "External Process",
				TargetPort:   app.Port,
				Status:       "LIVE",
				LastPID:      app.PID,
			}
			needsSave = true
		}
	}

	if needsSave {
		_ = config.SaveConfig(cfg)
	}

	ti := textinput.New()
	ti.Placeholder = "Type command... (Press 'Tab' to switch focus to table navigation)"
	ti.Prompt = "PLAN-B > "
	ti.PromptStyle = promptStyle
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 80

	return model{
		cpuLoad:      cpu,
		ramUsed:      ram,
		instances:    cfg.Instances,
		input:        ti,
		history:      []logLine{{text: "System Boot: Global Radar Scan Complete.", isErr: false, isCmd: false}},
		focusedPanel: "terminal", // Boot directly focused into the terminal prompt
		cursor:       0,
		scrollOffset: 0,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.ClearScreen, textinput.Blink, tick())
}

func (m *model) addLog(text string, isErr bool, isCmd bool) {
	m.history = append(m.history, logLine{text: text, isErr: isErr, isCmd: isCmd})
	if len(m.history) > 4 {
		m.history = m.history[len(m.history)-4:]
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Get total instances count for boundary protection
	totalInstances := len(m.instances)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, tea.ClearScreen
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyTab:
			// Toggle focus between the panel views seamlessly
			if m.focusedPanel == "terminal" {
				m.focusedPanel = "table"
				m.input.Blur() // Turn off blinking text cursor
			} else {
				m.focusedPanel = "terminal"
				m.input.Focus() // Turn on blinking text cursor
			}
			return m, nil

		case tea.KeyEnter:
			if m.focusedPanel == "terminal" {
				val := m.input.Value()
				m.input.SetValue("")
				m.processCommand(val)
			}
			return m, nil
		}

		// Handle specific navigation keystrokes if the table panel has active focus
		if m.focusedPanel == "table" && totalInstances > 0 {
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < totalInstances-1 {
					m.cursor++
				}
			}
			return m, nil
		}

	case tickMsg:
		m.cpuLoad, m.ramUsed = monitor.GetSystemStats()
		cfg, _ := config.LoadConfig()
		m.instances = cfg.Instances

		// Keep auto-port scanner logic active in background thread
		needsSave := false
		for k, inst := range cfg.Instances {
			if inst.Status == "LIVE" && inst.TargetPort == 0 {
				detectedPort := monitor.DetectPort(inst.LastPID)
				if detectedPort > 0 {
					inst.TargetPort = detectedPort
					cfg.Instances[k] = inst
					needsSave = true
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

	if m.focusedPanel == "terminal" {
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

func (m *model) processCommand(inputStr string) {
	inputStr = strings.TrimSpace(inputStr)
	if inputStr == "" {
		return
	}

	m.addLog(inputStr, false, true)
	parts := strings.SplitN(inputStr, " ", 3)
	action := strings.ToLower(parts[0])

	switch action {
	case "clear":
		m.history = []logLine{}
		return

	case "nchange":
		if len(parts) < 3 {
			m.addLog("Usage: nchange [ID] [NewName]", true, false)
			return
		}
		targetID := strings.ToUpper(parts[1])
		newName := parts[2]

		cfg, _ := config.LoadConfig()
		inst, exists := cfg.Instances[targetID]
		if !exists {
			m.addLog(fmt.Sprintf("Error: No instance found with ID '%s'", targetID), true, false)
			return
		}

		inst.Name = newName
		cfg.Instances[targetID] = inst
		_ = config.SaveConfig(cfg)
		m.addLog(fmt.Sprintf("Successfully renamed %s to '%s'", targetID, newName), false, false)

	case "info":
		if len(parts) < 2 {
			m.addLog("Usage: info [ID]", true, false)
			return
		}
		targetID := strings.ToUpper(parts[1])
		cfg, _ := config.LoadConfig()
		inst, exists := cfg.Instances[targetID]

		if !exists {
			m.addLog(fmt.Sprintf("Error: No instance found with ID '%s'", targetID), true, false)
			return
		}

		m.addLog(fmt.Sprintf("--- INFO: %s (%s) ---", targetID, inst.Name), false, false)
		m.addLog(fmt.Sprintf("State: %s | Port: %d | PID: %d", inst.Status, inst.TargetPort, inst.LastPID), false, false)
		m.addLog(fmt.Sprintf("Path: %s", inst.Path), false, false)
		m.addLog(fmt.Sprintf("Cmd:  %s", inst.StartCommand), false, false)

	case "logs":
		if len(parts) < 2 {
			m.addLog("Usage: logs [ID]", true, false)
			return
		}
		targetID := strings.ToUpper(parts[1])
		homeDir, _ := os.UserHomeDir()
		logPath := filepath.Join(homeDir, ".planb", "logs", fmt.Sprintf("%s_stdout.log", targetID))

		content, err := os.ReadFile(logPath)
		if err != nil {
			m.addLog(fmt.Sprintf("No logs found for %s", targetID), true, false)
			return
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		startIdx := len(lines) - 3
		if startIdx < 0 {
			startIdx = 0
		}

		m.addLog(fmt.Sprintf("--- Last logs for %s ---", targetID), false, false)
		for i := startIdx; i < len(lines); i++ {
			if lines[i] != "" {
				m.addLog(lines[i], false, false)
			}
		}

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

		// --- SYSTEM SAFETY LOCK ---
		if inst.StartCommand == "External Process" && !strings.HasPrefix(inst.Path, "/Users/") {
			m.addLog(fmt.Sprintf("SAFETY LOCK: '%s' is an OS-level process. Access Denied.", targetID), true, false)
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
			"Core Engine: Active (Live)   | Active Instances: %-5d | Ports Bound: %-5d\n"+
			"Host CPU   : %-14s  | Host Memory     : %-14s\n", activeCount, activeCount, m.cpuLoad, m.ramUsed),
	)
	systemBox := baseStyle.Render(systemState)

	// --- SMART SORTING: DEV APPS TOP, SYSTEM APPS BOTTOM ---
	var devKeys []string
	var systemKeys []string

	for k, inst := range m.instances {
		// If it's an external process AND not in your user folder, it's a System app
		isSystem := inst.StartCommand == "External Process" && !strings.HasPrefix(inst.Path, "/Users/")

		if isSystem {
			systemKeys = append(systemKeys, k)
		} else {
			devKeys = append(devKeys, k)
		}
	}
	sort.Strings(devKeys)
	sort.Strings(systemKeys)
	allKeys := append(devKeys, systemKeys...)

	// --- FIXED VIEWPORT SLICING LOGIC (MAX 6 ROWS SHOWN) ---
	maxRowsShown := 6
	totalRows := len(allKeys)

	// Automatically adjust scrolling frame offsets based on cursor position
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+maxRowsShown {
		m.scrollOffset = m.cursor - maxRowsShown + 1
	}

	var rows []string
	if totalRows == 0 {
		rows = append(rows, textStyle.Render("No instances tracked inside system registry."))
	} else {
		endIdx := m.scrollOffset + maxRowsShown
		if endIdx > totalRows {
			endIdx = totalRows
		}

		for i := m.scrollOffset; i < endIdx; i++ {
			id := allKeys[i]
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
			nameLower := strings.ToLower(inst.Name)
			if strings.Contains(cmdLower, "python") || strings.Contains(nameLower, "python") {
				runtime = "Python"
			} else if strings.Contains(cmdLower, "node") || strings.Contains(nameLower, "node") {
				runtime = "Node.js"
			} else if inst.StartCommand == "External Process" {
				runtime = "System"
			}

			rowText := fmt.Sprintf("%-6s | %-18s | %-8s | %-6s | %-8s | %-30s", id, inst.Name, stateStr, portStr, runtime, path)

			// Highlight the row if the table panel has active focus and this row matches the cursor pointer
			if m.focusedPanel == "table" && i == m.cursor {
				rows = append(rows, lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("255")).Bold(true).Render("➔ "+rowText[2:]))
			} else {
				rows = append(rows, textStyle.Render("  "+rowText[2:]))
			}
		}
	}

	tableHeader := textStyle.Bold(true).Render(
		fmt.Sprintf("   %-6s | %-18s | %-8s | %-6s | %-8s | %-30s", "ID", "Name", "State", "Port", "Runtime", "Deployment Path"),
	)

	tableTitle := "[INSTANCE REPOSITORY]"
	if m.focusedPanel == "table" {
		tableTitle = "[INSTANCE REPOSITORY] ● ACTIVE NAVIGATION MODE (Use Up/Down Arrows)"
	}
	tableArea := textStyle.Render(tableTitle + "\n" + tableHeader + "\n" + strings.Repeat("-", 88) + "\n" + strings.Join(rows, "\n"))

	// Dynamic border rendering for the table
	var finalTableBox string
	if m.focusedPanel == "table" {
		finalTableBox = focusedStyle.Render(tableArea)
	} else {
		finalTableBox = baseStyle.Render(tableArea)
	}

	// Build the Fixed-Height Embedded Terminal History
	historyLines := ""
	emptyLinesNeeded := 4 - len(m.history)
	for i := 0; i < emptyLinesNeeded; i++ {
		historyLines += "\n"
	}
	for _, msg := range m.history {
		if msg.isCmd {
			historyLines += echoStyle.Render("PLAN-B > "+msg.text) + "\n"
		} else if msg.isErr {
			historyLines += errorStyle.Render(">> "+msg.text) + "\n"
		} else {
			historyLines += successStyle.Render(">> "+msg.text) + "\n"
		}
	}
	// --- NEW BORDER-INTEGRATED HEADER ---
	terminalTitle := "╭──── TERMINAL ─"
	if m.focusedPanel == "terminal" {
		terminalTitle = "╭──── Terminal ──"
	}

	// FIX: Set total target width to 89 (87 inner width + 2 borders)
	targetTotalWidth := 89
	// Subtract the title length AND the 1 closing "╮" character
	fillCount := targetTotalWidth - len([]rune(terminalTitle)) - 1
	if fillCount < 0 {
		fillCount = 0
	}
	topLine := terminalTitle + strings.Repeat("─", fillCount) + "╮"

	if m.focusedPanel == "terminal" {
		topLine = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Render(topLine)
	} else {
		topLine = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(topLine)
	}

	inputArea := historyLines + m.input.View()

	// Bottom box uses 87 inner width (which + 2 borders equals 89 total)
	var finalInputBox string
	if m.focusedPanel == "terminal" {
		finalInputBox = topLine + "\n" + lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, true, true).
			BorderForeground(lipgloss.Color("69")).
			Width(87).
			Render(inputArea)
	} else {
		finalInputBox = topLine + "\n" + lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, true, true).
			BorderForeground(lipgloss.Color("240")).
			Width(87).
			Render(inputArea)
	}

	footer := textStyle.Render("Controls: [Tab] Switch Panels | [Esc/Ctrl+C] Exit back to OS terminal.")
	footerBox := baseStyle.Render(footer)

	return lipgloss.JoinVertical(lipgloss.Left, header, systemBox, finalTableBox, finalInputBox, footerBox)
}
