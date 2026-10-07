---
sidebar_position: 1
slug: /intro
---

# Introduction

Aerion is an open source, lightweight email client built for people who want a modern email experience without the bloat.

## Why Aerion?

The Linux desktop has long needed a proper email client. The existing options each have their drawbacks:

- **Thunderbird** - Clunky interface with too much legacy baggage
- **Geary** - Crippled by its dependency on Gnome Online Accounts, and search is unreliable
- **Mailspring** - Built on Electron, which means heavy resource consumption
- **Evolution** - Feels like it's still stuck in 1999

Aerion was created to fill this gap: a standalone, lightweight email client with a modern interface that just works.

## Design Goals

Aerion is built around five core principles:

1. **Resource Efficiency** - Minimal CPU, RAM, and battery consumption
2. **Modern UX** - Clean, intuitive interface with dark mode support
3. **Keyboard & Mouse Friendly** - Full keyboard navigation with vim-style shortcuts
4. **Independence** - No dependency on Gnome Online Accounts or other system services
5. **Search That Works** - Basic search that actually finds your emails

## Technology

Aerion is built with [Wails](https://wails.io) and [Svelte](https://svelte.dev), not Electron. This means native performance with a modern web-based UI, without the memory overhead of running a full Chromium instance.

Aerion is CASA Tier 2 Certified by Google's preferred [authorized assessor](https://appdefensealliance.dev/casa/casa-assessors): [TAC Security](https://tacsecurity.com/)

## Platform Support

While Linux is a first-class citizen, Aerion also runs on:

- **Linux** - Flatpak or Binary
- **macOS** - Native .app bundle
- **Windows** - Standard installer

## Features at a Glance

- Multiple email accounts with unified inbox
- Conversation threads
- OAuth support for Gmail and Outlook
- CardDAV, Google, and Microsoft contact sync
- WYSIWYG composer with detachable window
- WYSIWYG signature editor
- Basic tracking element removal
- Full-text search
- Comprehensive keyboard shortcuts
- Light and dark themes
- S/MIME and PGP support
- 1st party extension system:
    - Contacts - ALPHA
    - Calendar - ALPHA

## Open Source

Aerion is licensed under Apache 2.0. You can view the source code, report issues, and contribute at [github.com/hkdb/aerion](https://github.com/hkdb/aerion).

#### Sponsorship

[3DF](https://3df.io) is sponsoring by way of dedicating the team's time to work on this. There's otherwise currently no sponsorship. If you like this project, please feel free to give us a star on [Github](https://github.com/hkdb/aerion) or buy us a coffee:

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/yellow_img.png)](https://www.buymeacoffee.com/3dfosi)

#### Issue Contributors

Aerion is largely driven by community feedback. Big thanks to the contributors who submitted issues which led to meaningful improvements we all now enjoy. This project would not be the same without them!

Check out who they are at the Github [repo](https://github.com/hkdb/aerion) README.md under the **Issue Contributors** section.

#### Translation Contributors

Special thanks to translation contributors for making Aerion more accessible.

Check out who they are at the Github [repo](https://github.com/hkdb/aerion) README.md under the **Translation Contributors** section.

## Get Started

Ready to try Aerion? Head to the [Installation Guide](/docs/getting-started/installation) to download and install it on your system.
