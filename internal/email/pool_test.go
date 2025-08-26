package email

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	configpkg "github.com/ybarbara/pombo/internal/config"
)

// MockEmailClient implements EmailClient interface for testing
type MockEmailClient struct {
	connected     bool
	connectErr    error
	disconnectErr error
	pingErr       error
	mu            sync.RWMutex
	connectionID  string
	connectCalls  int
	pingCalls     int
}

func NewMockEmailClient(id string) *MockEmailClient {
	return &MockEmailClient{
		connectionID: id,
		connected:    false,
	}
}

func (m *MockEmailClient) Connect(ctx context.Context, account *AccountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectCalls++
	
	if m.connectErr != nil {
		return m.connectErr
	}
	
	m.connected = true
	return nil
}

func (m *MockEmailClient) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.disconnectErr != nil {
		return m.disconnectErr
	}
	
	m.connected = false
	return nil
}

func (m *MockEmailClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

func (m *MockEmailClient) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingCalls++
	
	if m.pingErr != nil {
		return m.pingErr
	}
	
	if !m.connected {
		return NewEmailError(ErrorTypeNetwork, ErrCodeConnectionFailed, "not connected", nil, true)
	}
	
	return nil
}

// Implement required interface methods (minimal implementations for testing)
func (m *MockEmailClient) GetFolders(ctx context.Context) ([]*Folder, error) {
	return nil, nil
}

func (m *MockEmailClient) SelectFolder(ctx context.Context, name string) (*FolderStatus, error) {
	return nil, nil
}

func (m *MockEmailClient) CreateFolder(ctx context.Context, name string) error {
	return nil
}

func (m *MockEmailClient) DeleteFolder(ctx context.Context, name string) error {
	return nil
}

func (m *MockEmailClient) GetMessages(ctx context.Context, folderName string, criteria *SearchCriteria) ([]*Message, error) {
	return nil, nil
}

func (m *MockEmailClient) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	return nil, nil
}

func (m *MockEmailClient) SendMessage(ctx context.Context, msg *OutgoingMessage) error {
	return nil
}

func (m *MockEmailClient) MarkRead(ctx context.Context, messageIDs []string) error {
	return nil
}

func (m *MockEmailClient) MarkUnread(ctx context.Context, messageIDs []string) error {
	return nil
}

func (m *MockEmailClient) MoveMessages(ctx context.Context, messageIDs []string, targetFolder string) error {
	return nil
}

func (m *MockEmailClient) DeleteMessages(ctx context.Context, messageIDs []string) error {
	return nil
}

func (m *MockEmailClient) DeleteMessage(ctx context.Context, messageID string) error {
	return nil
}

func (m *MockEmailClient) SetFlag(ctx context.Context, messageIDs []string, flag string) error {
	return nil
}

func (m *MockEmailClient) RemoveFlag(ctx context.Context, messageIDs []string, flag string) error {
	return nil
}

func (m *MockEmailClient) MoveMessage(ctx context.Context, messageID, targetFolder string) error {
	return nil
}

func (m *MockEmailClient) Subscribe(ctx context.Context, updates chan<- *EmailUpdate) error {
	return nil
}

func (m *MockEmailClient) Unsubscribe(ctx context.Context) error {
	return nil
}

func (m *MockEmailClient) SearchMessages(ctx context.Context, criteria *SearchCriteria) ([]*Message, error) {
	return nil, nil
}

func (m *MockEmailClient) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	return nil, nil
}

// Test helpers
func (m *MockEmailClient) SetConnectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectErr = err
}

func (m *MockEmailClient) SetDisconnectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectErr = err
}

func (m *MockEmailClient) SetPingError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingErr = err
}

func (m *MockEmailClient) GetConnectCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connectCalls
}

func (m *MockEmailClient) GetPingCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pingCalls
}

func (m *MockEmailClient) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

// Removed duplicate MockClientFactory - using the one from client_factory.go

// Test helpers
func createTestPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConnections:      5,
		MaxIdleConnections:  2,
		ConnectionLifetime:  30 * time.Minute,
		IdleTimeout:         5 * time.Minute,
		HealthCheckInterval: 1 * time.Minute,
		ConnectTimeout:      30 * time.Second,
	}
}

func createTestAccountConfig(id string) *configpkg.AccountConfig {
	return &configpkg.AccountConfig{
		ID:    id,
		Name:  fmt.Sprintf("Test Account %s", id),
		Email: fmt.Sprintf("%s@example.com", id),
		IMAP: configpkg.IMAPConfig{
			Host:     "imap.example.com",
			Port:     993,
			TLS:      true,
			Username: fmt.Sprintf("%s@example.com", id),
			Password: "password123",
		},
		SMTP: configpkg.SMTPConfig{
			Host:     "smtp.example.com",
			Port:     587,
			StartTLS: true,
			Username: fmt.Sprintf("%s@example.com", id),
			Password: "password123",
		},
	}
}

func TestConnectionPool_GetConnection(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	// Start the pool
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	t.Run("get connection for new account", func(t *testing.T) {
		accountConfig := createTestAccountConfig("test1")
		
		// Add account to pool
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		// Get connection
		client, err := pool.GetConnection(ctx, "test1")
		if err != nil {
			t.Errorf("GetConnection() error = %v", err)
			return
		}
		
		if client == nil {
			t.Error("GetConnection() returned nil client")
		}
		
		if !client.IsConnected() {
			t.Error("Client should be connected")
		}
		
		// Verify factory was called
		if clientFactory.GetCallCount() != 1 {
			t.Errorf("Factory call count = %d, want 1", clientFactory.GetCallCount())
		}
	})
	
	t.Run("get connection from pool not running", func(t *testing.T) {
		stoppedPool := NewConnectionPool(config, authFactory, clientFactory)
		
		_, err := stoppedPool.GetConnection(ctx, "test1")
		if err == nil {
			t.Error("GetConnection() should fail when pool is not running")
		}
		
		var emailErr *EmailError
		if !errors.As(err, &emailErr) {
			t.Error("Should return EmailError")
			return
		}
		
		if emailErr.Code != "POOL_NOT_RUNNING" {
			t.Errorf("Error code = %s, want POOL_NOT_RUNNING", emailErr.Code)
		}
	})
	
	t.Run("get connection for non-existent account", func(t *testing.T) {
		_, err := pool.GetConnection(ctx, "nonexistent")
		if err == nil {
			t.Error("GetConnection() should fail for non-existent account")
		}
		
		var emailErr *EmailError
		if !errors.As(err, &emailErr) {
			t.Error("Should return EmailError")
			return
		}
		
		if emailErr.Code != "ACCOUNT_POOL_ERROR" {
			t.Errorf("Error code = %s, want ACCOUNT_POOL_ERROR", emailErr.Code)
		}
	})
}

func TestConnectionPool_ReleaseConnection(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	t.Run("release connection successfully", func(t *testing.T) {
		accountConfig := createTestAccountConfig("test2")
		
		// Add account and get connection
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		client, err := pool.GetConnection(ctx, "test2")
		if err != nil {
			t.Fatalf("Failed to get connection: %v", err)
		}
		
		if client == nil {
			t.Fatal("Client is nil")
		}
		
		// Release connection
		if err := pool.ReleaseConnection("test2"); err != nil {
			t.Errorf("ReleaseConnection() error = %v", err)
		}
	})
	
	t.Run("release connection for non-existent account", func(t *testing.T) {
		err := pool.ReleaseConnection("nonexistent")
		if err == nil {
			t.Error("ReleaseConnection() should fail for non-existent account")
		}
		
		var emailErr *EmailError
		errors.As(err, &emailErr)
		if emailErr.Code != "ACCOUNT_NOT_FOUND" {
			t.Errorf("Error code = %s, want ACCOUNT_NOT_FOUND", emailErr.Code)
		}
	})
}

func TestConnectionPool_HealthCheck(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	t.Run("health check successful", func(t *testing.T) {
		accountConfig := createTestAccountConfig("test3")
		
		// Add account and get connection
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		client, err := pool.GetConnection(ctx, "test3")
		if err != nil {
			t.Fatalf("Failed to get connection: %v", err)
		}
		
		if client == nil {
			t.Fatal("Client is nil")
		}
		
		// Perform health check
		if err := pool.HealthCheck(ctx, "test3"); err != nil {
			t.Errorf("HealthCheck() error = %v", err)
		}
	})
	
	t.Run("health check for non-existent account", func(t *testing.T) {
		err := pool.HealthCheck(ctx, "nonexistent")
		if err == nil {
			t.Error("HealthCheck() should fail for non-existent account")
		}
		
		var emailErr *EmailError
		errors.As(err, &emailErr)
		if emailErr.Code != "ACCOUNT_NOT_FOUND" {
			t.Errorf("Error code = %s, want ACCOUNT_NOT_FOUND", emailErr.Code)
		}
	})
	
	t.Run("health check with failing ping", func(t *testing.T) {
		accountConfig := createTestAccountConfig("test4")
		
		// Set up client factory to return clients that fail ping
		failingFactory := NewMockClientFactory(nil, nil)
		failingPool := NewConnectionPool(config, authFactory, failingFactory)
		
		if err := failingPool.Start(ctx); err != nil {
			t.Fatalf("Failed to start pool: %v", err)
		}
		defer failingPool.Stop(ctx)
		
		if err := failingPool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		client, err := failingPool.GetConnection(ctx, "test4")
		if err != nil {
			t.Fatalf("Failed to get connection: %v", err)
		}
		
		// Make ping fail
		if mockClient, ok := client.(*MockEmailClient); ok {
			mockClient.SetPingError(NewEmailError(ErrorTypeNetwork, ErrCodeConnectionFailed, "ping failed", nil, true))
		}
		
		// Health check should fail
		if err := failingPool.HealthCheck(ctx, "test4"); err == nil {
			t.Error("HealthCheck() should fail when ping fails")
		}
	})
}

func TestConnectionPool_ConfigUpdate(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	t.Run("update config successfully", func(t *testing.T) {
		newConfig := &PoolConfig{
			MaxConnections:      10,
			MaxIdleConnections:  5,
			ConnectionLifetime:  60 * time.Minute,
			IdleTimeout:         10 * time.Minute,
			HealthCheckInterval: 2 * time.Minute,
			ConnectTimeout:      60 * time.Second,
		}
		
		if err := pool.SetPoolConfig(newConfig); err != nil {
			t.Errorf("SetPoolConfig() error = %v", err)
		}
		
		// Verify config was updated
		stats := pool.GetPoolStats()
		if stats.MaxConnections != 10 {
			t.Errorf("MaxConnections = %d, want 10", stats.MaxConnections)
		}
	})
	
	t.Run("update config with nil", func(t *testing.T) {
		err := pool.SetPoolConfig(nil)
		if err == nil {
			t.Error("SetPoolConfig() should fail with nil config")
		}
		
		var emailErr *EmailError
		errors.As(err, &emailErr)
		if emailErr.Code != "INVALID_CONFIG" {
			t.Errorf("Error code = %s, want INVALID_CONFIG", emailErr.Code)
		}
	})
	
	t.Run("update config with invalid values", func(t *testing.T) {
		invalidConfigs := []*PoolConfig{
			{MaxConnections: 0},  // Invalid max connections
			{MaxConnections: 5, MaxIdleConnections: -1},  // Invalid idle connections
			{MaxConnections: 5, MaxIdleConnections: 10},  // Idle > max
		}
		
		for i, invalidConfig := range invalidConfigs {
			err := pool.SetPoolConfig(invalidConfig)
			if err == nil {
				t.Errorf("SetPoolConfig() should fail for invalid config %d", i)
			}
			
			var emailErr *EmailError
			errors.As(err, &emailErr)
			if emailErr.Code != "INVALID_CONFIG" {
				t.Errorf("Error code = %s, want INVALID_CONFIG for config %d", emailErr.Code, i)
			}
		}
	})
}

func TestConnectionPool_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	config.MaxConnections = 20  // Increase limit for concurrent test
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	// Add test account
	accountConfig := createTestAccountConfig("concurrent")
	if err := pool.AddAccount(ctx, accountConfig); err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}
	
	const numGoroutines = 10
	const operationsPerGoroutine = 5
	
	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines*operationsPerGoroutine)
	
	// Start multiple goroutines that get and release connections
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				// Get connection
				client, err := pool.GetConnection(ctx, "concurrent")
				if err != nil {
					errorChan <- fmt.Errorf("goroutine %d operation %d: GetConnection failed: %v", goroutineID, j, err)
					continue
				}
				
				if client == nil {
					errorChan <- fmt.Errorf("goroutine %d operation %d: GetConnection returned nil client", goroutineID, j)
					continue
				}
				
				// Simulate some work
				time.Sleep(10 * time.Millisecond)
				
				// Release connection
				if err := pool.ReleaseConnection("concurrent"); err != nil {
					errorChan <- fmt.Errorf("goroutine %d operation %d: ReleaseConnection failed: %v", goroutineID, j, err)
				}
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
	close(errorChan)
	
	// Check for errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		t.Errorf("Concurrent access test failed with %d errors:", len(errors))
		for _, err := range errors {
			t.Errorf("  %v", err)
		}
	}
}

func TestConnectionPool_ConnectionReuse(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	config.MaxConnections = 2  // Limit to 2 connections to test reuse
	config.MaxIdleConnections = 1
	
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	// Add test account
	accountConfig := createTestAccountConfig("reuse")
	if err := pool.AddAccount(ctx, accountConfig); err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}
	
	t.Run("connection reuse", func(t *testing.T) {
		// Get first connection
		_, err := pool.GetConnection(ctx, "reuse")
		if err != nil {
			t.Fatalf("Failed to get first connection: %v", err)
		}
		
		// Release it
		if err := pool.ReleaseConnection("reuse"); err != nil {
			t.Fatalf("Failed to release first connection: %v", err)
		}
		
		initialCallCount := clientFactory.GetCallCount()
		
		// Get second connection - should reuse the idle connection
		client2, err := pool.GetConnection(ctx, "reuse")
		if err != nil {
			t.Fatalf("Failed to get second connection: %v", err)
		}
		
		// Factory should not have been called again (connection reused)
		if clientFactory.GetCallCount() > initialCallCount {
			t.Error("Connection should have been reused, but factory was called again")
		}
		
		if client2 == nil {
			t.Error("Second client should not be nil")
		}
	})
}

func TestConnectionPool_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	
	t.Run("client factory error", func(t *testing.T) {
		clientFactory := NewMockClientFactory(nil, nil)
		clientFactory.SetCreateError(NewEmailError(ErrorTypeClient, ErrCodeConnectionFailed, "factory error", nil, false))
		
		pool := NewConnectionPool(config, authFactory, clientFactory)
		
		if err := pool.Start(ctx); err != nil {
			t.Fatalf("Failed to start pool: %v", err)
		}
		defer pool.Stop(ctx)
		
		// Add test account
		accountConfig := createTestAccountConfig("error")
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		// Try to get connection - should fail
		_, err := pool.GetConnection(ctx, "error")
		if err == nil {
			t.Error("GetConnection() should fail when client factory fails")
		}
	})
	
	t.Run("authentication error", func(t *testing.T) {
		// Create account config without credentials
		accountConfig := &configpkg.AccountConfig{
			ID:    "no-auth",
			Name:  "No Auth Account",
			Email: "noauth@example.com",
			// No IMAP/SMTP config with credentials
		}
		
		clientFactory := NewMockClientFactory(nil, nil)
		pool := NewConnectionPool(config, authFactory, clientFactory)
		
		if err := pool.Start(ctx); err != nil {
			t.Fatalf("Failed to start pool: %v", err)
		}
		defer pool.Stop(ctx)
		
		// Try to add account - should fail due to auth provider creation
		err := pool.AddAccount(ctx, accountConfig)
		if err == nil {
			t.Error("AddAccount() should fail for account without auth config")
		}
	})
	
	t.Run("connection limit exceeded", func(t *testing.T) {
		limitedConfig := createTestPoolConfig()
		limitedConfig.MaxConnections = 1  // Only allow 1 connection
		
		clientFactory := NewMockClientFactory(nil, nil)
		pool := NewConnectionPool(limitedConfig, authFactory, clientFactory)
		
		if err := pool.Start(ctx); err != nil {
			t.Fatalf("Failed to start pool: %v", err)
		}
		defer pool.Stop(ctx)
		
		// Add test account
		accountConfig := createTestAccountConfig("limited")
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		// Get first connection (should succeed)
		client1, err := pool.GetConnection(ctx, "limited")
		if err != nil {
			t.Fatalf("Failed to get first connection: %v", err)
		}
		if client1 == nil {
			t.Fatal("First client is nil")
		}
		
		// Try to get second connection (should fail - limit exceeded)
		_, err = pool.GetConnection(ctx, "limited")
		if err == nil {
			t.Error("GetConnection() should fail when connection limit is exceeded")
		}
		
		var emailErr *EmailError
		errors.As(err, &emailErr)
		if emailErr.Code != "CONNECTION_ERROR" {
			t.Errorf("Error code = %s, want CONNECTION_ERROR", emailErr.Code)
		}
	})
}

func TestConnectionPool_Stats(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	// Add test account
	accountConfig := createTestAccountConfig("stats")
	if err := pool.AddAccount(ctx, accountConfig); err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}
	
	t.Run("initial stats", func(t *testing.T) {
		stats := pool.GetPoolStats()
		if stats == nil {
			t.Fatal("GetPoolStats() returned nil")
		}
		
		if stats.ActiveConnections != 0 {
			t.Errorf("ActiveConnections = %d, want 0", stats.ActiveConnections)
		}
		
		if stats.IdleConnections != 0 {
			t.Errorf("IdleConnections = %d, want 0", stats.IdleConnections)
		}
		
		if stats.TotalConnections != 0 {
			t.Errorf("TotalConnections = %d, want 0", stats.TotalConnections)
		}
		
		if stats.MaxConnections != config.MaxConnections {
			t.Errorf("MaxConnections = %d, want %d", stats.MaxConnections, config.MaxConnections)
		}
	})
	
	t.Run("stats with active connection", func(t *testing.T) {
		// Get connection
		client, err := pool.GetConnection(ctx, "stats")
		if err != nil {
			t.Fatalf("Failed to get connection: %v", err)
		}
		if client == nil {
			t.Fatal("Client is nil")
		}
		
		stats := pool.GetPoolStats()
		if stats.ActiveConnections != 1 {
			t.Errorf("ActiveConnections = %d, want 1", stats.ActiveConnections)
		}
		
		if stats.TotalConnections != 1 {
			t.Errorf("TotalConnections = %d, want 1", stats.TotalConnections)
		}
	})
	
	t.Run("stats with released connection", func(t *testing.T) {
		// Release connection
		if err := pool.ReleaseConnection("stats"); err != nil {
			t.Fatalf("Failed to release connection: %v", err)
		}
		
		stats := pool.GetPoolStats()
		if stats.ActiveConnections != 0 {
			t.Errorf("ActiveConnections = %d, want 0", stats.ActiveConnections)
		}
		
		if stats.IdleConnections != 1 {
			t.Errorf("IdleConnections = %d, want 1", stats.IdleConnections)
		}
		
		if stats.TotalConnections != 1 {
			t.Errorf("TotalConnections = %d, want 1", stats.TotalConnections)
		}
	})
}

func TestConnectionPool_Cleanup(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	config.IdleTimeout = 100 * time.Millisecond  // Short timeout for testing
	config.HealthCheckInterval = 50 * time.Millisecond  // Frequent cleanup
	
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	// Add test account
	accountConfig := createTestAccountConfig("cleanup")
	if err := pool.AddAccount(ctx, accountConfig); err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}
	
	t.Run("idle connection cleanup", func(t *testing.T) {
		// Get and release connection
		client, err := pool.GetConnection(ctx, "cleanup")
		if err != nil {
			t.Fatalf("Failed to get connection: %v", err)
		}
		if client == nil {
			t.Fatal("Client is nil")
		}
		
		if err := pool.ReleaseConnection("cleanup"); err != nil {
			t.Fatalf("Failed to release connection: %v", err)
		}
		
		// Verify connection is idle
		stats := pool.GetPoolStats()
		if stats.IdleConnections != 1 {
			t.Errorf("IdleConnections = %d, want 1", stats.IdleConnections)
		}
		
		// Wait for cleanup to occur
		time.Sleep(200 * time.Millisecond)
		
		// Verify idle connection was cleaned up
		stats = pool.GetPoolStats()
		if stats.IdleConnections != 0 {
			t.Errorf("IdleConnections = %d, want 0 after cleanup", stats.IdleConnections)
		}
	})
}

func TestConnectionPool_Shutdown(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	
	// Add test account and connections
	accountConfig := createTestAccountConfig("shutdown")
	if err := pool.AddAccount(ctx, accountConfig); err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}
	
	// Get some connections
	client1, err := pool.GetConnection(ctx, "shutdown")
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}
	if client1 == nil {
		t.Fatal("Client1 is nil")
	}
	
	client2, err := pool.GetConnection(ctx, "shutdown")
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}
	if client2 == nil {
		t.Fatal("Client2 is nil")
	}
	
	// Release one connection to create an idle connection
	if err := pool.ReleaseConnection("shutdown"); err != nil {
		t.Fatalf("Failed to release connection: %v", err)
	}
	
	t.Run("graceful shutdown", func(t *testing.T) {
		// Stop the pool
		if err := pool.Stop(ctx); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
		
		// Verify connections are closed
		stats := pool.GetPoolStats()
		if stats.TotalConnections != 0 {
			t.Errorf("TotalConnections = %d, want 0 after shutdown", stats.TotalConnections)
		}
		
		// Verify pool is not running
		_, err := pool.GetConnection(ctx, "shutdown")
		if err == nil {
			t.Error("GetConnection() should fail after shutdown")
		}
	})
	
	t.Run("double stop", func(t *testing.T) {
		// Stopping again should not error
		if err := pool.Stop(ctx); err != nil {
			t.Errorf("Second Stop() error = %v", err)
		}
	})
}

func TestConnectionPool_AddRemoveAccount(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	t.Run("add account successfully", func(t *testing.T) {
		accountConfig := createTestAccountConfig("add-remove")
		
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Errorf("AddAccount() error = %v", err)
		}
		
		// Verify account was added by getting connection status
		status := pool.GetConnectionStatus("add-remove")
		if status == nil {
			t.Error("GetConnectionStatus() returned nil")
		}
		
		if status.AccountID != "add-remove" {
			t.Errorf("AccountID = %s, want add-remove", status.AccountID)
		}
	})
	
	t.Run("add duplicate account", func(t *testing.T) {
		accountConfig := createTestAccountConfig("add-remove")  // Same ID as above
		
		err := pool.AddAccount(ctx, accountConfig)
		if err == nil {
			t.Error("AddAccount() should fail for duplicate account")
		}
		
		var emailErr *EmailError
		errors.As(err, &emailErr)
		if emailErr.Code != "ACCOUNT_EXISTS" {
			t.Errorf("Error code = %s, want ACCOUNT_EXISTS", emailErr.Code)
		}
	})
	
	t.Run("remove account successfully", func(t *testing.T) {
		// Get a connection first to create some state
		client, err := pool.GetConnection(ctx, "add-remove")
		if err != nil {
			t.Fatalf("Failed to get connection: %v", err)
		}
		if client == nil {
			t.Fatal("Client is nil")
		}
		
		// Remove account
		if err := pool.RemoveAccount("add-remove"); err != nil {
			t.Errorf("RemoveAccount() error = %v", err)
		}
		
		// Verify account was removed
		status := pool.GetConnectionStatus("add-remove")
		if status.Connected {
			t.Error("Account should not be connected after removal")
		}
	})
	
	t.Run("remove non-existent account", func(t *testing.T) {
		err := pool.RemoveAccount("nonexistent")
		if err == nil {
			t.Error("RemoveAccount() should fail for non-existent account")
		}
		
		var emailErr *EmailError
		errors.As(err, &emailErr)
		if emailErr.Code != "ACCOUNT_NOT_FOUND" {
			t.Errorf("Error code = %s, want ACCOUNT_NOT_FOUND", emailErr.Code)
		}
	})
}

func TestConnectionPool_ConnectionStatus(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	t.Run("status for non-existent account", func(t *testing.T) {
		status := pool.GetConnectionStatus("nonexistent")
		if status == nil {
			t.Fatal("GetConnectionStatus() returned nil")
		}
		
		if status.AccountID != "nonexistent" {
			t.Errorf("AccountID = %s, want nonexistent", status.AccountID)
		}
		
		if status.Connected {
			t.Error("Non-existent account should not be connected")
		}
		
		if status.LastError == nil {
			t.Error("LastError should not be nil for non-existent account")
		}
	})
	
	t.Run("status for existing account", func(t *testing.T) {
		accountConfig := createTestAccountConfig("status")
		
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		// Initially not connected
		status := pool.GetConnectionStatus("status")
		if status.Connected {
			t.Error("Account should not be connected initially")
		}
		
		// Get connection to make it connected
		client, err := pool.GetConnection(ctx, "status")
		if err != nil {
			t.Fatalf("Failed to get connection: %v", err)
		}
		if client == nil {
			t.Fatal("Client is nil")
		}
		
		// Now should be connected
		status = pool.GetConnectionStatus("status")
		if !status.Connected {
			t.Error("Account should be connected after getting connection")
		}
		
		if status.AccountID != "status" {
			t.Errorf("AccountID = %s, want status", status.AccountID)
		}
	})
}

func TestConnectionPool_CloseConnection(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	t.Run("close connection successfully", func(t *testing.T) {
		accountConfig := createTestAccountConfig("close-test")
		
		// Add account and get connection
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		client, err := pool.GetConnection(ctx, "close-test")
		if err != nil {
			t.Fatalf("Failed to get connection: %v", err)
		}
		if client == nil {
			t.Fatal("Client is nil")
		}
		
		// Close connection
		if err := pool.CloseConnection("close-test"); err != nil {
			t.Errorf("CloseConnection() error = %v", err)
		}
	})
	
	t.Run("close connection for non-existent account", func(t *testing.T) {
		err := pool.CloseConnection("nonexistent")
		if err == nil {
			t.Error("CloseConnection() should fail for non-existent account")
		}
		
		var emailErr *EmailError
		errors.As(err, &emailErr)
		if emailErr.Code != "ACCOUNT_NOT_FOUND" {
			t.Errorf("Error code = %s, want ACCOUNT_NOT_FOUND", emailErr.Code)
		}
	})
}

func TestConnectionPool_CloseAll(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	t.Run("close all connections", func(t *testing.T) {
		// Add multiple accounts and get connections
		for i := 0; i < 3; i++ {
			accountID := fmt.Sprintf("closeall-%d", i)
			accountConfig := createTestAccountConfig(accountID)
			
			if err := pool.AddAccount(ctx, accountConfig); err != nil {
				t.Fatalf("Failed to add account %s: %v", accountID, err)
			}
			
			client, err := pool.GetConnection(ctx, accountID)
			if err != nil {
				t.Fatalf("Failed to get connection for %s: %v", accountID, err)
			}
			if client == nil {
				t.Fatalf("Client is nil for %s", accountID)
			}
		}
		
		// Verify connections exist
		initialStats := pool.GetPoolStats()
		if initialStats.TotalConnections == 0 {
			t.Error("Should have some connections before CloseAll")
		}
		
		// Close all connections
		if err := pool.CloseAll(); err != nil {
			t.Errorf("CloseAll() error = %v", err)
		}
		
		// Verify all connections are closed
		finalStats := pool.GetPoolStats()
		if finalStats.TotalConnections != 0 {
			t.Errorf("TotalConnections = %d, want 0 after CloseAll", finalStats.TotalConnections)
		}
	})
}

func TestConnectionPool_EdgeCases(t *testing.T) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	t.Run("connection validation failure", func(t *testing.T) {
		pool := NewConnectionPool(config, authFactory, clientFactory)
		
		if err := pool.Start(ctx); err != nil {
			t.Fatalf("Failed to start pool: %v", err)
		}
		defer pool.Stop(ctx)
		
		accountConfig := createTestAccountConfig("validation-test")
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		// Get a connection and immediately disconnect it to test stale detection
		client, err := pool.GetConnection(ctx, "validation-test")
		if err != nil {
			t.Fatalf("Failed to get connection: %v", err)
		}
		
		// Release it to idle pool
		if err := pool.ReleaseConnection("validation-test"); err != nil {
			t.Fatalf("Failed to release connection: %v", err)
		}
		
		// Manually disconnect the client to make it stale
		if mockClient, ok := client.(*MockEmailClient); ok {
			mockClient.SetConnected(false)
		}
		
		// Try to get connection again - should detect stale connection and create new one
		client2, err := pool.GetConnection(ctx, "validation-test")
		if err != nil {
			t.Fatalf("Failed to get connection after stale detection: %v", err)
		}
		if client2 == nil {
			t.Error("Second client should not be nil")
		}
	})
	
	t.Run("factory nil check", func(t *testing.T) {
		pool := NewConnectionPool(config, authFactory, nil)  // nil factory
		
		if err := pool.Start(ctx); err != nil {
			t.Fatalf("Failed to start pool: %v", err)
		}
		defer pool.Stop(ctx)
		
		accountConfig := createTestAccountConfig("nil-factory-test")
		if err := pool.AddAccount(ctx, accountConfig); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
		
		// Try to get connection with nil factory
		_, err := pool.GetConnection(ctx, "nil-factory-test")
		if err == nil {
			t.Error("GetConnection() should fail with nil factory")
		}
	})
}

// Benchmark tests for performance verification
func BenchmarkConnectionPool_GetConnection(b *testing.B) {
	ctx := context.Background()
	config := createTestPoolConfig()
	config.MaxConnections = 100  // Increase limit for benchmark
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		b.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	// Add test account
	accountConfig := createTestAccountConfig("bench")
	if err := pool.AddAccount(ctx, accountConfig); err != nil {
		b.Fatalf("Failed to add account: %v", err)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client, err := pool.GetConnection(ctx, "bench")
			if err != nil {
				b.Errorf("GetConnection() error = %v", err)
			}
			if client == nil {
				b.Error("GetConnection() returned nil client")
			}
			
			// Release immediately for reuse
			_ = pool.ReleaseConnection("bench")
		}
	})
}

func BenchmarkConnectionPool_Stats(b *testing.B) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		b.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	// Add test account and get some connections
	accountConfig := createTestAccountConfig("bench-stats")
	if err := pool.AddAccount(ctx, accountConfig); err != nil {
		b.Fatalf("Failed to add account: %v", err)
	}
	
	// Create some pool state
	for i := 0; i < 3; i++ {
		_, _ = pool.GetConnection(ctx, "bench-stats")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats := pool.GetPoolStats()
		if stats == nil {
			b.Error("GetPoolStats() returned nil")
		}
	}
}

func BenchmarkConnectionPool_HealthCheck(b *testing.B) {
	ctx := context.Background()
	config := createTestPoolConfig()
	credStore := NewMockCredentialStore()
	authFactory := NewAuthProviderFactory(credStore)
	clientFactory := NewMockClientFactory(nil, nil)
	
	pool := NewConnectionPool(config, authFactory, clientFactory)
	
	if err := pool.Start(ctx); err != nil {
		b.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)
	
	// Add test account and get connection
	accountConfig := createTestAccountConfig("bench-health")
	if err := pool.AddAccount(ctx, accountConfig); err != nil {
		b.Fatalf("Failed to add account: %v", err)
	}
	
	client, err := pool.GetConnection(ctx, "bench-health")
	if err != nil {
		b.Fatalf("Failed to get connection: %v", err)
	}
	if client == nil {
		b.Fatal("Client is nil")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := pool.HealthCheck(ctx, "bench-health"); err != nil {
			b.Errorf("HealthCheck() error = %v", err)
		}
	}
}