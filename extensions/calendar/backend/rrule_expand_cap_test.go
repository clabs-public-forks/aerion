package backend

import (
	"testing"
	"time"
)

// TestExpandInRangeCapsRunawayRules checks that rules generating huge
// numbers of occurrences are cut off at the expansion limits.
func TestExpandInRangeCapsRunawayRules(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	tests := []struct {
		name  string
		start time.Time
		rule  string
		max   int
	}{
		{"secondly in window", from, "FREQ=SECONDLY", maxExpandInstances},
		{"minutely from years before", from.AddDate(-5, 0, 0), "FREQ=MINUTELY", 0},
		{"daily stays whole", from, "FREQ=DAILY", 32},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := EventInput{
				Summary:     "Runaway",
				DTStartUnix: tt.start.Unix(),
				DTEndUnix:   tt.start.Unix() + 1,
			}
			blob, err := serializeVEVENT("uid@a", in)
			if err != nil {
				t.Fatalf("serialize: %v", err)
			}
			blob, err = reserializeMasterICS(Event{ICSBlob: blob}, tt.rule)
			if err != nil {
				t.Fatalf("reserialize: %v", err)
			}
			ev := Event{ICSBlob: blob, RRuleText: tt.rule, DTStartUnix: in.DTStartUnix, DTEndUnix: in.DTEndUnix}
			began := time.Now()
			insts, err := ExpandInRange(ev, nil, from, to)
			if err != nil {
				t.Fatalf("expand: %v", err)
			}
			if len(insts) > tt.max {
				t.Errorf("got %d instances, want at most %d", len(insts), tt.max)
			}
			if d := time.Since(began); d > 5*time.Second {
				t.Errorf("expansion took %v", d)
			}
		})
	}
}
