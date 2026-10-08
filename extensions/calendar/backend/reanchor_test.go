package backend

import (
	"database/sql"
	"testing"
	"time"
)

// TestSetDisplayTimezoneReanchorsTzlessTimes checks that changing the display
// timezone moves stored all-day times and override keys to the new zone and
// leaves events with an explicit TZID alone.
func TestSetDisplayTimezoneReanchorsTzlessTimes(t *testing.T) {
	tokyo, err1 := time.LoadLocation("Asia/Tokyo")
	ny, err2 := time.LoadLocation("America/New_York")
	if err1 != nil || err2 != nil {
		t.Skip("tz data unavailable")
	}
	t.Cleanup(func() { SetConfiguredTimezone("") })

	store := newTestStore(t)
	_, calID := seedGoogleSource(t, store, "")
	api := NewAPI(store, nil, nil, nil)
	if _, err := api.SetDisplayTimezone("Asia/Tokyo"); err != nil {
		t.Fatalf("set tz: %v", err)
	}

	const allDay = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//t//t//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:a@t\r\nDTSTAMP:20261001T000000Z\r\nSUMMARY:Daily\r\n" +
		"DTSTART;VALUE=DATE:20261005\r\nDTEND;VALUE=DATE:20261006\r\nRRULE:FREQ=DAILY;COUNT=5\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:a@t\r\nDTSTAMP:20261001T000000Z\r\nSUMMARY:Moved\r\n" +
		"RECURRENCE-ID;VALUE=DATE:20261006\r\nDTSTART;VALUE=DATE:20261006\r\nDTEND;VALUE=DATE:20261007\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"
	const zoned = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//t//t//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:z@t\r\nDTSTAMP:20261001T000000Z\r\nSUMMARY:Call\r\n" +
		"DTSTART;TZID=Europe/Berlin:20261005T090000\r\nDTEND;TZID=Europe/Berlin:20261005T100000\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	store2 := func(id, blob string) Event {
		parsed, err := ParseCalendarObject(blob)
		if err != nil {
			t.Fatalf("parse %s: %v", id, err)
		}
		ev := parsed.Master
		ev.ID, ev.CalendarID, ev.ETag, ev.Href = id, calID, "e", id+".ics"
		err = store.WithTx(func(tx *sql.Tx) error {
			if err := store.UpsertEventTx(tx, ev); err != nil {
				return err
			}
			for _, ov := range parsed.Overrides {
				if err := store.UpsertOverrideTx(tx, id, ov.RecurrenceIDUnix, ov.ICSBlob); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("store %s: %v", id, err)
		}
		return ev
	}
	store2("ev-allday", allDay)
	before := store2("ev-zoned", zoned)

	if got := mustGetEvent(t, store, "ev-allday").DTStartUnix; got != time.Date(2026, 10, 5, 0, 0, 0, 0, tokyo).Unix() {
		t.Fatalf("seeded all-day start = %v, want Tokyo midnight", time.Unix(got, 0))
	}

	changed, err := api.SetDisplayTimezone("America/New_York")
	if err != nil || !changed {
		t.Fatalf("SetDisplayTimezone = %v, %v; want changed", changed, err)
	}

	ev := mustGetEvent(t, store, "ev-allday")
	if want := time.Date(2026, 10, 5, 0, 0, 0, 0, ny).Unix(); ev.DTStartUnix != want {
		t.Errorf("all-day start = %v, want %v", time.Unix(ev.DTStartUnix, 0).In(ny), time.Unix(want, 0).In(ny))
	}
	if want := time.Date(2026, 10, 6, 0, 0, 0, 0, ny).Unix(); ev.DTEndUnix != want {
		t.Errorf("all-day end = %v, want %v", time.Unix(ev.DTEndUnix, 0).In(ny), time.Unix(want, 0).In(ny))
	}
	ovs, err := store.ListOverrides("ev-allday")
	if err != nil || len(ovs) != 1 {
		t.Fatalf("overrides = %v, %v", ovs, err)
	}
	if want := time.Date(2026, 10, 6, 0, 0, 0, 0, ny).Unix(); ovs[0].RecurrenceIDUnix != want {
		t.Errorf("override key = %v, want %v", time.Unix(ovs[0].RecurrenceIDUnix, 0).In(ny), time.Unix(want, 0).In(ny))
	}
	if z := mustGetEvent(t, store, "ev-zoned"); z.DTStartUnix != before.DTStartUnix {
		t.Errorf("zoned event moved: %v, want %v", z.DTStartUnix, before.DTStartUnix)
	}

	if changed, err := api.SetDisplayTimezone("America/New_York"); err != nil || changed {
		t.Errorf("same tz again = %v, %v; want unchanged", changed, err)
	}
}

func mustGetEvent(t *testing.T, store *Store, id string) *Event {
	t.Helper()
	ev, err := store.GetEvent(id)
	if err != nil || ev == nil {
		t.Fatalf("GetEvent(%s) = %v, %v", id, ev, err)
	}
	return ev
}

// TestExpandSkipsCancelledMaster checks that a master VEVENT with
// STATUS:CANCELLED yields no instances, for single and recurring events.
func TestExpandSkipsCancelledMaster(t *testing.T) {
	const head = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//t//t//EN\r\nBEGIN:VEVENT\r\nUID:c@t\r\nDTSTAMP:20261001T000000Z\r\n" +
		"DTSTART:20261005T090000Z\r\nDTEND:20261005T100000Z\r\n"
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, extra string
		want        int
	}{
		{"single", "", 1},
		{"single cancelled", "STATUS:CANCELLED\r\n", 0},
		{"series", "RRULE:FREQ=DAILY;COUNT=3\r\n", 3},
		{"series cancelled", "RRULE:FREQ=DAILY;COUNT=3\r\nSTATUS:CANCELLED\r\n", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseCalendarObject(head + tc.extra + "END:VEVENT\r\nEND:VCALENDAR\r\n")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			insts, err := ExpandInRange(parsed.Master, nil, from, to)
			if err != nil {
				t.Fatalf("expand: %v", err)
			}
			if len(insts) != tc.want {
				t.Errorf("instances = %d, want %d", len(insts), tc.want)
			}
		})
	}
}
