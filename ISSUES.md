# Issues

## Open

### Old Flatpak build cache holds OAuth secrets

Before the dev manifest was restricted to packaging inputs, local Flatpak builds copied `.env.local` and `internal/oauth2/credentials_gen.go` into `.flatpak-builder/` (synced folder). Delete it with `rm -rf .flatpak-builder` (it is only a cache), and rotate the OAuth client secrets if the sync service may have uploaded them.

### Mail-slot token refresh is not shared with the compose path

`app/compose.go` (`getValidOAuthToken` → `refreshOAuthToken`) refreshes mail-slot tokens outside the extension broker's per-slot lock in `internal/extensions/auth`. With providers that rotate refresh tokens (Microsoft, custom OIDC), an extension refresh and an IMAP/SMTP refresh can race and one can send an already-used refresh token. Fix by moving a single-flight "refresh unless the token already changed" into `credentials.Store` or `oauth2.Manager` and calling it from both.

### All-day RECURRENCE-ID formatted in UTC

`extensions/calendar/backend/provider_caldav_compose.go` `setRecurrenceID` formats all-day RECURRENCE-ID dates in UTC, while `setDateValue` uses the configured display tz. West of UTC an override can name the previous day. Verify, then use `configuredTZ()`.
