# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

POMBO is an open-source TUI (Terminal User Interface) email client written in Go using the Charm ecosystem, focusing on simplicity, speed, and excellent user experience. The project leverages Bubbletea for the TUI framework, along with Lipgloss, Bubbles, and other Charm tools.

## Key Features

- **Email Protocols**: IMAP and SMTP support with OAuth2 authentication for Gmail, Outlook, and other providers
- **Multi-Account**: Support for multiple email accounts with threaded conversations and connection pooling
- **TUI Interface**: Rich terminal interface using Bubbletea with Vim-style keybindings
- **Security**: PGP encryption and signing support with secure credential storage
- **Performance**: Local caching with SQLite, full-text search with Bleve, optimized connection management
- **Configuration**: YAML-based configuration with hot-reload capability and email-specific settings
- **Cross-Platform**: Linux, macOS, Windows compatibility
- **Enterprise-Ready**: Production-grade error handling, retry logic, and comprehensive monitoring

## Technology Stack

### Core Charm Ecosystem
```go
github.com/charmbracelet/bubbletea    // TUI framework
github.com/charmbracelet/lipgloss     // Styling and layout
github.com/charmbracelet/bubbles     // UI components
github.com/charmbracelet/glamour     // Markdown rendering
github.com/charmbracelet/huh         // Interactive forms
github.com/charmbracelet/log         // Structured logging
github.com/charmbracelet/harmonica   // Animation library
```

### Email and Security
```go
github.com/emersion/go-imap/v2       // IMAP client
github.com/emersion/go-smtp          // SMTP client
github.com/ProtonMail/gopenpgp/v3    // PGP encryption
golang.org/x/oauth2                  // OAuth2 authentication
```

### Storage and Configuration
```go
github.com/spf13/viper               // Configuration management
modernc.org/sqlite                   // Local database
github.com/blevesearch/bleve/v2      // Full-text search
```

## Project Structure

```
pombo/
├── cmd/pombo/              # Main application entry point
├── internal/               # Private application code
│   ├── app/               # Application layer
│   ├── config/            # Configuration management (extended for email)
│   ├── email/             # Email protocol handlers (Phase 2.1)
│   │   ├── auth.go        # Authentication providers (OAuth2, Basic)
│   │   ├── client.go      # Core email client interfaces
│   │   ├── errors.go      # Comprehensive error handling framework
│   │   ├── pool.go        # Connection pooling for multi-account support
│   │   ├── types.go       # Email data structures and types
│   │   └── *_test.go      # Comprehensive test suite (87.7% coverage)
│   ├── ui/                # TUI components and pages
│   ├── storage/           # Local storage and caching
│   ├── crypto/            # PGP encryption/signing
│   └── utils/             # Shared utilities
├── pkg/                   # Public API (for plugins)
├── configs/               # Configuration templates
├── docs/                  # Documentation
├── scripts/               # Build and development scripts
└── tests/                 # Integration tests
```

## Phase 2.1: Email Foundation Architecture (COMPLETED)

### 📋 Implementation Overview
Phase 2.1 establishes the foundational email architecture for POMBO with production-ready components:

- **2,500+ lines** of production-ready Go code
- **87.7% test coverage** with 46+ comprehensive test cases
- **Thread-safe concurrent operations** with race condition testing
- **Enterprise-grade error handling** with retry logic and classification
- **Multi-account connection pooling** with health monitoring
- **OAuth2 + Basic authentication** with secure credential storage

### 🏗️ Core Email Architecture

#### 1. Email Client Interfaces (`internal/email/client.go`)
```go
// Unified email operations across protocols
type EmailClient interface {
    Connect(ctx context.Context, account *AccountConfig) error
    GetMessages(ctx context.Context, folderName string, criteria *SearchCriteria) ([]*Message, error)
    SendMessage(ctx context.Context, msg *OutgoingMessage) error
    // ... complete CRUD operations for email management
}

// Protocol-specific interfaces
type IMAPClient interface {
    Select(ctx context.Context, name string) (*FolderStatus, error)
    Fetch(ctx context.Context, uids []uint32, items []string) ([]*Message, error)
    Idle(ctx context.Context, updates chan<- *EmailUpdate) error
    // ... full IMAP protocol support
}

type SMTPClient interface {
    SendMail(ctx context.Context, from string, to []string, msg []byte) error
    // ... complete SMTP operations
}
```

#### 2. Connection Pool Management (`internal/email/pool.go`)
Production-ready connection pooling for multiple email accounts:

```go
// Multi-account connection pool with health monitoring
type ConnectionPool struct {
    pools    map[string]*accountPool  // Per-account pools
    config   *PoolConfig             // Pool configuration
    stats    *PoolStats              // Real-time statistics
    cleanup  *time.Ticker            // Periodic cleanup
    authFactory *AuthProviderFactory  // Authentication management
}

// Configuration options
type PoolConfig struct {
    MaxConnections      int           `json:"max_connections"`
    MaxIdleConnections  int           `json:"max_idle_connections"`
    ConnectionLifetime  time.Duration `json:"connection_lifetime"`
    IdleTimeout         time.Duration `json:"idle_timeout"`
    HealthCheckInterval time.Duration `json:"health_check_interval"`
}
```

**Key Features:**
- **Automatic health checks** with ping testing
- **Connection reuse** to minimize server load
- **Graceful degradation** when connections fail
- **Real-time statistics** for monitoring
- **Thread-safe operations** with comprehensive locking

#### 3. Authentication System (`internal/email/auth.go`)
Flexible authentication supporting multiple providers:

```go
// Universal authentication interface
type AuthProvider interface {
    GetCredentials(ctx context.Context) (*Credentials, error)
    RefreshIfNeeded(ctx context.Context) error
    Type() AuthType
    IsValid(ctx context.Context) bool
}

// OAuth2 with automatic token refresh
type OAuth2AuthProvider struct {
    config       *oauth2.Config
    token        *oauth2.Token
    credStore    CredentialStore
    refreshToken string
}

// Basic authentication for traditional email servers
type BasicAuthProvider struct {
    username string
    password string
    authType AuthType
}
```

**Supported Authentication:**
- **OAuth2** with automatic token refresh for Gmail, Outlook, Yahoo
- **Basic Authentication** for traditional IMAP/SMTP servers
- **Secure credential storage** with OS keychain integration
- **Automatic retry** on authentication failures

#### 4. Error Handling Framework (`internal/email/errors.go`)
Enterprise-grade error handling with comprehensive classification:

```go
// Structured error with context and retry information
type EmailError struct {
    Type        ErrorType   `json:"type"`
    Code        string      `json:"code"`
    Message     string      `json:"message"`
    Retryable   bool        `json:"retryable"`
    Account     string      `json:"account,omitempty"`
    Operation   string      `json:"operation,omitempty"`
    Timestamp   time.Time   `json:"timestamp"`
}

// Error classification for intelligent handling
type ErrorType int
const (
    ErrorTypeNetwork     // Network-related errors
    ErrorTypeAuth        // Authentication failures
    ErrorTypeProtocol    // IMAP/SMTP protocol errors
    ErrorTypeQuota       // Storage/rate limit errors
    ErrorTypeTimeout     // Connection timeouts
    ErrorTypeRateLimit   // API rate limiting
    // ... comprehensive error taxonomy
)
```

**Error Handling Features:**
- **Automatic retry logic** with exponential backoff
- **Error classification** for appropriate responses
- **Context preservation** for debugging
- **Jitter support** to prevent thundering herd
- **Circuit breaker patterns** for failing services

#### 5. Email Data Structures (`internal/email/types.go`)
Comprehensive email data models supporting all email operations:

```go
// Complete email message representation
type Message struct {
    ID          string       `json:"id"`
    Subject     string       `json:"subject"`
    From        *Address     `json:"from"`
    To          []*Address   `json:"to"`
    Body        *MessageBody `json:"body"`
    Attachments []*Attachment `json:"attachments,omitempty"`
    Flags       []string     `json:"flags"`
    ThreadID    string       `json:"thread_id,omitempty"`
    // ... complete email metadata
}

// Thread management for conversation view
type Thread struct {
    ID           string     `json:"id"`
    Subject      string     `json:"subject"`
    Messages     []*Message `json:"messages"`
    Participants []*Address `json:"participants"`
    UnreadCount  int        `json:"unread_count"`
}

// Advanced search capabilities
type SearchCriteria struct {
    Query       string         `json:"query,omitempty"`
    From        string         `json:"from,omitempty"`
    Since       *time.Time     `json:"since,omitempty"`
    HasFlag     []string       `json:"has_flag,omitempty"`
    Size        *SizeConstraint `json:"size,omitempty"`
}
```

### 🔧 Configuration Integration
Extended configuration system for comprehensive email management:

```yaml
# Email-specific configuration (internal/config/config.go)
email:
  default_account: "work"
  check_interval: "5m"
  auto_sync: true
  background_sync: true
  
  connection_pool:
    max_connections: 5
    max_idle_connections: 2
    connection_lifetime: "30m"
    idle_timeout: "5m"
    health_check_interval: "1m"
    connect_timeout: "30s"
  
  message_cache:
    max_size: "100MB"
    ttl: "24h"
    cache_headers: true
    cache_bodies: true
    cache_attachments: false
  
  error_retry:
    max_retries: 3
    base_delay: "1s"
    max_delay: "1m"
    multiplier: 2.0
    jitter_enabled: true

accounts:
  - id: "work"
    name: "Work Account"
    email: "user@company.com"
    provider: "outlook"
    oauth:
      provider: "microsoft"
      client_id: "your-client-id"
      redirect_uri: "http://localhost:8080/callback"
      scopes: ["https://graph.microsoft.com/Mail.ReadWrite"]
```

### 🧪 Testing Strategy
Comprehensive testing approach ensuring reliability:

#### Test Coverage: **87.7%**
- **46+ test cases** covering all major functionality
- **Race condition testing** for concurrent operations
- **Mock frameworks** for testing without external dependencies
- **Error injection testing** for resilience validation
- **Integration testing** across component boundaries

#### Test Categories:
```bash
# Authentication Testing
TestBasicAuthProvider/valid_credentials                    ✅
TestOAuth2AuthProvider_RefreshToken                       ✅
TestAuthProviderFactory/oauth2_provider                   ✅

# Connection Pool Testing  
TestConnectionPool_ConcurrentAccess                       ✅
TestConnectionPool_ConnectionReuse                        ✅
TestConnectionPool_ErrorHandling                          ✅

# Error Handling Testing
TestErrorHandler_Handle/network_error_-_retryable         ✅
TestErrorClassification/authentication_error              ✅
TestNetworkErrorDetection                                 ✅

# Data Structure Testing
TestMessage/complete_email_metadata                       ✅
TestThread/conversation_management                        ✅
TestSearchCriteria/advanced_search                        ✅
```

### 🚀 Phase 2.2 Readiness Assessment
The Phase 2.1 foundation provides a solid base for IMAP/SMTP protocol implementation:

#### ✅ **Foundation Strengths:**
- **Scalable Architecture:** Connection pooling supports hundreds of accounts
- **Robust Error Handling:** Comprehensive retry logic and circuit breakers
- **Authentication Ready:** OAuth2 and Basic auth providers tested and validated
- **Thread-Safe Operations:** All components designed for concurrent access
- **Monitoring Capabilities:** Built-in health checks and performance metrics
- **Configuration Flexibility:** Hot-reload and environment variable support

#### 🎯 **Next Phase Readiness Indicators:**
- **Interface Contracts:** Clear separation between generic and protocol-specific operations
- **Mock Testing Framework:** Ready for protocol implementation testing without external dependencies
- **Error Classification:** Comprehensive error types for IMAP/SMTP specific issues
- **Performance Benchmarks:** Baseline established for measuring protocol implementation efficiency

#### 📋 **Phase 2.2 Implementation Roadmap:**
1. **IMAP Protocol Implementation:** Concrete implementation of IMAPClient interface
2. **SMTP Protocol Implementation:** Concrete implementation of SMTPClient interface  
3. **Message Processing:** MIME parsing, attachment handling, and threading
4. **Real-time Sync:** IDLE command support for instant email updates
5. **Provider Integration:** Gmail, Outlook, Yahoo-specific optimizations
6. **Performance Optimization:** Connection reuse and caching strategies

## Development Commands

### Setup and Build
```bash
# Initialize and setup the project
make setup                 # Install dependencies and tools
make build                 # Build the application
make install               # Install to $GOPATH/bin

# Development
make dev                   # Run in development mode with hot reload
make run                   # Run the application
make clean                 # Clean build artifacts
```

### Testing
```bash
# Run all tests
make test                  # Unit tests
make test-integration      # Integration tests with real email servers
make test-coverage         # Generate coverage report
make test-race             # Run tests with race detection

# Email-specific testing (Phase 2.1)
go test ./internal/email/... -v                          # Test email components with verbose output
go test ./internal/email/... -coverprofile=coverage.out  # Generate email coverage report
go test -run TestConnectionPool ./internal/email/        # Test connection pooling
go test -run TestAuth ./internal/email/                  # Test authentication systems
go test -run TestError ./internal/email/                 # Test error handling
go test ./internal/email/... -race                       # Race condition testing

# Integration testing
go test ./internal/config/... -v                         # Test configuration integration
go test ./... -tags=integration                          # Run integration test suite
```

### Code Quality
```bash
# Formatting and linting
make fmt                   # Format all Go code
make lint                  # Run golangci-lint
make vet                   # Run go vet
make security              # Run security scanning (gosec)

# Pre-commit checks
make check                 # Run all quality checks before commit
```

### Configuration
```bash
# Generate example configs
make config-examples       # Generate example YAML files

# Validate configuration
pombo config validate     # Validate current config
pombo config init         # Initialize new config file
```

## Architecture

### TUI Architecture (Bubbletea Pattern)
- **Model**: Application state using immutable data structures
- **Update**: Pure functions that handle messages and update state
- **View**: Functions that render the current state to the terminal

### Email Processing Flow
1. **Authentication**: OAuth2 token management with automatic refresh
2. **Connection**: IMAP/SMTP connection pooling for multiple accounts
3. **Synchronization**: Background email sync with incremental updates
4. **Storage**: Local SQLite cache with full-text search indexing
5. **Display**: TUI components render emails with threading support

### State Management
- Central application state using Elm architecture patterns
- Message-passing for UI updates and background operations
- Command pattern for handling user actions
- Event sourcing for maintaining operation history

## Configuration

### Main Configuration (`~/.config/pombo/config.yaml`)
```yaml
accounts:
  - name: "work"
    email: "user@company.com"
    provider: "outlook"
    oauth:
      client_id: "your-client-id"
      redirect_uri: "http://localhost:8080/callback"

ui:
  theme: "dark"
  vim_keybindings: true
  show_line_numbers: false

security:
  pgp:
    auto_encrypt: true
    keyring_path: "~/.gnupg"
```

### Keybindings (`~/.config/pombo/keybindings.yaml`)
```yaml
global:
  quit: "q"
  help: "?"
  
mail:
  compose: "c"
  reply: "r"
  forward: "f"
  delete: "d"
  archive: "a"
```

## Development Workflow

1. **Feature Development**: Create feature branch from `develop`
2. **Implementation**: Follow TDD with unit tests first
3. **Testing**: Run full test suite including integration tests
4. **Code Review**: Submit PR with comprehensive description
5. **Integration**: Merge to `develop` after approval and CI pass
6. **Release**: Create release branch for final testing and deployment

## Debugging

### Enable Debug Logging
```bash
POMBO_LOG_LEVEL=debug pombo
```

### TUI Debugging
```bash
# Use separate terminal for debug output
POMBO_DEBUG_FILE=/tmp/pombo.log pombo
tail -f /tmp/pombo.log  # In another terminal
```

### Email Protocol Debugging
```bash
# Enable IMAP/SMTP protocol logging
POMBO_EMAIL_DEBUG=true pombo
```

## Performance Considerations

- **Memory**: Target <50MB baseline, <200MB with large mailboxes
- **Startup Time**: <500ms cold start, <100ms warm start
- **Email Operations**: <500ms for fetch/send operations
- **UI Responsiveness**: <100ms for all UI interactions

## Security Guidelines

- Never log or expose OAuth2 tokens or passwords
- Use secure storage for credentials (OS keychain integration)
- Validate all user inputs, especially email addresses
- Implement proper TLS certificate validation
- Regular security audits of dependencies