package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/1CL0UD/cloudm-cli/liftshift"
)

// =========================================================
// STATE MANAGEMENT
// =========================================================

type Step int

const (
	StepIdle Step = iota
	StepChecking
	StepBackup
	StepDump
	StepRestore
	StepOpSetup
	StepValidation
	StepDone
	StepError
)

func (s Step) String() string {
	switch s {
	case StepChecking:
		return "Checking Requirements & Connections..."
	case StepBackup:
		return "Backing up Target Database..."
	case StepDump:
		return "Dumping Source Database..."
	case StepRestore:
		return "Restoring to Target Database..."
	case StepOpSetup:
		return "Configuring Ownership & Permissions..."
	case StepValidation:
		return "Validating Migration..."
	case StepDone:
		return "Migration Completed Successfully!"
	case StepError:
		return "Migration Failed."
	default:
		return "Initializing..."
	}
}

// =========================================================
// MESSAGES (Events that trigger state changes)
// =========================================================

type errMsg error
type checkDoneMsg struct{}
type backupDoneMsg struct{ file string }
type dumpDoneMsg struct{ structFile, dataFile string }
type restoreDoneMsg struct{}
type opSetupDoneMsg struct{}
type validationDoneMsg struct{}

// =========================================================
// MODEL
// =========================================================

type Model struct {
	Config      *liftshift.Config
	Logger      *liftshift.Logger
	Spinner     spinner.Model
	CurrentStep Step
	Err         error

	// State variables to pass data between steps
	BackupFile string
	StructFile string
	DataFile   string
	StartTime  time.Time
}

func InitialModel(cfg *liftshift.Config, logger *liftshift.Logger) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		Config:      cfg,
		Logger:      logger,
		Spinner:     s,
		CurrentStep: StepIdle,
		StartTime:   time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	// 1. Start Spinner
	// 2. Start the first step (Checking) immediately
	return tea.Batch(
		m.Spinner.Tick,
		runChecksCmd(m.Config, m.Logger),
	)
}

// =========================================================
// UPDATE LOOP
// =========================================================

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	// Handle Key Presses (Quit)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Logger.Close()
			return m, tea.Quit
		}

	// Handle Window Resizing (Pass to Logger Viewport)
	case tea.WindowSizeMsg:
		headerHeight := 4 // Approximate height of header
		m.Logger.UpdateViewport(msg.Width, msg.Height-headerHeight)

	// Handle Spinner Ticks
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	// ------------------------------------------------------
	// STATE MACHINE TRANSITIONS
	// ------------------------------------------------------

	// 1. Checks Done -> Start Backup
	case checkDoneMsg:
		m.CurrentStep = StepBackup
		return m, runBackupCmd(m.Config, m.Logger)

	// 2. Backup Done -> Start Dump
	case backupDoneMsg:
		m.BackupFile = msg.file
		m.CurrentStep = StepDump
		return m, runDumpCmd(m.Config, m.Logger)

	// 3. Dump Done -> Start Restore
	case dumpDoneMsg:
		m.StructFile = msg.structFile
		m.DataFile = msg.dataFile
		m.CurrentStep = StepRestore
		return m, runRestoreCmd(m.Config, m.Logger, m.StructFile, m.DataFile)

	// 4. Restore Done -> Start Op Setup
	case restoreDoneMsg:
		m.CurrentStep = StepOpSetup
		return m, runOpSetupCmd(m.Config, m.Logger)

	// 5. Op Setup Done -> Start Validation
	case opSetupDoneMsg:
		m.CurrentStep = StepValidation
		return m, runValidationCmd(m.Config, m.Logger)

	// 6. Validation Done -> Finished
	case validationDoneMsg:
		m.CurrentStep = StepDone
		duration := time.Since(m.StartTime)
		m.Logger.Log(fmt.Sprintf("\n==== MIGRATION COMPLETED in %v ====", duration))
		return m, nil // No more commands, just wait for user to quit

	// Handle Errors
	case errMsg:
		m.Err = msg
		m.CurrentStep = StepError
		m.Logger.Log(fmt.Sprintf("CRITICAL ERROR: %v", msg))
		return m, nil
	}

	return m, cmd
}

// =========================================================
// VIEW
// =========================================================

func (m Model) View() string {
	header := ""

	if m.Err != nil {
		header = fmt.Sprintf("❌ Error: %v\nPress 'q' to exit.", m.Err)
	} else if m.CurrentStep == StepDone {
		header = "✅ Migration Complete! Press 'q' to exit."
	} else {
		header = fmt.Sprintf("%s %s\nPress 'q' to abort.", m.Spinner.View(), m.CurrentStep.String())
	}

	// Render the logs viewport below the header
	return fmt.Sprintf("%s\n\n%s", header, m.Logger.View())
}

// =========================================================
// COMMANDS (Async wrappers for liftshift functions)
// =========================================================

func runChecksCmd(cfg *liftshift.Config, l *liftshift.Logger) tea.Cmd {
	return func() tea.Msg {
		l.Log("Starting Pre-flight Checks...")
		if err := liftshift.CheckRequirements(); err != nil {
			return errMsg(err)
		}
		if err := liftshift.CheckDatabaseConnections(cfg); err != nil {
			return errMsg(err)
		}
		return checkDoneMsg{}
	}
}

func runBackupCmd(cfg *liftshift.Config, l *liftshift.Logger) tea.Cmd {
	return func() tea.Msg {
		file, err := liftshift.BackupTargetDatabase(cfg)
		if err != nil {
			return errMsg(err)
		}
		return backupDoneMsg{file: file}
	}
}

func runDumpCmd(cfg *liftshift.Config, l *liftshift.Logger) tea.Cmd {
	return func() tea.Msg {
		sFile, dFile, err := liftshift.DumpFromSource(cfg)
		if err != nil {
			return errMsg(err)
		}
		return dumpDoneMsg{structFile: sFile, dataFile: dFile}
	}
}

func runRestoreCmd(cfg *liftshift.Config, l *liftshift.Logger, sFile, dFile string) tea.Cmd {
	return func() tea.Msg {
		if err := liftshift.RestoreToTarget(cfg, sFile, dFile); err != nil {
			return errMsg(err)
		}
		return restoreDoneMsg{}
	}
}

func runOpSetupCmd(cfg *liftshift.Config, l *liftshift.Logger) tea.Cmd {
	return func() tea.Msg {
		if err := liftshift.ConfigureOwnership(cfg); err != nil {
			return errMsg(err)
		}
		return opSetupDoneMsg{}
	}
}

func runValidationCmd(cfg *liftshift.Config, l *liftshift.Logger) tea.Cmd {
	return func() tea.Msg {
		if err := liftshift.ValidateMigration(cfg); err != nil {
			return errMsg(err)
		}
		return validationDoneMsg{}
	}
}

// =========================================================
// MAIN
// =========================================================

func main() {
	// 1. Setup Config (Load from YAML)
	configFile := "config.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1] // Allow specifying config file as command line argument
	}

	config, err := liftshift.LoadConfigWithDefaults(configFile)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", configFile, err)
	}

	// 2. Setup Logger
	logger, err := liftshift.NewLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// 3. Start Program
	p := tea.NewProgram(
		InitialModel(config, logger),
		tea.WithAltScreen(),       // Use full screen
		tea.WithMouseCellMotion(), // Enable mouse scrolling for logs
	)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
