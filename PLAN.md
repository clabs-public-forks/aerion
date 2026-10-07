# Plan: Collapsible left sidebar (Mail, Contacts, Calendar)

## Context

The left sidebar (mail accounts/folders, contact sources, calendar subscriptions) is always visible in full and medium layouts. The user wants to hide it with a toggle to free horizontal space. Today only the narrow (<768px) layout can hide it, using a slide-in overlay opened by `ResponsiveSidebarToggle` / mail's `showFolderToggle` button.

Decisions:
- **Per-view state:** Mail, Contacts, and Calendar each store their own collapsed flag, persisted across restarts.
- **Fully hidden:** a collapsed sidebar takes zero width. The toggle stays at the leading edge of each view's list/calendar toolbar.
- **Shortcut:** `Ctrl/Cmd+Shift+B` toggles the active view's sidebar. It is a global shortcut and is inert while the composer is open.
- **Narrow layout:** unchanged. The toggle still opens the slide-in overlay; the collapsed flag applies only in full and medium layouts.

## Scope

In scope: the collapse toggle for all three views, persistence, the shortcut, keyboard pane-focus behavior while collapsed, English locale strings, and docs.
Out of scope: a thin or icon-strip collapsed mode, animations beyond the existing CSS, and non-English locales.

## Design

### 1. Persistence
- `internal/appstate/model.go`: add `CollapsedSidebars map[string]bool \`json:"collapsedSidebars,omitempty"\`` to `UIState`, keyed by rail id (`mail`, `contacts`, `calendar`; works for future extensions). Missing means expanded.
- `internal/appstate/store.go` `GetUIState`: initialize the map in both default branches and in the nil guard, next to `ExpandedAccounts` and `CollapsedFolders`. Add a round-trip and old-state case to `store_test.go`.
- Run `make generate` so `frontend/wailsjs/go/models.ts` gains the field.
- `frontend/src/lib/stores/uiState.svelte.ts`: add `collapsedSidebars` to `UIState`, `defaultState`, and `loadUIState`. Add a module-level `$state` mirror, using the same pattern as `activeExtensionState`. Export `isSidebarCollapsed(view = getActiveExtension())` and `toggleSidebarCollapsed(view)`; the latter flips the flag and calls `saveUIState`.

### 2. One toggle action
- `frontend/src/lib/stores/layout.svelte.ts`: add `toggleActiveSidebar()`. In narrow mode it calls the existing `showSidebar()`. Otherwise it calls `toggleSidebarCollapsed(getActiveExtension())`, and if `getFocusedPane() === 'sidebar'` while collapsing, it moves focus to `messageList`.
- Add `isSidebarHidden()`, which returns `layoutMode !== 'narrow' && isSidebarCollapsed()`, for layout and keyboard consumers.

### 3. Toggle button
- `frontend/src/lib/components/kit/ResponsiveSidebarToggle.svelte`: render in all layout modes. It calls `toggleActiveSidebar()`, sets `aria-expanded`, and uses the title/label "Show sidebar" or "Hide sidebar" (narrow keeps "Toggle sidebar"). Icon: `mdi:dock-left` (existing). Update the header comment. Contacts (`ListHeader`) and Calendar (`ViewSwitcher`) already render this component, so they get the toggle with no change.
- `frontend/src/lib/components/list/MessageList.svelte`: replace the inline narrow-only button (~line 1413) with `<ResponsiveSidebarToggle />`. Drop the now-unused `showFolderToggle`/`onToggleSidebar` props and their wiring in `App.svelte` (~line 1658).

### 4. Hiding the sidebar
- **Extensions:** `frontend/src/lib/components/kit/SidebarFrame.svelte` adds `hidden` to the `<aside>` class when `isSidebarHidden()`. The sidebar stays mounted, so selection and keyboard state survive. This covers Contacts (through `SourceSidebar`) and Calendar.
- **Mail:** `frontend/src/App.svelte` (~line 1590) adds `hidden` to the mail `<aside>` when `isSidebarHidden()` and skips the sidebar resize handle in that case. `sidebarWidth` is kept, so expanding restores the previous width. In `handleMouseMove`, list resizing must subtract `0` instead of `sidebarWidth` while the sidebar is hidden.
- Use display-none (`hidden`), not unmounting. `sidebarRef` methods must stay callable, and the mail tree already relies on staying mounted.

### 5. Keyboard
- `frontend/src/lib/keyboard/shortcuts.ts`: add a `SIDEBAR_TOGGLE` predicate: `ctrlOrMeta(e) && e.shiftKey && e.key.toLowerCase() === 'b'`.
- `App.svelte` global Ctrl/Cmd switch (~line 924, alongside `q`/`tab`): call `toggleActiveSidebar()`. It sits after the existing `showComposer` early return, so it stays inert while composing.
- `frontend/src/lib/stores/keyboard.svelte.ts` `focusNextPane`/`focusPreviousPane`: skip `'sidebar'` while `isSidebarHidden()`, so `Alt+H/L` cycles only list and viewer.
- `Alt+J/K/G` sidebar navigation keeps working while collapsed; the result is visible in the list.

### 6. Strings and docs
- `frontend/src/lib/i18n/locales/en.json` `aria`: add `showSidebar` and `hideSidebar`. Edit only English.
- `docs/KEYBOARD_SHORTCUTS.md` (global table plus the quick-reference block) and `docs/user-guide/features/keyboard-shortcuts.md`: add `Ctrl+Shift+B`. Add one line about the toolbar toggle to `docs/user-guide/features/overview.md` if it describes the layout.

## Milestones

**M1: Collapse toggle in all three views (sections 1–4, 6 strings) — done**
Acceptance:
- In full and medium layouts, the toolbar toggle hides and shows the sidebar in Mail, Contacts, and Calendar independently.
- State persists per view across an app restart. Old saved UI state, which lacks the field, loads with every sidebar expanded.
- Expanding mail restores the previous resized width. The list resize handle behaves correctly while collapsed.
- Narrow layout behaves exactly as before: the toggle opens the overlay, and the scrim or back button closes it.

**M2: Keyboard and docs (sections 5–6)**
Acceptance:
- `Ctrl+Shift+B` toggles the active view's sidebar. It does nothing while the composer is open, and in narrow mode it opens the overlay.
- Collapsing while the sidebar has focus moves focus to the list. `Alt+H/L` skips a hidden sidebar.
- The keyboard shortcut docs list the new shortcut.

Each milestone: review the diff, have a read-only subagent check the acceptance criteria, run simplify, then commit locally (`feat:` prefix, no push).

## Verification

- `go test ./internal/appstate/...`, then `make test`.
- `make generate`; confirm `models.ts` has `collapsedSidebars`.
- `cd frontend && npm run check && npm run lint` (or `make lint`).
- `cd frontend && npm run build`, then run the app with `make dev`. In each of Mail, Contacts, and Calendar: toggle by button and by `Ctrl+Shift+B`, then restart and confirm the state persists per view. Resize the window into medium and narrow and confirm overlay behavior is unchanged. Check `Alt+H/L` with a collapsed sidebar. Resize the mail list while the sidebar is collapsed, then expand and confirm the width is restored.
