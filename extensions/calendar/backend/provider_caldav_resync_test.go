package backend

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const sampleRecurringWithOverrideICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:weekly-ov@example.com
DTSTAMP:20251101T120000Z
DTSTART:20251103T140000Z
DTEND:20251103T143000Z
SUMMARY:Standup
RRULE:FREQ=WEEKLY
END:VEVENT
BEGIN:VEVENT
UID:weekly-ov@example.com
DTSTAMP:20251101T120000Z
RECURRENCE-ID:20251110T140000Z
DTSTART:20251110T150000Z
DTEND:20251110T153000Z
SUMMARY:Standup (moved)
END:VEVENT
END:VCALENDAR
`

// TestCalDAVProvider_ForceResyncKeepsOverridesAndDeletes covers K7: after
// ClearEventETagsForSource, a resync must reuse the existing row id for
// overrides (not a fresh UUID the events row never gets) and must still
// delete events that vanished from the server.
func TestCalDAVProvider_ForceResyncKeepsOverridesAndDeletes(t *testing.T) {
	response := func(href, etag, ics string) string {
		return `<D:response><D:href>` + href + `</D:href><D:propstat><D:prop><D:getetag>"` + etag +
			`"</D:getetag><C:calendar-data>` + ics + `</C:calendar-data></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response>`
	}
	var dropSecond atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		body := response("/cal/r.ics", "r1", sampleRecurringWithOverrideICS)
		if !dropSecond.Load() {
			body += response("/cal/e1.ics", "e1", sampleNonRecurringICS)
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">` + body + `</D:multistatus>`))
	}))
	defer srv.Close()

	store := newTestStore(t)
	now := time.Now().Unix()
	const srcID, calID = "src-cd", "cal-cd"
	if err := store.WithTx(func(tx *sql.Tx) error {
		if err := store.CreateSourceTx(tx, Source{
			ID: srcID, Type: SourceTypeCalDAV, Name: "dav",
			URL: srv.URL, Username: "u", Enabled: true, Writable: true, CreatedAt: now,
		}); err != nil {
			return err
		}
		return store.CreateCalendarTx(tx, Calendar{
			ID: calID, SourceID: srcID, URL: "/cal/", DisplayName: "Personal", Visible: true, CreatedAt: now,
		})
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p := caldavProvider{store: store, secrets: fakeSecrets{password: "x"}}
	src, err := store.GetSource(srcID)
	if err != nil {
		t.Fatalf("GetSource: %v", err)
	}
	cal := Calendar{ID: calID, SourceID: srcID, URL: "/cal/"}
	if err := p.SyncCalendar(context.Background(), *src, cal); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	if err := store.ClearEventETagsForSource(srcID); err != nil {
		t.Fatalf("ClearEventETagsForSource: %v", err)
	}
	dropSecond.Store(true)
	if err := p.SyncCalendar(context.Background(), *src, cal); err != nil {
		t.Fatalf("resync after clear: %v", err)
	}

	var orphans, overrides int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM event_recurrence_overrides WHERE event_id NOT IN (SELECT id FROM events)`).Scan(&orphans); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM event_recurrence_overrides`).Scan(&overrides); err != nil {
		t.Fatal(err)
	}
	if orphans != 0 || overrides != 1 {
		t.Errorf("overrides = %d, orphans = %d; want 1 override and no orphans", overrides, orphans)
	}

	etags, err := store.ListEventETags(calID)
	if err != nil {
		t.Fatalf("ListEventETags: %v", err)
	}
	if _, ok := etags["non-recurring-1@example.com"]; ok {
		t.Errorf("event removed from the server survived the resync")
	}
	if got := etags["weekly-ov@example.com"]; !strings.Contains(got, "r1") {
		t.Errorf("recurring event etag = %q, want r1", got)
	}
}
