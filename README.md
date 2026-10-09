![Aerion logo](frontend/src/assets/images/logo-universal.png)

# Aerion Email Client (_Personal Fork_)

## Overview

This is a personal fork of [Aerion](https://github.com/hkdb/aerion), the cross-platform email client by [hkdb](https://github.com/hkdb). It turns Aerion into a Messenger/Chat-style email client: you read and answer conversations the way you would in a chat app, not as a list of messages.

[This repository](https://github.com/clabs-public-forks/aerion) is where that work happens.

![Aerion chat mail view in the Nord dark theme](docs/ss.png)

## Chat Mail

- Threads show as chat bubbles, with quoted history and signatures hidden
- A docked reply box at the bottom of each conversation
- Keyboard triage: Done, pin, snooze, and mark unread
- A Low priority split that keeps newsletters and other bulk mail out of the main list
- Pin, snooze, and sender priority are stored on this computer and are not synced to the server

See [Email Management](docs/user-guide/features/email-management.md#chat-mail) for details.

## Other Changes From Upstream

- Collapsible and resizable panes
- Clearer call-to-action buttons (Compose, Reply, and so on)
- Several bug fixes

## Inherited From Aerion

- Resource efficiency: minimal CPU, RAM, and battery use
- Keyboard and mouse friendly, with vim-style shortcuts
- No dependency on GNOME Online Accounts or other system services
- Search that finds your email

## Relationship to Upstream

This fork tracks upstream Aerion for now, but it is likely to diverge and may eventually stop tracking it. Expect these differences:

- Upstream release binaries do not include these changes. Build from source to use this fork.
- Only the English UI is maintained. Other languages may be incomplete or out of date.
- Report problems with fork features here, not upstream.

## Installation

Build from source with the [Build Guide](docs/BUILD.md). The [Installation Guide](docs/user-guide/getting-started/installation/index.md) covers upstream's release binaries, which do not include this fork's changes.

## Documentation

- [User Guide](docs/user-guide/intro.md)

## Development

This application is built with [Wails](https://wails.io) and [Svelte](https://svelte.dev/).

### Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).
