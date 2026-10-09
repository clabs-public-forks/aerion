# PLAN: Chat-style Mail

## Context

Mail in Aerion today is a classic three-pane client: folder sidebar → `MessageList` → `ConversationViewer`, with replies in a modal `Composer`. The owner wants email to feel like a messenger (Beeper-inspired): a unified chat list of people, threads shown as chat bubbles with quoted history hidden, a reply box always docked at the bottom, and fast triage (Done/archive, pin, snooze, mark unread, low-priority split) so mail is quick to process.

User decisions (2026-10-08):
- The chat UI **replaces** the classic list/viewer in the mail section (no toggle).
- A chat = **one email thread**, presented people-first (avatars, names, last line).
- Triage features: **archive-on-reply / Done flow**, **low-priority (newsletters/bulk) split**, **pin, snooze, mark unread**.
- **No AI**: everything is local and deterministic.

## Scope

In scope:
- Backend: quoted-text/signature stripping for chat bubbles, bulk-mail classification, per-thread local state (pin, snooze), sender overrides, chat-list queries, a chat reply binding.
- Frontend: new `frontend/src/lib/components/chat/` (list, row, thread view, bubble, docked composer), mail-section routing in `App.svelte`, keyboard triage, settings, English strings.
- Docs for the new behavior and settings.

Out of scope: AI summaries/suggested replies; reactions; per-person merged timelines; syncing pin/snooze to the server (local only, see Decision Log); changes to the Contacts/Calendar extensions.

## Design overview

```
Rail │ Chat list (filters: All · Unread · Low priority · Snoozed)  │ Chat thread
     │ ★ Pinned                                                    │ Subject · participants · [Done] [Snooze] [⋯]
     │  (A) Alice Chen        Re: Q3 numbers        10:42  ●2      │ ── Tue 7 Oct ──
     │ Chats                                                       │ (A) Hi! Attached the draft.   [file.pdf]
     │  (B) Bob, Carol        Offsite dates          Mon           │              Looks good, ship it ✓ (me)
     │ Low priority (12) ▸                                         │ ┌ rich HTML mail → collapsed card ┐
     │                                                             │ [Reply ▾ all] [ Type a reply…  ] [⏎ Send] [Send & Done]
```

- **Chat list** = conversations from the Unified Inbox by default (scope menu switches to an account inbox or any folder, reusing the existing folder tree; search reuses `SearchConversations`/`SearchUnifiedInbox`). Sections: Pinned, Chats (priority), Low priority (collapsed group, Beeper "low priority" style), and a Snoozed filter.
- **Chat thread** = existing `GetConversation` result (already merges current folder + Sent + Drafts, `internal/message/store.go:1488`) rendered as bubbles: mine right-aligned, others left with avatars, consecutive same-sender grouping, day separators, a "•••" toggle to reveal stripped quoted text. Messages classified as rich (bulk or layout-heavy HTML) render as a collapsed card that expands into the existing sandboxed `EmailBody` iframe.
- **Docked composer** sends a reply without leaving the chat; "Expand" hands the text to the full `Composer` for formatting, recipients, or attachments edits.

## Reuse (do not reimplement)

- Conversations: `ListConversationsUnifiedInbox` (`internal/message/store.go:113`), `ListConversationsByFolder` (`:1281`), `GetConversation` (`:1488`); bindings in `app/message.go`, `app/search.go`.
- Actions + undo: `Archive` (`app/actions.go:782`), `MarkAsRead/MarkAsUnread`, `Star`, `MoveToFolder`, `DeleteLocalMessages` and `Undo` (`app/undo.go`), undo commands in `internal/undo/commands.go`.
- Replies/drafts: `PrepareReply` (`app/compose.go:646`), `SendMessage` (`:555`), `SaveDraft`/`DeleteDraft` (`app/draft.go`), signatures (`frontend/src/lib/components/composer/composerSignature.ts`), full `Composer.svelte` + `composerApi.ts`/`OpenComposerWindow` for detached mode.
- Rendering: `EmailBody.svelte` (sandboxed iframe, image allowlist, dark mode), `AttachmentList.svelte`, `MessageContextMenu.svelte`, `FolderPickerDialog.svelte`.
- Avatars: `contactPhotos` store (`frontend/src/lib/stores/contactPhotos.svelte.ts:87`).
- Layout/keyboard: `stores/layout.svelte.ts` (narrow list↔viewer switching), `lib/keyboard/shortcuts.ts` (`KEY`, `matchesAny`), toast + undo toast.
- Headers at sync: full header block already fetched (`internal/sync/messages.go:742`, parsed near `:935`).
- HTML handling: `bluemonday` and `golang.org/x/net/html` (via go.mod) for HTML→text.
- Settings pattern: `internal/settings/store.go` key/default/getter → `app/settings.go` wrapper → `make generate` → `frontend/src/lib/stores/settings.svelte.ts`.

## Milestones

### M1 — Chat text extraction (backend)
New `internal/message/chattext.go`: `ExtractChatText(bodyText, bodyHTML string) ChatText{Text, Quoted string, HasQuoted, IsRich bool}`.
- HTML path: drop `.gmail_quote`, `blockquote[type=cite]`, `#divRplyFwdMsg`/`#appendonsend` (Outlook), `.moz-cite-prefix` + following blockquote (Thunderbird), Apple Mail `blockquote type=cite`; then HTML→text.
- Text path: cut at "On … wrote:" (multi-line variants), `-----Original Message-----`, Outlook `From:/Sent:/To:/Subject:` header blocks, runs of `>` lines; strip signature after `-- ` delimiter and common mobile footers ("Sent from my iPhone").
- `IsRich`: HTML with layout tables, many images, or very long text after stripping.
- Expose on `Message` as non-persisted fields filled in the `GetConversation` binding path (`app/message.go:115`); no migration.

Acceptance: stripping never returns empty for a message with content (fall back to full text); quoted part preserved in `Quoted`.
Validation: table-driven tests with fixtures for Gmail, Outlook (desktop + web), Apple Mail, Thunderbird, plain-text `>` quoting, top-posting and inline/bottom-posting (inline replies must keep full text); `go test ./internal/message/...`.

### M2 — Triage data model and bindings (backend)
- Migration 46: `messages.is_bulk INTEGER` (NULL = unknown); table `conversation_state(account_id, thread_key, pinned_at, snoozed_until, PRIMARY KEY(account_id, thread_key))`; table `sender_category(account_id, email, category CHECK IN ('priority','low'))`. `thread_key` = normalized `COALESCE(thread_id, id)` (same normalization as `GetConversation`).
- `internal/sync`: `classifyBulk(header, fromEmail) bool` from `List-Unsubscribe`, `List-Id`, `Precedence: bulk|list|junk`, `Auto-Submitted` ≠ `no`, `noreply`/`no-reply`/`notifications@` senders; set at header sync. Background backfill for inbox rows with NULL `is_bulk`: batched `BODY.PEEK[HEADER.FIELDS (LIST-UNSUBSCRIBE LIST-ID PRECEDENCE AUTO-SUBMITTED)]`, cancellable, folder-lock aware (`internal/sync/folderlock.go`).
- Store: extend conversation listing with `IsPinned`, `SnoozedUntil`, `IsLowPriority` (sender override wins over `is_bulk`), `LastFromMe` (latest message is from an account identity → "awaiting reply"), filtered by section (`all|unread|priority|low|snoozed`); snoozed threads are hidden from other sections until due; pinned sort first.
- App bindings (`app/chat.go`): `GetChats(scope, section, offset, limit)`, `GetChatCount`, `PinChat(accountID, threadKey, pinned)`, `SnoozeChat(accountID, threadKey, until)`, `UnsnoozeChat`, `SetSenderCategory(accountID, email, category)`. Pin/snooze go through undoable commands in `internal/undo`.
- Snooze wake-up: a timer in the app layer re-arms on start and on change; on due it clears `snoozed_until`, marks the latest message unread, emits the existing conversation-changed event, and notifies via `internal/notification`. A thread that gets a new message while snoozed unsnoozes.
- `make generate`.

Acceptance: archive/delete/move of a thread keeps or drops state consistently (state rows for threads with no remaining messages are cleaned up); sender override flips classification for existing and new mail.
Validation: table-driven tests for `classifyBulk`, section queries, pin/snooze undo, migration on a v45 DB; `make test`.

### M3 — Chat list shell (frontend)
- New `components/chat/ChatList.svelte`, `ChatRow.svelte`, `ChatListHeader.svelte` (scope menu, search field, filter chips), store `lib/stores/chat.svelte.ts` (selection, section, scope, pagination, refresh on sync events).
- Row: avatar or stacked avatars (`contactPhotos`), people names (not addresses), subject as secondary line, chat-text snippet, relative time, unread badge, pin/snooze/attachment/encrypted icons, account color dot in unified scope, "↩ you" marker when `LastFromMe`.
- `App.svelte`: mail section renders `ChatList` + `ChatView` instead of `MessageList` + `ConversationViewer`; folder `Sidebar` becomes collapsible and collapsed by default (scope menu covers inbox switching). Narrow layout uses existing list↔viewer switching. Classic components stay in the tree unmounted (see Decision Log).

Acceptance: unified inbox, per-account inbox, any folder and search all list correctly; virtual/paged loading keeps performance on large inboxes; keyboard focus and screen-reader labels on rows.
Validation: `npm run lint`, `npm run check`; run-aerion skill: browse scopes, filters, and search on a real account.

### M4 — Chat thread view (frontend)
- `ChatView.svelte` (header: subject, participant avatars, Done/Snooze/Pin/⋯ menu reusing `MessageContextMenu` actions), `ChatBubble.svelte`, `ChatRichCard.svelte` (wraps `EmailBody`), `ChatDaySeparator.svelte`.
- Bubbles show `ChatText`; "•••" reveals `Quoted`; attachments via `AttachmentList`; S/MIME/PGP status badges and remote-image prompt preserved from the classic viewer; per-message menu (reply to this message, forward, view original/full HTML, copy).
- Opening a chat marks it read (existing behavior and settings); scroll to first unread, else bottom.
- Split if `ChatView` approaches 500 lines.

Acceptance: every message in a thread is reachable with full original content; inline (interleaved) replies are not mangled; encrypted messages render as before.
Validation: lint/check; run-aerion on threads from Gmail, Outlook, a newsletter, an encrypted message, and a thread with my own Sent replies.

### M5 — Docked chat composer
- `ChatComposer.svelte`: autosizing textarea, Enter sends / Shift+Enter newline (setting to swap to Ctrl+Enter), reply ↔ reply-all chip (default from the last message's recipients), attach button, "Expand" opens the full `Composer` (inline or detached per `getComposerMode`) with current text and attachments, **Send & Done** button.
- Backend `SendChatReply(accountID, messageID, text, replyAll, attachments, archiveAfter)` in `app/chat.go`: builds on `PrepareReply` + `SendMessage`, appends account signature, includes quoted history below (setting, default on) so traditional clients keep context; HTML part is escaped paragraphs.
- Debounced draft autosave per chat via `SaveDraft`; draft restored when returning to the chat; deleted on send.
- Sent message appears immediately as a pending bubble, confirmed after send/sync; failure shows retry.

Acceptance: replies thread correctly in Gmail/Outlook (In-Reply-To/References from `PrepareReply`); no lost text on chat switch or app close.
Validation: Go tests for reply assembly (signature, quote, recipients); manual send to a test account; run-aerion.

### M6 — Triage flow and keyboard
- Keys (added to `lib/keyboard/shortcuts.ts`, shown in the existing shortcuts help): `J/K` next/prev chat, `Enter`/`R` focus composer, `Shift+R` reply-all, `E` Done (archive, then auto-advance), `Shift+U` mark unread, `P` pin, `H` snooze picker (Later today, Tomorrow, This weekend, Next week, Pick date), `L` toggle sender low priority, `#` delete, `V` move, `/` search, `Esc` back to list.
- Done / Send & Done auto-advance to the next chat (direction setting), with undo toast via existing `Undo`.
- Row context menu and hover actions (Done, Pin, Snooze, Unread); inbox-zero empty state.

Acceptance: every action is undoable where the backend supports it; shortcuts don't fire while typing in the composer; focus stays predictable after auto-advance.
Validation: lint/check; run-aerion keyboard-only triage of 10+ chats.

### M7 — Low-priority and notification polish
- Low-priority group header with count and "Archive all" (undoable bulk `Archive`) and "Mark all read".
- Row action and thread-header action "Move sender to Low priority / Priority" (`SetSenderCategory`).
- Notifications: setting to notify only for priority chats (default on); snooze wake-ups always notify.

Acceptance: low-priority mail never raises a notification when the setting is on; overrides apply immediately to list sections.
Validation: Go tests for notification filtering; run-aerion with a newsletter and a person.

### M8 — Settings, docs, cleanup
- Settings (Settings → Mail → Chat): send key, include quoted history, auto-advance direction, notify only priority, show low-priority group.
- English locale strings only (`frontend/src/lib/i18n/locales/en.json`), per `docs/LANGUAGE.md`.
- Update user docs/README for the chat mail UI, shortcuts, and local-only pin/snooze.
- Remove classic-only code paths in `App.svelte` that are now dead; keep classic component files (Decision Log).

Validation: `make lint`, `make test`, `cd frontend && npm run check`, full run-aerion pass.

### M9 — Review: app verification and loose ends
Close the checks that earlier milestones skipped or faked, and fix the issues they recorded, before the branch lands. Fix each finding in this milestone if it is small. Otherwise move it to `ISSUES.md` with a reason.

App verification (run-aerion unless noted; follow the skill's real-account gotchas):
- **Dev sync first.** Find out why dev-mode sync fails with "failed to get OAuth token" (the repo has `.env.local` but no `.env`; check how `make dev` loads credentials and where dev tokens are stored), and fix it or document the setup in `CONTRIBUTING.md`/the run-aerion skill. If it needs the owner, ask once and record the answer. Everything below that needs real sync depends on this.
- **M2/M7 with real sync:** the bulk backfill classifies existing inbox mail (low count > 0, newsletters in Low priority, people in Chats); a sender override moves existing and newly arriving mail; Archive all / Mark all read on real low chats, then Undo archive.
- **M7 notifications** (owner-assisted if no test sender is available): with priority-only on, a newsletter raises no notification and a person does; with it off, both notify; a snooze wake-up notifies.
- **M3** on a synced account: unified vs per-account scopes, a non-inbox folder via "All folders…", the Unread/Low/Snoozed chips and search, paging past 50 rows.
- **M4:** an Outlook thread and an encrypted or signed message (fake the binding response if the account has none, and say so).
- **M5 (never run):** docked composer typing, Enter vs the Ctrl+Enter setting, reply ↔ reply-all chip, draft restored after switching chats and after an app restart, a failed send's Retry/Edit (fake the send binding to fail). The owner does one real send to Gmail and to Outlook, confirms threading in each, and checks Expand in inline and detached composer modes. Don't open the composer against a real account without the owner's go-ahead.
- **M6 (never run):** keyboard-only triage of 10+ chats with every key in the M6 list, the row "Pick date..." flow, undo toasts, focus after auto-advance in both direction settings, shortcuts ignored while typing, and the narrow layout (resize the Chrome window myself; the owner allows it).
- **M8 (never run):** each Settings → Chat option takes effect and persists across reload; the user guide matches what the app does.

Code fixes recorded earlier and still open:
- List snippets show HTML entities (`&amp;`) literally (M3); decode them, ideally by building row previews from `ExtractChatText` (M1 note).
- Rows for threads where every message is mine (Sent folder) show my own name; use the recipients (M3).
- Short Gmail replies with a logo table in the trailing signature count as rich and render as cards (M4); ignore tables in the signature block.
- A body fetch error drops the message from the view; show an error state with Retry (M4 review).
- `bind:this={rowRefs[c.key]}` in `ChatList` logs `binding_property_non_reactive`; declare the refs with `$state` or a non-reactive `Map` to silence it (M7).
- `make lint` Go half: rebuild golangci-lint with Go 1.26 (install instructions in `CONTRIBUTING.md` if missing) and run the full `make lint`; fix findings in files this branch changed only.
- Decide the deferred items: the inbox lookup / sync toggle / trash-with-undo copies (M3 review, M8 Decision Log), `ExtractChatText` caching and a backend `mine` flag (M4 review), the notifier's UID-window misclassification (M7). Fix the ones that are cheap; log the rest in `ISSUES.md`.

Branch review:
- Read the whole branch diff (`git diff main...HEAD`) for cross-milestone regressions, then run `/code-review` on it.
- Classic mail behavior still reachable through the chat UI: context menu actions (move, delete, star, mark spam), undo, account switching, and the Contacts/Calendar views unaffected.
- Remove `PLAN.md` Surprises entries that this milestone resolves, so the file stays current.

Acceptance: every check above is done and recorded in Surprises & Discoveries (pass, fixed, or owner-confirmed), or has a reason it can't be done; the code fixes are in or tracked in `ISSUES.md`; `make lint`, `make test`, `cd frontend && npm run lint && npm run check && npm run build` pass. The branch is then ready to land.
Validation: the commands above; run-aerion passes as listed; the owner's checklist answers (real send, notifications), asked once in a single AskUserQuestion prompt.

## Progress

- [x] M1 — Chat text extraction (2026-10-08)
- [x] M2 — Triage data model and bindings (2026-10-08)
- [x] M3 — Chat list shell (2026-10-08)
  - [x] Chat store (`stores/chat.svelte.ts`): paging, sections, search, event refresh (2026-10-08)
  - [x] `ChatRow`, `ChatListHeader`, `ChatList` components + English strings (2026-10-08)
  - [x] `App.svelte` wiring; mail sidebar collapsed by default (2026-10-08)
  - [x] lint/check pass; run-aerion check of scopes, filters, search (2026-10-08)
- [x] M4 — Chat thread view (2026-10-08)
  - [x] Backend: `FetchMessageBody` fills `chat`; encrypted/signed messages skip chat text (2026-10-08)
  - [x] `chatThread` controller (load, refresh, events, mark read, S/MIME/PGP, read receipts) (2026-10-08)
  - [x] `ChatView`, `ChatViewHeader`, `ChatBubble`, `ChatRichCard`, `ChatDaySeparator`, `ChatSecurityBanners` + English strings (2026-10-08)
  - [x] `App.svelte` wiring (replace `ConversationViewer` in the mail section) (2026-10-08)
  - [x] lint/check pass; run-aerion check of plain, rich, Sent-reply, and encrypted threads (2026-10-08)
- [x] M5 — Docked chat composer (2026-10-09)
  - [x] Backend: `SendChatReply`, `SaveChatDraft`, `GetChatDraft`, `ReleaseChatDraft`, `chat_drafts` table (v47), `chat_send_key`/`chat_include_quote` settings, shutdown flush handshake (2026-10-09)
  - [x] `chatComposer` controller, `ChatComposer`, `ChatPendingBubble`, `ChatView`/`App.svelte` wiring + English strings (2026-10-09)
  - [x] Go tests, lint/check pass; run-aerion and a real send not done (cloud session: no display, browser tools, or accounts) (2026-10-09)
- [x] M6 — Triage flow and keyboard (2026-10-09)
  - [x] Keys in `shortcuts.ts` + `handleChatKey` in App; `chatTriage.ts` shared actions; `ChatSnoozeMenu` with Pick date; row hover actions and context-menu items; inbox-zero state; docs (2026-10-09)
  - [x] lint/check/build pass; run-aerion keyboard pass not done (cloud session: no display) (2026-10-09)
- [x] M7 — Low-priority and notification polish (2026-10-09)
  - [x] Backend: `ListNewestMail`, `pickNotifyMail` filter in `handleNewMailNotification`, `chat_notify_priority_only` setting + bindings; Go tests (2026-10-09)
  - [x] `ChatLowGroupHeader` (Archive all, Mark all read), thread-header sender item, English strings; lint/check/build pass (2026-10-09)
  - [x] run-aerion UI pass with faked low-priority rows: group header, Archive all / Mark all read cover every low chat, sender items in the header and row menus (2026-10-09). Real newsletter sync and notifications still unchecked: dev-mode sync fails to get the OAuth token
- [x] M8 — Settings, docs, cleanup (2026-10-09)
  - [x] Backend: `chat_auto_advance`, `chat_show_low_group` settings + bindings; Go tests (2026-10-09)
  - [x] Settings → Chat tab (send key, quoted history, auto-advance, Low priority group, priority-only notifications) + English strings (2026-10-09)
  - [x] Dead checkbox paths removed from `App.svelte` and `ChatList`; user guide, README, shortcut docs (2026-10-09)
  - [x] `make test`, frontend lint/check/build pass; `make lint` Go half blocked (installed golangci-lint built with Go 1.25, repo targets 1.26), `go vet`/`gofmt` used instead; full run-aerion pass not done (cloud session: no display) (2026-10-09)
- [x] M9 — Review: app verification and loose ends (2026-10-09)
  - [x] Dev sync: diagnose "failed to get OAuth token", fix or document (2026-10-09)
  - [x] Code fixes: snippet entities, Sent-row names, signature tables, body fetch error state, rowRefs warning (2026-10-09)
  - [x] `make lint` with a Go 1.26 golangci-lint; fix findings in branch files (2026-10-09: 0 issues; installed v2.14.0 now built with Go 1.27)
  - [x] Deferred items decided (fixed or logged in `ISSUES.md`): all four logged as CM1–CM4, none cheap (2026-10-09)
  - [x] Branch diff read + `/code-review` (2026-10-09: fixed a NULL subject in `ListDueSnoozes`, a NULL `latest_received` dropping a snoozed thread from every section, and one shared snippet `DOMParser`; accepted the rest, see Decision Log)
  - [x] run-aerion passes: M3, M4, M5 (faked), M6, M8, classic actions (2026-10-09; M2/M7 backfill checked read-only after re-authorization, narrow layout not checkable)
  - [x] `/finish-milestone`: review fixes (unified Sent names, stale body fetch, inert entity decode), `/simplify` (recipients for Sent rows only via `parseAggregatedToListJSON`, `rowRefs` as `$state.raw`), checks re-run (2026-10-09)
  - [x] Owner checklist: real sends, Expand, notifications (2026-10-09: items 1–2 done; 3–5 run by Claude with owner-sent mail; all pass)

## Surprises & Discoveries

- `GetConversation` already merges the current folder with Sent and Drafts, so chat timelines include my replies without new queries.
- Full message headers are fetched during header sync but not stored; bulk classification must happen at sync, with a one-time header backfill for existing mail.
- No existing quote/signature stripping; `isSignaturePart` in `internal/sync/parse.go` is about S/MIME/PGP parts only.
- M1: `internal/email` imports `internal/message`, so the chat extractor can't reuse `email.ExtractPlainTextFromHTML`; it has its own DOM walker, which it needs anyway for blockquote depth and Outlook markers.
- M1: sync-time snippets (`generateSnippet`, `internal/sync/helpers.go`) use a cruder `>` rule than chat text. M3 may want list previews built from `ExtractChatText` for consistency.
- M1: `make generate` with the installed Wails CLI also rewrites `frontend/wailsjs/runtime/*`; revert that churn unless upgrading Wails.
- M2: there is no "conversation-changed" event; existing `messages:updated` listeners need a matching `folderId`. Pin/snooze/sender changes emit a new `chats:changed` `{accountId}` event that M3's list should listen to (alongside `messages:updated`/`messages:readChanged`).
- M2: an exported `App` method becomes a Wails binding, so the undo restorer is an unexported adapter (`chatStateRestorer`).
- M2: `received_at` is stored as Go `time.String()` UTC text; SQL compares its first 19 characters against `strftime(..., 'unixepoch')`.
- M2: `TestSyncerSubscriptionsPerInstance` (calendar) failed once in `make test` with a TempDir cleanup race and passed on rerun; unrelated flake.
- M2: the bulk backfill (`backfillBulkFlags`) needs IMAP and has no unit test; it is covered by store tests for `ListUnclassifiedBulk`/`SetBulkFlags` and needs a manual check on a real account.
- M3: on the dev account nothing is classified bulk yet (low count 0), so Low priority grouping was checked in run-aerion with locally mutated rows only; real data needs the M2 backfill or a sync first.
- M3: `IntersectionObserver` callbacks stall in hidden windows, so the list pages with a scroll-distance check instead.
- M4: run-aerion checked a newsletter card (expand/collapse, remote-image prompt), a Gmail thread with my Sent reply (right-aligned bubble, "•••" quote toggle), day separators, and the Snooze and ⋯ menus. The dev account had no encrypted or Outlook threads in view, so those paths are covered by the ported logic and the backend tests only.
- M4: the hover actions overlay a card's top-right corner, which covered the "Show message" toggle; the card header reserves right padding for them.
- M3: the mail sidebar now defaults to collapsed, but users with a saved `collapsedSidebars.mail` value (the dev account has `false`) keep their setting.
- M3: the Low priority toggle button sits inside the `listbox`, which is not an allowed listbox child. Groups now use `role="group"`, but keyboard expansion of the low group is left to M6.
- M5: Wails quits 150 ms after `app:shutting-down`, too soon for a draft save. Shutdown now waits up to 2 s for the frontend's `ChatDraftsFlushed` after flushing every mounted chat composer.
- M5: `GetConversation` includes Drafts, so the chat draft would show as a bubble while typing; the view hides draft messages while the docked composer holds a draft.
- M5: still needs a manual pass: run-aerion, a real send to Gmail/Outlook to confirm threading, and Expand in detached mode.
- M6: `MessageContextMenu` gained an optional `extraItems` snippet (the one change to a shared classic component), so chat rows add triage items on top.
- M6: the snooze date dialog lives inside the row's hover toolbar, so the toolbar stays mounted while the menu or dialog is open (found in review).
- M6: sender Undo clears the override instead of restoring a previous one; the backend keeps no category history. Mark unread has no Undo (backend records none).
- M7: the notify-priority-only setting has bindings but no Settings UI yet; M8 adds the toggle. Snooze wake-ups use their own notifier path and always notify.
- M6: still needs a manual run-aerion keyboard-only triage pass of 10+ chats, including the row "Pick date..." flow and narrow layout.
- M7 run-aerion (2026-10-09): local dev sync fails with "failed to get OAuth token", so the bulk backfill never runs and the real low count stays 0; the pass used real rows flagged low in a page script, with write bindings blocked. With 341 priority chats, the Low priority group header appears only after every Chats page loads (matches the design mock; the Low priority chip is the quick path). Classification after a real sync and priority-only notifications still need a check on an account that syncs.
- M8: there is no Mail settings tab, so the plan's "Settings → Mail → Chat" became a top-level Chat tab.
- M8: the installed golangci-lint is built with Go 1.25 and refuses the repo's Go 1.26 target; `make lint` needs a rebuilt linter.
- Baseline: `gofmt -l` already lists `internal/message/store.go` and several `app/*.go` files on `main`; left untouched.
- M9: dev sync's "failed to get OAuth token" is `token refresh failed: unauthorized_client`. The account's refresh token was issued to another OAuth client than the one `make dev` builds in, and dev shares the native build's data directory and keyring. Not a code bug: the owner re-authorizes once in the dev app (documented in `CONTRIBUTING.md` and the run-aerion skill).
- M9: `make generate` silently writes nothing while Aerion runs: the bindings binary hits the single-instance lock, activates the running app and exits 0. Stop dev first (noted in `AGENTS.md`).
- M9 run-aerion M3 (2026-10-09, cached data): chips All 50 / Unread / Low 0 / Snoozed 0; search "Docusign" gave 7 rows, 5 with Unread; paging 50 → 100 of 341; All inboxes shows account dots; "All folders…" re-opens the sidebar; Sent rows name recipients. Search rows in Sent still name me (logged CM5).
- M9 run-aerion M4 (2026-10-09): a real Outlook thread (3 messages, Sent Mail) renders as bubbles with the "•••" quote toggle. The dev cache has no signed or encrypted mail, so three fake messages were injected through a `GetConversation` wrapper: S/MIME encrypted+signed and PGP signed show their banners above the decrypted text card, and incoming fakes sit left with name and avatar. A forced `FetchMessageBody` failure shows "Couldn't download this message" and Try again loads the body. Outlook signature images list as many inline attachment cards; classic does the same (logged CM6).
- M9 run-aerion M5 (2026-10-09, faked draft/send bindings): Enter sends, Shift+Enter adds a line; the Reply All ↔ Reply chip toggles; a draft and its mode come back after switching chats; a failed send shows "Not sent · Retry · Edit" and Edit restores the text; Reply All passes `replyAll:true` and the draft ID; with send key Ctrl+Enter, Enter adds a line and Ctrl+Enter sends. Draft persistence across a reload not checked (writes blocked).
- M9 run-aerion M6 (2026-10-09, faked writes): j/k move the selection; E archives with "Conversation archived · Undo" and focus returns to the list; auto-advance "next" picks the row that took its place, "previous" the row above; P pins, L moves the sender to Low priority, Shift+U marks unread, H opens the snooze menu ("Pick date..." defaults to tomorrow 08:00), # trashes, V opens "Move to" with search focused; each shows its toast and Undo calls the Undo binding. None of these keys fire while the composer has focus. Narrow layout not checked: the Chrome window can't be resized.
- M9 run-aerion M8 (2026-10-09): send key, Low priority group and auto-advance settings apply in-session, and the user guide's Chat tab section matches the labels. Persistence across a restart goes to the owner checklist.
- M9 run-aerion classic actions (2026-10-09): the row context menu's Star, Mark as Spam and Delete call their bindings; the scope menu offers All inboxes / Personal / All folders…; Calendar and Contacts (165, read-only Google source) render and Mail returns. Only one account exists, so account switching was checked through scopes only. M2/M7 real sync: see the entry below.
- M9 run-aerion: in a hidden Chrome tab rAF never fires, so a closed bits-ui dialog stays mounted, keeps the dialog guard active and swallows shortcuts; reload to clear it (added to the run-aerion skill).
- M9 M2/M7 real sync (2026-10-09, after the owner re-authorized; read-only DB check): the backfill marked 168 of 361 real INBOX threads bulk (Low priority: Builtin, Rippling, university no-reply), while person replies (the Gmail and Outlook test replies) stayed priority. Sender overrides, Archive all / Mark all read and Undo on real low chats were not run because they change the owner's mail; the faked M6 pass covers their UI, and the store tests cover overrides.
- M9 owner checklist (2026-10-09): (1) re-authorization and sync work; (2) a chat send to a Gmail and an Outlook address arrived and both replies threaded in Aerion and at both recipients, with reply text in the bubble and the quote behind "•••" (checked against the stored bodies). Gmail filed the first message as spam although SPF, DKIM and DMARC pass; the non-FQDN `Message-ID <uuid@aerion>` and the `EHLO localhost` greeting are the only odd headers (Message-ID fixed separately, HELO logged as CM8). A reply sent from the Gmail web UI showed up only after the next Sent sync (logged CM7). Items 3–5 (Expand inline/detached, notifications and snooze wake-up, settings persistence after restart) were not run.
- M9 owner checklist items 3–5, run by Claude (2026-10-09):
  - Expand (faked writes): inline Expand saves, then releases the chat draft and opens it in the full composer with the text. Detached mode calls `OpenComposerWindow` with the draft ID. An empty box takes the fresh-reply path. The native detached window itself was not opened.
  - Snooze wake-up (real thread, 30–60 s snoozes): the snooze clears on time and a portal notification "Snoozed chat with Sean Casey" / "Re: This is a new test thread" fires. On Linux Aerion uses `org.freedesktop.portal.Notification` (AddNotification), not `org.freedesktop.Notifications`, so a dbus-monitor watching only Notify sees nothing.
  - Settings persistence: send key, include quote, auto-advance, priority-only notifications and the Low group survive a full app restart (DB rows, getters and the Settings → Chat tab), then restored to defaults.
  - Narrow layout (600 px same-origin iframe, under 768 px): the list fills the width without horizontal overflow, the thread view has Back and a truncated header, and Back returns to the list.
  - Newsletter vs personal notification: PASS. With `caseysean@cityuniversity.edu` temporarily set to Low (sender category as the low signal, since webmail can't set list headers), the owner sent one mail from it and one from sc.junk.van. Only the personal one raised a portal notification ("New email from Linda Liu"), and the category was cleared afterwards. `TestClassifyBulk` covers header-based bulk detection.
  - Mail that arrives while Aerion isn't running raises no notification on the next start: notifications fire only when the inbox count grows during a running sync.
- M9: adding a `_test.go` file under `internal/` while `run-aerion` is up makes `wails dev` rebuild and restart the app; put scratch tests elsewhere.
- 2026-10-09 (M9, baseline): `gofmt -l internal/message` lists the unmodified upstream `store.go`; left alone to keep upstream merges clean (`make lint` still reports 0 issues).

## Decision Log

- 2026-10-08: Chat UI replaces the classic list/viewer (owner choice). Classic `MessageList`, `ConversationRow`, `ConversationViewer` files stay in the tree unmounted to keep upstream merges conflict-free; reused pieces (`EmailBody`, `AttachmentList`, `Composer`, context menu) are shared.
- 2026-10-08: A chat is one thread, not one person (owner choice); maps onto existing threading.
- 2026-10-08: Pin and snooze are local-only state (SQLite), not IMAP keywords or a Snoozed folder: keyword support varies by server and folder moves break other clients. Trade-off: not shared across devices.
- 2026-10-08: Pin is separate from Star (`\Flagged`) so starring keeps its server meaning.
- 2026-10-08: Chat text is computed on read, not stored, to avoid a body backfill; revisit if profiling shows cost.
- 2026-10-08: Chat replies include quoted history by default so recipients on traditional clients keep context.
- 2026-10-08: No AI features (owner choice).
- 2026-10-08 (M1): Attribution detection is one backward scan from each `>` run: the nearest line ending with ":" (covers "wrote:" and non-English forms like "a écrit :"), extended over an "On …" start or Gmail's `<`-wrapped lines. A trailing "On … wrote:" with no quote after it is kept as text.
- 2026-10-08 (M1): Review findings declined: no paragraph blank lines for `<p>`/`<div>` (Outlook and Gmail use one block per line); fallback text keeps `> ` markers like plain text does; Outlook forwards without a marker look like replies, so the forwarded part goes to `Quoted`; ordered lists use bullets.
- 2026-10-08 (M1): HTML is rendered to text with blockquotes as `> ` lines and Outlook `#divRplyFwdMsg`/`#appendonsend` as a hard cut, then one set of text rules handles every client. This replaces per-client DOM removal (`.gmail_quote`, `.moz-cite-prefix`, …), and inline-reply detection works the same for HTML and plain text. Forwards (Gmail `.gmail_quote` without a blockquote) are kept.
- 2026-10-08 (M1): Quotes are stripped only when all quoted regions sit on one side of the new text (top- or bottom-posting); interleaved quotes keep the full text. `Quoted` also holds the signature (shown behind "•••"); mobile footers are dropped outright. Exposed as `Message.Chat` (`*ChatText`, `json:"chat"`), filled only for messages whose body has been fetched.
- 2026-10-08 (M1): `IsRich` = a layout table (role=presentation, width 100%/≥500px, or nested table) or ≥3 non-tracking images in the new content, or >4000 runes of stripped text. Content inside quotes or after an Outlook cut is ignored. Known gap: an Outlook desktop original (no blockquote) is excluded only by line position.
- 2026-10-08 (M2): `conversation_state` gains `snoozed_at`. A snooze ends early when an inbox message (in the listed scope) arrives after `snoozed_at`, decided at query time, so sync needs no hook. Moving a thread message back into the inbox (e.g. undoing an archive) also counts as new mail.
- 2026-10-08 (M2): A thread is low priority only when every message in scope is low (sender override, else `is_bulk`); one human reply lifts a list thread into Chats. `all` includes low threads (flagged `isLowPriority`) so the list can group them; `priority`/`low` split them.
- 2026-10-08 (M2): State rows are kept while the thread has any message in the account (archived or Sent-only included) and cleaned at startup only. Cleanup during deletes risked dropping state mid-move, because sync re-creates moved rows.
- 2026-10-08 (M2): `LastFromMe` = a Sent copy at least as new as the latest listed message, or the latest listed message is from an account address or identity. It is computed per page after the main query.
- 2026-10-08 (M2): Bindings take `scope` as a folder ID ("" = unified inbox) and snooze time as Unix milliseconds. The wake timer waits at most 5 minutes so it stays correct across suspend. A due snooze clears (pin kept), marks the latest inbox message unread through `setReadStatus` (IMAP included), and notifies. Threads already woken by mail, or with no inbox messages left, are only cleared.
- 2026-10-08 (M2): The bulk backfill runs inside `FetchBodiesInBackground` for inbox folders, on its connection and under its `body:` folder lock, so every sync path covers it. Rows the server doesn't return are classified by sender alone so they aren't refetched each sync.
- 2026-10-08 (M2 review): Pin/snooze changes and the wake-up share `chatStateMu`; the wake rechecks each row before clearing. A failed wake retries after 1 minute, not every second. System wake re-arms the timer, because Go timers pause during suspend. A snooze stored without `snoozed_at` gets one. Startup cleanup runs in a goroutine.
- 2026-10-08 (M2 review, declined): The `received_at = now()` fallback only applies when the server omits INTERNALDATE, which is always requested. Leftover state rows after a delete are harmless until the next start. Classifying unreturned UIDs by sender is kept so they aren't refetched every sync. Kept the one-shot timer rather than a one-minute ticker, for snooze precision. Also deferred: normalizing stored `thread_id` brackets so the REPLACE/`IN (?, ?)` matching can go away, and batching the due-snooze lookups. Both are candidates if list performance needs it.
- 2026-10-08 (M3): `ChatList` keeps `MessageList`'s exported ref API (select/open/delete/move/sync, check stubs), so `App.svelte` keyboard and action wiring stay as they are. Multi-select (checkboxes) is not offered in the chat list; check APIs are no-ops. Until M4 adds `ChatView`, the right pane keeps `ConversationViewer`.
- 2026-10-08 (M3): Scope stays App's existing folder selection (`unified`/`inbox` = unified inbox → `GetChats("")`, else the folder ID), so persistence and sidebar selection keep working. The scope menu lists All inboxes, each account inbox, and "All folders…", which opens the sidebar. The store holds list data (section, pages, search, refresh), not scope.
- 2026-10-08 (M3): The All filter loads one `all` stream and groups loaded rows client-side into Pinned, Chats, and a collapsed Low priority group (count from `GetChatCount(low)`). Pinned low-priority threads stay in Pinned. One stream keeps paging simple; with a mostly-bulk inbox, the scroll check keeps loading pages until Chats fill the view.
- 2026-10-08 (M3): Performance relies on paging (50 per page, a scroll-distance check, not IntersectionObserver, which stalls in hidden windows) plus `content-visibility: auto` on rows, not a windowed virtual list, matching how `MessageList` scales today.
- 2026-10-08 (M3): Under the Unread filter, rows read in place stay until the next reload, so the open chat doesn't vanish from under the selection. During search, the Low and Snoozed chips are disabled and the list falls back to All, because search results carry no triage state.
- 2026-10-08 (M3): Search uses the existing FTS bindings (`SearchUnifiedInbox`/`SearchConversations`) and shows conversation results as chat rows without pin/snooze state. Server (IMAP) search and Empty Trash/Spam bars are not carried over.
- 2026-10-08 (M4): Thread logic (load, debounced refresh, sync/move/delete/undo events, mark-as-read, on-demand body fetch, S/MIME/PGP processing, read receipts) moves into a `ChatThread` controller class (`chat/chatThread.svelte.ts`), so `ChatView` stays under 500 lines. It is a port of `ConversationViewer`'s logic, not a shared refactor, because the classic file stays untouched (see the first entry).
- 2026-10-08 (M4): Each message renders in one of three modes. **Bubble**: `chat.text` as linkified plain text, with "•••" revealing `quoted`. **Card** (`isRich`): a collapsed card (subject line + first text) that expands into `EmailBody`. **Full**: `EmailBody` inline in a wide bubble, used for encrypted/signed messages (their stored body is the envelope; decrypted HTML comes from `ProcessSMIMEMessage`/`ProcessPGPMessage`), for bodies without chat text yet, and when the user picks "Show original". So every message's full content is reachable, and the remote-image prompt and dark-mail toggle come from `EmailBody` unchanged.
- 2026-10-08 (M4): The ChatView keeps `ConversationViewer`'s exported ref API (reply, archive, trash, scroll, context menu, focused message, …), so App's keyboard wiring is unchanged. Done = Archive of the whole thread. Header "⋯" opens the shared `MessageContextMenu` for the thread; bubbles keep it on right-click plus a hover menu (Reply, Reply all, Forward, Show original/chat view, View source, Copy text). Print and the focus-mode buttons are dropped from the chat header; App's focus-mode shortcuts still work (message focus filters the thread to one message), and the now-unused `toggleMessageFocus` is removed.
- 2026-10-08 (M4): Bubble mode shows plain chat text only, so HTML images, link targets and the remote-image prompt appear once a card is expanded or "Show original" is picked; the prompt itself is `EmailBody`'s, unchanged.
- 2026-10-08 (M4): Header snooze offers presets only (Later today, Tomorrow, This weekend, Next week) from a shared `chatSnooze.ts`; M6 adds "Pick date" and the `H` key. Pin and snooze state comes from the selected `chatList` item, updated on `chats:changed`.
- 2026-10-09 (M5): Bindings take a `ChatReply` struct (account, thread key, replied-to message, text, reply-all, attachments, draft ID) instead of positional args. Send & Done archives in the frontend after a successful send through the view's existing Done (undo toast, auto-advance), so the backend has no `archiveAfter`.
- 2026-10-09 (M5): Chat drafts are ordinary drafts (`SaveDraft`, so IMAP sync and draft encryption apply) holding the full reply. A `chat_drafts` row maps the thread to the draft and stores the typed text's byte length, so the text is recovered as the body's prefix without storing it unencrypted. The row cascades away with the draft.
- 2026-10-09 (M5): Replies answer the latest message from someone else (else my latest, as reply-all); reply-all is the default when that message involves more than one other person. Expand saves the draft, detaches it from the chat, and opens it as a draft in the full composer (detached per `getComposerMode`); with no text it opens a normal reply.
- 2026-10-09 (M5): A failed send keeps a "Not sent" bubble with Retry and Edit, and its text is saved as a draft so it survives a restart.

- 2026-10-09 (M6): Triage keys act on the list selection (the open chat after J/K or a click), falling back to the open chat. In wide layouts J/K open each chat as it is selected; in the narrow layout they only move the selection. In the open chat, J/K change chats and the arrows scroll.
- 2026-10-09 (M6): `V` is Move in the chat list (Enter opens and focuses the reply box), replacing the classic list's V = open; kit lists keep V = open. Esc from the composer or open chat returns focus to the list and keeps the chat open.
- 2026-10-09 (M6): After any auto-advancing action (Done, Snooze, delete, move), focus moves to the chat list so keyboard triage continues. Auto-advance goes to the row that takes the removed row's place; the direction setting is left to M8.
- 2026-10-09 (M6): Mark unread from the list closes the open chat (existing `ChatThread` behavior on an external mark-unread), so auto mark-as-read cannot undo it.
- 2026-10-09 (M7): Notification filtering is per message (sender override, else `is_bulk`), not per thread: a newsletter reply never notifies, a person writing in a list thread does. With the setting on, a failed lookup skips the notification (fails closed).
- 2026-10-09 (M7): Low group "Archive all" and "Mark all read" act on every unpinned low chat in scope (fetched with `GetChats(scope, 'low')`), not only loaded rows. Archive all is one undoable `Archive`; Mark all read has no undo, like mark unread.
- 2026-10-09 (M7, declined): review flagged the header sender item acting on the first other sender. A thread is low only when every message is low, so moving that sender to Priority always lifts it; kept, matching the row action.
- 2026-10-09 (M8): Hiding the Low priority group makes All load the backend `priority` section instead of filtering client-side, so paging, counts and the empty state stay correct. Trade-off: a pinned low-priority chat then shows only under Low priority.
- 2026-10-09 (M8): Auto-advance "previous" opens the row above the removed one (clamped to the top); "next" keeps M6's behavior. Include-quote and notify-priority-only are read by the backend only, so the frontend store holds just send key, auto-advance and the group toggle.
- 2026-10-09 (M8): Classic dead paths removed are the checkbox/multi-select branches in App's keyboard handler and ChatList's no-op check stubs. Consolidating the inbox lookup, sync toggle and trash-with-undo copies (M3 review) is left for later; the classic components stay in the tree.
- 2026-10-09 (M9, owner answers): Investigate the dev OAuth token failure myself; ask only if it needs a secret. The M5 composer pass uses faked draft and send bindings only, so nothing reaches the real mailbox. The owner does the real Gmail/Outlook sends and the Expand check from a checklist I write. For the M7 notification check, the owner sends one newsletter-style mail and one personal mail when asked, and reports whether each notified.
- 2026-10-09 (M9): Snippet entities are decoded in the frontend `chat` store (`DOMParser`), not in Go, so search results from upstream `store.go` are covered and nothing is decoded twice. Sent-only threads name their recipients through a new `Chat.Recipients` (deduped `to_list` of the thread's Sent messages, parsed by upstream's `parseAggregatedToListJSON`) that `chatPeople` falls back to. Content inside a signature block (`gmail_signature`, `moz-signature`, `#Signature`) no longer signals rich HTML. A failed body download keeps the message with an error bubble and Retry; failed IDs aren't refetched on reload, and a message deleted on the server leaves with the next sync's reload.
- 2026-10-09 (M9 `/code-review`): Accepted without change: the per-row `json_group_array(to_list)` (one indexed grouped query, already paged to 50 rows); the 2 s `chatFlushWait` bound on quit; `is_bulk` NULL for messages without headers counting as priority (the safe default); the bulk backfill covering inbox folders only, which is all the chat sections read; `rowRefs` as state (needed for `bind:this` into a record; `/simplify` made it `$state.raw` so rows aren't proxied).
- 2026-10-09 (M9 `/finish-milestone`): Fixed: the unified inbox named Sent rows by sender (now matches `ChatRow`), a body fetch finishing after a thread switch, and the entity decoder writing into a live document. Declined from `/simplify`: decoding entities in Go sync (upstream code, and search would still need it), a backend "all mine" flag, a Set helper in `chatThread`, the signature counter's `defer` (matches `depth`/`pre`), parallel body fetches (the serial loop predates M9), and telling not-found from network body errors.

## Outcomes & Retrospective

- M1: `ExtractChatText` in `internal/message/chattext.go` handles Gmail, Outlook desktop and web, Apple Mail, Thunderbird and plain-text quoting through one set of text rules, with 38 fixture tests. It is exposed as `Message.chat` from `GetConversation`. Review fixes: rich-signal boundary, bounded signature length, hidden preheaders, list bullets.
- M2: Migration 46 adds `is_bulk`, `conversation_state` and `sender_category`. `classifyBulk` runs at header sync, with an inbox backfill. `ListChats`/`CountChats` support the all, unread, priority, low and snoozed sections, pinned-first ordering, and the low-priority and awaiting-reply flags. Six bindings in `app/chat.go`: pin and snooze are undoable, and a snooze timer wakes due chats (marks them unread and notifies). Review fixes: state-lock race, wake retry backoff, re-arm on resume, NULL subjects, an index-friendly Sent lookup. The bulk backfill still needs a manual check on a real account.
- M3: `ChatList`, `ChatRow`, `ChatListHeader` and the `chatList` store replace `MessageList` in the mail section. The list has a scope menu (All inboxes, account inboxes, All folders…), All/Unread/Low/Snoozed chips, FTS search, Pinned/Chats/Low priority grouping, 50-row paging and `content-visibility` rows, and a listbox with option labels. The mail sidebar defaults to collapsed. Review fixes: stale-response invalidation on scope, filter and query changes; search filter fallback; no row drop under Unread; no retry loop after errors; listbox group roles.
- M4: `ChatView` (with `ChatViewHeader`, `ChatBubble`, `ChatRichCard`, `ChatDaySeparator`, `ChatSecurityBanners`) and the `ChatThread` controller replace `ConversationViewer` in the mail section. Plain mail renders as bubbles with a quote toggle, Sent replies on the right, rich mail as expandable cards, encrypted mail through `EmailBody`. Header Done/Snooze/Pin/Unread/⋯ with undo toasts; App's ref API and shortcuts are unchanged. `FetchMessageBody` now fills `chat`. Review fixes: stale thread cleared on switch, blank pane after an external mark-unread, menu "Mark as unread" no longer re-marked read, S/MIME/PGP results kept across refreshes, duplicate Delete handling removed, malformed mailto links, snooze preset duplicates, `aria-pressed` on non-toggles, pending read ids limited to changed messages.
- M8: Settings → Chat tab with all five chat options (two new settings: auto-advance direction, Low priority group), dead checkbox paths removed from App, chat mail documented in the user guide, settings page, README and shortcut docs.
- M6: Keyboard triage (J/K, Enter/R/Shift+R, E, Shift+U, P, H, L, #, V, /, Esc), shared `chatTriage` actions with undo toasts, a snooze menu with Pick date in header and rows, row hover actions and context-menu triage items, and an inbox-zero state. Review fixes: row snooze dialog unmounting on pointer leave, R not returning to reply after Shift+R, stale composer focus requests, Enter in the open chat.
- M9: Verified M2–M8 in the running app (faked bindings where writes were involved) and on real Gmail/Outlook round trips. Fixed snippet entities, Sent-row names, signature-only rich cards, body fetch errors (bubble with Retry), NULL subject/`received_at` edge cases, and the Go 1.26 lint setup. Logged CM1–CM8 in `ISSUES.md` (Sent sync waits for the schedule because IDLE covers INBOX only; `EHLO localhost`). Gmail spam-foldered the first test send, likely from the domainless `@aerion` Message-ID; fixed in a follow-up `fix:` commit. Owner checks 3–5 and the narrow layout remain.
- M7: `ChatLowGroupHeader` adds undoable Archive all and Mark all read to the Low priority group; the open chat's "⋯" menu gains Move sender to Low priority/Priority; new-mail notifications skip low-priority mail behind `chat_notify_priority_only` (default on). Review fix: fail closed when the classification lookup errors.
