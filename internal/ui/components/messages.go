package components

import (
	"github.com/ybarbara/pombo/internal/ui/services"
)

// MessageDetails represents the full message details for display
type MessageDetails struct {
	services.MessageInfo
	Body        string                      `json:"body"`
	BodyHTML    string                      `json:"body_html,omitempty"`
	Headers     map[string]string           `json:"headers"`
	Attachments []services.AttachmentInfo   `json:"attachments"`
	Thread      *ThreadInfo                 `json:"thread,omitempty"`
}

// ThreadInfo represents thread information for navigation
type ThreadInfo struct {
	ID           string                    `json:"id"`
	Subject      string                    `json:"subject"`
	Messages     []services.MessageInfo    `json:"messages"`
	CurrentIndex int                       `json:"current_index"`
	TotalCount   int                       `json:"total_count"`
}

// Shared message types for component integration
// These messages facilitate communication between different UI components

// Navigation Messages
type FocusChangeRequestMsg struct {
	Target ComponentFocus
}

// Account Selection Messages
type AccountSelectedMsg struct {
	AccountID   string
	AccountInfo *services.AccountInfo
}

// Folder Selection Messages (FolderSelectedMsg defined in folder_tree.go)

// Message Management Messages
type MessageOpenRequestMsg struct {
	Message services.MessageInfo
}

type MessageOpenedMsg struct {
	AccountID   string
	FolderName  string
	MessageID   string
}

type MessageReplyRequestMsg struct {
	Message  services.MessageInfo
	ReplyAll bool
}

type MessageForwardRequestMsg struct {
	Message services.MessageInfo
}

// Compose Messages
type ComposeNewMsg struct {
	AccountID string
}

type ComposeReplyMsg struct {
	AccountID       string
	OriginalMessage services.MessageInfo
}

type ComposeReplyAllMsg struct {
	AccountID       string
	OriginalMessage services.MessageInfo
}

type ComposeForwardMsg struct {
	AccountID       string
	OriginalMessage services.MessageInfo
}

type ComposeDraftMsg struct {
	AccountID string
	DraftID   string
	Draft     services.OutgoingMessage
}

type ComposeCompletedMsg struct {
	Success bool
}

type ComposeCancelledMsg struct{}

// Message View Messages
type MessageLoadedMsg struct {
	Message MessageDetails
}

type MessageLoadErrorMsg struct {
	Error string
}

type MessageDeletedMsg struct {
	MessageID string
}

type MessageArchivedMsg struct {
	MessageID string
}

type MessageReadToggleMsg struct {
	MessageID string
	IsRead    bool
}

type MessageFlagToggleMsg struct {
	MessageID string
	IsFlagged bool
}

// Thread Messages
type ThreadLoadedMsg struct {
	Thread ThreadInfo
}

type ThreadViewRequestMsg struct {
	ThreadID  string
	AccountID string
}

// Attachment Messages
type AttachmentSavedMsg struct {
	AttachmentID string
	Filename     string
}

type AttachmentAddedMsg struct {
	Attachment services.AttachmentInfo
}

// Navigation Messages
type BackToListRequestMsg struct{}

// State Messages
type MessagesRefreshedMsg struct {
	Messages []services.MessageInfo
}

type MessageRefreshErrorMsg struct {
	Error string
}

type SearchResultsMsg struct {
	Results services.SearchResults
}

// Operation Messages
type MessageSentMsg struct{}

type MessageSendErrorMsg struct {
	Error string
}

type DraftSavedMsg struct {
	DraftID string
}

type DraftSaveErrorMsg struct {
	Error string
}

type MessageOperationErrorMsg struct {
	Error string
}

// Status Messages
type StatusUpdateMsg struct {
	Message string
	Type    StatusType
}

type StatusType int

const (
	StatusInfo StatusType = iota
	StatusSuccess
	StatusWarning
	StatusError
)

// UI State Messages
type UIStateChangeMsg struct {
	State   UIState
	Context map[string]interface{}
}

type UIState int

const (
	StateAccountList UIState = iota
	StateFolderBrowse
	StateMessageList
	StateMessageView
	StateCompose
	StateSearch
	StateSettings
)