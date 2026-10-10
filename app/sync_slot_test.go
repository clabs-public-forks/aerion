package app

import (
	"context"
	"testing"
)

func TestBeginFolderSync(t *testing.T) {
	a := &App{ctx: context.Background(), syncContexts: make(map[string]*syncSlot)}
	const key = "acc:inbox"

	ctx1, release1, ok := a.beginFolderSync(key)
	if !ok {
		t.Fatal("first beginFolderSync: want ok")
	}
	if _, _, ok := a.beginFolderSync(key); ok {
		t.Fatal("second beginFolderSync while busy: want !ok")
	}

	// SyncFolder-style replacement cancels the old sync.
	a.syncMu.Lock()
	ctx2, release2 := a.registerFolderSyncLocked(key)
	a.syncMu.Unlock()
	if ctx1.Err() == nil {
		t.Fatal("replaced sync's context not cancelled")
	}

	release1()
	if !a.hasSyncSlot(key) {
		t.Fatal("old release removed the newer entry")
	}
	if ctx2.Err() != nil {
		t.Fatal("old release cancelled the newer context")
	}

	release2()
	if a.hasSyncSlot(key) {
		t.Fatal("release left its own entry in place")
	}
	if ctx2.Err() == nil {
		t.Fatal("release did not cancel its context")
	}
	if _, release3, ok := a.beginFolderSync(key); !ok {
		t.Fatal("beginFolderSync after release: want ok")
	} else {
		release3()
	}
}

func (a *App) hasSyncSlot(key string) bool {
	a.syncMu.Lock()
	defer a.syncMu.Unlock()
	_, ok := a.syncContexts[key]
	return ok
}
