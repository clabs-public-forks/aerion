package pgp

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"golang.org/x/net/idna"
)

// LookupWKD performs a Web Key Directory lookup for a given email address.
// Returns the ASCII-armored public key if found, or empty string + nil error if not found.
// Only a key with a user ID for email counts as found.
func LookupWKD(email string) (string, error) {
	localpart, domain, err := wkdTarget(email)
	if err != nil {
		return "", err
	}

	// z-base-32 encode SHA-1 hash of the lowercased localpart
	hash := sha1.Sum([]byte(strings.ToLower(localpart)))
	encoded := zBase32Encode(hash[:])
	l := url.QueryEscape(localpart)

	client := &http.Client{Timeout: 5 * time.Second}

	// Try direct method first: https://<domain>/.well-known/openpgpkey/hu/<hash>?l=<localpart>
	directURL := fmt.Sprintf("https://%s/.well-known/openpgpkey/hu/%s?l=%s", domain, encoded, l)
	if armored := fetchWKD(client, directURL, email); armored != "" {
		return armored, nil
	}

	// Try advanced method: https://openpgpkey.<domain>/.well-known/openpgpkey/<domain>/hu/<hash>?l=<localpart>
	advancedURL := fmt.Sprintf("https://openpgpkey.%s/.well-known/openpgpkey/%s/hu/%s?l=%s", domain, domain, encoded, l)
	if armored := fetchWKD(client, advancedURL, email); armored != "" {
		return armored, nil
	}

	return "", nil
}

// wkdTarget splits email for a WKD lookup. The domain must be a DNS name
// with at least two labels (IDNs are converted to ASCII), so an address
// can't point the lookup at an IP, port, path, or local host.
func wkdTarget(email string) (localpart, domain string, err error) {
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 || strings.ContainsAny(email[:at], "@/\\ \t\r\n") {
		return "", "", fmt.Errorf("invalid email address: %s", email)
	}
	domain, err = idna.Lookup.ToASCII(strings.ToLower(email[at+1:]))
	if err != nil || !isDNSName(domain) {
		return "", "", fmt.Errorf("invalid email domain: %s", email)
	}
	return email[:at], domain, nil
}

// isDNSName reports whether s is a hostname of two or more LDH labels with a
// non-numeric top-level label (which rules out IPv4 literals).
func isDNSName(s string) bool {
	if len(s) > 253 {
		return false
	}
	labels := strings.Split(s, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	tld := labels[len(labels)-1]
	return strings.Trim(tld, "0123456789") != ""
}

// fetchWKD fetches url and returns the armored key it holds for email, or
// "" if the fetch fails or no key there has a user ID for email.
func fetchWKD(client *http.Client, url, email string) string {
	entities, err := fetchWKDEntities(client, url)
	if err != nil {
		return ""
	}
	armored, err := keyForEmail(entities, email)
	if err != nil {
		return ""
	}
	return armored
}

// fetchWKDEntities fetches a WKD URL and parses the keys it holds
func fetchWKDEntities(client *http.Client, url string) (openpgp.EntityList, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	// WKD returns binary key data
	entities, err := ParseBinaryKey(data)
	if err != nil {
		// Maybe it's already armored
		entities, err = ParseArmoredKey(string(data))
		if err != nil {
			return nil, fmt.Errorf("failed to parse WKD response: %w", err)
		}
	}

	return entities, nil
}

// zBase32Encode encodes bytes using z-base-32 encoding (RFC 6189)
func zBase32Encode(data []byte) string {
	const alphabet = "ybndrfg8ejkmcpqxot1uwisza345h769"

	var result strings.Builder
	buffer := 0
	bitsLeft := 0

	for _, b := range data {
		buffer = (buffer << 8) | int(b)
		bitsLeft += 8

		for bitsLeft >= 5 {
			bitsLeft -= 5
			index := (buffer >> bitsLeft) & 0x1F
			result.WriteByte(alphabet[index])
		}
	}

	if bitsLeft > 0 {
		index := (buffer << (5 - bitsLeft)) & 0x1F
		result.WriteByte(alphabet[index])
	}

	return result.String()
}
