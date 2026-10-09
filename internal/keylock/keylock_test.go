package keylock

import (
	"testing"
	"time"
)

func TestLockSerializesPerKey(t *testing.T) {
	var m Map[string]
	unlockA := m.Lock("a")
	unlockB := m.Lock("b") // a different key is not blocked
	unlockB()

	acquired := make(chan struct{})
	go func() {
		unlock := m.Lock("a")
		close(acquired)
		unlock()
	}()
	select {
	case <-acquired:
		t.Fatal("second Lock of the same key did not wait")
	case <-time.After(50 * time.Millisecond):
	}
	unlockA()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("Lock did not proceed after unlock")
	}
}
