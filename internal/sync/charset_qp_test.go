package sync

import "testing"

// TestDecodeQuotedPrintableIfNeeded checks that the fallback only decodes
// bodies that don't declare an encoding go-message has already decoded.
func TestDecodeQuotedPrintableIfNeeded(t *testing.T) {
	tests := []struct {
		name, enc, in, want string
	}{
		{"declared qp already decoded", "quoted-printable", "a =\nb x=3D1", "a =\nb x=3D1"},
		{"declared base64 already decoded", "Base64", "x=3D1", "x=3D1"},
		{"undeclared qp decoded", "", "x=3D1 soft=\nbreak", "x=1 softbreak"},
		{"plain text untouched", "7bit", "x = 1", "x = 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(decodeQuotedPrintableIfNeeded([]byte(tt.in), tt.enc)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
