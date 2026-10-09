package backend

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

type alarmRow struct {
	instance, trigger int64
	status            string
}

func listAlarmRows(t *testing.T, store *Store, eventID string) []alarmRow {
	t.Helper()
	rows, err := store.DB().Query(`
		SELECT instance_unix, trigger_unix, status FROM event_alarms
		WHERE event_id = ? ORDER BY trigger_unix`, eventID)
	if err != nil {
		t.Fatalf("query alarms: %v", err)
	}
	defer rows.Close()
	var out []alarmRow
	for rows.Next() {
		var r alarmRow
		if err := rows.Scan(&r.instance, &r.trigger, &r.status); err != nil {
			t.Fatalf("scan alarm: %v", err)
		}
		out = append(out, r)
	}
	return out
}

func hasAlarm(rows []alarmRow, want alarmRow) bool {
	for _, r := range rows {
		if r == want {
			return true
		}
	}
	return false
}

func TestRefreshAllAlarms(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	day := func(n int, hour int) int64 {
		return now.AddDate(0, 0, n).Add(time.Duration(hour) * time.Hour).Unix()
	}
	masterICS := wrapICS(`BEGIN:VEVENT
UID:daily
DTSTART:20260601T140000Z
DTEND:20260601T150000Z
SUMMARY:Standup
RRULE:FREQ=DAILY
BEGIN:VALARM
ACTION:DISPLAY
TRIGGER:-PT30M
END:VALARM
END:VEVENT`)
	// The June 3 occurrence moves to 16:00 and carries its own 5-minute alarm.
	movedOverrideICS := wrapICS(`BEGIN:VEVENT
UID:daily
RECURRENCE-ID:20260603T140000Z
DTSTART:20260603T160000Z
DTEND:20260603T170000Z
SUMMARY:Standup (moved)
BEGIN:VALARM
ACTION:DISPLAY
TRIGGER:-PT5M
END:VALARM
END:VEVENT`)

	tests := []struct {
		name      string
		overrides map[int64]string
		seed      []alarmRow
		want      []alarmRow
		dontWant  []alarmRow
	}{
		{
			name: "occurrence more than 7 days out gets an alarm",
			want: []alarmRow{
				{day(0, 14), day(0, 14) - 30*60, "pending"},
				{day(8, 14), day(8, 14) - 30*60, "pending"},
			},
		},
		{
			name:      "moved override uses its own template",
			overrides: map[int64]string{day(2, 14): movedOverrideICS},
			want:      []alarmRow{{day(2, 16), day(2, 16) - 5*60, "pending"}},
			dontWant: []alarmRow{
				{day(2, 14), day(2, 14) - 30*60, "pending"},
				{day(2, 16), day(2, 16) - 30*60, "pending"},
			},
		},
		{
			name: "stale pending alarms are removed, fired ones kept",
			seed: []alarmRow{
				{day(1, 9), day(1, 9) - 30*60, "pending"},
				{day(1, 10), day(1, 10) - 30*60, "fired"},
			},
			want: []alarmRow{
				{day(1, 10), day(1, 10) - 30*60, "fired"},
				{day(1, 14), day(1, 14) - 30*60, "pending"},
			},
			dontWant: []alarmRow{{day(1, 9), day(1, 9) - 30*60, "pending"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestStore(t)
			_, calID := seedGoogleSource(t, store, "primary")
			ev := Event{
				ID:          "ev-daily",
				CalendarID:  calID,
				UID:         "daily",
				Summary:     "Standup",
				DTStartUnix: day(0, 14),
				DTEndUnix:   day(0, 15),
				RRuleText:   "RRULE:FREQ=DAILY",
				ICSBlob:     masterICS,
			}
			err := store.WithTx(func(tx *sql.Tx) error {
				if err := store.UpsertEventTx(tx, ev); err != nil {
					return err
				}
				for recID, blob := range tc.overrides {
					if err := store.UpsertOverrideTx(tx, ev.ID, recID, blob); err != nil {
						return err
					}
				}
				for i, r := range tc.seed {
					if _, err := tx.Exec(`
						INSERT INTO event_alarms
							(id, event_id, instance_unix, trigger_unix, status, action, created_at)
						VALUES (?, ?, ?, ?, ?, 'display', 0)`,
						"seed-"+string(rune('a'+i)), ev.ID, r.instance, r.trigger, r.status); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("seed: %v", err)
			}

			if err := RefreshAllAlarms(context.Background(), store, now); err != nil {
				t.Fatalf("RefreshAllAlarms: %v", err)
			}
			got := listAlarmRows(t, store, ev.ID)
			for _, w := range tc.want {
				if !hasAlarm(got, w) {
					t.Errorf("missing alarm %+v in %+v", w, got)
				}
			}
			for _, d := range tc.dontWant {
				if hasAlarm(got, d) {
					t.Errorf("unexpected alarm %+v", d)
				}
			}
			for _, r := range got {
				if r.status == "pending" && r.trigger < now.Unix() {
					t.Errorf("pending alarm in the past: %+v", r)
				}
			}
		})
	}
}
