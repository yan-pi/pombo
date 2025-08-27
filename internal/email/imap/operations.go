package imap

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/ybarbara/pombo/internal/email"
)

// List returns mailbox list
func (c *Client) List(ctx context.Context, ref, name string) ([]*email.Folder, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateAuthenticated || c.client == nil {
		return nil, email.NewEmailError(email.ErrorTypeProtocol, "NOT_AUTHENTICATED", "not authenticated", nil, false)
	}
	
	// Use LIST command
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	
	go func() {
		done <- c.client.List(ref, name, mailboxes)
	}()
	
	folders := []*email.Folder{}
	
	for m := range mailboxes {
		folder := &email.Folder{
			Name:         m.Name,
			FullName:     m.Name,
			Delimiter:    m.Delimiter,
			Attributes:   m.Attributes,
			IsSubscribed: false, // Will be updated by LSUB if needed
		}
		
		folders = append(folders, folder)
	}
	
	if err := <-done; err != nil {
		return nil, email.WrapError(err, email.ErrorTypeProtocol, "LIST_FAILED", "LIST command failed", true)
	}
	
	return folders, nil
}

// Subscribe subscribes to a mailbox
func (c *Client) Subscribe(ctx context.Context, name string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateAuthenticated {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_AUTHENTICATED", "not authenticated", nil, false)
	}
	
	err := c.client.Subscribe(name)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "SUBSCRIBE_FAILED", "SUBSCRIBE command failed", true)
	}
	
	return nil
}

// Unsubscribe unsubscribes from a mailbox
func (c *Client) Unsubscribe(ctx context.Context, name string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateAuthenticated {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_AUTHENTICATED", "not authenticated", nil, false)
	}
	
	err := c.client.Unsubscribe(name)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "UNSUBSCRIBE_FAILED", "UNSUBSCRIBE command failed", true)
	}
	
	return nil
}

// Create creates a new mailbox
func (c *Client) Create(ctx context.Context, name string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateAuthenticated {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_AUTHENTICATED", "not authenticated", nil, false)
	}
	
	err := c.client.Create(name)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "CREATE_FAILED", "CREATE command failed", true)
	}
	
	return nil
}

// Delete deletes a mailbox
func (c *Client) Delete(ctx context.Context, name string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateAuthenticated {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_AUTHENTICATED", "not authenticated", nil, false)
	}
	
	err := c.client.Delete(name)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "DELETE_FAILED", "DELETE command failed", true)
	}
	
	return nil
}

// Rename renames a mailbox
func (c *Client) Rename(ctx context.Context, oldName, newName string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateAuthenticated {
		return email.NewEmailError(email.ErrorTypeProtocol, "NOT_AUTHENTICATED", "not authenticated", nil, false)
	}
	
	err := c.client.Rename(oldName, newName)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "RENAME_FAILED", "RENAME command failed", true)
	}
	
	return nil
}

// Select selects a mailbox
func (c *Client) Select(ctx context.Context, name string) (*email.FolderStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.state < email.StateAuthenticated || c.client == nil {
		return nil, email.NewEmailError(email.ErrorTypeProtocol, "NOT_AUTHENTICATED", "not authenticated", nil, false)
	}
	
	mbox, err := c.client.Select(name, false)
	if err != nil {
		return nil, email.WrapError(err, email.ErrorTypeProtocol, "SELECT_FAILED", "SELECT command failed", true)
	}
	
	c.state = email.StateSelected
	
	return c.convertMailboxStatus(name, mbox), nil
}

// Examine examines a mailbox (read-only)
func (c *Client) Examine(ctx context.Context, name string) (*email.FolderStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateAuthenticated {
		return nil, email.NewEmailError(email.ErrorTypeProtocol, "NOT_AUTHENTICATED", "not authenticated", nil, false)
	}
	
	mbox, err := c.client.Select(name, true)
	if err != nil {
		return nil, email.WrapError(err, email.ErrorTypeProtocol, "EXAMINE_FAILED", "EXAMINE command failed", true)
	}
	
	status := c.convertMailboxStatus(name, mbox)
	status.ReadOnly = true
	
	return status, nil
}

// Search searches for messages
func (c *Client) Search(ctx context.Context, criteria *email.SearchCriteria) ([]uint32, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateSelected || c.client == nil {
		return nil, email.NewEmailError(email.ErrorTypeProtocol, "NO_MAILBOX_SELECTED", "no mailbox selected", nil, false)
	}
	
	// Build IMAP search criteria
	searchCriteria := c.buildSearchCriteria(criteria)
	
	uids, err := c.client.UidSearch(searchCriteria)
	if err != nil {
		return nil, email.WrapError(err, email.ErrorTypeProtocol, "SEARCH_FAILED", "SEARCH command failed", true)
	}
	
	return uids, nil
}

// Fetch fetches messages by UID
func (c *Client) Fetch(ctx context.Context, uids []uint32, items []string) ([]*email.Message, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateSelected {
		return nil, email.NewEmailError(email.ErrorTypeProtocol, "NO_MAILBOX_SELECTED", "no mailbox selected", nil, false)
	}
	
	if len(uids) == 0 {
		return []*email.Message{}, nil
	}
	
	// Build fetch items
	fetchItems := c.buildFetchItems(items)
	
	// Create sequence set from UIDs
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	
	go func() {
		done <- c.client.UidFetch(seqset, fetchItems, messages)
	}()
	
	result := []*email.Message{}
	
	for msg := range messages {
		emailMsg, err := c.convertMessage(msg)
		if err != nil {
			// Log error but continue with other messages
			continue
		}
		result = append(result, emailMsg)
	}
	
	if err := <-done; err != nil {
		return nil, email.WrapError(err, email.ErrorTypeProtocol, "FETCH_FAILED", "FETCH command failed", true)
	}
	
	return result, nil
}

// Store updates message flags
func (c *Client) Store(ctx context.Context, uids []uint32, flags []string, action string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateSelected {
		return email.NewEmailError(email.ErrorTypeProtocol, "NO_MAILBOX_SELECTED", "no mailbox selected", nil, false)
	}
	
	if len(uids) == 0 {
		return nil
	}
	
	// Create sequence set from UIDs
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	
	// Determine store operation
	var operation imap.StoreItem
	switch action {
	case "SET", "=":
		operation = imap.FormatFlagsOp(imap.SetFlags, false)
	case "ADD", "+":
		operation = imap.FormatFlagsOp(imap.AddFlags, false)
	case "REMOVE", "-":
		operation = imap.FormatFlagsOp(imap.RemoveFlags, false)
	default:
		return email.NewEmailError(email.ErrorTypeValidation, "INVALID_STORE_ACTION", 
			fmt.Sprintf("invalid store action: %s", action), nil, false)
	}
	
	// Convert string flags to interface{} slice
	flagsInterface := make([]interface{}, len(flags))
	for i, flag := range flags {
		flagsInterface[i] = flag
	}
	
	err := c.client.UidStore(seqset, operation, flagsInterface, nil)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "STORE_FAILED", "STORE command failed", true)
	}
	
	return nil
}

// Copy copies messages to another mailbox
func (c *Client) Copy(ctx context.Context, uids []uint32, dest string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateSelected {
		return email.NewEmailError(email.ErrorTypeProtocol, "NO_MAILBOX_SELECTED", "no mailbox selected", nil, false)
	}
	
	if len(uids) == 0 {
		return nil
	}
	
	// Create sequence set from UIDs
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	
	err := c.client.UidCopy(seqset, dest)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "COPY_FAILED", "COPY command failed", true)
	}
	
	return nil
}

// Move moves messages to another mailbox
func (c *Client) Move(ctx context.Context, uids []uint32, dest string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateSelected {
		return email.NewEmailError(email.ErrorTypeProtocol, "NO_MAILBOX_SELECTED", "no mailbox selected", nil, false)
	}
	
	if len(uids) == 0 {
		return nil
	}
	
	// Create sequence set from UIDs
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	
	// Check if server supports MOVE extension
	caps, err := c.client.Capability()
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "CAPABILITY_FAILED", "failed to get capabilities", true)
	}
	
	supportsMove := false
	for cap := range caps {
		if strings.ToUpper(cap) == "MOVE" {
			supportsMove = true
			break
		}
	}
	
	if supportsMove {
		// Use MOVE command if supported
		err = c.client.UidMove(seqset, dest)
	} else {
		// Fall back to COPY + STORE + EXPUNGE
		err = c.client.UidCopy(seqset, dest)
		if err != nil {
			return email.WrapError(err, email.ErrorTypeProtocol, "COPY_FAILED", "COPY command failed during move", true)
		}
		
		// Mark as deleted
		flagsInterface := []interface{}{imap.DeletedFlag}
		err = c.client.UidStore(seqset, imap.FormatFlagsOp(imap.AddFlags, false), flagsInterface, nil)
		if err != nil {
			return email.WrapError(err, email.ErrorTypeProtocol, "STORE_FAILED", "STORE command failed during move", true)
		}
		
		// Expunge to actually delete
		err = c.client.Expunge(nil)
	}
	
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "MOVE_FAILED", "MOVE operation failed", true)
	}
	
	return nil
}

// Expunge expunges deleted messages
func (c *Client) Expunge(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.state < email.StateSelected {
		return email.NewEmailError(email.ErrorTypeProtocol, "NO_MAILBOX_SELECTED", "no mailbox selected", nil, false)
	}
	
	err := c.client.Expunge(nil)
	if err != nil {
		return email.WrapError(err, email.ErrorTypeProtocol, "EXPUNGE_FAILED", "EXPUNGE command failed", true)
	}
	
	return nil
}

// convertMailboxStatus converts IMAP mailbox status to folder status
func (c *Client) convertMailboxStatus(name string, mbox *imap.MailboxStatus) *email.FolderStatus {
	status := &email.FolderStatus{
		Name:        name,
		Messages:    mbox.Messages,
		Recent:      mbox.Recent,
		UIDNext:     mbox.UidNext,
		UIDValidity: mbox.UidValidity,
		Unseen:      mbox.Unseen,
		ReadOnly:    false,
	}
	
	// Extract flags
	status.Flags = make([]string, len(mbox.Flags))
	for i, flag := range mbox.Flags {
		status.Flags[i] = flag
	}
	
	// Extract permanent flags
	status.PermanentFlags = make([]string, len(mbox.PermanentFlags))
	for i, flag := range mbox.PermanentFlags {
		status.PermanentFlags[i] = flag
	}
	
	return status
}

// buildSearchCriteria converts email search criteria to IMAP search criteria
func (c *Client) buildSearchCriteria(criteria *email.SearchCriteria) *imap.SearchCriteria {
	searchCriteria := imap.NewSearchCriteria()
	
	if criteria == nil {
		return searchCriteria
	}
	
	// Text searches
	if criteria.Query != "" {
		searchCriteria.Text = []string{criteria.Query}
	}
	
	if criteria.From != "" {
		searchCriteria.Header.Set("From", criteria.From)
	}
	
	if criteria.To != "" {
		searchCriteria.Header.Set("To", criteria.To)
	}
	
	if criteria.Subject != "" {
		searchCriteria.Header.Set("Subject", criteria.Subject)
	}
	
	if criteria.Body != "" {
		searchCriteria.Body = []string{criteria.Body}
	}
	
	// Date searches
	if criteria.Since != nil {
		searchCriteria.Since = *criteria.Since
	}
	
	if criteria.Before != nil {
		searchCriteria.Before = *criteria.Before
	}
	
	// Flag searches
	if len(criteria.HasFlag) > 0 {
		for _, flag := range criteria.HasFlag {
			searchCriteria.WithFlags = append(searchCriteria.WithFlags, flag)
		}
	}
	
	if len(criteria.NotFlag) > 0 {
		for _, flag := range criteria.NotFlag {
			searchCriteria.WithoutFlags = append(searchCriteria.WithoutFlags, flag)
		}
	}
	
	// Size constraints
	if criteria.Size != nil {
		switch criteria.Size.Operator {
		case email.SizeGreaterThan:
			searchCriteria.Larger = uint32(criteria.Size.Size)
		case email.SizeLessThan:
			searchCriteria.Smaller = uint32(criteria.Size.Size)
		}
	}
	
	return searchCriteria
}

// buildFetchItems converts fetch item strings to IMAP fetch items
func (c *Client) buildFetchItems(items []string) []imap.FetchItem {
	if len(items) == 0 {
		// Default fetch items for a complete message
		return []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchInternalDate,
			imap.FetchRFC822Size,
			imap.FetchUid,
			imap.FetchBodyStructure,
		}
	}
	
	fetchItems := make([]imap.FetchItem, 0, len(items))
	
	for _, item := range items {
		switch strings.ToUpper(item) {
		case "ENVELOPE":
			fetchItems = append(fetchItems, imap.FetchEnvelope)
		case "FLAGS":
			fetchItems = append(fetchItems, imap.FetchFlags)
		case "INTERNALDATE":
			fetchItems = append(fetchItems, imap.FetchInternalDate)
		case "RFC822.SIZE":
			fetchItems = append(fetchItems, imap.FetchRFC822Size)
		case "UID":
			fetchItems = append(fetchItems, imap.FetchUid)
		case "BODYSTRUCTURE":
			fetchItems = append(fetchItems, imap.FetchBodyStructure)
		case "RFC822":
			fetchItems = append(fetchItems, imap.FetchRFC822)
		case "RFC822.HEADER":
			fetchItems = append(fetchItems, imap.FetchRFC822Header)
		case "RFC822.TEXT":
			fetchItems = append(fetchItems, imap.FetchRFC822Text)
		}
	}
	
	return fetchItems
}

// convertMessage converts IMAP message to email message
func (c *Client) convertMessage(msg *imap.Message) (*email.Message, error) {
	message := &email.Message{
		UID: msg.Uid,
	}
	
	// Set basic message properties
	if msg.Size > 0 {
		message.Size = int64(msg.Size)
	}
	
	if !msg.InternalDate.IsZero() {
		message.Date = msg.InternalDate
	}
	
	// Convert flags
	message.Flags = msg.Flags
	for _, flag := range msg.Flags {
		switch flag {
		case imap.SeenFlag:
			message.IsRead = true
		case imap.FlaggedFlag:
			message.IsFlagged = true
		case imap.DraftFlag:
			message.IsDraft = true
		case imap.AnsweredFlag:
			message.IsAnswered = true
		case imap.DeletedFlag:
			message.IsDeleted = true
		}
	}
	
	// Convert envelope
	if msg.Envelope != nil {
		env := msg.Envelope
		
		message.Subject = env.Subject
		message.MessageID = env.MessageId
		message.InReplyTo = env.InReplyTo
		message.Date = env.Date
		
		// Convert addresses
		if len(env.From) > 0 {
			message.From = c.convertAddress(env.From[0])
		}
		
		message.To = c.convertAddresses(env.To)
		message.CC = c.convertAddresses(env.Cc)
		message.BCC = c.convertAddresses(env.Bcc)
		
		// References
		if len(env.ReplyTo) > 0 {
			message.References = []string{env.InReplyTo}
		}
	}
	
	// TODO: Parse body structure and content
	// This would require additional FETCH commands for message parts
	
	message.ID = strconv.FormatUint(uint64(message.UID), 10)
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()
	
	return message, nil
}

// convertAddress converts IMAP address to email address
func (c *Client) convertAddress(addr *imap.Address) *email.Address {
	if addr == nil {
		return nil
	}
	
	return &email.Address{
		Name:    addr.PersonalName,
		Address: fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName),
	}
}

// convertAddresses converts IMAP addresses to email addresses
func (c *Client) convertAddresses(addrs []*imap.Address) []*email.Address {
	if len(addrs) == 0 {
		return nil
	}
	
	result := make([]*email.Address, 0, len(addrs))
	for _, addr := range addrs {
		if converted := c.convertAddress(addr); converted != nil {
			result = append(result, converted)
		}
	}
	
	return result
}