package backend

import (
	"database/sql"
	"fmt"

	"github.com/emersion/go-ical"
)

// ReanchorFloatingTimes re-derives the stored instants of events whose times
// carry no timezone (all-day DATEs and floating times) and of their
// RECURRENCE-ID overrides. Those values are interpreted in configuredTZ() at
// parse time, so after the display timezone changes they would otherwise stay
// pinned to the previous zone. Rows whose blob no longer parses are left alone.
func (s *Store) ReanchorFloatingTimes() error {
	return s.WithTx(func(tx *sql.Tx) error {
		rows, err := tx.Query(`SELECT id, ics_blob FROM events WHERE COALESCE(tz_name, '') = ''`)
		if err != nil {
			return fmt.Errorf("list tz-less events: %w", err)
		}
		type row struct{ id, blob string }
		var events []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.blob); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan event: %w", err)
			}
			events = append(events, r)
		}
		_ = rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("list tz-less events: %w", err)
		}

		for _, ev := range events {
			parsed, err := ParseCalendarObject(ev.blob)
			if err != nil {
				continue
			}
			if _, err := tx.Exec(`UPDATE events SET dtstart_unix = ?, dtend_unix = ? WHERE id = ?`,
				parsed.Master.DTStartUnix, parsed.Master.DTEndUnix, ev.id); err != nil {
				return fmt.Errorf("reanchor event: %w", err)
			}
			if err := reanchorOverridesTx(tx, ev.id); err != nil {
				return err
			}
		}
		return nil
	})
}

// reanchorOverridesTx re-keys eventID's overrides by their RECURRENCE-ID as
// read in the current configured timezone.
func reanchorOverridesTx(tx *sql.Tx, eventID string) error {
	rows, err := tx.Query(`SELECT recurrence_id_unix, ics_blob FROM event_recurrence_overrides WHERE event_id = ?`, eventID)
	if err != nil {
		return fmt.Errorf("list overrides: %w", err)
	}
	type rekey struct {
		from, to int64
		blob     string
	}
	var moves []rekey
	for rows.Next() {
		var rid int64
		var blob string
		if err := rows.Scan(&rid, &blob); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan override: %w", err)
		}
		if to, ok := overrideRecurrenceID(blob); ok && to != rid {
			moves = append(moves, rekey{rid, to, blob})
		}
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("list overrides: %w", err)
	}

	// Delete every moved row before reinserting: a zone shift can exceed a
	// day, so one override's new key may equal another's old key.
	for _, m := range moves {
		if _, err := tx.Exec(`DELETE FROM event_recurrence_overrides WHERE event_id = ? AND recurrence_id_unix = ?`,
			eventID, m.from); err != nil {
			return fmt.Errorf("reanchor override: %w", err)
		}
	}
	for _, m := range moves {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO event_recurrence_overrides (event_id, recurrence_id_unix, ics_blob) VALUES (?, ?, ?)`,
			eventID, m.to, m.blob); err != nil {
			return fmt.Errorf("reanchor override: %w", err)
		}
	}
	return nil
}

// overrideRecurrenceID parses a stored override blob and returns its
// RECURRENCE-ID instant in the current configured timezone.
func overrideRecurrenceID(blob string) (int64, bool) {
	cal, err := decodeICS(blob)
	if err != nil {
		return 0, false
	}
	for i := range cal.Events() {
		ev := cal.Events()[i]
		if ev.Props.Get(ical.PropRecurrenceID) == nil {
			continue
		}
		ov, err := buildOverride(&ev)
		if err != nil {
			return 0, false
		}
		return ov.RecurrenceIDUnix, true
	}
	return 0, false
}
