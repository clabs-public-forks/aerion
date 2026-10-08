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

The toolchain is installed. Run `cd frontend && npm ci` first if
`frontend/node_modules` is missing.

## Run (agent path)

```bash
.claude/skills/run-aerion/dev.sh start   # make dev in its own session; prints the log path
.claude/skills/run-aerion/dev.sh wait    # polls http://localhost:34115 for HTTP 200 (first build takes ~1 min); fails early if make dev exits
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
modules, so importing a store returns the same instance the app is using.
After a hot reload the app may hold the module under a `?t=<timestamp>` URL,
and a plain import then gives a separate copy. Import the exact URL listed in
`performance.getEntriesByType('resource')` instead:

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
.claude/skills/run-aerion/dev.sh stop    # kills wails, vite and the app, then reverts mode-only changes in frontend/wailsjs/runtime
```

## Gotchas

- **Dev mode uses the user's real accounts and saved UI state.** It syncs real
  mail. Collapsed sidebars, the active view and widths are saved for real. Note
  the starting `GetUIState()` and restore it before stopping. The calendar
  view mode is not saved; it resets to Month on reload. Never open the
  composer, which could save a draft to the user's account, and avoid opening
  real messages, which marks them read.
- **Fake data instead of real data.** Bindings are looked up on
  `window.go.app.App` at call time, so a page script can replace one (for
  example a calendar or contacts list call) to return made-up records. Make the
  write bindings throw while doing this, so a stray click can't change real
  data. Call the store's reload (or switch views) to pick up the fake data,
  and reload the page to undo it. To check a mail layout without opening a
  message, add a copy of the markup to the page and measure it.
- **Layout modes come from `matchMedia` on the viewport**: narrow below 768px,
  medium up to 1024px, otherwise full. The `resize_window` tool has no effect
  under the user's window manager. To change modes, ask the user to resize or
  maximize the Chrome window that holds Claude's tab group (it may be minimized
  or behind other windows), then check `innerWidth`. To test a narrow layout
  without a resize, set a narrower width on the pane's element from
  `javascript_tool`. That checks how the pane's own content wraps, but it
  doesn't switch the app into its narrow layout mode.
- **The tab often reports `document.visibilityState` as `hidden`.** Screenshots
  can still work, but `requestAnimationFrame` never fires and timers are
  throttled. Never `await` an animation frame in `javascript_tool`: it hangs
  and Chrome reports the renderer as frozen. Pauses or freezes seen only in
  this state come from the background tab, not from the app.
- **The first screenshot after a hot reload often times out.** Wait a few
  seconds and retry. If it keeps failing, check layout with
  `getBoundingClientRect()` or `getClientRects()` from `javascript_tool`
  instead.
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
  `make dev` regenerates those files with different file modes. `dev.sh stop`
  reverts them when only modes changed and warns if their content changed.
