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

### Dependencies

- **D2 low** `frontend/package.json`: TypeScript is held at 6.0.x.
  TypeScript 7 is blocked by peer ranges: `typescript-eslint` requires
  `<6.1.0` and `svelte-check` requires `^5 || ^6`. Upgrade once both accept 7.
