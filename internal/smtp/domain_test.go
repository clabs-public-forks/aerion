package smtp

import "testing"

func TestDomainFromEmail(t *testing.T) {
	cases := []struct {
		email string
		want  string
	}{
		{"me@example.com", "example.com"},
		{"me", "fallback"},
		{"me@", "fallback"},
		{"a@b@example.com", "fallback"},
		{"", "fallback"},
	}
	for _, c := range cases {
		if got := domainFromEmail(c.email, "fallback"); got != c.want {
			t.Errorf("domainFromEmail(%q) = %q, want %q", c.email, got, c.want)
		}
	}
}
