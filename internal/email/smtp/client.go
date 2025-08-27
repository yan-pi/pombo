// Package smtp provides SMTP protocol implementation
package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ybarbara/pombo/internal/email"
)

// Client implements the email.SMTPClient interface
type Client struct {
	// Configuration
	config *email.SMTPConfig
	auth   email.AuthProvider
	
	// Connection state
	client *smtp.Client
	mu     sync.RWMutex
	
	// Server information
	serverName string
	extensions map[string]string
	
	// Connection metadata
	connectedAt time.Time
}

// NewClient creates a new SMTP client instance
func NewClient() *Client {
	return &Client{
		extensions: make(map[string]string),
	}
}

// Connect establishes a connection to the SMTP server
func (c *Client) Connect(ctx context.Context, config *email.SMTPConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if config == nil {
		return email.NewEmailError(email.ErrorTypeValidation, "CONFIG_REQUIRED", "SMTP config is required", nil, false)
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
	var conn net.Conn
	var err error
	
	if config.TLS {
		// Direct TLS connection (usually port 465)
		tlsConfig := &tls.Config{
			ServerName: config.Host,
			MinVersion: tls.VersionTLS12,
		}
		
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, tlsConfig)
	} else {
		// Plain connection (usually port 25 or 587)
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	
	if err != nil {
		return c.wrapConnectionError(err)
	}
	
	// Create SMTP client
	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		conn.Close()
		return email.WrapError(err, email.ErrorTypeProtocol, "CLIENT_CREATE_FAILED", "failed to create SMTP client", false)
	}
	
	c.client = client
	c.connectedAt = time.Now()
	
	// Get server info
	if err := c.updateServerInfo(); err != nil {
		c.client.Close()
		c.client = nil
		return err
	}
	
	// Start TLS if required and not already encrypted
	if config.StartTLS && !config.TLS {
		if err := c.startTLS(); err != nil {
			c.client.Close()
			c.client = nil
			return err
		}
	}
	
	return nil
}

// startTLS initiates STARTTLS if supported
func (c *Client) startTLS() error {
	// Check if STARTTLS is supported
	if supported, _ := c.Extension("STARTTLS"); !supported {
		return email.NewEmailError(email.ErrorTypeSecurity, "STARTTLS_UNSUPPORTED",
			"STARTTLS not supported by server", nil, false)
	}
	
	tlsConfig := &tls.Config{
		ServerName: c.config.Host,
		MinVersion: tls.VersionTLS12,
	}
	
	if err := c.client.StartTLS(tlsConfig); err != nil {
		return email.NewEmailError(email.ErrorTypeSecurity, "STARTTLS_FAILED",
			"STARTTLS failed", err, false).WithDetails(map[string]interface{}{
				"host": c.config.Host,
				"port": c.config.Port,
				"error": err.Error(),
			})
	}
	
	// Update server info after STARTTLS
	return c.updateServerInfo()
}

// Authenticate authenticates with the SMTP server
func (c *Client) Authenticate(ctx context.Context, auth email.AuthProvider) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.client == nil {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_CONNECTED", "not connected to server", nil, false)
	}
	
	if auth == nil {
		return email.NewEmailError(email.ErrorTypeValidation, "AUTH_REQUIRED", "auth provider is required", nil, false)
	}
	
	c.auth = auth
	
	// Get credentials
	creds, err := auth.GetCredentials(ctx)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeAuth, "CREDENTIALS_GET_FAILED", "failed to get credentials", false)
	}
	
	// Create appropriate SMTP auth mechanism
	var smtpAuth smtp.Auth
	
	switch creds.Type {
	case email.AuthTypePassword:
		if creds.Username == "" || creds.Password == "" {
			return email.NewEmailError(email.ErrorTypeAuth, "CREDENTIALS_INCOMPLETE",
				"username and password required", nil, false)
		}
		
		// Determine best auth mechanism based on server capabilities
		if supported, _ := c.Extension("AUTH"); supported {
			authMechs := strings.ToUpper(c.extensions["AUTH"])
			if strings.Contains(authMechs, "CRAM-MD5") {
				smtpAuth = smtp.CRAMMD5Auth(creds.Username, creds.Password)
			} else if strings.Contains(authMechs, "PLAIN") {
				smtpAuth = smtp.PlainAuth("", creds.Username, creds.Password, c.config.Host)
			} else {
				return email.NewEmailError(email.ErrorTypeAuth, "AUTH_UNSUPPORTED",
					"no supported authentication mechanism", nil, false)
			}
		} else {
			// Default to PLAIN if no AUTH capability advertised
			smtpAuth = smtp.PlainAuth("", creds.Username, creds.Password, c.config.Host)
		}
		
	case email.AuthTypeOAuth2:
		if creds.Token == nil {
			return email.NewEmailError(email.ErrorTypeAuth, "OAUTH_TOKEN_REQUIRED", "OAuth2 token required", nil, false)
		}
		
		// Check if token is expired and refresh if needed
		if time.Now().After(creds.Token.ExpiresAt) {
			if err := auth.RefreshIfNeeded(ctx); err != nil {
				return email.WrapError(err, email.ErrorTypeAuth, "TOKEN_REFRESH_FAILED", "failed to refresh OAuth2 token", true)
			}
			
			// Get updated credentials
			creds, err = auth.GetCredentials(ctx)
			if err != nil {
				return email.WrapError(err, email.ErrorTypeAuth, "CREDENTIALS_GET_FAILED", "failed to get refreshed credentials", false)
			}
		}
		
		// Use OAuth2 authentication
		username := creds.Username
		if username == "" && c.config != nil {
			username = c.config.Username
		}
		
		smtpAuth = &oauth2Auth{
			username: username,
			token:    creds.Token.AccessToken,
		}
		
	default:
		return email.NewEmailError(email.ErrorTypeAuth, "UNSUPPORTED_AUTH_TYPE",
			fmt.Sprintf("unsupported auth type: %s", creds.Type), nil, false)
	}
	
	// Perform authentication
	if err := c.client.Auth(smtpAuth); err != nil {
		return c.wrapAuthError(err)
	}
	
	return nil
}

// oauth2Auth implements smtp.Auth for OAuth2
type oauth2Auth struct {
	username string
	token    string
}

func (a *oauth2Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	// Use XOAUTH2 mechanism
	resp := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.username, a.token)
	return "XOAUTH2", []byte(resp), nil
}

func (a *oauth2Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// Server wants more data, but XOAUTH2 should be one-shot
		return nil, fmt.Errorf("unexpected server response during OAuth2 authentication")
	}
	return nil, nil
}

// Quit closes the SMTP session
func (c *Client) Quit() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.client == nil {
		return nil
	}
	
	err := c.client.Quit()
	c.client = nil
	
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "QUIT_FAILED", "QUIT command failed", false)
	}
	
	return nil
}

// Extension checks if the server supports a specific extension
func (c *Client) Extension(name string) (bool, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Check extensions map regardless of connection state for testing
	value, exists := c.extensions[strings.ToUpper(name)]
	if exists {
		return true, value
	}
	
	// If not in our cache and we have a live client, check it
	if c.client != nil {
		return c.client.Extension(name)
	}
	
	return false, ""
}

// ServerName returns the server name
func (c *Client) ServerName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.serverName
}

// Mail sends the MAIL FROM command
func (c *Client) Mail(ctx context.Context, from string, opts *email.MailOptions) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.client == nil {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_CONNECTED", "not connected to server", nil, false)
	}
	
	if from == "" {
		return email.NewEmailError(email.ErrorTypeValidation, "SENDER_REQUIRED", "sender address is required", nil, false)
	}
	
	err := c.client.Mail(from)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "MAIL_FAILED", "MAIL command failed", true)
	}
	
	return nil
}

// Rcpt sends the RCPT TO command
func (c *Client) Rcpt(ctx context.Context, to string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.client == nil {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_CONNECTED", "not connected to server", nil, false)
	}
	
	if to == "" {
		return email.NewEmailError(email.ErrorTypeValidation, "RECIPIENT_REQUIRED", "recipient address is required", nil, false)
	}
	
	err := c.client.Rcpt(to)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "RCPT_FAILED", "RCPT command failed", true)
	}
	
	return nil
}

// Data returns a writer for the message data
func (c *Client) Data(ctx context.Context) (io.WriteCloser, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.client == nil {
		return nil, email.NewEmailError(email.ErrorTypeProtocol, "NOT_CONNECTED", "not connected to server", nil, false)
	}
	
	writer, err := c.client.Data()
	if err != nil {
		return nil, email.WrapError(err, email.ErrorTypeProtocol, "DATA_FAILED", "DATA command failed", true)
	}
	
	return writer, nil
}

// SendMail sends a complete email message
func (c *Client) SendMail(ctx context.Context, from string, to []string, msg []byte) error {
	if len(to) == 0 {
		return email.NewEmailError(email.ErrorTypeValidation, "RECIPIENTS_REQUIRED", "at least one recipient is required", nil, false)
	}
	
	// Send MAIL FROM
	if err := c.Mail(ctx, from, nil); err != nil {
		return err
	}
	
	// Send RCPT TO for each recipient
	for _, recipient := range to {
		if err := c.Rcpt(ctx, recipient); err != nil {
			return err
		}
	}
	
	// Send message data
	writer, err := c.Data(ctx)
	if err != nil {
		return err
	}
	
	defer writer.Close()
	
	_, err = writer.Write(msg)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "WRITE_FAILED", "failed to write message data", true)
	}
	
	err = writer.Close()
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "CLOSE_FAILED", "failed to close message data", true)
	}
	
	return nil
}

// Reset sends the RSET command
func (c *Client) Reset() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.client == nil {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_CONNECTED", "not connected to server", nil, false)
	}
	
	err := c.client.Reset()
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "RSET_FAILED", "RSET command failed", true)
	}
	
	return nil
}

// Noop sends the NOOP command
func (c *Client) Noop() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.client == nil {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_CONNECTED", "not connected to server", nil, false)
	}
	
	err := c.client.Noop()
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "NOOP_FAILED", "NOOP command failed", true)
	}
	
	return nil
}

// Verify sends the VRFY command
func (c *Client) Verify(addr string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.client == nil {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_CONNECTED", "not connected to server", nil, false)
	}
	
	err := c.client.Verify(addr)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "VRFY_FAILED", "VRFY command failed", true)
	}
	
	return nil
}

// Close closes the connection
func (c *Client) Close() error {
	return c.Quit()
}

// updateServerInfo fetches and caches server information
func (c *Client) updateServerInfo() error {
	if c.client == nil {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_CONNECTED", "not connected", nil, false)
	}
	
	// Set server name from config
	c.serverName = c.config.Host
	
	// Get all supported extensions
	if ok, _ := c.client.Extension("EHLO"); ok {
		// Server supports EHLO, get extensions
		c.extensions = make(map[string]string)
		
		// Common extensions to check
		extList := []string{"AUTH", "STARTTLS", "PIPELINING", "8BITMIME", "SIZE", "DSN"}
		for _, ext := range extList {
			if supported, params := c.client.Extension(ext); supported {
				c.extensions[ext] = params
			}
		}
	}
	
	return nil
}

// wrapConnectionError wraps connection-related errors
func (c *Client) wrapConnectionError(err error) error {
	if err == nil {
		return nil
	}
	
	// Check for specific error types
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			return email.NewEmailError(email.ErrorTypeTimeout, "CONNECTION_TIMEOUT",
				"connection timeout", err, true).WithDetails(map[string]interface{}{
					"host": c.config.Host,
					"port": c.config.Port,
				})
		}
	}
	
	// Check for TLS errors
	if strings.Contains(err.Error(), "tls") || strings.Contains(err.Error(), "certificate") {
		return email.NewEmailError(email.ErrorTypeSecurity, "TLS_FAILED",
			"TLS connection failed", err, false).WithDetails(map[string]interface{}{
				"host": c.config.Host,
				"port": c.config.Port,
				"error": err.Error(),
			})
	}
	
	return email.NewEmailError(email.ErrorTypeNetwork, "CONNECTION_FAILED",
		"connection failed", err, true).WithDetails(map[string]interface{}{
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
		return email.NewEmailError(email.ErrorTypeAuth, "AUTH_FAILED",
			"authentication failed: invalid credentials", err, false)
	}
	
	if strings.Contains(errMsg, "authentication required") {
		return email.NewEmailError(email.ErrorTypeAuth, "AUTH_REQUIRED",
			"authentication required", err, false)
	}
	
	if strings.Contains(errMsg, "535") { // 535 Authentication credentials invalid
		return email.NewEmailError(email.ErrorTypeAuth, "AUTH_FAILED",
			"authentication failed: invalid credentials", err, false)
	}
	
	return email.WrapError(err, email.ErrorTypeAuth, "AUTH_ERROR", "authentication error", false)
}