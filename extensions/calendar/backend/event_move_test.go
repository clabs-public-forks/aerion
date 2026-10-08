package backend

import (
	"testing"
	"time"
)

// TestMoveEventKeepsFields covers K10: a drag or resize changes only the
// times and keeps the fields the timeline doesn't know about.
func TestMoveEventKeepsFields(t *testing.T) {
	api, calID := newLocalAPI(t)
	start := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC).Unix()
	id, err := api.CreateEvent(EventInput{
		CalendarID: calID, Summary: "Review", TZName: "UTC",
		Description: "line one\nline two", DescriptionHTML: "<p>line <b>one</b></p>",
		Location: "Room 4", DTStartUnix: start, DTEndUnix: start + 1800,
		Transparency: "free", Visibility: "private",
		Reminder: &ReminderSpec{OffsetMinutes: 25},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := api.MoveEvent(id, start+3600, start+7200, ""); err != nil {
		t.Fatalf("MoveEvent: %v", err)
	}

	ev, err := api.store.GetEvent(id)
	if err != nil {
		t.Fatal(err)
	}
	if ev.DTStartUnix != start+3600 || ev.DTEndUnix != start+7200 {
		t.Errorf("times = %d-%d, want %d-%d", ev.DTStartUnix, ev.DTEndUnix, start+3600, start+7200)
	}
	if ev.Summary != "Review" || ev.Location != "Room 4" || unescapeICalText(ev.Description) != "line one\nline two" {
		t.Errorf("text fields changed: %q %q %q", ev.Summary, ev.Location, ev.Description)
	}
	if ev.Transparency != "free" || ev.Visibility != "private" {
		t.Errorf("transparency/visibility = %q/%q, want free/private", ev.Transparency, ev.Visibility)
	}
	if m := primaryReminderMinutes(ev.ICSBlob); m == nil || *m != 25 {
		t.Errorf("reminder = %v, want 25", m)
	}
	if got := extractAltDescHTML(ev.ICSBlob); got != "<p>line <b>one</b></p>" {
		t.Errorf("rich body = %q", got)
	}
}

func TestMoveEventRejectsRecurring(t *testing.T) {
	api, calID := newLocalAPI(t)
	start := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC).Unix()
	id, err := api.CreateEvent(EventInput{
		CalendarID: calID, Summary: "Standup", TZName: "UTC",
		DTStartUnix: start, DTEndUnix: start + 1800,
		Recurrence: &RecurrenceSpec{Freq: "DAILY", Count: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := api.MoveEvent(id, start+3600, start+5400, ""); err == nil {
		t.Fatal("MoveEvent on a recurring event succeeded, want an error")
	}
}
