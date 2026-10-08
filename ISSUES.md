# Issues

## Open

### [P3] Resizable panes can overflow just above the full-layout breakpoint

At ~1025px wide, a 400px sidebar plus a 600px list (Mail, or Contacts' `contacts.list`) leaves the `flex-1` viewer/detail pane little or no room. Mail has always allowed this. A fix would cap list width to the space left after the sidebar and a minimum detail width (e.g. in `PaneResizeHandle` or `getPaneWidth`).

### [P2] Old Flatpak build cache holds OAuth secrets

Before the dev manifest was restricted to packaging inputs, local Flatpak builds copied `.env.local` and `internal/oauth2/credentials_gen.go` into `.flatpak-builder/` (synced folder). Delete it with `rm -rf .flatpak-builder` (it is only a cache), and rotate the OAuth client secrets if the sync service may have uploaded them.

### [P2] Mail-slot token refresh is not shared with the compose path

`app/compose.go` (`getValidOAuthToken` → `refreshOAuthToken`) refreshes mail-slot tokens outside the extension broker's per-slot lock in `internal/extensions/auth`. With providers that rotate refresh tokens (Microsoft, custom OIDC), an extension refresh and an IMAP/SMTP refresh can race and one can send an already-used refresh token. Fix by moving a single-flight "refresh unless the token already changed" into `credentials.Store` or `oauth2.Manager` and calling it from both.

### [P2] All-day RECURRENCE-ID formatted in UTC

`extensions/calendar/backend/provider_caldav_compose.go` `setRecurrenceID` formats all-day dates in UTC, while parsing and `setDateValue` use `configuredTZ()`. Midnight October 7 in Tokyo becomes October 6 UTC, targeting the wrong occurrence. Use the configured timezone for DATE serialization and fix `recurrenceIDMatches`, which compares UTC midnight against locally anchored timestamps.

### [P2] One offline calendar drain exhausts the retry budget

`extensions/calendar/backend/pending_writes.go` `Drain` immediately selects the same failed row until all three attempts are exhausted. Sync invokes drain even after failures, so one offline sync can strand pending writes permanently; restored connectivity does not retry exhausted rows and no recovery UI exists. Defer transport retries across drain cycles and retain a recovery path.

### [P2] Queued calendar writes lose invitation-delivery preferences

`extensions/calendar/backend/pending_writes.go` omits `SendUpdates` from its payload and reconstructed event. Retrying a Google write drops the user's explicit `all`, `externalOnly`, or `none` query parameter. Persist and restore the preference so offline and immediate writes behave consistently.

### [P2] Drafts lose their sender identity

`app/draft.go` `saveDraftToDB` never populates `IdentityID`, and `toComposeMessage` omits `From`. Reopening a draft saved with a nondefault alias selects the default identity; background pending-draft retries serialize an empty sender. Persist the selected identity on create/update and restore the sender for both editing and retries.

### [P2] Unlimited-history sync never reconciles an empty mailbox

`internal/sync/messages.go` `SyncMessages` skips deletion reconciliation when a successful remote search returns zero messages, local rows remain, and `syncPeriodDays == 0`. Emptying a folder in another client leaves stale messages indefinitely. Distinguish search errors from valid empty results and reconcile confirmed empty mailboxes.

### [P2] Untrusted signer certificates become encryption keys

`internal/smime/verifier.go` caches the signer certificate for unknown-CA and self-signed signatures, and `Store.GetSenderCertPEMs` encrypts to the most recently seen cert per email. Anyone can send a validly signed message claiming another address and replace the key used for future encrypted mail to it. Prefer chain-trusted certs for encryption, or require explicit user acceptance of untrusted ones.
