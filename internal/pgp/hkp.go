package pgp

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// DefaultHKPServers is the default list of HKP key servers to query.
// keys.openpgp.org is listed first as it is email-verified and most trustworthy.
var DefaultHKPServers = []string{
	"https://keys.openpgp.org",
	"https://keyserver.ubuntu.com",
	"https://pgp.mit.edu",
}

// LookupHKP queries HKP key servers sequentially for the given email address.
// Returns the ASCII-armored public key if found, or empty string + nil error if not found.
// Only a key with a user ID for email counts as found.
// If servers is empty, DefaultHKPServers are used.
func LookupHKP(email string, servers []string) (string, error) {
	if !strings.Contains(email, "@") {
		return "", fmt.Errorf("invalid email address: %s", email)
	}

	if len(servers) == 0 {
		servers = DefaultHKPServers
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, server := range servers {
		entities, err := fetchHKP(client, server, email)
		if err != nil {
			continue
		}
		if armored, err := keyForEmail(entities, email); err == nil && armored != "" {
			return armored, nil
		}
	}

	return "", nil
}

// fetchHKP performs a single HKP lookup against one server and parses the
// keys it returns. Returns nil + nil error for HTTP 404 (key not found).
func fetchHKP(client *http.Client, serverURL, email string) (openpgp.EntityList, error) {
	u := fmt.Sprintf("%s/pks/lookup?op=get&search=%s&options=mr",
		strings.TrimRight(serverURL, "/"),
		url.QueryEscape(email),
	)

	resp, err := client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, serverURL)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	// Validate that the response contains a parseable PGP key
	entities, err := ParseArmoredKey(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HKP response from %s: %w", serverURL, err)
	}

	return entities, nil
}
