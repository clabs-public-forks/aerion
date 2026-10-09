package message

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Chat list sections.
const (
	ChatSectionAll      = "all"
	ChatSectionUnread   = "unread"
	ChatSectionPriority = "priority"
	ChatSectionLow      = "low"
	ChatSectionSnoozed  = "snoozed"
)

// Sender categories that override header-based bulk classification.
const (
	SenderCategoryPriority = "priority"
	SenderCategoryLow      = "low"
)

// Chat is a conversation row in the chat list, with local triage state.
type Chat struct {
	Conversation
	ThreadKey     string     `json:"threadKey"` // thread id without angle brackets; key for pin/snooze
	IsPinned      bool       `json:"isPinned"`
	SnoozedUntil  *time.Time `json:"snoozedUntil,omitempty"`
	IsLowPriority bool       `json:"isLowPriority"`
	LastFromMe    bool       `json:"lastFromMe"` // latest message is mine: awaiting their reply
	// Recipients names the other side of threads where every sender is me
	// (Sent folder rows).
	Recipients []Address `json:"recipients,omitempty"`
}

// ChatState is the local pin/snooze state of one thread. Nil fields are unset.
type ChatState struct {
	PinnedAt     *time.Time
	SnoozedUntil *time.Time
	SnoozedAt    *time.Time
}

func (st ChatState) empty() bool {
	return st.PinnedAt == nil && st.SnoozedUntil == nil
}

// DueSnooze is a snoozed thread whose snooze has ended.
type DueSnooze struct {
	AccountID  string
	ThreadKey  string
	WokeByMail bool // a new inbox message arrived during the snooze

	// Latest inbox message of the thread; empty when none remain.
	LatestMessageID string
	FolderID        string
	ThreadID        string
	Subject         string
	FromName        string
	FromEmail       string
}

// threadKeyExpr is the normalized thread key of a messages row, matching
// GetConversation's normalization.
func threadKeyExpr(prefix string) string {
	return fmt.Sprintf("REPLACE(REPLACE(COALESCE(%[1]sthread_id, %[1]sid), '<', ''), '>', '')", prefix)
}

// threadKeyMatch is an index-friendly condition selecting a thread's
// messages by key; it takes args threadKeyArgs(key).
func threadKeyMatch(prefix string) string {
	return fmt.Sprintf("(%[1]sthread_id IN (?, ?) OR (%[1]sthread_id IS NULL AND %[1]sid = ?))", prefix)
}

func threadKeyArgs(key string) []any {
	return []any{key, "<" + key + ">", key}
}

// isLowExpr classifies one message: a sender override wins, otherwise is_bulk.
const isLowExpr = `CASE sc.category WHEN 'low' THEN 1 WHEN 'priority' THEN 0 ELSE COALESCE(m.is_bulk, 0) END`

// snoozeActiveExpr is true while a thread's snooze runs: not yet due, and no
// message in scope arrived after it started. received_at is stored as UTC
// text, so its first 19 characters compare against snoozed_at as text.
const snoozeActiveExpr = `(cs.snoozed_until IS NOT NULL AND cs.snoozed_until > ?
	AND COALESCE(g.latest_received, '') <= strftime('%Y-%m-%d %H:%M:%S', COALESCE(cs.snoozed_at, 0), 'unixepoch'))`

// recipientsJSONExpr aggregates the To lists of a group's Sent messages for
// parseAggregatedToListJSON; it needs the folders join as f.
const recipientsJSONExpr = `json_group_array(json(CASE WHEN f.folder_type = 'sent' AND json_valid(m.to_list) THEN m.to_list ELSE '[]' END))`

// chatListColumns are the per-thread columns ListChats scans.
var chatListColumns = `
				MIN(COALESCE(m.thread_id, m.id)) AS conv_thread_id,
				` + threadKeyExpr("m.") + ` AS thread_key,
				COALESCE(MIN(m.subject), '') AS subject,
				MAX(m.snippet) AS snippet,
				COUNT(*) AS message_count,
				SUM(CASE WHEN m.is_read = 0 THEN 1 ELSE 0 END) AS unread_count,
				MAX(CASE WHEN m.has_attachments = 1 THEN 1 ELSE 0 END) AS has_attachments,
				MAX(CASE WHEN m.is_starred = 1 THEN 1 ELSE 0 END) AS is_starred,
				MAX(m.date) AS latest_date,
				GROUP_CONCAT(m.id) AS message_ids,
				MAX(CASE WHEN m.smime_encrypted = 1 OR m.pgp_encrypted = 1 THEN 1 ELSE 0 END) AS is_encrypted,
				a.id AS account_id,
				a.name AS account_name,
				a.color AS account_color,
				MIN(f.id) AS folder_id,
				json_group_array(json_object('name', m.from_name, 'email', m.from_email, 'date', m.date, 'snippet', m.snippet)) AS participants_json,
				MIN(` + isLowExpr + `) AS is_low,
				MAX(REPLACE(SUBSTR(m.received_at, 1, 19), 'T', ' ')) AS latest_received,
				` + recipientsJSONExpr + ` AS recipients_json
`

// chatCountColumns are the columns chatBaseQuery's GROUP BY and section
// filters need; counting skips the JSON aggregates.
var chatCountColumns = `
				` + threadKeyExpr("m.") + ` AS thread_key,
				a.id AS account_id,
				SUM(CASE WHEN m.is_read = 0 THEN 1 ELSE 0 END) AS unread_count,
				MIN(` + isLowExpr + `) AS is_low,
				MAX(REPLACE(SUBSTR(m.received_at, 1, 19), 'T', ' ')) AS latest_received
`

// chatBaseQuery builds the grouped chat query for a scope (a folder ID, or
// "" for every enabled account's inbox) and section, selecting columns from
// each thread group; callers add ordering and paging or wrap it in a count.
func chatBaseQuery(scope, section, columns string, now time.Time) (string, []any, error) {
	folderCond := "f.folder_type = 'inbox'"
	var args []any
	if scope != "" {
		folderCond = "f.id = ?"
		args = append(args, scope)
	}

	var where string
	switch section {
	case ChatSectionAll, "":
		where = "NOT " + snoozeActiveExpr
	case ChatSectionUnread:
		where = "NOT " + snoozeActiveExpr + " AND g.unread_count > 0"
	case ChatSectionPriority:
		where = "NOT " + snoozeActiveExpr + " AND g.is_low = 0"
	case ChatSectionLow:
		where = "NOT " + snoozeActiveExpr + " AND g.is_low = 1"
	case ChatSectionSnoozed:
		where = snoozeActiveExpr
	default:
		return "", nil, fmt.Errorf("unknown chat section %q", section)
	}
	args = append(args, now.Unix())

	// A thread is low priority only when every message in scope is, so one
	// human reply in a newsletter thread lifts it into Chats.
	query := `
		SELECT g.*, cs.pinned_at, cs.snoozed_until
		FROM (
			SELECT ` + columns + `
			FROM messages m
			INNER JOIN folders f ON m.folder_id = f.id AND ` + folderCond + `
			INNER JOIN accounts a ON f.account_id = a.id AND a.enabled = 1
			LEFT JOIN sender_category sc ON sc.account_id = a.id AND sc.email = LOWER(m.from_email)
			GROUP BY thread_key, a.id
		) g
		LEFT JOIN conversation_state cs ON cs.account_id = g.account_id AND cs.thread_key = g.thread_key
		WHERE ` + where
	return query, args, nil
}

// ListChats returns a page of chats for a scope (folder ID, or "" for the
// unified inbox) and section. Pinned chats sort first; the snoozed section
// sorts by wake time. Snoozed chats appear only in the snoozed section.
func (s *Store) ListChats(scope, section string, now time.Time, offset, limit int) ([]*Chat, error) {
	query, args, err := chatBaseQuery(scope, section, chatListColumns, now)
	if err != nil {
		return nil, err
	}
	order := " ORDER BY (cs.pinned_at IS NOT NULL) DESC, g.latest_date DESC"
	if section == ChatSectionSnoozed {
		order = " ORDER BY cs.snoozed_until ASC"
	}
	query += order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats: %w", err)
	}
	defer rows.Close()

	var chats []*Chat
	for rows.Next() {
		c := &Chat{}
		var snippet, latestDate, messageIDs, participantsJSON, latestReceived, recipientsJSON sql.NullString
		var pinnedAt, snoozedUntil sql.NullInt64
		if err := rows.Scan(
			&c.ThreadID, &c.ThreadKey, &c.Subject, &snippet, &c.MessageCount, &c.UnreadCount,
			&c.HasAttachments, &c.IsStarred, &latestDate, &messageIDs, &c.IsEncrypted,
			&c.AccountID, &c.AccountName, &c.AccountColor, &c.FolderID, &participantsJSON,
			&c.IsLowPriority, &latestReceived, &recipientsJSON, &pinnedAt, &snoozedUntil,
		); err != nil {
			return nil, fmt.Errorf("failed to scan chat: %w", err)
		}
		c.Snippet = snippet.String
		if latestDate.String != "" {
			c.LatestDate = parseTimeString(latestDate.String)
		}
		if messageIDs.String != "" {
			c.MessageIDs = strings.Split(messageIDs.String, ",")
		}
		var latestSnippet string
		c.Participants, latestSnippet = parseParticipantsJSON(participantsJSON.String)
		if latestSnippet != "" {
			c.Snippet = latestSnippet
		}
		c.Recipients = parseAggregatedToListJSON(recipientsJSON.String)
		c.IsPinned = pinnedAt.Valid
		if snoozedUntil.Valid && section == ChatSectionSnoozed {
			t := time.Unix(snoozedUntil.Int64, 0).UTC()
			c.SnoozedUntil = &t
		}
		chats = append(chats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate chats: %w", err)
	}

	if err := s.fillLastFromMe(chats); err != nil {
		return nil, err
	}
	return chats, nil
}

// ChatSearchResult is a folder search result with the chat row's recipients.
type ChatSearchResult struct {
	ConversationSearchResult
	// Recipients names the other side of threads in a Sent folder, as
	// Chat.Recipients does.
	Recipients []Address `json:"recipients,omitempty"`
}

// SearchChats runs SearchConversations for a folder and, in a Sent folder,
// adds each thread's recipients so rows name them instead of me.
func (s *Store) SearchChats(folderID, query string, offset, limit int, filter string) ([]*ChatSearchResult, error) {
	found, _, err := s.SearchConversations(folderID, query, offset, limit, filter)
	if err != nil || len(found) == 0 {
		return nil, err
	}
	results := make([]*ChatSearchResult, len(found))
	keys := make([]string, len(found))
	unbracket := strings.NewReplacer("<", "", ">", "")
	for i, c := range found {
		results[i] = &ChatSearchResult{ConversationSearchResult: *c}
		keys[i] = unbracket.Replace(c.ThreadID)
	}
	if found[0].FolderType != "sent" {
		return results, nil
	}
	recipients, err := s.threadRecipients(folderID, keys)
	if err != nil {
		return nil, err
	}
	for i, r := range results {
		r.Recipients = recipients[keys[i]]
	}
	return results, nil
}

// threadRecipients returns the Sent recipients of threads in a folder, by
// thread key, aggregated as in the chat list.
func (s *Store) threadRecipients(folderID string, keys []string) (map[string][]Address, error) {
	conds := make([]string, len(keys))
	args := []any{folderID}
	for i, key := range keys {
		conds[i] = threadKeyMatch("m.")
		args = append(args, threadKeyArgs(key)...)
	}
	query := `
		SELECT ` + threadKeyExpr("m.") + ` AS thread_key, ` + recipientsJSONExpr + `
		FROM messages m
		INNER JOIN folders f ON m.folder_id = f.id
		WHERE m.folder_id = ? AND (` + strings.Join(conds, " OR ") + `)
		GROUP BY thread_key`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query thread recipients: %w", err)
	}
	defer rows.Close()

	recipients := make(map[string][]Address, len(keys))
	for rows.Next() {
		var key string
		var recipientsJSON sql.NullString
		if err := rows.Scan(&key, &recipientsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan thread recipients: %w", err)
		}
		recipients[key] = parseAggregatedToListJSON(recipientsJSON.String)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate thread recipients: %w", err)
	}
	return recipients, nil
}

// CountChats returns the number of chats ListChats would return unpaged.
func (s *Store) CountChats(scope, section string, now time.Time) (int, error) {
	query, args, err := chatBaseQuery(scope, section, chatCountColumns, now)
	if err != nil {
		return 0, err
	}
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM ("+query+")", args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("failed to count chats: %w", err)
	}
	return n, nil
}

// fillLastFromMe marks chats whose latest message is mine: either a Sent copy
// at least as new as the latest listed message, or a listed message from one
// of the account's identities.
func (s *Store) fillLastFromMe(chats []*Chat) error {
	if len(chats) == 0 {
		return nil
	}

	mine := map[string]map[string]bool{} // account -> lowercased identity emails
	rows, err := s.db.Query(`
		SELECT id, LOWER(email) FROM accounts
		UNION SELECT account_id, LOWER(email) FROM identities`)
	if err != nil {
		return fmt.Errorf("failed to load identities: %w", err)
	}
	for rows.Next() {
		var accountID, email string
		if err := rows.Scan(&accountID, &email); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan identity: %w", err)
		}
		if mine[accountID] == nil {
			mine[accountID] = map[string]bool{}
		}
		mine[accountID][email] = true
	}
	rows.Close()

	// Latest Sent date per (account, thread key) for the chats on this page,
	// matched per thread so the thread_id index is used.
	conds := make([]string, 0, len(chats))
	args := make([]any, 0, len(chats)*4)
	for _, c := range chats {
		conds = append(conds, "(m.account_id = ? AND "+threadKeyMatch("m.")+")")
		args = append(args, c.AccountID)
		args = append(args, threadKeyArgs(c.ThreadKey)...)
	}
	rows, err = s.db.Query(`
		SELECT m.account_id, `+threadKeyExpr("m.")+` AS k, m.date
		FROM messages m
		INNER JOIN folders f ON m.folder_id = f.id AND f.folder_type = 'sent'
		WHERE `+strings.Join(conds, " OR "), args...)
	if err != nil {
		return fmt.Errorf("failed to query sent replies: %w", err)
	}
	type chatRef struct{ accountID, key string }
	latestSent := map[chatRef]time.Time{}
	for rows.Next() {
		var accountID, key string
		var date sql.NullString
		if err := rows.Scan(&accountID, &key, &date); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan sent reply: %w", err)
		}
		t := parseTimeString(date.String)
		if ref := (chatRef{accountID, key}); t.After(latestSent[ref]) {
			latestSent[ref] = t
		}
	}
	rows.Close()

	for _, c := range chats {
		if sent, ok := latestSent[chatRef{c.AccountID, c.ThreadKey}]; ok && !sent.Before(c.LatestDate) {
			c.LastFromMe = true
			continue
		}
		if len(c.Participants) > 0 && mine[c.AccountID][strings.ToLower(c.Participants[0].Email)] {
			c.LastFromMe = true
		}
	}
	return nil
}

// GetChatState returns a thread's pin/snooze state (zero when unset).
func (s *Store) GetChatState(accountID, threadKey string) (ChatState, error) {
	var st ChatState
	var pinnedAt, snoozedUntil, snoozedAt sql.NullInt64
	err := s.db.QueryRow(`
		SELECT pinned_at, snoozed_until, snoozed_at FROM conversation_state
		WHERE account_id = ? AND thread_key = ?`,
		accountID, normalizeMessageID(threadKey),
	).Scan(&pinnedAt, &snoozedUntil, &snoozedAt)
	if err == sql.ErrNoRows {
		return st, nil
	}
	if err != nil {
		return st, fmt.Errorf("failed to get chat state: %w", err)
	}
	st.PinnedAt = unixPtr(pinnedAt)
	st.SnoozedUntil = unixPtr(snoozedUntil)
	st.SnoozedAt = unixPtr(snoozedAt)
	return st, nil
}

// SetChatState replaces a thread's pin/snooze state; an empty state deletes the row.
func (s *Store) SetChatState(accountID, threadKey string, st ChatState) error {
	threadKey = normalizeMessageID(threadKey)
	if threadKey == "" {
		return fmt.Errorf("empty thread key")
	}
	if st.empty() {
		if _, err := s.db.Exec(`DELETE FROM conversation_state WHERE account_id = ? AND thread_key = ?`, accountID, threadKey); err != nil {
			return fmt.Errorf("failed to clear chat state: %w", err)
		}
		return nil
	}
	switch {
	case st.SnoozedUntil == nil:
		st.SnoozedAt = nil
	case st.SnoozedAt == nil:
		// Without a start, any inbox mail would count as newer and end the snooze.
		now := time.Now()
		st.SnoozedAt = &now
	}
	_, err := s.db.Exec(`
		INSERT INTO conversation_state (account_id, thread_key, pinned_at, snoozed_until, snoozed_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(account_id, thread_key) DO UPDATE SET
			pinned_at = excluded.pinned_at,
			snoozed_until = excluded.snoozed_until,
			snoozed_at = excluded.snoozed_at`,
		accountID, threadKey, unixArg(st.PinnedAt), unixArg(st.SnoozedUntil), unixArg(st.SnoozedAt))
	if err != nil {
		return fmt.Errorf("failed to set chat state: %w", err)
	}
	return nil
}

// NextSnoozeWake returns the earliest snooze end, or nil when nothing is snoozed.
func (s *Store) NextSnoozeWake() (*time.Time, error) {
	var until sql.NullInt64
	if err := s.db.QueryRow(`SELECT MIN(snoozed_until) FROM conversation_state`).Scan(&until); err != nil {
		return nil, fmt.Errorf("failed to get next snooze: %w", err)
	}
	return unixPtr(until), nil
}

// ListDueSnoozes returns snoozed threads whose snooze ended at or before now,
// with the latest inbox message of each.
func (s *Store) ListDueSnoozes(now time.Time) ([]DueSnooze, error) {
	type due struct {
		accountID, threadKey string
		snoozedAt            int64
	}
	rows, err := s.db.Query(`
		SELECT account_id, thread_key, COALESCE(snoozed_at, 0) FROM conversation_state
		WHERE snoozed_until IS NOT NULL AND snoozed_until <= ?`, now.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to query due snoozes: %w", err)
	}
	var dues []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.accountID, &d.threadKey, &d.snoozedAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan due snooze: %w", err)
		}
		dues = append(dues, d)
	}
	rows.Close()

	result := make([]DueSnooze, 0, len(dues))
	for _, d := range dues {
		ds := DueSnooze{AccountID: d.accountID, ThreadKey: d.threadKey}
		var threadID, fromName, fromEmail sql.NullString
		var woke bool
		args := append([]any{d.snoozedAt, d.accountID}, threadKeyArgs(d.threadKey)...)
		err := s.db.QueryRow(`
			SELECT m.id, m.folder_id, COALESCE(m.thread_id, m.id), COALESCE(m.subject, ''), m.from_name, m.from_email,
				REPLACE(SUBSTR(m.received_at, 1, 19), 'T', ' ') > strftime('%Y-%m-%d %H:%M:%S', ?, 'unixepoch')
			FROM messages m
			INNER JOIN folders f ON m.folder_id = f.id AND f.folder_type = 'inbox'
			WHERE m.account_id = ? AND `+threadKeyMatch("m.")+`
			ORDER BY m.received_at DESC, m.date DESC
			LIMIT 1`, args...,
		).Scan(&ds.LatestMessageID, &ds.FolderID, &threadID, &ds.Subject, &fromName, &fromEmail, &woke)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to get snoozed thread message: %w", err)
		}
		ds.ThreadID, ds.FromName, ds.FromEmail, ds.WokeByMail = threadID.String, fromName.String, fromEmail.String, woke
		result = append(result, ds)
	}
	return result, nil
}

// CleanupChatState deletes pin/snooze rows whose thread has no messages left.
func (s *Store) CleanupChatState() (int64, error) {
	res, err := s.db.Exec(`
		DELETE FROM conversation_state
		WHERE NOT EXISTS (
			SELECT 1 FROM messages m
			WHERE m.account_id = conversation_state.account_id
				AND (m.thread_id IN (conversation_state.thread_key, '<' || conversation_state.thread_key || '>')
					OR (m.thread_id IS NULL AND m.id = conversation_state.thread_key))
		)`)
	if err != nil {
		return 0, fmt.Errorf("failed to clean up chat state: %w", err)
	}
	return res.RowsAffected()
}

// SetSenderCategory overrides bulk classification for a sender; an empty
// category removes the override.
func (s *Store) SetSenderCategory(accountID, email, category string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return fmt.Errorf("empty sender email")
	}
	if category != "" && category != SenderCategoryPriority && category != SenderCategoryLow {
		return fmt.Errorf("unknown sender category %q", category)
	}
	if category == "" {
		_, err := s.db.Exec(`DELETE FROM sender_category WHERE account_id = ? AND email = ?`, accountID, email)
		if err != nil {
			return fmt.Errorf("failed to clear sender category: %w", err)
		}
		return nil
	}
	_, err := s.db.Exec(`
		INSERT INTO sender_category (account_id, email, category) VALUES (?, ?, ?)
		ON CONFLICT(account_id, email) DO UPDATE SET category = excluded.category`,
		accountID, email, category)
	if err != nil {
		return fmt.Errorf("failed to set sender category: %w", err)
	}
	return nil
}

// UnclassifiedMessage is a message whose bulk flag is not yet known.
type UnclassifiedMessage struct {
	ID        string
	UID       uint32
	FromEmail string
}

// ListUnclassifiedBulk returns up to limit messages in a folder with NULL
// is_bulk and a UID above afterUID, in UID order.
func (s *Store) ListUnclassifiedBulk(folderID string, afterUID uint32, limit int) ([]UnclassifiedMessage, error) {
	rows, err := s.db.Query(`
		SELECT id, uid, from_email FROM messages
		WHERE folder_id = ? AND is_bulk IS NULL AND uid > ?
		ORDER BY uid LIMIT ?`, folderID, afterUID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list unclassified messages: %w", err)
	}
	defer rows.Close()
	var out []UnclassifiedMessage
	for rows.Next() {
		var m UnclassifiedMessage
		var from sql.NullString
		if err := rows.Scan(&m.ID, &m.UID, &from); err != nil {
			return nil, fmt.Errorf("failed to scan unclassified message: %w", err)
		}
		m.FromEmail = from.String
		out = append(out, m)
	}
	return out, rows.Err()
}

// SetBulkFlags stores bulk classifications by message ID.
func (s *Store) SetBulkFlags(flags map[string]bool) error {
	if len(flags) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin bulk flag update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`UPDATE messages SET is_bulk = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare bulk flag update: %w", err)
	}
	defer stmt.Close()
	for id, bulk := range flags {
		if _, err := stmt.Exec(bulk, id); err != nil {
			return fmt.Errorf("failed to set bulk flag: %w", err)
		}
	}
	return tx.Commit()
}

func unixPtr(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := time.Unix(v.Int64, 0).UTC()
	return &t
}

func unixArg(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Unix()
}

// NewMail is a recently arrived message, as new-mail notifications see it.
type NewMail struct {
	UID       uint32
	ThreadID  string
	Subject   string
	FromName  string
	FromEmail string
	IsLow     bool // sender override, else is_bulk
}

// newMailChunk keeps ListNewMail's IN list well under SQLite's variable limit.
const newMailChunk = 500

// ListNewMail returns the messages with the given IDs, highest UID first,
// each classified like chat rows (isLowExpr). Unknown IDs are skipped.
func (s *Store) ListNewMail(ids []string) ([]NewMail, error) {
	var out []NewMail
	for start := 0; start < len(ids); start += newMailChunk {
		chunk := ids[start:min(start+newMailChunk, len(ids))]
		args := make([]any, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		rows, err := s.db.Query(`
			SELECT m.uid, COALESCE(m.thread_id, m.id), COALESCE(m.subject, ''), COALESCE(m.from_name, ''), COALESCE(m.from_email, ''),
				`+isLowExpr+`
			FROM messages m
			INNER JOIN folders f ON f.id = m.folder_id
			LEFT JOIN sender_category sc ON sc.account_id = f.account_id AND sc.email = LOWER(m.from_email)
			WHERE m.id IN (?`+strings.Repeat(", ?", len(chunk)-1)+`)`, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to list new mail: %w", err)
		}
		for rows.Next() {
			var nm NewMail
			if err := rows.Scan(&nm.UID, &nm.ThreadID, &nm.Subject, &nm.FromName, &nm.FromEmail, &nm.IsLow); err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan new mail: %w", err)
			}
			out = append(out, nm)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to list new mail: %w", err)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UID > out[j].UID })
	return out, nil
}
