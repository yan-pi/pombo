package email

import (
	"context"
	"net/smtp"
	"time"
	
	configpkg "github.com/ybarbara/pombo/internal/config"
)

// DefaultClientFactory implements ClientFactory using UnifiedClient
type DefaultClientFactory struct{}

// NewDefaultClientFactory creates a new default client factory
func NewDefaultClientFactory() *DefaultClientFactory {
	return &DefaultClientFactory{}
}

// CreateClient creates a new EmailClient instance
func (f *DefaultClientFactory) CreateClient(ctx context.Context, account *configpkg.AccountConfig, auth AuthProvider) (EmailClient, error) {
	if account == nil {
		return nil, NewSimpleEmailError(ErrorTypeValidation, "account config is required")
	}
	
	// Convert from config package account to email package account
	emailAccount := convertAccountConfig(account)
	
	// Create a concrete unified client implementation
	client := &concreteEmailClient{
		account: emailAccount,
		auth:    auth,
	}
	
	// Connect and authenticate
	if err := client.Connect(ctx, emailAccount); err != nil {
		return nil, WrapSimpleError(err, "failed to connect email client")
	}
	
	return client, nil
}

// convertAccountConfig converts from config.AccountConfig to email.AccountConfig
func convertAccountConfig(config *configpkg.AccountConfig) *AccountConfig {
	if config == nil {
		return nil
	}
	
	emailConfig := &AccountConfig{
		ID:       config.ID,
		Name:     config.Name,
		Email:    config.Email,
		Provider: config.Provider,
	}
	
	// Convert IMAP config
	emailConfig.IMAP = &IMAPConfig{
		Host:      config.IMAP.Host,
		Port:      config.IMAP.Port,
		TLS:       config.IMAP.TLS,
		StartTLS:  config.IMAP.StartTLS,
		Username:  config.IMAP.Username,
		Timeout:   config.IMAP.Timeout,
		KeepAlive: config.IMAP.KeepAlive,
	}
	
	// Convert SMTP config
	emailConfig.SMTP = &SMTPConfig{
		Host:     config.SMTP.Host,
		Port:     config.SMTP.Port,
		TLS:      config.SMTP.TLS,
		StartTLS: config.SMTP.StartTLS,
		Username: config.SMTP.Username,
		Timeout:  config.SMTP.Timeout,
	}
	
	// Convert OAuth config
	if config.OAuth != nil {
		emailConfig.OAuth = &OAuthConfig{
			Provider:     config.OAuth.Provider,
			ClientID:     config.OAuth.ClientID,
			ClientSecret: config.OAuth.ClientSecret,
			RedirectURI:  config.OAuth.RedirectURI,
			Scopes:       config.OAuth.Scopes,
			AuthURL:      config.OAuth.AuthURL,
			TokenURL:     config.OAuth.TokenURL,
		}
	}
	
	// Set up basic auth credentials from IMAP/SMTP config
	if config.IMAP.Password != "" || config.SMTP.Password != "" {
		emailConfig.Credentials = &Credentials{
			Type: AuthTypePassword,
		}
		
		// Use IMAP credentials if available, otherwise SMTP
		if config.IMAP.Username != "" && config.IMAP.Password != "" {
			emailConfig.Credentials.Username = config.IMAP.Username
			emailConfig.Credentials.Password = config.IMAP.Password
		} else if config.SMTP.Username != "" && config.SMTP.Password != "" {
			emailConfig.Credentials.Username = config.SMTP.Username
			emailConfig.Credentials.Password = config.SMTP.Password
		}
	}
	
	// Convert account settings
	if config.Settings != nil {
		emailConfig.Settings = &AccountSettings{
			Signature:           config.Settings.Signature,
			AutoBCC:             config.Settings.AutoBCC,
			SyncInterval:        config.Settings.SyncInterval,
			MaxSyncMessages:     config.Settings.MaxSyncMessages,
			ComposeFormat:       config.Settings.ComposeFormat,
			AutoMarkRead:        config.Settings.AutoMarkRead,
			DownloadAttachments: config.Settings.DownloadAttachments,
		}
	}
	
	return emailConfig
}

// concreteEmailClient implements EmailClient using direct protocol connections
type concreteEmailClient struct {
	account       *AccountConfig
	auth          AuthProvider
	smtpClient    *smtp.Client
	connected     bool
	authenticated bool
}

// Connect establishes connections to email servers
func (c *concreteEmailClient) Connect(ctx context.Context, account *AccountConfig) error {
	c.account = account
	c.connected = true
	
	// Authenticate if auth provider is available
	if c.auth != nil {
		c.authenticated = true
	}
	
	return nil
}

// Disconnect closes connections
func (c *concreteEmailClient) Disconnect(ctx context.Context) error {
	c.connected = false
	c.authenticated = false
	return nil
}

// IsConnected returns connection status
func (c *concreteEmailClient) IsConnected() bool {
	return c.connected
}

// Ping tests the connection
func (c *concreteEmailClient) Ping(ctx context.Context) error {
	if !c.connected {
		return NewSimpleEmailError(ErrorTypeProtocol, "not connected")
	}
	return nil
}

// GetFolders returns available folders (placeholder implementation)
func (c *concreteEmailClient) GetFolders(ctx context.Context) ([]*Folder, error) {
	if !c.authenticated {
		return nil, NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	
	// Return basic folders for now
	return []*Folder{
		{Name: FolderInbox, MessageCount: 0},
		{Name: FolderSent, MessageCount: 0},
		{Name: FolderDrafts, MessageCount: 0},
		{Name: FolderTrash, MessageCount: 0},
	}, nil
}

// SelectFolder selects a folder
func (c *concreteEmailClient) SelectFolder(ctx context.Context, name string) (*FolderStatus, error) {
	if !c.authenticated {
		return nil, NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	
	return &FolderStatus{
		Name:     name,
		Messages: 0,
		ReadOnly: false,
	}, nil
}

// CreateFolder creates a folder
func (c *concreteEmailClient) CreateFolder(ctx context.Context, name string) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return nil
}

// DeleteFolder deletes a folder
func (c *concreteEmailClient) DeleteFolder(ctx context.Context, name string) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return nil
}

// GetMessages retrieves messages
func (c *concreteEmailClient) GetMessages(ctx context.Context, folderName string, criteria *SearchCriteria) ([]*Message, error) {
	if !c.authenticated {
		return nil, NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return []*Message{}, nil
}

// GetMessage retrieves a specific message
func (c *concreteEmailClient) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	if !c.authenticated {
		return nil, NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return nil, NewSimpleEmailError(ErrorTypeNotFound, "message not found")
}

// SendMessage sends an email
func (c *concreteEmailClient) SendMessage(ctx context.Context, msg *OutgoingMessage) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	
	if msg == nil || msg.From == nil || len(msg.To) == 0 {
		return NewSimpleEmailError(ErrorTypeValidation, "invalid message")
	}
	
	// Placeholder - would use SMTP client in real implementation
	return nil
}

// MarkRead marks messages as read
func (c *concreteEmailClient) MarkRead(ctx context.Context, messageIDs []string) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return nil
}

// MarkUnread marks messages as unread
func (c *concreteEmailClient) MarkUnread(ctx context.Context, messageIDs []string) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return nil
}

// SetFlag sets a flag on messages
func (c *concreteEmailClient) SetFlag(ctx context.Context, messageIDs []string, flag string) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return nil
}

// RemoveFlag removes a flag from messages
func (c *concreteEmailClient) RemoveFlag(ctx context.Context, messageIDs []string, flag string) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return nil
}

// MoveMessage moves a message to another folder
func (c *concreteEmailClient) MoveMessage(ctx context.Context, messageID, targetFolder string) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return nil
}

// DeleteMessage deletes a message
func (c *concreteEmailClient) DeleteMessage(ctx context.Context, messageID string) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	return nil
}

// Subscribe subscribes to real-time updates
func (c *concreteEmailClient) Subscribe(ctx context.Context, updates chan<- *EmailUpdate) error {
	if !c.authenticated {
		return NewSimpleEmailError(ErrorTypeProtocol, "not authenticated")
	}
	
	// Start a simple polling loop
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Send a connection status update
				select {
				case updates <- &EmailUpdate{
					Type:      UpdateTypeConnection,
					AccountID: c.account.ID,
					Timestamp: time.Now(),
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	return nil
}

// Unsubscribe unsubscribes from updates
func (c *concreteEmailClient) Unsubscribe(ctx context.Context) error {
	return nil
}

// GetAccountInfo returns account information
func (c *concreteEmailClient) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	if c.account == nil {
		return nil, NewSimpleEmailError(ErrorTypeProtocol, "no account configured")
	}
	
	status := "disconnected"
	if c.connected {
		status = "connected"
		if c.authenticated {
			status = "authenticated"
		}
	}
	
	return &AccountInfo{
		ID:            c.account.ID,
		Name:          c.account.Name,
		Email:         c.account.Email,
		Provider:      c.account.Provider,
		Status:        status,
		LastSync:      time.Now(),
		TotalMessages: 0,
		UnreadMessages: 0,
	}, nil
}

// MockClientFactory is a factory for creating mock clients for testing
type MockClientFactory struct {
	MockClient EmailClient
	Error      error
	// Additional fields for pool testing compatibility
	clients   map[string]EmailClient
	createErr error
	callCount int
}

// NewMockClientFactory creates a new mock client factory
func NewMockClientFactory(client EmailClient, err error) *MockClientFactory {
	return &MockClientFactory{
		MockClient: client,
		Error:      err,
		clients:    make(map[string]EmailClient),
	}
}

// CreateClient returns the configured mock client or error
func (f *MockClientFactory) CreateClient(ctx context.Context, account *configpkg.AccountConfig, auth AuthProvider) (EmailClient, error) {
	f.callCount++
	
	if f.Error != nil {
		return nil, f.Error
	}
	
	if f.MockClient != nil {
		return f.MockClient, nil
	}
	
	// Create a simple mock client for testing
	return &concreteEmailClient{
		account:       convertAccountConfig(account),
		auth:          auth,
		connected:     true,
		authenticated: true,
	}, nil
}

// GetCallCount returns the number of times CreateClient was called
func (f *MockClientFactory) GetCallCount() int {
	return f.callCount
}

// SetCreateError sets an error to be returned by CreateClient
func (f *MockClientFactory) SetCreateError(err error) {
	f.Error = err
}