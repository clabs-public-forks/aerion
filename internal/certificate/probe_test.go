package certificate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestProbe drives Probe against real self-signed TLS servers: it reports the
// untrusted certificate with the host that presented it (including the target
// of a cross-host redirect), and reports nothing once that host is trusted.
func TestProbe(t *testing.T) {
	// "localhost" and "127.0.0.1" reach the same listener but are distinct
	// hosts to the trust store, which lets one test cover a cross-host redirect.
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized) // status codes are irrelevant
	}))
	defer target.Close()
	targetURL, _ := url.Parse(target.URL)
	localhostURL := "https://localhost:" + targetURL.Port() + "/dav/"

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, localhostURL, http.StatusFound)
	}))
	defer redirector.Close()

	fingerprint := Fingerprint(target.Certificate().Raw)

	tests := []struct {
		name     string
		trust    []string // hosts the target certificate is trusted for
		url      string
		wantHost string // "" means no certificate reported
	}{
		{name: "untrusted direct", url: target.URL, wantHost: "127.0.0.1"},
		{name: "no scheme defaults to https", url: strings.TrimPrefix(target.URL, "https://"), wantHost: "127.0.0.1"},
		{name: "trusted for host", trust: []string{"127.0.0.1"}, url: target.URL},
		{name: "trusted for another host only", trust: []string{"localhost"}, url: target.URL, wantHost: "127.0.0.1"},
		{name: "redirect reports final host", url: redirector.URL, wantHost: "localhost"},
		{name: "redirect target trusted", trust: []string{"localhost"}, url: redirector.URL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openTestStore(t)
			for _, h := range tt.trust {
				if err := store.AcceptSession(h, fingerprint); err != nil {
					t.Fatalf("AcceptSession: %v", err)
				}
			}
			res, err := Probe(context.Background(), tt.url, store)
			if err != nil {
				t.Fatalf("Probe: %v", err)
			}
			if tt.wantHost == "" {
				if res.Info != nil {
					t.Fatalf("want no certificate, got host %q", res.Host)
				}
				return
			}
			if res.Info == nil {
				t.Fatal("want an untrusted certificate, got none")
			}
			if res.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", res.Host, tt.wantHost)
			}
			if res.Info.Fingerprint != fingerprint {
				t.Errorf("Fingerprint = %q, want %q", res.Info.Fingerprint, fingerprint)
			}
		})
	}
}

func TestProbeErrors(t *testing.T) {
	store := openTestStore(t)
	for _, u := range []string{"https://", "https://127.0.0.1:1/"} {
		if _, err := Probe(context.Background(), u, store); err == nil {
			t.Errorf("Probe(%q): want error", u)
		}
	}
}
