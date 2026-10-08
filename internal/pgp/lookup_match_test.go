package pgp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

// armorKeyRing armors one public key per address into a single key ring.
func armorKeyRing(t *testing.T, emails ...string) string {
	t.Helper()
	var buf bytes.Buffer
	w, err := armor.Encode(&buf, "PGP PUBLIC KEY BLOCK", nil)
	if err != nil {
		t.Fatalf("armor: %v", err)
	}
	for _, email := range emails {
		entity, err := openpgp.NewEntity("Someone", "", email, nil)
		if err != nil {
			t.Fatalf("new entity: %v", err)
		}
		if err := entity.Serialize(w); err != nil {
			t.Fatalf("serialize: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close armor: %v", err)
	}
	return buf.String()
}

func TestLookupHKPOnlyReturnsKeyForEmail(t *testing.T) {
	tests := []struct {
		name      string
		ringFor   []string
		lookup    string
		wantEmail string // "" means not found
	}{
		{name: "matching key", ringFor: []string{"alice@example.com"}, lookup: "alice@example.com", wantEmail: "alice@example.com"},
		{name: "case-insensitive match", ringFor: []string{"Alice@Example.com"}, lookup: "alice@example.com", wantEmail: "Alice@Example.com"},
		{name: "other address only", ringFor: []string{"mallory@example.com"}, lookup: "alice@example.com"},
		{name: "matching key after another", ringFor: []string{"mallory@example.com", "alice@example.com"}, lookup: "alice@example.com", wantEmail: "alice@example.com"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ring := armorKeyRing(t, tc.ringFor...)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(ring))
			}))
			defer srv.Close()

			armored, err := LookupHKP(tc.lookup, []string{srv.URL})
			if err != nil {
				t.Fatalf("LookupHKP: %v", err)
			}
			if tc.wantEmail == "" {
				if armored != "" {
					t.Fatal("LookupHKP returned a key for another address")
				}
				return
			}
			entities, err := ParseArmoredKey(armored)
			if err != nil {
				t.Fatalf("parse result: %v", err)
			}
			if len(entities) != 1 || ExtractEmailFromKey(entities[0]) != tc.wantEmail {
				t.Fatalf("LookupHKP returned %d keys for %q, want one for %q", len(entities), ExtractEmailFromKey(entities[0]), tc.wantEmail)
			}
		})
	}
}
