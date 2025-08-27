package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// MailboxModel represents the mailbox view showing email lists and folders
type MailboxModel struct {
	// Core dependencies
	service services.EmailService
	logger  *log.Logger

	// UI state
	width           int
	height          int
	ready           bool
	loading         bool
	error           *services.ServiceError

	// Service state
	serviceState    *services.ServiceState

	// UI components
	messageList     list.Model
	folderList      list.Model
	focusedPane     PaneType
	showFolders     bool

	// Message list state
	messages        []services.MessageInfo
	selectedMessage *services.MessageInfo

	// Folder list state
	folders         []services.FolderInfo
	selectedFolder  *services.FolderInfo

	// Key bindings
	keyMap          MailboxKeyMap
}

// PaneType represents which pane is currently focused
type PaneType int

const (
	PaneFolder PaneType = iota
	PaneMessage
)

// MailboxKeyMap defines key bindings for mailbox operations
type MailboxKeyMap struct {
	// Navigation
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	Enter        key.Binding
	
	// Pane switching
	SwitchPane   key.Binding
	ToggleFolders key.Binding
	
	// Message operations
	MarkRead     key.Binding
	MarkUnread   key.Binding
	Flag         key.Binding
	Delete       key.Binding
	Archive      key.Binding
	
	// View operations
	Compose      key.Binding
	Reply        key.Binding
	Forward      key.Binding
	Refresh      key.Binding
}

// MessageItem represents a message item for the list component
type MessageItem struct {
	services.MessageInfo
}

// FolderItem represents a folder item for the list component
type FolderItem struct {
	services.FolderInfo
}

// Implement list.Item interface for MessageItem
func (i MessageItem) FilterValue() string { return i.Subject }
func (i MessageItem) Title() string       { return i.Subject }
func (i MessageItem) Description() string { 
	return fmt.Sprintf("%s • %s", i.FromDisplay, i.Preview)
}

// Implement list.Item interface for FolderItem
func (i FolderItem) FilterValue() string { return i.Name }
func (i FolderItem) Title() string       { return fmt.Sprintf("%s %s", i.Icon, i.Name) }
func (i FolderItem) Description() string { 
	return fmt.Sprintf("%d messages, %d unread", i.MessageCount, i.UnreadCount)
}

// NewMailboxModel creates a new mailbox model
func NewMailboxModel(service services.EmailService, logger *log.Logger) *MailboxModel {
	// Initialize key bindings
	keyMap := MailboxKeyMap{
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
			key.WithHelp("h/←", "focus folders"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/→", "focus messages"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open message"),
		),
		SwitchPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		ToggleFolders: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle folders"),
		),
		MarkRead: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mark read"),
		),
		MarkUnread: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "mark unread"),
		),
		Flag: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "flag"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Archive: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "archive"),
		),
		Compose: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "compose"),
		),
		Reply: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reply"),
		),
		Forward: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "forward"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
	}

	// Create message list
	messageList := list.New([]list.Item{}, NewMessageDelegate(), 0, 0)
	messageList.Title = "Messages"
	messageList.Styles.Title = styles.TitleStyle
	messageList.SetShowStatusBar(false)
	messageList.SetFilteringEnabled(true)
	messageList.SetShowHelp(false)

	// Create folder list
	folderList := list.New([]list.Item{}, NewFolderDelegate(), 0, 0)
	folderList.Title = "Folders"
	folderList.Styles.Title = styles.TitleStyle
	folderList.SetShowStatusBar(false)
	folderList.SetFilteringEnabled(false)
	folderList.SetShowHelp(false)

	return &MailboxModel{
		service:      service,
		logger:       logger,
		ready:        false,
		loading:      false,
		messageList:  messageList,
		folderList:   folderList,
		focusedPane:  PaneMessage,
		showFolders:  true,
		messages:     make([]services.MessageInfo, 0),
		folders:      make([]services.FolderInfo, 0),
		keyMap:       keyMap,
	}
}

// SetSize updates the model dimensions
func (m *MailboxModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.ready = true

	// Calculate dimensions for panes
	folderWidth := 25
	messageWidth := width - folderWidth - 2

	if !m.showFolders {
		messageWidth = width
		folderWidth = 0
	}

	// Update list dimensions
	contentHeight := height - 4 // Header + footer + margins
	m.messageList.SetSize(messageWidth, contentHeight)
	m.folderList.SetSize(folderWidth, contentHeight)
}

// Update handles messages and updates the mailbox model
func (m *MailboxModel) Update(msg tea.Msg) (*MailboxModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle local key bindings
		switch {
		case key.Matches(msg, m.keyMap.SwitchPane):
			if m.showFolders {
				if m.focusedPane == PaneFolder {
					m.focusedPane = PaneMessage
				} else {
					m.focusedPane = PaneFolder
				}
			}
			return m, nil

		case key.Matches(msg, m.keyMap.ToggleFolders):
			m.showFolders = !m.showFolders
			m.focusedPane = PaneMessage
			m.SetSize(m.width, m.height) // Recalculate dimensions
			return m, nil

		case key.Matches(msg, m.keyMap.Left):
			if m.showFolders {
				m.focusedPane = PaneFolder
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Right):
			m.focusedPane = PaneMessage
			return m, nil

		case key.Matches(msg, m.keyMap.Enter):
			return m.handleEnterKey()

		case key.Matches(msg, m.keyMap.MarkRead):
			return m.handleMarkRead()

		case key.Matches(msg, m.keyMap.MarkUnread):
			return m.handleMarkUnread()

		case key.Matches(msg, m.keyMap.Flag):
			return m.handleFlag()

		case key.Matches(msg, m.keyMap.Delete):
			return m.handleDelete()

		case key.Matches(msg, m.keyMap.Archive):
			return m.handleArchive()

		case key.Matches(msg, m.keyMap.Refresh):
			return m.handleRefresh()
		}

		// Delegate to focused pane
		if m.focusedPane == PaneFolder && m.showFolders {
			m.folderList, cmd = m.folderList.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			m.messageList, cmd = m.messageList.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the mailbox view
func (m *MailboxModel) View() string {
	if !m.ready {
		return "Loading mailbox..."
	}

	// Render header with account info
	header := m.renderHeader()

	// Render main content
	var content string
	if m.showFolders {
		content = m.renderTwoPaneLayout()
	} else {
		content = m.renderSinglePaneLayout()
	}

	// Render footer with status and help
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// UpdateState updates the model with new service state
func (m *MailboxModel) UpdateState(state *services.ServiceState) {
	m.serviceState = state

	// Update messages
	if len(state.Messages) != len(m.messages) {
		m.updateMessageList(state.Messages)
	}

	// Update folders
	if len(state.Folders) != len(m.folders) {
		m.updateFolderList(state.Folders)
	}

	// Update selected message
	if len(state.Messages) > 0 {
		selectedIndex := m.messageList.Index()
		if selectedIndex >= 0 && selectedIndex < len(state.Messages) {
			m.selectedMessage = &state.Messages[selectedIndex]
		}
	}

	// Update selected folder
	if state.CurrentFolder != nil {
		m.selectedFolder = state.CurrentFolder
	}

	// Update loading state
	m.loading = state.Loading
	if state.Error != nil {
		m.error = state.Error
	}
}

// Helper methods

func (m *MailboxModel) updateMessageList(messages []services.MessageInfo) {
	m.messages = messages
	items := make([]list.Item, len(messages))
	for i, msg := range messages {
		items[i] = MessageItem{msg}
	}
	m.messageList.SetItems(items)
}

func (m *MailboxModel) updateFolderList(folders []services.FolderInfo) {
	m.folders = folders
	items := make([]list.Item, len(folders))
	for i, folder := range folders {
		items[i] = FolderItem{folder}
	}
	m.folderList.SetItems(items)
}

func (m *MailboxModel) renderHeader() string {
	if m.serviceState == nil || m.serviceState.CurrentAccount == nil {
		return styles.HeaderStyle.Render("POMBO - No Account")
	}

	account := m.serviceState.CurrentAccount
	accountInfo := fmt.Sprintf("📧 %s (%s)", account.Name, account.Email)
	
	var status string
	if account.Connected {
		status = styles.SuccessStyle.Render("● Connected")
	} else {
		status = styles.ErrorStyle.Render("● Disconnected")
	}

	unreadInfo := ""
	if account.UnreadCount > 0 {
		unreadInfo = styles.WarningStyle.Render(fmt.Sprintf(" • %d unread", account.UnreadCount))
	}

	headerContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		accountInfo,
		"  ",
		status,
		unreadInfo,
	)

	return styles.HeaderStyle.Width(m.width).Render(headerContent)
}

func (m *MailboxModel) renderTwoPaneLayout() string {
	folderPane := m.renderFolderPane()
	messagePane := m.renderMessagePane()

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		folderPane,
		messagePane,
	)
}

func (m *MailboxModel) renderSinglePaneLayout() string {
	return m.renderMessagePane()
}

func (m *MailboxModel) renderFolderPane() string {
	style := styles.SidebarStyle
	if m.focusedPane == PaneFolder {
		style = style.BorderForeground(styles.PrimaryColor)
	}

	return style.Render(m.folderList.View())
}

func (m *MailboxModel) renderMessagePane() string {
	var style lipgloss.Style
	if m.showFolders {
		style = styles.MainPaneStyle
		if m.focusedPane == PaneMessage {
			style = style.Border(lipgloss.NormalBorder()).
				BorderForeground(styles.PrimaryColor)
		}
	} else {
		style = lipgloss.NewStyle().Width(m.width).Height(m.height - 4)
	}

	content := m.messageList.View()
	
	if m.loading {
		loadingMsg := styles.LoadingStyle.Render("Loading messages...")
		content = lipgloss.JoinVertical(lipgloss.Center, content, loadingMsg)
	}

	if m.error != nil {
		errorMsg := styles.ErrorStyle.Render(fmt.Sprintf("Error: %s", m.error.UserMessage))
		content = lipgloss.JoinVertical(lipgloss.Center, content, errorMsg)
	}

	return style.Render(content)
}

func (m *MailboxModel) renderFooter() string {
	var help strings.Builder
	
	if m.focusedPane == PaneFolder && m.showFolders {
		help.WriteString("enter: select folder • ")
	} else {
		help.WriteString("enter: open message • r: reply • d: delete • ")
	}
	
	if m.showFolders {
		help.WriteString("tab: switch pane • ")
	}
	
	help.WriteString("t: toggle folders • ?: help")

	statusInfo := ""
	if m.serviceState != nil && m.serviceState.CurrentFolder != nil {
		folder := m.serviceState.CurrentFolder
		statusInfo = fmt.Sprintf("📁 %s (%d messages)", folder.Name, folder.MessageCount)
	}

	footerContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		statusInfo,
		strings.Repeat(" ", max(0, m.width-lipgloss.Width(statusInfo)-lipgloss.Width(help.String()))),
		styles.SubtleStyle.Render(help.String()),
	)

	return styles.FooterStyle.Width(m.width).Render(footerContent)
}

// Event handlers

func (m *MailboxModel) handleEnterKey() (*MailboxModel, tea.Cmd) {
	if m.focusedPane == PaneFolder && m.showFolders {
		// Select folder
		if item, ok := m.folderList.SelectedItem().(FolderItem); ok {
			return m, m.selectFolder(item.FolderInfo)
		}
	} else {
		// Open message
		if item, ok := m.messageList.SelectedItem().(MessageItem); ok {
			return m, m.openMessage(item.MessageInfo)
		}
	}
	return m, nil
}

func (m *MailboxModel) handleMarkRead() (*MailboxModel, tea.Cmd) {
	if item, ok := m.messageList.SelectedItem().(MessageItem); ok {
		return m, m.markMessageRead(item.MessageInfo)
	}
	return m, nil
}

func (m *MailboxModel) handleMarkUnread() (*MailboxModel, tea.Cmd) {
	if item, ok := m.messageList.SelectedItem().(MessageItem); ok {
		return m, m.markMessageUnread(item.MessageInfo)
	}
	return m, nil
}

func (m *MailboxModel) handleFlag() (*MailboxModel, tea.Cmd) {
	if item, ok := m.messageList.SelectedItem().(MessageItem); ok {
		return m, m.flagMessage(item.MessageInfo)
	}
	return m, nil
}

func (m *MailboxModel) handleDelete() (*MailboxModel, tea.Cmd) {
	if item, ok := m.messageList.SelectedItem().(MessageItem); ok {
		return m, m.deleteMessage(item.MessageInfo)
	}
	return m, nil
}

func (m *MailboxModel) handleArchive() (*MailboxModel, tea.Cmd) {
	if item, ok := m.messageList.SelectedItem().(MessageItem); ok {
		return m, m.archiveMessage(item.MessageInfo)
	}
	return m, nil
}

func (m *MailboxModel) handleRefresh() (*MailboxModel, tea.Cmd) {
	return m, m.refreshMailbox()
}

// Command generators

func (m *MailboxModel) selectFolder(folder services.FolderInfo) tea.Cmd {
	return func() tea.Msg {
		if m.serviceState == nil || m.serviceState.CurrentAccount == nil {
			return nil
		}

		if err := m.service.SelectFolder(m.serviceState.CurrentAccount.ID, folder.Name); err != nil {
			m.logger.Error("Failed to select folder", "folder", folder.Name, "error", err)
		}
		return nil
	}
}

func (m *MailboxModel) openMessage(msg services.MessageInfo) tea.Cmd {
	return func() tea.Msg {
		// Switch to message view
		// This would be handled by the parent EmailModel
		return ViewSwitchMsg{View: ViewMessage, Data: msg}
	}
}

func (m *MailboxModel) markMessageRead(msg services.MessageInfo) tea.Cmd {
	return func() tea.Msg {
		if m.serviceState == nil || m.serviceState.CurrentAccount == nil {
			return nil
		}

		if err := m.service.MarkRead(m.serviceState.CurrentAccount.ID, []string{msg.ID}); err != nil {
			m.logger.Error("Failed to mark message as read", "id", msg.ID, "error", err)
		}
		return nil
	}
}

func (m *MailboxModel) markMessageUnread(msg services.MessageInfo) tea.Cmd {
	return func() tea.Msg {
		if m.serviceState == nil || m.serviceState.CurrentAccount == nil {
			return nil
		}

		if err := m.service.MarkUnread(m.serviceState.CurrentAccount.ID, []string{msg.ID}); err != nil {
			m.logger.Error("Failed to mark message as unread", "id", msg.ID, "error", err)
		}
		return nil
	}
}

func (m *MailboxModel) flagMessage(msg services.MessageInfo) tea.Cmd {
	return func() tea.Msg {
		if m.serviceState == nil || m.serviceState.CurrentAccount == nil {
			return nil
		}

		var err error
		if msg.IsFlagged {
			err = m.service.UnflagMessage(m.serviceState.CurrentAccount.ID, []string{msg.ID})
		} else {
			err = m.service.FlagMessage(m.serviceState.CurrentAccount.ID, []string{msg.ID})
		}

		if err != nil {
			m.logger.Error("Failed to toggle message flag", "id", msg.ID, "error", err)
		}
		return nil
	}
}

func (m *MailboxModel) deleteMessage(msg services.MessageInfo) tea.Cmd {
	return func() tea.Msg {
		if m.serviceState == nil || m.serviceState.CurrentAccount == nil {
			return nil
		}

		if err := m.service.DeleteMessage(m.serviceState.CurrentAccount.ID, []string{msg.ID}); err != nil {
			m.logger.Error("Failed to delete message", "id", msg.ID, "error", err)
		}
		return nil
	}
}

func (m *MailboxModel) archiveMessage(msg services.MessageInfo) tea.Cmd {
	return func() tea.Msg {
		if m.serviceState == nil || m.serviceState.CurrentAccount == nil {
			return nil
		}

		// Move to archive folder
		if err := m.service.MoveMessage(m.serviceState.CurrentAccount.ID, []string{msg.ID}, "Archive"); err != nil {
			m.logger.Error("Failed to archive message", "id", msg.ID, "error", err)
		}
		return nil
	}
}

func (m *MailboxModel) refreshMailbox() tea.Cmd {
	return func() tea.Msg {
		if err := m.service.RefreshState(); err != nil {
			m.logger.Error("Failed to refresh mailbox", "error", err)
		}
		return nil
	}
}

// ViewSwitchMsg represents a request to switch views
type ViewSwitchMsg struct {
	View ViewState
	Data interface{}
}

// Utility function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}