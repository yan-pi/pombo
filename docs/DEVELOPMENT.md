# POMBO Development Guide

This guide helps developers get started with POMBO development and outlines our development practices.

## Getting Started

### Prerequisites
- Go 1.21 or later
- Git
- Make (for build automation)
- golangci-lint (for code quality)
- gosec (for security scanning)

### Initial Setup
```bash
# Clone the repository
git clone https://github.com/ybarbara/pombo.git
cd pombo

# Install dependencies and development tools
make setup

# Build the application
make build

# Run tests to verify setup
make test
```

### Development Workflow
```bash
# Run in development mode with debug logging
make dev

# Run all quality checks before committing
make check

# Format code and run lints
make fmt lint

# Run tests with coverage
make test-coverage
```

## Project Structure

### Key Directories
- `cmd/pombo/`: Application entry point and CLI setup
- `internal/app/`: Application initialization and lifecycle management
- `internal/config/`: Configuration management using Viper
- `internal/email/`: Email protocol implementations (IMAP/SMTP/OAuth)
- `internal/ui/`: Bubbletea TUI components and styling
- `internal/storage/`: Local data persistence and caching
- `internal/crypto/`: PGP encryption and security features

### Code Organization Principles
1. **Separation of Concerns**: Each package has a single responsibility
2. **Dependency Direction**: Dependencies flow inward (UI → App → Domain)
3. **Interface Abstraction**: Use interfaces to define contracts between layers
4. **Testability**: Design for easy unit testing and mocking

## Coding Standards

### Go Best Practices
- Follow standard Go formatting (`gofmt`, `goimports`)
- Use meaningful variable and function names
- Add documentation comments for all exported functions
- Handle errors explicitly and provide context
- Use interfaces for abstraction and testing

### TUI Development with Bubbletea
```go
// Model represents component state
type Model struct {
    // State fields
}

// Init initializes the component
func (m Model) Init() tea.Cmd {
    return nil
}

// Update handles messages and updates state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle keyboard input
    case customMsg:
        // Handle custom messages
    }
    return m, nil
}

// View renders the component
func (m Model) View() string {
    return "Component content"
}
```

### Error Handling
```go
// Always provide context in errors
return fmt.Errorf("failed to connect to IMAP server: %w", err)

// Use typed errors for specific conditions
var ErrAuthenticationFailed = errors.New("authentication failed")

// Log errors with structured data
logger.Error("operation failed", 
    "operation", "email_sync", 
    "account", account.Name,
    "error", err)
```

## Testing Strategy

### Unit Testing
- Test all public functions and methods
- Use table-driven tests for multiple test cases
- Mock external dependencies using interfaces
- Aim for >80% code coverage

```go
func TestEmailParser(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected *Email
        wantErr  bool
    }{
        {
            name:     "valid email",
            input:    "test@example.com",
            expected: &Email{Address: "test@example.com"},
            wantErr:  false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseEmail(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Testing
- Test email protocol implementations with real servers
- Use Docker containers for consistent test environments
- Test complete user workflows end-to-end

### TUI Testing
- Test component state transitions
- Verify rendering output for different states
- Test keyboard input handling

## Configuration Management

### Configuration Structure
Configuration uses a hierarchical YAML structure with environment variable overrides:

```yaml
app:
  cache_dir: ~/.cache/pombo
  log_level: info

accounts:
  - name: work
    email: user@company.com
    provider: outlook

ui:
  theme: dark
  vim_keybindings: true
```

### Adding New Configuration Options
1. Add fields to the appropriate config struct in `internal/config/config.go`
2. Set default values in the `setDefaults()` function
3. Update example configuration files
4. Add validation if necessary

## Email Protocol Implementation

### IMAP Client Guidelines
- Use connection pooling for efficiency
- Implement proper error handling and retries
- Support IDLE for real-time updates
- Handle network disconnections gracefully

### OAuth2 Implementation
- Store tokens securely using OS keychain
- Implement automatic token refresh
- Support multiple OAuth providers
- Never log or expose tokens

### Message Processing
- Parse email headers correctly
- Handle different content types (text, HTML, attachments)
- Implement thread detection and grouping
- Support PGP encryption/decryption

## UI Development Guidelines

### Bubbletea Components
- Keep components focused and reusable
- Use consistent styling through shared styles
- Handle window resize events properly
- Implement proper keyboard navigation

### Styling with Lipgloss
```go
// Define consistent styles
var titleStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#7C3AED")).
    Bold(true).
    MarginBottom(1)

// Use adaptive sizing
func (m Model) View() string {
    content := titleStyle.Width(m.width).Render("Title")
    return content
}
```

### Keyboard Shortcuts
- Follow Vim conventions where appropriate
- Provide help text for all shortcuts
- Support customization through configuration
- Ensure accessibility for screen readers

## Performance Guidelines

### Memory Management
- Use appropriate data structures for large datasets
- Implement LRU caching for frequently accessed data
- Avoid memory leaks in long-running operations
- Profile memory usage regularly

### UI Responsiveness
- Keep update functions fast and non-blocking
- Use background goroutines for I/O operations
- Provide visual feedback for long operations
- Implement progressive loading for large datasets

### Network Efficiency
- Batch operations where possible
- Use compression for large transfers
- Implement intelligent prefetching
- Cache frequently accessed data locally

## Security Guidelines

### Credential Handling
- Never log passwords, tokens, or other credentials
- Use secure storage (OS keychain) for sensitive data
- Implement proper session management
- Validate all user inputs

### Email Security
- Verify TLS certificates properly
- Support PGP encryption and signing
- Sanitize HTML content in emails
- Protect against email-based attacks

### Code Security
- Run security scans regularly (`make security`)
- Keep dependencies updated
- Follow secure coding practices
- Review security-sensitive code thoroughly

## Documentation

### Code Documentation
- Add package-level documentation for all packages
- Document all exported functions and types
- Include usage examples in documentation
- Keep documentation up-to-date with code changes

### API Documentation
- Use standard Go documentation conventions
- Generate documentation with `go doc`
- Include code examples in documentation
- Document configuration options thoroughly

## Contributing

### Pull Request Process
1. Create a feature branch from `develop`
2. Implement changes with appropriate tests
3. Run full test suite and quality checks
4. Update documentation as needed
5. Submit PR with clear description
6. Address review feedback promptly

### Code Review Guidelines
- Focus on correctness, security, and maintainability
- Check for proper error handling
- Verify test coverage for new code
- Ensure documentation is updated
- Consider performance implications

### Commit Messages
Use conventional commit format:
```
feat: add OAuth2 support for Gmail
fix: resolve memory leak in IMAP client
docs: update development guide
test: add integration tests for SMTP
```

## Debugging

### Debug Logging
```bash
# Enable debug logging
POMBO_LOG_LEVEL=debug make run

# Log to file for analysis
POMBO_LOG_FILE=/tmp/pombo.log make dev
```

### TUI Debugging
```bash
# Use separate terminal for debug output
POMBO_DEBUG_FILE=/tmp/pombo.log make dev
tail -f /tmp/pombo.log  # In another terminal
```

### Email Protocol Debugging
```bash
# Enable IMAP/SMTP protocol logging
POMBO_EMAIL_DEBUG=true make dev
```

### Common Issues
- **Build failures**: Check Go version and dependencies
- **Test failures**: Ensure clean test environment
- **TUI rendering issues**: Check terminal compatibility
- **Email connection issues**: Verify network and credentials