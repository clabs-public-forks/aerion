package database

import (
	"database/sql"
	"fmt"

	"github.com/hkdb/aerion/internal/cid"
)

// backfillEmbeddedAttachments marks the stored inline parts whose
// Content-ID their message's body_html embeds, then recomputes
// has_attachments for those messages from the parts left listed.
func backfillEmbeddedAttachments(tx *sql.Tx) error {
	rows, err := tx.Query(`
		SELECT a.id, a.message_id, a.content_id, m.body_html
		FROM attachments a
		JOIN messages m ON m.id = a.message_id
		WHERE a.is_inline = 1 AND a.content_id IS NOT NULL AND a.content_id != ''
		ORDER BY a.message_id
	`)
	if err != nil {
		return fmt.Errorf("query inline parts: %w", err)
	}
	var embeddedIDs []string
	var lastMessageID string
	var embedded map[string]struct{}
	for rows.Next() {
		var id, messageID, contentID string
		var bodyHTML sql.NullString
		if err := rows.Scan(&id, &messageID, &contentID, &bodyHTML); err != nil {
			rows.Close()
			return fmt.Errorf("scan inline part: %w", err)
		}
		if messageID != lastMessageID {
			lastMessageID = messageID
			embedded = cid.Embedded(bodyHTML.String)
		}
		if _, ok := embedded[contentID]; ok {
			embeddedIDs = append(embeddedIDs, id)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return fmt.Errorf("read inline parts: %w", err)
	}

	markStmt, err := tx.Prepare(`UPDATE attachments SET embedded = 1 WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("prepare mark: %w", err)
	}
	defer markStmt.Close()
	for _, id := range embeddedIDs {
		if _, err := markStmt.Exec(id); err != nil {
			return fmt.Errorf("mark embedded part: %w", err)
		}
	}
	if _, err := tx.Exec(`
		UPDATE messages SET has_attachments = EXISTS (
			SELECT 1 FROM attachments WHERE message_id = messages.id AND embedded = 0
		) WHERE id IN (
			SELECT message_id FROM attachments
			WHERE is_inline = 1 AND content_id IS NOT NULL AND content_id != ''
		)
	`); err != nil {
		return fmt.Errorf("recompute has_attachments: %w", err)
	}
	return nil
}
