package v1

import "testing"

func TestErrAdditionalConsentRequired_Error(t *testing.T) {
	missing := &ErrAdditionalConsentRequired{
		AccountID:      "acct",
		ClientConfigID: "google-calendar",
		MissingScopes:  []AuthScope{{Resource: "a"}, {Resource: "b"}},
	}
	if got, want := missing.Error(), "additional consent required for account acct under google-calendar: 2 scope(s) missing"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	revoked := *missing
	revoked.Reason = "refresh token expired or revoked"
	if got, want := revoked.Error(), "additional consent required for account acct under google-calendar: refresh token expired or revoked"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
