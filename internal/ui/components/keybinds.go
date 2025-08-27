package components

import (
	"github.com/charmbracelet/bubbles/key"
)

// GlobalKeyMap defines global key bindings that work across all components
type GlobalKeyMap struct {
	Quit        key.Binding
	Help        key.Binding
	Tab         key.Binding
	ShiftTab    key.Binding
	Compose     key.Binding
	Search      key.Binding
	Settings    key.Binding
	Refresh     key.Binding
}

// DefaultGlobalKeyMap returns the default global key bindings
func DefaultGlobalKeyMap() GlobalKeyMap {
	return GlobalKeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous panel"),
		),
		Compose: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "compose email"),
		),
		Search: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "global search"),
		),
		Settings: key.NewBinding(
			key.WithKeys("ctrl+,"),
			key.WithHelp("ctrl+,", "settings"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("F5"),
			key.WithHelp("F5", "refresh all"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Tab, k.Compose, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.ShiftTab, k.Compose, k.Search},
		{k.Settings, k.Refresh, k.Help, k.Quit},
	}
}

// NavigationKeyMap defines common navigation key bindings
type NavigationKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Enter key.Binding
	Back  key.Binding
}

// DefaultNavigationKeyMap returns the default navigation key bindings
func DefaultNavigationKeyMap() NavigationKeyMap {
	return NavigationKeyMap{
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
	}
}

// ComponentFocus represents which component currently has focus
type ComponentFocus int

const (
	FocusAccountList ComponentFocus = iota
	FocusFolderTree
	FocusMessageList
	FocusMessageView
	FocusComposeForm
)

// String returns the string representation of the component focus
func (f ComponentFocus) String() string {
	switch f {
	case FocusAccountList:
		return "accounts"
	case FocusFolderTree:
		return "folders"
	case FocusMessageList:
		return "messages"
	case FocusMessageView:
		return "message"
	case FocusComposeForm:
		return "compose"
	default:
		return "unknown"
	}
}

// FocusableComponent interface for components that can receive focus
type FocusableComponent interface {
	Focus()
	Blur()
	Focused() bool
}

// FocusManager manages focus between components
type FocusManager struct {
	components []FocusableComponent
	focused    ComponentFocus
	enabled    bool
}

// NewFocusManager creates a new focus manager
func NewFocusManager() *FocusManager {
	return &FocusManager{
		components: make([]FocusableComponent, 0),
		focused:    FocusAccountList,
		enabled:    true,
	}
}

// AddComponent adds a component to the focus manager
func (fm *FocusManager) AddComponent(component FocusableComponent) {
	fm.components = append(fm.components, component)
}

// SetFocus sets focus to a specific component
func (fm *FocusManager) SetFocus(focus ComponentFocus) {
	if !fm.enabled || int(focus) >= len(fm.components) {
		return
	}
	
	// Blur current component
	if int(fm.focused) < len(fm.components) {
		fm.components[fm.focused].Blur()
	}
	
	// Focus new component
	fm.focused = focus
	fm.components[fm.focused].Focus()
}

// GetFocus returns the currently focused component
func (fm *FocusManager) GetFocus() ComponentFocus {
	return fm.focused
}

// NextFocus moves focus to the next component
func (fm *FocusManager) NextFocus() {
	nextFocus := (int(fm.focused) + 1) % len(fm.components)
	fm.SetFocus(ComponentFocus(nextFocus))
}

// PreviousFocus moves focus to the previous component
func (fm *FocusManager) PreviousFocus() {
	prevFocus := int(fm.focused) - 1
	if prevFocus < 0 {
		prevFocus = len(fm.components) - 1
	}
	fm.SetFocus(ComponentFocus(prevFocus))
}

// Enable enables or disables the focus manager
func (fm *FocusManager) Enable(enabled bool) {
	fm.enabled = enabled
	if !enabled {
		// Blur all components
		for _, component := range fm.components {
			component.Blur()
		}
	} else {
		// Focus current component
		if int(fm.focused) < len(fm.components) {
			fm.components[fm.focused].Focus()
		}
	}
}

// GetFocusedComponent returns the currently focused component
func (fm *FocusManager) GetFocusedComponent() FocusableComponent {
	if !fm.enabled || int(fm.focused) >= len(fm.components) {
		return nil
	}
	return fm.components[fm.focused]
}

// KeyBindingGroup represents a group of related key bindings for help display
type KeyBindingGroup struct {
	Title    string
	Bindings []key.Binding
}

// GetAllKeyBindings returns all key bindings organized by group for help display
func GetAllKeyBindings() []KeyBindingGroup {
	global := DefaultGlobalKeyMap()
	navigation := DefaultNavigationKeyMap()
	account := DefaultAccountListKeyMap()
	folder := DefaultFolderTreeKeyMap()
	message := DefaultMessageListKeyMap()
	messageView := DefaultMessageViewKeyMap()
	compose := DefaultComposeFormKeyMap()
	
	return []KeyBindingGroup{
		{
			Title: "Global",
			Bindings: []key.Binding{
				global.Compose, global.Search, global.Settings,
				global.Refresh, global.Help, global.Quit,
			},
		},
		{
			Title: "Navigation",
			Bindings: []key.Binding{
				navigation.Up, navigation.Down, navigation.Left, navigation.Right,
				navigation.Enter, navigation.Back, global.Tab, global.ShiftTab,
			},
		},
		{
			Title: "Accounts",
			Bindings: []key.Binding{
				account.Up, account.Down, account.Select,
				account.Refresh, account.AddAccount,
			},
		},
		{
			Title: "Folders",
			Bindings: []key.Binding{
				folder.Up, folder.Down, folder.Select, folder.Expand,
				folder.Collapse, folder.Refresh, folder.NewFolder, folder.Delete,
			},
		},
		{
			Title: "Messages",
			Bindings: []key.Binding{
				message.Up, message.Down, message.Select, message.ToggleRead,
				message.Flag, message.Delete, message.Reply, message.ReplyAll,
				message.Forward, message.Move, message.Search, message.MultiSelect,
				message.SelectAll, message.Refresh,
			},
		},
		{
			Title: "Message View",
			Bindings: []key.Binding{
				messageView.ScrollUp, messageView.ScrollDown, messageView.PageUp, messageView.PageDown,
				messageView.Reply, messageView.ReplyAll, messageView.Forward,
				messageView.Delete, messageView.Archive, messageView.ToggleRead, messageView.Flag,
				messageView.SaveAttach, messageView.ViewThread, messageView.Back,
			},
		},
		{
			Title: "Compose",
			Bindings: []key.Binding{
				compose.NextField, compose.PrevField, compose.Send, compose.SaveDraft,
				compose.AddAttachment, compose.QuoteOriginal, compose.Preview, compose.Cancel,
			},
		},
	}
}

// FormatKeyBindings formats key bindings for display in help
func FormatKeyBindings(groups []KeyBindingGroup) [][]key.Binding {
	var result [][]key.Binding
	
	for _, group := range groups {
		if len(group.Bindings) > 0 {
			result = append(result, group.Bindings)
		}
	}
	
	return result
}