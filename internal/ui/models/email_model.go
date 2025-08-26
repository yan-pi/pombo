// Package models provides TUI models that integrate with the email service layer
package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/ybarbara/pombo/internal/config"
	"github.com/ybarbara/pombo/internal/ui/components"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// EmailModel represents the main email application model that integrates with the service layer
type EmailModel struct {
	// Core dependencies
	service     services.EmailService
	config      *config.Config
	logger      *log.Logger

	// UI state
	currentView ViewState
	width       int
	height      int
	ready       bool
	loading     bool
	error       *services.ServiceError

	// Service state binding
	serviceState *services.ServiceState
	lastUpdate   time.Time

	// Core email components
	accountList  *components.AccountList
	folderTree   *components.FolderTree
	messageList  *components.MessageList
	focusManager *components.FocusManager

	// Sub-models and components for different views
	mailboxModel  *MailboxModel
	messageView   *components.MessageView
	composeForm   *components.ComposeForm
	accountModel  *AccountModel

	// Update channel for service events
	updateChan <-chan services.ServiceUpdate

	// Key bindings
	keyMap       EmailKeyMap
	globalKeyMap components.GlobalKeyMap
}

// ViewState represents the current view state of the email application
type ViewState int

const (
	ViewMailbox ViewState = iota
	ViewMessage
	ViewCompose
	ViewAccounts
	ViewSettings
	ViewSearch
)

// EmailKeyMap defines key bindings for email operations
type EmailKeyMap struct {
	// Navigation
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Enter       key.Binding
	Back        key.Binding

	// Email operations
	Compose     key.Binding
	Reply       key.Binding
	ReplyAll    key.Binding
	Forward     key.Binding
	Delete      key.Binding
	Archive     key.Binding
	Flag        key.Binding
	MarkRead    key.Binding
	MarkUnread  key.Binding

	// View switching
	NextAccount key.Binding
	PrevAccount key.Binding
	NextFolder  key.Binding
	PrevFolder  key.Binding
	Search      key.Binding
	Refresh     key.Binding

	// Application
	Help        key.Binding
	Quit        key.Binding
}

// ServiceUpdateMsg wraps service updates for the TUI
type ServiceUpdateMsg struct {
	Update services.ServiceUpdate
}

// ServiceStateMsg represents a service state update
type ServiceStateMsg struct {
	State *services.ServiceState
}

// LoadingMsg represents loading state changes
type LoadingMsg struct {
	Loading bool
	Message string
}

// ErrorMsg represents error state changes
type ErrorMsg struct {
	Error *services.ServiceError
}

// NewEmailModel creates a new email model with service integration
func NewEmailModel(
	service services.EmailService,
	config *config.Config,
	logger *log.Logger,
) *EmailModel {
	// Initialize key bindings
	keyMap := EmailKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/←", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/→", "move right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Compose: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "compose"),
		),
		Reply: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reply"),
		),
		ReplyAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "reply all"),
		),
		Forward: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "forward"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Archive: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "archive"),
		),
		Flag: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "flag"),
		),
		MarkRead: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mark read"),
		),
		MarkUnread: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "mark unread"),
		),
		NextAccount: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next account"),
		),
		PrevAccount: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "prev account"),
		),
		NextFolder: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next folder"),
		),
		PrevFolder: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev folder"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}

	model := &EmailModel{
		service:      service,
		config:       config,
		logger:       logger,
		currentView:  ViewMailbox,
		ready:        false,
		loading:      false,
		serviceState: &services.ServiceState{},
		keyMap:       keyMap,
		globalKeyMap: components.DefaultGlobalKeyMap(),
	}

	// Initialize core email components
	model.accountList = components.NewAccountList(service)
	model.folderTree = components.NewFolderTree(service)
	model.messageList = components.NewMessageList(service)

	// Initialize focus manager
	model.focusManager = components.NewFocusManager()
	model.focusManager.AddComponent(model.accountList)
	model.focusManager.AddComponent(model.folderTree)
	model.focusManager.AddComponent(model.messageList)

	// Set initial focus
	model.focusManager.SetFocus(components.FocusAccountList)

	// Initialize sub-models and components
	model.mailboxModel = NewMailboxModel(service, logger)
	model.messageView = components.NewMessageView(service)
	model.composeForm = components.NewComposeForm(service)
	model.accountModel = NewAccountModel(service, logger)

	return model
}

// Init initializes the email model and starts the service
func (m *EmailModel) Init() tea.Cmd {
	// Initialize components
	accountCmd := m.accountList.Init()
	folderCmd := m.folderTree.Init()
	messageCmd := m.messageList.Init()
	messageViewCmd := m.messageView.Init()
	composeFormCmd := m.composeForm.Init()

	// Start the email service
	return tea.Batch(
		accountCmd,
		folderCmd,
		messageCmd,
		messageViewCmd,
		composeFormCmd,
		m.startService(),
		m.listenForUpdates(),
		m.loadInitialState(),
	)
}

// Update handles messages and updates the email model
func (m *EmailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Calculate component dimensions
		accountWidth := 25
		folderWidth := 30
		messageWidth := msg.Width - accountWidth - folderWidth - 4 // Account for borders
		contentHeight := msg.Height - 4 // Account for header and footer

		// Update core components with calculated dimensions
		m.accountList.SetSize(accountWidth, contentHeight)
		m.folderTree.SetSize(folderWidth, contentHeight)
		m.messageList.SetSize(messageWidth, contentHeight)

		// Update sub-models and components with new dimensions
		m.mailboxModel.SetSize(msg.Width, msg.Height)
		m.messageView.SetSize(msg.Width, msg.Height)
		m.composeForm.SetSize(msg.Width, msg.Height)
		m.accountModel.SetSize(msg.Width, msg.Height)

		return m, nil

	case ServiceUpdateMsg:
		return m.handleServiceUpdate(msg.Update)

	case ServiceStateMsg:
		m.serviceState = msg.State
		m.lastUpdate = time.Now()
		return m, nil

	case LoadingMsg:
		m.loading = msg.Loading
		return m, nil

	case ErrorMsg:
		m.error = msg.Error
		return m, nil

	// Handle component-specific messages
	case components.AccountSwitchedMsg:
		m.logger.Info("Account switched", "account_id", msg.AccountID)
		// Update folder tree with new account
		m.folderTree.SetAccount(msg.AccountID)
		return m, nil

	case components.FolderSelectedMsg:
		m.logger.Info("Folder selected", "account_id", msg.AccountID, "folder", msg.FolderName)
		// Update message list with new folder
		m.messageList.SetFolder(msg.AccountID, msg.FolderName)
		return m, nil

	case components.MessageOpenRequestMsg:
		m.logger.Info("Message open requested", "message_id", msg.Message.ID)
		// Switch to message view and load message
		m.currentView = ViewMessage
		m.messageView.Focus()
		// Get current account and folder from service
		currentAccount := m.service.GetCurrentAccount()
		currentFolder := m.service.GetCurrentFolder()
		if currentAccount != nil && currentFolder != nil {
			return m, m.messageView.LoadMessage(currentAccount.ID, currentFolder.Name, msg.Message.ID)
		}
		return m, nil

	case components.MessageReplyRequestMsg:
		m.logger.Info("Reply requested", "message_id", msg.Message.ID)
		// Switch to compose view and setup reply
		m.currentView = ViewCompose
		m.composeForm.Focus()
		currentAccount := m.service.GetCurrentAccount()
		if currentAccount != nil {
			return m, func() tea.Msg {
				return components.ComposeReplyMsg{
					AccountID:       currentAccount.ID,
					OriginalMessage: msg.Message,
				}
			}
		}
		return m, nil

	case components.MessageForwardRequestMsg:
		m.logger.Info("Forward requested", "message_id", msg.Message.ID)
		// Switch to compose view and setup forward
		m.currentView = ViewCompose
		m.composeForm.Focus()
		currentAccount := m.service.GetCurrentAccount()
		if currentAccount != nil {
			return m, func() tea.Msg {
				return components.ComposeForwardMsg{
					AccountID:       currentAccount.ID,
					OriginalMessage: msg.Message,
				}
			}
		}
		return m, nil

	// Handle component-specific messages
	case components.BackToListRequestMsg:
		m.currentView = ViewMailbox
		m.messageView.Blur()
		m.composeForm.Blur()
		return m, nil

	case components.ComposeCompletedMsg:
		m.currentView = ViewMailbox
		m.composeForm.Blur()
		return m, nil

	case components.ComposeCancelledMsg:
		m.currentView = ViewMailbox
		m.composeForm.Blur()
		return m, nil

	case components.MessageLoadErrorMsg:
		m.logger.Error("Message load error", "error", msg.Error)
		m.error = &services.ServiceError{
			Type:        services.ErrorTypeOperation,
			Message:     msg.Error,
			UserMessage: "Failed to load message",
			Retryable:   true,
			Timestamp:   time.Now(),
		}
		return m, nil

	case components.MessageSendErrorMsg:
		m.logger.Error("Message send error", "error", msg.Error)
		m.error = &services.ServiceError{
			Type:        services.ErrorTypeOperation,
			Message:     msg.Error,
			UserMessage: "Failed to send message",
			Retryable:   true,
			Timestamp:   time.Now(),
		}
		return m, nil

	case tea.KeyMsg:
		// Handle global key bindings first
		if cmd := m.handleGlobalKeys(msg); cmd != nil {
			return m, cmd
		}

		// Handle component-specific keys in mailbox view
		if m.currentView == ViewMailbox {
			// Update core components based on focus
			var accountList *components.AccountList
			var folderTree *components.FolderTree
			var messageList *components.MessageList

			accountList, cmd = m.accountList.Update(msg)
			m.accountList = accountList
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			folderTree, cmd = m.folderTree.Update(msg)
			m.folderTree = folderTree
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			messageList, cmd = m.messageList.Update(msg)
			m.messageList = messageList
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			// Delegate to current view for non-mailbox views
			switch m.currentView {
			case ViewMessage:
				m.messageView, cmd = m.messageView.Update(msg)
			case ViewCompose:
				m.composeForm, cmd = m.composeForm.Update(msg)
			case ViewAccounts:
				m.accountModel, cmd = m.accountModel.Update(msg)
			}

			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Update sub-models with current service state
	m.updateSubModels()

	return m, tea.Batch(cmds...)
}

// View renders the current email view
func (m *EmailModel) View() string {
	if !m.ready {
		return "Initializing POMBO Email Client..."
	}

	// Render based on current view
	switch m.currentView {
	case ViewMailbox:
		return m.renderMailboxView()
	case ViewMessage:
		return m.messageView.View()
	case ViewCompose:
		return m.composeForm.View()
	case ViewAccounts:
		return m.accountModel.View()
	default:
		return m.renderMailboxView()
	}
}

// renderMailboxView renders the main three-pane mailbox layout
func (m *EmailModel) renderMailboxView() string {
	if m.width == 0 || m.height == 0 {
		return "Waiting for terminal size..."
	}

	// Calculate pane widths
	accountWidth := 25
	folderWidth := 30
	messageWidth := m.width - accountWidth - folderWidth - 4

	// Ensure minimum widths
	if messageWidth < 40 {
		accountWidth = 20
		folderWidth = 25
		messageWidth = m.width - accountWidth - folderWidth - 4
	}

	// Render components
	accountView := m.accountList.View()
	folderView := m.folderTree.View()
	messageView := m.messageList.View()

	// Create three-pane layout
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styles.SidebarStyle.Width(accountWidth).Render(accountView),
		styles.SidebarStyle.Width(folderWidth).Render(folderView),
		styles.MainPaneStyle.Width(messageWidth).Render(messageView),
	)

	// Add header and footer
	header := m.renderHeader()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderHeader renders the application header
func (m *EmailModel) renderHeader() string {
	var headerParts []string

	// Application title
	title := "POMBO - Terminal Email Client"
	headerParts = append(headerParts, title)

	// Current account info
	if account := m.service.GetCurrentAccount(); account != nil {
		accountInfo := account.Name
		if account.UnreadCount > 0 {
			accountInfo += " (" + fmt.Sprintf("%d unread", account.UnreadCount) + ")"
		}
		headerParts = append(headerParts, accountInfo)
	}

	// Loading indicator
	if m.loading {
		headerParts = append(headerParts, "Loading...")
	}

	// Error indicator
	if m.error != nil {
		errorText := "Error: " + m.error.UserMessage
		headerParts = append(headerParts, errorText)
	}

	headerText := strings.Join(headerParts, " | ")
	return styles.HeaderStyle.Width(m.width).Render(headerText)
}

// renderFooter renders the application footer with key bindings
func (m *EmailModel) renderFooter() string {
	var footerParts []string

	// Current focus indicator
	focusText := "Focus: " + m.focusManager.GetFocus().String()
	footerParts = append(footerParts, focusText)

	// Quick help
	help := []string{
		"Tab: Switch Panel",
		"c: Compose",
		"?: Help",
		"q: Quit",
	}
	footerParts = append(footerParts, strings.Join(help, " • "))

	footerText := strings.Join(footerParts, " | ")
	return styles.FooterStyle.Width(m.width).Render(footerText)
}

// Helper methods

func (m *EmailModel) startService() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := m.service.Start(ctx); err != nil {
			m.logger.Error("Failed to start email service", "error", err)
			return ErrorMsg{
				Error: &services.ServiceError{
					Type:        services.ErrorTypeConfiguration,
					Message:     err.Error(),
					UserMessage: "Failed to start email service",
					Retryable:   true,
					Timestamp:   time.Now(),
				},
			}
		}
		return LoadingMsg{Loading: false, Message: "Email service started"}
	}
}

func (m *EmailModel) listenForUpdates() tea.Cmd {
	return func() tea.Msg {
		// Get the service update channel
		updateChan := m.service.GetUpdateChannel()
		
		// Listen for the first update
		select {
		case update := <-updateChan:
			return ServiceUpdateMsg{Update: update}
		}
	}
}

func (m *EmailModel) loadInitialState() tea.Cmd {
	return func() tea.Msg {
		// Load initial service state
		state := m.service.GetState()
		return ServiceStateMsg{State: state}
	}
}

func (m *EmailModel) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.globalKeyMap.Quit):
		return m.quitApplication()

	case key.Matches(msg, m.globalKeyMap.Compose):
		m.currentView = ViewCompose
		m.composeForm.Focus()
		return func() tea.Msg {
			return components.ComposeNewMsg{
				AccountID: m.service.GetCurrentAccount().ID,
			}
		}

	case key.Matches(msg, m.globalKeyMap.Tab):
		// Switch focus to next component in mailbox view
		if m.currentView == ViewMailbox {
			m.focusManager.NextFocus()
			return nil
		}

	case key.Matches(msg, m.globalKeyMap.ShiftTab):
		// Switch focus to previous component in mailbox view
		if m.currentView == ViewMailbox {
			m.focusManager.PreviousFocus()
			return nil
		}

	case key.Matches(msg, m.globalKeyMap.Refresh):
		return m.refreshCurrentView()

	case key.Matches(msg, m.globalKeyMap.Search):
		m.currentView = ViewSearch
		return nil

	case key.Matches(msg, m.keyMap.Back):
		return m.navigateBack()
	}

	return nil
}

func (m *EmailModel) handleServiceUpdate(update services.ServiceUpdate) (tea.Model, tea.Cmd) {
	m.logger.Debug("Received service update", "type", update.Type, "account", update.AccountID)

	switch update.Type {
	case services.UpdateTypeNewMessage:
		// Handle new message notification
		m.logger.Info("New message received", "account", update.AccountID)
		
	case services.UpdateTypeAccountConnected:
		// Handle account connection
		m.logger.Info("Account connected", "account", update.AccountID)
		
	case services.UpdateTypeError:
		// Handle service errors
		if update.Error != nil {
			m.error = update.Error
		}
	}

	// Continue listening for more updates
	return m, m.listenForUpdates()
}

func (m *EmailModel) updateSubModels() {
	// Update all sub-models with current service state
	if m.serviceState != nil {
		m.mailboxModel.UpdateState(m.serviceState)
		m.accountModel.UpdateState(m.serviceState)
		// Note: messageView and composeForm don't have UpdateState methods
		// as they manage their own state through the service
	}
}

func (m *EmailModel) quitApplication() tea.Cmd {
	return func() tea.Msg {
		// Stop the email service gracefully
		if err := m.service.Stop(); err != nil {
			m.logger.Error("Failed to stop email service", "error", err)
		}
		return tea.Quit()
	}
}

func (m *EmailModel) switchToNextAccount() tea.Cmd {
	return func() tea.Msg {
		accounts := m.service.GetAccounts()
		if len(accounts) <= 1 {
			return nil
		}

		current := m.service.GetCurrentAccount()
		if current == nil {
			return nil
		}

		// Find next account
		for i, account := range accounts {
			if account.ID == current.ID {
				nextIndex := (i + 1) % len(accounts)
				if err := m.service.SwitchAccount(accounts[nextIndex].ID); err != nil {
					m.logger.Error("Failed to switch account", "error", err)
				}
				break
			}
		}

		return nil
	}
}

func (m *EmailModel) switchToPrevAccount() tea.Cmd {
	return func() tea.Msg {
		accounts := m.service.GetAccounts()
		if len(accounts) <= 1 {
			return nil
		}

		current := m.service.GetCurrentAccount()
		if current == nil {
			return nil
		}

		// Find previous account
		for i, account := range accounts {
			if account.ID == current.ID {
				prevIndex := (i - 1 + len(accounts)) % len(accounts)
				if err := m.service.SwitchAccount(accounts[prevIndex].ID); err != nil {
					m.logger.Error("Failed to switch account", "error", err)
				}
				break
			}
		}

		return nil
	}
}

func (m *EmailModel) refreshCurrentView() tea.Cmd {
	return func() tea.Msg {
		if err := m.service.RefreshState(); err != nil {
			m.logger.Error("Failed to refresh state", "error", err)
			return ErrorMsg{
				Error: &services.ServiceError{
					Type:        services.ErrorTypeOperation,
					Message:     err.Error(),
					UserMessage: "Failed to refresh",
					Retryable:   true,
					Timestamp:   time.Now(),
				},
			}
		}
		return LoadingMsg{Loading: false, Message: "Refreshed"}
	}
}

func (m *EmailModel) navigateBack() tea.Cmd {
	switch m.currentView {
	case ViewMessage:
		m.currentView = ViewMailbox
	case ViewCompose:
		m.currentView = ViewMailbox
	case ViewAccounts:
		m.currentView = ViewMailbox
	case ViewSearch:
		m.currentView = ViewMailbox
	}
	return nil
}

// Public methods for view switching

// SwitchToView changes the current view
func (m *EmailModel) SwitchToView(view ViewState) {
	m.currentView = view
}

// GetCurrentView returns the current view state
func (m *EmailModel) GetCurrentView() ViewState {
	return m.currentView
}

// GetServiceState returns the current service state
func (m *EmailModel) GetServiceState() *services.ServiceState {
	return m.serviceState
}

// IsLoading returns whether the model is currently loading
func (m *EmailModel) IsLoading() bool {
	return m.loading
}

// GetError returns the current error state
func (m *EmailModel) GetError() *services.ServiceError {
	return m.error
}

// ClearError clears the current error state
func (m *EmailModel) ClearError() {
	m.error = nil
}