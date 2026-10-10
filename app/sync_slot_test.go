package app

import (
	"context"
	"testing"
	"time"
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

func TestReplaceFolderSyncWaitsForRelease(t *testing.T) {
	const key = "acc:inbox"
	tests := []struct {
		name        string
		releaseOld  bool // the old sync releases its slot once cancelled
		maxWait     time.Duration
		wantMinWait time.Duration
	}{
		{name: "old sync releases", releaseOld: true, maxWait: 10 * time.Second},
		{name: "old sync hangs", maxWait: 50 * time.Millisecond, wantMinWait: 50 * time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{ctx: context.Background(), syncContexts: make(map[string]*syncSlot)}
			oldCtx, oldRelease, _ := a.beginFolderSync(key)
			released := make(chan struct{})
			go func() {
				<-oldCtx.Done()
				if tt.releaseOld {
					time.Sleep(20 * time.Millisecond) // winding down
					oldRelease()
				}
				close(released)
			}()

			start := time.Now()
			a.syncMu.Lock()
			ctx, release := a.replaceFolderSyncLocked(key, tt.maxWait)
			a.syncMu.Unlock()
			waited := time.Since(start)
			<-released

			if tt.releaseOld && waited >= tt.maxWait {
				t.Fatalf("waited %v: want return on release, before the %v cap", waited, tt.maxWait)
			}
			if waited < tt.wantMinWait {
				t.Fatalf("waited %v: want at least %v", waited, tt.wantMinWait)
			}

			// A late release by the old sync must not remove the new slot.
			oldRelease()
			if !a.hasSyncSlot(key) || ctx.Err() != nil {
				t.Fatal("old release disturbed the new sync")
			}
			release()
			if a.hasSyncSlot(key) {
				t.Fatal("release left its own entry in place")
			}
		})
	}
}
