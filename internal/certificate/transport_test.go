package certificate

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestTransport checks that trust is scoped to the request host, including an
// IP-literal host (no SNI, so the TLS connection state carries no server name).
func TestTransport(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	fingerprint := Fingerprint(srv.Certificate().Raw)

	store := openTestStore(t)
	client := &http.Client{Transport: NewTransport(store)}
	get := func(rawURL string) error {
		resp, err := client.Get(rawURL)
		if err == nil {
			resp.Body.Close()
		}
		return err
	}

	var certErr *Error
	if err := get(srv.URL); !errors.As(err, &certErr) {
		t.Fatalf("untrusted: want *Error, got %v", err)
	}

	if err := store.AcceptSession("127.0.0.1", fingerprint); err != nil {
		t.Fatal(err)
	}
	if err := get(srv.URL); err != nil {
		t.Fatalf("trusted for 127.0.0.1: %v", err)
	}

	// Same listener, different host name: the pin does not carry over.
	if err := get("https://localhost:" + u.Port() + "/"); !errors.As(err, &certErr) {
		t.Fatalf("localhost: want *Error, got %v", err)
	}
}
