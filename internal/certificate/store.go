package certificate

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store manages trusted certificates in the database and session memory.
// Trust is scoped to a host: a fingerprint accepted for one server never
// vouches for another.
type Store struct {
	db      *sql.DB
	mu      sync.RWMutex
	session map[trustKey]bool // session-only trust
}

type trustKey struct{ host, fingerprint string }

// NewStore creates a new certificate trust store
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		session: make(map[trustKey]bool),
	}
}

func normalizeHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}

// IsTrusted reports whether the certificate fingerprint was accepted for
// host, permanently or for this session. A certificate accepted for an
// account's IMAP host also covers that account's SMTP host, since only IMAP
// connections can prompt for trust.
func (s *Store) IsTrusted(host, fingerprint string) bool {
	host = normalizeHost(host)
	if host == "" {
		return false
	}
	hosts := []string{host}
	rows, err := s.db.Query(
		"SELECT DISTINCT lower(imap_host) FROM accounts WHERE lower(smtp_host) = ? AND lower(imap_host) != ?",
		host, host,
	)
	if err == nil {
		for rows.Next() {
			var h string
			if rows.Scan(&h) == nil {
				hosts = append(hosts, h)
			}
		}
		rows.Close()
	}

	s.mu.RLock()
	for _, h := range hosts {
		if s.session[trustKey{h, fingerprint}] {
			s.mu.RUnlock()
			return true
		}
	}
	s.mu.RUnlock()

	query := "SELECT COUNT(*) FROM trusted_certificates WHERE fingerprint = ? AND host IN (?" +
		strings.Repeat(",?", len(hosts)-1) + ")"
	args := []interface{}{fingerprint}
	for _, h := range hosts {
		args = append(args, h)
	}
	var count int
	if err := s.db.QueryRow(query, args...).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

// AcceptPermanently stores a certificate in the database as trusted for host
func (s *Store) AcceptPermanently(host string, info *CertificateInfo) error {
	host = normalizeHost(host)
	if host == "" {
		return fmt.Errorf("host is required to trust a certificate")
	}
	id := uuid.New().String()
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO trusted_certificates (id, fingerprint, host, subject, issuer, not_before, not_after, accepted_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, info.Fingerprint, host, info.Subject, info.Issuer, info.NotBefore, info.NotAfter, time.Now(),
	)
	return err
}

// AcceptSession trusts a certificate fingerprint for host in session memory only
func (s *Store) AcceptSession(host, fingerprint string) error {
	host = normalizeHost(host)
	if host == "" {
		return fmt.Errorf("host is required to trust a certificate")
	}
	s.mu.Lock()
	s.session[trustKey{host, fingerprint}] = true
	s.mu.Unlock()
	return nil
}

// GetByHosts returns permanently trusted certificates for the given hosts
func (s *Store) GetByHosts(hosts []string) ([]*CertificateInfo, error) {
	if len(hosts) == 0 {
		return nil, nil
	}

	// Build query with placeholders
	query := "SELECT fingerprint, host, subject, issuer, not_before, not_after FROM trusted_certificates WHERE host IN ("
	args := make([]interface{}, len(hosts))
	for i, h := range hosts {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = normalizeHost(h)
	}
	query += ") ORDER BY accepted_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []*CertificateInfo
	for rows.Next() {
		var ci CertificateInfo
		var host string
		if err := rows.Scan(&ci.Fingerprint, &host, &ci.Subject, &ci.Issuer, &ci.NotBefore, &ci.NotAfter); err != nil {
			return nil, err
		}
		certs = append(certs, &ci)
	}
	return certs, rows.Err()
}

// Remove deletes a trusted certificate from the database by fingerprint
func (s *Store) Remove(fingerprint string) error {
	_, err := s.db.Exec("DELETE FROM trusted_certificates WHERE fingerprint = ?", fingerprint)
	if err != nil {
		return err
	}

	// Also remove from session, for every host
	s.mu.Lock()
	for k := range s.session {
		if k.fingerprint == fingerprint {
			delete(s.session, k)
		}
	}
	s.mu.Unlock()

	return nil
}
