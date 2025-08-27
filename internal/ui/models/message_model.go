package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// MessageModel represents the message view for reading individual emails
type MessageModel struct {
	// Core dependencies
	service services.EmailService
	logger  *log.Logger

	// UI state
	width        int
	height       int
	ready        bool
	loading      bool

	// Service state
	serviceState *services.ServiceState

	// Message state
	currentMessage *services.MessageInfo
	fullMessage    *services.MessageInfo // Full message with body content
	scrollOffset   int
	maxScroll      int

	// View mode
	showHeaders    bool
	showHTML       bool
}

// NewMessageModel creates a new message model
func NewMessageModel(service services.EmailService, logger *log.Logger) *MessageModel {
	return &MessageModel{
		service:     service,
		logger:      logger,
		ready:       false,
		loading:     false,
		showHeaders: false,
		showHTML:    false,
	}
}

// SetSize updates the model dimensions
func (m *MessageModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.ready = true

	// Recalculate scroll limits
	if m.fullMessage != nil {
		m.calculateScrollLimits()
	}
}

// Update handles messages and updates the message model
func (m *MessageModel) Update(msg tea.Msg) (*MessageModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.scrollOffset < m.maxScroll {
				m.scrollOffset++
			}
		case "k", "up":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "g":
			m.scrollOffset = 0
		case "G":
			m.scrollOffset = m.maxScroll
		case "h":
			m.showHeaders = !m.showHeaders
			m.calculateScrollLimits()
		case "ctrl+h":
			m.showHTML = !m.showHTML
			m.calculateScrollLimits()
		case "r":
			return m, m.startReply()
		case "f":
			return m, m.startForward()
		}

	case ViewSwitchMsg:
		if msg.View == ViewMessage {
			if msgInfo, ok := msg.Data.(services.MessageInfo); ok {
				m.currentMessage = &msgInfo
				return m, m.loadFullMessage()
			}
		}
	}

	return m, nil
}

// View renders the message view
func (m *MessageModel) View() string {
	if !m.ready {
		return "Loading message..."
	}

	if m.currentMessage == nil {
		return m.renderNoMessage()
	}

	if m.loading {
		return m.renderLoading()
	}

	if m.fullMessage == nil {
		return m.renderMessageError()
	}

	// Render the message
	content := m.renderMessage()
	
	// Add header
	header := m.renderMessageHeader()
	
	// Add footer with help
	footer := m.renderMessageFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// UpdateState updates the model with new service state
func (m *MessageModel) UpdateState(state *services.ServiceState) {
	m.serviceState = state
}

// SetMessage sets the current message to display
func (m *MessageModel) SetMessage(msg services.MessageInfo) tea.Cmd {
	m.currentMessage = &msg
	m.scrollOffset = 0
	return m.loadFullMessage()
}

// Helper methods

func (m *MessageModel) renderNoMessage() string {
	content := styles.ContentStyle.Render("No message selected")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *MessageModel) renderLoading() string {
	content := styles.LoadingStyle.Render("Loading message...")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *MessageModel) renderMessageError() string {
	content := styles.ErrorStyle.Render("Failed to load message")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *MessageModel) renderMessageHeader() string {
	if m.fullMessage == nil {
		return ""
	}

	msg := m.fullMessage

	// Status indicators
	var indicators []string
	if !msg.IsRead {
		indicators = append(indicators, styles.WarningStyle.Render("UNREAD"))
	}
	if msg.IsFlagged {
		indicators = append(indicators, styles.WarningStyle.Render("FLAGGED"))
	}
	if msg.HasAttachments {
		indicators = append(indicators, "📎")
	}
	if msg.IsEncrypted {
		indicators = append(indicators, "🔒")
	}
	if msg.IsSigned {
		indicators = append(indicators, "✓")
	}

	statusLine := ""
	if len(indicators) > 0 {
		statusLine = strings.Join(indicators, " ")
	}

	// Header content
	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.EmailSubjectStyle.Render(msg.Subject),
		styles.EmailFromStyle.Render(fmt.Sprintf("From: %s", msg.FromDisplay)),
		styles.EmailDateStyle.Render(fmt.Sprintf("Date: %s", msg.Date.Format("Mon, 02 Jan 2006 15:04:05 -0700"))),
		styles.SubtleStyle.Render(statusLine),
	)

	// Optional detailed headers
	if m.showHeaders {
		var headers strings.Builder
		headers.WriteString(fmt.Sprintf("To: %s\n", m.formatAddressList(msg.To)))
		if len(msg.To) > 0 {
			// This would show CC, BCC, Message-ID, etc.
			headers.WriteString(fmt.Sprintf("Message-ID: %s\n", msg.ID))
		}
		
		headerContent = lipgloss.JoinVertical(
			lipgloss.Left,
			headerContent,
			"",
			styles.SubtleStyle.Render(headers.String()),
		)
	}

	return styles.EmailHeaderStyle.Render(headerContent)
}

func (m *MessageModel) renderMessage() string {
	if m.fullMessage == nil {
		return ""
	}

	// For now, just show the preview since we haven't implemented full body loading
	content := m.fullMessage.Preview
	if content == "" {
		content = "Message body not available"
	}

	// Split into lines for scrolling
	lines := strings.Split(content, "\n")
	
	// Calculate visible lines
	availableHeight := m.height - 6 // Account for header and footer
	visibleLines := lines[m.scrollOffset:]
	if len(visibleLines) > availableHeight {
		visibleLines = visibleLines[:availableHeight]
	}

	messageContent := strings.Join(visibleLines, "\n")
	
	return styles.EmailBodyStyle.Render(messageContent)
}

func (m *MessageModel) renderMessageFooter() string {
	if m.fullMessage == nil {
		return ""
	}

	// Scroll indicator
	scrollInfo := ""
	if m.maxScroll > 0 {
		scrollInfo = fmt.Sprintf("Line %d-%d of %d", 
			m.scrollOffset+1, 
			min(m.scrollOffset+m.height-6, m.maxScroll+m.height-6), 
			m.maxScroll+m.height-6)
	}

	// Help text
	help := "r: reply • f: forward • h: toggle headers • j/k: scroll • esc: back"

	footerContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		scrollInfo,
		strings.Repeat(" ", max(0, m.width-lipgloss.Width(scrollInfo)-lipgloss.Width(help))),
		styles.SubtleStyle.Render(help),
	)

	return styles.FooterStyle.Width(m.width).Render(footerContent)
}

func (m *MessageModel) calculateScrollLimits() {
	if m.fullMessage == nil {
		m.maxScroll = 0
		return
	}

	// Calculate total content lines
	content := m.fullMessage.Preview
	if m.showHeaders {
		// Add extra lines for headers
		content = strings.Repeat("\n", 10) + content
	}
	
	lines := strings.Split(content, "\n")
	availableHeight := m.height - 6
	
	if len(lines) > availableHeight {
		m.maxScroll = len(lines) - availableHeight
	} else {
		m.maxScroll = 0
	}
}

func (m *MessageModel) formatAddressList(addresses []services.AddressInfo) string {
	if len(addresses) == 0 {
		return ""
	}
	
	var addrs []string
	for _, addr := range addresses {
		addrs = append(addrs, addr.Display)
	}
	
	return strings.Join(addrs, ", ")
}

func (m *MessageModel) loadFullMessage() tea.Cmd {
	return func() tea.Msg {
		if m.currentMessage == nil || m.serviceState == nil || m.serviceState.CurrentAccount == nil {
			return nil
		}

		m.loading = true
		
		// Load full message content
		fullMsg, err := m.service.GetMessage(m.serviceState.CurrentAccount.ID, m.currentMessage.ID)
		if err != nil {
			m.logger.Error("Failed to load full message", "id", m.currentMessage.ID, "error", err)
			return nil
		}

		m.fullMessage = fullMsg
		m.loading = false
		m.calculateScrollLimits()

		// Mark as read if not already
		if !fullMsg.IsRead {
			go func() {
				if err := m.service.MarkRead(m.serviceState.CurrentAccount.ID, []string{fullMsg.ID}); err != nil {
					m.logger.Error("Failed to mark message as read", "id", fullMsg.ID, "error", err)
				}
			}()
		}

		return nil
	}
}

func (m *MessageModel) startReply() tea.Cmd {
	return func() tea.Msg {
		if m.fullMessage == nil {
			return nil
		}

		return ViewSwitchMsg{
			View: ViewCompose,
			Data: map[string]interface{}{
				"replyTo": m.fullMessage,
				"type":    "reply",
			},
		}
	}
}

func (m *MessageModel) startForward() tea.Cmd {
	return func() tea.Msg {
		if m.fullMessage == nil {
			return nil
		}

		return ViewSwitchMsg{
			View: ViewCompose,
			Data: map[string]interface{}{
				"message": m.fullMessage,
				"type":    "forward",
			},
		}
	}
}

// Utility functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}