package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/ybarbara/pombo/internal/app"
	"github.com/ybarbara/pombo/internal/config"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx := context.Background()
	
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Error("failed to execute command", "error", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "pombo",
	Short: "A modern TUI email client",
	Long: `POMBO is a fast, secure, and user-friendly terminal-based email client
built with Go and the Charm ecosystem. It supports multiple accounts,
OAuth2 authentication, PGP encryption, and Vim-style keybindings.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Initialize and run the application
		application := app.New(cfg)
		return application.Run()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("POMBO %s\n", rootCmd.Version)
		fmt.Printf("Built with Go %s\n", "1.21+")
	},
}