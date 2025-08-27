// Package imap provides IMAP protocol implementation
package imap

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
	
	"github.com/ybarbara/pombo/internal/email"
)

// Client implements the email.IMAPClient interface using go-imap v1
type Client struct {
	// Configuration
	config *email.IMAPConfig
	auth   email.AuthProvider
	
	// Connection state
	client  *client.Client
	state   email.ConnectionState
	mu      sync.RWMutex
	
	// Server information
	serverInfo *email.ServerInfo
	
	// Connection metadata
	connectedAt time.Time
	lastPing    time.Time
	
	// IDLE support
	idleSupported bool
	idleMu        sync.Mutex
	idleCancel    context.CancelFunc
}

// NewClient creates a new IMAP client instance
func NewClient() *Client {
	return &Client{
		state: email.StateDisconnected,
	}
}

// Connect establishes a connection to the IMAP server
func (c *Client) Connect(ctx context.Context, config *email.IMAPConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if config == nil {
		return email.NewSimpleEmailError(email.ErrorTypeValidation, "IMAP config is required")
	}
	
	c.config = config
	
	// Build server address
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	
	// Apply connection timeout
	var dialer net.Dialer
	if config.Timeout > 0 {
		dialer.Timeout = config.Timeout
	}
	
	// Establish connection based on TLS settings
	var imapClient *client.Client
	var err error
	
	if config.TLS {
		// Direct TLS connection (usually port 993)
		tlsConfig := &tls.Config{
			ServerName: config.Host,
			MinVersion: tls.VersionTLS12,
		}
		
		imapClient, err = client.DialTLS(addr, tlsConfig)
	} else {
		// Plain connection (usually port 143)
		imapClient, err = client.Dial(addr)
		if err == nil && config.StartTLS {
			// Upgrade to TLS using STARTTLS
			tlsConfig := &tls.Config{
				ServerName: config.Host,
				MinVersion: tls.VersionTLS12,
			}
			if err = imapClient.StartTLS(tlsConfig); err != nil {
				imapClient.Close()
				return c.wrapConnectionError(err)
			}
		}
	}
	
	if err != nil {
		return c.wrapConnectionError(err)
	}
	
	c.client = imapClient
	c.state = email.StateConnected
	c.connectedAt = time.Now()
	c.lastPing = time.Now()
	
	// Get server capabilities and info
	if err := c.updateServerInfo(); err != nil {
		c.client.Close()
		c.client = nil
		c.state = email.StateDisconnected
		return err
	}
	
	return nil
}

// Authenticate authenticates with the IMAP server
func (c *Client) Authenticate(ctx context.Context, auth email.AuthProvider) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.state != email.StateConnected {
		return email.NewSimpleEmailError(email.ErrorTypeProtocol, "not connected to server")
	}
	
	if auth == nil {
		return email.NewSimpleEmailError(email.ErrorTypeValidation, "auth provider is required")
	}
	
	c.auth = auth
	
	// Get credentials
	creds, err := auth.GetCredentials(ctx)
	if err != nil {
		return email.WrapSimpleError(err, "failed to get credentials")
	}
	
	// Authenticate based on auth type
	switch creds.Type {
	case email.AuthTypePassword:
		if creds.Username == "" || creds.Password == "" {
			return email.NewSimpleEmailError(email.ErrorTypeAuth, "username and password required")
		}
		err = c.client.Login(creds.Username, creds.Password)
		
	case email.AuthTypeOAuth2:
		if creds.Token == nil {
			return email.NewEmailError(email.ErrorTypeAuth, "OAUTH_TOKEN_REQUIRED", "OAuth2 token required", nil, false)
		}
		// Use SASL OAUTHBEARER or XOAUTH2
		err = c.authenticateOAuth2(ctx, creds)
		
	default:
		return email.NewEmailError(email.ErrorTypeAuth, "UNSUPPORTED_AUTH_TYPE", 
			fmt.Sprintf("unsupported auth type: %s", creds.Type), nil, false)
	}
	
	if err != nil {
		return c.wrapAuthError(err)
	}
	
	c.state = email.StateAuthenticated
	return nil
}

// authenticateOAuth2 handles OAuth2 authentication
func (c *Client) authenticateOAuth2(ctx context.Context, creds *email.Credentials) error {
	token := creds.Token
	if token == nil {
		return email.NewEmailError(email.ErrorTypeAuth, "OAUTH_TOKEN_NIL", "OAuth2 token is nil", nil, false)
	}
	
	// Check if token is expired and refresh if needed
	if time.Now().After(token.ExpiresAt) {
		if c.auth != nil {
			if err := c.auth.RefreshIfNeeded(ctx); err != nil {
				return email.WrapError(err, email.ErrorTypeAuth, "TOKEN_REFRESH_FAILED", "failed to refresh OAuth2 token", true)
			}
			
			// Get updated credentials
			updatedCreds, err := c.auth.GetCredentials(ctx)
			if err != nil {
				return email.WrapError(err, email.ErrorTypeAuth, "CREDENTIALS_GET_FAILED", "failed to get refreshed credentials", false)
			}
			token = updatedCreds.Token
		}
	}
	
	// Get username
	username := creds.Username
	if username == "" && c.config != nil {
		username = c.config.Username
	}
	
	// Check for OAUTHBEARER support
	caps, err := c.client.Capability()
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "CAPABILITY_FAILED", "failed to get capabilities", true)
	}
	
	supportsOAuth := false
	for cap := range caps {
		if strings.Contains(strings.ToUpper(cap), "AUTH=OAUTHBEARER") {
			supportsOAuth = true
			break
		}
	}
	
	if supportsOAuth {
		saslClient := sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
			Username: username,
			Token:    token.AccessToken,
		})
		return c.client.Authenticate(saslClient)
	}
	
	return email.NewEmailError(email.ErrorTypeAuth, "OAUTH_UNSUPPORTED", 
		"server does not support OAUTHBEARER authentication", nil, false)
}

// Logout closes the IMAP session
func (c *Client) Logout(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.client == nil {
		return nil
	}
	
	// Cancel any active IDLE
	c.cancelIdle()
	
	// Send LOGOUT command
	err := c.client.Logout()
	
	// Close connection regardless of logout result
	c.client.Close()
	c.client = nil
	c.state = email.StateDisconnected
	
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "LOGOUT_FAILED", "logout failed", false)
	}
	
	return nil
}

// Capability returns server capabilities
func (c *Client) Capability(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.client == nil {
		return nil, email.NewEmailError(email.ErrorTypeProtocol, "NOT_CONNECTED", "not connected", nil, false)
	}
	
	caps, err := c.client.Capability()
	if err != nil {
		return nil, email.WrapError(err, email.ErrorTypeProtocol, "CAPABILITY_FAILED", "failed to get capabilities", true)
	}
	
	// Convert map[string]bool to []string
	result := make([]string, 0, len(caps))
	for cap := range caps {
		result = append(result, cap)
	}
	
	return result, nil
}

// ServerInfo returns server information
func (c *Client) ServerInfo() *email.ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.serverInfo
}

// State returns the current connection state
func (c *Client) State() email.ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.state
}

// Close closes the connection
func (c *Client) Close() error {
	return c.Logout(context.Background())
}

// updateServerInfo fetches and caches server information
func (c *Client) updateServerInfo() error {
	caps, err := c.client.Capability()
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "CAPABILITY_FAILED", "failed to get server capabilities", true)
	}
	
	// Convert map[string]bool to []string
	capList := make([]string, 0, len(caps))
	for cap := range caps {
		capList = append(capList, cap)
	}
	
	c.serverInfo = &email.ServerInfo{
		Name:         c.config.Host,
		Capabilities: capList,
	}
	
	// Check for IDLE support
	c.idleSupported = false
	for cap := range caps {
		if strings.ToUpper(cap) == "IDLE" {
			c.idleSupported = true
			break
		}
	}
	
	return nil
}

// cancelIdle cancels any active IDLE operation
func (c *Client) cancelIdle() {
	c.idleMu.Lock()
	defer c.idleMu.Unlock()
	
	if c.idleCancel != nil {
		c.idleCancel()
		c.idleCancel = nil
	}
}

// wrapConnectionError wraps connection-related errors
func (c *Client) wrapConnectionError(err error) error {
	if err == nil {
		return nil
	}
	
	// Check for specific error types
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			return email.NewEmailError(email.ErrorTypeTimeout, 
				"CONNECTION_TIMEOUT", "connection timeout", err, true).WithDetails(map[string]interface{}{
					"host": c.config.Host,
					"port": c.config.Port,
				})
		}
	}
	
	// Check for TLS errors
	if strings.Contains(err.Error(), "tls") || strings.Contains(err.Error(), "certificate") {
		return email.NewEmailError(email.ErrorTypeSecurity, 
			"TLS_FAILED", "TLS connection failed", err, false).WithDetails(map[string]interface{}{
				"host": c.config.Host,
				"port": c.config.Port,
				"error": err.Error(),
			})
	}
	
	return email.NewEmailError(email.ErrorTypeNetwork, 
		"CONNECTION_FAILED", "connection failed", err, true).WithDetails(map[string]interface{}{
			"host": c.config.Host,
			"port": c.config.Port,
			"error": err.Error(),
		})
}

// wrapAuthError wraps authentication-related errors
func (c *Client) wrapAuthError(err error) error {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// Check for specific authentication errors
	if strings.Contains(errMsg, "authentication failed") || 
	   strings.Contains(errMsg, "invalid credentials") ||
	   strings.Contains(errMsg, "login failed") {
		return email.NewEmailError(email.ErrorTypeAuth, 
			"AUTH_FAILED", "authentication failed: invalid credentials", err, false)
	}
	
	if strings.Contains(errMsg, "authentication required") {
		return email.NewEmailError(email.ErrorTypeAuth, 
			"AUTH_REQUIRED", "authentication required", err, false)
	}
	
	if strings.Contains(errMsg, "token") && strings.Contains(errMsg, "expired") {
		return email.NewEmailError(email.ErrorTypeAuth, 
			"AUTH_EXPIRED", "OAuth2 token expired", err, true).WithDetails(map[string]interface{}{
				"requires_refresh": true,
			})
	}
	
	return email.WrapError(err, email.ErrorTypeAuth, "AUTH_ERROR", "authentication error", false)
}