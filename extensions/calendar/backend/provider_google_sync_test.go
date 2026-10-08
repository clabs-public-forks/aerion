package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-ical"
)

// TestGoogleProvider_SyncExceptionsTombstonesAndFullResync covers K8:
// instance exceptions become overrides (cancelled ones hide the occurrence),
// id-only tombstones delete their event, and the full resync after a 410
// removes events the server no longer lists.
func TestGoogleProvider_SyncExceptionsTombstonesAndFullResync(t *testing.T) {
	master := map[string]any{
		"id": "m1", "iCalUID": "weekly@example.com", "etag": `"m"`, "status": "confirmed", "summary": "Standup",
		"start":      map[string]string{"dateTime": "2026-06-01T14:00:00Z"},
		"end":        map[string]string{"dateTime": "2026-06-01T14:30:00Z"},
		"recurrence": []string{"RRULE:FREQ=WEEKLY;COUNT=4"},
	}
	single := func(id string) map[string]any {
		return map[string]any{
			"id": id, "iCalUID": id + "@example.com", "etag": `"` + id + `"`, "status": "confirmed", "summary": id,
			"start": map[string]string{"dateTime": "2026-06-02T10:00:00Z"},
			"end":   map[string]string{"dateTime": "2026-06-02T11:00:00Z"},
		}
	}
	moved := map[string]any{
		"id": "m1_20260608T140000Z", "iCalUID": "weekly@example.com", "status": "confirmed", "summary": "Standup (moved)",
		"recurringEventId":  "m1",
		"originalStartTime": map[string]string{"dateTime": "2026-06-08T14:00:00Z"},
		"start":             map[string]string{"dateTime": "2026-06-08T16:00:00Z"},
		"end":               map[string]string{"dateTime": "2026-06-08T16:30:00Z"},
	}
	cancelled := map[string]any{
		"id": "m1_20260615T140000Z", "status": "cancelled", "recurringEventId": "m1",
		"originalStartTime": map[string]string{"dateTime": "2026-06-15T14:00:00Z"},
	}

	step := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var body map[string]any
		switch {
		case step == 0 && q.Get("pageToken") == "":
			// Exceptions arrive on the page before their master.
			body = map[string]any{"items": []any{moved, cancelled, single("s1")}, "nextPageToken": "p2"}
		case step == 0:
			body = map[string]any{"items": []any{master, single("s2")}, "nextSyncToken": "tok1"}
		case step == 1:
			body = map[string]any{"items": []any{map[string]any{"id": "s1", "status": "cancelled"}}, "nextSyncToken": "tok2"}
		case q.Get("syncToken") != "":
			w.WriteHeader(http.StatusGone)
			return
		default:
			body = map[string]any{"items": []any{moved, cancelled, master}, "nextSyncToken": "tok3"}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()

	store := newTestStore(t)
	srcID, calID := seedGoogleSource(t, store, "primary")
	p := googleProvider{store: store, auth: fakeAuth{target: srv.URL}}
	src, err := store.GetSource(srcID)
	if err != nil {
		t.Fatalf("GetSource: %v", err)
	}
	sync := func(label string) {
		t.Helper()
		cals, err := store.ListCalendars(srcID)
		if err != nil || len(cals) != 1 {
			t.Fatalf("%s: ListCalendars: %v", label, err)
		}
		if err := p.SyncCalendar(context.Background(), *src, cals[0]); err != nil {
			t.Fatalf("%s: SyncCalendar: %v", label, err)
		}
		step++
	}
	uids := func() map[string]string {
		t.Helper()
		m, err := store.ListEventETags(calID)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	starts := func() []string {
		t.Helper()
		id, err := p.lookupEventIDByUID(calID, "weekly@example.com")
		if err != nil || id == "" {
			t.Fatalf("master not stored: %v", err)
		}
		ev, err := store.GetEvent(id)
		if err != nil {
			t.Fatal(err)
		}
		ovs, err := store.ListOverrides(id)
		if err != nil {
			t.Fatal(err)
		}
		insts, err := ExpandInRange(*ev, ovs, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, in := range insts {
			out = append(out, time.Unix(in.InstanceStartUnix, 0).UTC().Format("01-02T15"))
		}
		return out
	}
	const want = "06-01T14 06-08T16 06-22T14"

	sync("full")
	if got := strings.Join(starts(), " "); got != want {
		t.Errorf("after full sync occurrences = %q, want %q", got, want)
	}
	if _, ok := uids()["s1@example.com"]; !ok {
		t.Fatalf("s1 not stored by the first sync")
	}

	sync("incremental tombstone")
	if _, ok := uids()["s1@example.com"]; ok {
		t.Errorf("id-only tombstone did not delete s1")
	}

	sync("410 then full")
	if _, ok := uids()["s2@example.com"]; ok {
		t.Errorf("full resync kept s2, which the server no longer lists")
	}
	if got := strings.Join(starts(), " "); got != want {
		t.Errorf("after full resync occurrences = %q, want %q", got, want)
	}
}

// TestGoogleTranslate_RecurrenceLinesKeepExdates covers K9: EXDATE and RDATE
// (with their parameters) reach Google, and come back with them intact.
func TestGoogleTranslate_RecurrenceLinesKeepExdates(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//T//EN\r\nBEGIN:VEVENT\r\n" +
		"UID:r@example.com\r\nDTSTAMP:20260101T000000Z\r\n" +
		"DTSTART;TZID=America/New_York:20260601T090000\r\nDTEND;TZID=America/New_York:20260601T093000\r\n" +
		"RRULE:FREQ=DAILY;COUNT=5\r\nEXDATE;TZID=America/New_York:20260603T090000\r\n" +
		"RDATE;VALUE=DATE:20260610\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	out, err := translateICSToGoogleJSON(ics)
	if err != nil {
		t.Fatalf("translateICSToGoogleJSON: %v", err)
	}
	want := []string{
		"RRULE:FREQ=DAILY;COUNT=5",
		"EXDATE;TZID=America/New_York:20260603T090000",
		"RDATE;VALUE=DATE:20260610",
	}
	if strings.Join(out.Recurrence, "|") != strings.Join(want, "|") {
		t.Fatalf("Recurrence = %q, want %q", out.Recurrence, want)
	}

	ev := ical.NewEvent()
	for _, line := range out.Recurrence {
		applyRecurrenceLine(ev, line)
	}
	if got := recurrenceLines(ev); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("round trip = %q, want %q", got, want)
	}
}
