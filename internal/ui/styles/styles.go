package styles

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	PrimaryColor   = lipgloss.Color("#7C3AED")
	SecondaryColor = lipgloss.Color("#EC4899") 
	AccentColor    = lipgloss.Color("#10B981")
	BackgroundColor = lipgloss.Color("#1E1E2E")
	SurfaceColor   = lipgloss.Color("#313244")
	TextColor      = lipgloss.Color("#CDD6F4")
	SubtleColor    = lipgloss.Color("#6C7086")
	ErrorColor     = lipgloss.Color("#F38BA8")
	WarningColor   = lipgloss.Color("#FAB387")
	SuccessColor   = lipgloss.Color("#A6E3A1")
)

// Base styles
var (
	// HeaderStyle for the application header
	HeaderStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Background(PrimaryColor).
		Padding(0, 1).
		Bold(true)

	// FooterStyle for the application footer
	FooterStyle = lipgloss.NewStyle().
		Foreground(SubtleColor).
		Background(SurfaceColor).
		Padding(0, 1)

	// TitleStyle for section titles
	TitleStyle = lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		MarginBottom(1)

	// SubtitleStyle for subsection titles
	SubtitleStyle = lipgloss.NewStyle().
		Foreground(SecondaryColor).
		Bold(true)

	// ContentStyle for main content areas
	ContentStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Padding(1)

	// SubtleStyle for less important text
	SubtleStyle = lipgloss.NewStyle().
		Foreground(SubtleColor)

	// ErrorStyle for error messages
	ErrorStyle = lipgloss.NewStyle().
		Foreground(ErrorColor).
		Bold(true)

	// WarningStyle for warning messages
	WarningStyle = lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(true)

	// SuccessStyle for success messages
	SuccessStyle = lipgloss.NewStyle().
		Foreground(SuccessColor).
		Bold(true)

	// BorderStyle for bordered containers
	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(SubtleColor).
		Padding(0, 1)

	// HighlightStyle for highlighted/selected items
	HighlightStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Background(PrimaryColor).
		Bold(true)

	// ButtonStyle for interactive buttons
	ButtonStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Background(AccentColor).
		Padding(0, 2).
		Bold(true).
		Border(lipgloss.RoundedBorder())

	// InputStyle for input fields
	InputStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Background(SurfaceColor).
		Padding(0, 1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(SubtleColor)

	// ActiveInputStyle for focused input fields
	ActiveInputStyle = InputStyle.Copy().
		BorderForeground(PrimaryColor)

	// ListItemStyle for list items
	ListItemStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Padding(0, 1)

	// SelectedListItemStyle for selected list items
	SelectedListItemStyle = ListItemStyle.Copy().
		Foreground(TextColor).
		Background(PrimaryColor).
		Bold(true)

	// UnreadStyle for unread items
	UnreadStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Bold(true)

	// ReadStyle for read items
	ReadStyle = lipgloss.NewStyle().
		Foreground(SubtleColor)
)

// Layout styles
var (
	// SidebarStyle for sidebar layout
	SidebarStyle = lipgloss.NewStyle().
		Width(25).
		Height(20).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(SubtleColor).
		Padding(1)

	// MainPaneStyle for main content pane
	MainPaneStyle = lipgloss.NewStyle().
		Padding(1)

	// PreviewPaneStyle for email preview pane
	PreviewPaneStyle = lipgloss.NewStyle().
		Width(40).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(SubtleColor).
		Padding(1)
)

// Email-specific styles
var (
	// EmailHeaderStyle for email headers
	EmailHeaderStyle = lipgloss.NewStyle().
		Foreground(SecondaryColor).
		Bold(true).
		MarginBottom(1)

	// EmailFromStyle for sender information
	EmailFromStyle = lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)

	// EmailSubjectStyle for email subjects
	EmailSubjectStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Bold(true)

	// EmailDateStyle for email dates
	EmailDateStyle = lipgloss.NewStyle().
		Foreground(SubtleColor)

	// EmailBodyStyle for email body content
	EmailBodyStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		MarginTop(1)

	// AttachmentStyle for attachment indicators
	AttachmentStyle = lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(true)

	// EncryptedStyle for encrypted email indicators
	EncryptedStyle = lipgloss.NewStyle().
		Foreground(SuccessColor).
		Bold(true)

	// SignedStyle for digitally signed email indicators
	SignedStyle = lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)
)

// Status styles
var (
	// StatusBarStyle for status bar
	StatusBarStyle = lipgloss.NewStyle().
		Foreground(TextColor).
		Background(SurfaceColor).
		Padding(0, 1)

	// ProgressStyle for progress indicators
	ProgressStyle = lipgloss.NewStyle().
		Foreground(AccentColor)

	// LoadingStyle for loading indicators
	LoadingStyle = lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)

	// OnlineStyle for online status
	OnlineStyle = lipgloss.NewStyle().
		Foreground(SuccessColor).
		Bold(true)

	// OfflineStyle for offline status
	OfflineStyle = lipgloss.NewStyle().
		Foreground(ErrorColor).
		Bold(true)
)

// Helper functions for dynamic styling

// GetThemeColors returns the current theme color palette
func GetThemeColors() map[string]lipgloss.Color {
	return map[string]lipgloss.Color{
		"primary":    PrimaryColor,
		"secondary":  SecondaryColor,
		"accent":     AccentColor,
		"background": BackgroundColor,
		"surface":    SurfaceColor,
		"text":       TextColor,
		"subtle":     SubtleColor,
		"error":      ErrorColor,
		"warning":    WarningColor,
		"success":    SuccessColor,
	}
}

// AdaptStyleToWidth adjusts style width based on terminal width
func AdaptStyleToWidth(style lipgloss.Style, width int) lipgloss.Style {
	return style.Width(width)
}

// AdaptStyleToHeight adjusts style height based on terminal height
func AdaptStyleToHeight(style lipgloss.Style, height int) lipgloss.Style {
	return style.Height(height)
}