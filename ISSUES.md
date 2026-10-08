# Issues

## Open

### [P3] No UI for calendar writes that keep failing

Queued calendar writes that hit non-transport errors are retried up to `pendingMaxAttempts` per connectivity restore (`ResetExhausted` on `system:network-online`/`system:wake`), but the user never sees them. A small list with retry/discard actions would surface stuck writes.

