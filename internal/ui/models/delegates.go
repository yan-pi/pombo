package models

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ybarbara/pombo/internal/ui/styles"
)

// MessageDelegate provides custom rendering for message list items
type MessageDelegate struct{}

// NewMessageDelegate creates a new message delegate
func NewMessageDelegate() MessageDelegate {
	return MessageDelegate{}
}

// Height returns the height of a message item
func (d MessageDelegate) Height() int { return 3 }

// Spacing returns the spacing between message items
func (d MessageDelegate) Spacing() int { return 1 }

// Update handles updates for message items
func (d MessageDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render renders a message item
func (d MessageDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(MessageItem)
	if !ok {
		return
	}

	var (
		title       = i.Title()
		description = i.Description()
		isSelected  = index == m.Index()
		isRead      = i.IsRead
		isFlagged   = i.IsFlagged
		hasAttach   = i.HasAttachments
	)

	// Prepare status indicators
	var indicators []string
	if !isRead {
		indicators = append(indicators, "●")
	}
	if isFlagged {
		indicators = append(indicators, "⭐")
	}
	if hasAttach {
		indicators = append(indicators, "📎")
	}
	if i.IsEncrypted {
		indicators = append(indicators, "🔒")
	}

	statusStr := ""
	if len(indicators) > 0 {
		statusStr = strings.Join(indicators, " ") + " "
	}

	// Style based on read status and selection
	var titleStyle, descStyle, metaStyle lipgloss.Style

	if isSelected {
		if isRead {
			titleStyle = styles.SelectedListItemStyle
			descStyle = styles.SelectedListItemStyle.Copy().Faint(true)
		} else {
			titleStyle = styles.SelectedListItemStyle.Copy().Bold(true)
			descStyle = styles.SelectedListItemStyle.Copy().Faint(true)
		}
		metaStyle = styles.SelectedListItemStyle.Copy().Faint(true)
	} else {
		if isRead {
			titleStyle = styles.ReadStyle
			descStyle = styles.ReadStyle.Copy().Faint(true)
		} else {
			titleStyle = styles.UnreadStyle
			descStyle = styles.ListItemStyle.Copy().Faint(true)
		}
		metaStyle = styles.SubtleStyle
	}

	// Truncate title and description to fit
	maxWidth := m.Width() - 4 // Account for padding and indicators
	if len(statusStr) > 0 {
		maxWidth -= lipgloss.Width(statusStr)
	}

	if lipgloss.Width(title) > maxWidth {
		title = title[:max(0, maxWidth-3)] + "..."
	}

	if lipgloss.Width(description) > maxWidth {
		description = description[:max(0, maxWidth-3)] + "..."
	}

	// Format metadata line (from, date, size)
	metadata := fmt.Sprintf("%s • %s • %s", 
		i.FromDisplay, 
		i.DisplayDate, 
		i.SizeDisplay)

	if lipgloss.Width(metadata) > maxWidth {
		metadata = metadata[:max(0, maxWidth-3)] + "..."
	}

	// Render the item
	titleLine := statusStr + titleStyle.Render(title)
	descLine := "  " + descStyle.Render(description)
	metaLine := "  " + metaStyle.Render(metadata)

	output := lipgloss.JoinVertical(lipgloss.Left, titleLine, descLine, metaLine)
	
	fmt.Fprint(w, output)
}

// FolderDelegate provides custom rendering for folder list items
type FolderDelegate struct{}

// NewFolderDelegate creates a new folder delegate
func NewFolderDelegate() FolderDelegate {
	return FolderDelegate{}
}

// Height returns the height of a folder item
func (d FolderDelegate) Height() int { return 2 }

// Spacing returns the spacing between folder items
func (d FolderDelegate) Spacing() int { return 0 }

// Update handles updates for folder items
func (d FolderDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render renders a folder item
func (d FolderDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(FolderItem)
	if !ok {
		return
	}

	var (
		title      = i.Title()
		desc       = i.Description()
		isSelected = index == m.Index()
		hasUnread  = i.UnreadCount > 0
	)

	// Style based on selection and unread status
	var titleStyle, descStyle lipgloss.Style

	if isSelected {
		titleStyle = styles.SelectedListItemStyle
		descStyle = styles.SelectedListItemStyle.Copy().Faint(true)
	} else {
		if hasUnread {
			titleStyle = styles.UnreadStyle
		} else {
			titleStyle = styles.ListItemStyle
		}
		descStyle = styles.SubtleStyle
	}

	// Truncate to fit
	maxWidth := m.Width() - 4
	if lipgloss.Width(title) > maxWidth {
		title = title[:max(0, maxWidth-3)] + "..."
	}
	if lipgloss.Width(desc) > maxWidth {
		desc = desc[:max(0, maxWidth-3)] + "..."
	}

	// Add unread count indicator
	if hasUnread && !isSelected {
		unreadIndicator := styles.WarningStyle.Render(fmt.Sprintf(" (%d)", i.UnreadCount))
		availableWidth := maxWidth - lipgloss.Width(unreadIndicator)
		if lipgloss.Width(title) > availableWidth {
			title = title[:max(0, availableWidth-3)] + "..."
		}
		title += unreadIndicator
	}

	// Render the item
	titleLine := titleStyle.Render(title)
	descLine := "  " + descStyle.Render(desc)

	output := lipgloss.JoinVertical(lipgloss.Left, titleLine, descLine)
	
	fmt.Fprint(w, output)
}

// AccountDelegate provides custom rendering for account list items
type AccountDelegate struct{}

// NewAccountDelegate creates a new account delegate
func NewAccountDelegate() AccountDelegate {
	return AccountDelegate{}
}

// Height returns the height of an account item
func (d AccountDelegate) Height() int { return 3 }

// Spacing returns the spacing between account items
func (d AccountDelegate) Spacing() int { return 1 }

// Update handles updates for account items
func (d AccountDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render renders an account item
func (d AccountDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// This would be implemented for account management view
	// For now, just render basic info
	fmt.Fprint(w, "Account item rendering not implemented")
}