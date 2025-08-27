package app

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/ybarbara/pombo/internal/config"
	"github.com/ybarbara/pombo/internal/email"
	"github.com/ybarbara/pombo/internal/ui/pages"
	"github.com/ybarbara/pombo/internal/ui/services"
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

	// Initialize email infrastructure
	emailService := a.createEmailService()

	// Create the main model with email service
	model := pages.NewMainModel(a.config, a.logger, emailService)

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

// createEmailService initializes the email service with all dependencies
func (a *Application) createEmailService() services.EmailService {
	// Create credential store (using a null implementation for now)
	credStore := &nullCredentialStore{}
	
	// Create authentication factory
	authFactory := email.NewAuthProviderFactory(credStore)
	
	// Create email client factory
	clientFactory := email.NewDefaultClientFactory()
	
	// Create connection pool
	poolConfig := &email.PoolConfig{
		MaxConnections:      a.config.Email.ConnectionPool.MaxConnections,
		MaxIdleConnections:  a.config.Email.ConnectionPool.MaxIdleConnections,
		ConnectionLifetime:  a.config.Email.ConnectionPool.ConnectionLifetime,
		IdleTimeout:         a.config.Email.ConnectionPool.IdleTimeout,
		HealthCheckInterval: a.config.Email.ConnectionPool.HealthCheckInterval,
		ConnectTimeout:      a.config.Email.ConnectionPool.ConnectTimeout,
	}
	
	pool := email.NewConnectionPool(poolConfig, authFactory, clientFactory)
	
	// Create email service
	return services.NewEmailService(pool, a.config, a.logger, authFactory, clientFactory)
}

// nullCredentialStore is a simple null implementation for development
type nullCredentialStore struct{}

func (n *nullCredentialStore) Store(ctx context.Context, accountID string, creds *email.Credentials) error {
	return nil
}

func (n *nullCredentialStore) Retrieve(ctx context.Context, accountID string) (*email.Credentials, error) {
	return &email.Credentials{
		Type:     email.AuthTypePassword,
		Username: "test@example.com",
		Password: "password",
	}, nil
}

func (n *nullCredentialStore) Delete(ctx context.Context, accountID string) error {
	return nil
}

func (n *nullCredentialStore) List(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (n *nullCredentialStore) StoreToken(ctx context.Context, accountID string, token *email.OAuthToken) error {
	return nil
}

func (n *nullCredentialStore) RetrieveToken(ctx context.Context, accountID string) (*email.OAuthToken, error) {
	return nil, nil
}

func (n *nullCredentialStore) DeleteToken(ctx context.Context, accountID string) error {
	return nil
}

func (n *nullCredentialStore) IsAvailable(ctx context.Context) bool {
	return true
}

func (n *nullCredentialStore) TestAccess(ctx context.Context) error {
	return nil
}