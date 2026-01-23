package liftshift

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Logger handles logging to both file and UI
type Logger struct {
	logFile  *os.File
	Viewport viewport.Model
	logs     []string
}

// NewLogger creates a new logger instance
func NewLogger() (*Logger, error) {
	timestamp := time.Now().Format("20060102_150405")

	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create the .cloudm-cli/logs directory
	logsDir := filepath.Join(homeDir, ".cloudm-cli", "logs")
	err = os.MkdirAll(logsDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	filename := filepath.Join(logsDir, fmt.Sprintf("migration_%s.log", timestamp))

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	vp := viewport.New(80, 20) // Default dimensions, will be adjusted by Bubble Tea

	logger := &Logger{
		logFile:  file,
		Viewport: vp,
		logs:     make([]string, 0),
	}

	return logger, nil
}

// Log writes a message to both the file and adds it to the logs
func (l *Logger) Log(message string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s", timestamp, message)

	// Write to file
	_, err := l.logFile.WriteString(logLine + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	// Add to logs slice
	l.logs = append(l.logs, logLine)

	// Update viewport content
	var sb strings.Builder
	for _, log := range l.logs {
		sb.WriteString(log + "\n")
	}
	l.Viewport.SetContent(sb.String())

	return nil
}

// Close closes the log file
func (l *Logger) Close() error {
	return l.logFile.Close()
}

// View returns the viewport view
func (l *Logger) View() string {
	return l.Viewport.View()
}

// UpdateViewport updates the viewport dimensions
func (l *Logger) UpdateViewport(width, height int) {
	l.Viewport.Width = width
	l.Viewport.Height = height
}

// LogCmd is a tea.Cmd that logs a message
func LogCmd(logger *Logger, message string) tea.Cmd {
	return func() tea.Msg {
		if logger != nil {
			_ = logger.Log(message)
		}
		return nil
	}
}