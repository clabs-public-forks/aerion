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
- **C14 low** `contactsView.svelte.ts:61-128`: no request sequencing, so a
  stale response can win. Unverified.
- **C15 low** `ContactDetail.svelte:146-172`: each blocks keyed by email can
  get duplicate keys. Unverified.

### Calendar

- **K15 low** `ical_convert.go`: all-day times are frozen to the timezone at
  parse time, and STATUS:CANCELLED is ignored. Unverified.
- **K16 low** `bridge.go`: R16 says disabled bridge methods should return
  empty results, not errors. `sync.go` uses a package-level sync.Once (R17).
  Escape in the composer is handled on window. Unverified.

### Frontend

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
