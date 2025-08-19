package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/ybarbara/pombo/internal/config"
	"github.com/ybarbara/pombo/internal/ui/pages"
)

// Application represents the main application
type Application struct {
	config *config.Config
	logger *log.Logger
}

// New creates a new application instance
func New(cfg *config.Config) *Application {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Prefix:          "POMBO",
	})

	// Set log level from config
	switch cfg.Logging.Level {
	case "debug":
		logger.SetLevel(log.DebugLevel)
	case "info":
		logger.SetLevel(log.InfoLevel)
	case "warn":
		logger.SetLevel(log.WarnLevel)
	case "error":
		logger.SetLevel(log.ErrorLevel)
	default:
		logger.SetLevel(log.InfoLevel)
	}

	return &Application{
		config: cfg,
		logger: logger,
	}
}

// Run starts the application
func (a *Application) Run() error {
	a.logger.Info("Starting POMBO email client")

	// Create the main model
	model := pages.NewMainModel(a.config, a.logger)

	// Create the Bubbletea program
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run the program
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run TUI program: %w", err)
	}

	a.logger.Info("POMBO email client stopped")
	return nil
}