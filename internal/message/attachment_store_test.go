package message

import (
	"fmt"
	"testing"
	"time"
)

func TestAttachmentStore_ReplaceForMessages(t *testing.T) {
	newAtt := func(msgID, name string, n int) *Attachment {
		return &Attachment{ID: fmt.Sprintf("%s-%s-%d", msgID, name, n), MessageID: msgID, Filename: name}
	}
	tests := []struct {
		name      string
		refetch   []*Attachment // attachments of the second fetch of "m1"
		wantM1    []string
		wantOther int
	}{
		{
			name:      "re-fetch with the same attachment keeps one copy",
			refetch:   []*Attachment{newAtt("m1", "report.pdf", 2)},
			wantM1:    []string{"report.pdf"},
			wantOther: 1,
		},
		{
			name:      "re-fetch without attachments clears the old ones",
			refetch:   nil,
			wantM1:    nil,
			wantOther: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, accountID, folderID := newBodyFailedTestStore(t)
			for i, id := range []string{"m1", "m2"} {
				if err := s.Create(&Message{ID: id, AccountID: accountID, FolderID: folderID, UID: uint32(i + 1), Date: time.Now()}); err != nil {
					t.Fatalf("create message: %v", err)
				}
			}
			as := NewAttachmentStore(s.db)
			first := []*Attachment{newAtt("m1", "report.pdf", 1), newAtt("m2", "photo.jpg", 1)}
			if err := as.ReplaceForMessages([]string{"m1", "m2"}, first); err != nil {
				t.Fatalf("first fetch: %v", err)
			}
			if err := as.ReplaceForMessages([]string{"m1"}, tc.refetch); err != nil {
				t.Fatalf("re-fetch: %v", err)
			}

			got, err := as.GetByMessage("m1")
			if err != nil {
				t.Fatalf("GetByMessage m1: %v", err)
			}
			if len(got) != len(tc.wantM1) {
				t.Fatalf("m1 has %d attachments, want %d", len(got), len(tc.wantM1))
			}
			for i, a := range got {
				if a.Filename != tc.wantM1[i] {
					t.Errorf("m1 attachment %d = %q, want %q", i, a.Filename, tc.wantM1[i])
				}
			}
			other, err := as.GetByMessage("m2")
			if err != nil {
				t.Fatalf("GetByMessage m2: %v", err)
			}
			if len(other) != tc.wantOther {
				t.Errorf("m2 has %d attachments, want %d", len(other), tc.wantOther)
			}
		})
	}
}

// TestAttachmentStore_Embedded checks that the embedded flag survives both
// insert paths and both reads.
func TestAttachmentStore_Embedded(t *testing.T) {
	s, accountID, folderID := newBodyFailedTestStore(t)
	if err := s.Create(&Message{ID: "m1", AccountID: accountID, FolderID: folderID, UID: 1, Date: time.Now()}); err != nil {
		t.Fatalf("create message: %v", err)
	}
	as := NewAttachmentStore(s.db)
	batch := []*Attachment{
		{ID: "logo", MessageID: "m1", Filename: "logo.png", IsInline: true, ContentID: "logo@x", Embedded: true},
		{ID: "pdf", MessageID: "m1", Filename: "report.pdf"},
	}
	if err := as.ReplaceForMessages([]string{"m1"}, batch); err != nil {
		t.Fatalf("ReplaceForMessages: %v", err)
	}
	if err := as.Create(&Attachment{ID: "banner", MessageID: "m1", Filename: "banner.png", IsInline: true, ContentID: "banner@x", Embedded: true}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	want := map[string]bool{"logo": true, "pdf": false, "banner": true}
	got, err := as.GetByMessage("m1")
	if err != nil {
		t.Fatalf("GetByMessage: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d attachments, want %d", len(got), len(want))
	}
	for _, a := range got {
		if a.Embedded != want[a.ID] {
			t.Errorf("GetByMessage %s embedded = %v, want %v", a.ID, a.Embedded, want[a.ID])
		}
		one, err := as.Get(a.ID)
		if err != nil || one == nil {
			t.Fatalf("Get %s: %v", a.ID, err)
		}
		if one.Embedded != want[a.ID] {
			t.Errorf("Get %s embedded = %v, want %v", a.ID, one.Embedded, want[a.ID])
		}
	}
}
