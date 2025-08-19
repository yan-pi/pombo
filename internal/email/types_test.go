package email

import (
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
	now := time.Now()
	
	msg := &Message{
		ID:        "test-123",
		UID:       456,
		MessageID: "<test@example.com>",
		Subject:   "Test Subject",
		From: &Address{
			Name:    "John Doe",
			Address: "john@example.com",
		},
		To: []*Address{
			{
				Name:    "Jane Doe",
				Address: "jane@example.com",
			},
		},
		Date:       now,
		IsRead:     false,
		IsFlagged:  true,
		ThreadID:   "thread-123",
		FolderName: "INBOX",
		AccountID:  "account-123",
		Size:       1024,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Test basic fields
	if msg.ID != "test-123" {
		t.Errorf("ID = %v, want %v", msg.ID, "test-123")
	}

	if msg.UID != 456 {
		t.Errorf("UID = %v, want %v", msg.UID, 456)
	}

	if msg.Subject != "Test Subject" {
		t.Errorf("Subject = %v, want %v", msg.Subject, "Test Subject")
	}

	if msg.From.Address != "john@example.com" {
		t.Errorf("From.Address = %v, want %v", msg.From.Address, "john@example.com")
	}

	if len(msg.To) != 1 {
		t.Errorf("To length = %v, want %v", len(msg.To), 1)
	}

	if msg.To[0].Address != "jane@example.com" {
		t.Errorf("To[0].Address = %v, want %v", msg.To[0].Address, "jane@example.com")
	}

	if !msg.IsFlagged {
		t.Error("IsFlagged should be true")
	}

	if msg.IsRead {
		t.Error("IsRead should be false")
	}
}

func TestMessageBody(t *testing.T) {
	body := &MessageBody{
		Text:        "Plain text content",
		HTML:        "<p>HTML content</p>",
		ContentType: "multipart/alternative",
		Charset:     "utf-8",
		Parts: []*BodyPart{
			{
				ContentType: "text/plain",
				Content:     "Plain text part",
				Headers:     map[string]string{"Content-Type": "text/plain"},
			},
			{
				ContentType: "text/html",
				Content:     "<p>HTML part</p>",
				Headers:     map[string]string{"Content-Type": "text/html"},
			},
		},
	}

	if body.Text != "Plain text content" {
		t.Errorf("Text = %v, want %v", body.Text, "Plain text content")
	}

	if body.HTML != "<p>HTML content</p>" {
		t.Errorf("HTML = %v, want %v", body.HTML, "<p>HTML content</p>")
	}

	if body.ContentType != "multipart/alternative" {
		t.Errorf("ContentType = %v, want %v", body.ContentType, "multipart/alternative")
	}

	if len(body.Parts) != 2 {
		t.Errorf("Parts length = %v, want %v", len(body.Parts), 2)
	}

	if body.Parts[0].ContentType != "text/plain" {
		t.Errorf("Parts[0].ContentType = %v, want %v", body.Parts[0].ContentType, "text/plain")
	}
}

func TestAddress(t *testing.T) {
	tests := []struct {
		name     string
		addr     *Address
		wantName string
		wantAddr string
	}{
		{
			name: "with display name",
			addr: &Address{
				Name:    "John Doe",
				Address: "john@example.com",
			},
			wantName: "John Doe",
			wantAddr: "john@example.com",
		},
		{
			name: "without display name",
			addr: &Address{
				Address: "jane@example.com",
			},
			wantName: "",
			wantAddr: "jane@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.addr.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", tt.addr.Name, tt.wantName)
			}

			if tt.addr.Address != tt.wantAddr {
				t.Errorf("Address = %v, want %v", tt.addr.Address, tt.wantAddr)
			}
		})
	}
}

func TestAttachment(t *testing.T) {
	attachment := &Attachment{
		ID:          "att-123",
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Size:        1024000,
		Content:     []byte("PDF content"),
		Headers: map[string]string{
			"Content-Disposition": "attachment; filename=\"document.pdf\"",
		},
		IsInline: false,
		CID:      "",
	}

	if attachment.ID != "att-123" {
		t.Errorf("ID = %v, want %v", attachment.ID, "att-123")
	}

	if attachment.Filename != "document.pdf" {
		t.Errorf("Filename = %v, want %v", attachment.Filename, "document.pdf")
	}

	if attachment.ContentType != "application/pdf" {
		t.Errorf("ContentType = %v, want %v", attachment.ContentType, "application/pdf")
	}

	if attachment.Size != 1024000 {
		t.Errorf("Size = %v, want %v", attachment.Size, 1024000)
	}

	if attachment.IsInline {
		t.Error("IsInline should be false")
	}

	if len(attachment.Content) != 11 {
		t.Errorf("Content length = %v, want %v", len(attachment.Content), 11)
	}
}

func TestFolder(t *testing.T) {
	now := time.Now()
	
	folder := &Folder{
		Name:         "INBOX",
		FullName:     "INBOX",
		Delimiter:    "/",
		Attributes:   []string{"\\HasNoChildren"},
		MessageCount: 100,
		UnseenCount:  5,
		RecentCount:  2,
		UIDNext:      1001,
		UIDValidity:  12345,
		AccountID:    "account-123",
		LastSync:     now,
		Parent:       "",
		Children:     []string{"INBOX/Sent", "INBOX/Drafts"},
		IsSubscribed: true,
	}

	if folder.Name != "INBOX" {
		t.Errorf("Name = %v, want %v", folder.Name, "INBOX")
	}

	if folder.MessageCount != 100 {
		t.Errorf("MessageCount = %v, want %v", folder.MessageCount, 100)
	}

	if folder.UnseenCount != 5 {
		t.Errorf("UnseenCount = %v, want %v", folder.UnseenCount, 5)
	}

	if !folder.IsSubscribed {
		t.Error("IsSubscribed should be true")
	}

	if len(folder.Children) != 2 {
		t.Errorf("Children length = %v, want %v", len(folder.Children), 2)
	}

	if len(folder.Attributes) != 1 {
		t.Errorf("Attributes length = %v, want %v", len(folder.Attributes), 1)
	}
}

func TestOutgoingMessage(t *testing.T) {
	msg := &OutgoingMessage{
		From: &Address{
			Name:    "John Doe",
			Address: "john@example.com",
		},
		To: []*Address{
			{Address: "jane@example.com"},
		},
		CC: []*Address{
			{Address: "cc@example.com"},
		},
		Subject:     "Test Subject",
		Body:        "Plain text body",
		BodyHTML:    "<p>HTML body</p>",
		InReplyTo:   "<original@example.com>",
		References:  []string{"<ref1@example.com>", "<ref2@example.com>"},
		Priority:    PriorityHigh,
		Encrypt:     true,
		Sign:        true,
		AccountID:   "account-123",
	}

	if msg.Subject != "Test Subject" {
		t.Errorf("Subject = %v, want %v", msg.Subject, "Test Subject")
	}

	if msg.From.Address != "john@example.com" {
		t.Errorf("From.Address = %v, want %v", msg.From.Address, "john@example.com")
	}

	if len(msg.To) != 1 {
		t.Errorf("To length = %v, want %v", len(msg.To), 1)
	}

	if len(msg.CC) != 1 {
		t.Errorf("CC length = %v, want %v", len(msg.CC), 1)
	}

	if msg.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want %v", msg.Priority, PriorityHigh)
	}

	if !msg.Encrypt {
		t.Error("Encrypt should be true")
	}

	if !msg.Sign {
		t.Error("Sign should be true")
	}

	if len(msg.References) != 2 {
		t.Errorf("References length = %v, want %v", len(msg.References), 2)
	}
}

func TestThread(t *testing.T) {
	now := time.Now()
	
	lastMessage := &Message{
		ID:      "msg-latest",
		Subject: "Re: Test Thread",
		Date:    now,
	}

	thread := &Thread{
		ID:       "thread-123",
		Subject:  "Test Thread",
		Messages: []*Message{
			{ID: "msg-1", Subject: "Test Thread"},
			{ID: "msg-2", Subject: "Re: Test Thread"},
			lastMessage,
		},
		Participants: []*Address{
			{Address: "john@example.com"},
			{Address: "jane@example.com"},
		},
		LastMessage:  lastMessage,
		MessageCount: 3,
		UnreadCount:  1,
		CreatedAt:    now.Add(-time.Hour),
		UpdatedAt:    now,
		FolderName:   "INBOX",
		AccountID:    "account-123",
	}

	if thread.ID != "thread-123" {
		t.Errorf("ID = %v, want %v", thread.ID, "thread-123")
	}

	if thread.MessageCount != 3 {
		t.Errorf("MessageCount = %v, want %v", thread.MessageCount, 3)
	}

	if len(thread.Messages) != 3 {
		t.Errorf("Messages length = %v, want %v", len(thread.Messages), 3)
	}

	if thread.LastMessage.ID != "msg-latest" {
		t.Errorf("LastMessage.ID = %v, want %v", thread.LastMessage.ID, "msg-latest")
	}

	if len(thread.Participants) != 2 {
		t.Errorf("Participants length = %v, want %v", len(thread.Participants), 2)
	}
}

func TestSearchCriteria(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)
	before := time.Now()

	criteria := &SearchCriteria{
		Query:   "test query",
		From:    "john@example.com",
		To:      "jane@example.com",
		Subject: "test subject",
		Body:    "test body",
		Since:   &since,
		Before:  &before,
		HasFlag: []string{FlagSeen, FlagFlagged},
		NotFlag: []string{FlagDeleted},
		Size: &SizeConstraint{
			Operator: SizeGreaterThan,
			Size:     1024,
		},
		Folder: "INBOX",
		Limit:  50,
		Offset: 0,
	}

	if criteria.Query != "test query" {
		t.Errorf("Query = %v, want %v", criteria.Query, "test query")
	}

	if criteria.From != "john@example.com" {
		t.Errorf("From = %v, want %v", criteria.From, "john@example.com")
	}

	if len(criteria.HasFlag) != 2 {
		t.Errorf("HasFlag length = %v, want %v", len(criteria.HasFlag), 2)
	}

	if criteria.Size.Operator != SizeGreaterThan {
		t.Errorf("Size.Operator = %v, want %v", criteria.Size.Operator, SizeGreaterThan)
	}

	if criteria.Size.Size != 1024 {
		t.Errorf("Size.Size = %v, want %v", criteria.Size.Size, 1024)
	}
}

func TestConnectionState(t *testing.T) {
	tests := []struct {
		state ConnectionState
		want  string
	}{
		{StateDisconnected, "disconnected"},
		{StateConnected, "connected"},
		{StateAuthenticated, "authenticated"},
		{StateSelected, "selected"},
		{StateIdle, "idle"},
		{StateLogout, "logout"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if string(tt.state) != tt.want {
				t.Errorf("ConnectionState = %v, want %v", string(tt.state), tt.want)
			}
		})
	}
}

func TestAuthType(t *testing.T) {
	tests := []struct {
		authType AuthType
		want     string
	}{
		{AuthTypePassword, "password"},
		{AuthTypeOAuth2, "oauth2"},
		{AuthTypeOAuth1, "oauth1"},
		{AuthTypeAPIKey, "apikey"},
	}

	for _, tt := range tests {
		t.Run(string(tt.authType), func(t *testing.T) {
			if string(tt.authType) != tt.want {
				t.Errorf("AuthType = %v, want %v", string(tt.authType), tt.want)
			}
		})
	}
}

func TestMessagePriority(t *testing.T) {
	tests := []struct {
		priority MessagePriority
		want     string
	}{
		{PriorityLow, "low"},
		{PriorityNormal, "normal"},
		{PriorityHigh, "high"},
		{PriorityUrgent, "urgent"},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if string(tt.priority) != tt.want {
				t.Errorf("MessagePriority = %v, want %v", string(tt.priority), tt.want)
			}
		})
	}
}

func TestUpdateType(t *testing.T) {
	tests := []struct {
		updateType UpdateType
		want       string
	}{
		{UpdateTypeNewMessage, "new_message"},
		{UpdateTypeMessageFlags, "message_flags"},
		{UpdateTypeMessageMove, "message_move"},
		{UpdateTypeMessageDelete, "message_delete"},
		{UpdateTypeFolderCreate, "folder_create"},
		{UpdateTypeFolderDelete, "folder_delete"},
		{UpdateTypeFolderRename, "folder_rename"},
		{UpdateTypeConnection, "connection"},
		{UpdateTypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(string(tt.updateType), func(t *testing.T) {
			if string(tt.updateType) != tt.want {
				t.Errorf("UpdateType = %v, want %v", string(tt.updateType), tt.want)
			}
		})
	}
}

func TestEmailUpdate(t *testing.T) {
	now := time.Now()
	
	update := &EmailUpdate{
		Type:       UpdateTypeNewMessage,
		AccountID:  "account-123",
		FolderName: "INBOX",
		Message: &Message{
			ID:      "msg-123",
			Subject: "New Message",
		},
		Timestamp: now,
	}

	if update.Type != UpdateTypeNewMessage {
		t.Errorf("Type = %v, want %v", update.Type, UpdateTypeNewMessage)
	}

	if update.AccountID != "account-123" {
		t.Errorf("AccountID = %v, want %v", update.AccountID, "account-123")
	}

	if update.Message.ID != "msg-123" {
		t.Errorf("Message.ID = %v, want %v", update.Message.ID, "msg-123")
	}

	if update.Timestamp != now {
		t.Errorf("Timestamp = %v, want %v", update.Timestamp, now)
	}
}

func TestCredentials(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	
	creds := &Credentials{
		Type:     AuthTypeOAuth2,
		Username: "",
		Password: "",
		Token: &OAuthToken{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			ExpiresAt:    expiresAt,
			Scope:        "email profile",
		},
		ExpiresAt: &expiresAt,
	}

	if creds.Type != AuthTypeOAuth2 {
		t.Errorf("Type = %v, want %v", creds.Type, AuthTypeOAuth2)
	}

	if creds.Token.AccessToken != "access-token" {
		t.Errorf("Token.AccessToken = %v, want %v", creds.Token.AccessToken, "access-token")
	}

	if creds.Token.TokenType != "Bearer" {
		t.Errorf("Token.TokenType = %v, want %v", creds.Token.TokenType, "Bearer")
	}

	if creds.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	}

	if !creds.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", creds.ExpiresAt, expiresAt)
	}
}

func TestCommonConstants(t *testing.T) {
	// Test flag constants
	flags := []string{FlagSeen, FlagAnswered, FlagFlagged, FlagDeleted, FlagDraft, FlagRecent}
	expectedFlags := []string{"\\Seen", "\\Answered", "\\Flagged", "\\Deleted", "\\Draft", "\\Recent"}

	for i, flag := range flags {
		if flag != expectedFlags[i] {
			t.Errorf("Flag %d = %v, want %v", i, flag, expectedFlags[i])
		}
	}

	// Test folder constants
	folders := []string{FolderInbox, FolderSent, FolderDrafts, FolderTrash, FolderSpam, FolderArchive}
	expectedFolders := []string{"INBOX", "Sent", "Drafts", "Trash", "Spam", "Archive"}

	for i, folder := range folders {
		if folder != expectedFolders[i] {
			t.Errorf("Folder %d = %v, want %v", i, folder, expectedFolders[i])
		}
	}
}

// Benchmark tests for performance
func BenchmarkMessage_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &Message{
			ID:      "test-123",
			Subject: "Test Subject",
			From:    &Address{Address: "test@example.com"},
			Date:    time.Now(),
		}
	}
}

func BenchmarkAddress_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &Address{
			Name:    "Test User",
			Address: "test@example.com",
		}
	}
}