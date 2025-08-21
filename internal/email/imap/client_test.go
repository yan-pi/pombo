package imap

import (
	"context"
	"testing"
	"time"

	"github.com/ybarbara/pombo/internal/email"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	
	if client.State() != email.StateDisconnected {
		t.Errorf("Expected initial state to be disconnected, got %v", client.State())
	}
	
	if client.IsIdleSupported() {
		t.Error("Expected IDLE to be unsupported initially")
	}
}

func TestClient_Connect_ValidationError(t *testing.T) {
	client := NewClient()
	ctx := context.Background()
	
	// Test with nil config
	err := client.Connect(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
	
	emailErr, ok := err.(*email.EmailError)
	if !ok {
		t.Errorf("Expected EmailError, got %T", err)
	}
	
	if emailErr.Type != email.ErrorTypeValidation {
		t.Errorf("Expected validation error, got %v", emailErr.Type)
	}
}

func TestClient_Connect_InvalidHost(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	config := &email.IMAPConfig{
		Host:    "invalid-host-that-does-not-exist.com",
		Port:    993,
		TLS:     true,
		Timeout: 1 * time.Second,
	}
	
	err := client.Connect(ctx, config)
	if err == nil {
		t.Error("Expected error for invalid host")
		client.Close()
		return
	}
	
	emailErr, ok := err.(*email.EmailError)
	if !ok {
		t.Errorf("Expected EmailError, got %T", err)
	}
	
	if emailErr.Type != email.ErrorTypeNetwork && emailErr.Type != email.ErrorTypeTimeout {
		t.Errorf("Expected network or timeout error, got %v", emailErr.Type)
	}
}

func TestClient_Authenticate_NotConnected(t *testing.T) {
	client := NewClient()
	ctx := context.Background()
	
	auth := email.NewBasicAuthProvider("test@example.com", "password")
	
	err := client.Authenticate(ctx, auth)
	if err == nil {
		t.Error("Expected error when not connected")
	}
	
	emailErr, ok := err.(*email.EmailError)
	if !ok {
		t.Errorf("Expected EmailError, got %T", err)
	}
	
	if emailErr.Type != email.ErrorTypeProtocol {
		t.Errorf("Expected protocol error, got %v", emailErr.Type)
	}
}

func TestClient_Authenticate_NilAuth(t *testing.T) {
	// This test would require a mock connection
	// For now, we test the validation logic
	client := NewClient()
	client.state = email.StateConnected // Simulate connected state
	ctx := context.Background()
	
	err := client.Authenticate(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil auth provider")
	}
	
	emailErr, ok := err.(*email.EmailError)
	if !ok {
		t.Errorf("Expected EmailError, got %T", err)
	}
	
	if emailErr.Type != email.ErrorTypeValidation {
		t.Errorf("Expected validation error, got %v", emailErr.Type)
	}
}

func TestClient_Operations_NotAuthenticated(t *testing.T) {
	client := NewClient()
	ctx := context.Background()
	
	// Test List operation
	_, err := client.List(ctx, "", "*")
	if err == nil {
		t.Error("Expected error for List when not authenticated")
	}
	
	// Test Select operation
	_, err = client.Select(ctx, "INBOX")
	if err == nil {
		t.Error("Expected error for Select when not authenticated")
	}
	
	// Test Search operation
	_, err = client.Search(ctx, &email.SearchCriteria{})
	if err == nil {
		t.Error("Expected error for Search when not selected")
	}
}

func TestClient_ServerInfo(t *testing.T) {
	client := NewClient()
	
	info := client.ServerInfo()
	if info != nil {
		t.Error("Expected nil server info when not connected")
	}
}

func TestClient_State(t *testing.T) {
	client := NewClient()
	
	if client.State() != email.StateDisconnected {
		t.Errorf("Expected disconnected state, got %v", client.State())
	}
}

func TestClient_Close(t *testing.T) {
	client := NewClient()
	
	// Should not error when closing unconnected client
	err := client.Close()
	if err != nil {
		t.Errorf("Unexpected error closing unconnected client: %v", err)
	}
}

func TestClient_SearchCriteria(t *testing.T) {
	client := NewClient()
	
	// Test buildSearchCriteria with nil criteria
	criteria := client.buildSearchCriteria(nil)
	if criteria == nil {
		t.Error("Expected non-nil search criteria")
	}
	
	// Test buildSearchCriteria with populated criteria
	searchCriteria := &email.SearchCriteria{
		Query:   "test",
		From:    "sender@example.com",
		To:      "recipient@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
		HasFlag: []string{email.FlagSeen},
		NotFlag: []string{email.FlagDeleted},
	}
	
	imapCriteria := client.buildSearchCriteria(searchCriteria)
	if len(imapCriteria.Text) == 0 || imapCriteria.Text[0] != "test" {
		t.Error("Expected text search criteria to be set")
	}
	
	// Check header criteria (From is set via Header.Set in go-imap v1)
	fromHeader := imapCriteria.Header.Get("From")
	if fromHeader != "sender@example.com" {
		t.Error("Expected from criteria to be set")
	}
}

func TestClient_FetchItems(t *testing.T) {
	client := NewClient()
	
	// Test buildFetchItems with empty items
	items := client.buildFetchItems([]string{})
	// Check that default items are present
	if len(items) == 0 {
		t.Error("Expected default fetch items to be set")
	}
	// Verify some common default items are included
	hasEnvelope := false
	hasFlags := false
	for _, item := range items {
		if item == "ENVELOPE" {
			hasEnvelope = true
		}
		if item == "FLAGS" {
			hasFlags = true
		}
	}
	if !hasEnvelope || !hasFlags {
		t.Error("Expected default fetch items to include ENVELOPE and FLAGS")
	}
	
	// Test buildFetchItems with specific items
	items = client.buildFetchItems([]string{"ENVELOPE", "FLAGS", "UID"})
	// Check that the expected FetchItems are present
	hasEnvelope = false
	hasFlags = false
	hasUID := false
	for _, item := range items {
		switch item {
		case "ENVELOPE":
			hasEnvelope = true
		case "FLAGS":
			hasFlags = true
		case "UID":
			hasUID = true
		}
	}
	if !hasEnvelope || !hasFlags || !hasUID {
		t.Error("Expected specified fetch items to be set")
	}
}

func TestClient_AddressConversion(t *testing.T) {
	client := NewClient()
	
	// Test convertAddress with nil
	addr := client.convertAddress(nil)
	if addr != nil {
		t.Error("Expected nil address for nil input")
	}
	
	// Note: Testing with actual IMAP address would require importing go-imap types
	// This would be done in integration tests
}

func TestClient_ErrorHandling(t *testing.T) {
	client := NewClient()
	client.config = &email.IMAPConfig{
		Host: "test.example.com",
		Port: 993,
	}
	
	// Test connection error wrapping
	err := client.wrapConnectionError(nil)
	if err != nil {
		t.Error("Expected nil for nil error")
	}
	
	// Test auth error wrapping
	err = client.wrapAuthError(nil)
	if err != nil {
		t.Error("Expected nil for nil error")
	}
}

func TestClient_IdleSupport(t *testing.T) {
	client := NewClient()
	
	// Initially should not support IDLE
	if client.IsIdleSupported() {
		t.Error("Expected IDLE to be unsupported initially")
	}
	
	// Test StopIdle (should not panic)
	client.StopIdle()
}

// Integration test helpers and mocks would go here
// For real integration tests, you would need actual IMAP server connections

func TestClient_MonitorUpdates_InvalidInterval(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	updates := make(chan *email.EmailUpdate, 1)
	
	// Test with very short interval (should be adjusted to minimum)
	// Since client is not authenticated, this should return an authentication error
	err := client.MonitorUpdates(ctx, "INBOX", updates, 1*time.Millisecond)
	if err == nil {
		t.Error("Expected error when not authenticated")
		return
	}
	
	emailErr, ok := err.(*email.EmailError)
	if !ok {
		t.Errorf("Expected EmailError, got %T", err)
		return
	}
	
	if emailErr.Type != email.ErrorTypeProtocol {
		t.Errorf("Expected protocol error, got %v", emailErr.Type)
	}
}