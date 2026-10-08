package credentials

import "sync"

// refreshKey identifies one account's tokens under one client config.
type refreshKey struct{ accountID, clientConfigID string }

// LockOAuthRefresh serializes token refreshes for one (account, client
// config) slot across every caller in the process — IMAP/SMTP auth and the
// extension auth broker alike. Callers must re-read the tokens after locking
// and skip the refresh if another holder already replaced them: providers
// that rotate refresh tokens reject a refresh token once it has been used.
// The returned function releases the lock.
func (s *Store) LockOAuthRefresh(accountID, clientConfigID string) (unlock func()) {
	mu, _ := s.refreshLocks.LoadOrStore(refreshKey{accountID, clientConfigID}, &sync.Mutex{})
	m := mu.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}
