# Issues

## Open

### [P3] Resizable panes can overflow just above the full-layout breakpoint

At ~1025px wide, a 400px sidebar plus a 600px list (Mail, or Contacts' `contacts.list`) leaves the `flex-1` viewer/detail pane little or no room. Mail has always allowed this. A fix would cap list width to the space left after the sidebar and a minimum detail width (e.g. in `PaneResizeHandle` or `getPaneWidth`).

### [P3] No UI for calendar writes that keep failing

Queued calendar writes that hit non-transport errors are retried up to `pendingMaxAttempts` per connectivity restore (`ResetExhausted` on `system:network-online`/`system:wake`), but the user never sees them. A small list with retry/discard actions would surface stuck writes.

### [P2] Old Flatpak build cache holds OAuth secrets

Before the dev manifest was restricted to packaging inputs, local Flatpak builds copied `.env.local` and `internal/oauth2/credentials_gen.go` into `.flatpak-builder/` (synced folder). Delete it with `rm -rf .flatpak-builder` (it is only a cache), and rotate the OAuth client secrets if the sync service may have uploaded them.

### [P2] Mail-slot token refresh is not shared with the compose path

`app/compose.go` (`getValidOAuthToken` → `refreshOAuthToken`) refreshes mail-slot tokens outside the extension broker's per-slot lock in `internal/extensions/auth`. With providers that rotate refresh tokens (Microsoft, custom OIDC), an extension refresh and an IMAP/SMTP refresh can race and one can send an already-used refresh token. Fix by moving a single-flight "refresh unless the token already changed" into `credentials.Store` or `oauth2.Manager` and calling it from both.


