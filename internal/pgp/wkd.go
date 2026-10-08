package pgp

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// LookupWKD performs a Web Key Directory lookup for a given email address.
// Returns the ASCII-armored public key if found, or empty string + nil error if not found.
// Only a key with a user ID for email counts as found.
func LookupWKD(email string) (string, error) {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid email address: %s", email)
	}

	localpart := strings.ToLower(parts[0])
	domain := strings.ToLower(parts[1])

	// z-base-32 encode SHA-1 hash of the localpart
	hash := sha1.Sum([]byte(localpart))
	encoded := zBase32Encode(hash[:])

	client := &http.Client{Timeout: 5 * time.Second}

	// Try direct method first: https://<domain>/.well-known/openpgpkey/hu/<hash>?l=<localpart>
	directURL := fmt.Sprintf("https://%s/.well-known/openpgpkey/hu/%s?l=%s", domain, encoded, localpart)
	if armored := fetchWKD(client, directURL, email); armored != "" {
		return armored, nil
	}

	// Try advanced method: https://openpgpkey.<domain>/.well-known/openpgpkey/<domain>/hu/<hash>?l=<localpart>
	advancedURL := fmt.Sprintf("https://openpgpkey.%s/.well-known/openpgpkey/%s/hu/%s?l=%s", domain, domain, encoded, localpart)
	if armored := fetchWKD(client, advancedURL, email); armored != "" {
		return armored, nil
	}

	return "", nil
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
