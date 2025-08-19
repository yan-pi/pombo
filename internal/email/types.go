package email

import (
	"time"
)

// Message represents an email message with all its components
type Message struct {
	// Basic identification
	ID          string    `json:"id"`
	UID         uint32    `json:"uid"`
	MessageID   string    `json:"message_id"`
	InReplyTo   string    `json:"in_reply_to,omitempty"`
	References  []string  `json:"references,omitempty"`
	
	// Headers and metadata
	Subject     string    `json:"subject"`
	From        *Address  `json:"from"`
	To          []*Address `json:"to"`
	CC          []*Address `json:"cc,omitempty"`
	BCC         []*Address `json:"bcc,omitempty"`
	Date        time.Time `json:"date"`
	
	// Content
	Body        *MessageBody `json:"body"`
	Attachments []*Attachment `json:"attachments,omitempty"`
	Headers     map[string][]string `json:"headers"`
	
	// Message status and flags
	Flags       []string  `json:"flags"`
	IsRead      bool      `json:"is_read"`
	IsFlagged   bool      `json:"is_flagged"`
	IsDraft     bool      `json:"is_draft"`
	IsAnswered  bool      `json:"is_answered"`
	IsDeleted   bool      `json:"is_deleted"`
	
	// Organization
	ThreadID    string    `json:"thread_id,omitempty"`
	FolderName  string    `json:"folder_name"`
	AccountID   string    `json:"account_id"`
	
	// Metadata
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MessageHeaders represents parsed email headers
type MessageHeaders struct {
	From         *Address    `json:"from"`
	To           []*Address  `json:"to"`
	CC           []*Address  `json:"cc,omitempty"`
	BCC          []*Address  `json:"bcc,omitempty"`
	Subject      string      `json:"subject"`
	Date         time.Time   `json:"date"`
	MessageID    string      `json:"message_id"`
	InReplyTo    string      `json:"in_reply_to,omitempty"`
	References   []string    `json:"references,omitempty"`
	ContentType  string      `json:"content_type"`
	Custom       map[string][]string `json:"custom,omitempty"`
}

// MessageBody represents the content of an email message
type MessageBody struct {
	Text        string      `json:"text,omitempty"`
	HTML        string      `json:"html,omitempty"`
	Parts       []*BodyPart `json:"parts,omitempty"`
	ContentType string      `json:"content_type"`
	Charset     string      `json:"charset"`
}

// BodyPart represents a part of a multipart message
type BodyPart struct {
	ContentType string            `json:"content_type"`
	Content     string            `json:"content"`
	Headers     map[string]string `json:"headers,omitempty"`
	Filename    string            `json:"filename,omitempty"`
	Size        int64             `json:"size"`
}

// Address represents an email address with optional display name
type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

// Attachment represents a file attachment
type Attachment struct {
	ID          string            `json:"id"`
	Filename    string            `json:"filename"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Content     []byte            `json:"content,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	IsInline    bool              `json:"is_inline"`
	CID         string            `json:"cid,omitempty"` // Content-ID for inline attachments
}

// Folder represents an email folder/mailbox
type Folder struct {
	Name         string   `json:"name"`
	FullName     string   `json:"full_name"`
	Delimiter    string   `json:"delimiter"`
	Attributes   []string `json:"attributes"`
	
	// Counts
	MessageCount int      `json:"message_count"`
	UnseenCount  int      `json:"unseen_count"`
	RecentCount  int      `json:"recent_count"`
	
	// Status
	UIDNext      uint32   `json:"uid_next"`
	UIDValidity  uint32   `json:"uid_validity"`
	
	// Metadata
	AccountID    string   `json:"account_id"`
	LastSync     time.Time `json:"last_sync"`
	
	// Hierarchy
	Parent       string   `json:"parent,omitempty"`
	Children     []string `json:"children,omitempty"`
	IsSubscribed bool     `json:"is_subscribed"`
}

// FolderStatus represents the status of a selected folder
type FolderStatus struct {
	Name        string    `json:"name"`
	Messages    uint32    `json:"messages"`
	Recent      uint32    `json:"recent"`
	Unseen      uint32    `json:"unseen"`
	UIDNext     uint32    `json:"uid_next"`
	UIDValidity uint32    `json:"uid_validity"`
	ReadOnly    bool      `json:"read_only"`
	Flags       []string  `json:"flags"`
	PermanentFlags []string `json:"permanent_flags"`
}

// OutgoingMessage represents a message being composed/sent
type OutgoingMessage struct {
	From        *Address            `json:"from"`
	To          []*Address          `json:"to"`
	CC          []*Address          `json:"cc,omitempty"`
	BCC         []*Address          `json:"bcc,omitempty"`
	Subject     string              `json:"subject"`
	Body        string              `json:"body"`
	BodyHTML    string              `json:"body_html,omitempty"`
	Attachments []*Attachment       `json:"attachments,omitempty"`
	Headers     map[string]string   `json:"headers,omitempty"`
	
	// Message options
	InReplyTo   string              `json:"in_reply_to,omitempty"`
	References  []string            `json:"references,omitempty"`
	Priority    MessagePriority     `json:"priority,omitempty"`
	
	// Security
	Encrypt     bool                `json:"encrypt"`
	Sign        bool                `json:"sign"`
	
	// Metadata
	DraftID     string              `json:"draft_id,omitempty"`
	AccountID   string              `json:"account_id"`
}

// Thread represents a conversation thread
type Thread struct {
	ID          string    `json:"id"`
	Subject     string    `json:"subject"`
	Messages    []*Message `json:"messages"`
	Participants []*Address `json:"participants"`
	LastMessage *Message  `json:"last_message"`
	MessageCount int      `json:"message_count"`
	UnreadCount int       `json:"unread_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	FolderName  string    `json:"folder_name"`
	AccountID   string    `json:"account_id"`
}

// SearchCriteria represents search parameters
type SearchCriteria struct {
	Query       string              `json:"query,omitempty"`
	From        string              `json:"from,omitempty"`
	To          string              `json:"to,omitempty"`
	Subject     string              `json:"subject,omitempty"`
	Body        string              `json:"body,omitempty"`
	Since       *time.Time          `json:"since,omitempty"`
	Before      *time.Time          `json:"before,omitempty"`
	HasFlag     []string            `json:"has_flag,omitempty"`
	NotFlag     []string            `json:"not_flag,omitempty"`
	Size        *SizeConstraint     `json:"size,omitempty"`
	Folder      string              `json:"folder,omitempty"`
	Limit       int                 `json:"limit,omitempty"`
	Offset      int                 `json:"offset,omitempty"`
}

// SearchQuery represents a full-text search query
type SearchQuery struct {
	Query       string              `json:"query"`
	Fields      []string            `json:"fields,omitempty"`
	Fuzzy       bool                `json:"fuzzy"`
	Highlight   bool                `json:"highlight"`
	SortBy      string              `json:"sort_by,omitempty"`
	SortOrder   SortOrder           `json:"sort_order,omitempty"`
	Limit       int                 `json:"limit,omitempty"`
	Offset      int                 `json:"offset,omitempty"`
	AccountID   string              `json:"account_id,omitempty"`
	FolderName  string              `json:"folder_name,omitempty"`
}

// SearchResults represents search results
type SearchResults struct {
	Messages    []*Message          `json:"messages"`
	Total       int                 `json:"total"`
	Query       string              `json:"query"`
	Highlights  map[string][]string `json:"highlights,omitempty"`
	Suggestions []string            `json:"suggestions,omitempty"`
	Took        time.Duration       `json:"took"`
}

// SizeConstraint represents size-based search criteria
type SizeConstraint struct {
	Operator SizeOperator `json:"operator"`
	Size     int64        `json:"size"`
}

// EmailUpdate represents real-time email updates
type EmailUpdate struct {
	Type        UpdateType     `json:"type"`
	AccountID   string         `json:"account_id"`
	FolderName  string         `json:"folder_name"`
	Message     *Message       `json:"message,omitempty"`
	Messages    []*Message     `json:"messages,omitempty"`
	Folder      *Folder        `json:"folder,omitempty"`
	Error       error          `json:"error,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
}

// AccountConfig represents email account configuration
type AccountConfig struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	Provider    string         `json:"provider"`
	IMAP        *IMAPConfig    `json:"imap"`
	SMTP        *SMTPConfig    `json:"smtp"`
	OAuth       *OAuthConfig   `json:"oauth,omitempty"`
	Credentials *Credentials   `json:"credentials,omitempty"`
	Settings    *AccountSettings `json:"settings,omitempty"`
}

// IMAPConfig holds IMAP server configuration
type IMAPConfig struct {
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	TLS         bool          `json:"tls"`
	StartTLS    bool          `json:"starttls"`
	Username    string        `json:"username"`
	Timeout     time.Duration `json:"timeout"`
	KeepAlive   time.Duration `json:"keepalive"`
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	TLS         bool          `json:"tls"`
	StartTLS    bool          `json:"starttls"`
	Username    string        `json:"username"`
	Timeout     time.Duration `json:"timeout"`
}

// OAuthConfig holds OAuth2 configuration
type OAuthConfig struct {
	Provider     string   `json:"provider"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	Scopes       []string `json:"scopes"`
	AuthURL      string   `json:"auth_url,omitempty"`
	TokenURL     string   `json:"token_url,omitempty"`
}

// Credentials represents authentication credentials
type Credentials struct {
	Type        AuthType      `json:"type"`
	Username    string        `json:"username,omitempty"`
	Password    string        `json:"password,omitempty"`
	Token       *OAuthToken   `json:"token,omitempty"`
	ExpiresAt   *time.Time    `json:"expires_at,omitempty"`
}

// OAuthToken represents an OAuth2 token
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope,omitempty"`
}

// AccountSettings represents account-specific settings
type AccountSettings struct {
	Signature        string        `json:"signature,omitempty"`
	AutoBCC          []string      `json:"auto_bcc,omitempty"`
	SyncInterval     time.Duration `json:"sync_interval"`
	MaxSyncMessages  int           `json:"max_sync_messages"`
	ComposeFormat    string        `json:"compose_format"` // "text" or "html"
	AutoMarkRead     bool          `json:"auto_mark_read"`
	DownloadAttachments bool       `json:"download_attachments"`
}

// AccountInfo represents account information
type AccountInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Provider    string    `json:"provider"`
	Status      string    `json:"status"`
	LastSync    time.Time `json:"last_sync"`
	TotalMessages int     `json:"total_messages"`
	UnreadMessages int    `json:"unread_messages"`
	QuotaUsed   int64     `json:"quota_used,omitempty"`
	QuotaTotal  int64     `json:"quota_total,omitempty"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name         string    `json:"name"`
	Version      string    `json:"version,omitempty"`
	Capabilities []string  `json:"capabilities"`
	TLSVersion   string    `json:"tls_version,omitempty"`
}

// ConnectionStatus represents connection status
type ConnectionStatus struct {
	AccountID    string    `json:"account_id"`
	Connected    bool      `json:"connected"`
	LastPing     time.Time `json:"last_ping"`
	LastError    error     `json:"last_error,omitempty"`
	ConnectedAt  time.Time `json:"connected_at"`
	ReconnectCount int     `json:"reconnect_count"`
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	ActiveConnections int           `json:"active_connections"`
	IdleConnections   int           `json:"idle_connections"`
	TotalConnections  int           `json:"total_connections"`
	MaxConnections    int           `json:"max_connections"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64       `json:"error_rate"`
}

// PoolConfig represents connection pool configuration
type PoolConfig struct {
	MaxConnections     int           `json:"max_connections"`
	MaxIdleConnections int           `json:"max_idle_connections"`
	ConnectionLifetime time.Duration `json:"connection_lifetime"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	ConnectTimeout     time.Duration `json:"connect_timeout"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalSize     int64         `json:"total_size"`
	ItemCount     int           `json:"item_count"`
	HitRate       float64       `json:"hit_rate"`
	MissRate      float64       `json:"miss_rate"`
	EvictionCount int           `json:"eviction_count"`
	MaxSize       int64         `json:"max_size"`
}

// IndexStats represents search index statistics
type IndexStats struct {
	DocumentCount int           `json:"document_count"`
	IndexSize     int64         `json:"index_size"`
	LastUpdate    time.Time     `json:"last_update"`
	QueryCount    int           `json:"query_count"`
	AverageQueryTime time.Duration `json:"average_query_time"`
}

// MailOptions represents options for sending mail
type MailOptions struct {
	Size        int64         `json:"size,omitempty"`
	Body        string        `json:"body,omitempty"`
	UTF8        bool          `json:"utf8"`
	RequireTLS  bool          `json:"require_tls"`
	Auth        AuthMechanism `json:"auth,omitempty"`
}

// Enums and constants

// ConnectionState represents the state of a connection
type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected"
	StateConnected    ConnectionState = "connected"
	StateAuthenticated ConnectionState = "authenticated"
	StateSelected     ConnectionState = "selected"
	StateIdle         ConnectionState = "idle"
	StateLogout       ConnectionState = "logout"
)

// AuthType represents authentication types
type AuthType string

const (
	AuthTypePassword AuthType = "password"
	AuthTypeOAuth2   AuthType = "oauth2"
	AuthTypeOAuth1   AuthType = "oauth1"
	AuthTypeAPIKey   AuthType = "apikey"
)

// MessagePriority represents message priority levels
type MessagePriority string

const (
	PriorityLow    MessagePriority = "low"
	PriorityNormal MessagePriority = "normal"
	PriorityHigh   MessagePriority = "high"
	PriorityUrgent MessagePriority = "urgent"
)

// UpdateType represents the type of email update
type UpdateType string

const (
	UpdateTypeNewMessage    UpdateType = "new_message"
	UpdateTypeMessageFlags UpdateType = "message_flags"
	UpdateTypeMessageMove  UpdateType = "message_move"
	UpdateTypeMessageDelete UpdateType = "message_delete"
	UpdateTypeFolderCreate UpdateType = "folder_create"
	UpdateTypeFolderDelete UpdateType = "folder_delete"
	UpdateTypeFolderRename UpdateType = "folder_rename"
	UpdateTypeConnection   UpdateType = "connection"
	UpdateTypeError        UpdateType = "error"
)

// SizeOperator represents size comparison operators
type SizeOperator string

const (
	SizeGreaterThan SizeOperator = "gt"
	SizeLessThan    SizeOperator = "lt"
	SizeEquals      SizeOperator = "eq"
)

// SortOrder represents sort order for search results
type SortOrder string

const (
	SortAscending  SortOrder = "asc"
	SortDescending SortOrder = "desc"
)

// AuthMechanism represents SMTP authentication mechanisms
type AuthMechanism string

const (
	AuthPlain     AuthMechanism = "PLAIN"
	AuthLogin     AuthMechanism = "LOGIN"
	AuthCRAMMD5   AuthMechanism = "CRAM-MD5"
	AuthOAuth2    AuthMechanism = "OAUTHBEARER"
	AuthXOAuth2   AuthMechanism = "XOAUTH2"
)

// Common email flags
const (
	FlagSeen     = "\\Seen"
	FlagAnswered = "\\Answered"
	FlagFlagged  = "\\Flagged"
	FlagDeleted  = "\\Deleted"
	FlagDraft    = "\\Draft"
	FlagRecent   = "\\Recent"
)

// Common IMAP folder names
const (
	FolderInbox   = "INBOX"
	FolderSent    = "Sent"
	FolderDrafts  = "Drafts"
	FolderTrash   = "Trash"
	FolderSpam    = "Spam"
	FolderArchive = "Archive"
)