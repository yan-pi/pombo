package models

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// AccountModel represents the account management view
type AccountModel struct {
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

	// Account list state
	accounts     []services.AccountInfo
	selectedIdx  int
}

// NewAccountModel creates a new account model
func NewAccountModel(service services.EmailService, logger *log.Logger) *AccountModel {
	return &AccountModel{
		service:     service,
		logger:      logger,
		ready:       false,
		loading:     false,
		accounts:    make([]services.AccountInfo, 0),
		selectedIdx: 0,
	}
}

// SetSize updates the model dimensions
func (m *AccountModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.ready = true
}

// Update handles messages and updates the account model
func (m *AccountModel) Update(msg tea.Msg) (*AccountModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selectedIdx < len(m.accounts)-1 {
				m.selectedIdx++
			}
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "enter":
			return m, m.selectAccount()
		}
	}

	return m, nil
}

// View renders the account view
func (m *AccountModel) View() string {
	if !m.ready {
		return "Loading accounts..."
	}

	header := styles.TitleStyle.Render("Account Management")
	
	if len(m.accounts) == 0 {
		content := styles.SubtleStyle.Render("No accounts configured")
		return lipgloss.JoinVertical(lipgloss.Left, header, "", content)
	}

	var accountList []string
	for i, account := range m.accounts {
		var style lipgloss.Style
		if i == m.selectedIdx {
			style = styles.SelectedListItemStyle
		} else {
			style = styles.ListItemStyle
		}

		status := "●"
		if account.Connected {
			status = styles.SuccessStyle.Render("●")
		} else {
			status = styles.ErrorStyle.Render("●")
		}

		accountLine := fmt.Sprintf("%s %s (%s) - %d unread", 
			status, account.Name, account.Email, account.UnreadCount)
		
		accountList = append(accountList, style.Render(accountLine))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, accountList...)
	footer := styles.SubtleStyle.Render("j/k: navigate • enter: select • esc: back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		content,
		"",
		footer,
	)
}

// UpdateState updates the model with new service state
func (m *AccountModel) UpdateState(state *services.ServiceState) {
	m.serviceState = state
	m.accounts = state.Accounts
	
	// Ensure selectedIdx is valid
	if m.selectedIdx >= len(m.accounts) {
		m.selectedIdx = max(0, len(m.accounts)-1)
	}
}

func (m *AccountModel) selectAccount() tea.Cmd {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.accounts) {
		return nil
	}

	selectedAccount := m.accounts[m.selectedIdx]
	
	return func() tea.Msg {
		if err := m.service.SwitchAccount(selectedAccount.ID); err != nil {
			m.logger.Error("Failed to switch account", "id", selectedAccount.ID, "error", err)
		}
		
		// Switch back to mailbox view
		return ViewSwitchMsg{View: ViewMailbox}
	}
}