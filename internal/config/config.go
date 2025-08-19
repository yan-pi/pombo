package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	App      AppConfig       `yaml:"app" mapstructure:"app"`
	Accounts []AccountConfig `yaml:"accounts" mapstructure:"accounts"`
	UI       UIConfig        `yaml:"ui" mapstructure:"ui"`
	Security SecurityConfig  `yaml:"security" mapstructure:"security"`
	Performance PerformanceConfig `yaml:"performance" mapstructure:"performance"`
	Logging  LoggingConfig   `yaml:"logging" mapstructure:"logging"`
}

// AppConfig holds general application settings
type AppConfig struct {
	CacheDir  string `yaml:"cache_dir" mapstructure:"cache_dir"`
	ConfigDir string `yaml:"config_dir" mapstructure:"config_dir"`
	DataDir   string `yaml:"data_dir" mapstructure:"data_dir"`
}

// AccountConfig represents an email account configuration
type AccountConfig struct {
	Name     string      `yaml:"name" mapstructure:"name"`
	Email    string      `yaml:"email" mapstructure:"email"`
	Provider string      `yaml:"provider" mapstructure:"provider"`
	IMAP     IMAPConfig  `yaml:"imap" mapstructure:"imap"`
	SMTP     SMTPConfig  `yaml:"smtp" mapstructure:"smtp"`
	OAuth    OAuthConfig `yaml:"oauth" mapstructure:"oauth"`
}

// IMAPConfig holds IMAP server configuration
type IMAPConfig struct {
	Host     string `yaml:"host" mapstructure:"host"`
	Port     int    `yaml:"port" mapstructure:"port"`
	TLS      bool   `yaml:"tls" mapstructure:"tls"`
	Username string `yaml:"username" mapstructure:"username"`
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host     string `yaml:"host" mapstructure:"host"`
	Port     int    `yaml:"port" mapstructure:"port"`
	TLS      bool   `yaml:"tls" mapstructure:"tls"`
	StartTLS bool   `yaml:"starttls" mapstructure:"starttls"`
	Username string `yaml:"username" mapstructure:"username"`
}

// OAuthConfig holds OAuth2 configuration
type OAuthConfig struct {
	Provider     string `yaml:"provider" mapstructure:"provider"`
	ClientID     string `yaml:"client_id" mapstructure:"client_id"`
	ClientSecret string `yaml:"client_secret" mapstructure:"client_secret"`
	RedirectURI  string `yaml:"redirect_uri" mapstructure:"redirect_uri"`
	Scopes       []string `yaml:"scopes" mapstructure:"scopes"`
}

// UIConfig holds user interface configuration
type UIConfig struct {
	Theme           string            `yaml:"theme" mapstructure:"theme"`
	VimKeybindings  bool              `yaml:"vim_keybindings" mapstructure:"vim_keybindings"`
	ShowLineNumbers bool              `yaml:"show_line_numbers" mapstructure:"show_line_numbers"`
	Layout          string            `yaml:"layout" mapstructure:"layout"`
	FontSize        int               `yaml:"font_size" mapstructure:"font_size"`
	Colors          map[string]string `yaml:"colors" mapstructure:"colors"`
}

// SecurityConfig holds security-related settings
type SecurityConfig struct {
	PGP PGPConfig `yaml:"pgp" mapstructure:"pgp"`
	TLS TLSConfig `yaml:"tls" mapstructure:"tls"`
}

// PGPConfig holds PGP encryption settings
type PGPConfig struct {
	AutoEncrypt   bool   `yaml:"auto_encrypt" mapstructure:"auto_encrypt"`
	AutoSign      bool   `yaml:"auto_sign" mapstructure:"auto_sign"`
	KeyringPath   string `yaml:"keyring_path" mapstructure:"keyring_path"`
	DefaultKeyID  string `yaml:"default_key_id" mapstructure:"default_key_id"`
}

// TLSConfig holds TLS security settings
type TLSConfig struct {
	VerifyCertificates bool   `yaml:"verify_certificates" mapstructure:"verify_certificates"`
	MinVersion         string `yaml:"min_version" mapstructure:"min_version"`
}

// PerformanceConfig holds performance-related settings
type PerformanceConfig struct {
	CacheSize             string `yaml:"cache_size" mapstructure:"cache_size"`
	SyncInterval          string `yaml:"sync_interval" mapstructure:"sync_interval"`
	ConcurrentConnections int    `yaml:"concurrent_connections" mapstructure:"concurrent_connections"`
	MessageBatchSize      int    `yaml:"message_batch_size" mapstructure:"message_batch_size"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level    string `yaml:"level" mapstructure:"level"`
	Format   string `yaml:"format" mapstructure:"format"`
	Output   string `yaml:"output" mapstructure:"output"`
	File     string `yaml:"file" mapstructure:"file"`
	MaxSize  int    `yaml:"max_size" mapstructure:"max_size"`
	MaxAge   int    `yaml:"max_age" mapstructure:"max_age"`
	Compress bool   `yaml:"compress" mapstructure:"compress"`
}

// Load loads the configuration from file and environment variables
func Load() (*Config, error) {
	v := viper.New()
	
	// Set configuration file name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	
	// Add configuration paths
	configDir := getConfigDir()
	v.AddConfigPath(configDir)
	v.AddConfigPath(".")
	v.AddConfigPath("configs")
	
	// Set default values
	setDefaults(v)
	
	// Enable environment variable support
	v.SetEnvPrefix("POMBO")
	v.AutomaticEnv()
	
	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults
	}
	
	// Unmarshal configuration
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Set computed values
	if cfg.App.ConfigDir == "" {
		cfg.App.ConfigDir = configDir
	}
	if cfg.App.CacheDir == "" {
		cfg.App.CacheDir = getCacheDir()
	}
	if cfg.App.DataDir == "" {
		cfg.App.DataDir = getDataDir()
	}
	
	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.cache_dir", getCacheDir())
	v.SetDefault("app.config_dir", getConfigDir())
	v.SetDefault("app.data_dir", getDataDir())
	
	// UI defaults
	v.SetDefault("ui.theme", "default")
	v.SetDefault("ui.vim_keybindings", true)
	v.SetDefault("ui.show_line_numbers", false)
	v.SetDefault("ui.layout", "three-pane")
	v.SetDefault("ui.font_size", 12)
	
	// Security defaults
	v.SetDefault("security.pgp.auto_encrypt", false)
	v.SetDefault("security.pgp.auto_sign", false)
	v.SetDefault("security.pgp.keyring_path", filepath.Join(getHomeDir(), ".gnupg"))
	v.SetDefault("security.tls.verify_certificates", true)
	v.SetDefault("security.tls.min_version", "1.2")
	
	// Performance defaults
	v.SetDefault("performance.cache_size", "100MB")
	v.SetDefault("performance.sync_interval", "5m")
	v.SetDefault("performance.concurrent_connections", 3)
	v.SetDefault("performance.message_batch_size", 50)
	
	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output", "stderr")
	v.SetDefault("logging.max_size", 10)
	v.SetDefault("logging.max_age", 30)
	v.SetDefault("logging.compress", true)
}

// Helper functions to get standard directories
func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func getConfigDir() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "pombo")
	}
	return filepath.Join(getHomeDir(), ".config", "pombo")
}

func getCacheDir() string {
	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return filepath.Join(cacheHome, "pombo")
	}
	return filepath.Join(getHomeDir(), ".cache", "pombo")
}

func getDataDir() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "pombo")
	}
	return filepath.Join(getHomeDir(), ".local", "share", "pombo")
}