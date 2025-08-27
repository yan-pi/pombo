// Package email provides core email client interfaces and data structures
package email

import (
	"context"
	"io"
	"time"
)

// EmailClient provides unified email operations across protocols
type EmailClient interface {
	// Connection management
	Connect(ctx context.Context, account *AccountConfig) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	Ping(ctx context.Context) error

	// Folder operations
	GetFolders(ctx context.Context) ([]*Folder, error)
	SelectFolder(ctx context.Context, name string) (*FolderStatus, error)
	CreateFolder(ctx context.Context, name string) error
	DeleteFolder(ctx context.Context, name string) error

	// Message operations
	GetMessages(ctx context.Context, folderName string, criteria *SearchCriteria) ([]*Message, error)
	GetMessage(ctx context.Context, messageID string) (*Message, error)
	SendMessage(ctx context.Context, msg *OutgoingMessage) error
	
	// Message management
	MarkRead(ctx context.Context, messageIDs []string) error
	MarkUnread(ctx context.Context, messageIDs []string) error
	SetFlag(ctx context.Context, messageIDs []string, flag string) error
	RemoveFlag(ctx context.Context, messageIDs []string, flag string) error
	MoveMessage(ctx context.Context, messageID, targetFolder string) error
	DeleteMessage(ctx context.Context, messageID string) error

	// Real-time updates
	Subscribe(ctx context.Context, updates chan<- *EmailUpdate) error
	Unsubscribe(ctx context.Context) error

	// Account information
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)
}

// IMAPClient handles IMAP protocol operations
type IMAPClient interface {
	// Connection and authentication
	Connect(ctx context.Context, config *IMAPConfig) error
	Authenticate(ctx context.Context, auth AuthProvider) error
	Logout(ctx context.Context) error
	
	// Capability and server information
	Capability(ctx context.Context) ([]string, error)
	ServerInfo() *ServerInfo
	
	// Folder management
	List(ctx context.Context, ref, name string) ([]*Folder, error)
	Subscribe(ctx context.Context, name string) error
	Unsubscribe(ctx context.Context, name string) error
	Create(ctx context.Context, name string) error
	Delete(ctx context.Context, name string) error
	Rename(ctx context.Context, oldName, newName string) error
	
	// Message operations
	Select(ctx context.Context, name string) (*FolderStatus, error)
	Examine(ctx context.Context, name string) (*FolderStatus, error)
	Search(ctx context.Context, criteria *SearchCriteria) ([]uint32, error)
	Fetch(ctx context.Context, uids []uint32, items []string) ([]*Message, error)
	Store(ctx context.Context, uids []uint32, flags []string, action string) error
	Copy(ctx context.Context, uids []uint32, dest string) error
	Move(ctx context.Context, uids []uint32, dest string) error
	Expunge(ctx context.Context) error
	
	// Real-time updates
	Idle(ctx context.Context, updates chan<- *EmailUpdate) error
	
	// Connection status
	State() ConnectionState
	Close() error
}

// SMTPClient handles message sending
type SMTPClient interface {
	// Connection and authentication
	Connect(ctx context.Context, config *SMTPConfig) error
	Authenticate(ctx context.Context, auth AuthProvider) error
	Quit() error
	
	// Server information
	Extension(name string) (bool, string)
	ServerName() string
	
	// Message sending
	Mail(ctx context.Context, from string, opts *MailOptions) error
	Rcpt(ctx context.Context, to string) error
	Data(ctx context.Context) (io.WriteCloser, error)
	SendMail(ctx context.Context, from string, to []string, msg []byte) error
	
	// Advanced operations
	Reset() error
	Noop() error
	Verify(addr string) error
	
	// Connection status
	Close() error
}

// AuthProvider abstracts authentication methods
type AuthProvider interface {
	// Authentication
	GetCredentials(ctx context.Context) (*Credentials, error)
	RefreshIfNeeded(ctx context.Context) error
	Type() AuthType
	
	// Token management (for OAuth2)
	GetToken(ctx context.Context) (*OAuthToken, error)
	RefreshToken(ctx context.Context) (*OAuthToken, error)
	
	// Validation
	IsValid(ctx context.Context) bool
	ExpiresAt() *time.Time
}

// ConnectionManager handles connection lifecycle and pooling
type ConnectionManager interface {
	// Connection management
	GetConnection(ctx context.Context, accountID string) (EmailClient, error)
	ReleaseConnection(accountID string) error
	CloseConnection(accountID string) error
	CloseAll() error
	
	// Health monitoring
	HealthCheck(ctx context.Context, accountID string) error
	GetConnectionStatus(accountID string) *ConnectionStatus
	
	// Pool management
	GetPoolStats() *PoolStats
	SetPoolConfig(config *PoolConfig) error
}

// MessageProcessor handles email parsing and processing
type MessageProcessor interface {
	// Message processing
	Process(ctx context.Context, raw []byte) (*Message, error)
	ParseHeaders(headers map[string][]string) (*MessageHeaders, error)
	ParseBody(contentType string, body io.Reader) (*MessageBody, error)
	ParseAttachments(msg *Message) ([]*Attachment, error)
	
	// Threading
	BuildThreads(messages []*Message) ([]*Thread, error)
	UpdateThread(thread *Thread, newMessage *Message) error
	
	// Validation
	ValidateMessage(msg *Message) error
	SanitizeContent(content string, contentType string) (string, error)
}

// SearchEngine provides email search capabilities
type SearchEngine interface {
	// Indexing
	IndexMessage(ctx context.Context, msg *Message) error
	RemoveMessage(ctx context.Context, messageID string) error
	UpdateIndex(ctx context.Context) error
	
	// Searching
	Search(ctx context.Context, query *SearchQuery) (*SearchResults, error)
	SuggestTerms(ctx context.Context, prefix string) ([]string, error)
	
	// Management
	GetIndexStats() *IndexStats
	OptimizeIndex(ctx context.Context) error
	RebuildIndex(ctx context.Context) error
}

// CacheManager handles local email caching
type CacheManager interface {
	// Message caching
	CacheMessage(ctx context.Context, msg *Message) error
	GetCachedMessage(ctx context.Context, messageID string) (*Message, error)
	RemoveCachedMessage(ctx context.Context, messageID string) error
	
	// Folder caching
	CacheFolder(ctx context.Context, folder *Folder) error
	GetCachedFolder(ctx context.Context, folderName string) (*Folder, error)
	
	// Cache management
	InvalidateCache(ctx context.Context, pattern string) error
	GetCacheStats() *CacheStats
	CleanupCache(ctx context.Context) error
}

// CredentialStore provides secure credential storage
type CredentialStore interface {
	// Credential management
	Store(ctx context.Context, accountID string, creds *Credentials) error
	Retrieve(ctx context.Context, accountID string) (*Credentials, error)
	Delete(ctx context.Context, accountID string) error
	List(ctx context.Context) ([]string, error)
	
	// Token management
	StoreToken(ctx context.Context, accountID string, token *OAuthToken) error
	RetrieveToken(ctx context.Context, accountID string) (*OAuthToken, error)
	DeleteToken(ctx context.Context, accountID string) error
	
	// Validation
	IsAvailable(ctx context.Context) bool
	TestAccess(ctx context.Context) error
}