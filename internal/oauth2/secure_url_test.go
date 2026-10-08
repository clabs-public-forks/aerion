package oauth2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireSecureURL(t *testing.T) {
	tests := []struct {
		url string
		ok  bool
	}{
		{"https://auth.example.com/token", true},
		{"http://localhost:8080/token", true},
		{"http://127.0.0.1/token", true},
		{"http://[::1]:9000/token", true},
		{"http://auth.example.com/token", false},
		{"http://localhost.evil.com/token", false},
		{"ftp://auth.example.com/token", false},
		{"/token", false},
		{"", false},
	}
	for _, tt := range tests {
		if err := RequireSecureURL(tt.url, "token URL"); (err == nil) != tt.ok {
			t.Errorf("RequireSecureURL(%q) err = %v, want ok=%v", tt.url, err, tt.ok)
		}
	}
}

// TestDiscoverOIDCRejectsInsecureEndpoints checks that a discovery document
// pointing the token endpoint at plain http is refused.
func TestDiscoverOIDCRejectsInsecureEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"authorization_endpoint":"https://idp.example/auth","token_endpoint":"http://idp.example/token"}`))
	}))
	defer srv.Close()

	_, err := DiscoverOIDC(context.Background(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "token endpoint must use https") {
		t.Fatalf("DiscoverOIDC err = %v, want insecure token endpoint error", err)
	}
}

// TestTokenRequestRefusesInsecureURL checks that a stored custom provider
// with an http token URL never gets the refresh token sent to it.
func TestTokenRequestRefusesInsecureURL(t *testing.T) {
	m := NewManager()
	p := ProviderConfig{Name: "custom", ClientID: "id", TokenURL: "http://idp.example/token"}
	if _, err := m.RefreshTokenWithProvider(p, "refresh"); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("RefreshTokenWithProvider err = %v, want https error", err)
	}
}
