package oauth2

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestRefreshTokenWithProviderErrors(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		status       int
		body         string
		wantRequest  bool
		wantSentinel error
		wantMsg      string
	}{
		{name: "missing client ID", clientID: "", wantSentinel: ErrNotConfigured,
			wantMsg: "OAuth provider is not configured (missing client ID): google"},
		{name: "server error", clientID: "id", status: http.StatusInternalServerError,
			body: `{"error":"server_error"}`, wantRequest: true},
		{name: "invalid_client", clientID: "id", status: http.StatusUnauthorized,
			body: `{"error":"invalid_client","error_description":"Unauthorized"}`, wantRequest: true},
		{name: "invalid_grant", clientID: "id", status: http.StatusBadRequest,
			body: `{"error":"invalid_grant","error_description":"Bad Request"}`, wantRequest: true,
			wantSentinel: ErrInvalidGrant, wantMsg: "token refresh failed: invalid_grant - Bad Request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(srv.Close)

			provider := ProviderConfig{Name: "google", TokenURL: srv.URL, ClientID: tt.clientID}
			_, err := NewManager().RefreshTokenWithProvider(provider, "refresh-token")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := requests.Load() > 0; got != tt.wantRequest {
				t.Errorf("request made: got %v, want %v", got, tt.wantRequest)
			}
			for _, sentinel := range []error{ErrNotConfigured, ErrInvalidGrant} {
				if got, want := errors.Is(err, sentinel), sentinel == tt.wantSentinel; got != want {
					t.Errorf("errors.Is(%v, %v) = %v, want %v", err, sentinel, got, want)
				}
			}
			if tt.wantMsg != "" && err.Error() != tt.wantMsg {
				t.Errorf("message: got %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}
