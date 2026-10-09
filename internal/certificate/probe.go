package certificate

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProbeResult reports the first certificate along a URL's redirect chain that
// neither the system CAs nor the trust store accept, and the host that
// presented it. Info is nil when every TLS handshake was trusted (or the URL
// is plain HTTP).
type ProbeResult struct {
	Host string
	Info *CertificateInfo
}

// Probe requests rawURL (following redirects) through the same host-scoped
// Transport the shared DAV clients use, so a setup
// flow whose connection failed can show the user which host's certificate to
// accept. A URL without a scheme is treated as https. Errors other than an
// untrusted certificate (unreachable host, bad URL) are returned as err; an
// HTTP error status is not an error, since only the TLS handshakes matter.
func Probe(ctx context.Context, rawURL string, store *Store) (ProbeResult, error) {
	rawURL = strings.TrimSpace(rawURL)
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ProbeResult{}, err
	}
	if u.Hostname() == "" {
		return ProbeResult{}, errors.New("URL has no host")
	}

	transport := NewTransport(store)
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ProbeResult{}, err
	}
	resp, err := client.Do(req)
	if err == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		resp.Body.Close()
		return ProbeResult{}, nil
	}

	var certErr *Error
	if !errors.As(err, &certErr) {
		return ProbeResult{}, err
	}
	// The *url.Error names the request that failed, which after a redirect
	// is not the URL we started with.
	host := u.Hostname()
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if failed, perr := url.Parse(urlErr.URL); perr == nil && failed.Hostname() != "" {
			host = failed.Hostname()
		}
	}
	return ProbeResult{Host: normalizeHost(host), Info: certErr.Info}, nil
}
