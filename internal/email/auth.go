package email

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"github.com/ybarbara/pombo/internal/config"
)

// BasicAuthProvider implements simple username/password authentication
type BasicAuthProvider struct {
	username string
	password string
	authType AuthType
}

// NewBasicAuthProvider creates a new basic authentication provider
func NewBasicAuthProvider(username, password string) *BasicAuthProvider {
	return &BasicAuthProvider{
		username: username,
		password: password,
		authType: AuthTypePassword,
	}
}

// GetCredentials returns the stored credentials
func (b *BasicAuthProvider) GetCredentials(ctx context.Context) (*Credentials, error) {
	return &Credentials{
		Type:     b.authType,
		Username: b.username,
		Password: b.password,
	}, nil
}

// RefreshIfNeeded is a no-op for basic auth
func (b *BasicAuthProvider) RefreshIfNeeded(ctx context.Context) error {
	return nil
}

// Type returns the authentication type
func (b *BasicAuthProvider) Type() AuthType {
	return b.authType
}

// GetToken returns nil for basic auth (not applicable)
func (b *BasicAuthProvider) GetToken(ctx context.Context) (*OAuthToken, error) {
	return nil, fmt.Errorf("basic auth does not support tokens")
}

// RefreshToken returns an error for basic auth (not applicable)
func (b *BasicAuthProvider) RefreshToken(ctx context.Context) (*OAuthToken, error) {
	return nil, fmt.Errorf("basic auth does not support token refresh")
}

// IsValid always returns true for basic auth if credentials are set
func (b *BasicAuthProvider) IsValid(ctx context.Context) bool {
	return b.username != "" && b.password != ""
}

// ExpiresAt returns nil for basic auth (doesn't expire)
func (b *BasicAuthProvider) ExpiresAt() *time.Time {
	return nil
}

// OAuth2AuthProvider implements OAuth2 authentication
type OAuth2AuthProvider struct {
	config       *oauth2.Config
	token        *oauth2.Token
	credStore    CredentialStore
	accountID    string
	refreshToken string
}

// NewOAuth2AuthProvider creates a new OAuth2 authentication provider
func NewOAuth2AuthProvider(config *oauth2.Config, credStore CredentialStore, accountID string) *OAuth2AuthProvider {
	return &OAuth2AuthProvider{
		config:    config,
		credStore: credStore,
		accountID: accountID,
	}
}

// GetCredentials returns OAuth2 credentials
func (o *OAuth2AuthProvider) GetCredentials(ctx context.Context) (*Credentials, error) {
	token, err := o.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	return &Credentials{
		Type:      AuthTypeOAuth2,
		Token:     token,
		ExpiresAt: &token.ExpiresAt,
	}, nil
}

// RefreshIfNeeded refreshes the token if it's expired or about to expire
func (o *OAuth2AuthProvider) RefreshIfNeeded(ctx context.Context) error {
	if o.token == nil {
		// Load token from store
		token, err := o.credStore.RetrieveToken(ctx, o.accountID)
		if err != nil {
			return fmt.Errorf("failed to retrieve token: %w", err)
		}
		o.token = &oauth2.Token{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenType:    token.TokenType,
			Expiry:       token.ExpiresAt,
		}
	}

	// Check if token needs refresh (expires within 5 minutes)
	if o.token.Expiry.Sub(time.Now()) < 5*time.Minute {
		newToken, err := o.config.TokenSource(ctx, o.token).Token()
		if err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}

		o.token = newToken

		// Store updated token
		oauthToken := &OAuthToken{
			AccessToken:  newToken.AccessToken,
			RefreshToken: newToken.RefreshToken,
			TokenType:    newToken.TokenType,
			ExpiresAt:    newToken.Expiry,
		}

		if err := o.credStore.StoreToken(ctx, o.accountID, oauthToken); err != nil {
			return fmt.Errorf("failed to store refreshed token: %w", err)
		}
	}

	return nil
}

// Type returns OAuth2 authentication type
func (o *OAuth2AuthProvider) Type() AuthType {
	return AuthTypeOAuth2
}

// GetToken returns the current OAuth2 token
func (o *OAuth2AuthProvider) GetToken(ctx context.Context) (*OAuthToken, error) {
	if err := o.RefreshIfNeeded(ctx); err != nil {
		return nil, err
	}

	if o.token == nil {
		return nil, fmt.Errorf("no OAuth2 token available")
	}

	scope := ""
	if scopeValue := o.token.Extra("scope"); scopeValue != nil {
		if s, ok := scopeValue.(string); ok {
			scope = s
		}
	}

	return &OAuthToken{
		AccessToken:  o.token.AccessToken,
		RefreshToken: o.token.RefreshToken,
		TokenType:    o.token.TokenType,
		ExpiresAt:    o.token.Expiry,
		Scope:        scope,
	}, nil
}

// RefreshToken explicitly refreshes the OAuth2 token
func (o *OAuth2AuthProvider) RefreshToken(ctx context.Context) (*OAuthToken, error) {
	if o.token == nil {
		return nil, fmt.Errorf("no token to refresh")
	}

	newToken, err := o.config.TokenSource(ctx, o.token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	o.token = newToken

	// Store updated token
	oauthToken := &OAuthToken{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		ExpiresAt:    newToken.Expiry,
	}

	if err := o.credStore.StoreToken(ctx, o.accountID, oauthToken); err != nil {
		return nil, fmt.Errorf("failed to store refreshed token: %w", err)
	}

	return oauthToken, nil
}

// IsValid checks if the OAuth2 token is valid
func (o *OAuth2AuthProvider) IsValid(ctx context.Context) bool {
	if o.token == nil {
		// Try to load from store
		token, err := o.credStore.RetrieveToken(ctx, o.accountID)
		if err != nil {
			return false
		}
		o.token = &oauth2.Token{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenType:    token.TokenType,
			Expiry:       token.ExpiresAt,
		}
	}

	return o.token.Valid()
}

// ExpiresAt returns when the token expires
func (o *OAuth2AuthProvider) ExpiresAt() *time.Time {
	if o.token == nil {
		return nil
	}
	return &o.token.Expiry
}

// SetToken sets the OAuth2 token (used during initial authentication)
func (o *OAuth2AuthProvider) SetToken(token *oauth2.Token) error {
	o.token = token
	
	// Store the token
	oauthToken := &OAuthToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresAt:    token.Expiry,
	}

	ctx := context.Background()
	return o.credStore.StoreToken(ctx, o.accountID, oauthToken)
}

// AuthProviderFactory creates authentication providers based on configuration
type AuthProviderFactory struct {
	credStore CredentialStore
}

// NewAuthProviderFactory creates a new authentication provider factory
func NewAuthProviderFactory(credStore CredentialStore) *AuthProviderFactory {
	return &AuthProviderFactory{
		credStore: credStore,
	}
}

// CreateProvider creates an authentication provider based on account configuration
func (f *AuthProviderFactory) CreateProvider(ctx context.Context, account *config.AccountConfig) (AuthProvider, error) {
	switch {
	case account.OAuth != nil:
		// OAuth2 authentication
		config := &oauth2.Config{
			ClientID:     account.OAuth.ClientID,
			ClientSecret: account.OAuth.ClientSecret,
			RedirectURL:  account.OAuth.RedirectURI,
			Scopes:       account.OAuth.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  account.OAuth.AuthURL,
				TokenURL: account.OAuth.TokenURL,
			},
		}

		provider := NewOAuth2AuthProvider(config, f.credStore, account.ID)
		
		// Try to load existing token
		if token, err := f.credStore.RetrieveToken(ctx, account.ID); err == nil {
			oauth2Token := &oauth2.Token{
				AccessToken:  token.AccessToken,
				RefreshToken: token.RefreshToken,
				TokenType:    token.TokenType,
				Expiry:       token.ExpiresAt,
			}
			_ = provider.SetToken(oauth2Token)
		}

		return provider, nil

	case account.IMAP.Password != "" || account.SMTP.Password != "":
		// Basic authentication
		username := account.IMAP.Username
		password := account.IMAP.Password
		if password == "" {
			password = account.SMTP.Password
		}

		if username == "" || password == "" {
			// Try to load from credential store
			creds, err := f.credStore.Retrieve(ctx, account.ID)
			if err != nil {
				return nil, fmt.Errorf("no credentials available for account %s: %w", account.ID, err)
			}
			username = creds.Username
			password = creds.Password
		}

		return NewBasicAuthProvider(username, password), nil

	default:
		// Try to load credentials from store as fallback
		creds, err := f.credStore.Retrieve(ctx, account.ID)
		if err != nil {
			return nil, fmt.Errorf("no authentication method configured for account %s: %w", account.ID, err)
		}
		
		return NewBasicAuthProvider(creds.Username, creds.Password), nil
	}
}

// ValidateProvider validates that an authentication provider is properly configured
func (f *AuthProviderFactory) ValidateProvider(ctx context.Context, provider AuthProvider) error {
	switch provider.Type() {
	case AuthTypePassword:
		creds, err := provider.GetCredentials(ctx)
		if err != nil {
			return fmt.Errorf("failed to get credentials: %w", err)
		}
		if creds.Username == "" || creds.Password == "" {
			return fmt.Errorf("username and password are required for basic authentication")
		}

	case AuthTypeOAuth2:
		if !provider.IsValid(ctx) {
			return fmt.Errorf("OAuth2 token is invalid or expired")
		}

	default:
		return fmt.Errorf("unsupported authentication type: %s", provider.Type())
	}

	return nil
}