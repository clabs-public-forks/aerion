package pgp

import "testing"

func TestWKDTarget(t *testing.T) {
	tests := []struct {
		email, local, domain string
		ok                   bool
	}{
		{"Alice@Example.COM", "Alice", "example.com", true},
		{"bob@mail.sub.example.org", "bob", "mail.sub.example.org", true},
		{"a&b#c@example.com", "a&b#c", "example.com", true},
		{"jo@bücher.example", "jo", "xn--bcher-kva.example", true},
		{"x@localhost", "", "", false},
		{"x@127.0.0.1", "", "", false},
		{"x@[::1]", "", "", false},
		{"x@example.com:8443", "", "", false},
		{"x@example.com/path", "", "", false},
		{"x@evil.com?y=", "", "", false},
		{"a@b@example.com", "", "", false},
		{"@example.com", "", "", false},
		{"x@", "", "", false},
		{"no-at-sign", "", "", false},
		{"x@-bad.example", "", "", false},
		{"x@example..com", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			local, domain, err := wkdTarget(tt.email)
			if (err == nil) != tt.ok {
				t.Fatalf("wkdTarget(%q) err = %v, want ok=%v", tt.email, err, tt.ok)
			}
			if local != tt.local || domain != tt.domain {
				t.Errorf("wkdTarget(%q) = %q, %q; want %q, %q", tt.email, local, domain, tt.local, tt.domain)
			}
		})
	}
}
