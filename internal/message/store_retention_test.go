package message

import (
	"testing"
	"time"
)

func TestStore_DeleteOlderThan(t *testing.T) {
	now := time.Now().UTC()
	old := now.AddDate(0, 0, -60)
	cutoff := time.Now().AddDate(0, 0, -30) // local time, as the sync engine passes it
	tests := []struct {
		name       string
		date       time.Time
		receivedAt time.Time
		wantKept   bool
	}{
		{name: "old date and old arrival", date: old, receivedAt: old, wantKept: false},
		{name: "recent date", date: now, receivedAt: old, wantKept: true},
		{name: "old date header but recent arrival", date: old, receivedAt: now, wantKept: true},
		{name: "no date header, recent arrival", date: time.Time{}, receivedAt: now, wantKept: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, accountID, folderID := newBodyFailedTestStore(t)
			if err := s.Create(&Message{ID: "m1", AccountID: accountID, FolderID: folderID, UID: 1, Date: tc.date, ReceivedAt: tc.receivedAt}); err != nil {
				t.Fatalf("create message: %v", err)
			}
			if _, err := s.DeleteOlderThan(accountID, cutoff); err != nil {
				t.Fatalf("DeleteOlderThan: %v", err)
			}
			n, err := s.CountByFolder(folderID)
			if err != nil {
				t.Fatalf("count: %v", err)
			}
			if kept := n == 1; kept != tc.wantKept {
				t.Fatalf("message kept = %v, want %v", kept, tc.wantKept)
			}
		})
	}
}
