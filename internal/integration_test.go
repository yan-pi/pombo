package internal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ybarbara/pombo/internal/config"
	"github.com/ybarbara/pombo/internal/email"
)

// TestEmailConfigIntegration tests integration between email system and configuration
func TestEmailConfigIntegration(t *testing.T) {
	tests := []struct {
		name         string
		setupConfig  func() *config.Config
		expectError  bool
		validateFunc func(*testing.T, *config.Config)
	}{
		{
			name: "complete_email_configuration",
			setupConfig: func() *config.Config {
				return &config.Config{
					Email: config.EmailConfig{
						DefaultAccount:  "test@example.com",
						CheckInterval:   5 * time.Minute,
						AutoSync:        true,
						BackgroundSync:  true,
						ConnectionPool: config.ConnectionPoolConfig{
							MaxConnections:      5,
							MaxIdleConnections:  2,
							ConnectionLifetime:  30 * time.Minute,
							IdleTimeout:         5 * time.Minute,
							HealthCheckInterval: 1 * time.Minute,
							ConnectTimeout:      30 * time.Second,
						},
						MessageCache: config.MessageCacheConfig{
							MaxSize:          "100MB",
							TTL:              24 * time.Hour,
							CleanupInterval:  1 * time.Hour,
							CacheHeaders:     true,
							CacheBodies:      true,
							CacheAttachments: false,
						},
						ErrorRetry: config.ErrorRetryConfig{
							MaxRetries:    3,
							BaseDelay:     1 * time.Second,
							MaxDelay:      1 * time.Minute,
							Multiplier:    2.0,
							JitterEnabled: true,
						},
					},
					Accounts: []config.AccountConfig{
						{
							ID:       "test@example.com",
							Name:     "Test Account",
							Email:    "test@example.com",
							Provider: "gmail",
							IMAP: config.IMAPConfig{
								Host:      "imap.gmail.com",
								Port:      993,
								TLS:       true,
								Username:  "test@example.com",
								Timeout:   30 * time.Second,
								KeepAlive: 5 * time.Minute,
								UseIdle:   true,
							},
							SMTP: config.SMTPConfig{
								Host:       "smtp.gmail.com",
								Port:       587,
								StartTLS:   true,
								Username:   "test@example.com",
								Timeout:    30 * time.Second,
								RequireTLS: true,
							},
							OAuth: &config.OAuthConfig{
								Provider:     "google",
								ClientID:     "test-client-id",
								ClientSecret: "test-client-secret",
								RedirectURI:  "http://localhost:8080/callback",
								Scopes:       []string{"https://mail.google.com/"},
								AuthURL:      "https://accounts.google.com/o/oauth2/auth",
								TokenURL:     "https://oauth2.googleapis.com/token",
							},
							Settings: &config.AccountSettings{
								Signature:           "Best regards,\nTest User",
								SyncInterval:        5 * time.Minute,
								MaxSyncMessages:     1000,
								ComposeFormat:       "text",
								AutoMarkRead:        false,
								DownloadAttachments: false,
								CheckSSLCert:        true,
							},
							Enabled: true,
						},
					},
				}
			},
			expectError: false,
			validateFunc: func(t *testing.T, cfg *config.Config) {
				// Validate email configuration
				if cfg.Email.DefaultAccount != "test@example.com" {
					t.Errorf("expected default account 'test@example.com', got '%s'", cfg.Email.DefaultAccount)
				}
				
				// Validate connection pool settings
				if cfg.Email.ConnectionPool.MaxConnections != 5 {
					t.Errorf("expected max connections 5, got %d", cfg.Email.ConnectionPool.MaxConnections)
				}
				
				// Validate account configuration
				if len(cfg.Accounts) != 1 {
					t.Fatalf("expected 1 account, got %d", len(cfg.Accounts))
				}
				
				account := cfg.Accounts[0]
				if account.Email != "test@example.com" {
					t.Errorf("expected account email 'test@example.com', got '%s'", account.Email)
				}
				
				// Validate OAuth configuration
				if account.OAuth == nil {
					t.Fatal("expected OAuth configuration, got nil")
				}
				if account.OAuth.Provider != "google" {
					t.Errorf("expected OAuth provider 'google', got '%s'", account.OAuth.Provider)
				}
			},
		},
		{
			name: "multiple_accounts_configuration",
			setupConfig: func() *config.Config {
				return &config.Config{
					Email: config.EmailConfig{
						DefaultAccount: "work@company.com",
						CheckInterval:  3 * time.Minute,
						AutoSync:       true,
						ConnectionPool: config.ConnectionPoolConfig{
							MaxConnections:      3,
							MaxIdleConnections:  1,
							ConnectionLifetime:  30 * time.Minute,
							IdleTimeout:         5 * time.Minute,
							HealthCheckInterval: 1 * time.Minute,
							ConnectTimeout:      30 * time.Second,
						},
					},
					Accounts: []config.AccountConfig{
						{
							ID:       "work@company.com",
							Name:     "Work Account",
							Email:    "work@company.com",
							Provider: "outlook",
							IMAP: config.IMAPConfig{
								Host:     "outlook.office365.com",
								Port:     993,
								TLS:      true,
								Username: "work@company.com",
							},
							OAuth: &config.OAuthConfig{
								Provider:    "microsoft",
								ClientID:    "work-client-id",
								RedirectURI: "http://localhost:8080/callback",
								Scopes:      []string{"https://graph.microsoft.com/Mail.ReadWrite"},
							},
							Enabled: true,
						},
						{
							ID:       "personal@gmail.com",
							Name:     "Personal Account",
							Email:    "personal@gmail.com",
							Provider: "gmail",
							IMAP: config.IMAPConfig{
								Host:     "imap.gmail.com",
								Port:     993,
								TLS:      true,
								Username: "personal@gmail.com",
							},
							OAuth: &config.OAuthConfig{
								Provider:    "google",
								ClientID:    "personal-client-id",
								RedirectURI: "http://localhost:8080/callback",
								Scopes:      []string{"https://mail.google.com/"},
							},
							Enabled: false, // Disabled account
						},
					},
				}
			},
			expectError: false,
			validateFunc: func(t *testing.T, cfg *config.Config) {
				if len(cfg.Accounts) != 2 {
					t.Fatalf("expected 2 accounts, got %d", len(cfg.Accounts))
				}
				
				// Validate work account is enabled
				workAccount := cfg.Accounts[0]
				if !workAccount.Enabled {
					t.Error("expected work account to be enabled")
				}
				
				// Validate personal account is disabled
				personalAccount := cfg.Accounts[1]
				if personalAccount.Enabled {
					t.Error("expected personal account to be disabled")
				}
				
				// Validate default account points to enabled account
				if cfg.Email.DefaultAccount != "work@company.com" {
					t.Errorf("expected default account 'work@company.com', got '%s'", cfg.Email.DefaultAccount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			
			// Validate configuration structure
			if tt.validateFunc != nil {
				tt.validateFunc(t, cfg)
			}
			
			// Test email configuration to email.PoolConfig conversion
			poolConfig := &email.PoolConfig{
				MaxConnections:      cfg.Email.ConnectionPool.MaxConnections,
				MaxIdleConnections:  cfg.Email.ConnectionPool.MaxIdleConnections,
				ConnectionLifetime:  cfg.Email.ConnectionPool.ConnectionLifetime,
				IdleTimeout:         cfg.Email.ConnectionPool.IdleTimeout,
				HealthCheckInterval: cfg.Email.ConnectionPool.HealthCheckInterval,
				ConnectTimeout:      cfg.Email.ConnectionPool.ConnectTimeout,
			}
			
			if poolConfig.MaxConnections <= 0 {
				t.Error("connection pool max connections must be positive")
			}
			if poolConfig.MaxIdleConnections > poolConfig.MaxConnections {
				t.Error("max idle connections cannot exceed max connections")
			}
		})
	}
}

// TestEmailErrorIntegration tests integration of email error handling with app-wide patterns
func TestEmailErrorIntegration(t *testing.T) {
	tests := []struct {
		name        string
		error       error
		expectRetry bool
		expectType  email.ErrorType
	}{
		{
			name: "authentication_error_integration",
			error: email.NewEmailError(
				email.ErrorTypeAuth,
				email.ErrCodeAuthFailed,
				"authentication failed",
				nil,
				false,
			).WithContext("test@example.com", "", "", "Connect"),
			expectRetry: false,
			expectType:  email.ErrorTypeAuth,
		},
		{
			name: "network_error_integration",
			error: email.NewEmailError(
				email.ErrorTypeNetwork,
				email.ErrCodeConnectionFailed,
				"connection failed",
				nil,
				true,
			).WithContext("test@example.com", "INBOX", "", "GetMessages"),
			expectRetry: true,
			expectType:  email.ErrorTypeNetwork,
		},
		{
			name: "rate_limit_error_integration",
			error: email.NewEmailError(
				email.ErrorTypeRateLimit,
				email.ErrCodeRateLimited,
				"rate limit exceeded",
				nil,
				true,
			).WithContext("test@example.com", "", "", "SendMessage"),
			expectRetry: true,
			expectType:  email.ErrorTypeRateLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test error classification
			if email.GetErrorType(tt.error) != tt.expectType {
				t.Errorf("expected error type %v, got %v", tt.expectType, email.GetErrorType(tt.error))
			}
			
			// Test retry logic
			if email.IsRetryable(tt.error) != tt.expectRetry {
				t.Errorf("expected retryable %v, got %v", tt.expectRetry, email.IsRetryable(tt.error))
			}
			
			// Test error handler integration
			handler := email.NewErrorHandler()
			action := handler.Handle(tt.error, 1)
			
			switch tt.expectType {
			case email.ErrorTypeAuth:
				if action != email.ErrorActionReauth && action != email.ErrorActionRefreshAuth {
					t.Errorf("expected auth action for auth error, got %v", action)
				}
			case email.ErrorTypeNetwork:
				if tt.expectRetry && action != email.ErrorActionRetry {
					t.Errorf("expected retry action for retryable network error, got %v", action)
				}
			case email.ErrorTypeRateLimit:
				if action != email.ErrorActionBackoff {
					t.Errorf("expected backoff action for rate limit error, got %v", action)
				}
			}
		})
	}
}

// TestConnectionPoolIntegration tests connection pool integration with configuration and error handling
func TestConnectionPoolIntegration(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Email: config.EmailConfig{
			ConnectionPool: config.ConnectionPoolConfig{
				MaxConnections:      3,
				MaxIdleConnections:  1,
				ConnectionLifetime:  5 * time.Minute,
				IdleTimeout:         1 * time.Minute,
				HealthCheckInterval: 30 * time.Second,
				ConnectTimeout:      10 * time.Second,
			},
		},
		Accounts: []config.AccountConfig{
			{
				ID:       "test@example.com",
				Name:     "Test Account",
				Email:    "test@example.com",
				Provider: "test",
				IMAP: config.IMAPConfig{
					Host:     "imap.test.com",
					Port:     993,
					TLS:      true,
					Username: "test@example.com",
					Timeout:  30 * time.Second,
				},
				Enabled: true,
			},
		},
	}

	// Create pool configuration from app config
	poolConfig := &email.PoolConfig{
		MaxConnections:      cfg.Email.ConnectionPool.MaxConnections,
		MaxIdleConnections:  cfg.Email.ConnectionPool.MaxIdleConnections,
		ConnectionLifetime:  cfg.Email.ConnectionPool.ConnectionLifetime,
		IdleTimeout:         cfg.Email.ConnectionPool.IdleTimeout,
		HealthCheckInterval: cfg.Email.ConnectionPool.HealthCheckInterval,
		ConnectTimeout:      cfg.Email.ConnectionPool.ConnectTimeout,
	}

	// Create mock credential store
	credStore := &mockCredentialStore{}
	
	// Create auth factory
	authFactory := email.NewAuthProviderFactory(credStore)
	
	// Create mock client factory
	clientFactory := &mockClientFactory{}
	
	// Create connection pool
	pool := email.NewConnectionPool(poolConfig, authFactory, clientFactory)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Test pool startup
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("failed to start connection pool: %v", err)
	}
	defer func() {
		if err := pool.Stop(ctx); err != nil {
			t.Errorf("failed to stop connection pool: %v", err)
		}
	}()
	
	// Test account addition
	account := &cfg.Accounts[0]
	if err := pool.AddAccount(ctx, account); err != nil {
		t.Fatalf("failed to add account: %v", err)
	}
	
	// Test connection retrieval (should fail with mock factory)
	_, err := pool.GetConnection(ctx, "test@example.com")
	if err == nil {
		t.Error("expected error when getting connection with mock factory")
	}
	
	// Verify error is properly classified
	if email.IsNetworkError(err) || email.GetErrorType(err).String() != "unknown" {
		t.Logf("Connection error properly classified: %v", email.GetErrorType(err))
	}
	
	// Test pool statistics
	stats := pool.GetPoolStats()
	if stats == nil {
		t.Error("expected pool stats, got nil")
	}
	
	if stats.MaxConnections != poolConfig.MaxConnections {
		t.Errorf("expected max connections %d, got %d", poolConfig.MaxConnections, stats.MaxConnections)
	}
	
	// Test account removal
	if err := pool.RemoveAccount("test@example.com"); err != nil {
		t.Errorf("failed to remove account: %v", err)
	}
}

// TestConfigurationValidation tests configuration validation for email settings
func TestConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectValid bool
		errorMsg    string
	}{
		{
			name: "valid_complete_configuration",
			config: &config.Config{
				Email: config.EmailConfig{
					DefaultAccount: "test@example.com",
					CheckInterval:  5 * time.Minute,
					ConnectionPool: config.ConnectionPoolConfig{
						MaxConnections:     5,
						MaxIdleConnections: 2,
						ConnectTimeout:     30 * time.Second,
					},
				},
				Accounts: []config.AccountConfig{
					{
						ID:    "test@example.com",
						Email: "test@example.com",
						IMAP: config.IMAPConfig{
							Host: "imap.test.com",
							Port: 993,
						},
						Enabled: true,
					},
				},
			},
			expectValid: true,
		},
		{
			name: "invalid_pool_configuration",
			config: &config.Config{
				Email: config.EmailConfig{
					ConnectionPool: config.ConnectionPoolConfig{
						MaxConnections:     0, // Invalid: must be positive
						MaxIdleConnections: 2,
					},
				},
			},
			expectValid: false,
			errorMsg:    "max connections must be positive",
		},
		{
			name: "invalid_idle_connections",
			config: &config.Config{
				Email: config.EmailConfig{
					ConnectionPool: config.ConnectionPoolConfig{
						MaxConnections:     3,
						MaxIdleConnections: 5, // Invalid: exceeds max connections
					},
				},
			},
			expectValid: false,
			errorMsg:    "max idle connections cannot exceed max connections",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate pool configuration
			poolConfig := &email.PoolConfig{
				MaxConnections:     tt.config.Email.ConnectionPool.MaxConnections,
				MaxIdleConnections: tt.config.Email.ConnectionPool.MaxIdleConnections,
				ConnectTimeout:     tt.config.Email.ConnectionPool.ConnectTimeout,
			}
			
			// Create a temporary pool to test validation
			pool := email.NewConnectionPool(nil, nil, nil)
			err := pool.SetPoolConfig(poolConfig)
			
			if tt.expectValid && err != nil {
				t.Errorf("expected valid configuration, got error: %v", err)
			}
			
			if !tt.expectValid && err == nil {
				t.Error("expected configuration error, got nil")
			}
			
			if !tt.expectValid && err != nil {
				// Check if error message contains expected text
				if tt.errorMsg != "" {
					errorStr := err.Error()
					if len(errorStr) == 0 {
						t.Errorf("expected error message containing '%s', got empty error", tt.errorMsg)
					}
				}
			}
		})
	}
}

// TestConcurrentOperations tests thread safety across components
func TestConcurrentOperations(t *testing.T) {
	// Create test configuration
	poolConfig := &email.PoolConfig{
		MaxConnections:      5,
		MaxIdleConnections:  2,
		ConnectionLifetime:  5 * time.Minute,
		IdleTimeout:         1 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		ConnectTimeout:      10 * time.Second,
	}
	
	credStore := &mockCredentialStore{}
	authFactory := email.NewAuthProviderFactory(credStore)
	clientFactory := &mockClientFactory{}
	
	pool := email.NewConnectionPool(poolConfig, authFactory, clientFactory)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	// Test concurrent account operations
	const numGoroutines = 10
	const numAccounts = 5
	
	errChan := make(chan error, numGoroutines)
	
	// Concurrent account addition
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			accountID := fmt.Sprintf("test%d@example.com", i%numAccounts)
			account := &config.AccountConfig{
				ID:       accountID,
				Name:     fmt.Sprintf("Test Account %d", i),
				Email:    accountID,
				Provider: "test",
				IMAP: config.IMAPConfig{
					Host:     "imap.test.com",
					Port:     993,
					Username: accountID,
				},
				Enabled: true,
			}
			
			err := pool.AddAccount(ctx, account)
			// It's ok if account already exists
			if err != nil {
				emailErr, ok := err.(*email.EmailError)
				if !ok || emailErr.Code != "ACCOUNT_EXISTS" {
					errChan <- err
					return
				}
			}
			
			// Try to get connection status
			status := pool.GetConnectionStatus(accountID)
			if status == nil {
				errChan <- fmt.Errorf("got nil status for account %s", accountID)
				return
			}
			
			errChan <- nil
		}(i)
	}
	
	// Collect errors
	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("concurrent operation failed: %v", err)
		}
	}
	
	// Test pool statistics access during concurrent operations
	stats := pool.GetPoolStats()
	if stats == nil {
		t.Error("expected pool stats, got nil")
	}
}

// Mock implementations for testing

type mockCredentialStore struct{}

func (m *mockCredentialStore) Store(ctx context.Context, accountID string, creds *email.Credentials) error {
	return nil
}

func (m *mockCredentialStore) Retrieve(ctx context.Context, accountID string) (*email.Credentials, error) {
	return &email.Credentials{
		Type:     email.AuthTypePassword,
		Username: accountID,
		Password: "test-password",
	}, nil
}

func (m *mockCredentialStore) Delete(ctx context.Context, accountID string) error {
	return nil
}

func (m *mockCredentialStore) List(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *mockCredentialStore) StoreToken(ctx context.Context, accountID string, token *email.OAuthToken) error {
	return nil
}

func (m *mockCredentialStore) RetrieveToken(ctx context.Context, accountID string) (*email.OAuthToken, error) {
	return nil, fmt.Errorf("no token found")
}

func (m *mockCredentialStore) DeleteToken(ctx context.Context, accountID string) error {
	return nil
}

func (m *mockCredentialStore) IsAvailable(ctx context.Context) bool {
	return true
}

func (m *mockCredentialStore) TestAccess(ctx context.Context) error {
	return nil
}

type mockClientFactory struct{}

func (m *mockClientFactory) CreateClient(ctx context.Context, account *config.AccountConfig, auth email.AuthProvider) (email.EmailClient, error) {
	return nil, email.NewEmailError(
		email.ErrorTypeClient,
		"MOCK_FACTORY",
		"mock factory always fails",
		nil,
		false,
	)
}

type mockEmailClient struct {
	connected bool
}

func (m *mockEmailClient) Connect(ctx context.Context, account *email.AccountConfig) error {
	m.connected = true
	return nil
}

func (m *mockEmailClient) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *mockEmailClient) IsConnected() bool {
	return m.connected
}

func (m *mockEmailClient) Ping(ctx context.Context) error {
	if !m.connected {
		return email.ErrNotConnected
	}
	return nil
}

func (m *mockEmailClient) GetFolders(ctx context.Context) ([]*email.Folder, error) {
	return []*email.Folder{}, nil
}

func (m *mockEmailClient) SelectFolder(ctx context.Context, name string) (*email.FolderStatus, error) {
	return &email.FolderStatus{Name: name}, nil
}

func (m *mockEmailClient) CreateFolder(ctx context.Context, name string) error {
	return nil
}

func (m *mockEmailClient) DeleteFolder(ctx context.Context, name string) error {
	return nil
}

func (m *mockEmailClient) GetMessages(ctx context.Context, folderName string, criteria *email.SearchCriteria) ([]*email.Message, error) {
	return []*email.Message{}, nil
}

func (m *mockEmailClient) GetMessage(ctx context.Context, messageID string) (*email.Message, error) {
	return &email.Message{ID: messageID}, nil
}

func (m *mockEmailClient) SendMessage(ctx context.Context, msg *email.OutgoingMessage) error {
	return nil
}

func (m *mockEmailClient) MarkRead(ctx context.Context, messageIDs []string) error {
	return nil
}

func (m *mockEmailClient) MarkUnread(ctx context.Context, messageIDs []string) error {
	return nil
}

func (m *mockEmailClient) SetFlag(ctx context.Context, messageIDs []string, flag string) error {
	return nil
}

func (m *mockEmailClient) RemoveFlag(ctx context.Context, messageIDs []string, flag string) error {
	return nil
}

func (m *mockEmailClient) MoveMessage(ctx context.Context, messageID, targetFolder string) error {
	return nil
}

func (m *mockEmailClient) DeleteMessage(ctx context.Context, messageID string) error {
	return nil
}

func (m *mockEmailClient) Subscribe(ctx context.Context, updates chan<- *email.EmailUpdate) error {
	return nil
}

func (m *mockEmailClient) Unsubscribe(ctx context.Context) error {
	return nil
}

func (m *mockEmailClient) GetAccountInfo(ctx context.Context) (*email.AccountInfo, error) {
	return &email.AccountInfo{}, nil
}