package backend

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-ical"
)

// TestAllDaySeriesWritesDateValues checks that UNTIL, EXDATE and
// RECURRENCE-ID written for an all-day series are DATEs in the display tz,
// as RFC 5545 requires when DTSTART is a DATE, and that the series still
// expands to the expected days.
func TestAllDaySeriesWritesDateValues(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skipf("tz data unavailable: %v", err)
	}
	SetConfiguredTimezone("Asia/Tokyo")
	t.Cleanup(func() { SetConfiguredTimezone("") })

	day := func(d int) int64 { return time.Date(2026, 10, d, 0, 0, 0, 0, tokyo).Unix() }
	in := EventInput{
		Summary:     "Daily",
		IsAllDay:    true,
		DTStartUnix: day(5),
		DTEndUnix:   day(6),
		Recurrence:  &RecurrenceSpec{Freq: "DAILY", UntilUnix: day(20)},
	}
	rrule := rruleText(in.Recurrence, true)
	if !strings.HasSuffix(rrule, "UNTIL=20261020") {
		t.Errorf("rruleText = %q, want a DATE UNTIL", rrule)
	}

	blob, err := serializeVEVENT("uid@a", in)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	blob, err = addEXDATE(blob, day(7), true)
	if err != nil {
		t.Fatalf("addEXDATE: %v", err)
	}
	clamped := clampRRuleUntil(rrule, day(9)-1, true)
	if !strings.HasSuffix(clamped, "UNTIL=20261008") {
		t.Errorf("clampRRuleUntil = %q, want UNTIL=20261008", clamped)
	}
	blob, err = reserializeMasterICS(Event{ICSBlob: blob}, clamped)
	if err != nil {
		t.Fatalf("reserialize: %v", err)
	}
	if !strings.Contains(blob, "EXDATE;VALUE=DATE:20261007") {
		t.Errorf("blob missing DATE EXDATE:\n%s", blob)
	}

	master := Event{ICSBlob: blob, RRuleText: clamped, IsAllDay: true, DTStartUnix: day(5), DTEndUnix: day(6)}
	insts, err := ExpandInRange(master, nil, time.Unix(day(1), 0), time.Unix(day(30), 0))
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	var got []int64
	for _, i := range insts {
		got = append(got, i.InstanceStartUnix)
	}
	want := []int64{day(5), day(6), day(8)}
	if len(got) != len(want) {
		t.Fatalf("instances = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("instance %d = %v, want %v", i, time.Unix(got[i], 0).In(tokyo), time.Unix(want[i], 0).In(tokyo))
		}
	}

	ovBlob, err := serializeVEVENTWithRecurrenceID("uid@a", in, day(6), true)
	if err != nil {
		t.Fatalf("override: %v", err)
	}
	cal, err := decodeICS(ovBlob)
	if err != nil {
		t.Fatalf("decode override: %v", err)
	}
	rid := cal.Events()[0].Props.Get(ical.PropRecurrenceID)
	if rid.Value != "20261006" || rid.Params.Get(ical.ParamValue) != "DATE" {
		t.Errorf("RECURRENCE-ID = %q %v, want DATE 20261006", rid.Value, rid.Params)
	}
	ov, err := buildOverride(&cal.Events()[0])
	if err != nil {
		t.Fatalf("buildOverride: %v", err)
	}
	if ov.RecurrenceIDUnix != day(6) {
		t.Errorf("RecurrenceIDUnix = %v, want %v", time.Unix(ov.RecurrenceIDUnix, 0).In(tokyo), time.Unix(day(6), 0).In(tokyo))
	}
}
