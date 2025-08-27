package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/ybarbara/pombo/internal/config"
	"github.com/ybarbara/pombo/internal/email"
)

// Mock implementations for testing

type MockConnectionManager struct {
	mock.Mock
}

func (m *MockConnectionManager) GetConnection(ctx context.Context, accountID string) (email.EmailClient, error) {
	args := m.Called(ctx, accountID)
	return args.Get(0).(email.EmailClient), args.Error(1)
}

func (m *MockConnectionManager) ReleaseConnection(accountID string) error {
	args := m.Called(accountID)
	return args.Error(0)
}

func (m *MockConnectionManager) CloseConnection(accountID string) error {
	args := m.Called(accountID)
	return args.Error(0)
}

func (m *MockConnectionManager) CloseAll() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConnectionManager) HealthCheck(ctx context.Context, accountID string) error {
	args := m.Called(ctx, accountID)
	return args.Error(0)
}

func (m *MockConnectionManager) GetConnectionStatus(accountID string) *email.ConnectionStatus {
	args := m.Called(accountID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*email.ConnectionStatus)
}

func (m *MockConnectionManager) GetPoolStats() *email.PoolStats {
	args := m.Called()
	return args.Get(0).(*email.PoolStats)
}

func (m *MockConnectionManager) SetPoolConfig(config *email.PoolConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

type MockEmailClient struct {
	mock.Mock
}

func (m *MockEmailClient) Connect(ctx context.Context, account *email.AccountConfig) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockEmailClient) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEmailClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockEmailClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEmailClient) GetFolders(ctx context.Context) ([]*email.Folder, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*email.Folder), args.Error(1)
}

func (m *MockEmailClient) SelectFolder(ctx context.Context, name string) (*email.FolderStatus, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*email.FolderStatus), args.Error(1)
}

func (m *MockEmailClient) CreateFolder(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockEmailClient) DeleteFolder(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockEmailClient) GetMessages(ctx context.Context, folderName string, criteria *email.SearchCriteria) ([]*email.Message, error) {
	args := m.Called(ctx, folderName, criteria)
	return args.Get(0).([]*email.Message), args.Error(1)
}

func (m *MockEmailClient) GetMessage(ctx context.Context, messageID string) (*email.Message, error) {
	args := m.Called(ctx, messageID)
	return args.Get(0).(*email.Message), args.Error(1)
}

func (m *MockEmailClient) SendMessage(ctx context.Context, msg *email.OutgoingMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockEmailClient) MarkRead(ctx context.Context, messageIDs []string) error {
	args := m.Called(ctx, messageIDs)
	return args.Error(0)
}

func (m *MockEmailClient) MarkUnread(ctx context.Context, messageIDs []string) error {
	args := m.Called(ctx, messageIDs)
	return args.Error(0)
}

func (m *MockEmailClient) SetFlag(ctx context.Context, messageIDs []string, flag string) error {
	args := m.Called(ctx, messageIDs, flag)
	return args.Error(0)
}

func (m *MockEmailClient) RemoveFlag(ctx context.Context, messageIDs []string, flag string) error {
	args := m.Called(ctx, messageIDs, flag)
	return args.Error(0)
}

func (m *MockEmailClient) MoveMessage(ctx context.Context, messageID, targetFolder string) error {
	args := m.Called(ctx, messageID, targetFolder)
	return args.Error(0)
}

func (m *MockEmailClient) DeleteMessage(ctx context.Context, messageID string) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockEmailClient) Subscribe(ctx context.Context, updates chan<- *email.EmailUpdate) error {
	args := m.Called(ctx, updates)
	return args.Error(0)
}

func (m *MockEmailClient) Unsubscribe(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEmailClient) GetAccountInfo(ctx context.Context) (*email.AccountInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).(*email.AccountInfo), args.Error(1)
}

// Simple mock credential store for testing
type TestCredentialStore struct{}

func (t *TestCredentialStore) Store(ctx context.Context, accountID string, creds *email.Credentials) error {
	return nil
}

func (t *TestCredentialStore) Retrieve(ctx context.Context, accountID string) (*email.Credentials, error) {
	return &email.Credentials{
		Type:     email.AuthTypePassword,
		Username: "test@example.com",
		Password: "password",
	}, nil
}

func (t *TestCredentialStore) Delete(ctx context.Context, accountID string) error {
	return nil
}

func (t *TestCredentialStore) List(ctx context.Context) ([]string, error) {
	return []string{"test1"}, nil
}

func (t *TestCredentialStore) StoreToken(ctx context.Context, accountID string, token *email.OAuthToken) error {
	return nil
}

func (t *TestCredentialStore) RetrieveToken(ctx context.Context, accountID string) (*email.OAuthToken, error) {
	return nil, nil
}

func (t *TestCredentialStore) DeleteToken(ctx context.Context, accountID string) error {
	return nil
}

func (t *TestCredentialStore) IsAvailable(ctx context.Context) bool {
	return true
}

func (t *TestCredentialStore) TestAccess(ctx context.Context) error {
	return nil
}

// Test Suite

func TestEmailService_Lifecycle(t *testing.T) {
	// Setup
	mockPool := &MockConnectionManager{}
	credStore := &TestCredentialStore{}
	authFactory := email.NewAuthProviderFactory(credStore)
	clientFactory := email.NewDefaultClientFactory()
	
	cfg := &config.Config{
		Accounts: []config.AccountConfig{},
	}
	
	logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.DebugLevel})
	
	service := NewEmailService(mockPool, cfg, logger, authFactory, clientFactory)
	
	// Test start
	ctx := context.Background()
	
	t.Run("Start Service", func(t *testing.T) {
		assert.False(t, service.IsRunning())
		
		err := service.Start(ctx)
		assert.NoError(t, err)
		assert.True(t, service.IsRunning())
	})
	
	t.Run("Stop Service", func(t *testing.T) {
		err := service.Stop()
		assert.NoError(t, err)
		assert.False(t, service.IsRunning())
	})
}

func TestEmailService_AccountManagement(t *testing.T) {
	// Setup
	mockPool := &MockConnectionManager{}
	credStore := &TestCredentialStore{}
	authFactory := email.NewAuthProviderFactory(credStore)
	clientFactory := email.NewDefaultClientFactory()
	mockClient := &MockEmailClient{}
	
	cfg := &config.Config{}
	logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.DebugLevel})
	
	service := NewEmailService(mockPool, cfg, logger, authFactory, clientFactory).(*EmailServiceImpl)
	
	// Start service
	ctx := context.Background()
	service.Start(ctx)
	
	t.Run("Add Account", func(t *testing.T) {
		account := &config.AccountConfig{
			ID:    "test1",
			Name:  "Test Account",
			Email: "test@example.com",
			IMAP: config.IMAPConfig{
				Host: "imap.example.com",
				Port: 993,
				TLS:  true,
			},
		}
		
		// Mock expectations for account monitoring and background operations
		mockPool.On("GetConnection", mock.Anything, "test1").Return(mockClient, nil).Maybe()
		mockPool.On("ReleaseConnection", "test1").Return(nil).Maybe()
		
		// Mock expectations for client operations
		mockClient.On("Connect", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockClient.On("Disconnect", mock.Anything).Return(nil).Maybe()
		mockClient.On("IsConnected").Return(true).Maybe()
		mockClient.On("Ping", mock.Anything).Return(nil).Maybe()
		mockClient.On("Subscribe", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockClient.On("Unsubscribe", mock.Anything).Return(nil).Maybe()
		mockClient.On("GetFolders", mock.Anything).Return([]*email.Folder{}, nil).Maybe()
		mockClient.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return([]*email.Message{}, nil).Maybe()
		
		err := service.AddAccount(account)
		assert.NoError(t, err)
		
		accounts := service.GetAccounts()
		assert.Len(t, accounts, 1)
		assert.Equal(t, "test1", accounts[0].ID)
		
		// Give a moment for background processes to start
		time.Sleep(100 * time.Millisecond)
	})
	
	t.Run("Switch Account", func(t *testing.T) {
		err := service.SwitchAccount("test1")
		assert.NoError(t, err)
		
		current := service.GetCurrentAccount()
		assert.NotNil(t, current)
		assert.Equal(t, "test1", current.ID)
	})
	
	t.Run("Remove Account", func(t *testing.T) {
		mockPool.On("CloseConnection", "test1").Return(nil)
		
		err := service.RemoveAccount("test1")
		assert.NoError(t, err)
		
		accounts := service.GetAccounts()
		assert.Len(t, accounts, 0)
		
		mockPool.AssertExpectations(t)
	})
	
	service.Stop()
}

func TestEmailService_MessageOperations(t *testing.T) {
	// Setup
	mockPool := &MockConnectionManager{}
	credStore := &TestCredentialStore{}
	authFactory := email.NewAuthProviderFactory(credStore)
	clientFactory := email.NewDefaultClientFactory()
	mockClient := &MockEmailClient{}
	
	cfg := &config.Config{}
	logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.DebugLevel})
	
	service := NewEmailService(mockPool, cfg, logger, authFactory, clientFactory).(*EmailServiceImpl)
	
	// Setup service state
	service.Start(context.Background())
	service.updateState(func(state *ServiceState) {
		state.CurrentAccount = &AccountInfo{
			ID:    "test1",
			Email: "test@example.com",
		}
	})
	
	t.Run("Get Messages", func(t *testing.T) {
		mockMessages := []*email.Message{
			{
				ID:      "msg1",
				Subject: "Test Message 1",
				From:    &email.Address{Address: "sender@example.com"},
				Date:    time.Now(),
				Size:    1024,
				IsRead:  false,
			},
			{
				ID:      "msg2",
				Subject: "Test Message 2",
				From:    &email.Address{Address: "sender2@example.com"},
				Date:    time.Now().Add(-time.Hour),
				Size:    2048,
				IsRead:  true,
			},
		}
		
		mockPool.On("GetConnection", mock.Anything, "test1").Return(mockClient, nil)
		mockPool.On("ReleaseConnection", "test1").Return(nil)
		mockClient.On("GetMessages", mock.Anything, "INBOX", mock.Anything).Return(mockMessages, nil)
		
		messages, err := service.GetMessages("test1", "INBOX", 10)
		assert.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, "Test Message 1", messages[0].Subject)
		assert.False(t, messages[0].IsRead)
		assert.True(t, messages[1].IsRead)
		
		mockPool.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})
	
	t.Run("Mark Message Read", func(t *testing.T) {
		mockPool.On("GetConnection", mock.Anything, "test1").Return(mockClient, nil)
		mockPool.On("ReleaseConnection", "test1").Return(nil)
		mockClient.On("MarkRead", mock.Anything, []string{"msg1"}).Return(nil)
		
		err := service.MarkRead("test1", []string{"msg1"})
		assert.NoError(t, err)
		
		mockPool.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})
	
	t.Run("Send Message", func(t *testing.T) {
		outgoingMsg := &OutgoingMessage{
			From: AddressInfo{
				Address: "test@example.com",
				Display: "test@example.com",
			},
			To: []AddressInfo{
				{Address: "recipient@example.com", Display: "recipient@example.com"},
			},
			Subject: "Test Subject",
			Body:    "Test body content",
		}
		
		mockPool.On("GetConnection", mock.Anything, "test1").Return(mockClient, nil)
		mockPool.On("ReleaseConnection", "test1").Return(nil)
		mockClient.On("SendMessage", mock.Anything, mock.AnythingOfType("*email.OutgoingMessage")).Return(nil)
		
		err := service.SendMessage("test1", outgoingMsg)
		assert.NoError(t, err)
		
		mockPool.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})
	
	service.Stop()
}

func TestEmailService_RealTimeUpdates(t *testing.T) {
	// Setup
	mockPool := &MockConnectionManager{}
	credStore := &TestCredentialStore{}
	authFactory := email.NewAuthProviderFactory(credStore)
	clientFactory := email.NewDefaultClientFactory()
	
	cfg := &config.Config{}
	logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.DebugLevel})
	
	service := NewEmailService(mockPool, cfg, logger, authFactory, clientFactory)
	
	service.Start(context.Background())
	
	t.Run("Update Channel Available", func(t *testing.T) {
		updateChan := service.GetUpdateChannel()
		assert.NotNil(t, updateChan)
	})
	
	t.Run("Service State Updates", func(t *testing.T) {
		state := service.GetState()
		assert.NotNil(t, state)
		assert.NotNil(t, state.Accounts)
		assert.NotNil(t, state.Messages)
		assert.NotNil(t, state.Folders)
	})
	
	service.Stop()
}

func TestEmailService_ErrorHandling(t *testing.T) {
	// Setup
	mockPool := &MockConnectionManager{}
	credStore := &TestCredentialStore{}
	authFactory := email.NewAuthProviderFactory(credStore)
	clientFactory := email.NewDefaultClientFactory()
	
	cfg := &config.Config{}
	logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.DebugLevel})
	
	service := NewEmailService(mockPool, cfg, logger, authFactory, clientFactory)
	
	t.Run("Service Not Running", func(t *testing.T) {
		err := service.AddAccount(&config.AccountConfig{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service is not running")
	})
	
	t.Run("Invalid Account Config", func(t *testing.T) {
		service.Start(context.Background())
		
		err := service.AddAccount(&config.AccountConfig{}) // Missing required fields
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account configuration")
		
		service.Stop()
	})
}

func TestEmailService_Performance(t *testing.T) {
	// Setup
	mockPool := &MockConnectionManager{}
	credStore := &TestCredentialStore{}
	authFactory := email.NewAuthProviderFactory(credStore)
	clientFactory := email.NewDefaultClientFactory()
	
	cfg := &config.Config{}
	logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.DebugLevel})
	
	service := NewEmailService(mockPool, cfg, logger, authFactory, clientFactory)
	
	service.Start(context.Background())
	
	t.Run("Statistics Available", func(t *testing.T) {
		mockPool.On("GetPoolStats").Return(&email.PoolStats{
			ActiveConnections: 2,
			IdleConnections:   1,
			AverageLatency:    time.Millisecond * 100,
			ErrorRate:         0.01,
		})
		
		stats := service.GetStatistics()
		assert.Equal(t, 2, stats.ActiveConnections)
		assert.Equal(t, time.Millisecond*100, stats.AverageResponseTime)
		assert.Equal(t, 0.01, stats.ErrorRate)
		
		mockPool.AssertExpectations(t)
	})
	
	t.Run("Connection Status", func(t *testing.T) {
		mockPool.On("GetConnectionStatus", mock.Anything).Return(&email.ConnectionStatus{
			AccountID:   "test1",
			Connected:   true,
			LastPing:    time.Now(),
			LastError:   nil,
		})
		
		// Add a mock account to test connection status
		service.(*EmailServiceImpl).updateState(func(state *ServiceState) {
			state.Accounts = []AccountInfo{
				{ID: "test1", Connected: true},
			}
		})
		
		status := service.GetConnectionStatus()
		assert.Contains(t, status, "test1")
		assert.True(t, status["test1"].Connected)
		
		mockPool.AssertExpectations(t)
	})
	
	service.Stop()
}

// Benchmark tests for performance validation

func BenchmarkEmailService_StateUpdates(b *testing.B) {
	mockPool := &MockConnectionManager{}
	credStore := &TestCredentialStore{}
	authFactory := email.NewAuthProviderFactory(credStore)
	clientFactory := email.NewDefaultClientFactory()
	
	cfg := &config.Config{}
	logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.ErrorLevel}) // Reduce log noise
	
	service := NewEmailService(mockPool, cfg, logger, authFactory, clientFactory).(*EmailServiceImpl)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		service.updateState(func(state *ServiceState) {
			state.LastUpdate = time.Now()
		})
	}
}

func BenchmarkEmailService_GetState(b *testing.B) {
	mockPool := &MockConnectionManager{}
	credStore := &TestCredentialStore{}
	authFactory := email.NewAuthProviderFactory(credStore)
	clientFactory := email.NewDefaultClientFactory()
	
	cfg := &config.Config{}
	logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.ErrorLevel})
	
	service := NewEmailService(mockPool, cfg, logger, authFactory, clientFactory)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = service.GetState()
	}
}