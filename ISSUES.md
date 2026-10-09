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

- **M1 low** `internal/sync/parse.go` `parseMessageBodyInternal`: inline
  parts the HTML body embeds no longer set `has_attachments`, but only for
  bodies parsed from now on. Messages fetched earlier keep a paperclip for
  signature logos the attachment list hides, because bodies are not
  re-fetched. A one-off migration recomputing the flag from stored
  attachments and `body_html` would fix them.
  Deeper fix: persist an `embedded` flag on attachment rows at parse time so
  `has_attachments` and `GetAttachments` share one stored answer, dropping
  the read-time `body_html` load and the `AttachmentList` visibility gate.
- **M3 low** `app/idle_sent.go` `syncSentAfterIdle` emits `folder:synced`
  and `sent:synced` even when the Sent sync stored nothing, so chat lists and
  open threads reload up to every 30 s per account during inbox flag
  activity. Snapshot Sent's highest UID before the sync, as `newmail.go`
  does for the inbox, and emit `sent:synced` only on change. Aerion's own
  read and star actions echo back as IDLE flag events, so ordinary use
  also triggers these Sent syncs.
- **M4 low** `app/sync.go`, `app/background.go`, `app/idle_sent.go`: three
  hand-written copies of the folder-sync slot bookkeeping (busy check,
  cancel registration). The IDLE inbox body fetch in `background.go` deletes
  its key without the ownership check `releaseSyncContext` does. One
  `beginFolderSync` helper with a per-call token would replace them.
- **M5 low** `internal/message/embedded.go` `embeddedCIDs` and
  `app/compose.go` `quotedHTMLReferencesCID` answer the same cid question
  with different matching (whole reference vs. substring), so a reply can
  re-attach a part the viewer hides. Unify them; compose's loose match is
  deliberate, so this changes reply behavior.
- **M6 low** `app/idle_sent.go`: Sent activity from other clients is
  inferred from inbox IDLE events, so a new (non-reply) message sent
  elsewhere reaches Sent and its chat thread only on the next scheduled
  sync. Deeper fix: IDLE on the Sent folder too (a second connection, or a
  folder set in `internal/imap`'s IDLE) routed to the same incremental sync,
  which would also retire the 30 s throttle state; cheaper: a shorter Sent
  poll interval.
- **M2 low, unverified** `internal/sync/fetch.go:188`
  `fetchMessageBodiesBatch`: `go test -race ./internal/sync/` fails in
  `TestFetchMessageBodiesBatch_IncompleteBodies`. `msg.Next()` discards an
  unread literal on the caller's goroutine while go-imap's read goroutine
  reads the same buffer, when the connection drops mid-literal. Decide
  whether it is a go-imap v2 beta bug, our misuse of the streaming API, or
  a test-only artifact; production sees dropped connections too.

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
- **U2** Go sources: `gofmt -l` lists 54 files (for example
  `internal/message/store.go`, `app/compose.go`, `app/account.go`, and many
  under `extensions/` and `internal/`), mostly struct-tag alignment and
  import order. They are left unformatted to keep upstream merges clean;
  run `gofmt -w` across the tree once the fork is independent.
- **U3** non-English locale files: this fork updates English only, so the
  other locales drift. Remove them, or decide to maintain them.

### Dependencies

- **D2 low** `frontend/package.json`: TypeScript is held at 6.0.x.
  TypeScript 7 is blocked by peer ranges: `typescript-eslint` requires
  `<6.1.0` and `svelte-check` requires `^5 || ^6`. Upgrade once both accept 7.
