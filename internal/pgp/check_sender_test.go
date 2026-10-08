package pgp

import "testing"

func TestCheckSender(t *testing.T) {
	signer := generateTestKey(t)
	email := ExtractEmailFromKey(signer)

	tests := []struct {
		name   string
		status SignatureStatus
		from   string
		want   SignatureStatus
	}{
		{"matching from", StatusSigned, email, StatusSigned},
		{"forged from", StatusSigned, "ceo@example.com", StatusSignerMismatch},
		{"empty from", StatusSigned, "", StatusSignerMismatch},
		{"unknown key left alone", StatusUnknownKey, "ceo@example.com", StatusUnknownKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &SignatureResult{Status: tt.status, signer: signer}
			r.CheckSender(tt.from)
			if r.Status != tt.want {
				t.Fatalf("status = %q, want %q", r.Status, tt.want)
			}
		})
	}
}
