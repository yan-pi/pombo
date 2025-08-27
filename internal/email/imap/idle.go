package imap

import (
	"context"
	"time"

	idle "github.com/emersion/go-imap-idle"
	"github.com/ybarbara/pombo/internal/email"
)

// Idle starts IDLE mode for real-time updates
func (c *Client) Idle(ctx context.Context, updates chan<- *email.EmailUpdate) error {
	c.mu.RLock()
	if c.state < email.StateSelected {
		c.mu.RUnlock()
		return email.NewEmailError(email.ErrorTypeProtocol, "NO_MAILBOX_SELECTED", "no mailbox selected", nil, false)
	}
	
	if !c.idleSupported {
		c.mu.RUnlock()
		return email.NewEmailError(email.ErrorTypeProtocol, "IDLE_UNSUPPORTED", "IDLE not supported by server", nil, false)
	}
	c.mu.RUnlock()
	
	// Setup IDLE context with cancellation
	c.idleMu.Lock()
	if c.idleCancel != nil {
		c.idleCancel() // Cancel any existing IDLE
	}
	
	idleCtx, cancel := context.WithCancel(ctx)
	c.idleCancel = cancel
	c.idleMu.Unlock()
	
	// Defer cleanup
	defer func() {
		c.idleMu.Lock()
		if c.idleCancel != nil {
			c.idleCancel = nil
		}
		c.idleMu.Unlock()
		cancel()
	}()
	
	// Setup update handlers
	updateHandler := &idleUpdateHandler{
		client:  c,
		updates: updates,
		ctx:     idleCtx,
	}
	
	// Start IDLE command
	c.mu.Lock()
	c.state = email.StateIdle
	
	// Create IDLE client
	idleClient := idle.NewClient(c.client)
	
	c.mu.Unlock()
	
	// Handle updates in background (polling-based since go-imap v1 doesn't have real-time updates)
	go updateHandler.handlePolling()
	
	// Start IDLE
	stop := make(chan struct{})
	done := make(chan error, 1)
	
	go func() {
		done <- idleClient.IdleWithFallback(stop, 0)
	}()
	
	// Wait for IDLE to complete or context cancellation
	select {
	case <-idleCtx.Done():
		// Context cancelled, stop IDLE
		c.mu.Lock()
		if c.state == email.StateIdle {
			c.state = email.StateSelected
		}
		c.mu.Unlock()
		
		// Send DONE to stop IDLE
		close(stop)
		return idleCtx.Err()
		
	case err := <-done:
		// IDLE command completed
		c.mu.Lock()
		if c.state == email.StateIdle {
			c.state = email.StateSelected
		}
		c.mu.Unlock()
		
		if err != nil {
			// Send error update
			select {
			case updates <- &email.EmailUpdate{
				Type:      email.UpdateTypeError,
				Error:     email.WrapError(err, email.ErrorTypeProtocol, "IDLE_FAILED", "IDLE command failed", true),
				Timestamp: time.Now(),
			}:
			case <-idleCtx.Done():
			}
			return email.WrapError(err, email.ErrorTypeProtocol, "IDLE_FAILED", "IDLE command failed", true)
		}
		
		return nil
	}
}

// idleUpdateHandler handles IDLE updates
type idleUpdateHandler struct {
	client  *Client
	updates chan<- *email.EmailUpdate
	ctx     context.Context
}

// handlePolling handles polling-based updates during IDLE
func (h *idleUpdateHandler) handlePolling() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-h.ctx.Done():
			return
			
		case <-ticker.C:
			// Send periodic connection status updates
			h.sendPingUpdate()
		}
	}
}

// sendPingUpdate sends a connection status update
func (h *idleUpdateHandler) sendPingUpdate() {
	update := &email.EmailUpdate{
		Type:      email.UpdateTypeConnection,
		Timestamp: time.Now(),
	}
	
	// Get account ID if available
	if h.client.config != nil {
		update.AccountID = h.client.config.Username
	}
	
	select {
	case h.updates <- update:
	case <-h.ctx.Done():
	}
}

// MonitorUpdates sets up continuous monitoring for email updates
// This function can be called independently of IDLE for polling-based updates
func (c *Client) MonitorUpdates(ctx context.Context, folderName string, updates chan<- *email.EmailUpdate, interval time.Duration) error {
	if interval < 30*time.Second {
		interval = 30 * time.Second // Minimum polling interval
	}
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	var lastUIDNext uint32
	var lastMessages uint32
	
	// Get initial state
	status, err := c.Select(ctx, folderName)
	if err != nil {
		return err
	}
	
	lastUIDNext = status.UIDNext
	lastMessages = status.Messages
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
			
		case <-ticker.C:
			// Check for changes
			currentStatus, err := c.Examine(ctx, folderName)
			if err != nil {
				// Send error update
				update := &email.EmailUpdate{
					Type:       email.UpdateTypeError,
					FolderName: folderName,
					Error:      err,
					Timestamp:  time.Now(),
				}
				
				select {
				case updates <- update:
				case <-ctx.Done():
					return ctx.Err()
				}
				continue
			}
			
			// Check for new messages
			if currentStatus.Messages > lastMessages || currentStatus.UIDNext > lastUIDNext {
				// Fetch new messages
				newMessages, err := c.fetchNewMessages(ctx, lastUIDNext, currentStatus.UIDNext)
				if err == nil && len(newMessages) > 0 {
					for _, msg := range newMessages {
						update := &email.EmailUpdate{
							Type:       email.UpdateTypeNewMessage,
							FolderName: folderName,
							Message:    msg,
							Timestamp:  time.Now(),
						}
						
						if c.config != nil {
							update.AccountID = c.config.Username
						}
						
						select {
						case updates <- update:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				}
				
				lastUIDNext = currentStatus.UIDNext
				lastMessages = currentStatus.Messages
			}
		}
	}
}

// fetchNewMessages fetches messages with UIDs between lastUID and currentUID
func (c *Client) fetchNewMessages(ctx context.Context, lastUID, currentUID uint32) ([]*email.Message, error) {
	if currentUID <= lastUID {
		return nil, nil
	}
	
	// Create UID range for new messages
	uids := make([]uint32, 0, currentUID-lastUID)
	for uid := lastUID; uid < currentUID; uid++ {
		uids = append(uids, uid)
	}
	
	// Fetch the new messages
	return c.Fetch(ctx, uids, []string{"ENVELOPE", "FLAGS", "INTERNALDATE", "RFC822.SIZE", "UID"})
}

// IsIdleSupported returns whether the server supports IDLE
func (c *Client) IsIdleSupported() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.idleSupported
}

// StopIdle stops any active IDLE operation
func (c *Client) StopIdle() {
	c.cancelIdle()
}