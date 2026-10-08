package backend

import (
	"errors"
	"testing"
)

func TestCalendarBridgeShutdown(t *testing.T) {
	t.Run("before init blocks later init", func(t *testing.T) {
		b := &CalendarBridge{}
		b.shutdown()
		if err := b.ensureInit(); !errors.Is(err, errShuttingDown) {
			t.Fatalf("ensureInit after shutdown = %v, want errShuttingDown", err)
		}
	})

	t.Run("after init stops work and closes store", func(t *testing.T) {
		store, err := NewStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		syncStopped := false
		b := &CalendarBridge{
			api:      &API{store: store},
			stopSync: func() { syncStopped = true },
			alarms:   NewAlarmScheduler(store, nil, nil, nil),
		}
		b.initOnce.Do(func() {})

		b.shutdown()

		if !syncStopped {
			t.Error("syncer not stopped")
		}
		if _, err := store.ListSources(); err == nil {
			t.Error("store still open after shutdown")
		}
	})
}
