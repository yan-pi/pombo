package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ybarbara/pombo/internal/config"
	"github.com/ybarbara/pombo/internal/email"
)

// Start initializes and starts the email service
func (s *EmailServiceImpl) Start(ctx context.Context) error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if s.running {
		return fmt.Errorf("email service is already running")
	}

	s.logger.Info("Starting email service")

	// Initialize configured accounts
	if err := s.initializeAccounts(); err != nil {
		return fmt.Errorf("failed to initialize accounts: %w", err)
	}

	// Start background monitoring
	s.startBackgroundMonitoring()

	s.running = true
	s.logger.Info("Email service started successfully")

	// Emit service started event
	s.emitUpdate(ServiceUpdate{
		Type: UpdateTypeSyncStarted,
		Data: "Email service started",
	})

	return nil
}

// Stop gracefully shuts down the email service
func (s *EmailServiceImpl) Stop() error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping email service")

	// Cancel context to stop background goroutines
	s.cancel()

	// Wait for background operations to complete
	done := make(chan struct{})
	go func() {
		s.bgWaitGroup.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		s.logger.Info("All background operations stopped")
	case <-time.After(10 * time.Second):
		s.logger.Warn("Background operations did not stop within timeout")
	}

	// Close update channel
	close(s.updateChan)

	s.running = false
	s.logger.Info("Email service stopped")

	return nil
}

// AddAccount adds a new email account to the service
func (s *EmailServiceImpl) AddAccount(account *config.AccountConfig) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Adding account", "id", account.ID, "email", account.Email)

	// Validate account configuration
	if err := s.validateAccountConfig(account); err != nil {
		return fmt.Errorf("invalid account configuration: %w", err)
	}

	// Create authentication provider
	ctx := context.Background()
	auth, err := s.authFactory.CreateProvider(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to create auth provider: %w", err)
	}

	// Test connection
	if err := s.testAccountConnection(account, auth); err != nil {
		return fmt.Errorf("failed to connect to account: %w", err)
	}

	// Create account info
	accountInfo := AccountInfo{
		ID:            account.ID,
		Name:          account.Name,
		Email:         account.Email,
		Provider:      account.Provider,
		Status:        "connected",
		Connected:     true,
		UnreadCount:   0,
		TotalMessages: 0,
		LastSync:      time.Now(),
	}

	// Update service state
	s.updateState(func(state *ServiceState) {
		// Check if account already exists
		for i, existing := range state.Accounts {
			if existing.ID == account.ID {
				state.Accounts[i] = accountInfo
				return
			}
		}
		// Add new account
		state.Accounts = append(state.Accounts, accountInfo)
		
		// Set as current account if it's the first one
		if state.CurrentAccount == nil {
			state.CurrentAccount = &accountInfo
		}
	})

	// Start monitoring for this account
	s.startAccountMonitoring(account.ID)

	// Emit account added event
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeAccountAdded,
		AccountID: account.ID,
		Data:      accountInfo,
	})

	s.logger.Info("Account added successfully", "id", account.ID)
	return nil
}

// addAccountInternal adds an account without checking if the service is running (used during initialization)
func (s *EmailServiceImpl) addAccountInternal(account *config.AccountConfig) error {
	s.logger.Info("Adding account", "id", account.ID, "email", account.Email)

	// Validate account configuration
	if err := s.validateAccountConfig(account); err != nil {
		return fmt.Errorf("invalid account configuration: %w", err)
	}

	// Create authentication provider
	ctx := context.Background()
	auth, err := s.authFactory.CreateProvider(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to create auth provider: %w", err)
	}

	// Test connection
	if err := s.testAccountConnection(account, auth); err != nil {
		return fmt.Errorf("failed to connect to account: %w", err)
	}

	// Create account info
	accountInfo := AccountInfo{
		ID:            account.ID,
		Name:          account.Name,
		Email:         account.Email,
		Provider:      account.Provider,
		Status:        "connected",
		Connected:     true,
		UnreadCount:   0,
		TotalMessages: 0,
		LastSync:      time.Now(),
	}

	// Update service state
	s.updateState(func(state *ServiceState) {
		// Check if account already exists
		for i, existing := range state.Accounts {
			if existing.ID == account.ID {
				state.Accounts[i] = accountInfo
				return
			}
		}
		// Add new account
		state.Accounts = append(state.Accounts, accountInfo)
		
		// Set as current account if it's the first one
		if state.CurrentAccount == nil {
			state.CurrentAccount = &accountInfo
		}
	})

	// Start monitoring for this account
	s.startAccountMonitoring(account.ID)

	// Emit account added event
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeAccountAdded,
		AccountID: account.ID,
		Data:      accountInfo,
	})

	s.logger.Info("Account added successfully", "id", account.ID)
	return nil
}

// RemoveAccount removes an email account from the service
func (s *EmailServiceImpl) RemoveAccount(accountID string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Removing account", "id", accountID)

	// Update service state
	found := false
	s.updateState(func(state *ServiceState) {
		for i, account := range state.Accounts {
			if account.ID == accountID {
				// Remove account from list
				state.Accounts = append(state.Accounts[:i], state.Accounts[i+1:]...)
				found = true
				
				// Update current account if it was removed
				if state.CurrentAccount != nil && state.CurrentAccount.ID == accountID {
					if len(state.Accounts) > 0 {
						state.CurrentAccount = &state.Accounts[0]
					} else {
						state.CurrentAccount = nil
					}
				}
				break
			}
		}
	})

	if !found {
		return fmt.Errorf("account not found: %s", accountID)
	}

	// Close connections for this account
	if err := s.pool.CloseConnection(accountID); err != nil {
		s.logger.Warn("Failed to close connections for account", "id", accountID, "error", err)
	}

	// Emit account removed event
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeAccountRemoved,
		AccountID: accountID,
	})

	s.logger.Info("Account removed successfully", "id", accountID)
	return nil
}

// SwitchAccount switches the current active account
func (s *EmailServiceImpl) SwitchAccount(accountID string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Switching to account", "id", accountID)

	// Find the account
	var targetAccount *AccountInfo
	s.stateMutex.RLock()
	for _, account := range s.state.Accounts {
		if account.ID == accountID {
			accountCopy := account
			targetAccount = &accountCopy
			break
		}
	}
	s.stateMutex.RUnlock()

	if targetAccount == nil {
		return fmt.Errorf("account not found: %s", accountID)
	}

	// Update current account
	s.updateState(func(state *ServiceState) {
		state.CurrentAccount = targetAccount
		// Clear messages and folders when switching accounts
		state.Messages = make([]MessageInfo, 0)
		state.Folders = make([]FolderInfo, 0)
		state.CurrentFolder = nil
	})

	// Load folders for the new account
	go s.refreshAccountData(accountID)

	s.logger.Info("Switched to account", "id", accountID)
	return nil
}

// GetAccounts returns all configured accounts
func (s *EmailServiceImpl) GetAccounts() []AccountInfo {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
	
	// Return a copy to prevent external modification
	accounts := make([]AccountInfo, len(s.state.Accounts))
	copy(accounts, s.state.Accounts)
	return accounts
}

// GetCurrentAccount returns the currently active account
func (s *EmailServiceImpl) GetCurrentAccount() *AccountInfo {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
	
	if s.state.CurrentAccount == nil {
		return nil
	}
	
	// Return a copy
	account := *s.state.CurrentAccount
	return &account
}

// GetFolders returns folders for the specified account
func (s *EmailServiceImpl) GetFolders(accountID string) ([]FolderInfo, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("service is not running")
	}

	// Check cache first
	s.cacheMutex.RLock()
	if cached, exists := s.folderCache[accountID]; exists {
		s.cacheMutex.RUnlock()
		return cached, nil
	}
	s.cacheMutex.RUnlock()

	// Get fresh folder list from backend
	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	folders, err := client.GetFolders(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}

	// Convert to UI-friendly format
	folderInfos := make([]FolderInfo, len(folders))
	for i, folder := range folders {
		folderInfos[i] = FolderInfo{
			Name:         folder.Name,
			FullName:     folder.FullName,
			MessageCount: folder.MessageCount,
			UnreadCount:  folder.UnseenCount,
			Type:         s.determineFolderType(folder.Name),
			Icon:         s.getFolderIcon(folder.Name),
			AccountID:    accountID,
			LastSync:     time.Now(),
		}
	}

	// Update cache
	s.cacheMutex.Lock()
	s.folderCache[accountID] = folderInfos
	s.cacheMutex.Unlock()

	return folderInfos, nil
}

// SelectFolder selects a folder and loads its messages
func (s *EmailServiceImpl) SelectFolder(accountID, folderName string) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Selecting folder", "account", accountID, "folder", folderName)

	// Get client connection
	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	// Select folder
	status, err := client.SelectFolder(s.ctx, folderName)
	if err != nil {
		return fmt.Errorf("failed to select folder: %w", err)
	}

	// Create folder info
	folderInfo := FolderInfo{
		Name:         folderName,
		FullName:     folderName,
		MessageCount: int(status.Messages),
		UnreadCount:  int(status.Unseen),
		Type:         s.determineFolderType(folderName),
		Icon:         s.getFolderIcon(folderName),
		AccountID:    accountID,
		LastSync:     time.Now(),
	}

	// Update service state
	s.updateState(func(state *ServiceState) {
		state.CurrentFolder = &folderInfo
		state.Loading = true
	})

	// Load messages in background
	go s.loadFolderMessages(accountID, folderName, 50)

	s.logger.Info("Folder selected", "account", accountID, "folder", folderName)
	return nil
}

// GetMessages returns messages from the specified folder
func (s *EmailServiceImpl) GetMessages(accountID, folderName string, limit int) ([]MessageInfo, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("service is not running")
	}

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	// Create search criteria for recent messages
	criteria := &email.SearchCriteria{
		Limit: limit,
	}

	messages, err := client.GetMessages(s.ctx, folderName, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Convert to UI-friendly format
	messageInfos := make([]MessageInfo, len(messages))
	for i, msg := range messages {
		messageInfos[i] = s.convertToMessageInfo(msg)
	}

	return messageInfos, nil
}

// SendMessage sends an email message
func (s *EmailServiceImpl) SendMessage(accountID string, msg *OutgoingMessage) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Sending message", "account", accountID, "subject", msg.Subject)

	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.pool.ReleaseConnection(accountID)

	// Convert to backend format
	outgoingMsg := s.convertToOutgoingMessage(msg)

	if err := client.SendMessage(s.ctx, outgoingMsg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	s.logger.Info("Message sent successfully", "account", accountID, "subject", msg.Subject)
	return nil
}

// EnableRealTimeUpdates enables or disables real-time IDLE monitoring
func (s *EmailServiceImpl) EnableRealTimeUpdates(enabled bool) error {
	if !s.IsRunning() {
		return fmt.Errorf("service is not running")
	}

	// Implementation would enable/disable IDLE monitoring
	s.logger.Info("Real-time updates", "enabled", enabled)
	return nil
}

// Helper methods

func (s *EmailServiceImpl) initializeAccounts() error {
	for _, accountConfig := range s.config.Accounts {
		if accountConfig.Enabled {
			if err := s.addAccountInternal(&accountConfig); err != nil {
				s.logger.Error("Failed to initialize account", "id", accountConfig.ID, "error", err)
				// Continue with other accounts
			}
		}
	}
	return nil
}

func (s *EmailServiceImpl) validateAccountConfig(account *config.AccountConfig) error {
	if account.ID == "" {
		return fmt.Errorf("account ID is required")
	}
	if account.Email == "" {
		return fmt.Errorf("account email is required")
	}
	if account.IMAP.Host == "" {
		return fmt.Errorf("IMAP host is required")
	}
	return nil
}

func (s *EmailServiceImpl) testAccountConnection(account *config.AccountConfig, auth email.AuthProvider) error {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	client, err := s.clientFactory.CreateClient(ctx, account, auth)
	if err != nil {
		return err
	}
	defer client.Disconnect(ctx)

	return client.Ping(ctx)
}

func (s *EmailServiceImpl) startBackgroundMonitoring() {
	s.bgWaitGroup.Add(1)
	go func() {
		defer s.bgWaitGroup.Done()
		
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.performPeriodicSync()
			}
		}
	}()
}

func (s *EmailServiceImpl) startAccountMonitoring(accountID string) {
	s.bgWaitGroup.Add(1)
	go func() {
		defer s.bgWaitGroup.Done()
		
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.monitorAccountUpdates(accountID)
				time.Sleep(time.Minute) // Reconnect interval
			}
		}
	}()
}

func (s *EmailServiceImpl) monitorAccountUpdates(accountID string) {
	client, err := s.pool.GetConnection(s.ctx, accountID)
	if err != nil {
		s.logger.Error("Failed to get connection for monitoring", "account", accountID, "error", err)
		return
	}
	defer s.pool.ReleaseConnection(accountID)

	updateChan := make(chan *email.EmailUpdate, 10)
	if err := client.Subscribe(s.ctx, updateChan); err != nil {
		s.logger.Error("Failed to subscribe to updates", "account", accountID, "error", err)
		return
	}
	defer client.Unsubscribe(s.ctx)

	for {
		select {
		case <-s.ctx.Done():
			return
		case update := <-updateChan:
			s.handleEmailUpdate(accountID, update)
		}
	}
}

func (s *EmailServiceImpl) handleEmailUpdate(accountID string, update *email.EmailUpdate) {
	switch update.Type {
	case email.UpdateTypeNewMessage:
		s.handleNewMessage(accountID, update)
	case email.UpdateTypeMessageFlags:
		s.handleMessageFlagsUpdate(accountID, update)
	case email.UpdateTypeMessageDelete:
		s.handleMessageDeleted(accountID, update)
	}
}

func (s *EmailServiceImpl) handleNewMessage(accountID string, update *email.EmailUpdate) {
	if update.Message != nil {
		messageInfo := s.convertToMessageInfo(update.Message)
		
		s.updateState(func(state *ServiceState) {
			// Add to beginning of messages list
			state.Messages = append([]MessageInfo{messageInfo}, state.Messages...)
			// Update unread count
			if !messageInfo.IsRead {
				for i := range state.Accounts {
					if state.Accounts[i].ID == accountID {
						state.Accounts[i].UnreadCount++
						break
					}
				}
			}
		})

		s.emitUpdate(ServiceUpdate{
			Type:      UpdateTypeNewMessage,
			AccountID: accountID,
			Data:      messageInfo,
		})
	}
}

func (s *EmailServiceImpl) handleMessageFlagsUpdate(accountID string, update *email.EmailUpdate) {
	// Implementation for flag updates
	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeMessageFlagged,
		AccountID: accountID,
		MessageID: update.Message.ID,
	})
}

func (s *EmailServiceImpl) handleMessageDeleted(accountID string, update *email.EmailUpdate) {
	s.updateState(func(state *ServiceState) {
		for i, msg := range state.Messages {
			if msg.ID == update.Message.ID {
				state.Messages = append(state.Messages[:i], state.Messages[i+1:]...)
				break
			}
		}
	})

	s.emitUpdate(ServiceUpdate{
		Type:      UpdateTypeMessageDeleted,
		AccountID: accountID,
		MessageID: update.Message.ID,
	})
}

func (s *EmailServiceImpl) performPeriodicSync() {
	s.logger.Debug("Performing periodic sync")
	
	accounts := s.GetAccounts()
	for _, account := range accounts {
		if account.Connected {
			go s.refreshAccountData(account.ID)
		}
	}
}

func (s *EmailServiceImpl) refreshAccountData(accountID string) {
	s.logger.Debug("Refreshing account data", "account", accountID)
	
	// Refresh folders
	if _, err := s.GetFolders(accountID); err != nil {
		s.logger.Error("Failed to refresh folders", "account", accountID, "error", err)
	}
}

func (s *EmailServiceImpl) loadFolderMessages(accountID, folderName string, limit int) {
	messages, err := s.GetMessages(accountID, folderName, limit)
	if err != nil {
		s.logger.Error("Failed to load folder messages", "account", accountID, "folder", folderName, "error", err)
		
		s.updateState(func(state *ServiceState) {
			state.Loading = false
			state.Error = &ServiceError{
				Type:        ErrorTypeOperation,
				Message:     err.Error(),
				UserMessage: "Failed to load messages",
				Retryable:   true,
				Timestamp:   time.Now(),
			}
		})
		return
	}

	s.updateState(func(state *ServiceState) {
		state.Messages = messages
		state.Loading = false
		state.Error = nil
	})

	s.emitUpdate(ServiceUpdate{
		Type:       UpdateTypeFolderRefreshed,
		AccountID:  accountID,
		FolderName: folderName,
		Data:       messages,
	})
}

func (s *EmailServiceImpl) convertToMessageInfo(msg *email.Message) MessageInfo {
	return MessageInfo{
		ID:              msg.ID,
		Subject:         msg.Subject,
		From:            AddressInfo{Name: msg.From.Name, Address: msg.From.Address, Display: s.formatAddress(msg.From)},
		To:              s.convertAddresses(msg.To),
		Date:            msg.Date,
		Preview:         s.generatePreview(msg),
		Size:            msg.Size,
		IsRead:          msg.IsRead,
		IsFlagged:       msg.IsFlagged,
		HasAttachments:  len(msg.Attachments) > 0,
		ThreadID:        msg.ThreadID,
		FolderName:      msg.FolderName,
		AccountID:       msg.AccountID,
		DisplayDate:     s.formatDisplayDate(msg.Date),
		SizeDisplay:     s.formatSize(msg.Size),
		FromDisplay:     s.formatAddress(msg.From),
	}
}

func (s *EmailServiceImpl) convertAddresses(addresses []*email.Address) []AddressInfo {
	result := make([]AddressInfo, len(addresses))
	for i, addr := range addresses {
		result[i] = AddressInfo{
			Name:    addr.Name,
			Address: addr.Address,
			Display: s.formatAddress(addr),
		}
	}
	return result
}

func (s *EmailServiceImpl) formatAddress(addr *email.Address) string {
	if addr.Name != "" {
		return fmt.Sprintf("%s <%s>", addr.Name, addr.Address)
	}
	return addr.Address
}

func (s *EmailServiceImpl) generatePreview(msg *email.Message) string {
	if msg.Body != nil && msg.Body.Text != "" {
		preview := strings.ReplaceAll(msg.Body.Text, "\n", " ")
		if len(preview) > 150 {
			return preview[:150] + "..."
		}
		return preview
	}
	return ""
}

func (s *EmailServiceImpl) formatDisplayDate(date time.Time) string {
	now := time.Now()
	if date.Year() == now.Year() && date.YearDay() == now.YearDay() {
		return date.Format("15:04")
	} else if date.Year() == now.Year() {
		return date.Format("Jan 02")
	}
	return date.Format("2006-01-02")
}

func (s *EmailServiceImpl) formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func (s *EmailServiceImpl) determineFolderType(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "inbox"):
		return "inbox"
	case strings.Contains(name, "sent"):
		return "sent"
	case strings.Contains(name, "draft"):
		return "drafts"
	case strings.Contains(name, "trash") || strings.Contains(name, "deleted"):
		return "trash"
	case strings.Contains(name, "spam") || strings.Contains(name, "junk"):
		return "spam"
	case strings.Contains(name, "archive"):
		return "archive"
	default:
		return "folder"
	}
}

func (s *EmailServiceImpl) getFolderIcon(name string) string {
	switch s.determineFolderType(name) {
	case "inbox":
		return "📥"
	case "sent":
		return "📤"
	case "drafts":
		return "📝"
	case "trash":
		return "🗑️"
	case "spam":
		return "⚠️"
	case "archive":
		return "📦"
	default:
		return "📁"
	}
}

func (s *EmailServiceImpl) convertToOutgoingMessage(msg *OutgoingMessage) *email.OutgoingMessage {
	return &email.OutgoingMessage{
		From:    &email.Address{Name: msg.From.Name, Address: msg.From.Address},
		To:      s.convertToEmailAddresses(msg.To),
		CC:      s.convertToEmailAddresses(msg.CC),
		BCC:     s.convertToEmailAddresses(msg.BCC),
		Subject: msg.Subject,
		Body:    msg.Body,
		BodyHTML: msg.BodyHTML,
		InReplyTo: msg.InReplyTo,
		References: msg.References,
		Encrypt: msg.Encrypt,
		Sign:    msg.Sign,
	}
}

func (s *EmailServiceImpl) convertToEmailAddresses(addresses []AddressInfo) []*email.Address {
	result := make([]*email.Address, len(addresses))
	for i, addr := range addresses {
		result[i] = &email.Address{Name: addr.Name, Address: addr.Address}
	}
	return result
}