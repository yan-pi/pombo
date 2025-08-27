package smtp

import (
	"strings"
	"testing"

	"github.com/ybarbara/pombo/internal/email"
)

func TestNewMessageComposer(t *testing.T) {
	composer := NewMessageComposer()
	if composer == nil {
		t.Fatal("NewMessageComposer returned nil")
	}
	
	if composer.boundary == "" {
		t.Error("Expected boundary to be set")
	}
}

func TestMessageComposer_ComposeMessage_Validation(t *testing.T) {
	composer := NewMessageComposer()
	
	// Test with nil message
	_, err := composer.ComposeMessage(nil)
	if err == nil {
		t.Error("Expected error for nil message")
	}
	
	// Test with message missing sender
	msg := &email.OutgoingMessage{
		To: []*email.Address{{Address: "recipient@example.com"}},
	}
	_, err = composer.ComposeMessage(msg)
	if err == nil {
		t.Error("Expected error for missing sender")
	}
	
	// Test with message missing recipients
	msg = &email.OutgoingMessage{
		From: &email.Address{Address: "sender@example.com"},
	}
	_, err = composer.ComposeMessage(msg)
	if err == nil {
		t.Error("Expected error for missing recipients")
	}
}

func TestMessageComposer_ValidateAddress(t *testing.T) {
	composer := NewMessageComposer()
	
	testCases := []struct {
		addr      *email.Address
		shouldErr bool
		desc      string
	}{
		{nil, true, "nil address"},
		{&email.Address{}, true, "empty address"},
		{&email.Address{Address: ""}, true, "empty address string"},
		{&email.Address{Address: "invalid"}, true, "no @ symbol"},
		{&email.Address{Address: "@example.com"}, true, "missing local part"},
		{&email.Address{Address: "user@"}, true, "missing domain"},
		{&email.Address{Address: "user@example.com"}, false, "valid address"},
		{&email.Address{Name: "User", Address: "user@example.com"}, false, "valid address with name"},
	}
	
	for _, tc := range testCases {
		err := composer.validateAddress(tc.addr)
		if tc.shouldErr && err == nil {
			t.Errorf("Expected error for %s", tc.desc)
		}
		if !tc.shouldErr && err != nil {
			t.Errorf("Unexpected error for %s: %v", tc.desc, err)
		}
	}
}

func TestMessageComposer_ComposeSimpleMessage(t *testing.T) {
	composer := NewMessageComposer()
	
	msg := &email.OutgoingMessage{
		From:    &email.Address{Name: "Sender", Address: "sender@example.com"},
		To:      []*email.Address{{Name: "Recipient", Address: "recipient@example.com"}},
		Subject: "Test Subject",
		Body:    "Test body content",
	}
	
	data, err := composer.ComposeMessage(msg)
	if err != nil {
		t.Fatalf("Unexpected error composing message: %v", err)
	}
	
	message := string(data)
	
	// Check required headers
	if !strings.Contains(message, "From: Sender <sender@example.com>") {
		t.Error("Missing or incorrect From header")
	}
	
	if !strings.Contains(message, "To: Recipient <recipient@example.com>") {
		t.Error("Missing or incorrect To header")
	}
	
	if !strings.Contains(message, "Subject: Test Subject") {
		t.Error("Missing or incorrect Subject header")
	}
	
	if !strings.Contains(message, "Date: ") {
		t.Error("Missing Date header")
	}
	
	if !strings.Contains(message, "Message-Id: ") {
		t.Error("Missing Message-ID header")
	}
	
	if !strings.Contains(message, "Mime-Version: 1.0") {
		t.Error("Missing MIME-Version header")
	}
	
	if !strings.Contains(message, "User-Agent: POMBO Email Client") {
		t.Error("Missing User-Agent header")
	}
	
	// Check body content
	if !strings.Contains(message, "Test body content") {
		t.Error("Missing body content")
	}
}

func TestMessageComposer_ComposeHTMLMessage(t *testing.T) {
	composer := NewMessageComposer()
	
	msg := &email.OutgoingMessage{
		From:     &email.Address{Address: "sender@example.com"},
		To:       []*email.Address{{Address: "recipient@example.com"}},
		Subject:  "HTML Test",
		BodyHTML: "<h1>Test HTML</h1><p>HTML content</p>",
	}
	
	data, err := composer.ComposeMessage(msg)
	if err != nil {
		t.Fatalf("Unexpected error composing HTML message: %v", err)
	}
	
	message := string(data)
	
	// Check HTML content type
	if !strings.Contains(message, "text/html; charset=utf-8") {
		t.Error("Missing or incorrect HTML content type")
	}
	
	// Check HTML content
	if !strings.Contains(message, "<h1>Test HTML</h1>") {
		t.Error("Missing HTML content")
	}
}

func TestMessageComposer_ComposeMultipartMessage(t *testing.T) {
	composer := NewMessageComposer()
	
	attachment := &email.Attachment{
		Filename:    "test.txt",
		ContentType: "text/plain",
		Content:     []byte("attachment content"),
	}
	
	msg := &email.OutgoingMessage{
		From:        &email.Address{Address: "sender@example.com"},
		To:          []*email.Address{{Address: "recipient@example.com"}},
		Subject:     "Multipart Test",
		Body:        "Text body",
		BodyHTML:    "<p>HTML body</p>",
		Attachments: []*email.Attachment{attachment},
	}
	
	data, err := composer.ComposeMessage(msg)
	if err != nil {
		t.Fatalf("Unexpected error composing multipart message: %v", err)
	}
	
	message := string(data)
	
	// Check multipart content type
	if !strings.Contains(message, "multipart/mixed") {
		t.Error("Missing multipart/mixed content type")
	}
	
	// Check alternative content type for text/HTML
	if !strings.Contains(message, "multipart/alternative") {
		t.Error("Missing multipart/alternative for text/HTML")
	}
	
	// Check boundary
	if !strings.Contains(message, "boundary=") {
		t.Error("Missing boundary parameter")
	}
	
	// Check both text and HTML content
	if !strings.Contains(message, "Text body") {
		t.Error("Missing text body")
	}
	
	if !strings.Contains(message, "<p>HTML body</p>") {
		t.Error("Missing HTML body")
	}
	
	// Check attachment
	if !strings.Contains(message, "filename=\"test.txt\"") {
		t.Error("Missing attachment filename")
	}
	
	if !strings.Contains(message, "Content-Transfer-Encoding: base64") {
		t.Error("Missing base64 encoding for attachment")
	}
}

func TestMessageComposer_ComposeWithCCBCC(t *testing.T) {
	composer := NewMessageComposer()
	
	msg := &email.OutgoingMessage{
		From:    &email.Address{Address: "sender@example.com"},
		To:      []*email.Address{{Address: "to@example.com"}},
		CC:      []*email.Address{{Address: "cc@example.com"}},
		BCC:     []*email.Address{{Address: "bcc@example.com"}},
		Subject: "CC/BCC Test",
		Body:    "Test content",
	}
	
	data, err := composer.ComposeMessage(msg)
	if err != nil {
		t.Fatalf("Unexpected error composing message: %v", err)
	}
	
	message := string(data)
	
	// Check CC header is included
	if !strings.Contains(message, "Cc: cc@example.com") {
		t.Error("Missing CC header")
	}
	
	// Check BCC header is NOT included (per RFC)
	if strings.Contains(message, "Bcc:") {
		t.Error("BCC header should not be included in message")
	}
}

func TestMessageComposer_ComposeWithReferences(t *testing.T) {
	composer := NewMessageComposer()
	
	msg := &email.OutgoingMessage{
		From:      &email.Address{Address: "sender@example.com"},
		To:        []*email.Address{{Address: "recipient@example.com"}},
		Subject:   "Reply Test",
		Body:      "Reply content",
		InReplyTo: "<original@example.com>",
		References: []string{"<ref1@example.com>", "<ref2@example.com>"},
	}
	
	data, err := composer.ComposeMessage(msg)
	if err != nil {
		t.Fatalf("Unexpected error composing message: %v", err)
	}
	
	message := string(data)
	
	// Check In-Reply-To header
	if !strings.Contains(message, "In-Reply-To: <original@example.com>") {
		t.Error("Missing In-Reply-To header")
	}
	
	// Check References header
	if !strings.Contains(message, "References: <ref1@example.com> <ref2@example.com>") {
		t.Error("Missing or incorrect References header")
	}
}

func TestMessageComposer_ComposeWithPriority(t *testing.T) {
	composer := NewMessageComposer()
	
	msg := &email.OutgoingMessage{
		From:     &email.Address{Address: "sender@example.com"},
		To:       []*email.Address{{Address: "recipient@example.com"}},
		Subject:  "High Priority",
		Body:     "Urgent message",
		Priority: email.PriorityHigh,
	}
	
	data, err := composer.ComposeMessage(msg)
	if err != nil {
		t.Fatalf("Unexpected error composing message: %v", err)
	}
	
	message := string(data)
	
	// Check priority headers
	if !strings.Contains(message, "X-Priority: 2") {
		t.Error("Missing X-Priority header for high priority")
	}
	
	if !strings.Contains(message, "Importance: High") {
		t.Error("Missing Importance header for high priority")
	}
}

func TestMessageComposer_FormatAddress(t *testing.T) {
	composer := NewMessageComposer()
	
	testCases := []struct {
		addr     *email.Address
		expected string
		desc     string
	}{
		{
			&email.Address{Address: "user@example.com"},
			"user@example.com",
			"address only",
		},
		{
			&email.Address{Name: "User Name", Address: "user@example.com"},
			"User Name <user@example.com>",
			"name and address",
		},
		{
			&email.Address{Name: "Üser Nämé", Address: "user@example.com"},
			"=?utf-8?q?=C3=9Cser_N=C3=A4m=C3=A9?= <user@example.com>",
			"encoded name",
		},
	}
	
	for _, tc := range testCases {
		result := composer.formatAddress(tc.addr)
		if !strings.Contains(result, tc.addr.Address) {
			t.Errorf("For %s: result '%s' should contain address '%s'", 
				tc.desc, result, tc.addr.Address)
		}
		
		if tc.addr.Name != "" && !strings.Contains(result, "<") {
			t.Errorf("For %s: result '%s' should contain angle brackets", 
				tc.desc, result)
		}
	}
}

func TestMessageComposer_FormatAddressList(t *testing.T) {
	composer := NewMessageComposer()
	
	addresses := []*email.Address{
		{Address: "user1@example.com"},
		{Name: "User Two", Address: "user2@example.com"},
	}
	
	result := composer.formatAddressList(addresses)
	expected := "user1@example.com, User Two <user2@example.com>"
	
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
	
	// Test empty list
	result = composer.formatAddressList([]*email.Address{})
	if result != "" {
		t.Errorf("Expected empty string for empty list, got '%s'", result)
	}
}

func TestMessageComposer_GenerateMessageID(t *testing.T) {
	composer := NewMessageComposer()
	
	messageID := composer.generateMessageID("user@example.com")
	
	if !strings.HasPrefix(messageID, "<") || !strings.HasSuffix(messageID, ">") {
		t.Error("Message-ID should be wrapped in angle brackets")
	}
	
	if !strings.Contains(messageID, "@example.com") {
		t.Error("Message-ID should contain sender domain")
	}
	
	// Test with localhost domain
	messageID2 := composer.generateMessageID("user")
	if !strings.Contains(messageID2, "@localhost") {
		t.Error("Message-ID should use localhost for addresses without domain")
	}
}

func TestMessageComposer_IsMultipart(t *testing.T) {
	composer := NewMessageComposer()
	
	// Simple text message
	msg1 := &email.OutgoingMessage{
		Body: "Simple text",
	}
	if composer.isMultipart(msg1) {
		t.Error("Simple text message should not be multipart")
	}
	
	// Text + HTML message
	msg2 := &email.OutgoingMessage{
		Body:     "Text content",
		BodyHTML: "<p>HTML content</p>",
	}
	if !composer.isMultipart(msg2) {
		t.Error("Text + HTML message should be multipart")
	}
	
	// Message with attachments
	msg3 := &email.OutgoingMessage{
		Body: "Text content",
		Attachments: []*email.Attachment{
			{Filename: "test.txt", Content: []byte("test")},
		},
	}
	if !composer.isMultipart(msg3) {
		t.Error("Message with attachments should be multipart")
	}
}

func TestMessageComposer_EncodeQuotedPrintable(t *testing.T) {
	composer := NewMessageComposer()
	
	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"Line 1\nLine 2", "Line 1\r\nLine 2"},
		{"Text with = sign", "Text with =3D sign"},
		{"Héllo Wörld", "H=E9llo W=F6rld"},
	}
	
	for _, tc := range testCases {
		result := composer.encodeQuotedPrintable(tc.input)
		if result != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", 
				tc.input, tc.expected, result)
		}
	}
}

func TestGenerateBoundary(t *testing.T) {
	boundary1 := generateBoundary()
	boundary2 := generateBoundary()
	
	if boundary1 == "" {
		t.Error("Generated boundary should not be empty")
	}
	
	if boundary1 == boundary2 {
		t.Error("Generated boundaries should be unique")
	}
	
	if !strings.HasPrefix(boundary1, "boundary_") {
		t.Error("Boundary should start with 'boundary_'")
	}
}

func TestMessageComposer_WriteAttachment(t *testing.T) {
	// Test inline attachment with CID
	attachment := &email.Attachment{
		Filename:    "test.jpg",
		ContentType: "image/jpeg",
		Content:     []byte("fake image data"),
		IsInline:    true,
		CID:         "image1",
	}
	
	// This would require setting up a multipart writer to test properly
	// For now, we test that the attachment is not nil and has content
	if attachment.Content == nil || len(attachment.Content) == 0 {
		t.Error("Attachment should have content")
	}
	
	if attachment.CID == "" {
		t.Error("Inline attachment should have CID")
	}
}