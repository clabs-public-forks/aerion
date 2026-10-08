package backend

import "testing"

func TestLoadTZ(t *testing.T) {
	tests := []struct {
		tzid, want string
	}{
		{"Europe/Berlin", "Europe/Berlin"},
		{`"Europe/Berlin"`, "Europe/Berlin"},
		{"W. Europe Standard Time", "Europe/Berlin"},
		{"Pacific Standard Time", "America/Los_Angeles"},
		{"/mozilla.org/20050126_1/America/New_York", "America/New_York"},
		{"/citadel.org/20190914_1/Asia/Tokyo", "Asia/Tokyo"},
		{"UTC", "UTC"},
	}
	for _, tt := range tests {
		l, err := loadTZ(tt.tzid)
		if err != nil {
			t.Errorf("loadTZ(%q): %v", tt.tzid, err)
			continue
		}
		if l.String() != tt.want {
			t.Errorf("loadTZ(%q) = %q, want %q", tt.tzid, l, tt.want)
		}
	}
	for _, bad := range []string{"", "Mars Standard Time", "/x/"} {
		if _, err := loadTZ(bad); err == nil {
			t.Errorf("loadTZ(%q) should fail", bad)
		}
	}
}

func TestParseCalendarObjectWindowsTZID(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//test//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:win-1\r\nDTSTAMP:20260101T000000Z\r\n" +
		"DTSTART;TZID=W. Europe Standard Time:20260115T100000\r\n" +
		"DTEND;TZID=W. Europe Standard Time:20260115T110000\r\n" +
		"SUMMARY:Outlook meeting\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	obj, err := ParseCalendarObject(ics)
	if err != nil {
		t.Fatal(err)
	}
	ev := obj.Master
	if ev.TZName != "Europe/Berlin" {
		t.Errorf("TZName = %q, want Europe/Berlin", ev.TZName)
	}
	// 10:00 CET is 09:00 UTC.
	if want := int64(1768467600); ev.DTStartUnix != want {
		t.Errorf("DTStartUnix = %d, want %d", ev.DTStartUnix, want)
	}
}

func TestParseCalendarObjectUnknownTZIDIsFloating(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//test//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:win-2\r\nDTSTAMP:20260101T000000Z\r\n" +
		"DTSTART;TZID=Nowhere Standard Time:20260115T100000\r\n" +
		"SUMMARY:Unknown zone\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	obj, err := ParseCalendarObject(ics)
	if err != nil {
		t.Fatalf("event with unknown TZID should still parse: %v", err)
	}
	if obj.Master.TZName != "" {
		t.Errorf("TZName = %q, want empty (floating)", obj.Master.TZName)
	}
}
