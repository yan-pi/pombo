# POMBO 📧

A modern, fast, and secure terminal-based email client built with Go and the [Charm](https://charm.sh/) ecosystem.

> **Note**: POMBO is currently in early development. The foundation has been laid with the Bubbletea TUI framework, configuration system, and project structure. Core email functionality is being actively developed.

## ✨ Features

### Current (Phase 1 - Foundation Complete)
- 🏗️ **Solid Architecture**: Built with Bubbletea for reactive TUI experiences
- ⚙️ **Configuration System**: YAML-based configuration with Viper
- 🎨 **Beautiful UI**: Styled with Lipgloss and Charm design principles
- 🔧 **Development Tooling**: Complete build system with Make, linting, and testing
- 📚 **Documentation**: Comprehensive development and architecture guides

### Planned (In Development)
- 📧 **Email Protocols**: IMAP and SMTP support with connection pooling
- 🔐 **OAuth2 Authentication**: Gmail, Outlook, and other major providers
- 👥 **Multiple Accounts**: Manage multiple email accounts seamlessly
- 🔍 **Full-Text Search**: Fast email search with Bleve search engine
- 💬 **Threaded Conversations**: Intelligent email threading
- 🔒 **PGP Encryption**: End-to-end encryption and digital signatures
- ⌨️ **Vim Keybindings**: Familiar navigation for Vim users
- 🌙 **Themes**: Customizable color schemes and layouts
- 📎 **Attachments**: Full attachment support with previews
- 🔄 **Offline Support**: Local caching for offline email access

## 🚀 Quick Start

### Prerequisites
- Go 1.21 or later
- Make (for build automation)

### Installation

```bash
# Clone the repository
git clone https://github.com/ybarbara/pombo.git
cd pombo

# Install dependencies and build
make setup
make build

# Run the application
./build/pombo
```

### Development

```bash
# Run in development mode with debug logging
make dev

# Run all quality checks
make check

# Run tests
make test

# See all available commands
make help
```

## 📖 Documentation

- **[Architecture Guide](docs/ARCHITECTURE.md)** - System design and patterns
- **[Development Guide](docs/DEVELOPMENT.md)** - Development setup and guidelines
- **[Contributing](CONTRIBUTING.md)** - How to contribute to the project
- **[CLAUDE.md](CLAUDE.md)** - Development reference for Claude Code

## 🛠️ Technology Stack

### Core Framework
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** - Elm-inspired TUI framework
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - CSS-like styling for terminals
- **[Bubbles](https://github.com/charmbracelet/bubbles)** - Pre-built TUI components
- **[Charm Log](https://github.com/charmbracelet/log)** - Structured logging

### Email & Security (Planned)
- **go-imap/v2** - Modern IMAP client
- **go-smtp** - SMTP client for sending
- **GopenPGP** - PGP encryption and signing
- **OAuth2** - Secure authentication

### Storage & Configuration
- **Viper** - Configuration management
- **SQLite** - Local email caching
- **Bleve** - Full-text search engine

## 🏗️ Project Structure

```
pombo/
├── cmd/pombo/              # Main application entry point
├── internal/               # Private application code
│   ├── app/               # Application initialization
│   ├── config/            # Configuration management  
│   ├── email/             # Email protocol handlers (planned)
│   ├── ui/                # TUI components and styling
│   ├── storage/           # Local storage and caching (planned)
│   └── crypto/            # PGP encryption (planned)
├── configs/               # Configuration examples
├── docs/                  # Documentation
└── Makefile              # Build automation
```

## ⌨️ Keybindings

POMBO will support Vim-style keybindings by default:

- `j/k` - Navigate up/down
- `h/l` - Navigate left/right  
- `q` - Quit
- `?` - Help
- `c` - Compose email
- `r` - Reply
- `/` - Search

Full keybinding customization will be available via YAML configuration.

## 📝 Configuration

POMBO uses YAML configuration files:

```yaml
# ~/.config/pombo/config.yaml
app:
  theme: "dark"
  vim_keybindings: true

accounts:
  - name: "work"
    email: "user@company.com" 
    provider: "outlook"

ui:
  show_line_numbers: false
  layout: "three-pane"
```

See [configs/pombo.example.yaml](configs/pombo.example.yaml) for a complete example.

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow
1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `make check` to verify quality
5. Submit a pull request

## 📊 Project Status

**Phase 1: Foundation (Complete)**
- ✅ Project structure and build system
- ✅ Configuration management with Viper
- ✅ Basic TUI framework with Bubbletea
- ✅ Styling system with Lipgloss
- ✅ Documentation and development guides

**Phase 2: Email Protocols (In Progress)**
- 🔄 IMAP client implementation
- 🔄 SMTP client for sending
- 🔄 OAuth2 authentication flow
- 🔄 Multi-account support

**Phase 3: Core Features (Planned)**
- ⏳ Message list and threading
- ⏳ Email composition interface
- ⏳ Search functionality
- ⏳ Local caching and sync

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Charm](https://charm.sh/) team for the amazing TUI ecosystem
- The Go community for excellent email libraries
- All contributors and early testers

---

**Built with ❤️ using the Charm ecosystem**
