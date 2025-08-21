package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"

	"github.com/ybarbara/pombo/internal/email"
)

// MessageComposer handles RFC 2822 message composition
type MessageComposer struct {
	boundary string
}

// NewMessageComposer creates a new message composer
func NewMessageComposer() *MessageComposer {
	return &MessageComposer{
		boundary: generateBoundary(),
	}
}

// ComposeMessage creates a complete RFC 2822 compliant email message
func (c *MessageComposer) ComposeMessage(msg *email.OutgoingMessage) ([]byte, error) {
	if msg == nil {
		return nil, email.NewEmailError(email.ErrorTypeValidation, "MESSAGE_REQUIRED", "message is required", nil, false)
	}
	
	if err := c.validateMessage(msg); err != nil {
		return nil, err
	}
	
	var buf bytes.Buffer
	
	// Write headers
	if err := c.writeHeaders(&buf, msg); err != nil {
		return nil, err
	}
	
	// Write body
	if err := c.writeBody(&buf, msg); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// validateMessage validates the outgoing message
func (c *MessageComposer) validateMessage(msg *email.OutgoingMessage) error {
	if msg.From == nil || msg.From.Address == "" {
		return email.NewEmailError(email.ErrorTypeValidation, "SENDER_REQUIRED", "sender address is required", nil, false)
	}
	
	if len(msg.To) == 0 {
		return email.NewEmailError(email.ErrorTypeValidation, "RECIPIENTS_REQUIRED", "at least one recipient is required", nil, false)
	}
	
	// Validate email addresses
	if err := c.validateAddress(msg.From); err != nil {
		return email.WrapError(err, email.ErrorTypeValidation, "INVALID_SENDER", "invalid sender address", false)
	}
	
	for _, addr := range msg.To {
		if err := c.validateAddress(addr); err != nil {
			return email.WrapError(err, email.ErrorTypeValidation, "INVALID_RECIPIENT", "invalid recipient address", false)
		}
	}
	
	for _, addr := range msg.CC {
		if err := c.validateAddress(addr); err != nil {
			return email.WrapError(err, email.ErrorTypeValidation, "INVALID_CC", "invalid CC address", false)
		}
	}
	
	for _, addr := range msg.BCC {
		if err := c.validateAddress(addr); err != nil {
			return email.WrapError(err, email.ErrorTypeValidation, "INVALID_BCC", "invalid BCC address", false)
		}
	}
	
	return nil
}

// validateAddress validates an email address
func (c *MessageComposer) validateAddress(addr *email.Address) error {
	if addr == nil || addr.Address == "" {
		return email.NewEmailError(email.ErrorTypeValidation, "ADDRESS_REQUIRED", "email address is required", nil, false)
	}
	
	// Basic email validation
	if !strings.Contains(addr.Address, "@") {
		return email.NewEmailError(email.ErrorTypeValidation, "INVALID_ADDRESS",
			fmt.Sprintf("invalid email address: %s", addr.Address), nil, false)
	}
	
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return email.NewEmailError(email.ErrorTypeValidation, "INVALID_ADDRESS",
			fmt.Sprintf("invalid email address: %s", addr.Address), nil, false)
	}
	
	return nil
}

// writeHeaders writes RFC 2822 headers
func (c *MessageComposer) writeHeaders(w io.Writer, msg *email.OutgoingMessage) error {
	headers := make(textproto.MIMEHeader)
	
	// Required headers
	headers.Set("From", c.formatAddress(msg.From))
	headers.Set("To", c.formatAddressList(msg.To))
	headers.Set("Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	headers.Set("Date", time.Now().Format(time.RFC1123Z))
	headers.Set("Message-ID", c.generateMessageID(msg.From.Address))
	
	// Optional headers
	if len(msg.CC) > 0 {
		headers.Set("Cc", c.formatAddressList(msg.CC))
	}
	
	// Note: BCC headers are not included in the final message
	
	if msg.InReplyTo != "" {
		headers.Set("In-Reply-To", msg.InReplyTo)
	}
	
	if len(msg.References) > 0 {
		headers.Set("References", strings.Join(msg.References, " "))
	}
	
	// Priority header
	if msg.Priority != "" && msg.Priority != email.PriorityNormal {
		switch msg.Priority {
		case email.PriorityLow:
			headers.Set("X-Priority", "5")
			headers.Set("Importance", "Low")
		case email.PriorityHigh:
			headers.Set("X-Priority", "2")
			headers.Set("Importance", "High")
		case email.PriorityUrgent:
			headers.Set("X-Priority", "1")
			headers.Set("Importance", "High")
		}
	}
	
	// Custom headers
	for key, value := range msg.Headers {
		headers.Set(key, value)
	}
	
	// MIME headers
	if c.isMultipart(msg) {
		headers.Set("MIME-Version", "1.0")
		headers.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", c.boundary))
	} else if msg.BodyHTML != "" {
		headers.Set("MIME-Version", "1.0")
		headers.Set("Content-Type", "text/html; charset=utf-8")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
	} else {
		headers.Set("MIME-Version", "1.0")
		headers.Set("Content-Type", "text/plain; charset=utf-8")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
	}
	
	// User-Agent
	headers.Set("User-Agent", "POMBO Email Client")
	
	// Write headers
	for key, values := range headers {
		for _, value := range values {
			if _, err := fmt.Fprintf(w, "%s: %s\r\n", key, value); err != nil {
				return err
			}
		}
	}
	
	// End headers
	if _, err := fmt.Fprint(w, "\r\n"); err != nil {
		return err
	}
	
	return nil
}

// writeBody writes the message body
func (c *MessageComposer) writeBody(w io.Writer, msg *email.OutgoingMessage) error {
	if c.isMultipart(msg) {
		return c.writeMultipartBody(w, msg)
	}
	
	// Simple body (text or HTML only)
	if msg.BodyHTML != "" {
		encoded := c.encodeQuotedPrintable(msg.BodyHTML)
		_, err := w.Write([]byte(encoded))
		return err
	}
	
	encoded := c.encodeQuotedPrintable(msg.Body)
	_, err := w.Write([]byte(encoded))
	return err
}

// writeMultipartBody writes a multipart message body
func (c *MessageComposer) writeMultipartBody(w io.Writer, msg *email.OutgoingMessage) error {
	writer := multipart.NewWriter(w)
	writer.SetBoundary(c.boundary)
	
	// Write text/HTML parts if both exist
	if msg.Body != "" && msg.BodyHTML != "" {
		// Create alternative multipart for text and HTML
		altBoundary := generateBoundary()
		
		// Write alternative part header
		altHeader := make(textproto.MIMEHeader)
		altHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%s", altBoundary))
		
		altPart, err := writer.CreatePart(altHeader)
		if err != nil {
			return err
		}
		
		altWriter := multipart.NewWriter(altPart)
		altWriter.SetBoundary(altBoundary)
		
		// Write text part
		textHeader := make(textproto.MIMEHeader)
		textHeader.Set("Content-Type", "text/plain; charset=utf-8")
		textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		
		textPart, err := altWriter.CreatePart(textHeader)
		if err != nil {
			return err
		}
		
		if _, err := textPart.Write([]byte(c.encodeQuotedPrintable(msg.Body))); err != nil {
			return err
		}
		
		// Write HTML part
		htmlHeader := make(textproto.MIMEHeader)
		htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
		htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		
		htmlPart, err := altWriter.CreatePart(htmlHeader)
		if err != nil {
			return err
		}
		
		if _, err := htmlPart.Write([]byte(c.encodeQuotedPrintable(msg.BodyHTML))); err != nil {
			return err
		}
		
		altWriter.Close()
		
	} else if msg.BodyHTML != "" {
		// HTML only
		htmlHeader := make(textproto.MIMEHeader)
		htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
		htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		
		htmlPart, err := writer.CreatePart(htmlHeader)
		if err != nil {
			return err
		}
		
		if _, err := htmlPart.Write([]byte(c.encodeQuotedPrintable(msg.BodyHTML))); err != nil {
			return err
		}
		
	} else {
		// Text only
		textHeader := make(textproto.MIMEHeader)
		textHeader.Set("Content-Type", "text/plain; charset=utf-8")
		textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		
		textPart, err := writer.CreatePart(textHeader)
		if err != nil {
			return err
		}
		
		if _, err := textPart.Write([]byte(c.encodeQuotedPrintable(msg.Body))); err != nil {
			return err
		}
	}
	
	// Write attachments
	for _, attachment := range msg.Attachments {
		if err := c.writeAttachment(writer, attachment); err != nil {
			return err
		}
	}
	
	return writer.Close()
}

// writeAttachment writes an attachment part
func (c *MessageComposer) writeAttachment(writer *multipart.Writer, attachment *email.Attachment) error {
	if attachment == nil || len(attachment.Content) == 0 {
		return nil
	}
	
	header := make(textproto.MIMEHeader)
	
	// Content-Type
	contentType := attachment.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	
	filename := attachment.Filename
	if filename != "" {
		contentType += fmt.Sprintf("; name=\"%s\"", filename)
	}
	header.Set("Content-Type", contentType)
	
	// Content-Disposition
	if attachment.IsInline {
		disposition := "inline"
		if filename != "" {
			disposition += fmt.Sprintf("; filename=\"%s\"", filename)
		}
		if attachment.CID != "" {
			header.Set("Content-ID", fmt.Sprintf("<%s>", attachment.CID))
		}
		header.Set("Content-Disposition", disposition)
	} else {
		disposition := "attachment"
		if filename != "" {
			disposition += fmt.Sprintf("; filename=\"%s\"", filename)
		}
		header.Set("Content-Disposition", disposition)
	}
	
	// Content-Transfer-Encoding
	header.Set("Content-Transfer-Encoding", "base64")
	
	// Create part
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	
	// Encode content as base64
	encoder := base64.NewEncoder(base64.StdEncoding, part)
	if _, err := encoder.Write(attachment.Content); err != nil {
		return err
	}
	
	return encoder.Close()
}

// isMultipart determines if the message needs multipart encoding
func (c *MessageComposer) isMultipart(msg *email.OutgoingMessage) bool {
	return len(msg.Attachments) > 0 || (msg.Body != "" && msg.BodyHTML != "")
}

// formatAddress formats an email address for headers
func (c *MessageComposer) formatAddress(addr *email.Address) string {
	if addr.Name != "" {
		// Encode display name if needed
		encodedName := mime.QEncoding.Encode("utf-8", addr.Name)
		return fmt.Sprintf("%s <%s>", encodedName, addr.Address)
	}
	return addr.Address
}

// formatAddressList formats a list of email addresses
func (c *MessageComposer) formatAddressList(addresses []*email.Address) string {
	if len(addresses) == 0 {
		return ""
	}
	
	formatted := make([]string, len(addresses))
	for i, addr := range addresses {
		formatted[i] = c.formatAddress(addr)
	}
	
	return strings.Join(formatted, ", ")
}

// generateMessageID generates a unique Message-ID
func (c *MessageComposer) generateMessageID(fromAddr string) string {
	domain := "localhost"
	if atIndex := strings.LastIndex(fromAddr, "@"); atIndex != -1 {
		domain = fromAddr[atIndex+1:]
	}
	
	timestamp := time.Now().Unix()
	return fmt.Sprintf("<%d.%s@%s>", timestamp, generateBoundary(), domain)
}

// generateBoundary generates a unique MIME boundary
func generateBoundary() string {
	return fmt.Sprintf("boundary_%d_%s", time.Now().Unix(), 
		base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
}

// encodeQuotedPrintable encodes text using quoted-printable encoding
func (c *MessageComposer) encodeQuotedPrintable(text string) string {
	// Simple quoted-printable implementation
	// For production use, consider using mime/quotedprintable package
	var buf bytes.Buffer
	
	for i, r := range text {
		if r == '\n' {
			buf.WriteString("\r\n")
		} else if r == '\r' {
			// Skip standalone CR
			if i+1 < len(text) && text[i+1] != '\n' {
				buf.WriteString("=0D")
			}
		} else if r > 126 || r < 32 {
			// Encode non-printable characters
			buf.WriteString(fmt.Sprintf("=%02X", r))
		} else if r == '=' {
			buf.WriteString("=3D")
		} else {
			buf.WriteRune(r)
		}
	}
	
	return buf.String()
}

// GetAllRecipients returns all recipients (To, CC, BCC) for envelope
func GetAllRecipients(msg *email.OutgoingMessage) []string {
	var recipients []string
	
	for _, addr := range msg.To {
		recipients = append(recipients, addr.Address)
	}
	
	for _, addr := range msg.CC {
		recipients = append(recipients, addr.Address)
	}
	
	for _, addr := range msg.BCC {
		recipients = append(recipients, addr.Address)
	}
	
	return recipients
}

// DetectContentType attempts to detect content type from filename
func DetectContentType(filename string) string {
	if filename == "" {
		return "application/octet-stream"
	}
	
	ext := strings.ToLower(filepath.Ext(filename))
	
	switch ext {
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".zip":
		return "application/zip"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	default:
		return "application/octet-stream"
	}
}