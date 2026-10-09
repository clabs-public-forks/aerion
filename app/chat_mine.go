package app

import (
	"strings"

	"github.com/hkdb/aerion/internal/folder"
	"github.com/hkdb/aerion/internal/logging"
	"github.com/hkdb/aerion/internal/message"
)

// chatOwner recognizes my own messages for chat bubbles: any account address
// as sender, or any message in a Sent folder (covers identities and aliases
// sent from that account).
type chatOwner struct {
	emails        map[string]struct{}
	sentFolderIDs map[string]struct{}
}

func (o chatOwner) mine(m *message.Message) bool {
	if _, ok := o.sentFolderIDs[m.FolderID]; ok {
		return true
	}
	_, ok := o.emails[strings.ToLower(strings.TrimSpace(m.FromEmail))]
	return ok
}

// loadChatOwner gathers every account's address and Sent folders. Errors
// are logged and leave the sets partial, so messages default to not mine.
func (a *App) loadChatOwner() chatOwner {
	o := a.loadChatOwnerEmails()
	ids, err := a.folderStore.ListIDsByType(folder.TypeSent)
	if err != nil {
		log := logging.WithComponent("app")
		log.Warn().Err(err).Msg("Chat ownership: failed to list Sent folders")
	}
	for _, id := range ids {
		o.sentFolderIDs[id] = struct{}{}
	}
	return o
}

// loadChatOwnerEmails returns an owner holding every account's address.
func (a *App) loadChatOwnerEmails() chatOwner {
	o := chatOwner{emails: map[string]struct{}{}, sentFolderIDs: map[string]struct{}{}}
	accounts, err := a.accountStore.List()
	if err != nil {
		log := logging.WithComponent("app")
		log.Warn().Err(err).Msg("Chat ownership: failed to list accounts")
		return o
	}
	for _, acc := range accounts {
		if acc.Email != "" {
			o.emails[strings.ToLower(strings.TrimSpace(acc.Email))] = struct{}{}
		}
	}
	return o
}

// attachChatFields sets the computed chat fields (Chat, Mine) on m.
func attachChatFields(m *message.Message, o chatOwner) {
	if m == nil {
		return
	}
	attachChatText(m)
	m.Mine = o.mine(m)
}
