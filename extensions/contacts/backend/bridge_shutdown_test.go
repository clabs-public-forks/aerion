package backend

import (
	"errors"
	"testing"
)

func TestContactsBridgeShutdown(t *testing.T) {
	t.Run("before init blocks later init", func(t *testing.T) {
		b := &ContactsBridge{}
		b.shutdown()
		if err := b.ensureInit(); !errors.Is(err, errShuttingDown) {
			t.Fatalf("ensureInit after shutdown = %v, want errShuttingDown", err)
		}
	})

	t.Run("after init closes store", func(t *testing.T) {
		store, err := NewStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		b := &ContactsBridge{api: &API{extStore: store}}
		b.initOnce.Do(func() {})

		b.shutdown()

		if _, err := store.GetETag("x"); err == nil {
			t.Error("store still open after shutdown")
		}
	})
}
