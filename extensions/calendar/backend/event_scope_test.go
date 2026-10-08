package backend

import (
	"database/sql"
	"testing"
	"time"
)

// newLocalAPI returns an API over a fresh store with one writable local
// calendar.
func newLocalAPI(t *testing.T) (*API, string) {
	t.Helper()
	store := newTestStore(t)
	now := time.Now().Unix()
	err := store.WithTx(func(tx *sql.Tx) error {
		if err := store.CreateSourceTx(tx, Source{ID: "src-l", Type: SourceTypeLocal, Name: "Local", Enabled: true, Writable: true, CreatedAt: now}); err != nil {
			return err
		}
		return store.CreateCalendarTx(tx, Calendar{ID: "cal-l", SourceID: "src-l", DisplayName: "Local", Visible: true, CreatedAt: now})
	})
	if err != nil {
		t.Fatal(err)
	}
	return NewAPI(store, nil, nil, nil), "cal-l"
}

// TestRecurringScopeTargetsClickedOccurrence checks that this and
// this-and-future act on the occurrence the caller names, not the series
// start or the edited start.
func TestRecurringScopeTargetsClickedOccurrence(t *testing.T) {
	api, calID := newLocalAPI(t)
	start := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC).Unix()
	const day = int64(86400)
	id, err := api.CreateEvent(EventInput{
		CalendarID: calID, Summary: "Standup", TZName: "UTC",
		DTStartUnix: start, DTEndUnix: start + 1800,
		Recurrence: &RecurrenceSpec{Freq: "DAILY", Count: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	expand := func() map[int64]EventInstance {
		t.Helper()
		ev, err := api.store.GetEvent(id)
		if err != nil {
			t.Fatal(err)
		}
		ovs, err := api.store.ListOverrides(id)
		if err != nil {
			t.Fatal(err)
		}
		insts, err := ExpandInRange(*ev, ovs, time.Unix(start-day, 0), time.Unix(start+10*day, 0))
		if err != nil {
			t.Fatal(err)
		}
		out := make(map[int64]EventInstance, len(insts))
		for _, in := range insts {
			out[in.RecurrenceIDUnix] = in
		}
		return out
	}

	if err := api.DeleteEvent(id, EditScopeThis, start+2*day); err != nil {
		t.Fatal(err)
	}
	got := expand()
	if _, ok := got[start+2*day]; ok || len(got) != 4 {
		t.Fatalf("delete this: want day 3 gone and 4 left, got %d", len(got))
	}

	moved := start + 3*day + 3600
	err = api.UpdateEvent(EventUpdateInput{EventID: id, InstanceUnix: start + 3*day, EventInput: EventInput{
		CalendarID: calID, Summary: "Moved", TZName: "UTC", DTStartUnix: moved, DTEndUnix: moved + 1800,
	}}, EditScopeThis)
	if err != nil {
		t.Fatal(err)
	}
	got = expand()
	if inst := got[start+3*day]; inst.InstanceStartUnix != moved || inst.Summary != "Moved" || len(got) != 4 {
		t.Fatalf("update this: got %+v (%d instances)", inst, len(got))
	}

	if err := api.DeleteEvent(id, EditScopeThisAndFuture, start+day); err != nil {
		t.Fatal(err)
	}
	got = expand()
	if _, ok := got[start]; !ok || len(got) != 1 {
		t.Fatalf("delete this-and-future: want only the first occurrence, got %d", len(got))
	}

	if err := api.DeleteEvent(id, EditScopeThis, 0); err == nil {
		t.Fatal("delete this without an instance time should fail")
	}
}
