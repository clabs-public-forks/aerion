---
sidebar_position: 2
---

# Calendar

The Calendar extension lets you view, create, and edit calendar events from CalDAV, Google Calendar, and Microsoft 365 in a dedicated view.

Like all [extensions](/docs/features/extensions), it's disabled by default — enable it under **Settings → Extensions**.

:::warning ALPHA
The Calendar extension is currently in ALPHA. Expect rough edges.
:::

## Enabling

1. Go to **Settings → Extensions**
2. Toggle **Calendar** on

A Calendar button is added to the [rail](/docs/features/extensions#the-rail) on the left of the window. The extension has its own settings dialog.

## Manual Sync

To sync calendar events on demand, click the footer at the bottom of the Calendar sidebar.

## Known Issues

- All providers (CalDAV, Google, and Microsoft) rely on the server to send and parse invites. If your provider isn't set up to handle invites server-side, invitations may appear somewhat broken.
