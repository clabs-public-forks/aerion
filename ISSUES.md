# Issues

## Open

Found by the full audit on 2026-10-08. Each entry gives a severity, the
location, and the failure. "Unverified" means a reviewer reported it and it
has not yet been confirmed against the code.

### Security

- **S6 medium** `internal/smime/verifier.go` `verifyChain`: there is no
  revocation check, so a revoked signer still shows as trusted. Needs an
  owner decision: OCSP or CRL fetches on every view tell the CA what is
  being read. (The signing-time half of S6 is fixed.)

### App layer

- **A4 medium** `app/extension_contacts.go:32-52` and contacts backend
  imports: EXT_RULES R1, R2 and R5. The bridge gets a token closure and the
  core DB, and the extension imports `internal/*`. This is architectural
  (see C12).

### Contacts

- **C12 medium** contacts backend: EXT_RULES R1, R2, R5 and R6 (see A4).
  Architectural.

### Calendar

- **C17 low** DAV certificate follow-ups: the setup dialogs now offer the
  certificate-accept prompt, but a certificate that changes on an existing
  CardDAV/CalDAV source's background sync only fails the sync (mail sync
  prompts), and certificates accepted permanently for DAV hosts don't show in
  the account Security tab's trusted-certificate list, which lists mail hosts
  only. Deeper fix: stamp the host on `certificate.Error` in
  `certificate.Transport` and return it structurally from the DAV bindings,
  replacing the frontend's error-string regex and the second `Probe` request
  (which also misses certificates only discovery redirects reach).

### Mail sync

- **M6 low** `app/idle_sent.go`: Sent activity from other clients is
  inferred from inbox IDLE events, so a new (non-reply) message sent
  elsewhere reaches Sent and its chat thread only on the next scheduled
  sync. Deeper fix: IDLE on the Sent folder too (a second connection, or a
  folder set in `internal/imap`'s IDLE) routed to the same incremental sync,
  which would also retire the 30 s throttle state; cheaper: a shorter Sent
  poll interval.
- **M2 low, confirmed upstream** go-imap v2 `imapclient.fetchLiteralReader`
  (beta.8 and HEAD c328f5e): when a FETCH literal read fails early (dropped
  connection, read timeout), it releases the read goroutine while its inner
  `LimitReader` still has bytes left. The next `Next`/`Close` then discards
  them from the shared `bufio.Reader`, so the two goroutines race on it.
  `fetchMessageBodiesBatch` now stops consuming after a failed literal. The
  other literal readers in `internal/sync` (`bulk.go`, `header_recovery.go`,
  `messages.go`, `search.go`, and `fetch.go`'s single-body fetch) still race.
  Fix: apply the same stop there, or report the bug upstream and upgrade
  once it is fixed.
- **M7 low** `internal/database/backfill.go` `backfillEmbeddedAttachments`
  (migration 49): the query joins `body_html` onto every inline attachment
  row, so a message with several inline parts has its whole body read once
  per part, all inside the migration transaction. A large mailbox can stall
  the first startup after upgrading. Fix: select the distinct messages with
  inline parts first, read each body once, then match its parts.
- **M8 low** `app/sync.go` `SyncFolder`: after cancelling an existing slot it
  drops `syncMu` for a 100 ms sleep. In that window the cancelled slot still
  looks busy, so `beginFolderSync` (IDLE body fetch) skips its work, and a
  caller that claims the slot is then cancelled by `registerFolderSyncLocked`.
  Present before the issues pass. Fix: wait for the cancelled slot's release
  instead of sleeping, or re-check the slot after relocking.
- **M9 low** `app/idle_sent.go` `syncSentAfterIdle`: whether to emit
  `sent:synced` is guessed by comparing the folder's highest UID, message
  count and UIDVALIDITY before and after the sync. A sync that only fetches
  bodies for messages already stored changes none of these, and another
  writer (a manual `SyncFolder`) can change Sent between the snapshots;
  either can leave chat views stale until the next scheduled sync. Fix:
  have `SyncMessages` and `FetchBodiesInBackground` report whether they
  stored or removed anything, and drop `folderSnapshot`.

### Upstream independence

This fork may eventually stop tracking upstream Aerion. Until then, some code
is kept or left untouched only to keep upstream merges clean. Record each such
cleanup here, so it can be done once the fork is independent.

- **U1** frontend: delete the unmounted classic `MessageList`,
  `ConversationRow` and `ConversationViewer`, kept only to avoid merge
  conflicts. They hold unshared copies of chat-component logic (the inbox
  lookup, the sync toggle, trash-with-undo) that go with them; the chat
  side uses `accountStore.getFolder`/`getInbox` and
  `components/chat/chatTriage.ts`.
- **U3** non-English locale files: this fork updates English only, so the
  other locales drift. Remove them, or decide to maintain them.

### Dependencies

- **D2 low** `frontend/package.json`: TypeScript is held at 6.0.x.
  TypeScript 7 is blocked by peer ranges: `typescript-eslint` requires
  `<6.1.0` and `svelte-check` requires `^5 || ^6`. Upgrade once both accept 7.
