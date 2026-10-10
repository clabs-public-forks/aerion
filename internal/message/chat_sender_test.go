package message

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

const lms = "noreply@lms.example"

var lmsKey = SenderChatKey(lms)

// senderChatFixture extends chatFixture with mail from an automated sender:
// three unthreaded notifications (n1-n3), a thread it started with a human
// reply (t1, t2), a thread alice started that it replied to (o1, o2), and a
// notification in a second account (n9).
func senderChatFixture(t *testing.T) (*Store, string, string, time.Time) {
	t.Helper()
	s, accountID, inboxID, now := chatFixture(t)
	for _, q := range []string{
		`INSERT INTO accounts (id, name, email, imap_host, smtp_host, username) VALUES ('acct-2', 'Two', 'two@example.com', 'imap', 'smtp', 'two')`,
		`INSERT INTO folders (id, account_id, name, path, folder_type) VALUES ('folder-2', 'acct-2', 'INBOX', 'INBOX', 'inbox')`,
	} {
		if _, err := s.db.Exec(q); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	bulk, human := true, false
	seed := []struct {
		id, account, folder, thread, from, subject string
		age                                        time.Duration
		read                                       bool
		isBulk                                     *bool
	}{
		{"n1", accountID, inboxID, "", "NoReply@lms.example", "Grade posted: Math", 20 * time.Hour, false, &bulk},
		{"n2", accountID, inboxID, "", lms, "Grade posted: Art", 19 * time.Hour, false, &bulk},
		{"n3", accountID, inboxID, "", lms, "Quiz due", 18 * time.Hour, true, &bulk},
		{"t1", accountID, inboxID, "<t@x>", lms, "Forum post", 17 * time.Hour, true, &bulk},
		{"t2", accountID, inboxID, "<t@x>", "carol@example.com", "Re: Forum post", 16 * time.Hour, false, &human},
		{"o1", accountID, inboxID, "<o@x>", "alice@example.com", "Question", 15 * time.Hour, true, &human},
		{"o2", accountID, inboxID, "<o@x>", lms, "Re: Question", 14 * time.Hour, true, &bulk},
		{"n9", "acct-2", "folder-2", "", lms, "Grade posted: Two", 20 * time.Hour, false, &bulk},
	}
	for i, m := range seed {
		err := s.Create(&Message{
			ID: m.id, AccountID: m.account, FolderID: m.folder, UID: uint32(200 + i),
			ThreadID: m.thread, Subject: m.subject, FromEmail: m.from,
			Date: now.Add(-m.age), ReceivedAt: now.Add(-m.age), IsRead: m.read, IsBulk: m.isBulk,
		})
		if err != nil {
			t.Fatalf("seed %s: %v", m.id, err)
		}
	}
	return s, accountID, inboxID, now
}

func chatByKey(chats []*Chat, key string) *Chat {
	for _, c := range chats {
		if c.ThreadKey == key {
			return c
		}
	}
	return nil
}

func TestSenderChatFlag(t *testing.T) {
	s, accountID, _, _ := senderChatFixture(t)
	if err := s.SetSenderChat(accountID, "  ", true); err == nil {
		t.Error("SetSenderChat with empty email: want error")
	}
	steps := []struct {
		email    string
		combined bool
		want     bool
	}{
		{" NoReply@LMS.example ", true, true},
		{lms, true, true}, // idempotent
		{"NOREPLY@lms.example", false, false},
		{lms, false, false},
	}
	for _, st := range steps {
		if err := s.SetSenderChat(accountID, st.email, st.combined); err != nil {
			t.Fatalf("SetSenderChat(%q, %v): %v", st.email, st.combined, err)
		}
		got, err := s.IsSenderChat(accountID, lms)
		if err != nil || got != st.want {
			t.Errorf("after SetSenderChat(%q, %v): IsSenderChat = %v, %v; want %v", st.email, st.combined, got, err, st.want)
		}
	}
}

func TestSenderChatMapping(t *testing.T) {
	s, accountID, _, now := senderChatFixture(t)
	threadKeys := []string{"n1", "n2", "n3", "t@x", "o@x"}

	list := func(t *testing.T) []*Chat {
		t.Helper()
		chats, err := s.ListChats("", ChatSectionAll, now, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		n, err := s.CountChats("", ChatSectionAll, now)
		if err != nil || n != len(chats) {
			t.Errorf("CountChats = %d, %v; want %d", n, err, len(chats))
		}
		return chats
	}

	before := list(t)
	for _, key := range append(threadKeys, "n9") {
		if chatByKey(before, key) == nil {
			t.Errorf("flag off: chat %s missing from %v", key, chatKeys(before))
		}
	}

	if err := s.SetSenderChat(accountID, lms, true); err != nil {
		t.Fatal(err)
	}
	combined := list(t)
	if len(combined) != len(before)-3 {
		t.Errorf("flag on: %d chats, want %d: %v", len(combined), len(before)-3, chatKeys(combined))
	}
	tests := []struct {
		key     string
		present bool
	}{
		{lmsKey, true},
		{"n1", false}, {"n2", false}, {"n3", false},
		{"t@x", false}, // started by the sender: whole thread joins, human reply included
		{"o@x", true},  // started by alice: stays a thread chat
		{"n9", true},   // other account is unaffected
		{"a@x", true},
	}
	for _, tt := range tests {
		if got := chatByKey(combined, tt.key) != nil; got != tt.present {
			t.Errorf("flag on: chat %s present = %v, want %v", tt.key, got, tt.present)
		}
	}

	c := chatByKey(combined, lmsKey)
	if c == nil {
		t.Fatal("sender chat missing")
	}
	ids := append([]string(nil), c.MessageIDs...)
	sort.Strings(ids)
	if want := []string{"n1", "n2", "n3", "t1", "t2"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("MessageIDs = %v, want %v", ids, want)
	}
	if c.MessageCount != 5 || c.UnreadCount != 3 || c.SenderEmail != lms || c.AccountID != accountID {
		t.Errorf("sender chat = count %d unread %d sender %q account %q; want 5, 3, %q, %q",
			c.MessageCount, c.UnreadCount, c.SenderEmail, c.AccountID, lms, accountID)
	}
	if !c.LatestDate.Equal(now.Add(-16 * time.Hour)) {
		t.Errorf("LatestDate = %v, want t2's %v", c.LatestDate, now.Add(-16*time.Hour))
	}
	if c.Subject != "Re: Forum post" {
		t.Errorf("Subject = %q, want the latest message's %q", c.Subject, "Re: Forum post")
	}
	if o := chatByKey(combined, "o@x"); o != nil && (o.SenderEmail != "" || o.Subject != "Question") {
		t.Errorf("thread chat SenderEmail, Subject = %q, %q; want empty, %q", o.SenderEmail, o.Subject, "Question")
	}

	if err := s.SetSenderChat(accountID, lms, false); err != nil {
		t.Fatal(err)
	}
	if after := list(t); !reflect.DeepEqual(chatKeys(after), chatKeys(before)) {
		t.Errorf("flag off again: chats = %v, want %v", chatKeys(after), chatKeys(before))
	}
}

func TestSenderChatSections(t *testing.T) {
	s, accountID, inboxID, now := senderChatFixture(t)
	if err := s.SetSenderChat(accountID, lms, true); err != nil {
		t.Fatal(err)
	}
	inSection := func(section string) bool {
		t.Helper()
		chats, err := s.ListChats("", section, now, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		return chatByKey(chats, lmsKey) != nil
	}

	// carol's human reply in t@x lifts the chat out of Low.
	if !inSection(ChatSectionPriority) || inSection(ChatSectionLow) {
		t.Error("with a human reply: want Priority, not Low")
	}
	if _, err := s.db.Exec(`DELETE FROM messages WHERE id = 't2'`); err != nil {
		t.Fatal(err)
	}
	if inSection(ChatSectionPriority) || !inSection(ChatSectionLow) {
		t.Error("all bulk: want Low, not Priority")
	}

	if err := s.SetChatState(accountID, lmsKey, ChatState{SnoozedUntil: ptr(now.Add(time.Hour)), SnoozedAt: ptr(now)}); err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{ChatSectionAll, ChatSectionUnread, ChatSectionPriority, ChatSectionLow, ChatSectionSnoozed} {
		if got, want := inSection(section), section == ChatSectionSnoozed; got != want {
			t.Errorf("snoozed sender chat in %s = %v, want %v", section, got, want)
		}
	}

	// Pin and snooze survive new mail from the sender (the snooze wakes, as for threads).
	if err := s.SetChatState(accountID, lmsKey, ChatState{PinnedAt: ptr(now)}); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(&Message{ID: "n4", AccountID: accountID, FolderID: inboxID, UID: 300, Subject: "New grade",
		FromEmail: lms, Date: now, ReceivedAt: now}); err != nil {
		t.Fatal(err)
	}
	chats, err := s.ListChats("", ChatSectionAll, now, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if c := chatByKey(chats, lmsKey); c == nil || !c.IsPinned || c.MessageCount != 5 {
		t.Errorf("after new mail: sender chat = %+v, want pinned with 5 messages", c)
	}
}

func TestSenderChatLastFromMe(t *testing.T) {
	s, accountID, _, now := senderChatFixture(t)
	if err := s.SetSenderChat(accountID, lms, true); err != nil {
		t.Fatal(err)
	}
	// My reply in t@x is newer than every message in the sender chat.
	if err := s.Create(&Message{ID: "ts", AccountID: accountID, FolderID: "sent-1", UID: 301, ThreadID: "<t@x>",
		Subject: "Re: Forum post", FromEmail: "test@example.com", Date: now.Add(-time.Hour), ReceivedAt: now.Add(-time.Hour), IsRead: true}); err != nil {
		t.Fatal(err)
	}
	chats, err := s.ListChats("", ChatSectionAll, now, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]bool{lmsKey: true, "o@x": false, "n9": false} {
		if c := chatByKey(chats, key); c == nil || c.LastFromMe != want {
			t.Errorf("chat %s LastFromMe = %v, want %v", key, c != nil && c.LastFromMe, want)
		}
	}
}

func TestSenderChatSearch(t *testing.T) {
	s, accountID, inboxID, _ := senderChatFixture(t)
	if err := s.SetSenderChat(accountID, lms, true); err != nil {
		t.Fatal(err)
	}
	results, err := s.SearchChats(inboxID, "grade", 0, 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1 merged sender chat", len(results))
	}
	r := results[0]
	ids := append([]string(nil), r.MessageIDs...)
	sort.Strings(ids)
	if r.ThreadKey != lmsKey || r.SenderEmail != lms || r.MessageCount != 2 || r.UnreadCount != 2 ||
		!reflect.DeepEqual(ids, []string{"n1", "n2"}) {
		t.Errorf("result = key %q sender %q count %d unread %d ids %v; want %q, %q, 2, 2, [n1 n2]",
			r.ThreadKey, r.SenderEmail, r.MessageCount, r.UnreadCount, ids, lmsKey, lms)
	}

	// The merged row takes its newest message's subject and every thread's
	// participants.
	results, err = s.SearchChats(inboxID, "post", 0, 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1 merged sender chat", len(results))
	}
	r = results[0]
	var emails []string
	for _, p := range r.Participants {
		emails = append(emails, p.Email)
	}
	sort.Strings(emails)
	if r.Subject != "Re: Forum post" || r.HighlightedSubject == "" || r.MessageCount != 4 ||
		!reflect.DeepEqual(emails, []string{"carol@example.com", lms}) {
		t.Errorf("result = subject %q highlighted %q count %d participants %v", r.Subject, r.HighlightedSubject, r.MessageCount, emails)
	}

	results, err = s.SearchChats(inboxID, "question", 0, 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ThreadKey != "o@x" || results[0].SenderEmail != "" {
		t.Errorf("thread search = %+v, want o@x thread chat", results)
	}
}

func TestSenderChatDueSnooze(t *testing.T) {
	s, accountID, inboxID, now := senderChatFixture(t)
	if err := s.SetSenderChat(accountID, lms, true); err != nil {
		t.Fatal(err)
	}
	// t@x maps to the sender chat; o@x, started by alice, keeps its own row.
	snoozed := ChatState{SnoozedUntil: ptr(now.Add(-time.Minute)), SnoozedAt: ptr(now.Add(-17 * time.Hour))}
	for _, key := range []string{lmsKey, "t@x", "o@x"} {
		if err := s.SetChatState(accountID, key, snoozed); err != nil {
			t.Fatal(err)
		}
	}
	due, err := s.ListDueSnoozes(now)
	if err != nil {
		t.Fatal(err)
	}
	want := []DueSnooze{
		{AccountID: accountID, ThreadKey: "o@x", WokeByMail: true, LatestMessageID: "o2", FolderID: inboxID,
			ThreadID: "<o@x>", Subject: "Re: Question", FromEmail: lms},
		{AccountID: accountID, ThreadKey: lmsKey, WokeByMail: true, LatestMessageID: "t2", FolderID: inboxID,
			ThreadID: "<t@x>", Subject: "Re: Forum post", FromEmail: "carol@example.com"},
		{AccountID: accountID, ThreadKey: "t@x", InSenderChat: true},
	}
	if !reflect.DeepEqual(due, want) {
		t.Errorf("due = %+v, want %+v", due, want)
	}
}

func TestSenderChatCleanup(t *testing.T) {
	s, accountID, _, now := senderChatFixture(t)
	if err := s.SetSenderChat(accountID, lms, true); err != nil {
		t.Fatal(err)
	}
	pinned := ChatState{PinnedAt: ptr(now)}
	for _, key := range []string{lmsKey, SenderChatKey("ghost@x")} {
		if err := s.SetChatState(accountID, key, pinned); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetSenderChat(accountID, "ghost@x", true); err != nil {
		t.Fatal(err)
	}

	steps := []struct {
		name    string
		before  func() error
		removed int64
		kept    bool // sender chat state for lms
	}{
		{"combined senders keep state, with or without messages", func() error { return nil }, 0, true},
		{"messages leaving the inbox keep state", func() error {
			_, err := s.db.Exec(`UPDATE messages SET folder_id = 'sent-1' WHERE from_email LIKE ?`, lms)
			return err
		}, 0, true},
		{"flag off drops state", func() error { return s.SetSenderChat(accountID, lms, false) }, 1, false},
	}
	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			if err := st.before(); err != nil {
				t.Fatal(err)
			}
			n, err := s.CleanupChatState()
			if err != nil || n != st.removed {
				t.Errorf("CleanupChatState = %d, %v; want %d", n, err, st.removed)
			}
			got, err := s.GetChatState(accountID, lmsKey)
			if err != nil {
				t.Fatal(err)
			}
			if kept := got.PinnedAt != nil; kept != st.kept {
				t.Errorf("sender chat state kept = %v, want %v", kept, st.kept)
			}
		})
	}
}

func TestSenderChatNewMail(t *testing.T) {
	s, accountID, _, _ := senderChatFixture(t)
	tests := []struct {
		name     string
		combined bool
		want     map[string]string // subject -> chat key
	}{
		{"flag off", false, map[string]string{"Grade posted: Math": "n1", "Re: Forum post": "t@x", "Re: Question": "o@x", "a1": "a@x", "Grade posted: Two": "n9"}},
		{"flag on", true, map[string]string{"Grade posted: Math": lmsKey, "Re: Forum post": lmsKey, "Re: Question": "o@x", "a1": "a@x", "Grade posted: Two": "n9"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := s.SetSenderChat(accountID, lms, tt.combined); err != nil {
				t.Fatal(err)
			}
			mail, err := s.ListNewMail([]string{"n1", "t2", "o2", "a1", "n9"})
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]string{}
			for _, m := range mail {
				got[m.Subject] = m.ChatKey
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("chat keys = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSenderChatSearchUnifiedInbox(t *testing.T) {
	tests := []struct {
		name     string
		accounts []string // accounts with lms as a sender chat
		want     []string // account|thread key|message IDs
	}{
		{"no sender chats", nil, []string{"acct-2|n9|n9", "acct-1|n1|n1", "acct-1|n2|n2"}},
		{"one account", []string{"acct-1"}, []string{"acct-2|n9|n9", "acct-1|" + lmsKey + "|n1,n2"}},
		{"both accounts stay apart", []string{"acct-1", "acct-2"}, []string{"acct-2|" + lmsKey + "|n9", "acct-1|" + lmsKey + "|n1,n2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _, _, _ := senderChatFixture(t)
			for _, a := range tt.accounts {
				if err := s.SetSenderChat(a, lms, true); err != nil {
					t.Fatal(err)
				}
			}
			results, err := s.SearchChatsUnifiedInbox("grade", 0, 50, "")
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, r := range results {
				ids := append([]string(nil), r.MessageIDs...)
				sort.Strings(ids)
				got = append(got, r.AccountID+"|"+r.ThreadKey+"|"+strings.Join(ids, ","))
			}
			sort.Strings(got)
			want := append([]string(nil), tt.want...)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("results = %v, want %v", got, want)
			}
		})
	}
}
