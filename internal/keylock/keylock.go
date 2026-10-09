// Package keylock provides a mutex per key.
package keylock

import "sync"

// Map holds one mutex per key. The zero value is ready to use.
type Map[K comparable] struct {
	locks sync.Map // K -> *sync.Mutex
}

// Lock locks the mutex for key and returns the function that unlocks it.
func (m *Map[K]) Lock(key K) (unlock func()) {
	mu, _ := m.locks.LoadOrStore(key, &sync.Mutex{})
	l := mu.(*sync.Mutex)
	l.Lock()
	return l.Unlock
}

// Forget drops key's mutex once the key will no longer be locked.
func (m *Map[K]) Forget(key K) {
	m.locks.Delete(key)
}
