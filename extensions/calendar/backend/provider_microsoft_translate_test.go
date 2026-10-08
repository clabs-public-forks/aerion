package backend

import (
	"strings"
	"testing"
	"time"
)

// graphRecurrenceToRRule must cover all 6 Microsoft Graph pattern types and the
// range. The previous converter dropped relativeMonthly/relativeYearly (index →
// BYSETPOS), yearly month → BYMONTH, and WKST — so recurring series were stored
// as dead non-recurring stubs at their origin date (#278).
func TestGraphRecurrenceToRRule(t *testing.T) {
	tests := []struct {
		name string
		rec  *graphRecurrence
		want string
	}{
		{"nil", nil, ""},
		{"unknown type", &graphRecurrence{Pattern: graphPattern{Type: "weird", Interval: 1}}, ""},
		{
			"daily",
			&graphRecurrence{Pattern: graphPattern{Type: "daily", Interval: 1}, Range: graphRange{Type: "noEnd"}},
			"FREQ=DAILY",
		},
		{
			"daily interval+count",
			&graphRecurrence{Pattern: graphPattern{Type: "daily", Interval: 3}, Range: graphRange{Type: "numbered", NumberOfOccurrences: 5}},
			"FREQ=DAILY;INTERVAL=3;COUNT=5",
		},
		{
			"weekly with WKST + until",
			&graphRecurrence{
				Pattern: graphPattern{Type: "weekly", Interval: 2, DaysOfWeek: []string{"monday", "wednesday"}, FirstDayOfWeek: "sunday"},
				Range:   graphRange{Type: "endDate", EndDate: "2026-12-31"},
			},
			"FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE;WKST=SU;UNTIL=20261231T235959Z",
		},
		{
			"absoluteMonthly",
			&graphRecurrence{Pattern: graphPattern{Type: "absoluteMonthly", Interval: 3, DayOfMonth: 15}, Range: graphRange{Type: "noEnd"}},
			"FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=15",
		},
		{
			"relativeMonthly second Thursday",
			&graphRecurrence{Pattern: graphPattern{Type: "relativeMonthly", Interval: 1, DaysOfWeek: []string{"thursday"}, Index: "second"}, Range: graphRange{Type: "noEnd"}},
			"FREQ=MONTHLY;BYDAY=TH;BYSETPOS=2",
		},
		{
			"relativeMonthly last Friday",
			&graphRecurrence{Pattern: graphPattern{Type: "relativeMonthly", Interval: 1, DaysOfWeek: []string{"friday"}, Index: "last"}, Range: graphRange{Type: "noEnd"}},
			"FREQ=MONTHLY;BYDAY=FR;BYSETPOS=-1",
		},
		{
			"relativeMonthly default index (first)",
			&graphRecurrence{Pattern: graphPattern{Type: "relativeMonthly", Interval: 1, DaysOfWeek: []string{"monday"}}, Range: graphRange{Type: "noEnd"}},
			"FREQ=MONTHLY;BYDAY=MO;BYSETPOS=1",
		},
		{
			"absoluteYearly March 15",
			&graphRecurrence{Pattern: graphPattern{Type: "absoluteYearly", Interval: 1, Month: 3, DayOfMonth: 15}, Range: graphRange{Type: "noEnd"}},
			"FREQ=YEARLY;BYMONTH=3;BYMONTHDAY=15",
		},
		{
			"relativeYearly last Wednesday of November",
			&graphRecurrence{Pattern: graphPattern{Type: "relativeYearly", Interval: 1, Month: 11, DaysOfWeek: []string{"wednesday"}, Index: "last"}, Range: graphRange{Type: "noEnd"}},
			"FREQ=YEARLY;BYMONTH=11;BYDAY=WE;BYSETPOS=-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graphRecurrenceToRRule(tt.rec)
			if got != tt.want {
				t.Errorf("graphRecurrenceToRRule()\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

// Regression (#278): a Graph event whose subject/body/location contains CRLF
// (Outlook HTML bodies do) must still translate. go-ical's encoder rejects raw
// CR/LF; icsText normalizes them so the event isn't dropped.
func TestMicrosoftTranslate_CRLFContentEncodes(t *testing.T) {
	src := graphEvent{
		ICalUID:  "crlf@example.com",
		Subject:  "Subject with\r\na newline",
		Body:     &graphBody{Content: "<html>\r\nline1\r\nline2\r\n</html>"},
		Location: &graphLocation{DisplayName: "Room\r\n123"},
		Start:    &graphTimePoint{DateTime: "2026-01-01T09:00:00.0000000", TimeZone: "UTC"},
		End:      &graphTimePoint{DateTime: "2026-01-01T10:00:00.0000000", TimeZone: "UTC"},
	}
	blob, err := translateGraphEventToICS(src)
	if err != nil {
		t.Fatalf("translate failed on CRLF content: %v", err)
	}
	if _, perr := ParseCalendarObject(blob); perr != nil {
		t.Fatalf("translated blob is not parseable: %v", perr)
	}
}

// Graph showAs ⇄ iCal TRANSP (2-state). showAs "free" → TRANSPARENT and back to
// showAs "free"; anything else → busy.
func TestMicrosoftTranslate_ShowAs(t *testing.T) {
	mk := func(showAs string) graphEvent {
		return graphEvent{
			ICalUID: "u", Subject: "s", ShowAs: showAs,
			Start: &graphTimePoint{DateTime: "2026-01-01T09:00:00.0000000", TimeZone: "UTC"},
			End:   &graphTimePoint{DateTime: "2026-01-01T10:00:00.0000000", TimeZone: "UTC"},
		}
	}

	freeBlob, err := translateGraphEventToICS(mk("free"))
	if err != nil {
		t.Fatalf("translate free: %v", err)
	}
	if !strings.Contains(freeBlob, "TRANSP:TRANSPARENT") {
		t.Errorf("showAs=free should map to TRANSP:TRANSPARENT:\n%s", freeBlob)
	}
	if g, _ := translateICSToGraphEvent(freeBlob); g.ShowAs != "free" {
		t.Errorf("free ICS → showAs %q, want free", g.ShowAs)
	}

	busyBlob, err := translateGraphEventToICS(mk("busy"))
	if err != nil {
		t.Fatalf("translate busy: %v", err)
	}
	if strings.Contains(busyBlob, "TRANSP:TRANSPARENT") {
		t.Errorf("showAs=busy should not be TRANSPARENT:\n%s", busyBlob)
	}
	if g, _ := translateICSToGraphEvent(busyBlob); g.ShowAs != "busy" {
		t.Errorf("busy ICS → showAs %q, want busy", g.ShowAs)
	}
}

// Graph HTML body ⇄ iCal X-ALT-DESC. An HTML body lands in X-ALT-DESC with the
// plaintext bodyPreview in DESCRIPTION; writing back sends body.contentType=html
// carrying the X-ALT-DESC markup. A text body uses DESCRIPTION only.
func TestMicrosoftTranslate_HTMLBody(t *testing.T) {
	htmlBody := "<p>Bring <b>laptop</b></p>"
	src := graphEvent{
		ICalUID:     "html@example.com",
		Subject:     "s",
		Body:        &graphBody{ContentType: "html", Content: htmlBody},
		BodyPreview: "Bring laptop",
		Start:       &graphTimePoint{DateTime: "2026-01-01T09:00:00.0000000", TimeZone: "UTC"},
		End:         &graphTimePoint{DateTime: "2026-01-01T10:00:00.0000000", TimeZone: "UTC"},
	}
	blob, err := translateGraphEventToICS(src)
	if err != nil {
		t.Fatalf("translate html body: %v", err)
	}
	if !strings.Contains(blob, "X-ALT-DESC") {
		t.Errorf("html body should produce X-ALT-DESC:\n%s", blob)
	}
	if !strings.Contains(blob, "DESCRIPTION:Bring laptop") {
		t.Errorf("html body should put bodyPreview in DESCRIPTION:\n%s", blob)
	}
	if got := extractAltDescHTML(blob); got != htmlBody {
		t.Errorf("extractAltDescHTML = %q, want %q", got, htmlBody)
	}

	// Write back: X-ALT-DESC → body.contentType=html.
	g, err := translateICSToGraphEvent(blob)
	if err != nil {
		t.Fatalf("translate back: %v", err)
	}
	if g.Body == nil || g.Body.ContentType != "html" {
		t.Fatalf("expected html body on write-back, got %+v", g.Body)
	}
	if g.Body.Content != htmlBody {
		t.Errorf("write-back body = %q, want %q", g.Body.Content, htmlBody)
	}

	// A plaintext body uses DESCRIPTION only (no X-ALT-DESC), write-back text.
	textSrc := graphEvent{
		ICalUID: "text@example.com", Subject: "s",
		Body:  &graphBody{ContentType: "text", Content: "just text"},
		Start: &graphTimePoint{DateTime: "2026-01-01T09:00:00.0000000", TimeZone: "UTC"},
		End:   &graphTimePoint{DateTime: "2026-01-01T10:00:00.0000000", TimeZone: "UTC"},
	}
	textBlob, err := translateGraphEventToICS(textSrc)
	if err != nil {
		t.Fatalf("translate text body: %v", err)
	}
	if strings.Contains(textBlob, "X-ALT-DESC") {
		t.Errorf("text body should omit X-ALT-DESC:\n%s", textBlob)
	}
	if tg, _ := translateICSToGraphEvent(textBlob); tg.Body == nil || tg.Body.ContentType != "text" {
		t.Errorf("text body write-back should stay text, got %+v", tg.Body)
	}
}

// Graph sensitivity ⇄ iCal CLASS (3-state).
func TestMicrosoftTranslate_Sensitivity(t *testing.T) {
	mk := func(sens string) graphEvent {
		return graphEvent{
			ICalUID: "u", Subject: "s", Sensitivity: sens,
			Start: &graphTimePoint{DateTime: "2026-01-01T09:00:00.0000000", TimeZone: "UTC"},
			End:   &graphTimePoint{DateTime: "2026-01-01T10:00:00.0000000", TimeZone: "UTC"},
		}
	}
	cases := []struct{ sens, wantClass, wantSens string }{
		{"normal", "", "normal"},
		{"private", "CLASS:PRIVATE", "private"},
		{"confidential", "CLASS:CONFIDENTIAL", "confidential"},
	}
	for _, c := range cases {
		blob, err := translateGraphEventToICS(mk(c.sens))
		if err != nil {
			t.Fatalf("translate %s: %v", c.sens, err)
		}
		if c.wantClass == "" && strings.Contains(blob, "CLASS:") {
			t.Errorf("%s should omit CLASS:\n%s", c.sens, blob)
		}
		if c.wantClass != "" && !strings.Contains(blob, c.wantClass) {
			t.Errorf("%s → %s missing:\n%s", c.sens, c.wantClass, blob)
		}
		if g, _ := translateICSToGraphEvent(blob); g.Sensitivity != c.wantSens {
			t.Errorf("%s round-trip sensitivity = %q, want %q", c.sens, g.Sensitivity, c.wantSens)
		}
	}
}

// A Graph series recurs on its own zone's wall clock. Times arrive in UTC
// (Prefer header), so the stored master must be re-anchored or BYDAY is
// matched against the UTC date and DST shifts the local time.
func TestMicrosoftTranslate_SeriesExpandsInItsZone(t *testing.T) {
	tests := []struct {
		name       string
		start, end string // UTC, as Graph returns them
		recTZ      string
		origTZ     string
		iana       string
		localHour  int
		until      time.Time
		wantCount  int
	}{
		{
			// Mon 09:00 Sydney = Sun 23:00Z; DST starts Oct 4.
			name:  "sydney across DST start, via originalStartTimeZone",
			start: "2026-09-06T23:00:00.0000000", end: "2026-09-07T00:00:00.0000000",
			origTZ: "AUS Eastern Standard Time", iana: "Australia/Sydney",
			localHour: 9, until: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), wantCount: 8,
		},
		{
			// Mon 21:00 New York = Tue 01:00Z; DST ends Nov 1.
			name:  "new york across DST end, via recurrenceTimeZone",
			start: "2026-10-06T01:00:00.0000000", end: "2026-10-06T02:00:00.0000000",
			recTZ: "Eastern Standard Time", origTZ: "tzone://Microsoft/Custom", iana: "America/New_York",
			localHour: 21, until: time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC), wantCount: 7,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ge := graphEvent{
				ICalUID: "series-1",
				Subject: "Weekly",
				Start:   &graphTimePoint{DateTime: tc.start, TimeZone: "UTC"},
				End:     &graphTimePoint{DateTime: tc.end, TimeZone: "UTC"},
				Recurrence: &graphRecurrence{
					Pattern: graphPattern{Type: "weekly", Interval: 1, DaysOfWeek: []string{"monday"}},
					Range:   graphRange{Type: "noEnd", StartDate: "2026-09-07", RecurrenceTimeZone: tc.recTZ},
				},
				OriginalStartTimeZone: tc.origTZ,
			}
			blob, err := translateGraphEventToICS(ge)
			if err != nil {
				t.Fatalf("translate: %v", err)
			}
			ev := Event{ID: "e1", UID: ge.ICalUID, ICSBlob: blob, RRuleText: graphRecurrenceToRRule(ge.Recurrence)}
			fillDenormalizedFieldsFromICS(&ev, blob)

			insts, err := ExpandInRange(ev, nil, time.Unix(ev.DTStartUnix, 0), tc.until)
			if err != nil {
				t.Fatalf("expand: %v", err)
			}
			if len(insts) != tc.wantCount {
				t.Fatalf("got %d instances, want %d", len(insts), tc.wantCount)
			}
			loc, _ := time.LoadLocation(tc.iana)
			for _, inst := range insts {
				local := time.Unix(inst.InstanceStartUnix, 0).In(loc)
				if local.Weekday() != time.Monday || local.Hour() != tc.localHour || local.Minute() != 0 {
					t.Errorf("instance at %s, want Monday %02d:00", local.Format(time.RFC1123), tc.localHour)
				}
			}
		})
	}
}

func TestMicrosoftTranslate_SeriesSentInItsZone(t *testing.T) {
	tests := []struct {
		name        string
		dtstart     string
		dtend       string
		wantStart   string
		wantZone    string
		wantRecZone string
		wantDayOfWk string
	}{
		{
			name:      "sydney monday morning stays local",
			dtstart:   "DTSTART;TZID=Australia/Sydney:20260907T090000",
			dtend:     "DTEND;TZID=Australia/Sydney:20260907T100000",
			wantStart: "2026-09-07T09:00:00.0000000", wantZone: "AUS Eastern Standard Time",
			wantRecZone: "AUS Eastern Standard Time", wantDayOfWk: "monday",
		},
		{
			name:      "utc series is unchanged",
			dtstart:   "DTSTART:20260907T090000Z",
			dtend:     "DTEND:20260907T100000Z",
			wantStart: "2026-09-07T09:00:00.0000000", wantZone: "UTC",
			wantRecZone: "", wantDayOfWk: "monday",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			blob := wrapICS("BEGIN:VEVENT\nUID:s1\nSUMMARY:Weekly\n" + tc.dtstart + "\n" + tc.dtend +
				"\nRRULE:FREQ=WEEKLY;BYDAY=MO\nEND:VEVENT")
			ge, err := translateICSToGraphEvent(blob)
			if err != nil {
				t.Fatalf("translate: %v", err)
			}
			if ge.Start.DateTime != tc.wantStart || ge.Start.TimeZone != tc.wantZone {
				t.Errorf("start = %s %s, want %s %s", ge.Start.DateTime, ge.Start.TimeZone, tc.wantStart, tc.wantZone)
			}
			if ge.End.TimeZone != tc.wantZone {
				t.Errorf("end zone = %s, want %s", ge.End.TimeZone, tc.wantZone)
			}
			if ge.Recurrence.Range.RecurrenceTimeZone != tc.wantRecZone {
				t.Errorf("recurrenceTimeZone = %q, want %q", ge.Recurrence.Range.RecurrenceTimeZone, tc.wantRecZone)
			}
			if len(ge.Recurrence.Pattern.DaysOfWeek) != 1 || ge.Recurrence.Pattern.DaysOfWeek[0] != tc.wantDayOfWk {
				t.Errorf("daysOfWeek = %v, want [%s]", ge.Recurrence.Pattern.DaysOfWeek, tc.wantDayOfWk)
			}
		})
	}
}
