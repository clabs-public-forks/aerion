package app

import (
	"testing"

	"github.com/hkdb/aerion/internal/folder"
	"github.com/hkdb/aerion/internal/message"
)

func TestSnapshotFolderDetectsChanges(t *testing.T) {
	tests := []struct {
		name    string
		change  func(t *testing.T, a *App, sent *folder.Folder)
		changed bool
	}{
		{"no-op sync", func(*testing.T, *App, *folder.Folder) {}, false},
		{"new message", func(t *testing.T, a *App, sent *folder.Folder) {
			if err := a.messageStore.Create(&message.Message{
				ID: "new", AccountID: sent.AccountID, FolderID: sent.ID, UID: 99,
			}); err != nil {
				t.Fatal(err)
			}
		}, true},
		{"deletion", func(t *testing.T, a *App, _ *folder.Folder) {
			if err := a.messageStore.Delete("sent"); err != nil {
				t.Fatal(err)
			}
		}, true},
		{"uidvalidity change", func(t *testing.T, a *App, sent *folder.Folder) {
			sent.UIDValidity = 42
			if err := a.folderStore.Update(sent); err != nil {
				t.Fatal(err)
			}
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, folders := newArchiveTestApp(t, folder.TypeSent)
			sent := folders[folder.TypeSent]
			before, err := a.snapshotFolder(sent.ID)
			if err != nil {
				t.Fatal(err)
			}
			tt.change(t, a, sent)
			after, err := a.snapshotFolder(sent.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got := after != before; got != tt.changed {
				t.Errorf("changed = %v, want %v (before %+v, after %+v)", got, tt.changed, before, after)
			}
		})
	}
}
