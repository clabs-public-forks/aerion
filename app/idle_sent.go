package app

import (
	"fmt"
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
	ctx, release, ok := a.beginFolderSync(syncKey)
	if !ok {
		if a.ctx.Err() == nil {
			a.scheduleSentSyncAfterIdle(accountID)
		}
		return
	}
	defer release()

	before, snapErr := a.snapshotFolder(sent.ID)
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
	// sent:synced reloads chat lists and open threads, so skip it when the
	// sync stored nothing new; IDLE flag activity triggers many no-op syncs.
	after, err := a.snapshotFolder(sent.ID)
	if snapErr == nil && err == nil && after == before {
		log.Debug().Str("accountID", accountID).Msg("Sent unchanged after IDLE sync; skipping sent:synced")
		return
	}
	wailsRuntime.EventsEmit(a.ctx, "sent:synced", map[string]interface{}{
		"accountId": accountID,
	})
}

// folderSnapshot records what a sync can change in a folder's stored
// messages: a new message raises the highest UID, a deletion lowers the
// count, and a UIDVALIDITY change re-stores the mailbox.
type folderSnapshot struct {
	highestUID  uint32
	count       int
	uidValidity uint32
}

func (a *App) snapshotFolder(folderID string) (folderSnapshot, error) {
	f, err := a.folderStore.Get(folderID)
	if err != nil {
		return folderSnapshot{}, err
	}
	if f == nil {
		return folderSnapshot{}, fmt.Errorf("folder %s not found", folderID)
	}
	uid, err := a.messageStore.GetHighestUID(folderID)
	if err != nil {
		return folderSnapshot{}, err
	}
	count, err := a.messageStore.CountByFolder(folderID)
	if err != nil {
		return folderSnapshot{}, err
	}
	return folderSnapshot{highestUID: uid, count: count, uidValidity: f.UIDValidity}, nil
}
