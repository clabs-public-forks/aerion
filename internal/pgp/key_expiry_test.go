package pgp

import (
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// TestIsKeyExpiredUsesPrimaryIdentity checks that expiry and metadata come
// from the primary identity every time, not whichever identity map
// iteration yields first, and that a zero lifetime means no expiry.
func TestIsKeyExpiredUsesPrimaryIdentity(t *testing.T) {
	entity, err := openpgp.NewEntity("Primary", "", "primary@example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := entity.AddUserId("Other", "", "other@example.com", nil); err != nil {
		t.Fatal(err)
	}
	entity.PrimaryKey.CreationTime = time.Now().Add(-time.Hour)
	minute := uint32(60)
	zero := uint32(0)
	for _, ident := range entity.Identities {
		primary := ident.UserId.Email == "primary@example.com"
		ident.SelfSignature.IsPrimaryId = &primary
		ident.SelfSignature.KeyLifetimeSecs = &minute
		if primary {
			ident.SelfSignature.KeyLifetimeSecs = &zero
		}
	}

	for i := 0; i < 50; i++ {
		if IsKeyExpired(entity) {
			t.Fatal("IsKeyExpired = true; the primary identity's lifetime is 0 (never expires)")
		}
		meta := ExtractKeyMetadata(entity)
		if meta.Email != "primary@example.com" || meta.IsExpired || meta.ExpiresAtKey != nil {
			t.Fatalf("metadata = %q expired=%v expires=%v; want primary identity, no expiry", meta.Email, meta.IsExpired, meta.ExpiresAtKey)
		}
		if got := ExtractEmailFromKey(entity); got != "primary@example.com" {
			t.Fatalf("ExtractEmailFromKey = %q, want primary@example.com", got)
		}
	}

	for _, ident := range entity.Identities {
		ident.SelfSignature.KeyLifetimeSecs = &minute
	}
	if !IsKeyExpired(entity) {
		t.Error("IsKeyExpired = false for a key whose 60s lifetime ended an hour ago")
	}
}
