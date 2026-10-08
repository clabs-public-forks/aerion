package backend

import (
	"testing"
	"time"
)

// TestExpandInRangeWindowEdges covers occurrences that start before the
// window but run into it, and overrides that move an occurrence into or
// out of the window.
func TestExpandInRangeWindowEdges(t *testing.T) {
	day := func(d, h int) int64 { return time.Date(2026, 10, d, h, 0, 0, 0, time.UTC).Unix() }
	override := func(rid, start, end int64) EventOverride {
		blob, err := serializeVEVENTWithRecurrenceID("uid@a", EventInput{Summary: "Moved", DTStartUnix: start, DTEndUnix: end}, rid, false)
		if err != nil {
			t.Fatalf("override: %v", err)
		}
		cal, err := decodeICS(blob)
		if err != nil {
			t.Fatalf("decode override: %v", err)
		}
		ov, err := buildOverride(&cal.Events()[0])
		if err != nil {
			t.Fatalf("buildOverride: %v", err)
		}
		return ov
	}

	// Weekly, Thursday 09:00 for three days, starting Oct 1.
	in := EventInput{Summary: "Trip", DTStartUnix: day(1, 9), DTEndUnix: day(4, 9)}
	blob, err := serializeVEVENT("uid@a", in)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	blob, err = reserializeMasterICS(Event{ICSBlob: blob}, "FREQ=WEEKLY")
	if err != nil {
		t.Fatalf("reserialize: %v", err)
	}
	master := Event{UID: "uid@a", ICSBlob: blob, RRuleText: "FREQ=WEEKLY", DTStartUnix: in.DTStartUnix, DTEndUnix: in.DTEndUnix}

	tests := []struct {
		name      string
		overrides []EventOverride
		from, to  int64
		want      []int64 // instance starts
	}{
		{"running into window", nil, day(10, 0), day(11, 0), []int64{day(8, 9)}},
		{"moved into window", []EventOverride{override(day(22, 9), day(10, 12), day(10, 13))}, day(10, 0), day(11, 0), []int64{day(8, 9), day(10, 12)}},
		{"moved out of window", []EventOverride{override(day(15, 9), day(25, 12), day(25, 13))}, day(15, 0), day(16, 0), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insts, err := ExpandInRange(master, tt.overrides, time.Unix(tt.from, 0), time.Unix(tt.to, 0))
			if err != nil {
				t.Fatalf("expand: %v", err)
			}
			var got []int64
			for _, i := range insts {
				got = append(got, i.InstanceStartUnix)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("starts = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("start %d = %v, want %v", i, time.Unix(got[i], 0).UTC(), time.Unix(tt.want[i], 0).UTC())
				}
			}
		})
	}
}
