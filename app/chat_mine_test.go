package app

import (
	"testing"

	"github.com/hkdb/aerion/internal/message"
)

func TestChatOwnerMine(t *testing.T) {
	owner := chatOwner{
		emails:        map[string]struct{}{"me@example.com": {}, "work@corp.example": {}},
		sentFolderIDs: map[string]struct{}{"sent-1": {}},
	}
	tests := []struct {
		name string
		msg  *message.Message
		want bool
	}{
		{"account address", &message.Message{FolderID: "inbox", FromEmail: "me@example.com"}, true},
		{"address case and spaces", &message.Message{FolderID: "inbox", FromEmail: " Work@Corp.Example "}, true},
		{"alias in sent folder", &message.Message{FolderID: "sent-1", FromEmail: "alias@example.com"}, true},
		{"someone else in inbox", &message.Message{FolderID: "inbox", FromEmail: "bob@example.com"}, false},
		{"empty sender", &message.Message{FolderID: "inbox"}, false},
		{"folder not a known sent folder", &message.Message{FolderID: "sent-2", FromEmail: "bob@example.com"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := owner.mine(tt.msg); got != tt.want {
				t.Errorf("mine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChatOwnerMineEmpty(t *testing.T) {
	var owner chatOwner
	if owner.mine(&message.Message{FolderID: "sent-1", FromEmail: "me@example.com"}) {
		t.Error("empty owner should match nothing")
	}
}

func TestAttachChatFields(t *testing.T) {
	owner := chatOwner{emails: map[string]struct{}{"me@example.com": {}}}
	attachChatFields(nil, owner)

	tests := []struct {
		name     string
		msg      *message.Message
		wantMine bool
		wantChat bool
	}{
		{"mine with body", &message.Message{BodyFetched: true, BodyText: "hi", FromEmail: "me@example.com"}, true, true},
		{"theirs without body", &message.Message{FromEmail: "bob@example.com"}, false, false},
		{"mine encrypted", &message.Message{BodyFetched: true, HasPGP: true, FromEmail: "ME@example.com"}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachChatFields(tt.msg, owner)
			if tt.msg.Mine != tt.wantMine {
				t.Errorf("Mine = %v, want %v", tt.msg.Mine, tt.wantMine)
			}
			if got := tt.msg.Chat != nil; got != tt.wantChat {
				t.Errorf("Chat set = %v, want %v", got, tt.wantChat)
			}
		})
	}
}
