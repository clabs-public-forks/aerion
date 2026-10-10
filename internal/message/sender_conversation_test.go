package message

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

func conversationIDs(c *Conversation) []string {
	if c == nil {
		return nil
	}
	var ids []string
	for _, m := range c.Messages {
		ids = append(ids, m.ID)
	}
	return ids
}

func TestGetSenderConversation(t *testing.T) {
	s, accountID, inboxID, now := senderChatFixture(t)
	if _, err := s.GetSenderConversation(accountID, " ", ""); err == nil {
		t.Error("GetSenderConversation with empty email: want error")
	}
	if c, err := s.GetSenderConversation(accountID, lms, ""); err != nil || c != nil {
		t.Errorf("before combining = %v, %v; want nil", conversationIDs(c), err)
	}

	// A Sent reply in the lms thread, and an unrelated Sent message.
	for _, m := range []*Message{
		{ID: "r1", AccountID: accountID, FolderID: "sent-1", UID: 300, ThreadID: "<t@x>", Subject: "Re: Forum post",
			FromEmail: "test@example.com", Date: now.Add(-15*time.Hour - 30*time.Minute), IsRead: true},
		{ID: "r2", AccountID: accountID, FolderID: "sent-1", UID: 301, ThreadID: "<o@x>", Subject: "Re: Question",
			FromEmail: "test@example.com", Date: now.Add(-time.Hour), IsRead: true},
		// A reply stored without a thread ID, matched by In-Reply-To.
		{ID: "r3", AccountID: accountID, FolderID: "sent-1", UID: 302, InReplyTo: "<t@x>", Subject: "Re: Forum post",
			FromEmail: "test@example.com", Date: now.Add(-15 * time.Hour), IsRead: true},
	} {
		if err := s.Create(m); err != nil {
			t.Fatal(err)
		}
	}
	for _, acct := range []string{accountID, "acct-2"} {
		if err := s.SetSenderChat(acct, lms, true); err != nil {
			t.Fatal(err)
		}
	}

	all := []string{"n1", "n2", "n3", "t1", "t2", "r1", "r3"}
	tests := []struct {
		name, account, email, folder string
		want                         []string
	}{
		{"inbox scope merges Sent", accountID, lms, "", all},
		{"folder scope", accountID, " NoReply@LMS.example ", inboxID, all},
		{"chat key as email", accountID, lmsKey, "", all},
		{"sent folder has no lms threads", accountID, lms, "sent-1", nil},
		{"other account", "acct-2", lms, "", []string{"n9"}},
		{"other sender", accountID, "alice@example.com", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := s.GetSenderConversation(tt.account, tt.email, tt.folder)
			if err != nil {
				t.Fatal(err)
			}
			if got := conversationIDs(c); !slices.Equal(got, tt.want) {
				t.Fatalf("messages = %v, want %v", got, tt.want)
			}
			if c == nil {
				return
			}
			if c.ThreadID != lmsKey || c.AccountID != tt.account || c.FolderID != tt.folder {
				t.Errorf("ThreadID, AccountID, FolderID = %q, %q, %q", c.ThreadID, c.AccountID, c.FolderID)
			}
			if c.MessageCount != len(tt.want) || !slices.Equal(c.MessageIDs, tt.want) {
				t.Errorf("MessageCount = %d, MessageIDs = %v", c.MessageCount, c.MessageIDs)
			}
		})
	}

	c, err := s.GetSenderConversation(accountID, lms, "")
	if err != nil {
		t.Fatal(err)
	}
	if c.UnreadCount != 3 || c.Subject != "Re: Forum post" || !c.LatestDate.Equal(now.Add(-15*time.Hour)) {
		t.Errorf("UnreadCount, Subject, LatestDate = %d, %q, %v", c.UnreadCount, c.Subject, c.LatestDate)
	}
	var senders []string
	for _, p := range c.Participants {
		senders = append(senders, p.Email)
	}
	if want := []string{"test@example.com", "carol@example.com", lms}; !slices.Equal(senders, want) {
		t.Errorf("Participants = %v, want %v", senders, want)
	}

	// The chat row lists the same inbox messages.
	chats, err := s.ListChats("", ChatSectionAll, now, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if row := chatByKey(chats, lmsKey); row == nil || row.MessageCount != 5 {
		t.Errorf("chat row = %+v, want 5 inbox messages", row)
	}
}

// TestGetSenderConversationTiming loads a 600-message sender chat and a
// 600-message thread, logging both so a regression shows in -v output.
func TestGetSenderConversationTiming(t *testing.T) {
	s, accountID, inboxID, now := senderChatFixture(t)
	const n = 600
	for i := range n {
		thread := ""
		if i%2 == 0 {
			thread = fmt.Sprintf("<big%d@x>", i/2) // half are one-message threads
		}
		if err := s.Create(&Message{
			ID: fmt.Sprintf("p%d", i), AccountID: accountID, FolderID: inboxID, UID: uint32(1000 + i), ThreadID: thread,
			Subject: "Notice", FromEmail: lms, Date: now.Add(-time.Duration(n-i) * time.Minute), BodyText: "body",
		}); err != nil {
			t.Fatal(err)
		}
		if err := s.Create(&Message{
			ID: fmt.Sprintf("q%d", i), AccountID: accountID, FolderID: inboxID, UID: uint32(5000 + i), ThreadID: "<long@x>",
			Subject: "Long", FromEmail: "dan@example.com", Date: now.Add(-time.Duration(n-i) * time.Minute), BodyText: "body",
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetSenderChat(accountID, lms, true); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	c, err := s.GetSenderConversation(accountID, lms, "")
	sender := time.Since(start)
	if err != nil || c == nil || c.MessageCount != n+5 {
		t.Fatalf("sender chat = %v, %v; want %d messages", c, err, n+5)
	}
	start = time.Now()
	th, err := s.GetConversation("<long@x>", inboxID)
	thread := time.Since(start)
	if err != nil || th == nil || th.MessageCount != n {
		t.Fatalf("thread = %v, %v; want %d messages", th, err, n)
	}
	t.Logf("sender chat (%d messages): %v; single thread (%d messages): %v", c.MessageCount, sender, n, thread)
	if sender > 2*time.Second {
		t.Errorf("sender chat took %v, want under 2s", sender)
	}
}
