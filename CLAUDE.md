# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

POMBO is an open-source TUI (Terminal User Interface) email client written in Go using the Charm ecosystem, focusing on simplicity, speed, and excellent user experience. The project leverages Bubbletea for the TUI framework, along with Lipgloss, Bubbles, and other Charm tools.

## Key Features

- **Email Protocols**: IMAP and SMTP support with OAuth2 authentication for Gmail, Outlook, and other providers
- **Multi-Account**: Support for multiple email accounts with threaded conversations
- **TUI Interface**: Rich terminal interface using Bubbletea with Vim-style keybindings
- **Security**: PGP encryption and signing support with secure credential storage
- **Performance**: Local caching with SQLite, full-text search with Bleve
- **Configuration**: YAML-based configuration with hot-reload capability
- **Cross-Platform**: Linux, macOS, Windows compatibility

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
│   ├── config/            # Configuration management
│   ├── email/             # Email protocol handlers
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

# Specific test commands
go test ./internal/email/...                    # Test email components
go test -run TestIMAPClient ./internal/email/   # Run specific test
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