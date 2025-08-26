package pages

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/ybarbara/pombo/internal/config"
	"github.com/ybarbara/pombo/internal/ui/models"
	"github.com/ybarbara/pombo/internal/ui/services"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// ViewState represents the current view state
type ViewState int

const (
	ViewWelcome ViewState = iota
	ViewMailbox
	ViewMessage
	ViewCompose
	ViewSettings
)

// MainModel represents the main application model
type MainModel struct {
	config     *config.Config
	logger     *log.Logger
	
	// Email service and model
	emailService services.EmailService
	emailModel   *models.EmailModel
	
	// UI state
	currentView  ViewState
	width        int
	height       int
	
	// Components
	help         help.Model
	
	// Key bindings
	keyMap       KeyMap
	
	// Application state
	ready        bool
	quitting     bool
	serviceReady bool
}

// KeyMap defines the key bindings for the application
type KeyMap struct {
	Quit    key.Binding
	Help    key.Binding
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Enter   key.Binding
	Back    key.Binding
	Compose key.Binding
	Reply   key.Binding
	Forward key.Binding
	Delete  key.Binding
	Search  key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Back},
		{k.Compose, k.Reply, k.Forward},
		{k.Delete, k.Search},
		{k.Help, k.Quit},
	}
}

// NewMainModel creates a new main model with email service integration
func NewMainModel(cfg *config.Config, logger *log.Logger, emailService services.EmailService) *MainModel {
	// Initialize key bindings
	keyMap := KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
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
		Forward: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "forward"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
	}

	// Create email model with service
	emailModel := models.NewEmailModel(emailService, cfg, logger)

	return &MainModel{
		config:       cfg,
		logger:       logger,
		emailService: emailService,
		emailModel:   emailModel,
		currentView:  ViewWelcome,
		help:         help.New(),
		keyMap:       keyMap,
		ready:        false,
		quitting:     false,
		serviceReady: false,
	}
}

// Init initializes the model
func (m *MainModel) Init() tea.Cmd {
	// Initialize email model first
	emailCmd := m.emailModel.Init()
	
	return tea.Batch(
		emailCmd,
		m.checkServiceStatus(),
	)
}

// Update handles messages and updates the model
func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		
		// Update help component size
		m.help.Width = msg.Width
		
		// Update email model size
		emailModel, emailCmd := m.emailModel.Update(msg)
		m.emailModel = emailModel.(*models.EmailModel)
		if emailCmd != nil {
			cmds = append(cmds, emailCmd)
		}
		
		return m, tea.Batch(cmds...)

	case ServiceReadyMsg:
		m.serviceReady = true
		m.currentView = ViewMailbox // Switch to mailbox once service is ready
		return m, nil

	case tea.KeyMsg:
		// Handle global keys first
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}

		// Delegate to email model if service is ready
		if m.serviceReady {
			emailModel, emailCmd := m.emailModel.Update(msg)
			m.emailModel = emailModel.(*models.EmailModel)
			if emailCmd != nil {
				cmds = append(cmds, emailCmd)
			}
		}
	
	default:
		// Always delegate other messages to email model
		if m.serviceReady {
			emailModel, emailCmd := m.emailModel.Update(msg)
			m.emailModel = emailModel.(*models.EmailModel)
			if emailCmd != nil {
				cmds = append(cmds, emailCmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view
func (m *MainModel) View() string {
	if !m.ready {
		return "Initializing POMBO..."
	}

	if m.quitting {
		return "Thanks for using POMBO! 📧"
	}

	// Show welcome screen until service is ready
	if !m.serviceReady {
		content := m.renderWelcomeView()
		return m.renderLayout(content)
	}

	// Delegate to email model when service is ready
	return m.emailModel.View()
}

// renderLayout renders the main application layout
func (m *MainModel) renderLayout(content string) string {
	// Calculate content height (total height minus header, footer, and margins)
	contentHeight := m.height - 4 // Header (1) + Footer (1) + Margins (2)
	
	// Header
	header := styles.HeaderStyle.Render("POMBO - Terminal Email Client")
	
	// Content area with proper height
	contentArea := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Render(content)
	
	// Footer with help
	helpView := m.help.View(m.keyMap)
	footer := styles.FooterStyle.Width(m.width).Render(helpView)
	
	// Combine all parts
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		contentArea,
		footer,
	)
}

// renderWelcomeView renders the welcome screen
func (m *MainModel) renderWelcomeView() string {
	title := styles.TitleStyle.Render("Welcome to POMBO!")
	
	features := []string{
		"📧 Multiple email account support",
		"🔐 OAuth2 authentication (Gmail, Outlook, etc.)",
		"⌨️  Vim-style keybindings",
		"🔍 Full-text search",
		"🔒 PGP encryption support",
		"⚡ Fast and lightweight",
	}
	
	var featureList strings.Builder
	for _, feature := range features {
		featureList.WriteString(fmt.Sprintf("  %s\n", feature))
	}
	
	instructions := styles.SubtleStyle.Render(
		"Press 'c' to compose an email, '?' for help, or 'q' to quit",
	)
	
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		title,
		"",
		featureList.String(),
		"",
		instructions,
		"",
	)
	
	return lipgloss.Place(
		m.width,
		m.height-4,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// ServiceReadyMsg indicates that the email service is ready
type ServiceReadyMsg struct{}

// checkServiceStatus checks if the email service is ready
func (m *MainModel) checkServiceStatus() tea.Cmd {
	return func() tea.Msg {
		if m.emailService.IsRunning() {
			return ServiceReadyMsg{}
		}
		// Keep checking until service is ready
		return tea.Tick(time.Second, func(time.Time) tea.Msg {
			return m.checkServiceStatus()()
		})
	}
}

// renderMailboxView renders the mailbox list view
func (m *MainModel) renderMailboxView() string {
	return styles.ContentStyle.Render("Mailbox view - Coming soon!")
}

// renderMessageView renders the message view
func (m *MainModel) renderMessageView() string {
	return styles.ContentStyle.Render("Message view - Coming soon!")
}

// renderComposeView renders the compose view
func (m *MainModel) renderComposeView() string {
	return styles.ContentStyle.Render("Compose view - Coming soon!\nPress 'esc' to go back.")
}

// renderSettingsView renders the settings view
func (m *MainModel) renderSettingsView() string {
	return styles.ContentStyle.Render("Settings view - Coming soon!")
}