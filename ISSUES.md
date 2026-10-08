# Issues

## Open

Found by the full audit on 2026-10-08. Each entry gives a severity, the
location, and the failure. "Unverified" means a reviewer reported it and it
has not yet been confirmed against the code.

### Mail sync and storage

- **M1 high** `internal/sync/folders.go:163-185`: SyncFolders stores the
  STATUS UIDVALIDITY and HIGHESTMODSEQ before SyncMessages compares them, so
  UIDVALIDITY resets and CONDSTORE flag changes are missed. Unverified.
- **M5 medium** `internal/sync/messages.go:70,340-350,447`: header-batch
  recovery can Release a connection that the deferred closure releases
  again. The pool then holds one session twice. Unverified.
- **M7 medium** `internal/sync/fetch.go:577,830`: re-fetching a body inserts
  the attachments again, so they show up twice. Unverified.
- **M2-M4 medium** `internal/imap/pool.go`: the slot check-then-dial window
  can exceed MaxConnections, a waiter that is cancelled can leak a
  connection handed to it, and a dial that finishes after the caller has
  gone is orphaned. Unverified.
- **M6 medium** `internal/message/store.go:1041`: retention deletes by Date
  header but the server SINCE uses INTERNALDATE, so messages with a zero
  Date loop between delete and re-fetch. Unverified.
- **M8/M9 medium** `internal/sync/fetch.go`: partial bodies are stored as
  fetched, and an empty body FETCH deletes the message locally. Unverified.
- **M10 low** `app/draft.go:296-428`: the old IMAP draft is deleted before
  the new one is appended, and UpdateSyncStatus can overwrite a newer edit's
  pending state. Unverified.
- **M11 low** `internal/email/download.go`, `attachment.go`: content
  transfer encoding may be decoded twice. Unverified.
- **M12 low** `internal/platform/singleinstance_linux.go`: the
  single-instance socket lives in /tmp instead of `$XDG_RUNTIME_DIR`.
  Unverified.

### Security

- **S2 medium** `internal/pgp/{store,hkp,wkd}.go`, `app/pgp.go:239-292`: a
  key from WKD or HKP is cached for the requested email without checking
  that one of its user IDs matches. Unverified.
- **S4 medium** `internal/credentials/store.go:62-88`, `oauth.go:241-262`:
  when the keyring write fails and the DB fallback is used, the stale
  keyring entry stays and is read first. Unverified.
- **S6 medium** `internal/smime/verifier.go:344-352`: the chain is validated
  at the signer's own signingTime, with no revocation check. Unverified.
- **S7 medium** `internal/smime/verifier.go:300-311`,
  `internal/pgp/verifier.go:187-206`: the signer identity is never compared
  with From. Unverified.
- **S8 low** `internal/certificate/store.go:26-38`: a trusted certificate
  fingerprint is accepted for any host. Unverified.
- **S9 low** `app/oauth.go:672-682`: TestOAuthConnection gives STARTTLS IMAP
  no ServerName, so the connection test always fails. Unverified.
- **S10 low** `ConversationViewer.svelte:1000-1008`: the print iframe has no
  sandbox. Unverified.
- **S11 low** `internal/pgp/wkd.go:29-34`: the email domain and local part go
  into the WKD URL unvalidated. Unverified.
- **S12 low** `internal/pgp/key.go:102-117`: IsKeyExpired checks a random
  identity. Unverified.
- **S13 low** `internal/oauth2/flow.go:152-210`: the active session and
  callback server have no mutex. The pending OAuth fields in
  `app/oauth.go` and `app/carddav.go` have the same problem (A6).
  Unverified.
- **S14 low** `app/oauth.go:265-280`, `internal/oauth2/discovery.go`: custom
  and discovered OAuth endpoints are not required to be https. Unverified.

### App layer

- **A4 medium** `app/extension_contacts.go:32-52` and contacts backend
  imports: EXT_RULES R1, R2 and R5. The bridge gets a token closure and the
  core DB, and the extension imports `internal/*`. This is architectural
  (see C12).
- **A5 medium** `app/app.go:989-1064`: Shutdown never runs
  `extensionUnregs`, stops the calendar alarm scheduler, or closes the
  extension stores. Unverified.
- **A8 low** `app/attachment.go:251-282`: validateOpenPath allows the whole
  data dir and doesn't resolve symlinks. Unverified.
- **A9 low** `app/compose.go:577`: an invalid mailto URL, including its
  recipients and body, is logged in full. Unverified.

### Contacts

- **C1 high** `internal/carddav/sync.go:420-476`, `store.go:717-790`: full
  sync deletes all records and then upserts. Its errors are only logged, and
  the sync token still advances, so contacts can be lost for good.
  Unverified.
- **C2 high** `internal/carddav/sync.go:55-130`, `scheduler.go:95-125`: no
  per-source guard, so overlapping syncs interleave delete-all and upsert.
  Unverified.
- **C8 medium** `internal/carddav/sync.go:225-245`: any incremental error
  falls back to a destructive full sync, which resets the email ranking
  history. Unverified.
- **C7 medium** `internal/carddav/sync.go:262-310`: errors from delta
  deletes are swallowed and the token advances. Unverified.
- **C3/C4 medium** `internal/carddav/vcard_build.go`: editing drops a
  PHOTO URI, N prefix/suffix/middle names, PREF, extra TYPEs and
  X-ABLabel. Unverified.
- **C5 medium** `internal/carddav/store.go:854-900,1010-1035`: an empty etag
  makes PUT and DELETE unconditional. Unverified.
- **C9 medium** `extensions/contacts/backend/imaging/imaging.go:55-75`:
  photos have no DecodeConfig dimension check before decoding, so a
  decompression bomb can exhaust memory. Unverified.
- **C10 medium** `internal/carddav/client.go:511-560`: CardDAV PHOTO data is
  stored uncapped and unvalidated. Unverified.
- **C11 medium** `ContactEditDialog.svelte:197-215`: a 412 conflict reports
  success and closes the dialog, losing the edits. Unverified.
- **C12 medium** contacts backend: EXT_RULES R1, R2, R5 and R6 (see A4).
  Architectural.
- **C13 low** `internal/contact/google_sync.go:70-75`: syncToken is sent only
  on the first page, and tokens are not URL-escaped. Unverified.
- **C14 low** `contactsView.svelte.ts:61-128`: no request sequencing, so a
  stale response can win. Unverified.
- **C15 low** `ContactDetail.svelte:146-172`: each blocks keyed by email can
  get duplicate keys. Unverified.

### Calendar

- **K1 high** `event_crud.go:530`, `EventDetail.svelte:326`: delete with
  scope this or this-and-future splits at the master start, not at the
  clicked occurrence. Unverified.
- **K2 high** `event_crud.go:287,966`: edit this or this-and-future uses the
  new start as RECURRENCE-ID. Unverified.
- **K3 high** `EventComposerDialog.svelte:358`, `event_crud.go:571,702`:
  editing resets the reminder, rebuilds RRULE from only FREQ, UNTIL and
  COUNT, and drops overrides and X-properties. Unverified.
- **K4 high** `ical_convert.go:128`, `provider_caldav.go:114,188`: a Windows
  TZID fails to parse, so the event is skipped and then deleted locally.
  Unverified.
- **K5 high** `provider_caldav.go:170`, `event_crud.go:926`: alarms are only
  generated for the 7 days after a write or sync. Unverified.
- **K6 high** `provider_microsoft_translate.go:685-775`: recurrence is
  expanded in UTC against a local-zone pattern, giving the wrong weekday
  and DST shifts. Unverified.
- **K7 medium** `store.go:927`, `provider_caldav.go:143-160`: after a force
  resync, events with overrides hit a foreign-key failure. Unverified.
- **K8/K9 medium** `provider_google.go:91,190-205`: instance exceptions and
  cancelled tombstones are ignored, and a full resync after 410 never
  deletes events. Unverified.
- **K10 medium** `TimelineView.svelte:632-651`: drag and resize drop
  transparency, visibility, reminder and HTML. Unverified.
- **K11 medium** `store.go:1246-1266`: stale pending alarms are never
  removed. Unverified.
- **K12 medium** `event_crud.go:708,966,1008,1042`: all-day UNTIL, EXDATE and
  RECURRENCE-ID are written as UTC date-times. Unverified.
- **K13 medium** `rrule_expand.go:77`: FREQ=SECONDLY has no expansion cap.
  Unverified.
- **K14 medium** `rrule_expand.go:61,77`: expansion misses multi-day
  overlaps and moved overrides. Unverified.
- **K15 low** `ical_convert.go`: all-day times are frozen to the timezone at
  parse time, and STATUS:CANCELLED is ignored. Unverified.
- **K16 low** `bridge.go`: R16 says disabled bridge methods should return
  empty results, not errors. `sync.go` uses a package-level sync.Once (R17).
  Escape in the composer is handled on window. Unverified.

### Frontend

- **F1 medium** `Composer.svelte:1229-1266,1506`: Ctrl+Enter has no
  `sending` guard, so the message can be sent twice. Unverified.
- **F2 medium** `Composer.svelte:623-632,1344-1360`: a save that starts while
  another is running is dropped, and Save & Close closes even when the save
  failed. Unverified.
- **F3 low** `Composer.svelte:1363-1367`: Keep Editing doesn't reschedule
  autosave. Unverified.
- **F4 low** `EmailBody.svelte:699-710`: linkify wraps emails inside
  generated links. Unverified.
- **F5 low** kit components: hardcoded English strings (R29). Unverified.
- **F6 low** `AccountSection.svelte:108-114`: the more button has no
  aria-label and no Escape handling. Unverified.
- **F7 low** `ConversationViewer.svelte:470-520`: a stale load's finally
  still clears loading. Unverified.

### Dependencies

- **D1** npm majors outstanding: tiptap 2 to 3, tailwindcss 3 to 4,
  svelte-i18n and any others behind latest. `npm audit --omit=dev` still
  reports moderate and high advisories that only these upgrades fix.
