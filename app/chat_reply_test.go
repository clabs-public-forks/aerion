package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hkdb/aerion/internal/account"
	"github.com/hkdb/aerion/internal/database"
	"github.com/hkdb/aerion/internal/draft"
	"github.com/hkdb/aerion/internal/message"
	"github.com/hkdb/aerion/internal/settings"
	"github.com/hkdb/aerion/internal/smtp"
)

func TestChatTextToHTML(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"  \n ", ""},
		{"hi", "<p>hi</p>"},
		{"a\nb", "<p>a<br>b</p>"},
		{"a\n\nb", "<p>a</p><p>b</p>"},
		{"a\n \n\n b", "<p>a</p><p>b</p>"},
		{"<b>&\"", "<p>&lt;b&gt;&amp;&quot;</p>"},
	}
	for _, tt := range tests {
		if got := chatTextToHTML(tt.in); got != tt.want {
			t.Errorf("chatTextToHTML(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestChatSignatureFor(t *testing.T) {
	on := account.Identity{SignatureEnabled: true, SignatureForReply: true, SignatureHTML: "<p>Al</p>", SignatureText: "Al"}
	tests := []struct {
		name string
		id   *account.Identity
		want chatSignature
	}{
		{"nil identity", nil, chatSignature{}},
		{"disabled", &account.Identity{SignatureForReply: true, SignatureHTML: "<p>Al</p>"}, chatSignature{}},
		{"not for replies", &account.Identity{SignatureEnabled: true, SignatureHTML: "<p>Al</p>"}, chatSignature{}},
		{"empty", &account.Identity{SignatureEnabled: true, SignatureForReply: true}, chatSignature{}},
		{"above", &on, chatSignature{html: "<p>Al</p>", text: "Al"}},
		{"below with separator", func() *account.Identity {
			id := on
			id.SignaturePlacement, id.SignatureSeparator = "below", true
			return &id
		}(), chatSignature{html: "<p>-- </p><p>Al</p>", text: "-- \nAl", below: true}},
		{"text derived from html", &account.Identity{SignatureEnabled: true, SignatureForReply: true, SignatureHTML: "<p>Al</p>"}, chatSignature{html: "<p>Al</p>", text: "Al"}},
		{"html derived from text", &account.Identity{SignatureEnabled: true, SignatureForReply: true, SignatureText: "A<l"}, chatSignature{html: "<p>A&lt;l</p>", text: "A<l"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chatSignatureFor(tt.id)
			got.text = strings.TrimSpace(got.text)
			if got != tt.want {
				t.Errorf("chatSignatureFor = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAssembleChatReply(t *testing.T) {
	prepared := func() *smtp.ComposeMessage {
		return &smtp.ComposeMessage{
			HTMLBody: `<p></p><p></p><p>On Mon, Bob wrote:</p><blockquote type="cite"><img src="cid:logo"></blockquote>`,
			TextBody: "\n\nOn Mon, Bob wrote:\n> hi",
			Attachments: []smtp.Attachment{
				{Filename: "logo.png", ContentID: "logo", Inline: true},
			},
		}
	}
	sig := chatSignature{html: "<p>Al</p>", text: "Al"}
	sigBelow := chatSignature{html: "<p>Al</p>", text: "Al", below: true}
	quoteHTML := `<p>On Mon, Bob wrote:</p><blockquote type="cite"><img src="cid:logo"></blockquote>`
	quoteText := "On Mon, Bob wrote:\n> hi"

	tests := []struct {
		name            string
		sig             chatSignature
		quote           bool
		wantHTML        string
		wantText        string
		wantAttachments int
	}{
		{"quote only", chatSignature{}, true, "<p>Thanks</p>" + quoteHTML, "Thanks\n\n" + quoteText, 1},
		{"signature above quote", sig, true, "<p>Thanks</p><p>Al</p>" + quoteHTML, "Thanks\n\nAl\n\n" + quoteText, 1},
		{"signature below quote", sigBelow, true, "<p>Thanks</p>" + quoteHTML + "<p>Al</p>", "Thanks\n\n" + quoteText + "\n\nAl", 1},
		{"no quote drops its inline parts", sigBelow, false, "<p>Thanks</p><p>Al</p>", "Thanks\n\nAl", 0},
		{"no quote, no signature", chatSignature{}, false, "<p>Thanks</p>", "Thanks", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := prepared()
			assembleChatReply(msg, "Thanks", tt.sig, tt.quote)
			if msg.HTMLBody != tt.wantHTML {
				t.Errorf("HTMLBody = %q, want %q", msg.HTMLBody, tt.wantHTML)
			}
			if msg.TextBody != tt.wantText {
				t.Errorf("TextBody = %q, want %q", msg.TextBody, tt.wantText)
			}
			if len(msg.Attachments) != tt.wantAttachments {
				t.Errorf("attachments = %d, want %d", len(msg.Attachments), tt.wantAttachments)
			}
			if got := chatDraftText(msg.TextBody, len("Thanks")); got != "Thanks" {
				t.Errorf("chatDraftText = %q, want the typed text", got)
			}
		})
	}
}

func TestChatDraftText(t *testing.T) {
	tests := []struct {
		body string
		n    int
		want string
	}{
		{"hi\n\nquote", 2, "hi"},
		{"hi", 0, ""},
		{"hi", 5, "hi"},       // body shorter than recorded: return it whole
		{"héllo", 2, "héllo"}, // cut inside a rune: return it whole
		{"hi", -1, "hi"},
	}
	for _, tt := range tests {
		if got := chatDraftText(tt.body, tt.n); got != tt.want {
			t.Errorf("chatDraftText(%q, %d) = %q, want %q", tt.body, tt.n, got, tt.want)
		}
	}
}

func TestNormalizeChatText(t *testing.T) {
	if got := normalizeChatText("a\r\nb  \n\n"); got != "a\nb" {
		t.Errorf("normalizeChatText = %q", got)
	}
}

// newChatReplyTestApp returns an App backed by a migrated database holding
// one account with a signed identity and one inbox message from Bob to the
// account and Carol.
func newChatReplyTestApp(t *testing.T) *App {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`INSERT INTO accounts (id, name, email, imap_host, smtp_host, username)
			VALUES ('acct-1', 'Me', 'me@example.com', 'imap.example.com', 'smtp.example.com', 'me@example.com')`,
		`INSERT INTO folders (id, account_id, name, path, folder_type) VALUES ('f-1', 'acct-1', 'INBOX', 'INBOX', 'inbox')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	a := &App{
		accountStore:    account.NewStore(db),
		messageStore:    message.NewStore(db),
		attachmentStore: message.NewAttachmentStore(db),
		draftStore:      draft.NewStore(db),
		settingsStore:   settings.NewStore(db),
	}
	if _, err := a.accountStore.CreateIdentity("acct-1", &account.IdentityConfig{
		Email: "me@example.com", Name: "Me", SignatureEnabled: true, SignatureForReply: true,
		SignatureHTML: "<p>Me</p>", SignatureText: "Me",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.messageStore.Create(&message.Message{
		ID: "m-1", AccountID: "acct-1", FolderID: "f-1", UID: 1, MessageID: "<orig@example.com>",
		Subject: "Lunch", FromName: "Bob", FromEmail: "bob@example.com",
		ToList: "me@example.com, Carol <carol@example.com>", Date: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		BodyText: "Lunch at noon?", BodyFetched: true,
	}); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestBuildChatReply(t *testing.T) {
	a := newChatReplyTestApp(t)
	addrs := func(list []smtp.Address) string {
		var out []string
		for _, ad := range list {
			out = append(out, ad.Address)
		}
		return strings.Join(out, ",")
	}

	tests := []struct {
		name     string
		replyAll bool
		quote    bool
		wantTo   string
		wantText string
	}{
		{"reply with quote", false, true, "bob@example.com", "Sure\n\nMe\n\nOn "},
		{"reply-all without quote", true, false, "bob@example.com,carol@example.com", "Sure\n\nMe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := a.settingsStore.SetChatIncludeQuote(tt.quote); err != nil {
				t.Fatal(err)
			}
			msg, err := a.buildChatReply(ChatReply{
				AccountID: "acct-1", MessageID: "m-1", Text: "Sure\r\n", ReplyAll: tt.replyAll,
				Attachments: []smtp.Attachment{{Filename: "a.txt", ContentBase64: "aGk="}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := addrs(msg.To); got != tt.wantTo {
				t.Errorf("To = %q, want %q", got, tt.wantTo)
			}
			if msg.Subject != "Re: Lunch" {
				t.Errorf("Subject = %q", msg.Subject)
			}
			if msg.InReplyTo != "<orig@example.com>" {
				t.Errorf("InReplyTo = %q", msg.InReplyTo)
			}
			if got := strings.Join(msg.References, " "); !strings.HasSuffix(got, "<orig@example.com>") {
				t.Errorf("References = %q", got)
			}
			if !strings.HasPrefix(msg.TextBody, tt.wantText) {
				t.Errorf("TextBody = %q, want prefix %q", msg.TextBody, tt.wantText)
			}
			if tt.quote != strings.Contains(msg.TextBody, "> Lunch at noon?") {
				t.Errorf("quote in TextBody = %v, want %v: %q", !tt.quote, tt.quote, msg.TextBody)
			}
			if len(msg.Attachments) != 1 || msg.Attachments[0].Filename != "a.txt" {
				t.Errorf("Attachments = %+v, want the user's file", msg.Attachments)
			}
		})
	}

	if _, err := a.buildChatReply(ChatReply{AccountID: "acct-1"}); err == nil {
		t.Error("buildChatReply without a message: want error")
	}
}
