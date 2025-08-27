package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// AccountListKeyMap defines key bindings for the account list
type AccountListKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Refresh  key.Binding
	AddAccount key.Binding
}

// DefaultAccountListKeyMap returns the default key bindings for account list
func DefaultAccountListKeyMap() AccountListKeyMap {
	return AccountListKeyMap{
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
			key.WithHelp("enter", "switch account"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		AddAccount: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add account"),
		),
	}
}

// AccountList represents the account list component
type AccountList struct {
	service     services.EmailService
	accounts    []services.AccountInfo
	selectedIdx int
	width       int
	height      int
	focused     bool
	keyMap      AccountListKeyMap
	
	// State tracking
	loading     bool
	lastUpdate  string
	error       string
}

// NewAccountList creates a new account list component
func NewAccountList(service services.EmailService) *AccountList {
	return &AccountList{
		service:     service,
		accounts:    make([]services.AccountInfo, 0),
		selectedIdx: 0,
		focused:     false,
		keyMap:      DefaultAccountListKeyMap(),
		loading:     false,
	}
}

// Init initializes the account list component
func (al *AccountList) Init() tea.Cmd {
	return al.refreshAccounts()
}

// Update handles messages and updates the account list
func (al *AccountList) Update(msg tea.Msg) (*AccountList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		al.width = msg.Width
		al.height = msg.Height
		return al, nil

	case tea.KeyMsg:
		if !al.focused {
			return al, nil
		}

		switch {
		case key.Matches(msg, al.keyMap.Up):
			if al.selectedIdx > 0 {
				al.selectedIdx--
			}
			return al, nil

		case key.Matches(msg, al.keyMap.Down):
			if al.selectedIdx < len(al.accounts)-1 {
				al.selectedIdx++
			}
			return al, nil

		case key.Matches(msg, al.keyMap.Select):
			return al, al.switchAccount()

		case key.Matches(msg, al.keyMap.Refresh):
			return al, al.refreshAccounts()

		case key.Matches(msg, al.keyMap.AddAccount):
			// Future: Open add account dialog
			return al, nil
		}

	case services.ServiceUpdate:
		// Handle real-time service updates
		switch msg.Type {
		case services.UpdateTypeAccountAdded,
			 services.UpdateTypeAccountRemoved,
			 services.UpdateTypeAccountConnected,
			 services.UpdateTypeAccountError:
			return al, al.refreshAccounts()
		}

	case AccountsRefreshedMsg:
		al.loading = false
		al.accounts = msg.Accounts
		al.error = ""
		
		// Ensure selected index is valid
		if al.selectedIdx >= len(al.accounts) {
			al.selectedIdx = len(al.accounts) - 1
		}
		if al.selectedIdx < 0 {
			al.selectedIdx = 0
		}
		
		return al, nil

	case AccountRefreshErrorMsg:
		al.loading = false
		al.error = msg.Error
		return al, nil
	}

	return al, nil
}

// View renders the account list
func (al *AccountList) View() string {
	if al.width == 0 || al.height == 0 {
		return ""
	}

	var content strings.Builder
	
	// Header
	headerText := "Accounts"
	if al.loading {
		headerText += " (Loading...)"
	}
	
	header := styles.SubtitleStyle.Render(headerText)
	content.WriteString(header)
	content.WriteString("\n")
	
	// Error display
	if al.error != "" {
		errorText := styles.ErrorStyle.Render("Error: " + al.error)
		content.WriteString(errorText)
		content.WriteString("\n")
	}

	// Account list
	if len(al.accounts) == 0 {
		if !al.loading {
			noAccounts := styles.SubtleStyle.Render("No accounts configured")
			content.WriteString(noAccounts)
		}
	} else {
		for i, account := range al.accounts {
			accountView := al.renderAccount(account, i == al.selectedIdx)
			content.WriteString(accountView)
			content.WriteString("\n")
		}
	}

	// Footer with keybindings if focused
	if al.focused {
		content.WriteString("\n")
		keybindings := al.renderKeybindings()
		content.WriteString(keybindings)
	}

	// Apply container styling
	containerStyle := styles.SidebarStyle.
		Width(al.width).
		Height(al.height)
	
	if al.focused {
		containerStyle = containerStyle.
			BorderForeground(styles.PrimaryColor)
	}

	return containerStyle.Render(content.String())
}

// renderAccount renders a single account item
func (al *AccountList) renderAccount(account services.AccountInfo, selected bool) string {
	var content strings.Builder
	
	// Account name and email
	nameStyle := styles.ListItemStyle
	emailStyle := styles.SubtleStyle
	
	if selected && al.focused {
		nameStyle = styles.SelectedListItemStyle
		emailStyle = styles.SelectedListItemStyle
	} else if account.UnreadCount > 0 {
		nameStyle = styles.UnreadStyle
	}

	// Status indicator
	var statusIcon string
	var statusColor lipgloss.Style
	
	switch {
	case !account.Connected:
		statusIcon = "●"
		statusColor = styles.OfflineStyle
	case account.Error != nil:
		statusIcon = "●"
		statusColor = styles.ErrorStyle
	default:
		statusIcon = "●"
		statusColor = styles.OnlineStyle
	}

	// Build the account line
	content.WriteString(statusColor.Render(statusIcon))
	content.WriteString(" ")
	content.WriteString(nameStyle.Render(account.Name))
	
	// Show unread count if any
	if account.UnreadCount > 0 {
		unreadText := fmt.Sprintf(" (%d)", account.UnreadCount)
		content.WriteString(lipgloss.NewStyle().Foreground(styles.AccentColor).Render(unreadText))
	}
	
	// Second line with email address (if space allows)
	if al.height > 10 && len(al.accounts) < 5 {
		content.WriteString("\n  ")
		content.WriteString(emailStyle.Render(account.Email))
	}

	return content.String()
}

// renderKeybindings renders the keybindings help
func (al *AccountList) renderKeybindings() string {
	bindings := []string{
		"j/k: navigate",
		"enter: switch",
		"r: refresh",
	}
	
	bindingText := strings.Join(bindings, " • ")
	return styles.SubtleStyle.Render(bindingText)
}

// Focus sets focus to the account list
func (al *AccountList) Focus() {
	al.focused = true
}

// Blur removes focus from the account list
func (al *AccountList) Blur() {
	al.focused = false
}

// Focused returns whether the account list is focused
func (al *AccountList) Focused() bool {
	return al.focused
}

// GetSelectedAccount returns the currently selected account
func (al *AccountList) GetSelectedAccount() *services.AccountInfo {
	if len(al.accounts) == 0 || al.selectedIdx < 0 || al.selectedIdx >= len(al.accounts) {
		return nil
	}
	return &al.accounts[al.selectedIdx]
}

// SetSize sets the dimensions of the account list
func (al *AccountList) SetSize(width, height int) {
	al.width = width
	al.height = height
}

// refreshAccounts refreshes the account list from the service
func (al *AccountList) refreshAccounts() tea.Cmd {
	al.loading = true
	
	return func() tea.Msg {
		accounts := al.service.GetAccounts()
		if accounts == nil {
			return AccountRefreshErrorMsg{Error: "Failed to fetch accounts"}
		}
		
		return AccountsRefreshedMsg{Accounts: accounts}
	}
}

// switchAccount switches to the selected account
func (al *AccountList) switchAccount() tea.Cmd {
	if al.selectedIdx < 0 || al.selectedIdx >= len(al.accounts) {
		return nil
	}
	
	selectedAccount := al.accounts[al.selectedIdx]
	
	return func() tea.Msg {
		err := al.service.SwitchAccount(selectedAccount.ID)
		if err != nil {
			return AccountSwitchErrorMsg{Error: err.Error()}
		}
		
		return AccountSwitchedMsg{AccountID: selectedAccount.ID}
	}
}

// Message types for account list operations
type AccountsRefreshedMsg struct {
	Accounts []services.AccountInfo
}

type AccountRefreshErrorMsg struct {
	Error string
}

type AccountSwitchedMsg struct {
	AccountID string
}

type AccountSwitchErrorMsg struct {
	Error string
}