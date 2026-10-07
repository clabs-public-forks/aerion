---
name: run-aerion
description: Run, start, launch, and drive the Aerion desktop email client for manual UI verification — start the dev server, open it in Chrome, click through Mail/Contacts/Calendar, read UI or store state, take screenshots, then stop it cleanly. Use when asked to run Aerion, verify a UI change in the running app, or screenshot it.
---

Aerion is a Wails v2 app (Go backend, Svelte 5 frontend). `make dev` serves the
frontend with the Go bindings at `http://localhost:34115`. An agent drives that
URL in Chrome with the Claude-in-Chrome tools. `.claude/skills/run-aerion/dev.sh`
starts, waits for and stops the dev session. All paths are relative to the repo
root.

## Prerequisites

Go 1.25, Node.js 24, the Wails v2 CLI, and GTK 3 / WebKitGTK 4.1 development
libraries (see `AGENTS.md`). The user's machine already has them. Install frontend
dependencies once with `cd frontend && npm ci` if `frontend/node_modules` is
missing.

## Run (agent path)

```bash
.claude/skills/run-aerion/dev.sh start   # make dev in its own session, log in /tmp/aerion-dev.log
.claude/skills/run-aerion/dev.sh wait    # polls http://localhost:34115 for HTTP 200 (first build takes ~1 min)
```

`make dev` also opens a native Aerion window on the user's desktop. Ignore it and
drive the browser tab:

1. `tabs_context_mcp` with `createIfEmpty: true`, then `navigate` that tab to
   `http://localhost:34115`.
2. If an "OAuth credentials missing" notice is showing, click its **OK** button.
   It only shows sometimes, and it can come back once after you dismiss it.
3. Interact with `computer` clicks and keys, and check results with
   `javascript_tool`. Pass `save_to_disk: true` on a `computer` screenshot to
   keep the image file.

Read state in the page with `javascript_tool`. The dev server serves the source
modules, so importing a store returns the same instance the app is using:

```js
const ui = await window.go.app.App.GetUIState()
const kb = await import('/src/lib/stores/keyboard.svelte.ts')
const layout = await import('/src/lib/stores/layout.svelte.ts')
;({ active: ui.activeExtension, collapsed: ui.collapsedSidebars, pane: kb.getFocusedPane(), mode: layout.getLayoutMode(), width: innerWidth })
```

The left rail switches views (Mail, Contacts, Calendar from the top). After a
click, confirm the switch with `(await window.go.app.App.GetUIState()).activeExtension`.
Wait about 1.5 s before reading saved state, because UI-state saves are debounced
by 1 s.

When finished, put back anything you changed (see Gotchas), close the tab, then:

```bash
.claude/skills/run-aerion/dev.sh stop    # kills wails, vite and the app, then reverts frontend/wailsjs/runtime
```

## Static checks

```bash
cd frontend && npm run check && npm run lint && npm run build
make test   # Go tests
```

## Gotchas

- **Dev mode uses the user's real accounts and saved UI state.** It syncs real
  mail. Collapsed sidebars, the active view, widths and the calendar view mode
  are all saved for real. Note the starting `GetUIState()`, restore it before
  stopping, and never open the composer, which could save a draft to the user's
  account.
- **Layout modes come from `matchMedia` on the viewport**: narrow below 768px,
  medium up to 1024px, otherwise full. The `resize_window` tool has no effect
  under the user's window manager. To change modes, ask the user to resize or
  maximize the Chrome window that holds Claude's tab group (it may be minimized
  or behind other windows), then check `innerWidth`. Ignore
  `document.visibilityState`: it can read `hidden` while screenshots still work.
- **Clicks use the screenshot's coordinate frame**, which is reported with every
  screenshot (e.g. 1568×778 for a 2560px-wide viewport), not CSS pixels. In that
  frame, the rail buttons are at about x=15, y=42 / 70 / 100.
- **The key tool can't send `` Ctrl+` ``.** Use the rail buttons. `ctrl+Tab`,
  `ctrl+shift+b`, `alt+h` and similar combinations work.
- **Kit panes (Contacts, Calendar) take DOM focus, mail panes don't.** Read
  `kb.getFocusedPane()` instead of `document.activeElement` to check pane focus.

## Troubleshooting

- **`dev.sh stop` prints "still up"**: the dev processes outlived the kill. List
  them with `ps -o pid,pgid,sid,cmd -p $(pgrep -d, -f "wails dev|node_modules/.bin/vite|aerion-dev-linux")`
  and run `pkill -TERM -s <SID>`. Vite runs in its own process group, so kill by
  session ID, not process group.
- **`git status` shows `frontend/wailsjs/runtime/*` changes after a dev run**:
  `make dev` regenerates those files with different file modes. Run
  `git checkout -- frontend/wailsjs/runtime` (`dev.sh stop` already does).
