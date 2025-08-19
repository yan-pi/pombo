# POMBO Architecture

This document describes the high-level architecture and design decisions for POMBO, a TUI email client built with the Charm ecosystem.

## Overview

POMBO follows a layered architecture pattern with clear separation of concerns:

- **UI Layer**: Bubbletea-based TUI components
- **Application Layer**: Business logic and state management
- **Email Layer**: IMAP/SMTP protocol handlers
- **Storage Layer**: Local caching and data persistence
- **Security Layer**: PGP encryption and OAuth2 authentication

## Technology Stack

### Core Framework
- **Bubbletea**: Elm-inspired TUI framework for reactive interfaces
- **Lipgloss**: CSS-like styling for terminal applications
- **Bubbles**: Pre-built UI components (lists, inputs, viewports)
- **Glamour**: Markdown rendering for email content

### Email Protocols
- **go-imap/v2**: Modern IMAP client implementation
- **go-smtp**: SMTP client for sending emails
- **OAuth2**: Secure authentication for major email providers

### Data & Storage
- **SQLite**: Local email caching and metadata storage
- **Bleve**: Full-text search engine for email content
- **Viper**: Configuration management with YAML support

## Architecture Patterns

### Model-View-Update (Elm Architecture)
POMBO uses the Elm architecture pattern via Bubbletea:
- **Model**: Immutable application state
- **Update**: Pure functions that handle messages and return new state
- **View**: Functions that render the current state to the terminal

### Component-Based UI
The TUI is built using composable components:
- Each view is a self-contained Bubbletea model
- Components can be nested and reused
- Consistent styling through shared Lipgloss styles

### Event-Driven Communication
- Message passing between components
- Background operations using Bubbletea commands
- Reactive updates based on state changes

## Directory Structure

```
pombo/
├── cmd/pombo/              # Main application entry point
├── internal/               # Private application code
│   ├── app/               # Application initialization and lifecycle
│   ├── config/            # Configuration management
│   ├── email/             # Email protocol implementations
│   │   ├── imap/          # IMAP client and operations
│   │   ├── smtp/          # SMTP client and operations
│   │   └── oauth/         # OAuth2 authentication
│   ├── ui/                # TUI components and styling
│   │   ├── components/    # Reusable UI components
│   │   ├── pages/         # Full-page views
│   │   ├── styles/        # Lipgloss styling definitions
│   │   └── keybinds/      # Keyboard shortcuts and bindings
│   ├── storage/           # Data persistence layer
│   │   ├── cache/         # Email caching system
│   │   ├── search/        # Full-text search implementation
│   │   └── db/            # Database operations
│   ├── crypto/            # PGP encryption and signing
│   └── utils/             # Shared utilities and helpers
├── pkg/                   # Public API (for future plugin system)
├── configs/               # Configuration templates and examples
├── docs/                  # Documentation
└── tests/                 # Integration and end-to-end tests
```

## State Management

### Application State
The application maintains a central state structure containing:
- Current user interface state (active view, selections, etc.)
- Email account configurations and connection status
- Cached email data and metadata
- User preferences and settings

### State Updates
State changes follow the Elm architecture:
1. User interactions generate messages
2. Update functions process messages and return new state
3. View functions render the updated state
4. Background operations send messages back to update state

## Email Processing Pipeline

### 1. Authentication
- OAuth2 flow for supported providers (Gmail, Outlook, etc.)
- Secure token storage and automatic refresh
- Fallback to traditional username/password authentication

### 2. Connection Management
- Connection pooling for multiple accounts
- Automatic reconnection on network issues
- Idle connection monitoring for real-time updates

### 3. Message Synchronization
- Incremental sync to minimize bandwidth
- Background fetching of new messages
- Local caching for offline access

### 4. Content Processing
- HTML email rendering with Glamour
- Attachment handling and preview
- PGP encryption/decryption integration

## Performance Considerations

### Memory Management
- Lazy loading of email content
- LRU cache for frequently accessed messages
- Efficient data structures for large mailboxes

### UI Responsiveness
- Non-blocking operations using goroutines
- Progressive loading with visual feedback
- Debounced search and filtering

### Network Efficiency
- Batch operations where possible
- Compression for large transfers
- Intelligent prefetching based on user behavior

## Security Architecture

### Credential Management
- OS keychain integration for secure storage
- Encrypted configuration files
- No plaintext passwords in memory or logs

### Email Security
- PGP encryption/decryption support
- Digital signature verification
- Secure attachment handling

### Network Security
- TLS/SSL for all connections
- Certificate validation
- Protection against common email security threats

## Extension Points

### Plugin System (Future)
- Well-defined interfaces for extensions
- Sandboxed execution environment
- Support for custom themes and keybindings

### Configuration
- Extensive YAML-based configuration
- Hot-reload capability for development
- Environment variable overrides