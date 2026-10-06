package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	coreapi "github.com/hkdb/aerion/internal/core/api/v1"
	"github.com/hkdb/aerion/internal/credentials"
)

// newRefreshingClient returns a client for a custom account whose stored
// access token the API rejects; the token endpoint issues "fresh-token". It
// also returns the API URL and the bodies the API received, in order.
func newRefreshingClient(t *testing.T) (*http.Client, string, func() []string) {
	t.Helper()
	broker, credStore, db := newTestBroker(t)
	insertTestAccount(t, db, "acct")

	var mu sync.Mutex
	var bodies []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer fresh-token" {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	t.Cleanup(api.Close)
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fresh-token","expires_in":3600}`))
	}))
	t.Cleanup(tokenSrv.Close)

	if err := credStore.SetOAuthTokens("acct", &credentials.OAuthTokens{
		Provider:     "custom",
		AccessToken:  "stale-token",
		RefreshToken: "refresh-token",
	}); err != nil {
		t.Fatalf("set tokens: %v", err)
	}
	if err := credStore.SetCustomOAuthProvider("acct", credentials.CustomOAuthProvider{
		AuthURL:  tokenSrv.URL + "/auth",
		TokenURL: tokenSrv.URL,
		ClientID: "client-id",
	}); err != nil {
		t.Fatalf("set custom provider: %v", err)
	}

	scopes := []coreapi.AuthScope{{Resource: "https://example.com/calendar"}}
	client, err := broker.HTTPClientForExtension("calendar", coreapi.Manifest{}, "acct", scopes)
	if err != nil {
		t.Fatalf("HTTPClientForExtension: %v", err)
	}
	return client, api.URL, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), bodies...)
	}
}

func TestTransport_RetryAfterRefreshResendsBody(t *testing.T) {
	client, url, bodies := newRefreshingClient(t)

	resp, err := client.Post(url, "text/calendar", strings.NewReader("BEGIN:VCALENDAR"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := bodies(); len(got) != 2 || got[1] != "BEGIN:VCALENDAR" {
		t.Fatalf("API bodies = %q, want the retry to resend the body", got)
	}
}

func TestTransport_UnreplayableBodyReturns401(t *testing.T) {
	client, url, bodies := newRefreshingClient(t)

	req, err := http.NewRequest(http.MethodPut, url, io.NopCloser(strings.NewReader("BEGIN:VCALENDAR")))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if got := bodies(); len(got) != 1 {
		t.Fatalf("API saw %d requests, want 1 (no empty-body retry)", len(got))
	}
}
