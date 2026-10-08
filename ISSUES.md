# Issues

## Open

### [P3] Resizable panes can overflow just above the full-layout breakpoint

At ~1025px wide, a 400px sidebar plus a 600px list (Mail, or Contacts' `contacts.list`) leaves the `flex-1` viewer/detail pane little or no room. Mail has always allowed this. A fix would cap list width to the space left after the sidebar and a minimum detail width (e.g. in `PaneResizeHandle` or `getPaneWidth`).

### [P3] No UI for calendar writes that keep failing

Queued calendar writes that hit non-transport errors are retried up to `pendingMaxAttempts` per connectivity restore (`ResetExhausted` on `system:network-online`/`system:wake`), but the user never sees them. A small list with retry/discard actions would surface stuck writes.

