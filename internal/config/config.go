package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	App         AppConfig           `yaml:"app" mapstructure:"app"`
	Accounts    []AccountConfig     `yaml:"accounts" mapstructure:"accounts"`
	Email       EmailConfig         `yaml:"email" mapstructure:"email"`
	UI          UIConfig            `yaml:"ui" mapstructure:"ui"`
	Security    SecurityConfig      `yaml:"security" mapstructure:"security"`
	Performance PerformanceConfig   `yaml:"performance" mapstructure:"performance"`
	Logging     LoggingConfig       `yaml:"logging" mapstructure:"logging"`
}

// AppConfig holds general application settings
type AppConfig struct {
	CacheDir  string `yaml:"cache_dir" mapstructure:"cache_dir"`
	ConfigDir string `yaml:"config_dir" mapstructure:"config_dir"`
	DataDir   string `yaml:"data_dir" mapstructure:"data_dir"`
}

// EmailConfig holds email-specific configuration
type EmailConfig struct {
	DefaultAccount      string             `yaml:"default_account" mapstructure:"default_account"`
	CheckInterval       time.Duration      `yaml:"check_interval" mapstructure:"check_interval"`
	AutoSync            bool               `yaml:"auto_sync" mapstructure:"auto_sync"`
	BackgroundSync      bool               `yaml:"background_sync" mapstructure:"background_sync"`
	ConnectionPool      ConnectionPoolConfig `yaml:"connection_pool" mapstructure:"connection_pool"`
	MessageCache        MessageCacheConfig   `yaml:"message_cache" mapstructure:"message_cache"`
	AttachmentLimits    AttachmentLimitsConfig `yaml:"attachment_limits" mapstructure:"attachment_limits"`
	ErrorRetry          ErrorRetryConfig     `yaml:"error_retry" mapstructure:"error_retry"`
}

// ConnectionPoolConfig holds connection pool settings
type ConnectionPoolConfig struct {
	MaxConnections      int           `yaml:"max_connections" mapstructure:"max_connections"`
	MaxIdleConnections  int           `yaml:"max_idle_connections" mapstructure:"max_idle_connections"`
	ConnectionLifetime  time.Duration `yaml:"connection_lifetime" mapstructure:"connection_lifetime"`
	IdleTimeout         time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" mapstructure:"health_check_interval"`
	ConnectTimeout      time.Duration `yaml:"connect_timeout" mapstructure:"connect_timeout"`
}

// MessageCacheConfig holds message caching settings
type MessageCacheConfig struct {
	MaxSize         string        `yaml:"max_size" mapstructure:"max_size"`
	TTL             time.Duration `yaml:"ttl" mapstructure:"ttl"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" mapstructure:"cleanup_interval"`
	CacheHeaders    bool          `yaml:"cache_headers" mapstructure:"cache_headers"`
	CacheBodies     bool          `yaml:"cache_bodies" mapstructure:"cache_bodies"`
	CacheAttachments bool         `yaml:"cache_attachments" mapstructure:"cache_attachments"`
}

// AttachmentLimitsConfig holds attachment handling limits
type AttachmentLimitsConfig struct {
	MaxSize         string   `yaml:"max_size" mapstructure:"max_size"`
	AllowedTypes    []string `yaml:"allowed_types,omitempty" mapstructure:"allowed_types"`
	BlockedTypes    []string `yaml:"blocked_types,omitempty" mapstructure:"blocked_types"`
	AutoDownload    bool     `yaml:"auto_download" mapstructure:"auto_download"`
	VirusScan       bool     `yaml:"virus_scan" mapstructure:"virus_scan"`
}

// ErrorRetryConfig holds error handling and retry settings
type ErrorRetryConfig struct {
	MaxRetries       int           `yaml:"max_retries" mapstructure:"max_retries"`
	BaseDelay        time.Duration `yaml:"base_delay" mapstructure:"base_delay"`
	MaxDelay         time.Duration `yaml:"max_delay" mapstructure:"max_delay"`
	Multiplier       float64       `yaml:"multiplier" mapstructure:"multiplier"`
	JitterEnabled    bool          `yaml:"jitter_enabled" mapstructure:"jitter_enabled"`
	RetryableErrors  []string      `yaml:"retryable_errors,omitempty" mapstructure:"retryable_errors"`
}

// AccountConfig represents an email account configuration
type AccountConfig struct {
	ID       string             `yaml:"id" mapstructure:"id"`
	Name     string             `yaml:"name" mapstructure:"name"`
	Email    string             `yaml:"email" mapstructure:"email"`
	Provider string             `yaml:"provider" mapstructure:"provider"`
	IMAP     IMAPConfig         `yaml:"imap" mapstructure:"imap"`
	SMTP     SMTPConfig         `yaml:"smtp" mapstructure:"smtp"`
	OAuth    *OAuthConfig       `yaml:"oauth,omitempty" mapstructure:"oauth"`
	Settings *AccountSettings   `yaml:"settings,omitempty" mapstructure:"settings"`
	Enabled  bool               `yaml:"enabled" mapstructure:"enabled"`
}

// IMAPConfig holds IMAP server configuration
type IMAPConfig struct {
	Host        string        `yaml:"host" mapstructure:"host"`
	Port        int           `yaml:"port" mapstructure:"port"`
	TLS         bool          `yaml:"tls" mapstructure:"tls"`
	StartTLS    bool          `yaml:"starttls" mapstructure:"starttls"`
	Username    string        `yaml:"username" mapstructure:"username"`
	Password    string        `yaml:"password,omitempty" mapstructure:"password"`
	Timeout     time.Duration `yaml:"timeout" mapstructure:"timeout"`
	KeepAlive   time.Duration `yaml:"keepalive" mapstructure:"keepalive"`
	UseIdle     bool          `yaml:"use_idle" mapstructure:"use_idle"`
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host        string        `yaml:"host" mapstructure:"host"`
	Port        int           `yaml:"port" mapstructure:"port"`
	TLS         bool          `yaml:"tls" mapstructure:"tls"`
	StartTLS    bool          `yaml:"starttls" mapstructure:"starttls"`
	Username    string        `yaml:"username" mapstructure:"username"`
	Password    string        `yaml:"password,omitempty" mapstructure:"password"`
	Timeout     time.Duration `yaml:"timeout" mapstructure:"timeout"`
	RequireTLS  bool          `yaml:"require_tls" mapstructure:"require_tls"`
}

// OAuthConfig holds OAuth2 configuration
type OAuthConfig struct {
	Provider     string   `yaml:"provider" mapstructure:"provider"`
	ClientID     string   `yaml:"client_id" mapstructure:"client_id"`
	ClientSecret string   `yaml:"client_secret,omitempty" mapstructure:"client_secret"`
	RedirectURI  string   `yaml:"redirect_uri" mapstructure:"redirect_uri"`
	Scopes       []string `yaml:"scopes" mapstructure:"scopes"`
	AuthURL      string   `yaml:"auth_url,omitempty" mapstructure:"auth_url"`
	TokenURL     string   `yaml:"token_url,omitempty" mapstructure:"token_url"`
}

// AccountSettings represents account-specific settings
type AccountSettings struct {
	Signature           string        `yaml:"signature,omitempty" mapstructure:"signature"`
	AutoBCC             []string      `yaml:"auto_bcc,omitempty" mapstructure:"auto_bcc"`
	SyncInterval        time.Duration `yaml:"sync_interval" mapstructure:"sync_interval"`
	MaxSyncMessages     int           `yaml:"max_sync_messages" mapstructure:"max_sync_messages"`
	ComposeFormat       string        `yaml:"compose_format" mapstructure:"compose_format"`
	AutoMarkRead        bool          `yaml:"auto_mark_read" mapstructure:"auto_mark_read"`
	DownloadAttachments bool          `yaml:"download_attachments" mapstructure:"download_attachments"`
	CheckSSLCert        bool          `yaml:"check_ssl_cert" mapstructure:"check_ssl_cert"`
	FolderMapping       map[string]string `yaml:"folder_mapping,omitempty" mapstructure:"folder_mapping"`
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
	
	// Email defaults
	v.SetDefault("email.check_interval", "5m")
	v.SetDefault("email.auto_sync", true)
	v.SetDefault("email.background_sync", true)
	
	// Connection pool defaults
	v.SetDefault("email.connection_pool.max_connections", 5)
	v.SetDefault("email.connection_pool.max_idle_connections", 2)
	v.SetDefault("email.connection_pool.connection_lifetime", "30m")
	v.SetDefault("email.connection_pool.idle_timeout", "5m")
	v.SetDefault("email.connection_pool.health_check_interval", "1m")
	v.SetDefault("email.connection_pool.connect_timeout", "30s")
	
	// Message cache defaults
	v.SetDefault("email.message_cache.max_size", "100MB")
	v.SetDefault("email.message_cache.ttl", "24h")
	v.SetDefault("email.message_cache.cleanup_interval", "1h")
	v.SetDefault("email.message_cache.cache_headers", true)
	v.SetDefault("email.message_cache.cache_bodies", true)
	v.SetDefault("email.message_cache.cache_attachments", false)
	
	// Attachment limits defaults
	v.SetDefault("email.attachment_limits.max_size", "25MB")
	v.SetDefault("email.attachment_limits.auto_download", false)
	v.SetDefault("email.attachment_limits.virus_scan", false)
	
	// Error retry defaults
	v.SetDefault("email.error_retry.max_retries", 3)
	v.SetDefault("email.error_retry.base_delay", "1s")
	v.SetDefault("email.error_retry.max_delay", "1m")
	v.SetDefault("email.error_retry.multiplier", 2.0)
	v.SetDefault("email.error_retry.jitter_enabled", true)
	
	// Account defaults
	v.SetDefault("accounts.enabled", true)
	v.SetDefault("accounts.imap.timeout", "30s")
	v.SetDefault("accounts.imap.keepalive", "5m")
	v.SetDefault("accounts.imap.use_idle", true)
	v.SetDefault("accounts.smtp.timeout", "30s")
	v.SetDefault("accounts.smtp.require_tls", true)
	v.SetDefault("accounts.settings.sync_interval", "5m")
	v.SetDefault("accounts.settings.max_sync_messages", 1000)
	v.SetDefault("accounts.settings.compose_format", "text")
	v.SetDefault("accounts.settings.auto_mark_read", false)
	v.SetDefault("accounts.settings.download_attachments", false)
	v.SetDefault("accounts.settings.check_ssl_cert", true)
	
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