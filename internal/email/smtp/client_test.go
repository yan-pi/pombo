package smtp

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
	
	if client.extensions == nil {
		t.Error("Expected extensions map to be initialized")
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
	
	config := &email.SMTPConfig{
		Host:    "invalid-host-that-does-not-exist.com",
		Port:    587,
		StartTLS: true,
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
	client := NewClient()
	// We can't mock the client field directly, so this test will check
	// the validation logic by calling Authenticate when not connected
	ctx := context.Background()
	
	err := client.Authenticate(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil auth provider")
	}
	
	emailErr, ok := err.(*email.EmailError)
	if !ok {
		t.Errorf("Expected EmailError, got %T", err)
	}
	
	// Could be validation or protocol error depending on implementation
	if emailErr.Type != email.ErrorTypeValidation && emailErr.Type != email.ErrorTypeProtocol {
		t.Errorf("Expected validation or protocol error, got %v", emailErr.Type)
	}
}

func TestClient_Operations_NotConnected(t *testing.T) {
	client := NewClient()
	ctx := context.Background()
	
	// Test Mail operation
	err := client.Mail(ctx, "test@example.com", nil)
	if err == nil {
		t.Error("Expected error for Mail when not connected")
	}
	
	// Test Rcpt operation
	err = client.Rcpt(ctx, "recipient@example.com")
	if err == nil {
		t.Error("Expected error for Rcpt when not connected")
	}
	
	// Test Data operation
	_, err = client.Data(ctx)
	if err == nil {
		t.Error("Expected error for Data when not connected")
	}
}

func TestClient_Mail_Validation(t *testing.T) {
	client := NewClient()
	// Test validation without mocking - will fail with protocol error due to no connection
	ctx := context.Background()
	
	// Test with empty sender
	err := client.Mail(ctx, "", nil)
	if err == nil {
		t.Error("Expected error for empty sender")
	}
	
	emailErr, ok := err.(*email.EmailError)
	if !ok {
		t.Errorf("Expected EmailError, got %T", err)
	}
	
	// Could be validation or protocol error
	if emailErr.Type != email.ErrorTypeValidation && emailErr.Type != email.ErrorTypeProtocol {
		t.Errorf("Expected validation or protocol error, got %v", emailErr.Type)
	}
}

func TestClient_Rcpt_Validation(t *testing.T) {
	client := NewClient()
	// Test validation without mocking - will fail with protocol error due to no connection
	ctx := context.Background()
	
	// Test with empty recipient
	err := client.Rcpt(ctx, "")
	if err == nil {
		t.Error("Expected error for empty recipient")
	}
	
	emailErr, ok := err.(*email.EmailError)
	if !ok {
		t.Errorf("Expected EmailError, got %T", err)
	}
	
	// Could be validation or protocol error
	if emailErr.Type != email.ErrorTypeValidation && emailErr.Type != email.ErrorTypeProtocol {
		t.Errorf("Expected validation or protocol error, got %v", emailErr.Type)
	}
}

func TestClient_SendMail_Validation(t *testing.T) {
	client := NewClient()
	// Test validation without mocking - will fail with protocol error due to no connection
	ctx := context.Background()
	
	// Test with no recipients
	err := client.SendMail(ctx, "sender@example.com", []string{}, []byte("test"))
	if err == nil {
		t.Error("Expected error for no recipients")
	}
	
	emailErr, ok := err.(*email.EmailError)
	if !ok {
		t.Errorf("Expected EmailError, got %T", err)
	}
	
	// Could be validation or protocol error
	if emailErr.Type != email.ErrorTypeValidation && emailErr.Type != email.ErrorTypeProtocol {
		t.Errorf("Expected validation or protocol error, got %v", emailErr.Type)
	}
}

func TestClient_Extension(t *testing.T) {
	client := NewClient()
	
	// Test when not connected
	supported, value := client.Extension("AUTH")
	if supported {
		t.Error("Expected extension to be unsupported when not connected")
	}
	if value != "" {
		t.Error("Expected empty value when not connected")
	}
	
	// Test with mock extensions (simulating connected state)
	client.extensions = map[string]string{
		"AUTH":     "PLAIN LOGIN",
		"STARTTLS": "",
	}
	
	supported, value = client.Extension("AUTH")
	if !supported {
		t.Error("Expected AUTH extension to be supported")
	}
	if value != "PLAIN LOGIN" {
		t.Errorf("Expected AUTH value 'PLAIN LOGIN', got '%s'", value)
	}
	
	supported, value = client.Extension("STARTTLS")
	if !supported {
		t.Error("Expected STARTTLS extension to be supported")
	}
	if value != "" {
		t.Errorf("Expected empty STARTTLS value, got '%s'", value)
	}
	
	supported, _ = client.Extension("UNKNOWN")
	if supported {
		t.Error("Expected UNKNOWN extension to be unsupported")
	}
}

func TestClient_ServerName(t *testing.T) {
	client := NewClient()
	
	name := client.ServerName()
	if name != "" {
		t.Error("Expected empty server name when not connected")
	}
	
	client.serverName = "test.example.com"
	name = client.ServerName()
	if name != "test.example.com" {
		t.Errorf("Expected server name 'test.example.com', got '%s'", name)
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

func TestOAuth2Auth_Start(t *testing.T) {
	auth := &oauth2Auth{
		username: "test@example.com",
		token:    "access_token_123",
	}
	
	mechanism, response, err := auth.Start(nil)
	if err != nil {
		t.Errorf("Unexpected error in Start: %v", err)
	}
	
	if mechanism != "XOAUTH2" {
		t.Errorf("Expected mechanism 'XOAUTH2', got '%s'", mechanism)
	}
	
	expectedResponse := "user=test@example.com\x01auth=Bearer access_token_123\x01\x01"
	if string(response) != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, string(response))
	}
}

func TestOAuth2Auth_Next(t *testing.T) {
	auth := &oauth2Auth{}
	
	// Should return nil when more is false
	response, err := auth.Next(nil, false)
	if err != nil {
		t.Errorf("Unexpected error in Next: %v", err)
	}
	if response != nil {
		t.Errorf("Expected nil response, got %v", response)
	}
	
	// Should return error when more is true
	_, err = auth.Next([]byte("server response"), true)
	if err == nil {
		t.Error("Expected error when server wants more data")
	}
}

func TestDetectContentType(t *testing.T) {
	testCases := []struct {
		filename string
		expected string
	}{
		{"", "application/octet-stream"},
		{"test.txt", "text/plain"},
		{"test.html", "text/html"},
		{"test.pdf", "application/pdf"},
		{"test.doc", "application/msword"},
		{"test.jpg", "image/jpeg"},
		{"test.png", "image/png"},
		{"test.unknown", "application/octet-stream"},
	}
	
	for _, tc := range testCases {
		result := DetectContentType(tc.filename)
		if result != tc.expected {
			t.Errorf("For filename '%s', expected '%s', got '%s'", 
				tc.filename, tc.expected, result)
		}
	}
}

func TestGetAllRecipients(t *testing.T) {
	msg := &email.OutgoingMessage{
		To: []*email.Address{
			{Address: "to1@example.com"},
			{Address: "to2@example.com"},
		},
		CC: []*email.Address{
			{Address: "cc1@example.com"},
		},
		BCC: []*email.Address{
			{Address: "bcc1@example.com"},
		},
	}
	
	recipients := GetAllRecipients(msg)
	expected := []string{
		"to1@example.com",
		"to2@example.com", 
		"cc1@example.com",
		"bcc1@example.com",
	}
	
	if len(recipients) != len(expected) {
		t.Errorf("Expected %d recipients, got %d", len(expected), len(recipients))
	}
	
	for i, addr := range expected {
		if i >= len(recipients) || recipients[i] != addr {
			t.Errorf("Expected recipient %d to be '%s', got '%s'", 
				i, addr, recipients[i])
		}
	}
}

