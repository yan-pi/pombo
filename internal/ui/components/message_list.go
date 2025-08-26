package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// MessageListKeyMap defines key bindings for the message list
type MessageListKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Select      key.Binding
	ToggleRead  key.Binding
	Flag        key.Binding
	Delete      key.Binding
	Reply       key.Binding
	ReplyAll    key.Binding
	Forward     key.Binding
	Move        key.Binding
	Search      key.Binding
	Refresh     key.Binding
	MultiSelect key.Binding
	SelectAll   key.Binding
}

// DefaultMessageListKeyMap returns the default key bindings for message list
func DefaultMessageListKeyMap() MessageListKeyMap {
	return MessageListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "move down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open message"),
		),
		ToggleRead: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "mark read/unread"),
		),
		Flag: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "flag/unflag"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Reply: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reply"),
		),
		ReplyAll: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "reply all"),
		),
		Forward: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "forward"),
		),
		Move: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "move to folder"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
		MultiSelect: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle selection"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "select all"),
		),
	}
}

// SortCriteria defines message sorting options
type SortCriteria int

const (
	SortByDate SortCriteria = iota
	SortBySender
	SortBySubject
	SortBySize
)

// MessageList represents the message list component
type MessageList struct {
	service       services.EmailService
	accountID     string
	folderName    string
	messages      []services.MessageInfo
	selectedIdx   int
	multiSelect   map[int]bool
	width         int
	height        int
	focused       bool
	keyMap        MessageListKeyMap
	
	// Display options
	sortBy        SortCriteria
	showThreads   bool
	showPreview   bool
	
	// Search state
	searchQuery   string
	searchActive  bool
	searchResults *services.SearchResults
	
	// State tracking
	loading       bool
	error         string
	scrollOffset  int
	maxDisplayed  int
}

// NewMessageList creates a new message list component
func NewMessageList(service services.EmailService) *MessageList {
	return &MessageList{
		service:     service,
		messages:    make([]services.MessageInfo, 0),
		selectedIdx: 0,
		multiSelect: make(map[int]bool),
		focused:     false,
		keyMap:      DefaultMessageListKeyMap(),
		sortBy:      SortByDate,
		showThreads: true,
		showPreview: false,
		loading:     false,
	}
}

// Init initializes the message list component
func (ml *MessageList) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the message list
func (ml *MessageList) Update(msg tea.Msg) (*MessageList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ml.width = msg.Width
		ml.height = msg.Height
		ml.calculateDisplayParameters()
		return ml, nil

	case tea.KeyMsg:
		if !ml.focused {
			return ml, nil
		}

		switch {
		case key.Matches(msg, ml.keyMap.Up):
			if ml.selectedIdx > 0 {
				ml.selectedIdx--
				ml.ensureVisible()
			}
			return ml, nil

		case key.Matches(msg, ml.keyMap.Down):
			if ml.selectedIdx < len(ml.messages)-1 {
				ml.selectedIdx++
				ml.ensureVisible()
			}
			return ml, nil

		case key.Matches(msg, ml.keyMap.Select):
			return ml, ml.openMessage()

		case key.Matches(msg, ml.keyMap.ToggleRead):
			return ml, ml.toggleRead()

		case key.Matches(msg, ml.keyMap.Flag):
			return ml, ml.toggleFlag()

		case key.Matches(msg, ml.keyMap.Delete):
			return ml, ml.deleteMessages()

		case key.Matches(msg, ml.keyMap.Reply):
			return ml, ml.replyMessage(false)

		case key.Matches(msg, ml.keyMap.ReplyAll):
			return ml, ml.replyMessage(true)

		case key.Matches(msg, ml.keyMap.Forward):
			return ml, ml.forwardMessage()

		case key.Matches(msg, ml.keyMap.Move):
			// Future: Open folder selection dialog
			return ml, nil

		case key.Matches(msg, ml.keyMap.Search):
			ml.searchActive = true
			// Future: Open search input
			return ml, nil

		case key.Matches(msg, ml.keyMap.Refresh):
			return ml, ml.refreshMessages()

		case key.Matches(msg, ml.keyMap.MultiSelect):
			ml.toggleSelection()
			return ml, nil

		case key.Matches(msg, ml.keyMap.SelectAll):
			ml.selectAll()
			return ml, nil
		}

	case services.ServiceUpdate:
		// Handle real-time service updates
		switch msg.Type {
		case services.UpdateTypeNewMessage,
			 services.UpdateTypeMessageRead,
			 services.UpdateTypeMessageFlagged,
			 services.UpdateTypeMessageDeleted,
			 services.UpdateTypeMessageMoved:
			if msg.AccountID == ml.accountID && msg.FolderName == ml.folderName {
				return ml, ml.refreshMessages()
			}
		}

	case FolderSelectedMsg:
		ml.accountID = msg.AccountID
		ml.folderName = msg.FolderName
		ml.selectedIdx = 0
		ml.scrollOffset = 0
		ml.clearSelection()
		return ml, ml.refreshMessages()

	case MessagesRefreshedMsg:
		ml.loading = false
		ml.messages = msg.Messages
		ml.error = ""
		
		// Ensure selected index is valid
		if ml.selectedIdx >= len(ml.messages) {
			ml.selectedIdx = len(ml.messages) - 1
		}
		if ml.selectedIdx < 0 {
			ml.selectedIdx = 0
		}
		
		ml.ensureVisible()
		return ml, nil

	case MessageRefreshErrorMsg:
		ml.loading = false
		ml.error = msg.Error
		return ml, nil

	case SearchResultsMsg:
		ml.searchResults = &msg.Results
		ml.messages = msg.Results.Messages
		ml.selectedIdx = 0
		ml.scrollOffset = 0
		return ml, nil
	}

	return ml, nil
}

// View renders the message list
func (ml *MessageList) View() string {
	if ml.width == 0 || ml.height == 0 {
		return ""
	}

	var content strings.Builder
	
	// Header with folder information
	header := ml.renderHeader()
	content.WriteString(header)
	content.WriteString("\n")
	
	// Error display
	if ml.error != "" {
		errorText := styles.ErrorStyle.Render("Error: " + ml.error)
		content.WriteString(errorText)
		content.WriteString("\n")
	}

	// Search results info
	if ml.searchResults != nil {
		searchInfo := ml.renderSearchInfo()
		content.WriteString(searchInfo)
		content.WriteString("\n")
	}

	// Messages
	if len(ml.messages) == 0 {
		if !ml.loading {
			noMessages := styles.SubtleStyle.Render("No messages in this folder")
			content.WriteString(noMessages)
		} else {
			loading := styles.LoadingStyle.Render("Loading messages...")
			content.WriteString(loading)
		}
	} else {
		messagesView := ml.renderMessages()
		content.WriteString(messagesView)
	}

	// Footer with keybindings if focused
	if ml.focused {
		content.WriteString("\n")
		keybindings := ml.renderKeybindings()
		content.WriteString(keybindings)
	}

	// Apply container styling
	containerStyle := styles.MainPaneStyle.
		Width(ml.width).
		Height(ml.height)
	
	if ml.focused {
		containerStyle = containerStyle.
			BorderForeground(styles.PrimaryColor)
	}

	return containerStyle.Render(content.String())
}

// renderHeader renders the message list header
func (ml *MessageList) renderHeader() string {
	var header strings.Builder
	
	// Folder name
	folderName := ml.folderName
	if folderName == "" {
		folderName = "Messages"
	}
	
	if ml.loading {
		folderName += " (Loading...)"
	}
	
	header.WriteString(styles.SubtitleStyle.Render(folderName))
	
	// Message count and selection info
	if len(ml.messages) > 0 {
		countInfo := fmt.Sprintf(" (%d messages", len(ml.messages))
		if len(ml.multiSelect) > 0 {
			countInfo += fmt.Sprintf(", %d selected", len(ml.multiSelect))
		}
		countInfo += ")"
		
		header.WriteString(styles.SubtleStyle.Render(countInfo))
	}
	
	return header.String()
}

// renderSearchInfo renders search results information
func (ml *MessageList) renderSearchInfo() string {
	if ml.searchResults == nil {
		return ""
	}
	
	searchInfo := fmt.Sprintf("Search: %s (%d results in %v)",
		ml.searchResults.Query,
		ml.searchResults.Total,
		ml.searchResults.Took)
	
	return styles.SubtleStyle.Render(searchInfo)
}

// renderMessages renders the message list
func (ml *MessageList) renderMessages() string {
	var content strings.Builder
	
	// Calculate which messages to display
	startIdx := ml.scrollOffset
	endIdx := ml.scrollOffset + ml.maxDisplayed
	if endIdx > len(ml.messages) {
		endIdx = len(ml.messages)
	}
	
	for i := startIdx; i < endIdx; i++ {
		if i < len(ml.messages) {
			messageView := ml.renderMessage(ml.messages[i], i, i == ml.selectedIdx)
			content.WriteString(messageView)
			if i < endIdx-1 {
				content.WriteString("\n")
			}
		}
	}
	
	return content.String()
}

// renderMessage renders a single message item
func (ml *MessageList) renderMessage(msg services.MessageInfo, index int, selected bool) string {
	var content strings.Builder
	
	// Selection indicator
	if _, isSelected := ml.multiSelect[index]; isSelected {
		content.WriteString(lipgloss.NewStyle().Foreground(styles.AccentColor).Render("● "))
	} else {
		content.WriteString("  ")
	}
	
	// Status indicators
	statusIcons := ml.getMessageIcons(msg)
	content.WriteString(statusIcons)
	content.WriteString(" ")
	
	// Sender (truncated to fit)
	senderWidth := 20
	sender := msg.FromDisplay
	if len(sender) > senderWidth {
		sender = sender[:senderWidth-3] + "..."
	}
	
	senderStyle := styles.ListItemStyle
	if !msg.IsRead {
		senderStyle = styles.UnreadStyle
	}
	if selected && ml.focused {
		senderStyle = styles.SelectedListItemStyle
	}
	
	content.WriteString(senderStyle.Render(fmt.Sprintf("%-*s", senderWidth, sender)))
	content.WriteString(" ")
	
	// Subject (truncated to fit remaining width)
	availableWidth := ml.width - senderWidth - 15 // Account for icons, spacing, date
	if availableWidth < 10 {
		availableWidth = 10
	}
	
	subject := msg.Subject
	if subject == "" {
		subject = "(No Subject)"
	}
	if len(subject) > availableWidth {
		subject = subject[:availableWidth-3] + "..."
	}
	
	subjectStyle := styles.ListItemStyle
	if !msg.IsRead {
		subjectStyle = styles.UnreadStyle
	}
	if selected && ml.focused {
		subjectStyle = styles.SelectedListItemStyle
	}
	
	content.WriteString(subjectStyle.Render(subject))
	
	// Date (right-aligned)
	dateStr := ml.formatDate(msg.Date)
	dateStyle := styles.SubtleStyle
	if selected && ml.focused {
		dateStyle = styles.SelectedListItemStyle
	}
	
	// Right-align the date
	padding := ml.width - len(content.String()) - len(dateStr) - 2
	if padding > 0 {
		content.WriteString(strings.Repeat(" ", padding))
	}
	content.WriteString(dateStyle.Render(dateStr))
	
	// Preview line if enabled and space allows
	if ml.showPreview && ml.height > 15 && msg.Preview != "" {
		content.WriteString("\n  ")
		preview := msg.Preview
		if len(preview) > ml.width-4 {
			preview = preview[:ml.width-7] + "..."
		}
		content.WriteString(styles.SubtleStyle.Render(preview))
	}
	
	return content.String()
}

// getMessageIcons returns status icons for a message
func (ml *MessageList) getMessageIcons(msg services.MessageInfo) string {
	var icons strings.Builder
	
	// Read/unread indicator
	if msg.IsRead {
		icons.WriteString(styles.SubtleStyle.Render("○"))
	} else {
		icons.WriteString(styles.UnreadStyle.Render("●"))
	}
	
	// Flagged indicator
	if msg.IsFlagged {
		icons.WriteString(lipgloss.NewStyle().Foreground(styles.WarningColor).Render("⚑"))
	} else {
		icons.WriteString(" ")
	}
	
	// Attachment indicator
	if msg.HasAttachments {
		icons.WriteString(styles.AttachmentStyle.Render("📎"))
	} else {
		icons.WriteString(" ")
	}
	
	// Encryption/signing indicators
	if msg.IsEncrypted {
		icons.WriteString(styles.EncryptedStyle.Render("🔒"))
	} else if msg.IsSigned {
		icons.WriteString(styles.SignedStyle.Render("✓"))
	} else {
		icons.WriteString(" ")
	}
	
	return icons.String()
}

// formatDate formats a date for display in the message list
func (ml *MessageList) formatDate(date time.Time) string {
	now := time.Now()
	
	// Today: show time only
	if date.Format("2006-01-02") == now.Format("2006-01-02") {
		return date.Format("15:04")
	}
	
	// This week: show day name
	if date.After(now.AddDate(0, 0, -7)) {
		return date.Format("Mon")
	}
	
	// This year: show month and day
	if date.Year() == now.Year() {
		return date.Format("Jan 02")
	}
	
	// Older: show year
	return date.Format("2006")
}

// calculateDisplayParameters calculates scrolling and display parameters
func (ml *MessageList) calculateDisplayParameters() {
	if ml.height == 0 {
		return
	}
	
	// Calculate available height for messages
	usedHeight := 3 // Header, spacing, keybindings
	if ml.error != "" {
		usedHeight++
	}
	if ml.searchResults != nil {
		usedHeight++
	}
	
	ml.maxDisplayed = ml.height - usedHeight
	if ml.showPreview {
		ml.maxDisplayed = ml.maxDisplayed / 2 // Each message takes 2 lines
	}
	
	if ml.maxDisplayed < 1 {
		ml.maxDisplayed = 1
	}
}

// ensureVisible ensures the selected message is visible
func (ml *MessageList) ensureVisible() {
	if ml.selectedIdx < ml.scrollOffset {
		ml.scrollOffset = ml.selectedIdx
	} else if ml.selectedIdx >= ml.scrollOffset+ml.maxDisplayed {
		ml.scrollOffset = ml.selectedIdx - ml.maxDisplayed + 1
	}
	
	if ml.scrollOffset < 0 {
		ml.scrollOffset = 0
	}
}

// toggleSelection toggles the selection of the current message
func (ml *MessageList) toggleSelection() {
	if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) {
		if _, exists := ml.multiSelect[ml.selectedIdx]; exists {
			delete(ml.multiSelect, ml.selectedIdx)
		} else {
			ml.multiSelect[ml.selectedIdx] = true
		}
	}
}

// selectAll selects or deselects all messages
func (ml *MessageList) selectAll() {
	if len(ml.multiSelect) == len(ml.messages) {
		// All selected, deselect all
		ml.clearSelection()
	} else {
		// Select all
		for i := range ml.messages {
			ml.multiSelect[i] = true
		}
	}
}

// clearSelection clears all selections
func (ml *MessageList) clearSelection() {
	ml.multiSelect = make(map[int]bool)
}

// getSelectedMessageIDs returns the IDs of selected messages
func (ml *MessageList) getSelectedMessageIDs() []string {
	if len(ml.multiSelect) == 0 {
		// If no multi-selection, use current message
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) {
			return []string{ml.messages[ml.selectedIdx].ID}
		}
		return []string{}
	}
	
	var ids []string
	for idx := range ml.multiSelect {
		if idx < len(ml.messages) {
			ids = append(ids, ml.messages[idx].ID)
		}
	}
	return ids
}

// renderKeybindings renders the keybindings help
func (ml *MessageList) renderKeybindings() string {
	bindings := []string{
		"j/k: navigate",
		"enter: open",
		"u: read/unread",
		"f: flag",
		"d: delete",
		"r: reply",
		"space: select",
		"/: search",
	}
	
	bindingText := strings.Join(bindings, " • ")
	return styles.SubtleStyle.Render(bindingText)
}

// Focus sets focus to the message list
func (ml *MessageList) Focus() {
	ml.focused = true
}

// Blur removes focus from the message list
func (ml *MessageList) Blur() {
	ml.focused = false
}

// Focused returns whether the message list is focused
func (ml *MessageList) Focused() bool {
	return ml.focused
}

// GetSelectedMessage returns the currently selected message
func (ml *MessageList) GetSelectedMessage() *services.MessageInfo {
	if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) {
		return &ml.messages[ml.selectedIdx]
	}
	return nil
}

// SetSize sets the dimensions of the message list
func (ml *MessageList) SetSize(width, height int) {
	ml.width = width
	ml.height = height
	ml.calculateDisplayParameters()
}

// SetFolder sets the current account and folder
func (ml *MessageList) SetFolder(accountID, folderName string) {
	if ml.accountID != accountID || ml.folderName != folderName {
		ml.accountID = accountID
		ml.folderName = folderName
		ml.selectedIdx = 0
		ml.scrollOffset = 0
		ml.clearSelection()
		ml.searchResults = nil
	}
}

// refreshMessages refreshes the message list from the service
func (ml *MessageList) refreshMessages() tea.Cmd {
	if ml.accountID == "" || ml.folderName == "" {
		return nil
	}
	
	ml.loading = true
	
	return func() tea.Msg {
		messages, err := ml.service.GetMessages(ml.accountID, ml.folderName, 100) // Limit to 100 for now
		if err != nil {
			return MessageRefreshErrorMsg{Error: err.Error()}
		}
		
		return MessagesRefreshedMsg{Messages: messages}
	}
}

// Message operations
func (ml *MessageList) openMessage() tea.Cmd {
	if msg := ml.GetSelectedMessage(); msg != nil {
		return func() tea.Msg {
			return MessageOpenRequestMsg{Message: *msg}
		}
	}
	return nil
}

func (ml *MessageList) toggleRead() tea.Cmd {
	ids := ml.getSelectedMessageIDs()
	if len(ids) == 0 {
		return nil
	}
	
	// Determine if we should mark as read or unread
	markAsRead := false
	if ml.selectedIdx < len(ml.messages) {
		markAsRead = !ml.messages[ml.selectedIdx].IsRead
	}
	
	return func() tea.Msg {
		var err error
		if markAsRead {
			err = ml.service.MarkRead(ml.accountID, ids)
		} else {
			err = ml.service.MarkUnread(ml.accountID, ids)
		}
		
		if err != nil {
			return MessageOperationErrorMsg{Error: err.Error()}
		}
		
		// Refresh to show updated status
		return ml.refreshMessages()()
	}
}

func (ml *MessageList) toggleFlag() tea.Cmd {
	ids := ml.getSelectedMessageIDs()
	if len(ids) == 0 {
		return nil
	}
	
	// Determine if we should flag or unflag
	flag := false
	if ml.selectedIdx < len(ml.messages) {
		flag = !ml.messages[ml.selectedIdx].IsFlagged
	}
	
	return func() tea.Msg {
		var err error
		if flag {
			err = ml.service.FlagMessage(ml.accountID, ids)
		} else {
			err = ml.service.UnflagMessage(ml.accountID, ids)
		}
		
		if err != nil {
			return MessageOperationErrorMsg{Error: err.Error()}
		}
		
		// Refresh to show updated status
		return ml.refreshMessages()()
	}
}

func (ml *MessageList) deleteMessages() tea.Cmd {
	ids := ml.getSelectedMessageIDs()
	if len(ids) == 0 {
		return nil
	}
	
	return func() tea.Msg {
		err := ml.service.DeleteMessage(ml.accountID, ids)
		if err != nil {
			return MessageOperationErrorMsg{Error: err.Error()}
		}
		
		// Refresh to show updated list
		return ml.refreshMessages()()
	}
}

func (ml *MessageList) replyMessage(replyAll bool) tea.Cmd {
	if msg := ml.GetSelectedMessage(); msg != nil {
		return func() tea.Msg {
			return MessageReplyRequestMsg{
				Message:  *msg,
				ReplyAll: replyAll,
			}
		}
	}
	return nil
}

func (ml *MessageList) forwardMessage() tea.Cmd {
	if msg := ml.GetSelectedMessage(); msg != nil {
		return func() tea.Msg {
			return MessageForwardRequestMsg{Message: *msg}
		}
	}
	return nil
}

// Note: Message types are now defined in messages.go for shared use across components