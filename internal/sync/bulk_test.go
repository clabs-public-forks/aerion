package sync

import (
	"testing"
)

func TestClassifyBulk(t *testing.T) {
	tests := []struct {
		name   string
		header string
		from   string
		want   bool
	}{
		{"person", "Subject: hi\r\n", "alice@example.com", false},
		{"list-unsubscribe", "List-Unsubscribe: <mailto:u@x>\r\n", "news@shop.example", true},
		{"list-id", "List-Id: Dev list <dev.lists.example>\r\n", "bob@example.com", true},
		{"precedence bulk", "Precedence: bulk\r\n", "a@x", true},
		{"precedence list", "Precedence: List\r\n", "a@x", true},
		{"precedence junk", "Precedence: junk\r\n", "a@x", true},
		{"precedence first-class", "Precedence: first-class\r\n", "a@x", false},
		{"auto-submitted generated", "Auto-Submitted: auto-generated\r\n", "a@x", true},
		{"auto-submitted with params", "Auto-Submitted: auto-replied; owner-email=\"a@x\"\r\n", "a@x", true},
		{"auto-submitted no", "Auto-Submitted: no\r\n", "a@x", false},
		{"empty list-unsubscribe", "List-Unsubscribe: \r\n", "a@x", false},
		{"noreply", "", "noreply@service.example", true},
		{"no-reply", "", "No-Reply@service.example", true},
		{"no_reply suffix", "", "no_reply-billing@service.example", true},
		{"donotreply", "", "do.not.reply@service.example", true},
		{"notifications", "", "notifications@github.example", true},
		{"notification", "", "notification@service.example", true},
		{"notifications prefix only", "", "notifications-team@service.example", false},
		{"reply in name", "", "replies@service.example", false},
		{"no at sign", "", "noreply", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := []byte(tt.header + "\r\n")
			got := bulkFlag(raw, tt.from)
			if got == nil {
				t.Fatal("bulkFlag returned nil for a header block")
			}
			if *got != tt.want {
				t.Errorf("classifyBulk = %v, want %v", *got, tt.want)
			}
		})
	}

	if got := bulkFlag(nil, "noreply@x"); got != nil {
		t.Errorf("bulkFlag(nil) = %v, want nil", *got)
	}
}
