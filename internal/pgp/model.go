// Package pgp provides PGP/MIME signing, verification, encryption, and decryption for email messages
package pgp

import (
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// SignatureStatus represents the PGP verification result
type SignatureStatus string

const (
	StatusNone       SignatureStatus = ""            // Not PGP
	StatusSigned     SignatureStatus = "signed"      // Valid signature, known key
	StatusInvalid    SignatureStatus = "invalid"     // Signature does not verify
	StatusUnknownKey SignatureStatus = "unknown_key" // Valid sig, no matching public key
	StatusExpiredKey SignatureStatus = "expired_key" // Valid sig, expired key
	StatusRevokedKey SignatureStatus = "revoked_key" // Valid sig, revoked key
	// StatusSignerMismatch: valid sig, known key, but no user ID on the key
	// names the message's From address
	StatusSignerMismatch SignatureStatus = "signer_mismatch"
)

// Key represents a user's imported PGP keypair
type Key struct {
	ID           string     `json:"id"`
	AccountID    string     `json:"accountId"`
	Email        string     `json:"email"`
	KeyID        string     `json:"keyId"`       // 16-hex short key ID
	Fingerprint  string     `json:"fingerprint"` // 40-hex full fingerprint
	UserID       string     `json:"userId"`      // "Name <email>" from key
	Algorithm    string     `json:"algorithm"`
	KeySize      int        `json:"keySize"`
	CreatedAtKey *time.Time `json:"createdAtKey,omitempty"`
	ExpiresAtKey *time.Time `json:"expiresAtKey,omitempty"`
	IsDefault    bool       `json:"isDefault"`
	IsExpired    bool       `json:"isExpired"`  // Computed, not stored
	HasPrivate   bool       `json:"hasPrivate"` // Computed, not stored
	CreatedAt    time.Time  `json:"createdAt"`
}

// SenderKey represents a cached public key from a signed message sender or WKD lookup
type SenderKey struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	KeyID        string     `json:"keyId"`
	Fingerprint  string     `json:"fingerprint"`
	UserID       string     `json:"userId"`
	Algorithm    string     `json:"algorithm"`
	KeySize      int        `json:"keySize"`
	CreatedAtKey *time.Time `json:"createdAtKey,omitempty"`
	ExpiresAtKey *time.Time `json:"expiresAtKey,omitempty"`
	Source       string     `json:"source"` // "message", "wkd", "manual"
	CollectedAt  time.Time  `json:"collectedAt"`
	LastSeenAt   time.Time  `json:"lastSeenAt"`
}

// SignatureResult holds the verification result for a message
type SignatureResult struct {
	Status       SignatureStatus `json:"status"`
	SignerEmail  string          `json:"signerEmail"`
	SignerKeyID  string          `json:"signerKeyId"`
	ErrorMessage string          `json:"errorMessage,omitempty"`

	// signer is the verified signing key, used by CheckSender.
	signer *openpgp.Entity
}

// CheckSender downgrades a valid signature to StatusSignerMismatch when no
// user ID on the signing key names from. A signature only proves who
// signed; shown as "signed by" beside a different From it would vouch for
// a forged sender.
func (r *SignatureResult) CheckSender(from string) {
	if r.Status != StatusSigned || r.signer == nil || HasUserIDForEmail(r.signer, from) {
		return
	}
	r.Status = StatusSignerMismatch
	r.ErrorMessage = "signing key does not match the From address"
}

// KeyServer represents an HKP key server entry
type KeyServer struct {
	ID         int    `json:"id"`
	URL        string `json:"url"`
	OrderIndex int    `json:"orderIndex"`
}

// ImportResult holds the result of a PGP key import
type ImportResult struct {
	Key         *Key `json:"key"`
	HasPrivate  bool `json:"hasPrivate"`
	SubkeyCount int  `json:"subkeyCount"`
}
