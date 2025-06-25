package ui

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"dbtop/monitor/stats"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// SortField represents the field to sort by
type SortField int

const (
	SortByID SortField = iota
	SortByUser
	SortByHost
	SortByDatabase
	SortByTime
	SortByState
)

// UI represents the terminal user interface
type UI struct {
	instanceName    string
	dbType          string
	refreshInterval time.Duration
	grid            *termui.Grid
	processList     *widgets.List
	statsTable      *widgets.Table
	infoBox         *widgets.Paragraph
	helpBox         *widgets.Paragraph
	sortField       SortField
	sortDescending  bool
	processes       []stats.ProcessInfo
}

// NewUI creates a new UI instance
func NewUI(instanceName, dbType string, refreshInterval time.Duration) *UI {
	if err := termui.Init(); err != nil {
		log.Fatal("Failed to initialize termui:", err)
	}

	ui := &UI{
		instanceName:    instanceName,
		dbType:          dbType,
		refreshInterval: refreshInterval,
		sortField:       SortByTime,
		sortDescending:  true,
	}

	ui.setupWidgets()
	ui.setupGrid()

	return ui
}

// setupWidgets initializes the UI widgets
func (ui *UI) setupWidgets() {
	// Process list
	ui.processList = widgets.NewList()
	ui.processList.Title = "Active Processes (Press 's' to sort, 'r' to reverse, 'h' for help)"
	ui.processList.TextStyle = termui.NewStyle(termui.ColorYellow)
	ui.processList.BorderStyle = termui.NewStyle(termui.ColorBlue)

	// Stats table
	ui.statsTable = widgets.NewTable()
	ui.statsTable.Title = "Database Statistics"
	ui.statsTable.TextStyle = termui.NewStyle(termui.ColorWhite)
	ui.statsTable.BorderStyle = termui.NewStyle(termui.ColorGreen)
	ui.statsTable.RowSeparator = true

	// Info box
	ui.infoBox = widgets.NewParagraph()
	ui.infoBox.Title = "Connection Info"
	ui.infoBox.TextStyle = termui.NewStyle(termui.ColorCyan)
	ui.infoBox.BorderStyle = termui.NewStyle(termui.ColorMagenta)

	// Help box
	ui.helpBox = widgets.NewParagraph()
	ui.helpBox.Title = "Controls"
	ui.helpBox.Text = "q: quit | s: sort | r: reverse | h: help | +/-: refresh rate"
	ui.helpBox.TextStyle = termui.NewStyle(termui.ColorWhite)
	ui.helpBox.BorderStyle = termui.NewStyle(termui.ColorRed)
}

// setupGrid arranges the widgets in a grid layout
func (ui *UI) setupGrid() {
	ui.grid = termui.NewGrid()
	termWidth, termHeight := termui.TerminalDimensions()
	ui.grid.SetRect(0, 0, termWidth, termHeight)

	ui.grid.Set(
		termui.NewRow(0.25,
			termui.NewCol(0.5, ui.infoBox),
			termui.NewCol(0.5, ui.statsTable),
		),
		termui.NewRow(0.05, ui.helpBox),
		termui.NewRow(0.7, ui.processList),
	)
}

// sortProcesses sorts the processes based on the current sort field
func (ui *UI) sortProcesses() {
	sort.Slice(ui.processes, func(i, j int) bool {
		var result bool
		switch ui.sortField {
		case SortByID:
			result = ui.processes[i].ID < ui.processes[j].ID
		case SortByUser:
			result = ui.processes[i].User < ui.processes[j].User
		case SortByHost:
			result = ui.processes[i].Host < ui.processes[j].Host
		case SortByDatabase:
			result = ui.processes[i].Database < ui.processes[j].Database
		case SortByTime:
			result = ui.processes[i].Time < ui.processes[j].Time
		case SortByState:
			result = ui.processes[i].State < ui.processes[j].State
		}
		if ui.sortDescending {
			return !result
		}
		return result
	})
}

// Update refreshes the UI with new statistics
func (ui *UI) Update(stats *stats.DatabaseStats) {
	// Update info box
	ui.infoBox.Text = fmt.Sprintf(
		"Instance: %s\nType: %s\nUptime: %s\nActive Connections: %d\nRefresh: %v",
		ui.instanceName,
		ui.dbType,
		stats.Uptime.String(),
		stats.ActiveConnections,
		ui.refreshInterval,
	)

	// Update stats table
	ui.statsTable.Rows = [][]string{
		{"Metric", "Value"},
		{"Total Connections", strconv.FormatInt(stats.TotalConnections, 10)},
		{"Queries/Second", fmt.Sprintf("%.2f", stats.QueriesPerSecond)},
		{"Slow Queries", strconv.FormatInt(stats.SlowQueries, 10)},
		{"Threads Running", strconv.FormatInt(stats.Threads.Running, 10)},
		{"Threads Connected", strconv.FormatInt(stats.Threads.Connected, 10)},
		{"Threads Sleeping", strconv.FormatInt(stats.Threads.Sleeping, 10)},
	}

	// Store and sort processes
	ui.processes = stats.Processes
	ui.sortProcesses()

	// Update process list with dynamic height
	_, termHeight := termui.TerminalDimensions()
	maxProcesses := int(float64(termHeight) * 0.6) // Use 60% of terminal height for processes

	if len(ui.processes) > maxProcesses {
		ui.processes = ui.processes[:maxProcesses]
	}

	var processLines []string
	for _, process := range ui.processes {
		line := fmt.Sprintf("[%d] %s@%s - %s (%s) - %s",
			process.ID, process.User, process.Host, process.Database, process.Command, process.State)
		if process.Time > 0 {
			line += fmt.Sprintf(" [%ds]", process.Time)
		}
		processLines = append(processLines, line)
	}
	ui.processList.Rows = processLines

	// Render the UI
	termui.Clear()
	termui.Render(ui.grid)
}

// Close cleans up the UI
func (ui *UI) Close() {
	termui.Close()
}

// HandleKey handles keyboard input
func (ui *UI) HandleKey(key string) bool {
	switch key {
	case "q", "<C-c>":
		return false // Exit
	case "s":
		// Cycle through sort fields
		ui.sortField = (ui.sortField + 1) % 6
		ui.sortProcesses()
	case "r":
		// Reverse sort order
		ui.sortDescending = !ui.sortDescending
		ui.sortProcesses()
	case "+":
		// Increase refresh rate
		if ui.refreshInterval > 500*time.Millisecond {
			ui.refreshInterval -= 500 * time.Millisecond
		}
	case "-":
		// Decrease refresh rate
		ui.refreshInterval += 500 * time.Millisecond
	}
	return true // Continue
}
