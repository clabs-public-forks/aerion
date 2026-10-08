package sync

import "testing"

func TestSkipEmptyRemoteReconcile(t *testing.T) {
	tests := []struct {
		name             string
		remote, local    int
		syncPeriodDays   int
		selectedMessages uint32
		want             bool
	}{
		{"confirmed empty mailbox reconciles", 0, 5, 0, 0, false},
		{"empty search but SELECT reports messages skips", 0, 5, 0, 3, true},
		{"date-filtered empty window reconciles", 0, 5, 30, 3, false},
		{"no local messages reconciles", 0, 0, 0, 3, false},
		{"non-empty search reconciles", 2, 5, 0, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipEmptyRemoteReconcile(tt.remote, tt.local, tt.syncPeriodDays, tt.selectedMessages)
			if got != tt.want {
				t.Errorf("skipEmptyRemoteReconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
