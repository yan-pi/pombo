package email

import (
	"context"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"github.com/ybarbara/pombo/internal/config"
)

// MockCredentialStore for testing
type MockCredentialStore struct {
	credentials map[string]*Credentials
	tokens      map[string]*OAuthToken
}

func NewMockCredentialStore() *MockCredentialStore {
	return &MockCredentialStore{
		credentials: make(map[string]*Credentials),
		tokens:      make(map[string]*OAuthToken),
	}
}

func (m *MockCredentialStore) Store(ctx context.Context, accountID string, creds *Credentials) error {
	m.credentials[accountID] = creds
	return nil
}

func (m *MockCredentialStore) Retrieve(ctx context.Context, accountID string) (*Credentials, error) {
	if creds, exists := m.credentials[accountID]; exists {
		return creds, nil
	}
	return nil, &EmailError{Type: ErrorTypeNotFound, Message: "credentials not found"}
}

func (m *MockCredentialStore) Delete(ctx context.Context, accountID string) error {
	delete(m.credentials, accountID)
	return nil
}

func (m *MockCredentialStore) List(ctx context.Context) ([]string, error) {
	var accounts []string
	for accountID := range m.credentials {
		accounts = append(accounts, accountID)
	}
	return accounts, nil
}

func (m *MockCredentialStore) StoreToken(ctx context.Context, accountID string, token *OAuthToken) error {
	m.tokens[accountID] = token
	return nil
}

func (m *MockCredentialStore) RetrieveToken(ctx context.Context, accountID string) (*OAuthToken, error) {
	if token, exists := m.tokens[accountID]; exists {
		return token, nil
	}
	return nil, &EmailError{Type: ErrorTypeNotFound, Message: "token not found"}
}

func (m *MockCredentialStore) DeleteToken(ctx context.Context, accountID string) error {
	delete(m.tokens, accountID)
	return nil
}

func (m *MockCredentialStore) IsAvailable(ctx context.Context) bool {
	return true
}

func (m *MockCredentialStore) TestAccess(ctx context.Context) error {
	return nil
}

func TestBasicAuthProvider(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantValid bool
	}{
		{
			name:     "valid credentials",
			username: "user@example.com",
			password: "password123",
			wantValid: true,
		},
		{
			name:     "empty username",
			username: "",
			password: "password123",
			wantValid: false,
		},
		{
			name:     "empty password",
			username: "user@example.com",
			password: "",
			wantValid: false,
		},
		{
			name:     "empty credentials",
			username: "",
			password: "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewBasicAuthProvider(tt.username, tt.password)

			// Test Type
			if provider.Type() != AuthTypePassword {
				t.Errorf("Type() = %v, want %v", provider.Type(), AuthTypePassword)
			}

			// Test IsValid
			ctx := context.Background()
			if valid := provider.IsValid(ctx); valid != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", valid, tt.wantValid)
			}

			// Test GetCredentials
			creds, err := provider.GetCredentials(ctx)
			if err != nil {
				t.Errorf("GetCredentials() error = %v", err)
				return
			}

			if creds.Type != AuthTypePassword {
				t.Errorf("GetCredentials().Type = %v, want %v", creds.Type, AuthTypePassword)
			}

			if creds.Username != tt.username {
				t.Errorf("GetCredentials().Username = %v, want %v", creds.Username, tt.username)
			}

			if creds.Password != tt.password {
				t.Errorf("GetCredentials().Password = %v, want %v", creds.Password, tt.password)
			}

			// Test RefreshIfNeeded (should be no-op)
			if err := provider.RefreshIfNeeded(ctx); err != nil {
				t.Errorf("RefreshIfNeeded() error = %v", err)
			}

			// Test GetToken (should return error)
			_, err = provider.GetToken(ctx)
			if err == nil {
				t.Error("GetToken() should return error for basic auth")
			}

			// Test RefreshToken (should return error)
			_, err = provider.RefreshToken(ctx)
			if err == nil {
				t.Error("RefreshToken() should return error for basic auth")
			}

			// Test ExpiresAt (should return nil)
			if expiresAt := provider.ExpiresAt(); expiresAt != nil {
				t.Errorf("ExpiresAt() = %v, want nil", expiresAt)
			}
		})
	}
}

func TestOAuth2AuthProvider(t *testing.T) {
	credStore := NewMockCredentialStore()
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
	}

	provider := NewOAuth2AuthProvider(config, credStore, "test-account")

	// Test Type
	if provider.Type() != AuthTypeOAuth2 {
		t.Errorf("Type() = %v, want %v", provider.Type(), AuthTypeOAuth2)
	}

	ctx := context.Background()

	// Test with no token (should be invalid)
	if valid := provider.IsValid(ctx); valid {
		t.Error("IsValid() should return false when no token is available")
	}

	// Set a valid token
	validToken := &oauth2.Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	if err := provider.SetToken(validToken); err != nil {
		t.Errorf("SetToken() error = %v", err)
	}

	// Test IsValid with token
	if valid := provider.IsValid(ctx); !valid {
		t.Error("IsValid() should return true when valid token is available")
	}

	// Test GetCredentials
	creds, err := provider.GetCredentials(ctx)
	if err != nil {
		t.Errorf("GetCredentials() error = %v", err)
		return
	}

	if creds.Type != AuthTypeOAuth2 {
		t.Errorf("GetCredentials().Type = %v, want %v", creds.Type, AuthTypeOAuth2)
	}

	if creds.Token == nil {
		t.Error("GetCredentials().Token should not be nil")
	}

	// Test GetToken
	token, err := provider.GetToken(ctx)
	if err != nil {
		t.Errorf("GetToken() error = %v", err)
		return
	}

	if token.AccessToken != validToken.AccessToken {
		t.Errorf("GetToken().AccessToken = %v, want %v", token.AccessToken, validToken.AccessToken)
	}

	// Test ExpiresAt
	expiresAt := provider.ExpiresAt()
	if expiresAt == nil {
		t.Error("ExpiresAt() should not return nil for OAuth2 token")
	}

	if !expiresAt.Equal(validToken.Expiry) {
		t.Errorf("ExpiresAt() = %v, want %v", expiresAt, validToken.Expiry)
	}
}

func TestAuthProviderFactory(t *testing.T) {
	credStore := NewMockCredentialStore()
	factory := NewAuthProviderFactory(credStore)

	ctx := context.Background()

	t.Run("basic auth provider", func(t *testing.T) {
		account := &config.AccountConfig{
			ID:    "test-basic",
			Name:  "Test Basic",
			Email: "test@example.com",
			IMAP: config.IMAPConfig{
				Username: "test@example.com",
				Password: "password123",
			},
		}

		provider, err := factory.CreateProvider(ctx, account)
		if err != nil {
			t.Errorf("CreateProvider() error = %v", err)
			return
		}

		if provider.Type() != AuthTypePassword {
			t.Errorf("Provider type = %v, want %v", provider.Type(), AuthTypePassword)
		}

		// Test validation
		if err := factory.ValidateProvider(ctx, provider); err != nil {
			t.Errorf("ValidateProvider() error = %v", err)
		}
	})

	t.Run("oauth2 provider", func(t *testing.T) {
		account := &config.AccountConfig{
			ID:    "test-oauth",
			Name:  "Test OAuth",
			Email: "test@gmail.com",
			OAuth: &config.OAuthConfig{
				Provider:     "gmail",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURI:  "http://localhost:8080/callback",
				Scopes:       []string{"email"},
				AuthURL:      "https://accounts.google.com/o/oauth2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
			},
		}

		provider, err := factory.CreateProvider(ctx, account)
		if err != nil {
			t.Errorf("CreateProvider() error = %v", err)
			return
		}

		if provider.Type() != AuthTypeOAuth2 {
			t.Errorf("Provider type = %v, want %v", provider.Type(), AuthTypeOAuth2)
		}

		// OAuth2 provider without token should be invalid
		if err := factory.ValidateProvider(ctx, provider); err == nil {
			t.Error("ValidateProvider() should return error for OAuth2 provider without token")
		}
	})

	t.Run("no auth configured", func(t *testing.T) {
		account := &config.AccountConfig{
			ID:    "test-no-auth",
			Name:  "Test No Auth",
			Email: "test@example.com",
		}

		_, err := factory.CreateProvider(ctx, account)
		if err == nil {
			t.Error("CreateProvider() should return error when no auth is configured")
		}
	})

	t.Run("basic auth from credential store", func(t *testing.T) {
		// Store credentials in mock store
		creds := &Credentials{
			Type:     AuthTypePassword,
			Username: "stored@example.com",
			Password: "stored-password",
		}
		credStore.Store(ctx, "test-stored", creds)

		account := &config.AccountConfig{
			ID:    "test-stored",
			Name:  "Test Stored",
			Email: "stored@example.com",
		}

		provider, err := factory.CreateProvider(ctx, account)
		if err != nil {
			t.Errorf("CreateProvider() error = %v", err)
			return
		}

		retrievedCreds, err := provider.GetCredentials(ctx)
		if err != nil {
			t.Errorf("GetCredentials() error = %v", err)
			return
		}

		if retrievedCreds.Username != creds.Username {
			t.Errorf("Username = %v, want %v", retrievedCreds.Username, creds.Username)
		}

		if retrievedCreds.Password != creds.Password {
			t.Errorf("Password = %v, want %v", retrievedCreds.Password, creds.Password)
		}
	})
}

func TestAuthProviderValidation(t *testing.T) {
	credStore := NewMockCredentialStore()
	factory := NewAuthProviderFactory(credStore)
	ctx := context.Background()

	t.Run("invalid basic auth - no username", func(t *testing.T) {
		provider := NewBasicAuthProvider("", "password")
		err := factory.ValidateProvider(ctx, provider)
		if err == nil {
			t.Error("ValidateProvider() should return error for empty username")
		}
	})

	t.Run("invalid basic auth - no password", func(t *testing.T) {
		provider := NewBasicAuthProvider("user", "")
		err := factory.ValidateProvider(ctx, provider)
		if err == nil {
			t.Error("ValidateProvider() should return error for empty password")
		}
	})

	t.Run("valid basic auth", func(t *testing.T) {
		provider := NewBasicAuthProvider("user@example.com", "password123")
		err := factory.ValidateProvider(ctx, provider)
		if err != nil {
			t.Errorf("ValidateProvider() should not return error for valid basic auth: %v", err)
		}
	})
}

// Benchmark tests for performance
func BenchmarkBasicAuthProvider_GetCredentials(b *testing.B) {
	provider := NewBasicAuthProvider("user@example.com", "password123")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GetCredentials(ctx)
		if err != nil {
			b.Errorf("GetCredentials() error = %v", err)
		}
	}
}

func TestOAuth2AuthProvider_RefreshToken(t *testing.T) {
	credStore := NewMockCredentialStore()
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	provider := NewOAuth2AuthProvider(config, credStore, "test-account")
	ctx := context.Background()

	t.Run("refresh with no token", func(t *testing.T) {
		_, err := provider.RefreshToken(ctx)
		if err == nil {
			t.Error("RefreshToken() should return error when no token is available")
		}
	})

	t.Run("refresh expired token", func(t *testing.T) {
		// Set an expired token
		expiredToken := &oauth2.Token{
			AccessToken:  "expired-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(-time.Hour), // Expired 1 hour ago
		}
		
		if err := provider.SetToken(expiredToken); err != nil {
			t.Errorf("SetToken() error = %v", err)
			return
		}

		// Note: This will fail with a real OAuth2 server, but tests the code path
		_, err := provider.RefreshToken(ctx)
		// We expect an error since we don't have a real OAuth2 server
		if err == nil {
			t.Error("RefreshToken() should return error without proper OAuth2 server")
		}
	})
}

func TestOAuth2AuthProvider_RefreshIfNeeded(t *testing.T) {
	credStore := NewMockCredentialStore()
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	provider := NewOAuth2AuthProvider(config, credStore, "test-account")
	ctx := context.Background()

	t.Run("refresh if needed with no token", func(t *testing.T) {
		err := provider.RefreshIfNeeded(ctx)
		if err == nil {
			t.Error("RefreshIfNeeded() should return error when no token in store")
		}
	})

	t.Run("refresh if needed with valid token", func(t *testing.T) {
		// Store a valid token in the credential store
		validToken := &OAuthToken{
			AccessToken:  "valid-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(time.Hour), // Valid for 1 hour
		}
		
		if err := credStore.StoreToken(ctx, "test-account", validToken); err != nil {
			t.Errorf("StoreToken() error = %v", err)
			return
		}

		err := provider.RefreshIfNeeded(ctx)
		if err != nil {
			t.Errorf("RefreshIfNeeded() should not error with valid token: %v", err)
		}
	})

	t.Run("refresh if needed with expiring token", func(t *testing.T) {
		// Create a new provider to avoid state from previous tests
		freshProvider := NewOAuth2AuthProvider(config, credStore, "test-account-expiring")
		
		// Set a token that expires soon (within 5 minutes)
		expiringToken := &oauth2.Token{
			AccessToken:  "expiring-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(2 * time.Minute), // Expires in 2 minutes
		}
		
		if err := freshProvider.SetToken(expiringToken); err != nil {
			t.Errorf("SetToken() error = %v", err)
			return
		}

		// Check if the token would be considered as needing refresh
		// The token should be expired/expiring by the 5-minute threshold
		if freshProvider.token.Expiry.Sub(time.Now()) >= 5*time.Minute {
			t.Error("Token should be considered expiring (within 5 minutes)")
		}

		// Test RefreshIfNeeded - it will try to refresh but may fail with mock setup
		// We'll just verify it doesn't panic and handles the expiring token case
		err := freshProvider.RefreshIfNeeded(ctx)
		// The actual result may vary depending on OAuth2 mock behavior
		// The important thing is that it doesn't panic and handles the case
		_ = err // We don't assert on error since mock OAuth2 behavior is unpredictable
	})
}

func TestOAuth2AuthProvider_IsValid(t *testing.T) {
	credStore := NewMockCredentialStore()
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	provider := NewOAuth2AuthProvider(config, credStore, "test-account")
	ctx := context.Background()

	t.Run("invalid with no token anywhere", func(t *testing.T) {
		if provider.IsValid(ctx) {
			t.Error("IsValid() should return false when no token is available")
		}
	})

	t.Run("valid with token from store", func(t *testing.T) {
		// Store a valid token in the credential store
		validToken := &OAuthToken{
			AccessToken:  "valid-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(time.Hour),
		}
		
		if err := credStore.StoreToken(ctx, "test-account", validToken); err != nil {
			t.Errorf("StoreToken() error = %v", err)
			return
		}

		if !provider.IsValid(ctx) {
			t.Error("IsValid() should return true when valid token is in store")
		}
	})
}

func TestOAuth2AuthProvider_ExpiresAt(t *testing.T) {
	credStore := NewMockCredentialStore()
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	provider := NewOAuth2AuthProvider(config, credStore, "test-account")

	t.Run("expires at with no token", func(t *testing.T) {
		expiresAt := provider.ExpiresAt()
		if expiresAt != nil {
			t.Error("ExpiresAt() should return nil when no token is available")
		}
	})

	t.Run("expires at with token", func(t *testing.T) {
		expiry := time.Now().Add(time.Hour)
		token := &oauth2.Token{
			AccessToken:  "test-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       expiry,
		}
		
		if err := provider.SetToken(token); err != nil {
			t.Errorf("SetToken() error = %v", err)
			return
		}

		expiresAt := provider.ExpiresAt()
		if expiresAt == nil {
			t.Error("ExpiresAt() should not return nil when token is available")
		} else if !expiresAt.Equal(expiry) {
			t.Errorf("ExpiresAt() = %v, want %v", expiresAt, expiry)
		}
	})
}

func BenchmarkOAuth2AuthProvider_GetToken(b *testing.B) {
	credStore := NewMockCredentialStore()
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	provider := NewOAuth2AuthProvider(config, credStore, "test-account")
	
	// Set a valid token
	validToken := &oauth2.Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	_ = provider.SetToken(validToken)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GetToken(ctx)
		if err != nil {
			b.Errorf("GetToken() error = %v", err)
		}
	}
}