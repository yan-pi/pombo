package email

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/ybarbara/pombo/internal/config"
)

func TestEmailError(t *testing.T) {
	tests := []struct {
		name        string
		err         *EmailError
		wantString  string
		wantRetryable bool
		wantType    ErrorType
	}{
		{
			name: "basic error",
			err: &EmailError{
				Type:    ErrorTypeNetwork,
				Code:    ErrCodeConnectionFailed,
				Message: "connection failed",
			},
			wantString:    "[code=CONNECTION_FAILED] connection failed",
			wantRetryable: false,
			wantType:      ErrorTypeNetwork,
		},
		{
			name: "error with context",
			err: &EmailError{
				Type:      ErrorTypeAuth,
				Code:      ErrCodeAuthFailed,
				Message:   "authentication failed",
				Account:   "test@example.com",
				Operation: "connect",
			},
			wantString:    "[account=test@example.com op=connect code=AUTH_FAILED] authentication failed",
			wantRetryable: false,
			wantType:      ErrorTypeAuth,
		},
		{
			name: "retryable error",
			err: &EmailError{
				Type:      ErrorTypeNetwork,
				Code:      ErrCodeConnectionTimeout,
				Message:   "connection timeout",
				Retryable: true,
			},
			wantString:    "[code=CONNECTION_TIMEOUT] connection timeout",
			wantRetryable: true,
			wantType:      ErrorTypeNetwork,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Error() method
			if got := tt.err.Error(); got != tt.wantString {
				t.Errorf("Error() = %v, want %v", got, tt.wantString)
			}

			// Test IsRetryable
			if got := IsRetryable(tt.err); got != tt.wantRetryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.wantRetryable)
			}

			// Test GetErrorType
			if got := GetErrorType(tt.err); got != tt.wantType {
				t.Errorf("GetErrorType() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestNewEmailError(t *testing.T) {
	err := NewEmailError(ErrorTypeAuth, ErrCodeAuthFailed, "test error", nil, true)

	if err.Type != ErrorTypeAuth {
		t.Errorf("Type = %v, want %v", err.Type, ErrorTypeAuth)
	}

	if err.Code != ErrCodeAuthFailed {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeAuthFailed)
	}

	if err.Message != "test error" {
		t.Errorf("Message = %v, want %v", err.Message, "test error")
	}

	if !err.Retryable {
		t.Error("Retryable should be true")
	}

	if err.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := WrapError(originalErr, ErrorTypeNetwork, ErrCodeConnectionFailed, "connection failed", true)

	if wrappedErr == nil {
		t.Fatal("WrapError should not return nil")
	}

	if wrappedErr.Type != ErrorTypeNetwork {
		t.Errorf("Type = %v, want %v", wrappedErr.Type, ErrorTypeNetwork)
	}

	if wrappedErr.Code != ErrCodeConnectionFailed {
		t.Errorf("Code = %v, want %v", wrappedErr.Code, ErrCodeConnectionFailed)
	}

	if wrappedErr.Cause != originalErr {
		t.Errorf("Cause = %v, want %v", wrappedErr.Cause, originalErr)
	}

	if !wrappedErr.Retryable {
		t.Error("Retryable should be true")
	}

	// Test wrapping nil error
	if nilWrapped := WrapError(nil, ErrorTypeNetwork, "TEST", "test", false); nilWrapped != nil {
		t.Error("WrapError(nil) should return nil")
	}
}

func TestEmailErrorWithContext(t *testing.T) {
	originalErr := &EmailError{
		Type:    ErrorTypeProtocol,
		Code:    ErrCodeServerError,
		Message: "server error",
	}

	contextErr := originalErr.WithContext("test@example.com", "INBOX", "123", "fetch")

	if contextErr.Account != "test@example.com" {
		t.Errorf("Account = %v, want %v", contextErr.Account, "test@example.com")
	}

	if contextErr.Folder != "INBOX" {
		t.Errorf("Folder = %v, want %v", contextErr.Folder, "INBOX")
	}

	if contextErr.MessageID != "123" {
		t.Errorf("MessageID = %v, want %v", contextErr.MessageID, "123")
	}

	if contextErr.Operation != "fetch" {
		t.Errorf("Operation = %v, want %v", contextErr.Operation, "fetch")
	}

	// Original error should be unchanged
	if originalErr.Account != "" {
		t.Error("Original error should not be modified")
	}
}

func TestEmailErrorWithDetails(t *testing.T) {
	originalErr := &EmailError{
		Type:    ErrorTypeProtocol,
		Code:    ErrCodeServerError,
		Message: "server error",
	}

	details := map[string]interface{}{
		"server_response": "Internal Server Error",
		"status_code":     500,
	}

	detailErr := originalErr.WithDetails(details)

	if detailErr.Details == nil {
		t.Error("Details should not be nil")
	}

	// Check specific detail values
	if detailsMap, ok := detailErr.Details.(map[string]interface{}); ok {
		if serverResponse, exists := detailsMap["server_response"]; !exists || serverResponse != "Internal Server Error" {
			t.Error("server_response should match provided value")
		}

		if statusCode, exists := detailsMap["status_code"]; !exists || statusCode != 500 {
			t.Error("status_code should match provided value")
		}
	} else {
		t.Error("Details should be a map[string]interface{}")
	}

	// Original error should be unchanged
	if originalErr.Details != nil {
		t.Error("Original error should not be modified")
	}
}

func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantType ErrorType
		wantAuth bool
		wantNetwork bool
		wantTimeout bool
		wantRetryable bool
	}{
		{
			name:     "authentication error",
			err:      errors.New("authentication failed"),
			wantType: ErrorTypeAuth,
			wantAuth: true,
		},
		{
			name:     "network error",
			err:      errors.New("connection refused"),
			wantType: ErrorTypeNetwork,
			wantNetwork: true,
			wantRetryable: true,
		},
		{
			name:     "timeout error",
			err:      errors.New("connection timeout"),
			wantType: ErrorTypeTimeout,
			wantTimeout: true,
			wantRetryable: true,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			wantType: ErrorTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorType(tt.err); got != tt.wantType {
				t.Errorf("GetErrorType() = %v, want %v", got, tt.wantType)
			}

			if got := IsAuthError(tt.err); got != tt.wantAuth {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.wantAuth)
			}

			if got := IsNetworkError(tt.err); got != tt.wantNetwork {
				t.Errorf("IsNetworkError() = %v, want %v", got, tt.wantNetwork)
			}

			if got := IsTimeoutError(tt.err); got != tt.wantTimeout {
				t.Errorf("IsTimeoutError() = %v, want %v", got, tt.wantTimeout)
			}

			if got := IsRetryable(tt.err); got != tt.wantRetryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.wantRetryable)
			}
		})
	}
}

func TestErrorHandler(t *testing.T) {
	handler := NewErrorHandler()

	if handler.maxRetries != 3 {
		t.Errorf("maxRetries = %v, want %v", handler.maxRetries, 3)
	}

	if handler.baseDelay != time.Second {
		t.Errorf("baseDelay = %v, want %v", handler.baseDelay, time.Second)
	}

	if handler.multiplier != 2.0 {
		t.Errorf("multiplier = %v, want %v", handler.multiplier, 2.0)
	}
}

func TestErrorHandlerShouldRetry(t *testing.T) {
	handler := NewErrorHandler()

	tests := []struct {
		name    string
		err     error
		attempt int
		want    bool
	}{
		{
			name:    "retryable error, first attempt",
			err:     NewEmailError(ErrorTypeNetwork, ErrCodeConnectionFailed, "connection failed", nil, true),
			attempt: 1,
			want:    true,
		},
		{
			name:    "retryable error, max attempts reached",
			err:     NewEmailError(ErrorTypeNetwork, ErrCodeConnectionFailed, "connection failed", nil, true),
			attempt: 3,
			want:    false,
		},
		{
			name:    "non-retryable error",
			err:     NewEmailError(ErrorTypeAuth, ErrCodeAuthFailed, "auth failed", nil, false),
			attempt: 1,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.ShouldRetry(tt.err, tt.attempt); got != tt.want {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorHandlerGetRetryDelay(t *testing.T) {
	handler := NewErrorHandler()

	// Test delay calculation
	delay1 := handler.GetRetryDelay(1)
	delay2 := handler.GetRetryDelay(2)

	if delay1 <= 0 {
		t.Error("First retry delay should be positive")
	}

	if delay2 <= delay1 {
		t.Error("Second retry delay should be larger than first")
	}

	// Test maximum delay cap
	delay10 := handler.GetRetryDelay(10)
	if delay10 > handler.maxDelay {
		t.Errorf("Delay should not exceed maxDelay: %v > %v", delay10, handler.maxDelay)
	}

	// Test zero attempt
	delay0 := handler.GetRetryDelay(0)
	if delay0 != 0 {
		t.Errorf("Zero attempt delay = %v, want 0", delay0)
	}
}

func TestErrorHandlerHandle(t *testing.T) {
	handler := NewErrorHandler()

	tests := []struct {
		name    string
		err     error
		attempt int
		want    ErrorAction
	}{
		{
			name:    "nil error",
			err:     nil,
			attempt: 1,
			want:    ErrorActionNone,
		},
		{
			name:    "auth expired error",
			err:     NewEmailError(ErrorTypeAuth, ErrCodeAuthExpired, "auth expired", nil, true),
			attempt: 1,
			want:    ErrorActionRefreshAuth,
		},
		{
			name:    "auth failed error",
			err:     NewEmailError(ErrorTypeAuth, ErrCodeAuthFailed, "auth failed", nil, false),
			attempt: 1,
			want:    ErrorActionReauth,
		},
		{
			name:    "network error - retryable",
			err:     NewEmailError(ErrorTypeNetwork, ErrCodeConnectionFailed, "connection failed", nil, true),
			attempt: 1,
			want:    ErrorActionRetry,
		},
		{
			name:    "network error - max retries",
			err:     NewEmailError(ErrorTypeNetwork, ErrCodeConnectionFailed, "connection failed", nil, true),
			attempt: 3,
			want:    ErrorActionReconnect,
		},
		{
			name:    "rate limit error",
			err:     NewEmailError(ErrorTypeRateLimit, ErrCodeRateLimited, "rate limited", nil, true),
			attempt: 1,
			want:    ErrorActionBackoff,
		},
		{
			name:    "quota error",
			err:     NewEmailError(ErrorTypeQuota, ErrCodeQuotaExceeded, "quota exceeded", nil, false),
			attempt: 1,
			want:    ErrorActionUserAction,
		},
		{
			name:    "unknown error - retryable",
			err:     NewEmailError(ErrorTypeUnknown, "UNKNOWN", "unknown error", nil, true),
			attempt: 1,
			want:    ErrorActionRetry,
		},
		{
			name:    "unknown error - not retryable",
			err:     NewEmailError(ErrorTypeUnknown, "UNKNOWN", "unknown error", nil, false),
			attempt: 1,
			want:    ErrorActionFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.Handle(tt.err, tt.attempt); got != tt.want {
				t.Errorf("Handle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorTypeString(t *testing.T) {
	tests := []struct {
		errType ErrorType
		want    string
	}{
		{ErrorTypeNetwork, "network"},
		{ErrorTypeAuth, "authentication"},
		{ErrorTypeProtocol, "protocol"},
		{ErrorTypeQuota, "quota"},
		{ErrorTypeServer, "server"},
		{ErrorTypeClient, "client"},
		{ErrorTypeTimeout, "timeout"},
		{ErrorTypeRateLimit, "rate_limit"},
		{ErrorTypeSecurity, "security"},
		{ErrorTypeValidation, "validation"},
		{ErrorTypeNotFound, "not_found"},
		{ErrorTypePermission, "permission"},
		{ErrorTypeConfiguration, "configuration"},
		{ErrorTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.errType.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorActionString(t *testing.T) {
	tests := []struct {
		action ErrorAction
		want   string
	}{
		{ErrorActionNone, "none"},
		{ErrorActionRetry, "retry"},
		{ErrorActionReconnect, "reconnect"},
		{ErrorActionReauth, "reauth"},
		{ErrorActionRefreshAuth, "refresh_auth"},
		{ErrorActionBackoff, "backoff"},
		{ErrorActionUserAction, "user_action"},
		{ErrorActionFail, "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.action.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorIs(t *testing.T) {
	err1 := &EmailError{Type: ErrorTypeAuth, Code: ErrCodeAuthFailed}
	err2 := &EmailError{Type: ErrorTypeAuth, Code: ErrCodeAuthFailed}
	err3 := &EmailError{Type: ErrorTypeAuth, Code: ErrCodeAuthExpired}

	if !errors.Is(err1, err2) {
		t.Error("errors with same type and code should be equal")
	}

	if errors.Is(err1, err3) {
		t.Error("errors with different codes should not be equal")
	}
}

func TestErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original")
	wrappedErr := WrapError(originalErr, ErrorTypeNetwork, "TEST", "wrapped", false)

	if unwrapped := errors.Unwrap(wrappedErr); unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

// Test with actual network error types
func TestNetworkErrorDetection(t *testing.T) {
	// Create a mock network error
	netErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("connection refused"),
	}

	if !IsNetworkError(netErr) {
		t.Error("Should detect net.OpError as network error")
	}

	if !IsRetryable(netErr) {
		t.Error("Network errors should be retryable")
	}
}

// Benchmark tests
func BenchmarkEmailError_Error(b *testing.B) {
	err := &EmailError{
		Type:      ErrorTypeNetwork,
		Code:      ErrCodeConnectionFailed,
		Message:   "connection failed",
		Account:   "test@example.com",
		Operation: "connect",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkIsRetryable(b *testing.B) {
	err := NewEmailError(ErrorTypeNetwork, ErrCodeConnectionFailed, "connection failed", nil, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsRetryable(err)
	}
}

func TestIsTemporary(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "temporary email error",
			err:  NewEmailError(ErrorTypeNetwork, ErrCodeConnectionFailed, "connection failed", nil, true),
			want: true,
		},
		{
			name: "non-temporary email error",
			err:  NewEmailError(ErrorTypeAuth, ErrCodeAuthFailed, "auth failed", nil, false),
			want: false,
		},
		{
			name: "temporary network error",
			err:  errors.New("connection timeout"),
			want: true,
		},
		{
			name: "non-temporary error",
			err:  errors.New("invalid argument"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTemporary(tt.err); got != tt.want {
				t.Errorf("IsTemporary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthProviderFactory_CreateProvider_EdgeCases(t *testing.T) {
	credStore := NewMockCredentialStore()
	factory := NewAuthProviderFactory(credStore)
	ctx := context.Background()

	t.Run("oauth2 with missing client secret", func(t *testing.T) {
		account := &config.AccountConfig{
			ID:    "test-oauth-missing",
			Name:  "Test OAuth Missing",
			Email: "test@gmail.com",
			OAuth: &config.OAuthConfig{
				Provider:    "gmail",
				ClientID:    "test-client-id",
				// Missing ClientSecret
				RedirectURI: "http://localhost:8080/callback",
				Scopes:      []string{"email"},
				AuthURL:     "https://accounts.google.com/o/oauth2/auth",
				TokenURL:    "https://oauth2.googleapis.com/token",
			},
		}

		provider, err := factory.CreateProvider(ctx, account)
		if err != nil {
			t.Errorf("CreateProvider() should not error with missing client secret: %v", err)
			return
		}

		if provider.Type() != AuthTypeOAuth2 {
			t.Errorf("Provider type = %v, want %v", provider.Type(), AuthTypeOAuth2)
		}
	})

	t.Run("basic auth with only SMTP password", func(t *testing.T) {
		account := &config.AccountConfig{
			ID:    "test-smtp-only",
			Name:  "Test SMTP Only",
			Email: "test@example.com",
			IMAP: config.IMAPConfig{
				Username: "test@example.com",
				// No Password
			},
			SMTP: config.SMTPConfig{
				Password: "smtp-password",
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

		creds, err := provider.GetCredentials(ctx)
		if err != nil {
			t.Errorf("GetCredentials() error = %v", err)
			return
		}

		if creds.Password != "smtp-password" {
			t.Errorf("Password = %v, want %v", creds.Password, "smtp-password")
		}
	})

	t.Run("basic auth with partial credentials from config and store", func(t *testing.T) {
		// Store partial credentials in mock store
		creds := &Credentials{
			Type:     AuthTypePassword,
			Username: "stored@example.com",
			Password: "stored-password",
		}
		credStore.Store(ctx, "test-partial", creds)

		account := &config.AccountConfig{
			ID:    "test-partial",
			Name:  "Test Partial",
			Email: "test@example.com",
			IMAP: config.IMAPConfig{
				Username: "config-user", // Username from config
				// No password in config, should use store
			},
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

		// Should use stored credentials when config is incomplete
		if retrievedCreds.Username != "stored@example.com" {
			t.Errorf("Username = %v, want %v", retrievedCreds.Username, "stored@example.com")
		}
	})
}

func TestErrorEdgeCases(t *testing.T) {
	t.Run("error Is with different types", func(t *testing.T) {
		emailErr := &EmailError{Type: ErrorTypeAuth, Code: ErrCodeAuthFailed}
		otherErr := errors.New("other error")

		if errors.Is(emailErr, otherErr) {
			t.Error("EmailError should not be equal to different error type")
		}
	})

	t.Run("wrap nil error", func(t *testing.T) {
		wrapped := WrapError(nil, ErrorTypeNetwork, "TEST", "test", false)
		if wrapped != nil {
			t.Error("WrapError(nil) should return nil")
		}
	})

	t.Run("timeout error with net.Error interface", func(t *testing.T) {
		// Create a custom error that implements net.Error with Timeout() = true
		timeoutErr := &customTimeoutError{msg: "custom timeout"}
		
		if !IsTimeoutError(timeoutErr) {
			t.Error("Should detect custom timeout error")
		}

		if !isTimeoutError(timeoutErr) {
			t.Error("Should detect custom timeout error in helper")
		}
	})

	t.Run("error classification edge cases", func(t *testing.T) {
		tests := []struct {
			name     string
			err      error
			wantAuth bool
			wantNet  bool
			wantTime bool
		}{
			{
				name:     "mixed case authentication",
				err:      errors.New("Authentication Failed"),
				wantAuth: true,
			},
			{
				name:    "mixed case network",
				err:     errors.New("Connection Refused"),
				wantNet: true,
			},
			{
				name:     "mixed case timeout",
				err:      errors.New("Connection Timeout"),
				wantTime: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := IsAuthError(tt.err); got != tt.wantAuth {
					t.Errorf("IsAuthError() = %v, want %v", got, tt.wantAuth)
				}
				if got := IsNetworkError(tt.err); got != tt.wantNet {
					t.Errorf("IsNetworkError() = %v, want %v", got, tt.wantNet)
				}
				if got := IsTimeoutError(tt.err); got != tt.wantTime {
					t.Errorf("IsTimeoutError() = %v, want %v", got, tt.wantTime)
				}
			})
		}
	})
}

// Custom error type for testing net.Error interface
type customTimeoutError struct {
	msg string
}

func (e *customTimeoutError) Error() string {
	return e.msg
}

func (e *customTimeoutError) Timeout() bool {
	return true
}

func (e *customTimeoutError) Temporary() bool {
	return true
}

func TestErrorHandler_GetRetryDelay_EdgeCases(t *testing.T) {
	handler := NewErrorHandler()

	t.Run("negative attempt", func(t *testing.T) {
		delay := handler.GetRetryDelay(-1)
		if delay != 0 {
			t.Errorf("GetRetryDelay(-1) = %v, want 0", delay)
		}
	})

	t.Run("very large attempt", func(t *testing.T) {
		delay := handler.GetRetryDelay(100)
		if delay > handler.maxDelay {
			t.Errorf("GetRetryDelay(100) = %v, should not exceed maxDelay %v", delay, handler.maxDelay)
		}
	})
}

func TestErrorHandler_Handle_EdgeCases(t *testing.T) {
	handler := NewErrorHandler()

	t.Run("unknown error type validation", func(t *testing.T) {
		// Create an error with unknown auth type
		unknownErr := NewEmailError(ErrorTypeValidation, "UNKNOWN_CODE", "unknown error", nil, false)
		action := handler.Handle(unknownErr, 1)
		
		// Should fall through to final condition
		if action != ErrorActionFail {
			t.Errorf("Handle() = %v, want %v for unknown validation error", action, ErrorActionFail)
		}
	})
}

func BenchmarkErrorHandler_Handle(b *testing.B) {
	handler := NewErrorHandler()
	err := NewEmailError(ErrorTypeNetwork, ErrCodeConnectionFailed, "connection failed", nil, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.Handle(err, 1)
	}
}