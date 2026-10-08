package backend

import (
	"strings"
	"testing"
	"time"
)

func TestMergeRRule(t *testing.T) {
	tests := []struct {
		name, old string
		spec      *RecurrenceSpec
		want      string
	}{
		{"no recurrence", "FREQ=WEEKLY;BYDAY=MO", nil, ""},
		{"new rule", "", &RecurrenceSpec{Freq: "DAILY", Count: 3}, "FREQ=DAILY;COUNT=3"},
		{"keep", "RRULE:FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE;UNTIL=20261231T225900Z",
			&RecurrenceSpec{Freq: "WEEKLY", Keep: true}, "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE;UNTIL=20261231T225900Z"},
		{"same freq keeps BY parts", "FREQ=WEEKLY;BYDAY=MO,WE;COUNT=4",
			&RecurrenceSpec{Freq: "WEEKLY", Count: 8}, "FREQ=WEEKLY;BYDAY=MO,WE;COUNT=8"},
		{"open-ended", "FREQ=MONTHLY;BYMONTHDAY=15;COUNT=4",
			&RecurrenceSpec{Freq: "MONTHLY"}, "FREQ=MONTHLY;BYMONTHDAY=15"},
		{"freq change drops BY parts", "FREQ=WEEKLY;BYDAY=MO,WE",
			&RecurrenceSpec{Freq: "DAILY", Keep: true}, "FREQ=DAILY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeRRule(tt.old, tt.spec); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

const mergeTestICS = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Other//EN\r\n" +
	"BEGIN:VEVENT\r\nUID:m1\r\nDTSTAMP:20260101T000000Z\r\nSUMMARY:Old\r\n" +
	"DTSTART:20260302T100000Z\r\nDTEND:20260302T103000Z\r\n" +
	"RRULE:FREQ=WEEKLY;BYDAY=MO,WE\r\nEXDATE:20260304T100000Z\r\n" +
	"CATEGORIES:Work\r\nX-VENDOR-FLAG:1\r\n" +
	"BEGIN:VALARM\r\nACTION:DISPLAY\r\nTRIGGER:-PT15M\r\nDESCRIPTION:a\r\nEND:VALARM\r\n" +
	"BEGIN:VALARM\r\nACTION:AUDIO\r\nTRIGGER:-PT1H\r\nEND:VALARM\r\n" +
	"END:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:m1\r\nDTSTAMP:20260101T000000Z\r\nSUMMARY:Moved\r\n" +
	"RECURRENCE-ID:20260309T100000Z\r\nDTSTART:20260309T120000Z\r\nDTEND:20260309T123000Z\r\n" +
	"END:VEVENT\r\nEND:VCALENDAR\r\n"

func TestMergeVEVENT(t *testing.T) {
	start := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC).Unix()
	master := Event{UID: "m1", DTStartUnix: start, DTEndUnix: start + 1800,
		RRuleText: "FREQ=WEEKLY;BYDAY=MO,WE", ICSBlob: mergeTestICS}
	base := EventInput{Summary: "New", DTStartUnix: start, DTEndUnix: start + 3600,
		Recurrence: &RecurrenceSpec{Freq: "WEEKLY", Keep: true}, Reminder: &ReminderSpec{OffsetMinutes: 15}}

	tests := []struct {
		name           string
		edit           func(*EventInput)
		keepExceptions bool
		has, lacks     []string
	}{
		{"title edit keeps everything", func(*EventInput) {}, true,
			[]string{"SUMMARY:New", "BYDAY=MO,WE", "EXDATE:20260304T100000Z", "RECURRENCE-ID",
				"CATEGORIES:Work", "X-VENDOR-FLAG:1", "ACTION:AUDIO", "DTEND:20260302T110000Z"},
			[]string{"SUMMARY:Old"}},
		{"reminder change replaces alarms", func(in *EventInput) { in.Reminder = &ReminderSpec{OffsetMinutes: 30} }, true,
			[]string{"TRIGGER:-PT30M", "EXDATE"},
			[]string{"ACTION:AUDIO", "TRIGGER:-PT15M"}},
		{"reminder removed", func(in *EventInput) { in.Reminder = nil }, true,
			nil, []string{"VALARM"}},
		{"start move drops exceptions", func(in *EventInput) { in.DTStartUnix += 3600; in.DTEndUnix += 3600 }, false,
			[]string{"X-VENDOR-FLAG:1", "BYDAY=MO,WE"},
			[]string{"EXDATE", "RECURRENCE-ID", "SUMMARY:Moved"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			tt.edit(&in)
			in.Recurrence = &RecurrenceSpec{rule: mergeRRule(master.RRuleText, in.Recurrence)}
			blob, keep, err := mergeVEVENT(master, in)
			if err != nil {
				t.Fatal(err)
			}
			if keep != tt.keepExceptions {
				t.Errorf("keepExceptions = %v, want %v", keep, tt.keepExceptions)
			}
			for _, s := range tt.has {
				if !strings.Contains(blob, s) {
					t.Errorf("blob lacks %q:\n%s", s, blob)
				}
			}
			for _, s := range tt.lacks {
				if strings.Contains(blob, s) {
					t.Errorf("blob still has %q:\n%s", s, blob)
				}
			}
		})
	}
}

func TestPrimaryReminderMinutes(t *testing.T) {
	got := primaryReminderMinutes(mergeTestICS)
	if got == nil || *got != 15 {
		t.Fatalf("got %v, want 15", got)
	}
	if got := primaryReminderMinutes(strings.Replace(mergeTestICS, "VALARM", "X-NONE", -1)); got != nil {
		t.Fatalf("no alarm: got %d", *got)
	}
}

// TestUpdateAllKeepsOverrides checks that a whole-series edit that leaves
// the timing alone keeps the overrides of a local event.
func TestUpdateAllKeepsOverrides(t *testing.T) {
	api, calID := newLocalAPI(t)
	start := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC).Unix()
	in := EventInput{CalendarID: calID, Summary: "Standup", TZName: "UTC",
		DTStartUnix: start, DTEndUnix: start + 1800,
		Recurrence: &RecurrenceSpec{Freq: "DAILY", Count: 5}}
	id, err := api.CreateEvent(in)
	if err != nil {
		t.Fatal(err)
	}
	moved := in
	moved.Recurrence = nil
	moved.DTStartUnix, moved.DTEndUnix = start+86400+3600, start+86400+5400
	if err := api.UpdateEvent(EventUpdateInput{EventID: id, InstanceUnix: start + 86400, EventInput: moved}, EditScopeThis); err != nil {
		t.Fatal(err)
	}

	renamed := in
	renamed.Summary = "Daily"
	renamed.TZName = ""
	renamed.Recurrence = &RecurrenceSpec{Freq: "DAILY", Count: 5, Keep: true}
	if err := api.UpdateEvent(EventUpdateInput{EventID: id, EventInput: renamed}, EditScopeAll); err != nil {
		t.Fatal(err)
	}
	ovs, err := api.store.ListOverrides(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(ovs) != 1 {
		t.Fatalf("overrides after rename = %d, want 1", len(ovs))
	}

	renamed.DTStartUnix += 600
	renamed.DTEndUnix += 600
	if err := api.UpdateEvent(EventUpdateInput{EventID: id, EventInput: renamed}, EditScopeAll); err != nil {
		t.Fatal(err)
	}
	if ovs, _ = api.store.ListOverrides(id); len(ovs) != 0 {
		t.Fatalf("overrides after start move = %d, want 0", len(ovs))
	}
}

func TestSetAttendeePartStat(t *testing.T) {
	blob := strings.Replace(mergeTestICS, "CATEGORIES:Work\r\n",
		"ATTENDEE;PARTSTAT=NEEDS-ACTION;RSVP=TRUE:mailto:Me@example.com\r\n"+
			"ATTENDEE;PARTSTAT=ACCEPTED:mailto:other@example.com\r\n", 1)
	out, ok := setAttendeePartStat(blob, map[string]struct{}{"me@example.com": {}}, PartStatAccepted)
	if !ok {
		t.Fatal("no attendee updated")
	}
	for _, s := range []string{"RRULE:FREQ=WEEKLY;BYDAY=MO,WE", "EXDATE", "RECURRENCE-ID", "VALARM",
		"ATTENDEE;PARTSTAT=ACCEPTED:mailto:Me@example.com"} {
		if !strings.Contains(out, s) {
			t.Errorf("result lacks %q:\n%s", s, out)
		}
	}
	if strings.Contains(out, "RSVP=TRUE") {
		t.Errorf("RSVP request kept:\n%s", out)
	}
	if _, ok := setAttendeePartStat(mergeTestICS, map[string]struct{}{"me@example.com": {}}, PartStatAccepted); ok {
		t.Error("blob without self attendee reported updated")
	}
}
