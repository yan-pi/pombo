package services

import (
	"fmt"
	"time"

	"github.com/ybarbara/pombo/internal/email"
)

// Message management operations

// MarkRead marks the specified messages as read
func (s *EmailServiceImpl) MarkRead(accountID string, messageIDs []string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Marking messages as read", "account", accountID, "count", len(messageIDs))

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	if err := client.MarkRead(s.ctx, messageIDs); err != nil {
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}

	// Update local state
	s.updateMessageFlags(messageIDs, func(msg *MessageInfo) {
		msg.IsRead = true
	})

	// Update account unread count
	s.updateAccountUnreadCount(accountID, -len(messageIDs))

	// Emit update
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeMessageRead,
		AccountID: accountID,
		Data:      messageIDs,
	})

	return nil
}

// MarkUnread marks the specified messages as unread
func (s *EmailServiceImpl) MarkUnread(accountID string, messageIDs []string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Marking messages as unread", "account", accountID, "count", len(messageIDs))

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	if err := client.MarkUnread(s.ctx, messageIDs); err != nil {
		return fmt.Errorf("failed to mark messages as unread: %w", err)
	}

	// Update local state
	s.updateMessageFlags(messageIDs, func(msg *MessageInfo) {
		msg.IsRead = false
	})

	// Update account unread count
	s.updateAccountUnreadCount(accountID, len(messageIDs))

	// Emit update
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeMessageRead,
		AccountID: accountID,
		Data:      messageIDs,
	})

	return nil
}

// FlagMessage flags the specified messages
func (s *EmailServiceImpl) FlagMessage(accountID string, messageIDs []string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Flagging messages", "account", accountID, "count", len(messageIDs))

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	if err := client.SetFlag(s.ctx, messageIDs, email.FlagFlagged); err != nil {
		return fmt.Errorf("failed to flag messages: %w", err)
	}

	// Update local state
	s.updateMessageFlags(messageIDs, func(msg *MessageInfo) {
		msg.IsFlagged = true
	})

	// Emit update
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeMessageFlagged,
		AccountID: accountID,
		Data:      messageIDs,
	})

	return nil
}

// UnflagMessage removes flags from the specified messages
func (s *EmailServiceImpl) UnflagMessage(accountID string, messageIDs []string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Unflagging messages", "account", accountID, "count", len(messageIDs))

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	if err := client.RemoveFlag(s.ctx, messageIDs, email.FlagFlagged); err != nil {
		return fmt.Errorf("failed to unflag messages: %w", err)
	}

	// Update local state
	s.updateMessageFlags(messageIDs, func(msg *MessageInfo) {
		msg.IsFlagged = false
	})

	// Emit update
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeMessageFlagged,
		AccountID: accountID,
		Data:      messageIDs,
	})

	return nil
}

// DeleteMessage deletes the specified messages
func (s *EmailServiceImpl) DeleteMessage(accountID string, messageIDs []string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Deleting messages", "account", accountID, "count", len(messageIDs))

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	// Delete messages one by one (could be optimized for batch operations)
	for _, messageID := range messageIDs {
		if err := client.DeleteMessage(s.ctx, messageID); err != nil {
			s.logger.Error("Failed to delete message", "id", messageID, "error", err)
			continue
		}
	}

	// Remove from local state
	s.removeMessagesFromState(messageIDs)

	// Emit update
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeMessageDeleted,
		AccountID: accountID,
		Data:      messageIDs,
	})

	return nil
}

// MoveMessage moves the specified messages to a target folder
func (s *EmailServiceImpl) MoveMessage(accountID string, messageIDs []string, targetFolder string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Moving messages", "account", accountID, "count", len(messageIDs), "target", targetFolder)

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	// Move messages one by one (could be optimized for batch operations)
	for _, messageID := range messageIDs {
		if err := client.MoveMessage(s.ctx, messageID, targetFolder); err != nil {
			s.logger.Error("Failed to move message", "id", messageID, "error", err)
			continue
		}
	}

	// Update local state (remove from current view if not viewing target folder)
	currentState := s.GetState()
	if currentState.CurrentFolder == nil || currentState.CurrentFolder.Name != targetFolder {
		s.removeMessagesFromState(messageIDs)
	}

	// Emit update
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeMessageMoved,
		AccountID: accountID,
		Data: map[string]interface{}{
			"messageIDs":   messageIDs,
			"targetFolder": targetFolder,
		},
	})

	return nil
}

// GetMessage retrieves a specific message with full content
func (s *EmailServiceImpl) GetMessage(accountID, messageID string) (*MessageInfo, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("service is not running")
	}

	// Check cache first
	s.cacheMutex.RLock()
	if cached, exists := s.messageCache[messageID]; exists {
		s.cacheMutex.RUnlock()
		return cached, nil
	}
	s.cacheMutex.RUnlock()

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	message, err := client.GetMessage(s.ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	messageInfo := s.convertToMessageInfo(message)

	// Cache the full message
	s.cacheMutex.Lock()
	s.messageCache[messageID] = &messageInfo
	s.cacheMutex.Unlock()

	// Clean up cache if it's getting too large
	go s.cleanupCache()

	return &messageInfo, nil
}

// SearchMessages performs a search across messages
func (s *EmailServiceImpl) SearchMessages(accountID string, query *SearchQuery) (*SearchResults, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("service is not running")
	}

	s.logger.Info("Searching messages", "account", accountID, "query", query.Query)

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	// Convert to backend search criteria
	criteria := &email.SearchCriteria{
		Query:      query.Query,
		From:       query.From,
		To:         query.To,
		Subject:    query.Subject,
		Since:      query.DateFrom,
		Before:     query.DateTo,
		Limit:      query.Limit,
		Folder:     query.FolderName,
	}

	startTime := time.Now()
	messages, err := client.GetMessages(s.ctx, query.FolderName, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	searchDuration := time.Since(startTime)

	// Convert to UI format
	messageInfos := make([]MessageInfo, len(messages))
	for i, msg := range messages {
		messageInfos[i] = s.convertToMessageInfo(msg)
	}

	results := &SearchResults{
		Messages:  messageInfos,
		Total:     len(messageInfos),
		Query:     query.Query,
		Took:      searchDuration,
		AccountID: accountID,
	}

	// Emit search results
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeSearchResults,
		AccountID: accountID,
		Data:      results,
	})

	return results, nil
}

// RefreshFolder refreshes the current folder contents
func (s *EmailServiceImpl) RefreshFolder(accountID, folderName string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Refreshing folder", "account", accountID, "folder", folderName)

	// Clear cache for this folder
	s.cacheMutex.Lock()
	delete(s.folderCache, accountID)
	s.cacheMutex.Unlock()

	// Reload folder data
	go s.loadFolderMessages(accountID, folderName, 50)

	return nil
}

// GetCurrentFolder returns the currently selected folder
func (s *EmailServiceImpl) GetCurrentFolder() *FolderInfo {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()

	if s.state.CurrentFolder == nil {
		return nil
	}

	// Return a copy
	folder := *s.state.CurrentFolder
	return &folder
}

// GetConnectionStatus returns connection status for all accounts
func (s *EmailServiceImpl) GetConnectionStatus() map[string]ConnectionInfo {
	status := make(map[string]ConnectionInfo)

	accounts := s.GetAccounts()
	for _, account := range accounts {
		connStatus := s.pool.GetConnectionStatus(account.ID)
		
		connectionInfo := ConnectionInfo{
			AccountID:    account.ID,
			Status:       account.Status,
			Connected:    account.Connected,
			ResponseTime: 0, // Would be populated from pool stats
			ErrorCount:   0, // Would be populated from pool stats
		}

		if connStatus != nil {
			connectionInfo.LastPing = connStatus.LastPing
			if connStatus.LastError != nil {
				errorMsg := connStatus.LastError.Error()
				connectionInfo.LastError = &errorMsg
			}
		}

		status[account.ID] = connectionInfo
	}

	return status
}

// GetStatistics returns service performance statistics
func (s *EmailServiceImpl) GetStatistics() ServiceStatistics {
	state := s.GetState()
	poolStats := s.pool.GetPoolStats()

	connectedAccounts := 0
	totalUnread := 0
	for _, account := range state.Accounts {
		if account.Connected {
			connectedAccounts++
		}
		totalUnread += account.UnreadCount
	}

	return ServiceStatistics{
		ActiveConnections:   poolStats.ActiveConnections,
		TotalAccounts:      len(state.Accounts),
		ConnectedAccounts:  connectedAccounts,
		TotalMessages:      state.TotalMessages,
		TotalUnread:        totalUnread,
		AverageResponseTime: poolStats.AverageLatency,
		CacheHitRate:       s.calculateCacheHitRate(),
		ErrorRate:          poolStats.ErrorRate,
		LastSync:           state.LastSync,
	}
}

// RefreshState forces a refresh of the service state
func (s *EmailServiceImpl) RefreshState() error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Refreshing service state")

	// Refresh all accounts in parallel
	accounts := s.GetAccounts()
	for _, account := range accounts {
		go s.refreshAccountData(account.ID)
	}

	return nil
}

// Helper methods for state management

func (s *EmailServiceImpl) updateMessageFlags(messageIDs []string, updateFunc func(*MessageInfo)) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	for i := range s.state.Messages {
		for _, msgID := range messageIDs {
			if s.state.Messages[i].ID == msgID {
				updateFunc(&s.state.Messages[i])
				break
			}
		}
	}

	s.state.LastUpdate = time.Now()
}

func (s *EmailServiceImpl) updateAccountUnreadCount(accountID string, delta int) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	for i := range s.state.Accounts {
		if s.state.Accounts[i].ID == accountID {
			s.state.Accounts[i].UnreadCount += delta
			if s.state.Accounts[i].UnreadCount < 0 {
				s.state.Accounts[i].UnreadCount = 0
			}
			break
		}
	}

	s.state.LastUpdate = time.Now()
}

func (s *EmailServiceImpl) removeMessagesFromState(messageIDs []string) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	messageSet := make(map[string]bool)
	for _, id := range messageIDs {
		messageSet[id] = true
	}

	filteredMessages := make([]MessageInfo, 0, len(s.state.Messages))
	for _, msg := range s.state.Messages {
		if !messageSet[msg.ID] {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	s.state.Messages = filteredMessages
	s.state.LastUpdate = time.Now()
}

func (s *EmailServiceImpl) cleanupCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Simple cache cleanup - remove entries older than cache expiry
	now := time.Now()
	for key := range s.messageCache {
		// This is a simplified cleanup - in a real implementation,
		// you'd track cache entry timestamps
		_ = now
		// For now, just limit cache size
		if len(s.messageCache) > 1000 {
			delete(s.messageCache, key)
			break
		}
	}
}

func (s *EmailServiceImpl) calculateCacheHitRate() float64 {
	// Simplified cache hit rate calculation
	// In a real implementation, you'd track hits and misses
	return 0.85 // Placeholder value
}