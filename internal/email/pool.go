package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	configpkg "github.com/ybarbara/pombo/internal/config"
)

// ConnectionPool manages a pool of email connections for multiple accounts
type ConnectionPool struct {
	pools    map[string]*accountPool // Per-account connection pools
	config   *PoolConfig             // Pool configuration
	stats    *PoolStats              // Pool statistics
	mu       sync.RWMutex            // Protects pools map and stats
	cleanup  *time.Ticker            // Periodic cleanup timer
	shutdown chan struct{}           // Shutdown signal channel
	authFactory *AuthProviderFactory // Factory for creating auth providers
	clientFactory ClientFactory      // Factory for creating email clients
	running  bool                    // Pool running state
}

// accountPool manages connections for a specific email account
type accountPool struct {
	accountID   string                     // Account identifier
	config      *configpkg.AccountConfig   // Account configuration
	auth        AuthProvider               // Authentication provider
	active      map[string]EmailClient     // Active connections (connection ID -> client)
	idle        []pooledConnection         // Idle connections ready for reuse
	mu          sync.RWMutex               // Protects connection lists
	lastUsed    time.Time                  // Last time a connection was requested
	created     time.Time                  // When this pool was created
	connCounter int                        // Counter for generating connection IDs
}

// pooledConnection represents a connection with metadata
type pooledConnection struct {
	client    EmailClient  // The actual email client
	id        string       // Unique connection identifier
	createdAt time.Time    // When this connection was created
	lastUsed  time.Time    // Last time this connection was used
}

// ClientFactory creates email clients for specific account configurations
type ClientFactory interface {
	CreateClient(ctx context.Context, account *configpkg.AccountConfig, auth AuthProvider) (EmailClient, error)
}

// NewConnectionPool creates a new connection pool with the given configuration
func NewConnectionPool(config *PoolConfig, authFactory *AuthProviderFactory, clientFactory ClientFactory) *ConnectionPool {
	if config == nil {
		config = &PoolConfig{
			MaxConnections:      5,
			MaxIdleConnections:  2,
			ConnectionLifetime:  30 * time.Minute,
			IdleTimeout:         5 * time.Minute,
			HealthCheckInterval: 1 * time.Minute,
			ConnectTimeout:      30 * time.Second,
		}
	}

	pool := &ConnectionPool{
		pools:       make(map[string]*accountPool),
		config:      config,
		stats:       &PoolStats{MaxConnections: config.MaxConnections},
		authFactory: authFactory,
		clientFactory: clientFactory,
		shutdown:    make(chan struct{}),
	}

	return pool
}

// Start initializes the connection pool and starts background tasks
func (cp *ConnectionPool) Start(ctx context.Context) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.running {
		return NewEmailError(ErrorTypeClient, "POOL_ALREADY_RUNNING", "connection pool is already running", nil, false)
	}

	// Start periodic cleanup
	cp.cleanup = time.NewTicker(cp.config.HealthCheckInterval)
	cp.running = true

	// Start cleanup goroutine
	go cp.cleanupWorker(ctx)

	return nil
}

// Stop gracefully shuts down the connection pool
func (cp *ConnectionPool) Stop(ctx context.Context) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.running {
		return nil
	}

	cp.running = false
	close(cp.shutdown)

	if cp.cleanup != nil {
		cp.cleanup.Stop()
	}

	// Close all connections
	return cp.closeAllInternal()
}

// GetConnection retrieves or creates a connection for the specified account
func (cp *ConnectionPool) GetConnection(ctx context.Context, accountID string) (EmailClient, error) {
	// Check if pool is running
	cp.mu.RLock()
	if !cp.running {
		cp.mu.RUnlock()
		return nil, NewEmailError(ErrorTypeClient, "POOL_NOT_RUNNING", "connection pool is not running", nil, false).
			WithContext(accountID, "", "", "GetConnection")
	}
	cp.mu.RUnlock()

	// Get or create account pool
	accountPool, err := cp.getOrCreateAccountPool(ctx, accountID)
	if err != nil {
		return nil, WrapError(err, ErrorTypeClient, "ACCOUNT_POOL_ERROR", "failed to get account pool", false).
			WithContext(accountID, "", "", "GetConnection")
	}

	// Get connection from account pool
	client, err := accountPool.getConnection(ctx, cp.config, cp.clientFactory)
	if err != nil {
		return nil, WrapError(err, ErrorTypeClient, "CONNECTION_ERROR", "failed to get connection", true).
			WithContext(accountID, "", "", "GetConnection")
	}

	// Update statistics
	cp.updateStatsOnAcquire()

	return client, nil
}

// ReleaseConnection returns a connection to the pool for reuse
func (cp *ConnectionPool) ReleaseConnection(accountID string) error {
	cp.mu.RLock()
	accountPool, exists := cp.pools[accountID]
	cp.mu.RUnlock()

	if !exists {
		return NewEmailError(ErrorTypeNotFound, "ACCOUNT_NOT_FOUND", fmt.Sprintf("account pool not found: %s", accountID), nil, false).
			WithContext(accountID, "", "", "ReleaseConnection")
	}

	err := accountPool.releaseConnection()
	if err == nil {
		cp.updateStatsOnRelease()
	}

	return err
}

// CloseConnection closes a specific connection for an account
func (cp *ConnectionPool) CloseConnection(accountID string) error {
	cp.mu.RLock()
	accountPool, exists := cp.pools[accountID]
	cp.mu.RUnlock()

	if !exists {
		return NewEmailError(ErrorTypeNotFound, "ACCOUNT_NOT_FOUND", fmt.Sprintf("account pool not found: %s", accountID), nil, false).
			WithContext(accountID, "", "", "CloseConnection")
	}

	err := accountPool.closeConnection()
	if err == nil {
		cp.updateStatsOnClose()
	}

	return err
}

// CloseAll closes all connections in the pool
func (cp *ConnectionPool) CloseAll() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.closeAllInternal()
}

// closeAllInternal closes all connections (internal method, requires lock)
func (cp *ConnectionPool) closeAllInternal() error {
	var errs []error

	for accountID, accountPool := range cp.pools {
		if err := accountPool.closeAll(); err != nil {
			errs = append(errs, WrapError(err, ErrorTypeClient, "CLOSE_ERROR", 
				fmt.Sprintf("failed to close connections for account %s", accountID), false))
		}
	}

	// Clear pools
	cp.pools = make(map[string]*accountPool)
	cp.resetStats()

	if len(errs) > 0 {
		return NewEmailError(ErrorTypeClient, "CLOSE_ALL_ERROR", 
			fmt.Sprintf("failed to close %d account pools", len(errs)), errs[0], false).
			WithDetails(errs)
	}

	return nil
}

// HealthCheck performs a health check on connections for the specified account
func (cp *ConnectionPool) HealthCheck(ctx context.Context, accountID string) error {
	cp.mu.RLock()
	accountPool, exists := cp.pools[accountID]
	cp.mu.RUnlock()

	if !exists {
		return NewEmailError(ErrorTypeNotFound, "ACCOUNT_NOT_FOUND", fmt.Sprintf("account pool not found: %s", accountID), nil, false).
			WithContext(accountID, "", "", "HealthCheck")
	}

	return accountPool.healthCheck(ctx)
}

// GetConnectionStatus returns the connection status for an account
func (cp *ConnectionPool) GetConnectionStatus(accountID string) *ConnectionStatus {
	cp.mu.RLock()
	accountPool, exists := cp.pools[accountID]
	cp.mu.RUnlock()

	if !exists {
		return &ConnectionStatus{
			AccountID:   accountID,
			Connected:   false,
			LastError:   NewEmailError(ErrorTypeNotFound, "ACCOUNT_NOT_FOUND", "account not found", nil, false),
		}
	}

	return accountPool.getStatus()
}

// GetPoolStats returns current pool statistics
func (cp *ConnectionPool) GetPoolStats() *PoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	// Calculate current statistics
	totalActive := 0
	totalIdle := 0
	totalLatency := time.Duration(0)
	errorCount := 0
	successCount := 0

	for _, pool := range cp.pools {
		stats := pool.getPoolStats()
		totalActive += stats.activeConnections
		totalIdle += stats.idleConnections
		totalLatency += stats.averageLatency
		errorCount += stats.errors
		successCount += stats.successes
	}

	// Calculate average latency
	var avgLatency time.Duration
	if len(cp.pools) > 0 {
		avgLatency = totalLatency / time.Duration(len(cp.pools))
	}

	// Calculate error rate
	var errorRate float64
	if total := errorCount + successCount; total > 0 {
		errorRate = float64(errorCount) / float64(total)
	}

	return &PoolStats{
		ActiveConnections: totalActive,
		IdleConnections:   totalIdle,
		TotalConnections:  totalActive + totalIdle,
		MaxConnections:    cp.config.MaxConnections,
		AverageLatency:    avgLatency,
		ErrorRate:         errorRate,
	}
}

// SetPoolConfig updates the pool configuration
func (cp *ConnectionPool) SetPoolConfig(config *PoolConfig) error {
	if config == nil {
		return NewEmailError(ErrorTypeValidation, "INVALID_CONFIG", "pool config cannot be nil", nil, false)
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Validate configuration
	if config.MaxConnections <= 0 {
		return NewEmailError(ErrorTypeValidation, "INVALID_CONFIG", "max connections must be positive", nil, false)
	}
	if config.MaxIdleConnections < 0 {
		return NewEmailError(ErrorTypeValidation, "INVALID_CONFIG", "max idle connections cannot be negative", nil, false)
	}
	if config.MaxIdleConnections > config.MaxConnections {
		return NewEmailError(ErrorTypeValidation, "INVALID_CONFIG", "max idle connections cannot exceed max connections", nil, false)
	}

	cp.config = config
	cp.stats.MaxConnections = config.MaxConnections

	// Restart cleanup timer if interval changed
	if cp.cleanup != nil && cp.running {
		cp.cleanup.Stop()
		cp.cleanup = time.NewTicker(config.HealthCheckInterval)
	}

	return nil
}

// getOrCreateAccountPool gets an existing account pool or creates a new one
func (cp *ConnectionPool) getOrCreateAccountPool(ctx context.Context, accountID string) (*accountPool, error) {
	cp.mu.RLock()
	if pool, exists := cp.pools[accountID]; exists {
		cp.mu.RUnlock()
		return pool, nil
	}
	cp.mu.RUnlock()

	// Need to create a new account pool
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := cp.pools[accountID]; exists {
		return pool, nil
	}

	// Load account configuration - this would typically come from a config service
	// For now, we'll return an error indicating the account needs to be registered
	return nil, NewEmailError(ErrorTypeConfiguration, "ACCOUNT_NOT_CONFIGURED", 
		fmt.Sprintf("account %s is not configured", accountID), nil, false)
}

// AddAccount registers an account with the pool
func (cp *ConnectionPool) AddAccount(ctx context.Context, account *configpkg.AccountConfig) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if _, exists := cp.pools[account.ID]; exists {
		return NewEmailError(ErrorTypeClient, "ACCOUNT_EXISTS", 
			fmt.Sprintf("account %s already exists", account.ID), nil, false)
	}

	// Create authentication provider
	auth, err := cp.authFactory.CreateProvider(ctx, account)
	if err != nil {
		return WrapError(err, ErrorTypeAuth, "AUTH_PROVIDER_ERROR", 
			"failed to create authentication provider", false).
			WithContext(account.ID, "", "", "AddAccount")
	}

	// Create account pool
	pool := &accountPool{
		accountID: account.ID,
		config:    account,
		auth:      auth,
		active:    make(map[string]EmailClient),
		idle:      make([]pooledConnection, 0),
		created:   time.Now(),
		lastUsed:  time.Now(),
	}

	cp.pools[account.ID] = pool

	return nil
}

// RemoveAccount removes an account from the pool
func (cp *ConnectionPool) RemoveAccount(accountID string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	pool, exists := cp.pools[accountID]
	if !exists {
		return NewEmailError(ErrorTypeNotFound, "ACCOUNT_NOT_FOUND", 
			fmt.Sprintf("account %s not found", accountID), nil, false)
	}

	// Close all connections for this account
	if err := pool.closeAll(); err != nil {
		return WrapError(err, ErrorTypeClient, "CLOSE_ERROR", 
			"failed to close account connections", false).
			WithContext(accountID, "", "", "RemoveAccount")
	}

	delete(cp.pools, accountID)
	return nil
}

// cleanupWorker runs periodic cleanup tasks
func (cp *ConnectionPool) cleanupWorker(ctx context.Context) {
	defer func() {
		if cp.cleanup != nil {
			cp.cleanup.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cp.shutdown:
			return
		case <-cp.cleanup.C:
			cp.performCleanup(ctx)
		}
	}
}

// performCleanup performs cleanup tasks
func (cp *ConnectionPool) performCleanup(ctx context.Context) {
	cp.mu.RLock()
	pools := make([]*accountPool, 0, len(cp.pools))
	for _, pool := range cp.pools {
		pools = append(pools, pool)
	}
	cp.mu.RUnlock()

	// Clean up each account pool
	for _, pool := range pools {
		if err := pool.cleanup(ctx, cp.config); err != nil {
			// Log cleanup errors but don't stop the cleanup process
			_ = err // TODO: Add proper logging
		}
	}
}

// updateStatsOnAcquire updates statistics when a connection is acquired
func (cp *ConnectionPool) updateStatsOnAcquire() {
	// This would typically update detailed metrics
	// For now, stats are calculated dynamically in GetPoolStats()
}

// updateStatsOnRelease updates statistics when a connection is released
func (cp *ConnectionPool) updateStatsOnRelease() {
	// This would typically update detailed metrics
}

// updateStatsOnClose updates statistics when a connection is closed
func (cp *ConnectionPool) updateStatsOnClose() {
	// This would typically update detailed metrics
}

// resetStats resets pool statistics
func (cp *ConnectionPool) resetStats() {
	cp.stats = &PoolStats{
		MaxConnections: cp.config.MaxConnections,
	}
}

// Account pool methods

// poolStats holds account pool statistics
type poolStats struct {
	activeConnections int
	idleConnections   int
	averageLatency    time.Duration
	errors            int
	successes         int
}

// getConnection gets a connection from the account pool
func (ap *accountPool) getConnection(ctx context.Context, config *PoolConfig, clientFactory ClientFactory) (EmailClient, error) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.lastUsed = time.Now()

	// Try to reuse an idle connection
	if len(ap.idle) > 0 {
		conn := ap.idle[len(ap.idle)-1]
		ap.idle = ap.idle[:len(ap.idle)-1]

		// Check if connection is still valid
		if ap.isConnectionValid(ctx, conn, config) {
			// Move to active connections
			ap.active[conn.id] = conn.client
			return conn.client, nil
		} else {
			// Connection is stale, disconnect it
			_ = conn.client.Disconnect(ctx)
		}
	}

	// Check if we can create a new connection
	totalConnections := len(ap.active) + len(ap.idle)
	if totalConnections >= config.MaxConnections {
		return nil, NewEmailError(ErrorTypeRateLimit, ErrCodeTooManyConns, 
			fmt.Sprintf("maximum connections reached for account %s", ap.accountID), nil, true).
			WithContext(ap.accountID, "", "", "getConnection")
	}

	// Create new connection
	return ap.createConnection(ctx, config, clientFactory)
}

// createConnection creates a new connection for the account
func (ap *accountPool) createConnection(ctx context.Context, config *PoolConfig, clientFactory ClientFactory) (EmailClient, error) {
	// Create connection context with timeout
	connCtx, cancel := context.WithTimeout(ctx, config.ConnectTimeout)
	defer cancel()

	// Refresh authentication if needed
	if err := ap.auth.RefreshIfNeeded(connCtx); err != nil {
		return nil, WrapError(err, ErrorTypeAuth, ErrCodeAuthFailed, 
			"failed to refresh authentication", true).
			WithContext(ap.accountID, "", "", "createConnection")
	}

	// Create email client using the factory
	client, err := ap.createEmailClientWithFactory(connCtx, clientFactory)
	if err != nil {
		return nil, WrapError(err, ErrorTypeClient, ErrCodeConnectionFailed, 
			"failed to create email client", true).
			WithContext(ap.accountID, "", "", "createConnection")
	}

	// Convert config.AccountConfig to email.AccountConfig for the client
	emailConfig := ap.configToEmailConfig(ap.config)

	// Connect the client
	if err := client.Connect(connCtx, emailConfig); err != nil {
		return nil, WrapError(err, ErrorTypeNetwork, ErrCodeConnectionFailed, 
			"failed to connect email client", true).
			WithContext(ap.accountID, "", "", "createConnection")
	}

	// Generate connection ID and add to active connections
	ap.connCounter++
	connID := fmt.Sprintf("%s-%d", ap.accountID, ap.connCounter)
	ap.active[connID] = client

	return client, nil
}

// configToEmailConfig converts configpkg.AccountConfig to email.AccountConfig
func (ap *accountPool) configToEmailConfig(cfg *configpkg.AccountConfig) *AccountConfig {
	emailConfig := &AccountConfig{
		ID:       cfg.ID,
		Name:     cfg.Name,
		Email:    cfg.Email,
		Provider: cfg.Provider,
	}

	// Convert IMAP config
	if cfg.IMAP.Host != "" {
		emailConfig.IMAP = &IMAPConfig{
			Host:      cfg.IMAP.Host,
			Port:      cfg.IMAP.Port,
			TLS:       cfg.IMAP.TLS,
			StartTLS:  cfg.IMAP.StartTLS,
			Username:  cfg.IMAP.Username,
			Timeout:   cfg.IMAP.Timeout,
			KeepAlive: cfg.IMAP.KeepAlive,
		}
	}

	// Convert SMTP config
	if cfg.SMTP.Host != "" {
		emailConfig.SMTP = &SMTPConfig{
			Host:     cfg.SMTP.Host,
			Port:     cfg.SMTP.Port,
			TLS:      cfg.SMTP.TLS,
			StartTLS: cfg.SMTP.StartTLS,
			Username: cfg.SMTP.Username,
			Timeout:  cfg.SMTP.Timeout,
		}
	}

	// Convert OAuth config
	if cfg.OAuth != nil {
		emailConfig.OAuth = &OAuthConfig{
			Provider:     cfg.OAuth.Provider,
			ClientID:     cfg.OAuth.ClientID,
			ClientSecret: cfg.OAuth.ClientSecret,
			RedirectURI:  cfg.OAuth.RedirectURI,
			Scopes:       cfg.OAuth.Scopes,
			AuthURL:      cfg.OAuth.AuthURL,
			TokenURL:     cfg.OAuth.TokenURL,
		}
	}

	// Convert settings
	if cfg.Settings != nil {
		emailConfig.Settings = &AccountSettings{
			Signature:           cfg.Settings.Signature,
			AutoBCC:             cfg.Settings.AutoBCC,
			SyncInterval:        cfg.Settings.SyncInterval,
			MaxSyncMessages:     cfg.Settings.MaxSyncMessages,
			ComposeFormat:       cfg.Settings.ComposeFormat,
			AutoMarkRead:        cfg.Settings.AutoMarkRead,
			DownloadAttachments: cfg.Settings.DownloadAttachments,
		}
	}

	return emailConfig
}

// createEmailClient creates an email client using the client factory
func (ap *accountPool) createEmailClientWithFactory(ctx context.Context, clientFactory ClientFactory) (EmailClient, error) {
	if clientFactory == nil {
		return nil, NewEmailError(ErrorTypeClient, "NO_CLIENT_FACTORY", 
			"client factory not provided", nil, false)
	}
	
	return clientFactory.CreateClient(ctx, ap.config, ap.auth)
}

// isConnectionValid checks if a connection is still valid and usable
func (ap *accountPool) isConnectionValid(ctx context.Context, conn pooledConnection, config *PoolConfig) bool {
	// Check connection age
	if time.Since(conn.createdAt) > config.ConnectionLifetime {
		return false
	}

	// Check idle time
	if time.Since(conn.lastUsed) > config.IdleTimeout {
		return false
	}

	// Check if client is still connected
	if !conn.client.IsConnected() {
		return false
	}

	// Perform ping test with short timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := conn.client.Ping(pingCtx); err != nil {
		return false
	}

	return true
}

// releaseConnection releases the most recently used active connection to idle pool
func (ap *accountPool) releaseConnection() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	// Find a connection to release (for simplicity, release any active connection)
	for connID, client := range ap.active {
		// Remove from active
		delete(ap.active, connID)

		// Add to idle pool if under limit
		if len(ap.idle) < ap.getMaxIdleConnections() {
			conn := pooledConnection{
				client:    client,
				id:        connID,
				createdAt: time.Now(), // This should be the original creation time in practice
				lastUsed:  time.Now(),
			}
			ap.idle = append(ap.idle, conn)
		} else {
			// Too many idle connections, disconnect this one
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_ = client.Disconnect(ctx)
			}()
		}
		return nil
	}

	return NewEmailError(ErrorTypeClient, "NO_ACTIVE_CONNECTION", 
		"no active connection to release", nil, false).
		WithContext(ap.accountID, "", "", "releaseConnection")
}

// closeConnection closes one connection (prioritizing idle connections)
func (ap *accountPool) closeConnection() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	// First try to close an idle connection
	if len(ap.idle) > 0 {
		conn := ap.idle[len(ap.idle)-1]
		ap.idle = ap.idle[:len(ap.idle)-1]

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = conn.client.Disconnect(ctx)
		}()
		return nil
	}

	// If no idle connections, close an active connection
	for connID, client := range ap.active {
		delete(ap.active, connID)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = client.Disconnect(ctx)
		}()
		return nil
	}

	return NewEmailError(ErrorTypeClient, "NO_CONNECTION", 
		"no connection to close", nil, false).
		WithContext(ap.accountID, "", "", "closeConnection")
}

// closeAll closes all connections in the account pool
func (ap *accountPool) closeAll() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	var errs []error

	// Close active connections
	for connID, client := range ap.active {
		if err := ap.disconnectClient(client); err != nil {
			errs = append(errs, WrapError(err, ErrorTypeClient, "DISCONNECT_ERROR", 
				fmt.Sprintf("failed to disconnect active connection %s", connID), false))
		}
	}

	// Close idle connections
	for _, conn := range ap.idle {
		if err := ap.disconnectClient(conn.client); err != nil {
			errs = append(errs, WrapError(err, ErrorTypeClient, "DISCONNECT_ERROR", 
				fmt.Sprintf("failed to disconnect idle connection %s", conn.id), false))
		}
	}

	// Clear connection lists
	ap.active = make(map[string]EmailClient)
	ap.idle = make([]pooledConnection, 0)

	if len(errs) > 0 {
		return NewEmailError(ErrorTypeClient, "CLOSE_ALL_ERROR", 
			fmt.Sprintf("failed to close %d connections", len(errs)), errs[0], false).
			WithContext(ap.accountID, "", "", "closeAll").
			WithDetails(errs)
	}

	return nil
}

// disconnectClient safely disconnects a client
func (ap *accountPool) disconnectClient(client EmailClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return client.Disconnect(ctx)
}

// healthCheck performs health checks on connections
func (ap *accountPool) healthCheck(ctx context.Context) error {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	var errs []error

	// Check active connections
	for connID, client := range ap.active {
		if err := client.Ping(ctx); err != nil {
			errs = append(errs, WrapError(err, ErrorTypeNetwork, "HEALTH_CHECK_FAILED", 
				fmt.Sprintf("health check failed for active connection %s", connID), true))
		}
	}

	// Check idle connections
	for _, conn := range ap.idle {
		if err := conn.client.Ping(ctx); err != nil {
			errs = append(errs, WrapError(err, ErrorTypeNetwork, "HEALTH_CHECK_FAILED", 
				fmt.Sprintf("health check failed for idle connection %s", conn.id), true))
		}
	}

	if len(errs) > 0 {
		return NewEmailError(ErrorTypeNetwork, "HEALTH_CHECK_FAILED", 
			fmt.Sprintf("health check failed for %d connections", len(errs)), errs[0], true).
			WithContext(ap.accountID, "", "", "healthCheck").
			WithDetails(errs)
	}

	return nil
}

// getStatus returns the connection status for the account
func (ap *accountPool) getStatus() *ConnectionStatus {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	connected := len(ap.active) > 0 || len(ap.idle) > 0
	var lastError error

	// Check if any connection has recent errors
	// In a real implementation, this would track recent errors

	return &ConnectionStatus{
		AccountID:      ap.accountID,
		Connected:      connected,
		LastPing:       time.Now(), // This should track actual last ping
		LastError:      lastError,
		ConnectedAt:    ap.created,
		ReconnectCount: 0, // This should track actual reconnect count
	}
}

// getPoolStats returns statistics for this account pool
func (ap *accountPool) getPoolStats() poolStats {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	return poolStats{
		activeConnections: len(ap.active),
		idleConnections:   len(ap.idle),
		averageLatency:    0, // This would be calculated from actual latency measurements
		errors:            0, // This would track actual error count
		successes:         0, // This would track actual success count
	}
}

// cleanup performs cleanup tasks for the account pool
func (ap *accountPool) cleanup(ctx context.Context, config *PoolConfig) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	var errs []error
	now := time.Now()

	// Clean up stale idle connections
	validIdle := make([]pooledConnection, 0, len(ap.idle))
	for _, conn := range ap.idle {
		if now.Sub(conn.lastUsed) > config.IdleTimeout || 
		   now.Sub(conn.createdAt) > config.ConnectionLifetime {
			// Connection is stale, disconnect it
			if err := ap.disconnectClient(conn.client); err != nil {
				errs = append(errs, err)
			}
		} else {
			validIdle = append(validIdle, conn)
		}
	}
	ap.idle = validIdle

	// Check active connections for staleness
	for connID, client := range ap.active {
		// In a real implementation, we'd track creation time and last use for active connections
		// For now, just check if they're still connected
		if !client.IsConnected() {
			delete(ap.active, connID)
			if err := ap.disconnectClient(client); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return NewEmailError(ErrorTypeClient, "CLEANUP_ERROR", 
			fmt.Sprintf("cleanup encountered %d errors", len(errs)), errs[0], false).
			WithContext(ap.accountID, "", "", "cleanup").
			WithDetails(errs)
	}

	return nil
}

// getMaxIdleConnections returns the maximum number of idle connections for this pool
func (ap *accountPool) getMaxIdleConnections() int {
	// This could be configurable per account, for now use a default
	return 2
}