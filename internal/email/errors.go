package email

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// EmailError represents a structured email operation error
type EmailError struct {
	Type        ErrorType   `json:"type"`
	Code        string      `json:"code"`
	Message     string      `json:"message"`
	Cause       error       `json:"cause,omitempty"`
	Retryable   bool        `json:"retryable"`
	Account     string      `json:"account,omitempty"`
	Folder      string      `json:"folder,omitempty"`
	MessageID   string      `json:"message_id,omitempty"`
	Operation   string      `json:"operation,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
	Details     interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *EmailError) Error() string {
	var parts []string
	
	if e.Account != "" {
		parts = append(parts, fmt.Sprintf("account=%s", e.Account))
	}
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("op=%s", e.Operation))
	}
	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("code=%s", e.Code))
	}
	
	context := ""
	if len(parts) > 0 {
		context = fmt.Sprintf("[%s] ", strings.Join(parts, " "))
	}
	
	return fmt.Sprintf("%s%s", context, e.Message)
}

// Unwrap returns the underlying error
func (e *EmailError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *EmailError) Is(target error) bool {
	if e == target {
		return true
	}
	
	var emailErr *EmailError
	if errors.As(target, &emailErr) {
		return e.Type == emailErr.Type && e.Code == emailErr.Code
	}
	
	return false
}

// ErrorType represents the category of email error
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeNetwork
	ErrorTypeAuth
	ErrorTypeProtocol
	ErrorTypeQuota
	ErrorTypeServer
	ErrorTypeClient
	ErrorTypeTimeout
	ErrorTypeRateLimit
	ErrorTypeSecurity
	ErrorTypeValidation
	ErrorTypeNotFound
	ErrorTypePermission
	ErrorTypeConfiguration
)

// String returns the string representation of ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeNetwork:
		return "network"
	case ErrorTypeAuth:
		return "authentication"
	case ErrorTypeProtocol:
		return "protocol"
	case ErrorTypeQuota:
		return "quota"
	case ErrorTypeServer:
		return "server"
	case ErrorTypeClient:
		return "client"
	case ErrorTypeTimeout:
		return "timeout"
	case ErrorTypeRateLimit:
		return "rate_limit"
	case ErrorTypeSecurity:
		return "security"
	case ErrorTypeValidation:
		return "validation"
	case ErrorTypeNotFound:
		return "not_found"
	case ErrorTypePermission:
		return "permission"
	case ErrorTypeConfiguration:
		return "configuration"
	default:
		return "unknown"
	}
}

// Predefined error codes
const (
	// Authentication errors
	ErrCodeAuthFailed        = "AUTH_FAILED"
	ErrCodeAuthExpired       = "AUTH_EXPIRED"
	ErrCodeAuthInvalid       = "AUTH_INVALID"
	ErrCodeCredentialsNotFound = "CREDENTIALS_NOT_FOUND"
	ErrCodeOAuthTokenExpired = "OAUTH_TOKEN_EXPIRED"
	ErrCodeOAuthRefreshFailed = "OAUTH_REFRESH_FAILED"
	
	// Connection errors
	ErrCodeConnectionFailed  = "CONNECTION_FAILED"
	ErrCodeConnectionLost    = "CONNECTION_LOST"
	ErrCodeConnectionTimeout = "CONNECTION_TIMEOUT"
	ErrCodeTLSFailed        = "TLS_FAILED"
	ErrCodeDNSResolveFailed = "DNS_RESOLVE_FAILED"
	
	// Protocol errors
	ErrCodeProtocolError    = "PROTOCOL_ERROR"
	ErrCodeUnsupportedOp    = "UNSUPPORTED_OPERATION"
	ErrCodeInvalidResponse  = "INVALID_RESPONSE"
	ErrCodeServerError      = "SERVER_ERROR"
	ErrCodeCommandFailed    = "COMMAND_FAILED"
	
	// Message errors
	ErrCodeMessageNotFound  = "MESSAGE_NOT_FOUND"
	ErrCodeMessageTooLarge  = "MESSAGE_TOO_LARGE"
	ErrCodeInvalidMessage   = "INVALID_MESSAGE"
	ErrCodeAttachmentError  = "ATTACHMENT_ERROR"
	ErrCodeEncodingError    = "ENCODING_ERROR"
	
	// Folder errors
	ErrCodeFolderNotFound   = "FOLDER_NOT_FOUND"
	ErrCodeFolderExists     = "FOLDER_EXISTS"
	ErrCodeFolderReadOnly   = "FOLDER_READ_ONLY"
	ErrCodeInvalidFolder    = "INVALID_FOLDER"
	
	// Quota and limits
	ErrCodeQuotaExceeded    = "QUOTA_EXCEEDED"
	ErrCodeRateLimited      = "RATE_LIMITED"
	ErrCodeTooManyConns     = "TOO_MANY_CONNECTIONS"
	
	// Configuration errors
	ErrCodeInvalidConfig    = "INVALID_CONFIG"
	ErrCodeMissingConfig    = "MISSING_CONFIG"
	ErrCodeConfigLoad       = "CONFIG_LOAD_ERROR"
	
	// Storage errors
	ErrCodeStorageError     = "STORAGE_ERROR"
	ErrCodeCacheError       = "CACHE_ERROR"
	ErrCodeIndexError       = "INDEX_ERROR"
)

// Common error variables
var (
	ErrNotConnected        = NewEmailError(ErrorTypeClient, ErrCodeConnectionFailed, "not connected to email server", nil, false)
	ErrAlreadyConnected    = NewEmailError(ErrorTypeClient, "ALREADY_CONNECTED", "already connected to email server", nil, false)
	ErrConnectionLost      = NewEmailError(ErrorTypeNetwork, ErrCodeConnectionLost, "connection to email server lost", nil, true)
	ErrAuthenticationFailed = NewEmailError(ErrorTypeAuth, ErrCodeAuthFailed, "authentication failed", nil, false)
	ErrInvalidCredentials  = NewEmailError(ErrorTypeAuth, ErrCodeAuthInvalid, "invalid credentials", nil, false)
	ErrTokenExpired        = NewEmailError(ErrorTypeAuth, ErrCodeAuthExpired, "authentication token expired", nil, true)
	ErrFolderNotFound      = NewEmailError(ErrorTypeNotFound, ErrCodeFolderNotFound, "folder not found", nil, false)
	ErrMessageNotFound     = NewEmailError(ErrorTypeNotFound, ErrCodeMessageNotFound, "message not found", nil, false)
	ErrQuotaExceeded       = NewEmailError(ErrorTypeQuota, ErrCodeQuotaExceeded, "storage quota exceeded", nil, false)
	ErrRateLimited         = NewEmailError(ErrorTypeRateLimit, ErrCodeRateLimited, "rate limit exceeded", nil, true)
	ErrInvalidMessage      = NewEmailError(ErrorTypeValidation, ErrCodeInvalidMessage, "invalid message format", nil, false)
	ErrUnsupportedOperation = NewEmailError(ErrorTypeProtocol, ErrCodeUnsupportedOp, "operation not supported by server", nil, false)
)

// NewEmailError creates a new EmailError with the specified parameters
func NewEmailError(errType ErrorType, code, message string, cause error, retryable bool) *EmailError {
	return &EmailError{
		Type:      errType,
		Code:      code,
		Message:   message,
		Cause:     cause,
		Retryable: retryable,
		Timestamp: time.Now(),
	}
}

// WrapError wraps an existing error with additional context
func WrapError(err error, errType ErrorType, code, message string, retryable bool) *EmailError {
	if err == nil {
		return nil
	}
	
	// If it's already an EmailError, add context
	var emailErr *EmailError
	if errors.As(err, &emailErr) {
		return &EmailError{
			Type:      errType,
			Code:      code,
			Message:   message + ": " + emailErr.Message,
			Cause:     emailErr,
			Retryable: retryable && emailErr.Retryable,
			Account:   emailErr.Account,
			Folder:    emailErr.Folder,
			MessageID: emailErr.MessageID,
			Operation: emailErr.Operation,
			Timestamp: time.Now(),
		}
	}
	
	return &EmailError{
		Type:      errType,
		Code:      code,
		Message:   message,
		Cause:     err,
		Retryable: retryable,
		Timestamp: time.Now(),
	}
}

// WithContext adds context information to an error
func (e *EmailError) WithContext(account, folder, messageID, operation string) *EmailError {
	return &EmailError{
		Type:      e.Type,
		Code:      e.Code,
		Message:   e.Message,
		Cause:     e.Cause,
		Retryable: e.Retryable,
		Account:   account,
		Folder:    folder,
		MessageID: messageID,
		Operation: operation,
		Timestamp: e.Timestamp,
		Details:   e.Details,
	}
}

// WithDetails adds additional details to an error
func (e *EmailError) WithDetails(details interface{}) *EmailError {
	return &EmailError{
		Type:      e.Type,
		Code:      e.Code,
		Message:   e.Message,
		Cause:     e.Cause,
		Retryable: e.Retryable,
		Account:   e.Account,
		Folder:    e.Folder,
		MessageID: e.MessageID,
		Operation: e.Operation,
		Timestamp: e.Timestamp,
		Details:   details,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var emailErr *EmailError
	if errors.As(err, &emailErr) {
		return emailErr.Retryable
	}
	
	// Check for common retryable error types
	if isNetworkError(err) || isTimeoutError(err) {
		return true
	}
	
	return false
}

// IsTemporary checks if an error is temporary
func IsTemporary(err error) bool {
	type temporary interface {
		Temporary() bool
	}
	
	if t, ok := err.(temporary); ok {
		return t.Temporary()
	}
	
	return IsRetryable(err)
}

// GetErrorType returns the error type for any error
func GetErrorType(err error) ErrorType {
	var emailErr *EmailError
	if errors.As(err, &emailErr) {
		return emailErr.Type
	}
	
	// Classify common error types
	if isNetworkError(err) {
		return ErrorTypeNetwork
	}
	if isTimeoutError(err) {
		return ErrorTypeTimeout
	}
	if isAuthError(err) {
		return ErrorTypeAuth
	}
	
	return ErrorTypeUnknown
}

// Helper functions for error classification

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	
	// Check for common network error strings
	errStr := strings.ToLower(err.Error())
	networkKeywords := []string{
		"connection refused",
		"connection reset",
		"network unreachable",
		"host unreachable",
		"no route to host",
	}
	
	for _, keyword := range networkKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}
	
	return false
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	
	errStr := strings.ToLower(err.Error())
	timeoutKeywords := []string{
		"timeout",
		"deadline exceeded",
		"i/o timeout",
		"connection timeout",
	}
	
	for _, keyword := range timeoutKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}
	
	return false
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := strings.ToLower(err.Error())
	authKeywords := []string{
		"authentication failed",
		"invalid credentials",
		"login failed",
		"unauthorized",
		"access denied",
		"permission denied",
	}
	
	for _, keyword := range authKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}
	
	return false
}

// IsAuthError checks if an error is authentication-related
func IsAuthError(err error) bool {
	var emailErr *EmailError
	if errors.As(err, &emailErr) {
		return emailErr.Type == ErrorTypeAuth
	}
	
	return isAuthError(err)
}

// IsNetworkError checks if an error is network-related
func IsNetworkError(err error) bool {
	var emailErr *EmailError
	if errors.As(err, &emailErr) {
		return emailErr.Type == ErrorTypeNetwork
	}
	
	return isNetworkError(err)
}

// IsTimeoutError checks if an error is timeout-related
func IsTimeoutError(err error) bool {
	var emailErr *EmailError
	if errors.As(err, &emailErr) {
		return emailErr.Type == ErrorTypeTimeout
	}
	
	return isTimeoutError(err)
}

// ErrorHandler provides centralized error handling logic
type ErrorHandler struct {
	maxRetries     int
	baseDelay      time.Duration
	maxDelay       time.Duration
	multiplier     float64
	jitterEnabled  bool
}

// NewErrorHandler creates a new error handler with default settings
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		maxRetries:    3,
		baseDelay:     time.Second,
		maxDelay:      time.Minute,
		multiplier:    2.0,
		jitterEnabled: true,
	}
}

// ShouldRetry determines if an operation should be retried
func (h *ErrorHandler) ShouldRetry(err error, attempt int) bool {
	if attempt >= h.maxRetries {
		return false
	}
	
	return IsRetryable(err)
}

// GetRetryDelay calculates the delay before the next retry
func (h *ErrorHandler) GetRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	
	// Exponential backoff
	delay := time.Duration(float64(h.baseDelay) * (h.multiplier * float64(attempt)))
	
	// Add jitter to prevent thundering herd
	if h.jitterEnabled {
		jitterFactor := float64(time.Now().UnixNano()%1000) / 1000.0 * 0.2 - 0.1 // -0.1 to +0.1
		jitter := time.Duration(float64(delay) * jitterFactor)
		delay += jitter
	}
	
	// Apply max delay after jitter
	if delay > h.maxDelay {
		delay = h.maxDelay
	}
	
	return delay
}

// Handle processes an error and returns appropriate actions
func (h *ErrorHandler) Handle(err error, attempt int) ErrorAction {
	if err == nil {
		return ErrorActionNone
	}
	
	var emailErr *EmailError
	if !errors.As(err, &emailErr) {
		// Convert to EmailError for consistent handling
		emailErr = WrapError(err, GetErrorType(err), "UNKNOWN_ERROR", err.Error(), IsRetryable(err))
	}
	
	// Determine action based on error type
	switch emailErr.Type {
	case ErrorTypeAuth:
		if emailErr.Code == ErrCodeAuthExpired || emailErr.Code == ErrCodeOAuthTokenExpired {
			return ErrorActionRefreshAuth
		}
		return ErrorActionReauth
	
	case ErrorTypeNetwork, ErrorTypeTimeout:
		if h.ShouldRetry(err, attempt) {
			return ErrorActionRetry
		}
		return ErrorActionReconnect
	
	case ErrorTypeRateLimit:
		return ErrorActionBackoff
	
	case ErrorTypeQuota:
		return ErrorActionUserAction
	
	case ErrorTypeConfiguration:
		return ErrorActionUserAction
	
	default:
		if h.ShouldRetry(err, attempt) {
			return ErrorActionRetry
		}
		return ErrorActionFail
	}
}

// ErrorAction represents the recommended action for handling an error
type ErrorAction int

const (
	ErrorActionNone ErrorAction = iota
	ErrorActionRetry
	ErrorActionReconnect
	ErrorActionReauth
	ErrorActionRefreshAuth
	ErrorActionBackoff
	ErrorActionUserAction
	ErrorActionFail
)

// String returns the string representation of ErrorAction
func (ea ErrorAction) String() string {
	switch ea {
	case ErrorActionNone:
		return "none"
	case ErrorActionRetry:
		return "retry"
	case ErrorActionReconnect:
		return "reconnect"
	case ErrorActionReauth:
		return "reauth"
	case ErrorActionRefreshAuth:
		return "refresh_auth"
	case ErrorActionBackoff:
		return "backoff"
	case ErrorActionUserAction:
		return "user_action"
	case ErrorActionFail:
		return "fail"
	default:
		return "unknown"
	}
}