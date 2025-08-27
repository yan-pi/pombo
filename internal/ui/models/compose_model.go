package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// ComposeModel represents the compose view for writing emails
type ComposeModel struct {
	// Core dependencies
	service services.EmailService
	logger  *log.Logger

	// UI state
	width        int
	height       int
	ready        bool
	composing    bool
	sending      bool

	// Service state
	serviceState *services.ServiceState

	// Form fields
	toInput      textinput.Model
	ccInput      textinput.Model
	bccInput     textinput.Model
	subjectInput textinput.Model
	bodyInput    textarea.Model

	// Form state
	focusedField int
	fieldOrder   []int

	// Compose context
	replyTo      *services.MessageInfo
	composeType  string // "new", "reply", "reply-all", "forward"
}

// Field constants
const (
	FieldTo = iota
	FieldCC
	FieldBCC
	FieldSubject
	FieldBody
)

// NewComposeModel creates a new compose model
func NewComposeModel(service services.EmailService, logger *log.Logger) *ComposeModel {
	// Initialize input fields
	toInput := textinput.New()
	toInput.Placeholder = "recipient@example.com"
	toInput.Focus()

	ccInput := textinput.New()
	ccInput.Placeholder = "cc@example.com"

	bccInput := textinput.New()
	bccInput.Placeholder = "bcc@example.com"

	subjectInput := textinput.New()
	subjectInput.Placeholder = "Email subject"

	bodyInput := textarea.New()
	bodyInput.Placeholder = "Type your message here..."
	bodyInput.SetWidth(80)
	bodyInput.SetHeight(10)

	return &ComposeModel{
		service:      service,
		logger:       logger,
		ready:        false,
		composing:    false,
		sending:      false,
		toInput:      toInput,
		ccInput:      ccInput,
		bccInput:     bccInput,
		subjectInput: subjectInput,
		bodyInput:    bodyInput,
		focusedField: FieldTo,
		fieldOrder:   []int{FieldTo, FieldCC, FieldBCC, FieldSubject, FieldBody},
	}
}

// SetSize updates the model dimensions
func (m *ComposeModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.ready = true

	// Update input field widths
	fieldWidth := width - 20 // Account for labels and padding
	m.toInput.Width = fieldWidth
	m.ccInput.Width = fieldWidth
	m.bccInput.Width = fieldWidth
	m.subjectInput.Width = fieldWidth

	// Update body textarea size
	bodyHeight := height - 15 // Account for header fields and margins
	if bodyHeight < 5 {
		bodyHeight = 5
	}
	m.bodyInput.SetWidth(fieldWidth)
	m.bodyInput.SetHeight(bodyHeight)
}

// Update handles messages and updates the compose model
func (m *ComposeModel) Update(msg tea.Msg) (*ComposeModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.sending {
			return m, nil // Ignore input while sending
		}

		switch msg.String() {
		case "ctrl+c":
			return m, m.cancelCompose()

		case "ctrl+s", "ctrl+enter":
			return m, m.sendMessage()

		case "tab":
			m.nextField()
			return m, nil

		case "shift+tab":
			m.prevField()
			return m, nil

		case "esc":
			if m.focusedField == FieldBody && m.bodyInput.Focused() {
				m.bodyInput.Blur()
			} else {
				return m, m.cancelCompose()
			}
			return m, nil
		}

		// Update focused field
		switch m.focusedField {
		case FieldTo:
			m.toInput, cmd = m.toInput.Update(msg)
		case FieldCC:
			m.ccInput, cmd = m.ccInput.Update(msg)
		case FieldBCC:
			m.bccInput, cmd = m.bccInput.Update(msg)
		case FieldSubject:
			m.subjectInput, cmd = m.subjectInput.Update(msg)
		case FieldBody:
			m.bodyInput, cmd = m.bodyInput.Update(msg)
		}

		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ViewSwitchMsg:
		if msg.View == ViewCompose {
			if data, ok := msg.Data.(map[string]interface{}); ok {
				return m, m.initializeCompose(data)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the compose view
func (m *ComposeModel) View() string {
	if !m.ready {
		return "Loading compose..."
	}

	if m.sending {
		return m.renderSending()
	}

	// Render the compose form
	return m.renderComposeForm()
}

// UpdateState updates the model with new service state
func (m *ComposeModel) UpdateState(state *services.ServiceState) {
	m.serviceState = state
}

// StartComposing starts a new compose session
func (m *ComposeModel) StartComposing(composeType string) tea.Cmd {
	m.composing = true
	m.composeType = composeType
	m.focusedField = FieldTo
	m.updateFieldFocus()
	return nil
}

// Helper methods

func (m *ComposeModel) renderSending() string {
	content := styles.LoadingStyle.Render("Sending message...")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *ComposeModel) renderComposeForm() string {
	// Header
	header := m.renderComposeHeader()

	// Form fields
	form := m.renderFormFields()

	// Footer with help
	footer := m.renderComposeFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		form,
		footer,
	)
}

func (m *ComposeModel) renderComposeHeader() string {
	title := "Compose New Message"
	if m.composeType == "reply" {
		title = "Reply to Message"
	} else if m.composeType == "forward" {
		title = "Forward Message"
	}

	// Account info
	accountInfo := ""
	if m.serviceState != nil && m.serviceState.CurrentAccount != nil {
		accountInfo = m.serviceState.CurrentAccount.Email
	}

	headerContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		styles.TitleStyle.Render(title),
		strings.Repeat(" ", max(0, m.width-lipgloss.Width(title)-lipgloss.Width(accountInfo))),
		styles.SubtleStyle.Render(accountInfo),
	)

	return styles.HeaderStyle.Width(m.width).Render(headerContent)
}

func (m *ComposeModel) renderFormFields() string {
	var fields []string

	// To field
	toStyle := styles.InputStyle
	if m.focusedField == FieldTo {
		toStyle = styles.ActiveInputStyle
	}
	toField := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styles.EmailHeaderStyle.Width(8).Render("To:"),
		toStyle.Render(m.toInput.View()),
	)
	fields = append(fields, toField)

	// CC field (only show if it has content or is focused)
	if m.ccInput.Value() != "" || m.focusedField == FieldCC {
		ccStyle := styles.InputStyle
		if m.focusedField == FieldCC {
			ccStyle = styles.ActiveInputStyle
		}
		ccField := lipgloss.JoinHorizontal(
			lipgloss.Top,
			styles.SubtleStyle.Width(8).Render("CC:"),
			ccStyle.Render(m.ccInput.View()),
		)
		fields = append(fields, ccField)
	}

	// BCC field (only show if it has content or is focused)
	if m.bccInput.Value() != "" || m.focusedField == FieldBCC {
		bccStyle := styles.InputStyle
		if m.focusedField == FieldBCC {
			bccStyle = styles.ActiveInputStyle
		}
		bccField := lipgloss.JoinHorizontal(
			lipgloss.Top,
			styles.SubtleStyle.Width(8).Render("BCC:"),
			bccStyle.Render(m.bccInput.View()),
		)
		fields = append(fields, bccField)
	}

	// Subject field
	subjectStyle := styles.InputStyle
	if m.focusedField == FieldSubject {
		subjectStyle = styles.ActiveInputStyle
	}
	subjectField := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styles.EmailHeaderStyle.Width(8).Render("Subject:"),
		subjectStyle.Render(m.subjectInput.View()),
	)
	fields = append(fields, subjectField)

	// Separator
	fields = append(fields, strings.Repeat("─", m.width-2))

	// Body field
	bodyStyle := styles.BorderStyle
	if m.focusedField == FieldBody {
		bodyStyle = bodyStyle.BorderForeground(styles.PrimaryColor)
	}
	bodyField := bodyStyle.Render(m.bodyInput.View())
	fields = append(fields, bodyField)

	return lipgloss.JoinVertical(lipgloss.Left, fields...)
}

func (m *ComposeModel) renderComposeFooter() string {
	help := "Tab: next field • Ctrl+S: send • Ctrl+C: cancel • Esc: back"
	
	// Show character/word count for body
	stats := ""
	if m.focusedField == FieldBody {
		bodyText := m.bodyInput.Value()
		charCount := len(bodyText)
		wordCount := len(strings.Fields(bodyText))
		stats = styles.SubtleStyle.Render(
			fmt.Sprintf("%d chars, %d words", charCount, wordCount))
	}

	footerContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		stats,
		strings.Repeat(" ", max(0, m.width-lipgloss.Width(stats)-lipgloss.Width(help))),
		styles.SubtleStyle.Render(help),
	)

	return styles.FooterStyle.Width(m.width).Render(footerContent)
}

func (m *ComposeModel) nextField() {
	m.focusedField = (m.focusedField + 1) % len(m.fieldOrder)
	m.updateFieldFocus()
}

func (m *ComposeModel) prevField() {
	m.focusedField = (m.focusedField - 1 + len(m.fieldOrder)) % len(m.fieldOrder)
	m.updateFieldFocus()
}

func (m *ComposeModel) updateFieldFocus() {
	// Blur all fields
	m.toInput.Blur()
	m.ccInput.Blur()
	m.bccInput.Blur()
	m.subjectInput.Blur()
	m.bodyInput.Blur()

	// Focus current field
	switch m.focusedField {
	case FieldTo:
		m.toInput.Focus()
	case FieldCC:
		m.ccInput.Focus()
	case FieldBCC:
		m.bccInput.Focus()
	case FieldSubject:
		m.subjectInput.Focus()
	case FieldBody:
		m.bodyInput.Focus()
	}
}

func (m *ComposeModel) initializeCompose(data map[string]interface{}) tea.Cmd {
	composeType, _ := data["type"].(string)
	m.composeType = composeType

	switch composeType {
	case "reply":
		if replyTo, ok := data["replyTo"].(*services.MessageInfo); ok {
			m.replyTo = replyTo
			m.subjectInput.SetValue("Re: " + replyTo.Subject)
			m.toInput.SetValue(replyTo.FromDisplay)
		}
	case "forward":
		if message, ok := data["message"].(*services.MessageInfo); ok {
			m.subjectInput.SetValue("Fwd: " + message.Subject)
			// Would include original message in body
		}
	}

	m.composing = true
	return nil
}

func (m *ComposeModel) sendMessage() tea.Cmd {
	return func() tea.Msg {
		if m.serviceState == nil || m.serviceState.CurrentAccount == nil {
			return nil
		}

		// Validate required fields
		if strings.TrimSpace(m.toInput.Value()) == "" {
			m.logger.Warn("Cannot send message without recipient")
			return nil
		}

		m.sending = true

		// Parse recipients
		to := m.parseAddresses(m.toInput.Value())
		cc := m.parseAddresses(m.ccInput.Value())
		bcc := m.parseAddresses(m.bccInput.Value())

		// Create outgoing message
		msg := &services.OutgoingMessage{
			From: services.AddressInfo{
				Address: m.serviceState.CurrentAccount.Email,
				Display: m.serviceState.CurrentAccount.Email,
			},
			To:      to,
			CC:      cc,
			BCC:     bcc,
			Subject: m.subjectInput.Value(),
			Body:    m.bodyInput.Value(),
		}

		// Add reply context if applicable
		if m.replyTo != nil {
			msg.InReplyTo = m.replyTo.ID
		}

		// Send the message
		if err := m.service.SendMessage(m.serviceState.CurrentAccount.ID, msg); err != nil {
			m.logger.Error("Failed to send message", "error", err)
			m.sending = false
			return nil
		}

		m.logger.Info("Message sent successfully")
		m.sending = false
		m.composing = false

		// Switch back to mailbox
		return ViewSwitchMsg{View: ViewMailbox}
	}
}

func (m *ComposeModel) cancelCompose() tea.Cmd {
	m.composing = false
	m.clearForm()
	return func() tea.Msg {
		return ViewSwitchMsg{View: ViewMailbox}
	}
}

func (m *ComposeModel) clearForm() {
	m.toInput.SetValue("")
	m.ccInput.SetValue("")
	m.bccInput.SetValue("")
	m.subjectInput.SetValue("")
	m.bodyInput.SetValue("")
	m.replyTo = nil
	m.composeType = "new"
}

func (m *ComposeModel) parseAddresses(input string) []services.AddressInfo {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	addresses := strings.Split(input, ",")
	result := make([]services.AddressInfo, 0, len(addresses))

	for _, addr := range addresses {
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