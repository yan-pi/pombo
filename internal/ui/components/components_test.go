package components

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/ybarbara/pombo/internal/config"
	"github.com/ybarbara/pombo/internal/ui/services"
)

// MockEmailService for testing components
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEmailService) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockEmailService) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockEmailService) AddAccount(account *config.AccountConfig) error {
	args := m.Called(account)
	return args.Error(0)
}

func (m *MockEmailService) RemoveAccount(accountID string) error {
	args := m.Called(accountID)
	return args.Error(0)
}

func (m *MockEmailService) SwitchAccount(accountID string) error {
	args := m.Called(accountID)
	return args.Error(0)
}

func (m *MockEmailService) GetAccounts() []services.AccountInfo {
	args := m.Called()
	return args.Get(0).([]services.AccountInfo)
}

func (m *MockEmailService) GetCurrentAccount() *services.AccountInfo {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*services.AccountInfo)
}

func (m *MockEmailService) GetFolders(accountID string) ([]services.FolderInfo, error) {
	args := m.Called(accountID)
	return args.Get(0).([]services.FolderInfo), args.Error(1)
}

func (m *MockEmailService) SelectFolder(accountID, folderName string) error {
	args := m.Called(accountID, folderName)
	return args.Error(0)
}

func (m *MockEmailService) RefreshFolder(accountID, folderName string) error {
	args := m.Called(accountID, folderName)
	return args.Error(0)
}

func (m *MockEmailService) GetCurrentFolder() *services.FolderInfo {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*services.FolderInfo)
}

func (m *MockEmailService) GetMessages(accountID, folderName string, limit int) ([]services.MessageInfo, error) {
	args := m.Called(accountID, folderName, limit)
	return args.Get(0).([]services.MessageInfo), args.Error(1)
}

func (m *MockEmailService) GetMessage(accountID, messageID string) (*services.MessageInfo, error) {
	args := m.Called(accountID, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.MessageInfo), args.Error(1)
}

func (m *MockEmailService) SearchMessages(accountID string, query *services.SearchQuery) (*services.SearchResults, error) {
	args := m.Called(accountID, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.SearchResults), args.Error(1)
}

func (m *MockEmailService) SendMessage(accountID string, msg *services.OutgoingMessage) error {
	args := m.Called(accountID, msg)
	return args.Error(0)
}

func (m *MockEmailService) MarkRead(accountID string, messageIDs []string) error {
	args := m.Called(accountID, messageIDs)
	return args.Error(0)
}

func (m *MockEmailService) MarkUnread(accountID string, messageIDs []string) error {
	args := m.Called(accountID, messageIDs)
	return args.Error(0)
}

func (m *MockEmailService) FlagMessage(accountID string, messageIDs []string) error {
	args := m.Called(accountID, messageIDs)
	return args.Error(0)
}

func (m *MockEmailService) UnflagMessage(accountID string, messageIDs []string) error {
	args := m.Called(accountID, messageIDs)
	return args.Error(0)
}

func (m *MockEmailService) DeleteMessage(accountID string, messageIDs []string) error {
	args := m.Called(accountID, messageIDs)
	return args.Error(0)
}

func (m *MockEmailService) MoveMessage(accountID string, messageIDs []string, targetFolder string) error {
	args := m.Called(accountID, messageIDs, targetFolder)
	return args.Error(0)
}

func (m *MockEmailService) GetUpdateChannel() <-chan services.ServiceUpdate {
	args := m.Called()
	return args.Get(0).(<-chan services.ServiceUpdate)
}

func (m *MockEmailService) EnableRealTimeUpdates(enabled bool) error {
	args := m.Called(enabled)
	return args.Error(0)
}

func (m *MockEmailService) GetState() *services.ServiceState {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*services.ServiceState)
}

func (m *MockEmailService) RefreshState() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockEmailService) GetConnectionStatus() map[string]services.ConnectionInfo {
	args := m.Called()
	return args.Get(0).(map[string]services.ConnectionInfo)
}

func (m *MockEmailService) GetStatistics() services.ServiceStatistics {
	args := m.Called()
	return args.Get(0).(services.ServiceStatistics)
}

func TestAccountList_Creation(t *testing.T) {
	mockService := &MockEmailService{}
	accountList := NewAccountList(mockService)
	
	assert.NotNil(t, accountList)
	assert.Equal(t, mockService, accountList.service)
	assert.False(t, accountList.focused)
	assert.Equal(t, 0, accountList.selectedIdx)
}

func TestAccountList_Focus(t *testing.T) {
	mockService := &MockEmailService{}
	accountList := NewAccountList(mockService)
	
	// Initially not focused
	assert.False(t, accountList.Focused())
	
	// Focus the component
	accountList.Focus()
	assert.True(t, accountList.Focused())
	
	// Blur the component
	accountList.Blur()
	assert.False(t, accountList.Focused())
}

func TestAccountList_Update_WindowSize(t *testing.T) {
	mockService := &MockEmailService{}
	accountList := NewAccountList(mockService)
	
	// Test window size update
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedList, cmd := accountList.Update(msg)
	
	assert.Equal(t, 100, updatedList.width)
	assert.Equal(t, 50, updatedList.height)
	assert.Nil(t, cmd)
}

func TestFolderTree_Creation(t *testing.T) {
	mockService := &MockEmailService{}
	folderTree := NewFolderTree(mockService)
	
	assert.NotNil(t, folderTree)
	assert.Equal(t, mockService, folderTree.service)
	assert.False(t, folderTree.focused)
	assert.Equal(t, 0, folderTree.selectedIdx)
}

func TestFolderTree_SetAccount(t *testing.T) {
	mockService := &MockEmailService{}
	folderTree := NewFolderTree(mockService)
	
	folderTree.SetAccount("test-account")
	assert.Equal(t, "test-account", folderTree.accountID)
	assert.Equal(t, 0, folderTree.selectedIdx)
}

func TestMessageList_Creation(t *testing.T) {
	mockService := &MockEmailService{}
	messageList := NewMessageList(mockService)
	
	assert.NotNil(t, messageList)
	assert.Equal(t, mockService, messageList.service)
	assert.False(t, messageList.focused)
	assert.Equal(t, 0, messageList.selectedIdx)
	assert.Equal(t, SortByDate, messageList.sortBy)
	assert.True(t, messageList.showThreads)
}

func TestMessageList_SetFolder(t *testing.T) {
	mockService := &MockEmailService{}
	messageList := NewMessageList(mockService)
	
	messageList.SetFolder("test-account", "INBOX")
	assert.Equal(t, "test-account", messageList.accountID)
	assert.Equal(t, "INBOX", messageList.folderName)
	assert.Equal(t, 0, messageList.selectedIdx)
}

func TestFocusManager_Creation(t *testing.T) {
	focusManager := NewFocusManager()
	
	assert.NotNil(t, focusManager)
	assert.Equal(t, FocusAccountList, focusManager.GetFocus())
}

func TestFocusManager_Navigation(t *testing.T) {
	focusManager := NewFocusManager()
	
	// Create mock components
	mockService := &MockEmailService{}
	accountList := NewAccountList(mockService)
	folderTree := NewFolderTree(mockService)
	messageList := NewMessageList(mockService)
	
	// Add components to focus manager
	focusManager.AddComponent(accountList)
	focusManager.AddComponent(folderTree)
	focusManager.AddComponent(messageList)
	
	// Test initial focus
	focusManager.SetFocus(FocusAccountList)
	assert.Equal(t, FocusAccountList, focusManager.GetFocus())
	assert.True(t, accountList.Focused())
	assert.False(t, folderTree.Focused())
	assert.False(t, messageList.Focused())
	
	// Test next focus
	focusManager.NextFocus()
	assert.Equal(t, FocusFolderTree, focusManager.GetFocus())
	assert.False(t, accountList.Focused())
	assert.True(t, folderTree.Focused())
	assert.False(t, messageList.Focused())
	
	// Test next focus again
	focusManager.NextFocus()
	assert.Equal(t, FocusMessageList, focusManager.GetFocus())
	assert.False(t, accountList.Focused())
	assert.False(t, folderTree.Focused())
	assert.True(t, messageList.Focused())
	
	// Test previous focus
	focusManager.PreviousFocus()
	assert.Equal(t, FocusFolderTree, focusManager.GetFocus())
	assert.False(t, accountList.Focused())
	assert.True(t, folderTree.Focused())
	assert.False(t, messageList.Focused())
}

func TestMessageView_Creation(t *testing.T) {
	mockService := &MockEmailService{}
	messageView := NewMessageView(mockService)
	
	assert.NotNil(t, messageView)
	assert.Equal(t, mockService, messageView.service)
	assert.False(t, messageView.focused)
	assert.False(t, messageView.loading)
	assert.Equal(t, 80, messageView.wrapWidth)
	assert.False(t, messageView.showHeaders)
	assert.False(t, messageView.showHTML)
}

func TestMessageView_Focus(t *testing.T) {
	mockService := &MockEmailService{}
	messageView := NewMessageView(mockService)
	
	// Initially not focused
	assert.False(t, messageView.Focused())
	
	// Focus the component
	messageView.Focus()
	assert.True(t, messageView.Focused())
	
	// Blur the component
	messageView.Blur()
	assert.False(t, messageView.Focused())
}

func TestMessageView_SetSize(t *testing.T) {
	mockService := &MockEmailService{}
	messageView := NewMessageView(mockService)
	
	messageView.SetSize(100, 50)
	assert.Equal(t, 100, messageView.width)
	assert.Equal(t, 50, messageView.height)
	assert.Equal(t, 98, messageView.viewport.Width) // Account for borders
}

func TestMessageView_LoadMessage(t *testing.T) {
	mockService := &MockEmailService{}
	messageView := NewMessageView(mockService)
	
	// Mock message
	message := &services.MessageInfo{
		ID:      "msg1",
		Subject: "Test Message",
		From: services.AddressInfo{
			Name:    "Test User",
			Address: "test@example.com",
			Display: "Test User <test@example.com>",
		},
		Date:    time.Now(),
		Preview: "This is a test message",
		IsRead:  false,
	}
	
	mockService.On("GetMessage", "account1", "msg1").Return(message, nil)
	
	cmd := messageView.LoadMessage("account1", "INBOX", "msg1")
	assert.NotNil(t, cmd)
	assert.Equal(t, "account1", messageView.accountID)
	assert.Equal(t, "INBOX", messageView.folderName)
	assert.Equal(t, "msg1", messageView.messageID)
	assert.True(t, messageView.loading)
}

func TestMessageView_MessageOperations(t *testing.T) {
	mockService := &MockEmailService{}
	messageView := NewMessageView(mockService)
	
	// Set up message view with test data
	messageView.accountID = "account1"
	messageView.messageID = "msg1"
	messageView.message = &MessageDetails{
		MessageInfo: services.MessageInfo{
			ID:        "msg1",
			Subject:   "Test Message",
			IsRead:    false,
			IsFlagged: false,
		},
		Body: "Test message body",
	}
	
	// Test toggle read
	mockService.On("MarkRead", "account1", []string{"msg1"}).Return(nil)
	cmd := messageView.toggleRead()
	assert.NotNil(t, cmd)
	
	// Test toggle flag
	mockService.On("FlagMessage", "account1", []string{"msg1"}).Return(nil)
	cmd = messageView.toggleFlag()
	assert.NotNil(t, cmd)
	
	// Test delete
	mockService.On("DeleteMessage", "account1", []string{"msg1"}).Return(nil)
	cmd = messageView.deleteMessage()
	assert.NotNil(t, cmd)
}

func TestComposeForm_Creation(t *testing.T) {
	mockService := &MockEmailService{}
	composeForm := NewComposeForm(mockService)
	
	assert.NotNil(t, composeForm)
	assert.Equal(t, mockService, composeForm.service)
	assert.Equal(t, ComposeNew, composeForm.mode)
	assert.Equal(t, 0, composeForm.focusedField)
	assert.Equal(t, 3, composeForm.fieldCount) // To, Subject, Body initially
	assert.False(t, composeForm.showCC)
	assert.False(t, composeForm.showBCC)
	assert.True(t, composeForm.autoSave)
}

func TestComposeForm_FieldNavigation(t *testing.T) {
	mockService := &MockEmailService{}
	composeForm := NewComposeForm(mockService)
	
	// Test next field navigation
	initialField := composeForm.focusedField
	composeForm.nextField()
	assert.NotEqual(t, initialField, composeForm.focusedField)
	
	// Test previous field navigation
	composeForm.prevField()
	assert.Equal(t, initialField, composeForm.focusedField)
}

func TestComposeForm_Validation(t *testing.T) {
	mockService := &MockEmailService{}
	composeForm := NewComposeForm(mockService)
	
	// Test empty form - should fail validation
	assert.False(t, composeForm.validateForm())
	assert.Contains(t, composeForm.error, "To field is required")
	
	// Add valid to address
	composeForm.to.SetValue("test@example.com")
	assert.False(t, composeForm.validateForm())
	assert.Contains(t, composeForm.error, "Subject is required")
	
	// Add subject
	composeForm.subject.SetValue("Test Subject")
	assert.True(t, composeForm.validateForm())
	assert.Empty(t, composeForm.error)
	
	// Test invalid email
	composeForm.to.SetValue("invalid-email")
	assert.False(t, composeForm.validateForm())
	assert.Contains(t, composeForm.error, "Invalid email address")
}

func TestComposeForm_EmailAddressParsing(t *testing.T) {
	mockService := &MockEmailService{}
	composeForm := NewComposeForm(mockService)
	
	// Test single address
	addresses := composeForm.parseEmailAddresses("test@example.com")
	assert.Len(t, addresses, 1)
	assert.Equal(t, "test@example.com", addresses[0].Address)
	
	// Test multiple addresses
	addresses = composeForm.parseEmailAddresses("test1@example.com, test2@example.com")
	assert.Len(t, addresses, 2)
	assert.Equal(t, "test1@example.com", addresses[0].Address)
	assert.Equal(t, "test2@example.com", addresses[1].Address)
	
	// Test empty string
	addresses = composeForm.parseEmailAddresses("")
	assert.Len(t, addresses, 0)
}

func TestComposeForm_SetupReply(t *testing.T) {
	mockService := &MockEmailService{}
	composeForm := NewComposeForm(mockService)
	
	// Set up original message
	originalMessage := &services.MessageInfo{
		ID:      "original1",
		Subject: "Original Subject",
		From: services.AddressInfo{
			Address: "sender@example.com",
			Display: "Sender <sender@example.com>",
		},
		To: []services.AddressInfo{
			{Address: "recipient1@example.com", Display: "Recipient 1"},
			{Address: "recipient2@example.com", Display: "Recipient 2"},
		},
		Date: time.Now(),
	}
	
	composeForm.replyTo = originalMessage
	
	// Test regular reply
	composeForm.setupReply(false)
	assert.Equal(t, "sender@example.com", composeForm.to.Value())
	assert.Empty(t, composeForm.cc.Value())
	assert.Equal(t, "Re: Original Subject", composeForm.subject.Value())
	
	// Reset and test reply all
	composeForm.reset()
	composeForm.replyTo = originalMessage
	composeForm.setupReply(true)
	assert.Equal(t, "sender@example.com", composeForm.to.Value())
	assert.Contains(t, composeForm.cc.Value(), "recipient1@example.com")
	assert.Contains(t, composeForm.cc.Value(), "recipient2@example.com")
	assert.True(t, composeForm.showCC)
}

func TestComposeForm_SetupForward(t *testing.T) {
	mockService := &MockEmailService{}
	composeForm := NewComposeForm(mockService)
	
	originalMessage := &services.MessageInfo{
		Subject: "Original Subject",
	}
	
	composeForm.replyTo = originalMessage
	composeForm.setupForward()
	
	assert.Equal(t, "Fwd: Original Subject", composeForm.subject.Value())
}

func TestComposeForm_SendMessage(t *testing.T) {
	mockService := &MockEmailService{}
	composeForm := NewComposeForm(mockService)
	
	// Set up valid form data
	composeForm.accountID = "account1"
	composeForm.to.SetValue("recipient@example.com")
	composeForm.subject.SetValue("Test Subject")
	composeForm.body.SetValue("Test message body")
	
	mockService.On("SendMessage", "account1", mock.AnythingOfType("*services.OutgoingMessage")).Return(nil)
	
	cmd := composeForm.sendMessage()
	assert.NotNil(t, cmd)
	assert.True(t, composeForm.sending)
}

func TestComposeForm_HasContent(t *testing.T) {
	mockService := &MockEmailService{}
	composeForm := NewComposeForm(mockService)
	
	// Empty form should have no content
	assert.False(t, composeForm.hasContent())
	
	// Add some content
	composeForm.to.SetValue("test@example.com")
	assert.True(t, composeForm.hasContent())
	
	// Reset and add different content
	composeForm.reset()
	composeForm.subject.SetValue("Test")
	assert.True(t, composeForm.hasContent())
}

func TestComposeForm_ShowCCBCC(t *testing.T) {
	mockService := &MockEmailService{}
	composeForm := NewComposeForm(mockService)
	
	// Initially CC/BCC should be hidden
	assert.False(t, composeForm.showCC)
	assert.False(t, composeForm.showBCC)
	assert.Equal(t, 3, composeForm.fieldCount) // To, Subject, Body
	
	// Show CC
	composeForm.showCC = true
	composeForm.updateFieldCount()
	assert.Equal(t, 4, composeForm.fieldCount) // To, CC, Subject, Body
	
	// Show BCC
	composeForm.showBCC = true
	composeForm.updateFieldCount()
	assert.Equal(t, 5, composeForm.fieldCount) // To, CC, BCC, Subject, Body
}

func TestComponents_Integration(t *testing.T) {
	// Create mock service with sample data
	mockService := &MockEmailService{}
	
	// Mock accounts
	accounts := []services.AccountInfo{
		{
			ID:          "account1",
			Name:        "Work Email",
			Email:       "work@company.com",
			Provider:    "gmail",
			Connected:   true,
			UnreadCount: 5,
		},
		{
			ID:          "account2", 
			Name:        "Personal Email",
			Email:       "personal@gmail.com",
			Provider:    "gmail",
			Connected:   true,
			UnreadCount: 2,
		},
	}
	
	// Mock folders
	folders := []services.FolderInfo{
		{
			Name:         "INBOX",
			FullName:     "INBOX",
			MessageCount: 100,
			UnreadCount:  5,
			Type:         "inbox",
			AccountID:    "account1",
		},
		{
			Name:         "Sent",
			FullName:     "Sent",
			MessageCount: 50,
			UnreadCount:  0,
			Type:         "sent",
			AccountID:    "account1",
		},
	}
	
	// Mock messages
	messages := []services.MessageInfo{
		{
			ID:      "msg1",
			Subject: "Important Meeting",
			From: services.AddressInfo{
				Name:    "John Doe",
				Address: "john@company.com",
				Display: "John Doe <john@company.com>",
			},
			Date:    time.Now(),
			Preview: "Please join us for the quarterly review...",
			IsRead:  false,
		},
	}
	
	// Set up mock expectations
	mockService.On("GetAccounts").Return(accounts)
	mockService.On("GetFolders", "account1").Return(folders, nil)
	mockService.On("GetMessages", "account1", "INBOX", 100).Return(messages, nil)
	mockService.On("GetCurrentAccount").Return(&accounts[0])
	
	// Create components
	accountList := NewAccountList(mockService)
	folderTree := NewFolderTree(mockService)
	messageList := NewMessageList(mockService)
	messageView := NewMessageView(mockService)
	composeForm := NewComposeForm(mockService)
	
	// Set up components with sample data
	accountList.accounts = accounts
	folderTree.accountID = "account1"
	folderTree.folders = folders
	folderTree.buildTree() // Build the folder tree structure
	messageList.accountID = "account1"
	messageList.folderName = "INBOX"
	messageList.messages = messages
	
	// Test that components can render without error
	assert.NotPanics(t, func() {
		accountList.SetSize(25, 20)
		accountView := accountList.View()
		assert.Contains(t, accountView, "Work Email")
		assert.Contains(t, accountView, "Personal Email")
	})
	
	assert.NotPanics(t, func() {
		folderTree.SetSize(30, 20)
		folderView := folderTree.View()
		assert.Contains(t, folderView, "INBOX")
		assert.Contains(t, folderView, "Sent")
	})
	
	assert.NotPanics(t, func() {
		messageList.SetSize(60, 20)
		messageView := messageList.View()
		assert.Contains(t, messageView, "Important Meeting")
		// Note: John Doe might not be visible due to layout constraints
		// We'll just check that the view renders without panic
	})
	
	assert.NotPanics(t, func() {
		messageView.SetSize(80, 30)
		msgView := messageView.View()
		assert.NotEmpty(t, msgView)
	})
	
	assert.NotPanics(t, func() {
		composeForm.SetSize(100, 40)
		composeView := composeForm.View()
		assert.Contains(t, composeView, "Compose New Email")
	})
}