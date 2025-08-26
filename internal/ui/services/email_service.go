// Package services provides the service layer that bridges the email backend with the TUI frontend
package services

import (
	"context"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/ybarbara/pombo/internal/config"
	"github.com/ybarbara/pombo/internal/email"
)

// EmailService provides the main interface between TUI and email backend
// This service abstracts email operations and manages real-time state synchronization
type EmailService interface {
	// Lifecycle management
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool

	// Account management
	AddAccount(account *config.AccountConfig) error
	RemoveAccount(accountID string) error
	SwitchAccount(accountID string) error
	GetAccounts() []AccountInfo
	GetCurrentAccount() *AccountInfo

	// Folder operations
	GetFolders(accountID string) ([]FolderInfo, error)
	SelectFolder(accountID, folderName string) error
	RefreshFolder(accountID, folderName string) error
	GetCurrentFolder() *FolderInfo

	// Message operations
	GetMessages(accountID, folderName string, limit int) ([]MessageInfo, error)
	GetMessage(accountID, messageID string) (*MessageInfo, error)
	SearchMessages(accountID string, query *SearchQuery) (*SearchResults, error)
	SendMessage(accountID string, msg *OutgoingMessage) error

	// Message management
	MarkRead(accountID string, messageIDs []string) error
	MarkUnread(accountID string, messageIDs []string) error
	FlagMessage(accountID string, messageIDs []string) error
	UnflagMessage(accountID string, messageIDs []string) error
	DeleteMessage(accountID string, messageIDs []string) error
	MoveMessage(accountID string, messageIDs []string, targetFolder string) error

	// Real-time updates
	GetUpdateChannel() <-chan ServiceUpdate
	EnableRealTimeUpdates(enabled bool) error

	// State management
	GetState() *ServiceState
	RefreshState() error
	
	// Statistics and monitoring
	GetConnectionStatus() map[string]ConnectionInfo
	GetStatistics() ServiceStatistics
}

// EmailServiceImpl implements the EmailService interface
type EmailServiceImpl struct {
	// Core dependencies
	pool          email.ConnectionManager
	config        *config.Config
	logger        *log.Logger
	authFactory   *email.AuthProviderFactory
	clientFactory email.ClientFactory

	// State management
	state         *ServiceState
	stateMutex    sync.RWMutex

	// Update channel for real-time notifications
	updateChan    chan ServiceUpdate
	subscribers   map[string]chan<- ServiceUpdate
	subMutex      sync.RWMutex

	// Background processing
	ctx           context.Context
	cancel        context.CancelFunc
	bgWaitGroup   sync.WaitGroup
	running       bool
	runningMutex  sync.RWMutex

	// Caching and performance
	messageCache  map[string]*MessageInfo
	folderCache   map[string][]FolderInfo
	cacheMutex    sync.RWMutex
	cacheExpiry   time.Duration
}

// ServiceState represents the current state of the email service
type ServiceState struct {
	// Account state
	Accounts        []AccountInfo `json:"accounts"`
	CurrentAccount  *AccountInfo  `json:"current_account,omitempty"`

	// Folder state
	Folders         []FolderInfo  `json:"folders"`
	CurrentFolder   *FolderInfo   `json:"current_folder,omitempty"`

	// Message state
	Messages        []MessageInfo `json:"messages"`
	SelectedMessage *MessageInfo  `json:"selected_message,omitempty"`
	TotalMessages   int           `json:"total_messages"`
	UnreadCount     int           `json:"unread_count"`

	// UI state
	Loading         bool          `json:"loading"`
	Error          *ServiceError  `json:"error,omitempty"`
	LastUpdate     time.Time     `json:"last_update"`

	// Real-time sync state
	SyncStatus      SyncStatus    `json:"sync_status"`
	LastSync        time.Time     `json:"last_sync"`
}

// AccountInfo represents UI-friendly account information
type AccountInfo struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Provider        string    `json:"provider"`
	Status          string    `json:"status"`
	Connected       bool      `json:"connected"`
	UnreadCount     int       `json:"unread_count"`
	TotalMessages   int       `json:"total_messages"`
	LastSync        time.Time `json:"last_sync"`
	Error          *string    `json:"error,omitempty"`
}

// FolderInfo represents UI-friendly folder information
type FolderInfo struct {
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	MessageCount    int       `json:"message_count"`
	UnreadCount     int       `json:"unread_count"`
	Type            string    `json:"type"` // inbox, sent, drafts, trash, etc.
	Icon            string    `json:"icon"`
	AccountID       string    `json:"account_id"`
	LastSync        time.Time `json:"last_sync"`
}

// MessageInfo represents UI-friendly message information
type MessageInfo struct {
	ID              string         `json:"id"`
	Subject         string         `json:"subject"`
	From            AddressInfo    `json:"from"`
	To              []AddressInfo  `json:"to"`
	Date            time.Time      `json:"date"`
	Preview         string         `json:"preview"`
	Size            int64          `json:"size"`
	
	// Status flags
	IsRead          bool           `json:"is_read"`
	IsFlagged       bool           `json:"is_flagged"`
	HasAttachments  bool           `json:"has_attachments"`
	IsEncrypted     bool           `json:"is_encrypted"`
	IsSigned        bool           `json:"is_signed"`
	
	// Organization
	ThreadID        string         `json:"thread_id,omitempty"`
	ThreadCount     int            `json:"thread_count"`
	FolderName      string         `json:"folder_name"`
	AccountID       string         `json:"account_id"`
	
	// UI metadata
	DisplayDate     string         `json:"display_date"`
	SizeDisplay     string         `json:"size_display"`
	FromDisplay     string         `json:"from_display"`
	PreviewHTML     string         `json:"preview_html,omitempty"`
}

// AddressInfo represents a simplified email address for UI display
type AddressInfo struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
	Display string `json:"display"`
}

// ServiceUpdate represents real-time updates from the email service
type ServiceUpdate struct {
	Type        UpdateType      `json:"type"`
	AccountID   string          `json:"account_id,omitempty"`
	FolderName  string          `json:"folder_name,omitempty"`
	MessageID   string          `json:"message_id,omitempty"`
	Data        interface{}     `json:"data,omitempty"`
	Error       *ServiceError   `json:"error,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

// UpdateType represents the type of service update
type UpdateType string

const (
	UpdateTypeAccountAdded      UpdateType = "account_added"
	UpdateTypeAccountRemoved    UpdateType = "account_removed"
	UpdateTypeAccountConnected  UpdateType = "account_connected"
	UpdateTypeAccountError      UpdateType = "account_error"
	UpdateTypeFolderRefreshed   UpdateType = "folder_refreshed"
	UpdateTypeNewMessage        UpdateType = "new_message"
	UpdateTypeMessageRead       UpdateType = "message_read"
	UpdateTypeMessageFlagged    UpdateType = "message_flagged"
	UpdateTypeMessageDeleted    UpdateType = "message_deleted"
	UpdateTypeMessageMoved      UpdateType = "message_moved"
	UpdateTypeSearchResults     UpdateType = "search_results"
	UpdateTypeSyncStarted       UpdateType = "sync_started"
	UpdateTypeSyncCompleted     UpdateType = "sync_completed"
	UpdateTypeError             UpdateType = "error"
)

// ServiceError represents service-layer errors with user-friendly messages
type ServiceError struct {
	Type        ErrorType `json:"type"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	UserMessage string    `json:"user_message"`
	Retryable   bool      `json:"retryable"`
	Timestamp   time.Time `json:"timestamp"`
}

// ErrorType categorizes service errors for appropriate UI handling
type ErrorType string

const (
	ErrorTypeConnection     ErrorType = "connection"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeNetwork        ErrorType = "network"
	ErrorTypeConfiguration  ErrorType = "configuration"
	ErrorTypeOperation      ErrorType = "operation"
	ErrorTypeTimeout        ErrorType = "timeout"
)

// SyncStatus represents the current synchronization status
type SyncStatus string

const (
	SyncStatusIdle      SyncStatus = "idle"
	SyncStatusSyncing   SyncStatus = "syncing"
	SyncStatusError     SyncStatus = "error"
	SyncStatusOffline   SyncStatus = "offline"
)

// ConnectionInfo represents connection status for monitoring
type ConnectionInfo struct {
	AccountID       string        `json:"account_id"`
	Status          string        `json:"status"`
	Connected       bool          `json:"connected"`
	LastPing        time.Time     `json:"last_ping"`
	ResponseTime    time.Duration `json:"response_time"`
	ErrorCount      int           `json:"error_count"`
	LastError       *string       `json:"last_error,omitempty"`
}

// ServiceStatistics provides performance and usage metrics
type ServiceStatistics struct {
	// Connection stats
	ActiveConnections   int           `json:"active_connections"`
	TotalAccounts      int           `json:"total_accounts"`
	ConnectedAccounts  int           `json:"connected_accounts"`

	// Message stats
	TotalMessages      int           `json:"total_messages"`
	TotalUnread        int           `json:"total_unread"`
	MessagesSynced     int           `json:"messages_synced"`

	// Performance stats
	AverageResponseTime time.Duration `json:"average_response_time"`
	CacheHitRate       float64       `json:"cache_hit_rate"`
	ErrorRate          float64       `json:"error_rate"`

	// Timing
	Uptime             time.Duration `json:"uptime"`
	LastSync           time.Time     `json:"last_sync"`
}

// SearchQuery represents a search query with UI-friendly options
type SearchQuery struct {
	Query      string            `json:"query"`
	AccountID  string            `json:"account_id,omitempty"`
	FolderName string            `json:"folder_name,omitempty"`
	DateFrom   *time.Time        `json:"date_from,omitempty"`
	DateTo     *time.Time        `json:"date_to,omitempty"`
	From       string            `json:"from,omitempty"`
	To         string            `json:"to,omitempty"`
	Subject    string            `json:"subject,omitempty"`
	HasFlag    []string          `json:"has_flag,omitempty"`
	Limit      int               `json:"limit,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
}

// SearchResults represents search results with UI enhancements
type SearchResults struct {
	Messages    []MessageInfo     `json:"messages"`
	Total       int               `json:"total"`
	Query       string            `json:"query"`
	Highlights  map[string][]string `json:"highlights,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
	Took        time.Duration     `json:"took"`
	AccountID   string            `json:"account_id"`
}

// OutgoingMessage represents a message being composed for sending
type OutgoingMessage struct {
	From        AddressInfo       `json:"from"`
	To          []AddressInfo     `json:"to"`
	CC          []AddressInfo     `json:"cc,omitempty"`
	BCC         []AddressInfo     `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	BodyHTML    string            `json:"body_html,omitempty"`
	Attachments []AttachmentInfo  `json:"attachments,omitempty"`
	InReplyTo   string            `json:"in_reply_to,omitempty"`
	References  []string          `json:"references,omitempty"`
	Priority    string            `json:"priority,omitempty"`
	Encrypt     bool              `json:"encrypt"`
	Sign        bool              `json:"sign"`
	SaveDraft   bool              `json:"save_draft"`
}

// AttachmentInfo represents attachment information for the UI
type AttachmentInfo struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	SizeDisplay string `json:"size_display"`
	IsInline    bool   `json:"is_inline"`
	Downloaded  bool   `json:"downloaded"`
	LocalPath   string `json:"local_path,omitempty"`
}

// NewEmailService creates a new email service instance
func NewEmailService(
	pool email.ConnectionManager,
	config *config.Config,
	logger *log.Logger,
	authFactory *email.AuthProviderFactory,
	clientFactory email.ClientFactory,
) EmailService {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &EmailServiceImpl{
		pool:          pool,
		config:        config,
		logger:        logger,
		authFactory:   authFactory,
		clientFactory: clientFactory,
		state:         &ServiceState{
			Accounts:    make([]AccountInfo, 0),
			Folders:     make([]FolderInfo, 0),
			Messages:    make([]MessageInfo, 0),
			SyncStatus:  SyncStatusIdle,
			LastUpdate:  time.Now(),
		},
		updateChan:    make(chan ServiceUpdate, 100), // Buffered channel for updates
		subscribers:   make(map[string]chan<- ServiceUpdate),
		messageCache:  make(map[string]*MessageInfo),
		folderCache:   make(map[string][]FolderInfo),
		cacheExpiry:   15 * time.Minute, // Cache TTL
		ctx:           ctx,
		cancel:        cancel,
		running:       false,
	}
}

// GetUpdateChannel returns the channel for receiving real-time updates
func (s *EmailServiceImpl) GetUpdateChannel() <-chan ServiceUpdate {
	return s.updateChan
}

// GetState returns a copy of the current service state
func (s *EmailServiceImpl) GetState() *ServiceState {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
	
	// Return a deep copy to prevent external modification
	stateCopy := *s.state
	return &stateCopy
}

// IsRunning returns whether the service is currently running
func (s *EmailServiceImpl) IsRunning() bool {
	s.runningMutex.RLock()
	defer s.runningMutex.RUnlock()
	return s.running
}

// Helper method to emit service updates
func (s *EmailServiceImpl) emitUpdate(update ServiceUpdate) {
	update.Timestamp = time.Now()
	
	// Send to main update channel (non-blocking)
	select {
	case s.updateChan <- update:
	default:
		// Channel is full, log warning
		s.logger.Warn("Service update channel full, dropping update", "type", update.Type)
	}
	
	// Send to individual subscribers
	s.subMutex.RLock()
	for _, subscriber := range s.subscribers {
		select {
		case subscriber <- update:
		default:
			// Subscriber channel is full, skip
		}
	}
	s.subMutex.RUnlock()
}

// Helper method to update service state safely
func (s *EmailServiceImpl) updateState(updateFunc func(*ServiceState)) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	
	updateFunc(s.state)
	s.state.LastUpdate = time.Now()
}