# Issues

## Open

### [P2] Old Flatpak build cache holds OAuth secrets

Before the dev manifest was restricted to packaging inputs, local Flatpak builds copied `.env.local` and `internal/oauth2/credentials_gen.go` into `.flatpak-builder/` (synced folder). Delete it with `rm -rf .flatpak-builder` (it is only a cache), and rotate the OAuth client secrets if the sync service may have uploaded them.

### [P2] Mail-slot token refresh is not shared with the compose path

`app/compose.go` (`getValidOAuthToken` → `refreshOAuthToken`) refreshes mail-slot tokens outside the extension broker's per-slot lock in `internal/extensions/auth`. With providers that rotate refresh tokens (Microsoft, custom OIDC), an extension refresh and an IMAP/SMTP refresh can race and one can send an already-used refresh token. Fix by moving a single-flight "refresh unless the token already changed" into `credentials.Store` or `oauth2.Manager` and calling it from both.

### [P2] All-day RECURRENCE-ID formatted in UTC

`extensions/calendar/backend/provider_caldav_compose.go` `setRecurrenceID` formats all-day dates in UTC, while parsing and `setDateValue` use `configuredTZ()`. Midnight October 7 in Tokyo becomes October 6 UTC, targeting the wrong occurrence. Use the configured timezone for DATE serialization and fix `recurrenceIDMatches`, which compares UTC midnight against locally anchored timestamps.

### [P1] SMTP silently falls back from STARTTLS to plaintext

`internal/smtp/client.go` `Connect` only warns when requested STARTTLS is unavailable, then succeeds. LOGIN and XOAUTH2 do not enforce TLS, exposing passwords or bearer tokens if STARTTLS is stripped or unsupported. Fail connection setup when configured STARTTLS cannot be negotiated.

### [P1] S/MIME trusted status lacks certificate-chain verification

`internal/smime/verifier.go` `verifyPKCS7` calls `p7.Verify()`, which disables certificate-chain verification in the installed library. A valid signature from a leaf issued by an attacker's untrusted CA receives normal signed status because the leaf is not self-signed. Verify against trusted roots before reporting a trusted signature.

### [P1] S/MIME identity and certificate caching can select a nonsigner

`internal/smime/verifier.go` `extractSignerInfo` selects the first embedded certificate with an email address; `cacheSenderCert` selects the first leaf. Neither resolves the certificate referenced by SignerInfo. An unrelated embedded certificate can supply the displayed identity or cached key. Use the actual signer certificate consistently.

### [P1] Attachment-only messages lose their attachments

`internal/smtp/message.go` `ToRFC822` selects multipart serialization only when attachments accompany a nonempty body. An attachment-only message in plain-text mode is sent as an empty message without its files. Serialize attachments regardless of body content.

### [P1] Sequential offline calendar writes retain stale transport state

`extensions/calendar/backend/pending_writes.go` replays each saved ETag and provider ID unchanged. Two offline edits retain the same ETag: the first succeeds, the second conflicts and is discarded, losing the latest edit. An offline create followed by an edit can create a duplicate because the provider ID remains empty. Advance dependent queued operations after success or coalesce them while preserving conflict detection.

### [P2] One offline calendar drain exhausts the retry budget

`extensions/calendar/backend/pending_writes.go` `Drain` immediately selects the same failed row until all three attempts are exhausted. Sync invokes drain even after failures, so one offline sync can strand pending writes permanently; restored connectivity does not retry exhausted rows and no recovery UI exists. Defer transport retries across drain cycles and retain a recovery path.

### [P2] Queued calendar writes lose invitation-delivery preferences

`extensions/calendar/backend/pending_writes.go` omits `SendUpdates` from its payload and reconstructed event. Retrying a Google write drops the user's explicit `all`, `externalOnly`, or `none` query parameter. Persist and restore the preference so offline and immediate writes behave consistently.

### [P2] Drafts lose their sender identity

`app/draft.go` `saveDraftToDB` never populates `IdentityID`, and `toComposeMessage` omits `From`. Reopening a draft saved with a nondefault alias selects the default identity; background pending-draft retries serialize an empty sender. Persist the selected identity on create/update and restore the sender for both editing and retries.

### [P2] Undoing a move pushes another undo command

`app/undo.go` `MoveMessagesToFolder` delegates to `MoveToFolder`, which pushes a new move command. Repeated undo toggles the same message between folders instead of reaching earlier user actions. Suppress undo recording while executing an undo.

### [P2] Unhealthy IMAP connections continue consuming pool capacity

`internal/imap/pool.go` `Release` logs that an unhealthy connection is discarded but leaves it tracked. Acquisition skips it while counting it toward the limit; a pool full of dead connections causes waits and timeouts until idle cleanup. Remove unhealthy connections immediately and let waiting requests obtain replacements.

### [P2] Unlimited-history sync never reconciles an empty mailbox

`internal/sync/messages.go` `SyncMessages` skips deletion reconciliation when a successful remote search returns zero messages, local rows remain, and `syncPeriodDays == 0`. Emptying a folder in another client leaves stale messages indefinitely. Distinguish search errors from valid empty results and reconcile confirmed empty mailboxes.

### [P3] Narrow-layout sidebar toggle not verified manually

The collapsible-sidebar feature changed the narrow (<768px) toggle path: the toolbar button and `Ctrl+Shift+B` now open and close the slide-in overlay through `toggleActiveSidebar()`. This was checked only statically, because the test window could not be resized. In a narrow window, confirm in Mail, Contacts, and Calendar that the overlay opens, closes, and that the scrim and back button still dismiss it.
