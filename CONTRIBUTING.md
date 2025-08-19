# Contributing to POMBO

Thank you for your interest in contributing to POMBO! This document outlines how to contribute to the project effectively.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it to understand what behaviors are expected.

## Getting Started

### Prerequisites
- Go 1.21 or later
- Git
- Make
- Basic understanding of terminal applications and email protocols

### Development Setup
1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/pombo.git
   cd pombo
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/ybarbara/pombo.git
   ```
4. Install dependencies and tools:
   ```bash
   make setup
   ```
5. Verify your setup:
   ```bash
   make test
   ```

## Development Workflow

### Creating a Feature Branch
```bash
# Sync with upstream
git checkout develop
git pull upstream develop

# Create feature branch
git checkout -b feature/your-feature-name
```

### Making Changes
1. Write code following our [coding standards](#coding-standards)
2. Add or update tests for your changes
3. Update documentation as needed
4. Run quality checks:
   ```bash
   make check
   ```

### Committing Changes
We use [Conventional Commits](https://www.conventionalcommits.org/):

```bash
# Examples
git commit -m "feat: add OAuth2 support for Gmail"
git commit -m "fix: resolve memory leak in IMAP client"
git commit -m "docs: update API documentation"
git commit -m "test: add integration tests for SMTP"
```

**Commit Types:**
- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `style`: Code style changes
- `chore`: Build process or auxiliary tool changes

### Submitting a Pull Request
1. Push your branch to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
2. Open a pull request against the `develop` branch
3. Fill out the pull request template completely
4. Ensure all CI checks pass
5. Address any review feedback promptly

## Coding Standards

### Go Guidelines
- Follow standard Go conventions and idioms
- Use `gofmt` and `goimports` for formatting
- Write meaningful variable and function names
- Add documentation comments for all exported functions
- Handle errors explicitly with proper context

### Code Quality
- Maintain >80% test coverage for new code
- Write unit tests for all public functions
- Add integration tests for complex workflows
- Follow the established architecture patterns

### TUI Development
- Use Bubbletea patterns consistently
- Keep components focused and reusable
- Follow established styling conventions
- Ensure keyboard navigation works properly

## Testing

### Running Tests
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Run with race detection
make test-race
```

### Writing Tests
- Write table-driven tests for multiple scenarios
- Mock external dependencies using interfaces
- Test both success and error cases
- Include edge cases and boundary conditions

Example test structure:
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    inputType
        expected expectedType
        wantErr  bool
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Documentation

### Code Documentation
- Add package comments for all packages
- Document all exported functions and types
- Include usage examples where helpful
- Keep documentation current with code changes

### User Documentation
- Update relevant documentation for user-facing changes
- Include configuration examples
- Add troubleshooting information for common issues
- Update the CHANGELOG for notable changes

## Issue Guidelines

### Reporting Bugs
When reporting bugs, please include:
- Clear description of the issue
- Steps to reproduce the problem
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Relevant log output or error messages

Use the bug report template provided.

### Feature Requests
For feature requests, please include:
- Clear description of the proposed feature
- Use case or problem it solves
- Proposed implementation approach (if any)
- Willingness to implement the feature yourself

Use the feature request template provided.

## Pull Request Guidelines

### Before Submitting
- [ ] Code follows the project's coding standards
- [ ] All tests pass (`make test`)
- [ ] Code has been formatted (`make fmt`)
- [ ] Linting passes (`make lint`)
- [ ] Security scan passes (`make security`)
- [ ] Documentation has been updated
- [ ] CHANGELOG has been updated (for notable changes)

### Pull Request Template
Please fill out the entire pull request template, including:
- Description of changes
- Type of change (bug fix, feature, etc.)
- Testing performed
- Related issues
- Screenshots (for UI changes)

### Review Process
1. Automated checks must pass (CI/CD pipeline)
2. At least one maintainer must approve the PR
3. All review feedback must be addressed
4. PR will be merged using squash and merge

## Architecture Guidelines

### Adding New Features
1. Review the [architecture documentation](docs/ARCHITECTURE.md)
2. Ensure the feature fits within the existing architecture
3. Design interfaces before implementation
4. Consider performance and security implications
5. Plan for testing and documentation

### Email Protocol Implementation
- Follow RFC specifications carefully
- Implement proper error handling and retries
- Support standard authentication methods
- Test with multiple email providers
- Handle edge cases and malformed data

### UI Components
- Follow Bubbletea patterns and conventions
- Ensure components are reusable and composable
- Implement proper keyboard navigation
- Support responsive design for different terminal sizes
- Follow accessibility guidelines

## Security Guidelines

### Secure Coding Practices
- Never log or expose sensitive information
- Validate all user inputs
- Use secure communication protocols
- Follow the principle of least privilege
- Regular security reviews and updates

### Credential Handling
- Use OS keychain for secure storage
- Implement proper session management
- Support secure authentication methods
- Never store passwords in plaintext

### Dependencies
- Keep dependencies updated
- Review security advisories regularly
- Use minimal required permissions
- Audit third-party packages

## Performance Guidelines

### Code Performance
- Profile code for performance bottlenecks
- Use efficient algorithms and data structures
- Avoid unnecessary allocations
- Implement proper caching strategies

### UI Responsiveness
- Keep UI updates fast and responsive
- Use background processing for I/O operations
- Provide visual feedback for long operations
- Implement progressive loading

### Memory Management
- Monitor memory usage and leaks
- Use appropriate data structures
- Implement proper cleanup
- Consider garbage collection impact

## Communication

### Getting Help
- Check existing documentation first
- Search existing issues and discussions
- Ask questions in GitHub Discussions
- Join our community chat (if available)

### Reporting Security Issues
For security vulnerabilities, please:
1. **Do not** open a public issue
2. Email security concerns to: [security@pombo.dev]
3. Include detailed information about the vulnerability
4. Allow time for the issue to be addressed before disclosure

## Recognition

Contributors are recognized in:
- The project README
- Release notes for significant contributions
- Annual contributor acknowledgments

We appreciate all forms of contribution, including:
- Code contributions
- Documentation improvements
- Bug reports and testing
- Feature suggestions and discussions
- Community support and outreach

## Questions?

If you have questions about contributing, please:
1. Check the [development documentation](docs/DEVELOPMENT.md)
2. Search existing GitHub issues and discussions
3. Open a new discussion for general questions
4. Contact the maintainers directly for urgent matters

Thank you for contributing to POMBO! 🚀