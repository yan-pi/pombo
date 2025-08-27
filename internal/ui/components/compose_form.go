package components

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// ComposeFormKeyMap defines key bindings for the compose form
type ComposeFormKeyMap struct {
	NextField     key.Binding
	PrevField     key.Binding
	Send          key.Binding
	SaveDraft     key.Binding
	AddAttachment key.Binding
	Cancel        key.Binding
	QuoteOriginal key.Binding
	Preview       key.Binding
}

// DefaultComposeFormKeyMap returns the default key bindings for compose form
func DefaultComposeFormKeyMap() ComposeFormKeyMap {
	return ComposeFormKeyMap{
		NextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous field"),
		),
		Send: key.NewBinding(
			key.WithKeys("ctrl+enter"),
			key.WithHelp("ctrl+enter", "send email"),
		),
		SaveDraft: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save draft"),
		),
		AddAttachment: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "add attachment"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		QuoteOriginal: key.NewBinding(
			key.WithKeys("ctrl+q"),
			key.WithHelp("ctrl+q", "quote original"),
		),
		Preview: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "preview"),
		),
	}
}

// ComposeMode defines the composition mode
type ComposeMode int

const (
	ComposeNew ComposeMode = iota
	ComposeReply
	ComposeReplyAll
	ComposeForward
	ComposeDraft
)

// ComposeForm represents the email composition form
type ComposeForm struct {
	service       services.EmailService
	accountID     string
	mode          ComposeMode
	replyTo       *services.MessageInfo
	draftID       string
	
	// Form fields
	to            textinput.Model
	cc            textinput.Model
	bcc           textinput.Model
	subject       textinput.Model
	body          textarea.Model
	
	// State
	focusedField  int
	fieldCount    int
	attachments   []services.AttachmentInfo
	isDraft       bool
	sending       bool
	saving        bool
	error         string
	
	// Dimensions
	width         int
	height        int
	keyMap        ComposeFormKeyMap
	
	// Display options
	showCC        bool
	showBCC       bool
	showPreview   bool
	autoSaveTimer time.Time
	autoSave      bool
}

// NewComposeForm creates a new compose form component
func NewComposeForm(service services.EmailService) *ComposeForm {
	to := textinput.New()
	to.Placeholder = "recipient@example.com"
	to.CharLimit = 1000
	to.Width = 50
	
	cc := textinput.New()
	cc.Placeholder = "cc@example.com"
	cc.CharLimit = 1000
	cc.Width = 50
	
	bcc := textinput.New()
	bcc.Placeholder = "bcc@example.com"
	bcc.CharLimit = 1000
	bcc.Width = 50
	
	subject := textinput.New()
	subject.Placeholder = "Enter subject..."
	subject.CharLimit = 500
	subject.Width = 50
	
	body := textarea.New()
	body.Placeholder = "Write your email here..."
	body.CharLimit = 50000
	body.ShowLineNumbers = false
	
	cf := &ComposeForm{
		service:      service,
		mode:         ComposeNew,
		to:           to,
		cc:           cc,
		bcc:          bcc,
		subject:      subject,
		body:         body,
		focusedField: 0,
		attachments:  make([]services.AttachmentInfo, 0),
		keyMap:       DefaultComposeFormKeyMap(),
		showCC:       false,
		showBCC:      false,
		autoSave:     true,
		autoSaveTimer: time.Now(),
	}
	
	// Calculate initial field count
	cf.updateFieldCount()
	
	return cf
}

// Init initializes the compose form component
func (cf *ComposeForm) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the compose form
func (cf *ComposeForm) Update(msg tea.Msg) (*ComposeForm, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cf.width = msg.Width
		cf.height = msg.Height
		cf.updateDimensions()
		return cf, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, cf.keyMap.NextField):
			cf.nextField()
			return cf, nil

		case key.Matches(msg, cf.keyMap.PrevField):
			cf.prevField()
			return cf, nil

		case key.Matches(msg, cf.keyMap.Send):
			if cf.validateForm() {
				return cf, cf.sendMessage()
			}
			return cf, nil

		case key.Matches(msg, cf.keyMap.SaveDraft):
			return cf, cf.saveDraft()

		case key.Matches(msg, cf.keyMap.AddAttachment):
			return cf, cf.addAttachment()

		case key.Matches(msg, cf.keyMap.Cancel):
			return cf, cf.cancel()

		case key.Matches(msg, cf.keyMap.QuoteOriginal):
			if cf.replyTo != nil {
				cf.insertQuotedText()
			}
			return cf, nil

		case key.Matches(msg, cf.keyMap.Preview):
			cf.showPreview = !cf.showPreview
			return cf, nil

		// Special key handling for showing CC/BCC fields
		case msg.String() == "ctrl+c":
			cf.showCC = !cf.showCC
			cf.updateFieldCount()
			return cf, nil

		case msg.String() == "ctrl+b":
			cf.showBCC = !cf.showBCC
			cf.updateFieldCount()
			return cf, nil
		}

	case ComposeNewMsg:
		cf.reset()
		cf.mode = ComposeNew
		cf.accountID = msg.AccountID
		cf.setFromAccount()
		return cf, nil

	case ComposeReplyMsg:
		cf.reset()
		cf.mode = ComposeReply
		cf.accountID = msg.AccountID
		cf.replyTo = &msg.OriginalMessage
		cf.setupReply(false)
		return cf, nil

	case ComposeReplyAllMsg:
		cf.reset()
		cf.mode = ComposeReplyAll
		cf.accountID = msg.AccountID
		cf.replyTo = &msg.OriginalMessage
		cf.setupReply(true)
		return cf, nil

	case ComposeForwardMsg:
		cf.reset()
		cf.mode = ComposeForward
		cf.accountID = msg.AccountID
		cf.replyTo = &msg.OriginalMessage
		cf.setupForward()
		return cf, nil

	case ComposeDraftMsg:
		cf.mode = ComposeDraft
		cf.accountID = msg.AccountID
		cf.draftID = msg.DraftID
		cf.loadDraft(msg.Draft)
		return cf, nil

	case MessageSentMsg:
		cf.sending = false
		cf.error = ""
		return cf, func() tea.Msg {
			return ComposeCompletedMsg{Success: true}
		}

	case MessageSendErrorMsg:
		cf.sending = false
		cf.error = msg.Error
		return cf, nil

	case DraftSavedMsg:
		cf.saving = false
		cf.isDraft = true
		cf.draftID = msg.DraftID
		return cf, nil

	case DraftSaveErrorMsg:
		cf.saving = false
		cf.error = msg.Error
		return cf, nil

	case AttachmentAddedMsg:
		cf.attachments = append(cf.attachments, msg.Attachment)
		return cf, nil
	}

	// Update the focused field
	switch cf.focusedField {
	case 0: // To field
		cf.to, cmds = cf.updateInput(cf.to, msg, cmds)
	case 1: // CC field (if visible)
		if cf.showCC {
			cf.cc, cmds = cf.updateInput(cf.cc, msg, cmds)
		}
	case 2: // BCC field (if visible)
		if cf.showBCC {
			cf.bcc, cmds = cf.updateInput(cf.bcc, msg, cmds)
		}
	case cf.getSubjectFieldIndex(): // Subject field
		cf.subject, cmds = cf.updateInput(cf.subject, msg, cmds)
	case cf.getBodyFieldIndex(): // Body field
		var cmd tea.Cmd
		cf.body, cmd = cf.body.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Auto-save functionality
	if cf.autoSave && time.Since(cf.autoSaveTimer) > 30*time.Second {
		if cf.hasContent() {
			cmds = append(cmds, cf.saveDraft())
			cf.autoSaveTimer = time.Now()
		}
	}

	return cf, tea.Batch(cmds...)
}

// View renders the compose form
func (cf *ComposeForm) View() string {
	if cf.width == 0 || cf.height == 0 {
		return ""
	}

	var content strings.Builder

	// Header
	header := cf.renderHeader()
	content.WriteString(header)
	content.WriteString("\n")

	// Error display
	if cf.error != "" {
		errorText := styles.ErrorStyle.Render("Error: " + cf.error)
		content.WriteString(errorText)
		content.WriteString("\n")
	}

	// Form fields
	form := cf.renderForm()
	content.WriteString(form)

	// Status bar
	status := cf.renderStatus()
	content.WriteString("\n")
	content.WriteString(status)

	// Apply container styling
	containerStyle := styles.MainPaneStyle.
		Width(cf.width).
		Height(cf.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.PrimaryColor)

	return containerStyle.Render(content.String())
}

// updateDimensions updates the component dimensions
func (cf *ComposeForm) updateDimensions() {
	if cf.width == 0 || cf.height == 0 {
		return
	}

	// Calculate available width for inputs
	inputWidth := cf.width - 20 // Account for labels and padding
	if inputWidth < 20 {
		inputWidth = 20
	}

	// Update input field widths
	cf.to.Width = inputWidth
	cf.cc.Width = inputWidth
	cf.bcc.Width = inputWidth
	cf.subject.Width = inputWidth

	// Calculate available dimensions for body
	bodyHeight := cf.height - 15 // Account for form fields, header, status
	if cf.showCC {
		bodyHeight--
	}
	if cf.showBCC {
		bodyHeight--
	}
	if len(cf.attachments) > 0 {
		bodyHeight -= 3 // Space for attachments
	}
	if bodyHeight < 5 {
		bodyHeight = 5
	}

	cf.body.SetWidth(inputWidth)
	cf.body.SetHeight(bodyHeight)
}

// renderHeader renders the compose form header
func (cf *ComposeForm) renderHeader() string {
	var title string
	switch cf.mode {
	case ComposeNew:
		title = "📝 Compose New Email"
	case ComposeReply:
		title = "📧 Reply to Email"
	case ComposeReplyAll:
		title = "🔄 Reply All"
	case ComposeForward:
		title = "📤 Forward Email"
	case ComposeDraft:
		title = "📝 Edit Draft"
	}

	headerText := styles.TitleStyle.Render(title)

	// Add sending/saving indicators
	if cf.sending {
		headerText += " " + styles.LoadingStyle.Render("(Sending...)")
	} else if cf.saving {
		headerText += " " + styles.LoadingStyle.Render("(Saving...)")
	} else if cf.isDraft {
		headerText += " " + styles.SubtleStyle.Render("(Draft)")
	}

	return headerText
}

// renderForm renders the form fields
func (cf *ComposeForm) renderForm() string {
	var form strings.Builder

	// From field (read-only, shows current account)
	fromText := "From: " + cf.getFromAddress()
	form.WriteString(styles.SubtitleStyle.Render(fromText))
	form.WriteString("\n")

	// To field
	toLabel := cf.renderFieldLabel("To:", 0, true)
	form.WriteString(toLabel)
	form.WriteString(cf.to.View())
	form.WriteString("\n")

	// CC field (if visible)
	if cf.showCC {
		ccLabel := cf.renderFieldLabel("CC:", cf.getCCFieldIndex(), false)
		form.WriteString(ccLabel)
		form.WriteString(cf.cc.View())
		form.WriteString("\n")
	}

	// BCC field (if visible)
	if cf.showBCC {
		bccLabel := cf.renderFieldLabel("BCC:", cf.getBCCFieldIndex(), false)
		form.WriteString(bccLabel)
		form.WriteString(cf.bcc.View())
		form.WriteString("\n")
	}

	// Subject field
	subjectLabel := cf.renderFieldLabel("Subject:", cf.getSubjectFieldIndex(), true)
	form.WriteString(subjectLabel)
	form.WriteString(cf.subject.View())
	form.WriteString("\n")

	// Attachments (if any)
	if len(cf.attachments) > 0 {
		attachments := cf.renderAttachments()
		form.WriteString(attachments)
		form.WriteString("\n")
	}

	// Body field
	bodyLabel := cf.renderFieldLabel("Message:", cf.getBodyFieldIndex(), true)
	form.WriteString(bodyLabel)
	form.WriteString(cf.body.View())

	// Original message quote (for replies/forwards)
	if cf.replyTo != nil && (cf.mode == ComposeReply || cf.mode == ComposeReplyAll || cf.mode == ComposeForward) {
		form.WriteString("\n")
		quoted := cf.renderQuotedMessage()
		form.WriteString(quoted)
	}

	return form.String()
}

// renderFieldLabel renders a field label with focus indication
func (cf *ComposeForm) renderFieldLabel(label string, fieldIndex int, required bool) string {
	labelStyle := styles.SubtitleStyle
	
	if cf.focusedField == fieldIndex {
		labelStyle = styles.HighlightStyle
	}
	
	labelText := label
	if required {
		labelText += "*"
	}
	
	// Fixed width for alignment
	return fmt.Sprintf("%-10s", labelStyle.Render(labelText))
}

// renderAttachments renders the attachments list
func (cf *ComposeForm) renderAttachments() string {
	var attachments strings.Builder
	attachments.WriteString(styles.SubtitleStyle.Render("Attachments:"))
	attachments.WriteString("\n")

	for i, attachment := range cf.attachments {
		line := fmt.Sprintf("  📎 %s (%s)",
			attachment.Filename,
			attachment.SizeDisplay)
		
		attachments.WriteString(styles.ListItemStyle.Render(line))
		if i < len(cf.attachments)-1 {
			attachments.WriteString("\n")
		}
	}

	return attachments.String()
}

// renderQuotedMessage renders the original message for replies/forwards
func (cf *ComposeForm) renderQuotedMessage() string {
	if cf.replyTo == nil {
		return ""
	}

	var quoted strings.Builder
	quoted.WriteString(styles.SubtleStyle.Render("--- Original Message ---"))
	quoted.WriteString("\n")

	// Original message header
	dateStr := cf.replyTo.Date.Format("Mon, Jan 02, 2006 at 15:04")
	headerText := fmt.Sprintf("On %s, %s wrote:",
		dateStr,
		cf.replyTo.FromDisplay)
	quoted.WriteString(styles.SubtleStyle.Render(headerText))
	quoted.WriteString("\n")

	// Quote the message preview (in a real implementation, this would be the full body)
	if cf.replyTo.Preview != "" {
		lines := strings.Split(cf.replyTo.Preview, "\n")
		for _, line := range lines {
			quoted.WriteString(styles.SubtleStyle.Render("> " + line))
			quoted.WriteString("\n")
		}
	}

	return quoted.String()
}

// renderStatus renders the status bar with key bindings
func (cf *ComposeForm) renderStatus() string {
	var status strings.Builder

	// Key bindings
	bindings := []string{
		"Tab: Next Field",
		"Ctrl+Enter: Send",
		"Ctrl+S: Save Draft",
		"Ctrl+A: Attach File",
		"Esc: Cancel",
	}

	if !cf.showCC {
		bindings = append(bindings, "Ctrl+C: Show CC")
	}
	if !cf.showBCC {
		bindings = append(bindings, "Ctrl+B: Show BCC")
	}

	bindingText := strings.Join(bindings, " • ")
	status.WriteString(styles.SubtleStyle.Render(bindingText))

	return status.String()
}

// Field navigation and management
func (cf *ComposeForm) nextField() {
	cf.blurCurrentField()
	cf.focusedField = (cf.focusedField + 1) % cf.fieldCount
	cf.focusCurrentField()
}

func (cf *ComposeForm) prevField() {
	cf.blurCurrentField()
	if cf.focusedField == 0 {
		cf.focusedField = cf.fieldCount - 1
	} else {
		cf.focusedField--
	}
	cf.focusCurrentField()
}

func (cf *ComposeForm) focusCurrentField() {
	switch cf.focusedField {
	case 0:
		cf.to.Focus()
	case 1:
		if cf.showCC {
			cf.cc.Focus()
		}
	case 2:
		if cf.showBCC {
			cf.bcc.Focus()
		}
	case cf.getSubjectFieldIndex():
		cf.subject.Focus()
	case cf.getBodyFieldIndex():
		cf.body.Focus()
	}
}

func (cf *ComposeForm) blurCurrentField() {
	cf.to.Blur()
	cf.cc.Blur()
	cf.bcc.Blur()
	cf.subject.Blur()
	cf.body.Blur()
}

func (cf *ComposeForm) updateFieldCount() {
	count := 2 // To and Subject are always visible
	if cf.showCC {
		count++
	}
	if cf.showBCC {
		count++
	}
	count++ // Body field
	cf.fieldCount = count
}

// Field index helpers
func (cf *ComposeForm) getCCFieldIndex() int {
	return 1
}

func (cf *ComposeForm) getBCCFieldIndex() int {
	if cf.showCC {
		return 2
	}
	return 1
}

func (cf *ComposeForm) getSubjectFieldIndex() int {
	index := 1
	if cf.showCC {
		index++
	}
	if cf.showBCC {
		index++
	}
	return index
}

func (cf *ComposeForm) getBodyFieldIndex() int {
	return cf.getSubjectFieldIndex() + 1
}

// Form setup methods
func (cf *ComposeForm) reset() {
	cf.to.SetValue("")
	cf.cc.SetValue("")
	cf.bcc.SetValue("")
	cf.subject.SetValue("")
	cf.body.SetValue("")
	cf.attachments = make([]services.AttachmentInfo, 0)
	cf.error = ""
	cf.sending = false
	cf.saving = false
	cf.isDraft = false
	cf.draftID = ""
	cf.replyTo = nil
	cf.focusedField = 0
	cf.showCC = false
	cf.showBCC = false
	cf.updateFieldCount()
	cf.focusCurrentField()
}

func (cf *ComposeForm) setFromAccount() {
	// This would get the from address from the current account
	// For now, just placeholder
}

func (cf *ComposeForm) setupReply(replyAll bool) {
	if cf.replyTo == nil {
		return
	}

	// Set recipient
	cf.to.SetValue(cf.replyTo.From.Address)

	// For reply all, add other recipients to CC
	if replyAll {
		var ccAddresses []string
		for _, addr := range cf.replyTo.To {
			ccAddresses = append(ccAddresses, addr.Address)
		}
		if len(ccAddresses) > 0 {
			cf.cc.SetValue(strings.Join(ccAddresses, ", "))
			cf.showCC = true
		}
	}

	// Set subject
	subject := cf.replyTo.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}
	cf.subject.SetValue(subject)

	cf.updateFieldCount()
	cf.focusCurrentField()
}

func (cf *ComposeForm) setupForward() {
	if cf.replyTo == nil {
		return
	}

	// Set subject
	subject := cf.replyTo.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "fwd:") {
		subject = "Fwd: " + subject
	}
	cf.subject.SetValue(subject)

	cf.focusCurrentField()
}

func (cf *ComposeForm) loadDraft(draft services.OutgoingMessage) {
	// Load draft data into form
	if len(draft.To) > 0 {
		toAddresses := make([]string, len(draft.To))
		for i, addr := range draft.To {
			toAddresses[i] = addr.Address
		}
		cf.to.SetValue(strings.Join(toAddresses, ", "))
	}

	if len(draft.CC) > 0 {
		ccAddresses := make([]string, len(draft.CC))
		for i, addr := range draft.CC {
			ccAddresses[i] = addr.Address
		}
		cf.cc.SetValue(strings.Join(ccAddresses, ", "))
		cf.showCC = true
	}

	if len(draft.BCC) > 0 {
		bccAddresses := make([]string, len(draft.BCC))
		for i, addr := range draft.BCC {
			bccAddresses[i] = addr.Address
		}
		cf.bcc.SetValue(strings.Join(bccAddresses, ", "))
		cf.showBCC = true
	}

	cf.subject.SetValue(draft.Subject)
	cf.body.SetValue(draft.Body)
	cf.attachments = draft.Attachments

	cf.updateFieldCount()
	cf.focusCurrentField()
}

func (cf *ComposeForm) insertQuotedText() {
	if cf.replyTo == nil {
		return
	}

	// Insert quoted text at cursor position in body
	currentBody := cf.body.Value()
	quotedText := fmt.Sprintf("\n\nOn %s, %s wrote:\n> %s",
		cf.replyTo.Date.Format("Mon, Jan 02, 2006"),
		cf.replyTo.FromDisplay,
		strings.ReplaceAll(cf.replyTo.Preview, "\n", "\n> "))

	cf.body.SetValue(currentBody + quotedText)
}

// Utility methods
func (cf *ComposeForm) getFromAddress() string {
	// This would get the from address from the service
	// For now, return placeholder
	return "user@example.com"
}

func (cf *ComposeForm) validateForm() bool {
	cf.error = ""

	// Check required fields
	if strings.TrimSpace(cf.to.Value()) == "" {
		cf.error = "To field is required"
		return false
	}

	if strings.TrimSpace(cf.subject.Value()) == "" {
		cf.error = "Subject is required"
		return false
	}

	// Validate email addresses
	if !cf.validateEmailAddresses(cf.to.Value()) {
		cf.error = "Invalid email address in To field"
		return false
	}

	if cf.showCC && cf.cc.Value() != "" {
		if !cf.validateEmailAddresses(cf.cc.Value()) {
			cf.error = "Invalid email address in CC field"
			return false
		}
	}

	if cf.showBCC && cf.bcc.Value() != "" {
		if !cf.validateEmailAddresses(cf.bcc.Value()) {
			cf.error = "Invalid email address in BCC field"
			return false
		}
	}

	return true
}

func (cf *ComposeForm) validateEmailAddresses(addresses string) bool {
	if addresses == "" {
		return true
	}

	// Simple email regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	// Split by comma and validate each address
	addrs := strings.Split(addresses, ",")
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr != "" && !emailRegex.MatchString(addr) {
			return false
		}
	}

	return true
}

func (cf *ComposeForm) hasContent() bool {
	return strings.TrimSpace(cf.to.Value()) != "" ||
		strings.TrimSpace(cf.cc.Value()) != "" ||
		strings.TrimSpace(cf.bcc.Value()) != "" ||
		strings.TrimSpace(cf.subject.Value()) != "" ||
		strings.TrimSpace(cf.body.Value()) != ""
}

func (cf *ComposeForm) parseEmailAddresses(addresses string) []services.AddressInfo {
	if addresses == "" {
		return []services.AddressInfo{}
	}

	var result []services.AddressInfo
	addrs := strings.Split(addresses, ",")
	
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr != "" {
			result = append(result, services.AddressInfo{
				Address: addr,
				Display: addr,
			})
		}
	}

	return result
}

// Helper method to update text inputs consistently
func (cf *ComposeForm) updateInput(input textinput.Model, msg tea.Msg, cmds []tea.Cmd) (textinput.Model, []tea.Cmd) {
	var cmd tea.Cmd
	input, cmd = input.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return input, cmds
}

// Focus sets focus to the compose form
func (cf *ComposeForm) Focus() {
	cf.focusCurrentField()
}

// Blur removes focus from the compose form
func (cf *ComposeForm) Blur() {
	cf.blurCurrentField()
}

// SetSize sets the dimensions of the compose form
func (cf *ComposeForm) SetSize(width, height int) {
	cf.width = width
	cf.height = height
	cf.updateDimensions()
}

// Message operations
func (cf *ComposeForm) sendMessage() tea.Cmd {
	if !cf.validateForm() {
		return nil
	}

	cf.sending = true
	cf.error = ""

	return func() tea.Msg {
		message := services.OutgoingMessage{
			To:          cf.parseEmailAddresses(cf.to.Value()),
			CC:          cf.parseEmailAddresses(cf.cc.Value()),
			BCC:         cf.parseEmailAddresses(cf.bcc.Value()),
			Subject:     cf.subject.Value(),
			Body:        cf.body.Value(),
			Attachments: cf.attachments,
		}

		// Add reply headers if this is a reply
		if cf.replyTo != nil && (cf.mode == ComposeReply || cf.mode == ComposeReplyAll) {
			message.InReplyTo = cf.replyTo.ID
			message.References = []string{cf.replyTo.ID}
		}

		err := cf.service.SendMessage(cf.accountID, &message)
		if err != nil {
			return MessageSendErrorMsg{Error: err.Error()}
		}

		return MessageSentMsg{}
	}
}

func (cf *ComposeForm) saveDraft() tea.Cmd {
	if !cf.hasContent() {
		return nil
	}

	cf.saving = true
	cf.error = ""

	return func() tea.Msg {
		// In a real implementation, this would call a service method to save draft
		// message := services.OutgoingMessage{
		//	To:          cf.parseEmailAddresses(cf.to.Value()),
		//	CC:          cf.parseEmailAddresses(cf.cc.Value()),
		//	BCC:         cf.parseEmailAddresses(cf.bcc.Value()),
		//	Subject:     cf.subject.Value(),
		//	Body:        cf.body.Value(),
		//	Attachments: cf.attachments,
		//	SaveDraft:   true,
		// }
		
		// For now, just simulate success
		draftID := fmt.Sprintf("draft-%d", time.Now().Unix())

		return DraftSavedMsg{DraftID: draftID}
	}
}

func (cf *ComposeForm) addAttachment() tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, this would open a file picker
		// For now, just simulate adding an attachment
		attachment := services.AttachmentInfo{
			ID:          fmt.Sprintf("attach-%d", time.Now().Unix()),
			Filename:    "example.pdf",
			ContentType: "application/pdf",
			Size:        1024000,
			SizeDisplay: "1.0 MB",
		}

		return AttachmentAddedMsg{Attachment: attachment}
	}
}

func (cf *ComposeForm) cancel() tea.Cmd {
	// Save draft before canceling if there's content
	if cf.hasContent() && !cf.isDraft {
		return tea.Batch(
			cf.saveDraft(),
			func() tea.Msg {
				return ComposeCancelledMsg{}
			},
		)
	}

	return func() tea.Msg {
		return ComposeCancelledMsg{}
	}
}

// Note: Message types are now defined in messages.go for shared use across components