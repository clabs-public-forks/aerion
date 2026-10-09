package app

import (
	"context"
	"time"

	"github.com/hkdb/aerion/internal/folder"
	"github.com/hkdb/aerion/internal/logging"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// idleSentMinInterval is the minimum gap between Sent syncs triggered by IDLE
// inbox activity for one account, so a stream of flag events (mail read on a
// phone) doesn't start a Sent sync for each one.
const idleSentMinInterval = 30 * time.Second

// scheduleSentSyncAfterIdle runs syncSentAfterIdle at most once per
// idleSentMinInterval per account. Activity inside the interval coalesces into
// one trailing sync at its end, so a reply sent elsewhere is still picked up.
func (a *App) scheduleSentSyncAfterIdle(accountID string) {
	a.idleSentMu.Lock()
	defer a.idleSentMu.Unlock()
	if a.idleSentPending[accountID] {
		return
	}
	a.idleSentPending[accountID] = true
	wait := max(idleSentMinInterval-time.Since(a.idleSentLast[accountID]), 0)
	time.AfterFunc(wait, func() {
		a.idleSentMu.Lock()
		delete(a.idleSentPending, accountID)
		a.idleSentLast[accountID] = time.Now()
		a.idleSentMu.Unlock()
		a.syncSentAfterIdle(accountID)
	})
}

// syncSentAfterIdle syncs the Sent folder after IDLE reports inbox activity.
// IDLE watches INBOX only, so mail sent from another client (e.g. the Gmail
// web UI) would otherwise reach Sent, and the chat thread, only on the next
// scheduled sync. A reply sent elsewhere also flags its inbox original
// \Answered, so the IDLE flag event catches most of them. When a sync of the
// Sent folder is already running, it schedules another attempt rather than
// dropping the activity, since that sync may have listed Sent before the
// reply arrived.
func (a *App) syncSentAfterIdle(accountID string) {
	defer recoverPanic("app.idle", "sync Sent after IDLE")
	log := logging.WithComponent("app.idle")

	sent, err := a.GetSpecialFolder(accountID, folder.TypeSent)
	if err != nil || sent == nil {
		return
	}
	syncKey := accountID + ":" + sent.ID
	a.syncMu.Lock()
	if _, busy := a.syncContexts[syncKey]; busy {
		a.syncMu.Unlock()
		if a.ctx.Err() == nil {
			a.scheduleSentSyncAfterIdle(accountID)
		}
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.syncContexts[syncKey] = cancel
	a.syncMu.Unlock()
	defer func() {
		cancel()
		a.releaseSyncContext(syncKey, cancel)
	}()

	syncPeriodDays := a.syncPeriodDays(accountID)
	// Incremental, like the IDLE inbox sync; the scheduled sync stays the
	// full reconciliation.
	err = a.syncEngine.SyncMessages(ctx, accountID, sent.ID, syncPeriodDays, true)
	if err == nil {
		err = a.syncEngine.FetchBodiesInBackground(ctx, accountID, sent.ID, syncPeriodDays)
	}
	if err != nil && ctx.Err() == nil {
		log.Warn().Err(err).Str("accountID", accountID).Msg("Sent sync after IDLE failed")
	}

	// Cancelled means a manual sync replaced this one; it emits its own
	// completion, and emitting here would clear its progress indicator early.
	if ctx.Err() != nil {
		return
	}

	// Clears the Sent progress indicator and refreshes chat threads.
	wailsRuntime.EventsEmit(a.ctx, "folder:synced", map[string]interface{}{
		"accountId": accountID,
		"folderId":  sent.ID,
	})
	wailsRuntime.EventsEmit(a.ctx, "sent:synced", map[string]interface{}{
		"accountId": accountID,
	})
}
