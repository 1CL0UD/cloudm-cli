package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

type Logger struct {
	verbose bool
	logFile *os.File
	noColor bool
}

type LoggerOptions struct {
	Verbose bool
	LogFile string
	NoColor bool
}

// New creates a new logger instance
func New(opts LoggerOptions) (*Logger, error) {
	l := &Logger{
		verbose: opts.Verbose,
		noColor: opts.NoColor,
	}

	if opts.NoColor {
		color.NoColor = true
	}

	if opts.LogFile != "" {
		f, err := os.OpenFile(opts.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		l.logFile = f
	}

	return l, nil
}

// timestamp returns a formatted timestamp
func (l *Logger) timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// writeToFile writes a message to the log file if configured
func (l *Logger) writeToFile(msg string) {
	if l.logFile != nil {
		fmt.Fprintf(l.logFile, "[%s] %s\n", l.timestamp(), msg)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("[%s] %s %s\n", l.timestamp(), cyan("INFO"), formatted)
	l.writeToFile(fmt.Sprintf("INFO: %s", formatted))
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	fmt.Fprintf(os.Stderr, "[%s] %s %s\n", l.timestamp(), red("ERROR"), formatted)
	l.writeToFile(fmt.Sprintf("ERROR: %s", formatted))
}

// Success logs a success message
func (l *Logger) Success(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	fmt.Printf("[%s] %s %s\n", l.timestamp(), green("✓"), formatted)
	l.writeToFile(fmt.Sprintf("SUCCESS: %s", formatted))
}

// Warning logs a warning message
func (l *Logger) Warning(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("[%s] %s %s\n", l.timestamp(), yellow("WARN"), formatted)
	l.writeToFile(fmt.Sprintf("WARN: %s", formatted))
}

// Debug logs a debug message (only if verbose)
func (l *Logger) Debug(msg string, args ...any) {
	if !l.verbose {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	gray := color.New(color.FgHiBlack).SprintFunc()
	fmt.Printf("[%s] %s %s\n", l.timestamp(), gray("DEBUG"), formatted)
	l.writeToFile(fmt.Sprintf("DEBUG: %s", formatted))
}

// Step logs a step in a multi-step process
func (l *Logger) Step(current, total int, msg string) {
	blue := color.New(color.FgBlue, color.Bold).SprintFunc()
	stepLabel := fmt.Sprintf("[%d/%d]", current, total)
	fmt.Printf("[%s] %s %s\n", l.timestamp(), blue(stepLabel), msg)
	l.writeToFile(fmt.Sprintf("STEP %s: %s", stepLabel, msg))
}

// Phase logs the start of a major phase
func (l *Logger) Phase(name string) {
	magenta := color.New(color.FgMagenta, color.Bold).SprintFunc()
	fmt.Printf("\n[%s] %s\n", l.timestamp(), magenta("== "+name+" =="))
	l.writeToFile(fmt.Sprintf("PHASE: %s", name))
}

// DryRun logs a dry run message
func (l *Logger) DryRun(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	fmt.Printf("[%s] %s %s\n", l.timestamp(), yellow("[DRY-RUN]"), formatted)
}

// Close closes the log file if open
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}