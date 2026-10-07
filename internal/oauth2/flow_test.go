package oauth2

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func tokenEndpoint(t *testing.T, status int, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestRefreshTokenWithProvider_InvalidGrant(t *testing.T) {
	url := tokenEndpoint(t, http.StatusBadRequest, `{"error":"invalid_grant","error_description":"Bad Request"}`)

	_, err := NewManager().RefreshTokenWithProvider(ProviderConfig{TokenURL: url, ClientID: "id"}, "dead-refresh-token")
	if !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant, got %v", err)
	}
	if got, want := err.Error(), "token refresh failed: invalid_grant - Bad Request"; got != want {
		t.Errorf("message: got %q, want %q", got, want)
	}
}

func TestRefreshTokenWithProvider_OtherErrorIsNotInvalidGrant(t *testing.T) {
	url := tokenEndpoint(t, http.StatusUnauthorized, `{"error":"invalid_client","error_description":"Unauthorized"}`)

	_, err := NewManager().RefreshTokenWithProvider(ProviderConfig{TokenURL: url, ClientID: "id"}, "refresh-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("invalid_client must not match ErrInvalidGrant: %v", err)
	}
}
