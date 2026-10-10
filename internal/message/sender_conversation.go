package message

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// GetSenderConversation returns a sender chat as one conversation: every
// thread in scope (a folder ID, or "" for the account's inbox) that maps to
// the sender's chat, plus those threads' Sent and Drafts messages, in date
// order. It returns nil when the sender chat has no messages in scope.
func (s *Store) GetSenderConversation(accountID, email, folderID string) (*Conversation, error) {
	start := time.Now()
	email = NormalizeSenderEmail(strings.TrimPrefix(strings.TrimSpace(email), SenderChatPrefix))
	if email == "" {
		return nil, fmt.Errorf("empty sender email")
	}
	cond, scopeArgs := "f.folder_type = 'inbox' AND f.account_id = ?", []any{accountID}
	if folderID != "" {
		cond, scopeArgs = "f.id = ? AND f.account_id = ?", []any{folderID, accountID}
	}
	chatKey := SenderChatKey(email)

	keys, err := s.senderChatThreadKeys(cond, scopeArgs, accountID, email)
	if err != nil || len(keys) == 0 {
		return nil, err
	}

	inScope := map[string]bool{}
	rows, err := s.db.Query(`SELECT f.id FROM folders f WHERE `+cond, scopeArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query scope folders: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan scope folder: %w", err)
		}
		inScope[id] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read scope folders: %w", err)
	}

	// Like GetConversation, also match messages whose Message-ID or
	// In-Reply-To is a thread key, such as a reply stored without a thread ID.
	// Matching the bare and bracketed forms keeps both columns' indexes usable.
	const bracketedKeys = `SELECT value FROM json_each(?) UNION ALL SELECT '<' || value || '>' FROM json_each(?)`
	args := append([]any{accountID}, threadKeysArgs(keys)...)
	j := keysJSON(keys)
	args = append(args, j, j, j, j)
	args = append(args, scopeArgs...)
	rows, err = s.db.Query(`
		SELECT `+conversationColumns+`
		FROM messages m
		INNER JOIN folders f ON m.folder_id = f.id
		WHERE m.account_id = ? AND (`+threadKeysMatch("m.")+`
			OR m.message_id IN (`+bracketedKeys+`)
			OR m.in_reply_to IN (`+bracketedKeys+`))
			AND ((`+cond+`) OR f.folder_type IN ('sent', 'drafts'))
		ORDER BY m.date ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sender chat messages: %w", err)
	}
	defer rows.Close()
	msgs, err := s.scanConversationMessages(rows)
	if err != nil {
		return nil, err
	}
	msgs = dedupeCopies(msgs, func(m *Message) bool { return inScope[m.FolderID] })
	if len(msgs) == 0 {
		return nil, nil
	}

	c := &Conversation{ThreadID: chatKey, AccountID: accountID, FolderID: folderID, Messages: msgs}
	for _, m := range msgs {
		c.MessageIDs = append(c.MessageIDs, m.ID)
		if !m.IsRead {
			c.UnreadCount++
		}
		c.HasAttachments = c.HasAttachments || m.HasAttachments
		c.IsStarred = c.IsStarred || m.IsStarred
		c.IsEncrypted = c.IsEncrypted || m.SMIMEEncrypted || m.PGPEncrypted
	}
	c.MessageCount = len(msgs)
	latest := msgs[len(msgs)-1]
	c.Subject, c.Snippet, c.LatestDate = latest.Subject, latest.Snippet, latest.Date
	c.Participants = conversationParticipants(msgs)

	s.log.Debug().
		Str("chatKey", chatKey).
		Int("threads", len(keys)).
		Int("messageCount", len(msgs)).
		Dur("elapsed", time.Since(start)).
		Msg("GetSenderConversation returning")
	return c, nil
}

// senderChatThreadKeys returns the thread keys in scope cond (with
// scopeArgs) that map to chatKey. Only a combined sender's threads can, and
// only those holding a message from the sender, so the starter check runs
// over those threads alone.
func (s *Store) senderChatThreadKeys(cond string, scopeArgs []any, accountID, email string) ([]string, error) {
	combined, err := s.IsSenderChat(accountID, email)
	if err != nil || !combined {
		return nil, err
	}
	key := threadKeyExpr("m.")
	candidates, err := s.queryStrings(`
		SELECT DISTINCT `+key+` FROM messages m
		INNER JOIN folders f ON m.folder_id = f.id
		WHERE `+cond+` AND LOWER(m.from_email) = ?`, append(slices.Clone(scopeArgs), email)...)
	if err != nil || len(candidates) == 0 {
		return nil, err
	}
	args := append(slices.Clone(scopeArgs), threadKeysArgs(candidates)...)
	return s.queryStrings(`
		SELECT thread_key FROM (
			SELECT `+key+` AS thread_key, LOWER(m.from_email) AS sender,
				ROW_NUMBER() OVER (PARTITION BY `+key+` ORDER BY m.date, m.id) AS n
			FROM messages m
			INNER JOIN folders f ON m.folder_id = f.id
			WHERE `+cond+` AND `+threadKeysMatch("m.")+`
		) WHERE n = 1 AND sender = ?`, append(args, email)...)
}

// queryStrings runs a query selecting one text column.
func (s *Store) queryStrings(query string, args ...any) ([]string, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sender chat threads: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("failed to scan thread key: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
