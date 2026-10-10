# Plan: ISSUES.md fix pass

## Scope

Fix the ISSUES.md items that are actionable now: A5, M4, M3, C1, M5, M1 and
M2. Out of scope: S6, M6 and C17 (owner's choice), A4/C12 (architectural),
U1/U3 (blocked on upstream independence), and D2 (blocked on peer ranges).
Work on branch `fix/issues-pass`, with one commit per milestone. Each commit
removes its fixed entry from ISSUES.md. Land the branch after the last
milestone.

## Milestones


### 1. A5: OAuth refresh prompts re-auth only for invalid_grant
- `internal/oauth2/flow.go` `RefreshTokenWithProvider`: if `provider.ClientID == ""`,
  return a new `ErrNotConfigured` sentinel before any HTTP call, worded like
  `StartAuthFlowWithProvider`'s "is not configured (missing client ID)".
  This covers every caller: `RefreshToken`, `app/compose.go`, `app/app.go:1143`
  and the extension auth broker.
- `app/compose.go` `getValidOAuthToken` (~line 90): emit `oauth:reauth-required`
  and say "re-authorization required" only when `errors.Is(err, oauth2.ErrInvalidGrant)`.
  Return other errors as plain refresh failures, logging not-configured at Warn.
- **Acceptance:** an empty ClientID makes no request and returns ErrNotConfigured.
  A network error or 5xx response emits no reauth event. invalid_grant still emits it.
- **Validation:** table-driven test in `internal/oauth2` using an httptest server
  (cases: missing client ID, 500 response, invalid_grant); `go test ./internal/oauth2/ ./app/`.

### 2. M4: one `beginFolderSync` helper for sync-slot bookkeeping
- In `app/sync.go`, add `beginFolderSync(key) (ctx, release func(), ok bool)`.
  It does the busy check and registers the cancel under `syncMu`. `release`
  cancels the context and deletes the key only if this call still owns it,
  using a per-call token: compare a `*struct{}` or a wrapper pointer in place
  of the `fmt.Sprintf("%p")` comparison that `releaseSyncContext` does today.
- Use it in `app/idle_sent.go` `syncSentAfterIdle`, and in `app/background.go`
  `handleIdleNewMail` (~lines 245–290, whose deferred `delete` currently skips
  the ownership check). `SyncFolder` (`app/sync.go:42`) cancels and replaces an
  existing sync rather than skipping. Give the helper a replace mode, or have
  SyncFolder register through the same token type, so all three share the
  release path. Keep `CancelFolderSync`/`CancelAccountSync` behavior as is.
- **Acceptance:** no hand-written `syncContexts[...] = cancel` or unconditional
  `delete(a.syncContexts, …)` remains outside the helper and the Cancel* APIs.
  A replaced sync's release does not remove the newer entry.
- **Validation:** unit test for the helper (busy, replace, then old release
  leaves the new entry in place); `go test -race ./app/`.

### 3. M3: emit `sent:synced` only when Sent changed
- `syncSentAfterIdle`: take a snapshot before the sync with
  `messageStore.GetHighestUID(sent.ID)` (`internal/message/store.go:862`), as
  `internal/sync/newmail.go` does for the inbox. Also snapshot the folder's
  message count or UIDValidity, so a deletion still counts as a change.
  Afterward, emit `sent:synced` only if something differs.
- Still emit `folder:synced` every time, because it clears the progress
  indicator. If that alone reloads chat lists, check the frontend listeners in
  `frontend/src/lib/stores/chat.svelte.ts` and gate the reload there instead.
- **Acceptance:** a no-op Sent sync after inbox flag activity triggers no
  chat-list or thread reload. A new Sent message still triggers one.
- **Validation:** `go test ./app/`; run-aerion: star a message, then confirm in
  the logs or the store that no `sent:synced` reload happens.

### 4. C1: unified-inbox search merges sender chats
- `internal/message/chat_store.go`: factor the sender-chat merge (and the
  Sent-recipients enrichment) out of `SearchChats` into a helper. Add
  `SearchChatsUnifiedInbox(query, offset, limit, filter)`, wrapping
  `SearchConversationsUnifiedInbox`. Merge keys must include the account,
  because sender chats are per account.
- Add the `App.SearchChatsUnifiedInbox` binding in `app/search.go` and switch
  `frontend/src/lib/stores/chat.svelte.ts:255` to it. Keep `SearchUnifiedInbox`
  for the unmounted classic `MessageList` (U1). Run `make generate` with
  Aerion stopped, then revert the `wailsjs/runtime` churn.
- **Acceptance:** threads of a combined sender come back as one result in
  unified search, and same-email senders on two accounts are not merged.
- **Validation:** table-driven store test; `npm run lint && npm run check`;
  run-aerion unified-inbox search.

### 5. M5: one cid matcher for the viewer and compose
- `internal/message/embedded.go`: export `ReferencesCID(html, cid string) bool`,
  backed by `cidSrcPattern`/`embeddedCIDs` (exact ID, `src=` only). Replace
  `quotedHTMLReferencesCID` (`app/compose.go:874`) with it. The #381 guard
  stays: replies still skip parts the body doesn't embed.
- Record the behavior change in the Decision Log: replies no longer re-attach
  a part whose cid only prefix-matches, or appears outside `src=`.
- **Acceptance:** a reply re-attaches exactly the inline parts that
  `FilterEmbeddedInline` hides. `part1` vs `part10` is handled correctly.
- **Validation:** update compose tests for `quotedHTMLReferencesCID` and add
  table cases; `go test ./internal/message/ ./app/`.

### 6. M1: persist an `embedded` flag on attachment rows
- `internal/database/migrations.go` has SQL-only migrations. Add an optional
  `Run func(*sql.Tx) error` to `Migration`, and run it inside `applyMigration`'s
  transaction after the SQL.
- Migration 49: `ALTER TABLE attachments ADD COLUMN embedded INTEGER NOT NULL DEFAULT 0`.
  Its Go backfill sets `embedded` for inline parts with a Content-ID that the
  message's `body_html` embeds (`message.embeddedCIDs`). It then recomputes
  `messages.has_attachments` to `EXISTS (non-embedded attachment)`, limited to
  messages that have inline-cid parts.
- Parse/store path: in `internal/sync/parse.go` (~line 157), mark `Embedded` on
  each attachment and derive `HasAttachments` from it. Add the field to
  `message.Attachment` and the attachment store insert/scan.
- `app/attachment.go` `GetAttachments`: filter with `embedded = 0` (store
  query or slice filter), and drop the `messageStore.Get` body load and
  `HasInlineCID`. Delete `FilterEmbeddedInline` and `HasInlineCID` if no other
  callers remain. Remove the `AttachmentList.svelte` visibility gate only if
  the empty heading no longer flashes while loading; otherwise keep it and
  note why.
- **Acceptance:** after the migration, an older message whose only
  attachments are an embedded signature logo has no paperclip. New messages
  behave the same. GetAttachments no longer loads `body_html`.
- **Validation:** migration test on a seeded DB in `internal/database` (or the
  store tests); `go test ./internal/... ./app/`; run-aerion on a message with
  a signature logo.

### 7. M2: `-race` failure in `fetchMessageBodiesBatch`
- Reproduce with `go test -race -run TestFetchMessageBodiesBatch_IncompleteBodies ./internal/sync/`.
  Read `internal/sync/fetch.go:188` and go-imap v2's `FetchCommand.Next` /
  literal-discard code in the module cache.
- Decide which case applies. If it is our misuse (for example, calling `Next`
  while a literal is unread after a dropped connection), fix the call
  pattern. If it is a go-imap bug, check for a newer go-imap v2 release that
  fixes it and upgrade; if none exists, record it in ISSUES.md with the
  upstream reference. If it is a test artifact, fix the test's fake server.
- **Acceptance:** the race test passes, or the root cause is documented with
  evidence and M2 is rewritten as "confirmed upstream", no longer "unverified".
- **Validation:** `go test -race ./internal/sync/`; finish with `make test` and
  `make lint`.

Critical files: `internal/oauth2/flow.go`, `app/compose.go`, `app/sync.go`, `app/background.go`,
`app/idle_sent.go`, `internal/message/chat_store.go`, `app/search.go`,
`frontend/src/lib/stores/chat.svelte.ts`, `internal/message/embedded.go`,
`internal/database/{migrations,database}.go`, `internal/sync/parse.go`,
`app/attachment.go`, `internal/sync/fetch.go`, `ISSUES.md`, `PLAN.md`.

Before landing:

- Run each milestone's focused tests, then `make test`, `make lint`, and
  `cd frontend && npm run check` before landing.
- Run the run-aerion skill for UI-visible milestones (3, 4, 6). If no display
  is available, say so in Progress.
- Land per AGENTS.md: rebase if `main` moved, then squash-merge into `main`,
  delete the branch once `git diff main fix/issues-pass` is empty, and remove
  PLAN.md once nothing references it.

## Progress

- [ ] 1. A5: OAuth refresh prompts re-auth only for invalid_grant
- [ ] 2. M4: one `beginFolderSync` helper for sync-slot bookkeeping
- [ ] 3. M3: emit `sent:synced` only when Sent changed
- [ ] 4. C1: unified-inbox search merges sender chats
- [ ] 5. M5: one cid matcher for the viewer and compose
- [ ] 6. M1: persist an `embedded` flag on attachment rows
- [ ] 7. M2: `-race` failure in `fetchMessageBodiesBatch`

## Surprises & Discoveries

- Migrations are SQL-only (`Migration{Version, SQL}`), so M1's backfill
  needs an optional Go hook in the migration runner.

## Decision Log

- 2026-10-09: S6 (revocation checks), M6 (Sent IDLE) and C17 (DAV certificate
  follow-ups) stay open in ISSUES.md (owner's choice).
- 2026-10-09: M1 uses a persisted `attachments.embedded` flag with a migration
  backfill, not a one-off has_attachments recompute.
- 2026-10-09: M5 unifies on the viewer's exact `src="cid:"` matching. Replies
  no longer re-attach a part whose cid only prefix-matches or appears outside
  `src=`.

## Outcomes & Retrospective

(Filled in at landing.)
