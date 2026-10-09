![Plasmail logo](frontend/src/assets/images/logo-universal.png)

# Aerion Email Client (_Personal Fork_)

## Overview

_Aerion_ is a cross-platform email client created by [hkdb/aerion](https://github.com/hkdb/aerion).

[This repository](https://github.com/clabs-public-forks/aerion) is a personal fork for custom development.

## Features

![screenshot](docs/ss.png)

- Resource Efficiency - Minimal CPU, RAM, and battery consumption
- Modern UX - Clean, intuitive interface with dark mode support
- Keyboard & Mouse Friendly - Full keyboard navigation with vim-style shortcuts
- Independence - No dependency on Gnome Online Accounts or other system services
- Search That Works - Basic search that actually finds your emails

## Fork Differences

This personal focus has the follow major changes from the upstream Aerion repo:

- Chat-style mail: threads as chat bubbles with quoted history hidden, a docked reply box, and keyboard triage (Done, pin, snooze, mark unread, a Low priority split for newsletters). Pin and snooze are local only. See [Email Management](docs/user-guide/features/email-management.md#chat-mail).
- Collapsible/resizable panes
- Clearer call to action buttons (such as Compose, Reply, etc)
- Several bug fixes

## Installation

- [Installation Guide](docs/user-guide/getting-started/installation/index.md)

## Documentation

- [User Guide](docs/user-guide/intro.md)

## Development

This application was built with [Wails](https://wails.io) + [Svelte](https://svelte.dev/).

### Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md)
