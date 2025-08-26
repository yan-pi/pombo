package components

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// MessageViewKeyMap defines key bindings for the message view
type MessageViewKeyMap struct {
	ScrollUp     key.Binding
	ScrollDown   key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Reply        key.Binding
	ReplyAll     key.Binding
	Forward      key.Binding
	Delete       key.Binding
	Archive      key.Binding
	ToggleRead   key.Binding
	Flag         key.Binding
	SaveAttach   key.Binding
	ViewThread   key.Binding
	NextInThread key.Binding
	PrevInThread key.Binding
	Back         key.Binding
}

// DefaultMessageViewKeyMap returns the default key bindings for message view
func DefaultMessageViewKeyMap() MessageViewKeyMap {
	return MessageViewKeyMap{
		ScrollUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("b", "pgup"),
			key.WithHelp("b/pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("f", "pgdn"),
			key.WithHelp("f/pgdn", "page down"),
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
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Archive: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "archive"),
		),
		ToggleRead: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "toggle read/unread"),
		),
		Flag: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "flag/unflag"),
		),
		SaveAttach: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "save attachment"),
		),
		ViewThread: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "view thread"),
		),
		NextInThread: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next in thread"),
		),
		PrevInThread: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "previous in thread"),
		),
		Back: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q/esc", "back to list"),
		),
	}
}

// MessageDetails and ThreadInfo are now defined in messages.go

// MessageView represents the message view component
type MessageView struct {
	service       services.EmailService
	message       *MessageDetails
	viewport      viewport.Model
	accountID     string
	folderName    string
	messageID     string
	width         int
	height        int
	focused       bool
	loading       bool
	error         string
	keyMap        MessageViewKeyMap
	
	// Display options
	showHeaders   bool
	showHTML      bool
	wrapWidth     int
	
	// Thread navigation
	threadMode    bool
	threadIndex   int
	
	// Action state
	actionsHeight int
}

// NewMessageView creates a new message view component
func NewMessageView(service services.EmailService) *MessageView {
	vp := viewport.New(0, 0)
	vp.Style = styles.ContentStyle
	
	return &MessageView{
		service:       service,
		viewport:      vp,
		focused:       false,
		loading:       false,
		keyMap:        DefaultMessageViewKeyMap(),
		showHeaders:   false,
		showHTML:      false,
		wrapWidth:     80,
		threadMode:    false,
		actionsHeight: 3,
	}
}

// Init initializes the message view component
func (mv *MessageView) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the message view
func (mv *MessageView) Update(msg tea.Msg) (*MessageView, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		mv.width = msg.Width
		mv.height = msg.Height
		mv.updateDimensions()
		return mv, nil

	case tea.KeyMsg:
		if !mv.focused {
			return mv, nil
		}

		switch {
		case key.Matches(msg, mv.keyMap.ScrollUp):
			mv.viewport.LineUp(1)
			return mv, nil

		case key.Matches(msg, mv.keyMap.ScrollDown):
			mv.viewport.LineDown(1)
			return mv, nil

		case key.Matches(msg, mv.keyMap.PageUp):
			mv.viewport.HalfViewUp()
			return mv, nil

		case key.Matches(msg, mv.keyMap.PageDown):
			mv.viewport.HalfViewDown()
			return mv, nil

		case key.Matches(msg, mv.keyMap.Reply):
			return mv, mv.replyMessage(false)

		case key.Matches(msg, mv.keyMap.ReplyAll):
			return mv, mv.replyMessage(true)

		case key.Matches(msg, mv.keyMap.Forward):
			return mv, mv.forwardMessage()

		case key.Matches(msg, mv.keyMap.Delete):
			return mv, mv.deleteMessage()

		case key.Matches(msg, mv.keyMap.Archive):
			return mv, mv.archiveMessage()

		case key.Matches(msg, mv.keyMap.ToggleRead):
			return mv, mv.toggleRead()

		case key.Matches(msg, mv.keyMap.Flag):
			return mv, mv.toggleFlag()

		case key.Matches(msg, mv.keyMap.SaveAttach):
			return mv, mv.saveAttachment()

		case key.Matches(msg, mv.keyMap.ViewThread):
			return mv, mv.viewThread()

		case key.Matches(msg, mv.keyMap.NextInThread):
			return mv, mv.nextInThread()

		case key.Matches(msg, mv.keyMap.PrevInThread):
			return mv, mv.prevInThread()

		case key.Matches(msg, mv.keyMap.Back):
			return mv, mv.backToList()

		// Toggle display options
		case msg.String() == "h":
			mv.showHeaders = !mv.showHeaders
			mv.renderContent()
			return mv, nil

		case msg.String() == "H":
			mv.showHTML = !mv.showHTML
			mv.renderContent()
			return mv, nil
		}

	case MessageOpenedMsg:
		mv.accountID = msg.AccountID
		mv.folderName = msg.FolderName
		mv.messageID = msg.MessageID
		mv.loading = true
		mv.error = ""
		return mv, mv.loadMessage()

	case MessageLoadedMsg:
		mv.loading = false
		mv.message = &msg.Message
		mv.error = ""
		mv.renderContent()
		return mv, nil

	case MessageLoadErrorMsg:
		mv.loading = false
		mv.error = msg.Error
		return mv, nil

	case ThreadLoadedMsg:
		if mv.message != nil {
			mv.message.Thread = &msg.Thread
			mv.threadMode = true
			mv.threadIndex = msg.Thread.CurrentIndex
			mv.renderContent()
		}
		return mv, nil
	}

	// Update viewport
	mv.viewport, cmd = mv.viewport.Update(msg)
	return mv, cmd
}

// View renders the message view
func (mv *MessageView) View() string {
	if mv.width == 0 || mv.height == 0 {
		return ""
	}

	var content strings.Builder

	// Loading state
	if mv.loading {
		loading := styles.LoadingStyle.Render("Loading message...")
		return mv.wrapInContainer(loading)
	}

	// Error state
	if mv.error != "" {
		errorText := styles.ErrorStyle.Render("Error: " + mv.error)
		return mv.wrapInContainer(errorText)
	}

	// No message loaded
	if mv.message == nil {
		noMessage := styles.SubtleStyle.Render("No message selected")
		return mv.wrapInContainer(noMessage)
	}

	// Message content (handled by viewport)
	content.WriteString(mv.viewport.View())

	// Action buttons
	if mv.focused {
		content.WriteString("\n")
		content.WriteString(mv.renderActions())
	}

	return mv.wrapInContainer(content.String())
}

// updateDimensions updates the component dimensions
func (mv *MessageView) updateDimensions() {
	if mv.width == 0 || mv.height == 0 {
		return
	}

	// Calculate available height for viewport
	availableHeight := mv.height - 2 // Border
	if mv.focused {
		availableHeight -= mv.actionsHeight
	}

	mv.viewport.Width = mv.width - 2  // Account for borders
	mv.viewport.Height = availableHeight

	// Update wrap width based on viewport width
	mv.wrapWidth = mv.viewport.Width - 4 // Account for padding
	if mv.wrapWidth < 20 {
		mv.wrapWidth = 20
	}
}

// renderContent renders the message content into the viewport
func (mv *MessageView) renderContent() {
	if mv.message == nil {
		return
	}

	var content strings.Builder

	// Message header
	header := mv.renderHeader()
	content.WriteString(header)
	content.WriteString("\n\n")

	// Message body
	body := mv.renderBody()
	content.WriteString(body)

	// Attachments
	if len(mv.message.Attachments) > 0 {
		content.WriteString("\n\n")
		attachments := mv.renderAttachments()
		content.WriteString(attachments)
	}

	// Thread information
	if mv.message.Thread != nil && mv.threadMode {
		content.WriteString("\n\n")
		threadInfo := mv.renderThreadInfo()
		content.WriteString(threadInfo)
	}

	// Additional headers (if enabled)
	if mv.showHeaders {
		content.WriteString("\n\n")
		headers := mv.renderDetailedHeaders()
		content.WriteString(headers)
	}

	mv.viewport.SetContent(content.String())
}

// renderHeader renders the message header information
func (mv *MessageView) renderHeader() string {
	var header strings.Builder

	// Subject
	subject := mv.message.Subject
	if subject == "" {
		subject = "(No Subject)"
	}
	header.WriteString(styles.EmailSubjectStyle.Render(subject))
	header.WriteString("\n")

	// From
	header.WriteString(styles.SubtleStyle.Render("From: "))
	fromDisplay := mv.message.FromDisplay
	if fromDisplay == "" {
		fromDisplay = mv.message.From.Display
	}
	header.WriteString(styles.EmailFromStyle.Render(fromDisplay))
	header.WriteString("\n")

	// To (if multiple recipients)
	if len(mv.message.To) > 0 {
		header.WriteString(styles.SubtleStyle.Render("To: "))
		toDisplay := make([]string, len(mv.message.To))
		for i, addr := range mv.message.To {
			toDisplay[i] = addr.Display
		}
		header.WriteString(styles.ListItemStyle.Render(strings.Join(toDisplay, ", ")))
		header.WriteString("\n")
	}

	// Date and status indicators
	dateStr := mv.formatDetailedDate(mv.message.Date)
	header.WriteString(styles.SubtleStyle.Render("Date: "))
	header.WriteString(styles.EmailDateStyle.Render(dateStr))
	
	// Add status indicators in same line
	statusIcons := mv.getMessageStatusIcons()
	if statusIcons != "" {
		padding := mv.width - len("Date: "+dateStr) - len(statusIcons) - 10
		if padding > 0 {
			header.WriteString(strings.Repeat(" ", padding))
		}
		header.WriteString(statusIcons)
	}

	return header.String()
}

// renderBody renders the message body content
func (mv *MessageView) renderBody() string {
	var body string

	// Use HTML body if available and HTML mode is enabled
	if mv.showHTML && mv.message.BodyHTML != "" {
		body = mv.convertHTMLToText(mv.message.BodyHTML)
	} else {
		body = mv.message.Body
	}

	// Wrap text to fit width
	wrappedBody := mv.wrapText(body, mv.wrapWidth)
	
	return styles.EmailBodyStyle.Render(wrappedBody)
}

// renderAttachments renders the attachments list
func (mv *MessageView) renderAttachments() string {
	if len(mv.message.Attachments) == 0 {
		return ""
	}

	var attachments strings.Builder
	attachments.WriteString(styles.SubtitleStyle.Render("Attachments:"))
	attachments.WriteString("\n")

	for i, attachment := range mv.message.Attachments {
		var line strings.Builder
		
		// Attachment icon
		line.WriteString(styles.AttachmentStyle.Render("📎 "))
		
		// Filename
		line.WriteString(styles.ListItemStyle.Render(attachment.Filename))
		
		// Size
		line.WriteString(" ")
		line.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("(%s)", attachment.SizeDisplay)))
		
		// Download status
		if attachment.Downloaded {
			line.WriteString(" ")
			line.WriteString(styles.SuccessStyle.Render("✓"))
		}

		attachments.WriteString(line.String())
		if i < len(mv.message.Attachments)-1 {
			attachments.WriteString("\n")
		}
	}

	return attachments.String()
}

// renderThreadInfo renders thread navigation information
func (mv *MessageView) renderThreadInfo() string {
	if mv.message.Thread == nil {
		return ""
	}

	thread := mv.message.Thread
	var threadInfo strings.Builder

	// Thread title
	threadInfo.WriteString(styles.SubtitleStyle.Render("Thread:"))
	threadInfo.WriteString(" ")
	threadInfo.WriteString(styles.ListItemStyle.Render(thread.Subject))
	threadInfo.WriteString("\n")

	// Thread navigation
	navInfo := fmt.Sprintf("[%d/%d]", thread.CurrentIndex+1, thread.TotalCount)
	threadInfo.WriteString(styles.SubtleStyle.Render(navInfo))
	
	if thread.TotalCount > 1 {
		threadInfo.WriteString(" ")
		if thread.CurrentIndex > 0 {
			threadInfo.WriteString(styles.ButtonStyle.Render("← Previous"))
		} else {
			threadInfo.WriteString(styles.SubtleStyle.Render("← Previous"))
		}
		
		threadInfo.WriteString(" ")
		if thread.CurrentIndex < thread.TotalCount-1 {
			threadInfo.WriteString(styles.ButtonStyle.Render("Next →"))
		} else {
			threadInfo.WriteString(styles.SubtleStyle.Render("Next →"))
		}
		
		threadInfo.WriteString(" ")
		threadInfo.WriteString(styles.ButtonStyle.Render("View Thread"))
	}

	return threadInfo.String()
}

// renderDetailedHeaders renders additional email headers
func (mv *MessageView) renderDetailedHeaders() string {
	if len(mv.message.Headers) == 0 {
		return ""
	}

	var headers strings.Builder
	headers.WriteString(styles.SubtitleStyle.Render("Headers:"))
	headers.WriteString("\n")

	// Show important headers first
	importantHeaders := []string{
		"Message-ID",
		"In-Reply-To", 
		"References",
		"X-Mailer",
		"User-Agent",
		"Content-Type",
	}

	for _, headerName := range importantHeaders {
		if value, exists := mv.message.Headers[headerName]; exists {
			headers.WriteString(styles.SubtleStyle.Render(headerName + ": "))
			headers.WriteString(styles.ListItemStyle.Render(value))
			headers.WriteString("\n")
		}
	}

	return headers.String()
}

// renderActions renders the action buttons
func (mv *MessageView) renderActions() string {
	actions := []string{
		"📧 Reply (r)",
		"🔄 Reply All (R)",
		"📤 Forward (F)",
		"🗑️ Delete (d)",
		"📁 Archive (a)",
	}
	
	if mv.message != nil {
		if mv.message.IsRead {
			actions = append(actions, "👁️ Mark Unread (u)")
		} else {
			actions = append(actions, "✉️ Mark Read (u)")
		}
		
		if mv.message.IsFlagged {
			actions = append(actions, "🚩 Unflag (s)")
		} else {
			actions = append(actions, "⭐ Flag (s)")
		}
		
		if len(mv.message.Attachments) > 0 {
			actions = append(actions, "💾 Save Attachment (S)")
		}
		
		if mv.message.ThreadCount > 1 {
			actions = append(actions, "🧵 View Thread (t)")
		}
	}

	actionText := strings.Join(actions, " • ")
	wrapped := mv.wrapText(actionText, mv.width-4)
	
	return styles.SubtleStyle.Render(wrapped)
}

// getMessageStatusIcons returns status icons for the message
func (mv *MessageView) getMessageStatusIcons() string {
	if mv.message == nil {
		return ""
	}

	var icons strings.Builder
	
	// Read status
	if mv.message.IsRead {
		icons.WriteString(styles.ReadStyle.Render("👁️ "))
	} else {
		icons.WriteString(styles.UnreadStyle.Render("✉️ "))
	}
	
	// Flag status
	if mv.message.IsFlagged {
		icons.WriteString(styles.WarningStyle.Render("🚩 "))
	}
	
	// Attachments
	if mv.message.HasAttachments {
		icons.WriteString(styles.AttachmentStyle.Render("📎 "))
	}
	
	// Encryption/Signing
	if mv.message.IsEncrypted {
		icons.WriteString(styles.EncryptedStyle.Render("🔒 "))
	}
	if mv.message.IsSigned {
		icons.WriteString(styles.SignedStyle.Render("✅ "))
	}
	
	return icons.String()
}

// formatDetailedDate formats a date for detailed display
func (mv *MessageView) formatDetailedDate(date time.Time) string {
	now := time.Now()
	
	// Today: show full time
	if date.Format("2006-01-02") == now.Format("2006-01-02") {
		return date.Format("Today at 15:04")
	}
	
	// Yesterday
	yesterday := now.AddDate(0, 0, -1)
	if date.Format("2006-01-02") == yesterday.Format("2006-01-02") {
		return date.Format("Yesterday at 15:04")
	}
	
	// This week: show day and time
	if date.After(now.AddDate(0, 0, -7)) {
		return date.Format("Monday at 15:04")
	}
	
	// This year: show month, day, and time
	if date.Year() == now.Year() {
		return date.Format("Jan 02 at 15:04")
	}
	
	// Full date and time
	return date.Format("Jan 02, 2006 at 15:04")
}

// convertHTMLToText converts HTML content to plain text
func (mv *MessageView) convertHTMLToText(htmlContent string) string {
	// Simple HTML to text conversion
	text := htmlContent
	
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text = re.ReplaceAllString(text, "")
	
	// Decode HTML entities
	text = html.UnescapeString(text)
	
	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	
	return text
}

// wrapText wraps text to the specified width
func (mv *MessageView) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	
	lines := strings.Split(text, "\n")
	var wrapped strings.Builder
	
	for i, line := range lines {
		if len(line) <= width {
			wrapped.WriteString(line)
		} else {
			// Simple word wrapping
			words := strings.Fields(line)
			currentLine := ""
			
			for _, word := range words {
				if len(currentLine) == 0 {
					currentLine = word
				} else if len(currentLine)+1+len(word) <= width {
					currentLine += " " + word
				} else {
					wrapped.WriteString(currentLine)
					wrapped.WriteString("\n")
					currentLine = word
				}
			}
			
			if currentLine != "" {
				wrapped.WriteString(currentLine)
			}
		}
		
		if i < len(lines)-1 {
			wrapped.WriteString("\n")
		}
	}
	
	return wrapped.String()
}

// wrapInContainer wraps content in the appropriate container style
func (mv *MessageView) wrapInContainer(content string) string {
	containerStyle := styles.MainPaneStyle.
		Width(mv.width).
		Height(mv.height)
	
	if mv.focused {
		containerStyle = containerStyle.
			BorderForeground(styles.PrimaryColor).
			Border(lipgloss.RoundedBorder())
	}
	
	return containerStyle.Render(content)
}

// Focus sets focus to the message view
func (mv *MessageView) Focus() {
	mv.focused = true
	mv.updateDimensions()
}

// Blur removes focus from the message view
func (mv *MessageView) Blur() {
	mv.focused = false
	mv.updateDimensions()
}

// Focused returns whether the message view is focused
func (mv *MessageView) Focused() bool {
	return mv.focused
}

// SetSize sets the dimensions of the message view
func (mv *MessageView) SetSize(width, height int) {
	mv.width = width
	mv.height = height
	mv.updateDimensions()
}

// LoadMessage loads a message for display
func (mv *MessageView) LoadMessage(accountID, folderName, messageID string) tea.Cmd {
	mv.accountID = accountID
	mv.folderName = folderName  
	mv.messageID = messageID
	mv.loading = true
	mv.error = ""
	return mv.loadMessage()
}

// Message operations
func (mv *MessageView) loadMessage() tea.Cmd {
	return func() tea.Msg {
		message, err := mv.service.GetMessage(mv.accountID, mv.messageID)
		if err != nil {
			return MessageLoadErrorMsg{Error: err.Error()}
		}
		
		// Convert to MessageDetails (would need service method to get full details)
		messageDetails := MessageDetails{
			MessageInfo: *message,
			Body:        "Message body would be loaded here...", // Placeholder
			Headers:     make(map[string]string),
			Attachments: make([]services.AttachmentInfo, 0),
		}
		
		return MessageLoadedMsg{Message: messageDetails}
	}
}

func (mv *MessageView) replyMessage(replyAll bool) tea.Cmd {
	if mv.message == nil {
		return nil
	}
	
	return func() tea.Msg {
		return MessageReplyRequestMsg{
			Message:  mv.message.MessageInfo,
			ReplyAll: replyAll,
		}
	}
}

func (mv *MessageView) forwardMessage() tea.Cmd {
	if mv.message == nil {
		return nil
	}
	
	return func() tea.Msg {
		return MessageForwardRequestMsg{
			Message: mv.message.MessageInfo,
		}
	}
}

func (mv *MessageView) deleteMessage() tea.Cmd {
	if mv.message == nil {
		return nil
	}
	
	return func() tea.Msg {
		err := mv.service.DeleteMessage(mv.accountID, []string{mv.messageID})
		if err != nil {
			return MessageOperationErrorMsg{Error: err.Error()}
		}
		
		return MessageDeletedMsg{MessageID: mv.messageID}
	}
}

func (mv *MessageView) archiveMessage() tea.Cmd {
	if mv.message == nil {
		return nil
	}
	
	return func() tea.Msg {
		// Move to Archive folder (implementation depends on service)
		err := mv.service.MoveMessage(mv.accountID, []string{mv.messageID}, "Archive")
		if err != nil {
			return MessageOperationErrorMsg{Error: err.Error()}
		}
		
		return MessageArchivedMsg{MessageID: mv.messageID}
	}
}

func (mv *MessageView) toggleRead() tea.Cmd {
	if mv.message == nil {
		return nil
	}
	
	return func() tea.Msg {
		var err error
		if mv.message.IsRead {
			err = mv.service.MarkUnread(mv.accountID, []string{mv.messageID})
		} else {
			err = mv.service.MarkRead(mv.accountID, []string{mv.messageID})
		}
		
		if err != nil {
			return MessageOperationErrorMsg{Error: err.Error()}
		}
		
		// Update local state
		mv.message.IsRead = !mv.message.IsRead
		mv.renderContent()
		
		return MessageReadToggleMsg{
			MessageID: mv.messageID,
			IsRead:    mv.message.IsRead,
		}
	}
}

func (mv *MessageView) toggleFlag() tea.Cmd {
	if mv.message == nil {
		return nil
	}
	
	return func() tea.Msg {
		var err error
		if mv.message.IsFlagged {
			err = mv.service.UnflagMessage(mv.accountID, []string{mv.messageID})
		} else {
			err = mv.service.FlagMessage(mv.accountID, []string{mv.messageID})
		}
		
		if err != nil {
			return MessageOperationErrorMsg{Error: err.Error()}
		}
		
		// Update local state
		mv.message.IsFlagged = !mv.message.IsFlagged
		mv.renderContent()
		
		return MessageFlagToggleMsg{
			MessageID: mv.messageID,
			IsFlagged: mv.message.IsFlagged,
		}
	}
}

func (mv *MessageView) saveAttachment() tea.Cmd {
	if mv.message == nil || len(mv.message.Attachments) == 0 {
		return nil
	}
	
	// For now, just save the first attachment
	// In a real implementation, this would open an attachment selector
	attachment := mv.message.Attachments[0]
	
	return func() tea.Msg {
		// Service method to save attachment would be called here
		return AttachmentSavedMsg{
			AttachmentID: attachment.ID,
			Filename:     attachment.Filename,
		}
	}
}

func (mv *MessageView) viewThread() tea.Cmd {
	if mv.message == nil || mv.message.ThreadID == "" {
		return nil
	}
	
	return func() tea.Msg {
		return ThreadViewRequestMsg{
			ThreadID:  mv.message.ThreadID,
			AccountID: mv.accountID,
		}
	}
}

func (mv *MessageView) nextInThread() tea.Cmd {
	if mv.message == nil || mv.message.Thread == nil {
		return nil
	}
	
	thread := mv.message.Thread
	if thread.CurrentIndex >= thread.TotalCount-1 {
		return nil // Already at last message
	}
	
	nextIndex := thread.CurrentIndex + 1
	if nextIndex < len(thread.Messages) {
		nextMessage := thread.Messages[nextIndex]
		return mv.LoadMessage(mv.accountID, mv.folderName, nextMessage.ID)
	}
	
	return nil
}

func (mv *MessageView) prevInThread() tea.Cmd {
	if mv.message == nil || mv.message.Thread == nil {
		return nil
	}
	
	thread := mv.message.Thread
	if thread.CurrentIndex <= 0 {
		return nil // Already at first message
	}
	
	prevIndex := thread.CurrentIndex - 1
	if prevIndex >= 0 && prevIndex < len(thread.Messages) {
		prevMessage := thread.Messages[prevIndex]
		return mv.LoadMessage(mv.accountID, mv.folderName, prevMessage.ID)
	}
	
	return nil
}

func (mv *MessageView) backToList() tea.Cmd {
	return func() tea.Msg {
		return BackToListRequestMsg{}
	}
}

// Note: Message types are now defined in messages.go for shared use across components