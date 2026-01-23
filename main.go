package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/1CL0UD/cloudm-cli/configs"
	"github.com/1CL0UD/cloudm-cli/pkg/configloader"
)

type StateType int

const (
	StateChecking StateType = iota
	StateDumping
	StateRestoring
	StateValidating
	StateComplete
	StateError
)

type Model struct {
	Config     *configs.Config
	CurrentStep StateType
	Logs       []string
	Error      error
	Quitting   bool
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.Quitting {
		return "Exiting cloudm-cli...\n"
	}

	status := "Running cloudm-cli..."
	switch m.CurrentStep {
	case StateChecking:
		status = "Checking database connections..."
	case StateDumping:
		status = "Dumping source database..."
	case StateRestoring:
		status = "Restoring to target database..."
	case StateValidating:
		status = "Validating migration..."
	case StateComplete:
		status = "Migration completed successfully!"
	case StateError:
		status = "Error occurred during migration!"
	}

	view := "CloudM-CLI: PostgreSQL Migration Tool\n\n"
	view += "Status: " + status + "\n\n"

	if m.Error != nil {
		view += "Error: " + m.Error.Error() + "\n"
	}

	view += "Press 'q' to quit\n"
	return view
}

func main() {
	// Load configuration from environment variables
	config := configloader.LoadConfigFromEnv()

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Printf("Configuration validation error: %v", err)
		os.Exit(1)
	}

	p := tea.NewProgram(Model{
		Config:    config,
		Logs:      make([]string, 0),
	})

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}