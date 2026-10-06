package auth

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	coreapi "github.com/hkdb/aerion/internal/core/api/v1"
	"github.com/hkdb/aerion/internal/credentials"
	"github.com/hkdb/aerion/internal/oauth2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func respond(r *http.Request, status int) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(http.StatusText(status))), Request: r}
}

// newCustomAccountClient returns an extension client for a custom ("bring
// your own app") account holding a stale access token. api stands in for the
// network (it does no body rewinding of its own, unlike http.Transport) and
// token serves the account's token endpoint.
func newCustomAccountClient(t *testing.T, api roundTripFunc, token http.HandlerFunc) *http.Client {
	t.Helper()
	broker, credStore, db := newTestBroker(t)
	broker.baseTransport = api
	insertTestAccount(t, db, "acct")

	tokenSrv := httptest.NewServer(token)
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
	return client
}

func tokenResponse(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// newRefreshingClient's API rejects every token but "fresh-token", which the
// token endpoint issues. It records each request body it receives.
func newRefreshingClient(t *testing.T) (*http.Client, *[]string) {
	var bodies []string
	api := func(r *http.Request) (*http.Response, error) {
		var b []byte
		if r.Body != nil {
			b, _ = io.ReadAll(r.Body)
		}
		bodies = append(bodies, string(b))
		if r.Header.Get("Authorization") != "Bearer fresh-token" {
			return respond(r, http.StatusUnauthorized), nil
		}
		return respond(r, http.StatusOK), nil
	}
	token := tokenResponse(http.StatusOK, `{"access_token":"fresh-token","expires_in":3600}`)
	return newCustomAccountClient(t, api, token), &bodies
}

// A refresh token the provider rejects (invalid_grant: expired, revoked) on a
// custom ("bring your own app") account lives in the custom-mail slot, which no
// extension grant flow can re-grant. The transport must keep the refresh error
// (so mail re-auth handles it) instead of offering a "Grant access" button that
// would loop.
func TestTransport_InvalidGrantOnMailSlotKeepsRefreshError(t *testing.T) {
	api := func(r *http.Request) (*http.Response, error) { return respond(r, http.StatusUnauthorized), nil }
	token := tokenResponse(http.StatusBadRequest, `{"error":"invalid_grant","error_description":"Bad Request"}`)
	client := newCustomAccountClient(t, api, token)

	if _, err := client.Get("https://api.test/"); !errors.Is(err, oauth2.ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant, got %T: %v", err, err)
	}
}

func TestTransport_RetryAfterRefreshResendsBody(t *testing.T) {
	client, bodies := newRefreshingClient(t)

	resp, err := client.Post("https://api.test/", "text/calendar", strings.NewReader("BEGIN:VCALENDAR"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := *bodies; len(got) != 2 || got[1] != "BEGIN:VCALENDAR" {
		t.Fatalf("API bodies = %q, want the retry to resend the body", got)
	}
}

// A body without GetBody can't be resent, so the caller gets the 401 (body
// intact) instead of an empty-body retry — but the token is still refreshed,
// so the caller's own retry succeeds.
func TestTransport_UnreplayableBodyReturns401AfterRefresh(t *testing.T) {
	client, bodies := newRefreshingClient(t)

	req, err := http.NewRequest(http.MethodPut, "https://api.test/", io.NopCloser(strings.NewReader("BEGIN:VCALENDAR")))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || string(body) != "Unauthorized" {
		t.Fatalf("got %d %q, want the 401 with its body", resp.StatusCode, body)
	}
	if n := len(*bodies); n != 1 {
		t.Fatalf("API saw %d requests, want 1 (no empty-body retry)", n)
	}

	resp, err = client.Get("https://api.test/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || len(*bodies) != 2 {
		t.Fatalf("follow-up: status %d after %d API requests, want 200 on the first try", resp.StatusCode, len(*bodies))
	}
}
