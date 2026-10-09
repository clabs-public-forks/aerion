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

- **C16 low** `extensions/calendar/backend/alarm.go` `RefreshAllAlarms` and
  `alarm_scheduler.go`: the alarm refresh scales poorly with large
  calendars. It runs one `ListOverrides` query per event. It recomputes every
  event on each `calendar:sync-complete`, which `sync.go` publishes once per
  source, so syncing N sources does N full refreshes. `Start` runs the first
  refresh synchronously inside `ensureInit`, which blocks the first calendar
  call after launch. Fixing it needs batched override loading, per-source
  refresh, and an async initial pass.
- **C17 low** DAV TLS trust (follow-up to S8): CardDAV and CalDAV servers
  with self-signed certificates now need trust pinned to that exact host,
  and the DAV setup flow has no certificate-accept prompt like the mail
  account flow does. Such servers can't be added from the UI.

### Chat mail

Deferred from the chat mail feature (`feat/chat-mail`, M9 review).

- **CM1 low** `app/background.go` `handleNewMailNotification`: sync reports
  only a new-mail count, so priority-only notifications classify the `count`
  highest-UID inbox rows. A sync that also stores older mail with higher
  UIDs (moves, UIDVALIDITY reset) can misclassify that window. The fix is
  for `sync.NewMailInfo` to carry the new message IDs.
- **CM2 low** frontend: the inbox lookup (`MessageList`, `Sidebar`), the
  sync toggle and trash-with-undo exist as unshared copies in the classic
  and chat components. Consolidate them, or delete the classic copies once
  the classic components leave the tree.
- **CM3 low** chat views infer `isMine` from account emails and Sent folder
  ids in the frontend; a backend `mine` flag set beside `attachChatText`
  would give one source of truth.
- **CM4 low** `internal/message/chattext.go` `ExtractChatText` reruns on
  every conversation reload (~4 ms and 2 MB per 165 KB newsletter body, per
  `BenchmarkExtractChatTextNewsletter`). Cache it per message if long
  threads feel slow.
- **CM5 low** chat search results in Sent Mail name me instead of the
  recipients: `ConversationSearchResult` (upstream `store.go`) has no
  `recipients`, so the Sent-row fallback in `chatPeople` covers chats only.
  Add recipients to search results, or reuse the chat recipients query.
- **CM6 low** `viewer/AttachmentList.svelte` (chat and classic): Outlook
  signature images and other `Content-Disposition: inline` parts show as
  attachment cards, about a dozen per message on one real thread. Hiding
  parts that the HTML body references by `cid:` would cut the noise in both
  views.
- **CM7 low** `app/background.go` IDLE handler: an IDLE push syncs INBOX
  only, so mail sent from another client (e.g. the Gmail web UI) reaches
  Sent, and the chat thread, only on the next scheduled sync (30 min by
  default) or a manual sync. Syncing Sent after an IDLE-triggered INBOX sync,
  or a shorter Sent interval, would show replies from other clients sooner.
- **CM8 low** `internal/smtp/client.go`: `net/smtp` greets with
  `EHLO localhost` because `Hello` is never called. Gmail recorded it in
  `Received`, and it is a mild spam signal. Sending an address literal
  (`[ip]`, as Thunderbird does) avoids leaking the hostname.
- **CM9 low** `internal/message/chat_store.go` `CountChats` wraps the whole
  grouped chat query in `SELECT COUNT(*)`, so every list load runs the
  aggregation two or three times (page, count, Low count). Fine at current
  mailbox sizes; a lighter count query or one query returning both would
  halve the work if large inboxes feel slow.
- **CM10 low** `internal/smtp/message.go`: the Message-ID now uses the
  sender's domain, and `domainFromEmail` falls back to `localhost` for an
  address without exactly one `@`. Compose validates From, so this is
  unreachable today; keep `aerion` as the fallback if that changes.

### Dependencies

- **D2 low** `frontend/package.json`: TypeScript is held at 6.0.x.
  TypeScript 7 is blocked by peer ranges: `typescript-eslint` requires
  `<6.1.0` and `svelte-check` requires `^5 || ^6`. Upgrade once both accept 7.
