# Plan: ISSUES.md fix pass 2

## Scope

Fix the contained low-severity ISSUES.md items M2, M7, M8 and M9. Out of
scope: S6 (owner left it undecided), C17 and M6 (larger, not chosen), A4/C12
(architectural), U1/U3 (blocked on upstream independence), D2 (blocked on peer
ranges). Work on branch `fix/issues-pass-2`, one commit per milestone, each
removing its fixed entry from ISSUES.md. Land the branch after the last one.

## Milestones

### 1. M2: stop consuming a FETCH after a failed literal read
- `internal/sync/fetch.go`: add `readLiteral(lit imap.LiteralReader, limit int64)`
  that reads up to `limit` bytes and reports a read shorter than the literal's
  declared size as `io.ErrUnexpectedEOF`. Use it in `fetchMessageBodiesBatch`.
- After a failed read, every literal reader stops calling `Next`/`Close` on the
  message and the command and closes the raw client instead (the pool's health
  check, or `Discard`, drops the connection):
  - `bulk.go` `fetchHeaderFields`: return the error.
  - `messages.go` `fetchMessageHeaders`: skip the partial message, still
    reconcile threads for the saved ones, and return an error wrapping
    `errStreamBroken` and the read error, so `imap.IsConnectionError` matches
    and the batch retries on a new connection.
  - `header_recovery.go` `recoverFailedHeaderBatch`: return what was recovered
    plus the error.
  - `search.go` `FetchServerMessage` and `fetch.go`'s single-body fetch: return
    the error; discard the pooled connection.
- **Acceptance:** no literal reader in `internal/sync` calls `Next` or `Close`
  on a fetch after a literal read error or short read.
- **Validation:** table-driven test using `scriptedFetchClient` with a literal
  cut off mid-stream for `fetchHeaderFields` and `fetchMessageHeaders`;
  `go test -race ./internal/sync/`.

### 2. M7: read each body once in the embedded-attachment backfill
- `internal/database/backfill.go` `backfillEmbeddedAttachments`: first collect
  inline parts `(id, message_id, content_id)` without the body, grouped by
  message; then read each message's `body_html` once with a prepared statement.
- **Acceptance:** the query that reads `body_html` runs once per message, not
  once per inline part; results are unchanged.
- **Validation:** table-driven test (two inline parts in one message, one
  referenced and one not; a message with no HTML) checking `embedded` and
  `has_attachments`; `go test ./internal/database/`.

### 3. M8: SyncFolder waits for the cancelled slot instead of sleeping
- `app/sync.go`: give `syncSlot` a `done` channel closed once by its release.
  `SyncFolder` cancels the existing slot, drops `syncMu`, waits for `done`
  (capped at a few seconds), relocks and re-checks; it replaces the slot only
  once it is free or the cap has passed.
- **Acceptance:** no fixed sleep in `SyncFolder`; while the old sync is
  winding down, the new one does not start; a slow old sync can't block
  `SyncFolder` forever.
- **Validation:** extend `app/sync_slot_test.go` (wait returns when the old
  slot releases; returns at the cap otherwise); `go test -race ./app/`.

### 4. M9: report Sent changes from the engine instead of snapshots
- `internal/sync`: a per-folder change counter on `Engine`
  (`FolderChangeSeq(folderID) uint64`), bumped where the engine stores or
  removes messages or bodies: header upserts, deletions (by UID, by folder on
  UIDVALIDITY change, older than the sync period), body updates (batch and
  on-demand), server-search creates. Flag-only updates don't bump it.
- `app/idle_sent.go`: compare the counter before and after; emit `sent:synced`
  only if it moved. Remove `folderSnapshot`/`snapshotFolder` and its test.
- **Acceptance:** a sync that only stores bodies emits `sent:synced`; a no-op
  sync does not.
- **Validation:** unit test for the counter; `go test -race ./app/ ./internal/sync/`.

## Progress

- [x] 1. M2: stop consuming a FETCH after a failed literal read
- [ ] 2. M7: read each body once in the embedded-attachment backfill
- [ ] 3. M8: SyncFolder waits for the cancelled slot instead of sleeping
- [ ] 4. M9: report Sent changes from the engine instead of snapshots

## Surprises & Discoveries

- M2: `errStreamBroken`'s text matches no `imap.IsConnectionError` pattern,
  so the header batch retry also checks `errors.Is(err, errStreamBroken)`
  instead of relying on the wrapped read error containing "EOF".

## Decision Log

- M9: a per-folder change counter on the engine, rather than changing the
  return values of `SyncMessages`/`FetchBodiesInBackground` (8 callers). A
  change by another writer between the reads can only cause an extra reload.

## Outcomes & Retrospective
