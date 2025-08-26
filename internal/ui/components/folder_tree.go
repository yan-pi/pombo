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

// FolderTreeKeyMap defines key bindings for the folder tree
type FolderTreeKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Select     key.Binding
	Expand     key.Binding
	Collapse   key.Binding
	Refresh    key.Binding
	NewFolder  key.Binding
	Delete     key.Binding
}

// DefaultFolderTreeKeyMap returns the default key bindings for folder tree
func DefaultFolderTreeKeyMap() FolderTreeKeyMap {
	return FolderTreeKeyMap{
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
			key.WithHelp("enter", "select folder"),
		),
		Expand: key.NewBinding(
			key.WithKeys("o", "right"),
			key.WithHelp("o/→", "expand folder"),
		),
		Collapse: key.NewBinding(
			key.WithKeys("c", "left"),
			key.WithHelp("c/←", "collapse folder"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		NewFolder: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new folder"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
	}
}

// FolderTreeNode represents a node in the folder tree
type FolderTreeNode struct {
	Folder     services.FolderInfo
	Children   []*FolderTreeNode
	Parent     *FolderTreeNode
	Expanded   bool
	Level      int
	IsVisible  bool
}

// FolderTree represents the folder tree component
type FolderTree struct {
	service      services.EmailService
	accountID    string
	folders      []services.FolderInfo
	tree         *FolderTreeNode
	flatList     []*FolderTreeNode
	selectedIdx  int
	width        int
	height       int
	focused      bool
	keyMap       FolderTreeKeyMap
	
	// State tracking
	loading      bool
	error        string
	selectedFolder *services.FolderInfo
}

// NewFolderTree creates a new folder tree component
func NewFolderTree(service services.EmailService) *FolderTree {
	return &FolderTree{
		service:     service,
		folders:     make([]services.FolderInfo, 0),
		selectedIdx: 0,
		focused:     false,
		keyMap:      DefaultFolderTreeKeyMap(),
		loading:     false,
		flatList:    make([]*FolderTreeNode, 0),
	}
}

// Init initializes the folder tree component
func (ft *FolderTree) Init() tea.Cmd {
	return ft.refreshFolders()
}

// Update handles messages and updates the folder tree
func (ft *FolderTree) Update(msg tea.Msg) (*FolderTree, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ft.width = msg.Width
		ft.height = msg.Height
		return ft, nil

	case tea.KeyMsg:
		if !ft.focused {
			return ft, nil
		}

		switch {
		case key.Matches(msg, ft.keyMap.Up):
			if ft.selectedIdx > 0 {
				ft.selectedIdx--
				ft.updateSelectedFolder()
			}
			return ft, nil

		case key.Matches(msg, ft.keyMap.Down):
			if ft.selectedIdx < len(ft.flatList)-1 {
				ft.selectedIdx++
				ft.updateSelectedFolder()
			}
			return ft, nil

		case key.Matches(msg, ft.keyMap.Select):
			return ft, ft.selectFolder()

		case key.Matches(msg, ft.keyMap.Expand):
			ft.expandSelected()
			return ft, nil

		case key.Matches(msg, ft.keyMap.Collapse):
			ft.collapseSelected()
			return ft, nil

		case key.Matches(msg, ft.keyMap.Refresh):
			return ft, ft.refreshFolders()

		case key.Matches(msg, ft.keyMap.NewFolder):
			// Future: Open new folder dialog
			return ft, nil

		case key.Matches(msg, ft.keyMap.Delete):
			// Future: Delete folder with confirmation
			return ft, nil
		}

	case services.ServiceUpdate:
		// Handle real-time service updates
		switch msg.Type {
		case services.UpdateTypeFolderRefreshed:
			if msg.AccountID == ft.accountID {
				return ft, ft.refreshFolders()
			}
		}

	case AccountSwitchedMsg:
		ft.accountID = msg.AccountID
		return ft, ft.refreshFolders()

	case FoldersRefreshedMsg:
		ft.loading = false
		ft.folders = msg.Folders
		ft.error = ""
		ft.buildTree()
		ft.updateSelectedFolder()
		return ft, nil

	case FolderRefreshErrorMsg:
		ft.loading = false
		ft.error = msg.Error
		return ft, nil
	}

	return ft, nil
}

// View renders the folder tree
func (ft *FolderTree) View() string {
	if ft.width == 0 || ft.height == 0 {
		return ""
	}

	var content strings.Builder
	
	// Header
	headerText := "Folders"
	if ft.loading {
		headerText += " (Loading...)"
	}
	
	header := styles.SubtitleStyle.Render(headerText)
	content.WriteString(header)
	content.WriteString("\n")
	
	// Error display
	if ft.error != "" {
		errorText := styles.ErrorStyle.Render("Error: " + ft.error)
		content.WriteString(errorText)
		content.WriteString("\n")
	}

	// Account info
	if ft.accountID != "" && ft.service.GetCurrentAccount() != nil {
		accountInfo := ft.service.GetCurrentAccount()
		accountText := styles.SubtleStyle.Render(fmt.Sprintf("Account: %s", accountInfo.Name))
		content.WriteString(accountText)
		content.WriteString("\n")
	}

	// Folder tree
	if len(ft.flatList) == 0 {
		if !ft.loading {
			noFolders := styles.SubtleStyle.Render("No folders available")
			content.WriteString(noFolders)
		}
	} else {
		// Calculate available height for folders
		usedHeight := 4 // Header, account info, spacing
		if ft.error != "" {
			usedHeight++
		}
		if ft.focused {
			usedHeight += 2 // Keybindings
		}
		
		availableHeight := ft.height - usedHeight
		startIdx := 0
		endIdx := len(ft.flatList)
		
		// Handle scrolling if content exceeds available height
		if len(ft.flatList) > availableHeight {
			if ft.selectedIdx >= availableHeight/2 {
				startIdx = ft.selectedIdx - availableHeight/2
				endIdx = startIdx + availableHeight
				if endIdx > len(ft.flatList) {
					endIdx = len(ft.flatList)
					startIdx = endIdx - availableHeight
				}
			} else {
				endIdx = availableHeight
			}
		}
		
		for i := startIdx; i < endIdx; i++ {
			if i < len(ft.flatList) {
				folderView := ft.renderFolderNode(ft.flatList[i], i == ft.selectedIdx)
				content.WriteString(folderView)
				content.WriteString("\n")
			}
		}
	}

	// Footer with keybindings if focused
	if ft.focused {
		content.WriteString("\n")
		keybindings := ft.renderKeybindings()
		content.WriteString(keybindings)
	}

	// Apply container styling
	containerStyle := styles.SidebarStyle.
		Width(ft.width).
		Height(ft.height)
	
	if ft.focused {
		containerStyle = containerStyle.
			BorderForeground(styles.PrimaryColor)
	}

	return containerStyle.Render(content.String())
}

// renderFolderNode renders a single folder node
func (ft *FolderTree) renderFolderNode(node *FolderTreeNode, selected bool) string {
	var content strings.Builder
	
	// Indentation based on level
	indent := strings.Repeat("  ", node.Level)
	content.WriteString(indent)
	
	// Expansion indicator for folders with children
	if len(node.Children) > 0 {
		if node.Expanded {
			content.WriteString("▼ ")
		} else {
			content.WriteString("▶ ")
		}
	} else {
		content.WriteString("  ")
	}
	
	// Folder icon based on type
	icon := ft.getFolderIcon(node.Folder.Type)
	content.WriteString(icon)
	content.WriteString(" ")
	
	// Folder name with styling
	nameStyle := styles.ListItemStyle
	if selected && ft.focused {
		nameStyle = styles.SelectedListItemStyle
	} else if node.Folder.UnreadCount > 0 {
		nameStyle = styles.UnreadStyle
	}
	
	content.WriteString(nameStyle.Render(node.Folder.Name))
	
	// Unread count
	if node.Folder.UnreadCount > 0 {
		unreadText := fmt.Sprintf(" (%d)", node.Folder.UnreadCount)
		content.WriteString(lipgloss.NewStyle().Foreground(styles.AccentColor).Render(unreadText))
	}
	
	// Message count for special folders
	if node.Folder.Type == "inbox" || node.Folder.Type == "sent" {
		totalText := fmt.Sprintf(" [%d]", node.Folder.MessageCount)
		content.WriteString(styles.SubtleStyle.Render(totalText))
	}

	return content.String()
}

// getFolderIcon returns the appropriate icon for a folder type
func (ft *FolderTree) getFolderIcon(folderType string) string {
	switch folderType {
	case "inbox":
		return "📥"
	case "sent":
		return "📤"
	case "drafts":
		return "📝"
	case "trash":
		return "🗑️"
	case "spam", "junk":
		return "🚫"
	case "archive":
		return "📦"
	default:
		return "📁"
	}
}

// renderKeybindings renders the keybindings help
func (ft *FolderTree) renderKeybindings() string {
	bindings := []string{
		"j/k: navigate",
		"enter: select",
		"o: expand",
		"c: collapse",
		"r: refresh",
	}
	
	bindingText := strings.Join(bindings, " • ")
	return styles.SubtleStyle.Render(bindingText)
}

// buildTree constructs the folder tree from the flat folder list
func (ft *FolderTree) buildTree() {
	if len(ft.folders) == 0 {
		ft.tree = nil
		ft.flatList = make([]*FolderTreeNode, 0)
		return
	}

	// Create nodes for all folders
	nodeMap := make(map[string]*FolderTreeNode)
	var rootNodes []*FolderTreeNode
	
	for _, folder := range ft.folders {
		node := &FolderTreeNode{
			Folder:    folder,
			Children:  make([]*FolderTreeNode, 0),
			Expanded:  folder.Type == "inbox" || strings.HasPrefix(folder.Name, "INBOX"), // Auto-expand INBOX
			Level:     0,
			IsVisible: true,
		}
		nodeMap[folder.FullName] = node
		
		// For now, treat all folders as root level
		// TODO: Implement proper hierarchical parsing based on folder separator
		rootNodes = append(rootNodes, node)
	}
	
	// Sort folders by type and name (INBOX first, then alphabetical)
	ft.sortFolders(rootNodes)
	
	// Create a virtual root
	ft.tree = &FolderTreeNode{
		Children:  rootNodes,
		Expanded:  true,
		Level:     -1,
		IsVisible: false,
	}
	
	// Set parent references and levels
	for _, child := range rootNodes {
		child.Parent = ft.tree
		child.Level = 0
	}
	
	// Build flat list for rendering
	ft.rebuildFlatList()
}

// sortFolders sorts folders with special folders first, then alphabetically
func (ft *FolderTree) sortFolders(nodes []*FolderTreeNode) {
	// Priority order for special folders
	priority := map[string]int{
		"inbox":   1,
		"sent":    2,
		"drafts":  3,
		"archive": 4,
		"spam":    5,
		"junk":    5,
		"trash":   6,
	}
	
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			iPriority := priority[nodes[i].Folder.Type]
			jPriority := priority[nodes[j].Folder.Type]
			
			// Sort by priority first, then by name
			if iPriority == 0 && jPriority == 0 {
				// Both are regular folders, sort alphabetically
				if nodes[i].Folder.Name > nodes[j].Folder.Name {
					nodes[i], nodes[j] = nodes[j], nodes[i]
				}
			} else if iPriority == 0 {
				// i is regular, j is special, j comes first
				nodes[i], nodes[j] = nodes[j], nodes[i]
			} else if jPriority == 0 {
				// i is special, j is regular, i comes first
				continue
			} else {
				// Both are special, sort by priority
				if iPriority > jPriority {
					nodes[i], nodes[j] = nodes[j], nodes[i]
				}
			}
		}
	}
}

// rebuildFlatList rebuilds the flat list for rendering
func (ft *FolderTree) rebuildFlatList() {
	ft.flatList = make([]*FolderTreeNode, 0)
	if ft.tree != nil {
		ft.addNodeToFlatList(ft.tree)
	}
}

// addNodeToFlatList recursively adds nodes to the flat list
func (ft *FolderTree) addNodeToFlatList(node *FolderTreeNode) {
	if node.IsVisible && node.Level >= 0 {
		ft.flatList = append(ft.flatList, node)
	}
	
	if node.Expanded {
		for _, child := range node.Children {
			ft.addNodeToFlatList(child)
		}
	}
}

// expandSelected expands the selected folder
func (ft *FolderTree) expandSelected() {
	if ft.selectedIdx >= 0 && ft.selectedIdx < len(ft.flatList) {
		node := ft.flatList[ft.selectedIdx]
		if len(node.Children) > 0 {
			node.Expanded = true
			ft.rebuildFlatList()
		}
	}
}

// collapseSelected collapses the selected folder
func (ft *FolderTree) collapseSelected() {
	if ft.selectedIdx >= 0 && ft.selectedIdx < len(ft.flatList) {
		node := ft.flatList[ft.selectedIdx]
		if len(node.Children) > 0 {
			node.Expanded = false
			ft.rebuildFlatList()
			
			// Ensure selected index is still valid
			if ft.selectedIdx >= len(ft.flatList) {
				ft.selectedIdx = len(ft.flatList) - 1
			}
		}
	}
}

// updateSelectedFolder updates the selected folder reference
func (ft *FolderTree) updateSelectedFolder() {
	if ft.selectedIdx >= 0 && ft.selectedIdx < len(ft.flatList) {
		ft.selectedFolder = &ft.flatList[ft.selectedIdx].Folder
	} else {
		ft.selectedFolder = nil
	}
}

// Focus sets focus to the folder tree
func (ft *FolderTree) Focus() {
	ft.focused = true
}

// Blur removes focus from the folder tree
func (ft *FolderTree) Blur() {
	ft.focused = false
}

// Focused returns whether the folder tree is focused
func (ft *FolderTree) Focused() bool {
	return ft.focused
}

// GetSelectedFolder returns the currently selected folder
func (ft *FolderTree) GetSelectedFolder() *services.FolderInfo {
	return ft.selectedFolder
}

// SetSize sets the dimensions of the folder tree
func (ft *FolderTree) SetSize(width, height int) {
	ft.width = width
	ft.height = height
}

// SetAccount sets the current account ID
func (ft *FolderTree) SetAccount(accountID string) {
	if ft.accountID != accountID {
		ft.accountID = accountID
		ft.selectedIdx = 0
		ft.selectedFolder = nil
	}
}

// refreshFolders refreshes the folder list from the service
func (ft *FolderTree) refreshFolders() tea.Cmd {
	if ft.accountID == "" {
		return nil
	}
	
	ft.loading = true
	
	return func() tea.Msg {
		folders, err := ft.service.GetFolders(ft.accountID)
		if err != nil {
			return FolderRefreshErrorMsg{Error: err.Error()}
		}
		
		return FoldersRefreshedMsg{Folders: folders}
	}
}

// selectFolder selects the current folder
func (ft *FolderTree) selectFolder() tea.Cmd {
	if ft.selectedFolder == nil {
		return nil
	}
	
	return func() tea.Msg {
		err := ft.service.SelectFolder(ft.accountID, ft.selectedFolder.FullName)
		if err != nil {
			return FolderSelectErrorMsg{Error: err.Error()}
		}
		
		return FolderSelectedMsg{
			AccountID:  ft.accountID,
			FolderName: ft.selectedFolder.FullName,
		}
	}
}

// Message types for folder tree operations
type FoldersRefreshedMsg struct {
	Folders []services.FolderInfo
}

type FolderRefreshErrorMsg struct {
	Error string
}

type FolderSelectedMsg struct {
	AccountID  string
	FolderName string
}

type FolderSelectErrorMsg struct {
	Error string
}