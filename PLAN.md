# PLAN: Sender Chats

## Context

Chat mail groups the chat list by email thread: `chatBaseQuery` (`internal/message/chat_store.go:130`) groups on `thread_key, a.id`, and `thread_id` comes from `Message-ID`/`In-Reply-To`/`References` (`FindThreadID`, `internal/message/store.go:1792`). Automated senders such as `notifications-noreply@cityu.brightspace.com` send every notification as a new, unthreaded message, so each one becomes its own chat and floods the list.

The owner wants such senders combined into one chat even when subjects differ (decision 2026-10-09). It is opt-in per sender, so human conversations keep their thread-per-chat behavior.

## Scope

In scope:
- Backend: a per-account, per-sender "combine" flag; a chat key that maps that sender's threads to one sender chat; chat list, count, search, pin/snooze, snooze wake-up, orphan cleanup, new-mail notifications and chat drafts keyed by the chat key; a binding that loads a sender chat as a conversation.
- Frontend: a "Combine messages from this sender" / "Show as separate threads" action with undo, sender chat rows and thread view (per-message subject labels), and opening sender chats from notifications.
- English strings and user-guide docs.

Out of scope: combining automatically (heuristics or bulk detection); combining across accounts; combining by domain or by display name; per-person timelines for human contacts beyond what the flag gives; carrying pin/snooze over between thread chats and a sender chat (see Decision Log); classic message list behavior.

## Design overview

- **Flag:** migration 48 adds `sender_chat(account_id, email, PRIMARY KEY (account_id, email))`, with emails stored lowercased like `sender_category`.
- **Chat key:** a thread belongs to the sender chat when its **starter** (the earliest message of the thread in the chat scope) is from a combined sender. Its chat key is then `sender:<email>`; otherwise it stays the normalized thread key (`threadKeyExpr`). Grouping whole threads, not single messages, keeps a human reply inside its thread chat instead of splitting the thread across two chats. `sender:` cannot collide with a thread key, which is a Message-ID or UUID.
- **Query shape:** the inner grouped query becomes two levels: group by `(thread_key, account)` to find each thread's starter, map each thread to its chat key, and then group by `(chat_key, account)` with the existing aggregate columns. `conversation_state` and `chat_drafts` join on the chat key, so pin, snooze and drafts work for sender chats without schema changes.
- **Thread view:** `GetSenderChat(accountID, email, folderID)` returns a `message.Conversation` whose messages are those of every thread in scope that maps to the sender chat, plus Sent/Drafts messages in those threads (as `GetConversation` merges them), in date order. Bubbles show a subject label when a message's subject differs from the previous bubble's.
- **Replying:** the docked composer replies to the latest received message in the sender chat. Threading headers come from that message, so outgoing mail is unchanged. Done/archive acts on all of the chat's message IDs, as today.

## Reuse (do not reimplement)

- Chat queries and state: `chatBaseQuery`, `chatListColumns`, `chatCountColumns`, `ListChats`, `CountChats`, `SearchChats`, `threadRecipients`, `fillLastFromMe`, `GetChatState`/`SetChatState`, `ListDueSnoozes`, orphan cleanup (`chat_store.go:508`), `ListNewMail` (`chat_store.go:620`), `threadKeyExpr`/`threadKeyMatch`/`threadKeyArgs`.
- Sender override pattern: `SetSenderCategory` (`chat_store.go:523`, `app/chat.go:78`) and its frontend undo toast (`frontend/src/lib/components/chat/chatTriage.ts:133`).
- Conversation loading: `GetConversation` (`internal/message/store.go:1509`, binding `app/message.go:131`) for the Sent/Drafts merge; `chatThread.svelte.ts` for loading and refresh.
- Undo: `changeChatState` and `chatStateRestorer` (`app/chat.go`), undo commands in `internal/undo/commands.go`.
- Chat drafts: `chat_drafts` keyed by thread key (`internal/draft/chat.go`, `app/chat_reply.go`).
- Bindings: `make generate`, then revert unrelated churn in `frontend/wailsjs/runtime/*`.

## Milestones

### M1: Sender chat data model and list queries (backend)

- Migration 48: `sender_chat` table (see Design overview).
- Store: `SetSenderChat(accountID, email string, combined bool)` and `IsSenderChat`. Normalize the email, and reject an empty one.
- Chat key: add a chat-key mapping (see Design overview) and use it in `chatBaseQuery` grouping, `conversation_state` joins, `fillLastFromMe`, `threadRecipients` and `SearchChats`. Add `SenderEmail string` to `Chat` (set only for sender chats) so the frontend can tell the two kinds apart. `ThreadKey` carries the chat key.
- Make the state paths accept `sender:` keys: `ListDueSnoozes` finds a sender chat's latest inbox message by mapped threads; orphan cleanup keeps a `sender:` state row while the flag is set and the sender chat has inbox messages; `ListNewMail` reports the chat key so notifications open the right chat.
- On turning the flag off, delete the sender chat's `conversation_state` row and chat-draft link (the draft itself is kept, as `ReleaseChatDraft` does).
- Binding `SetSenderChat(accountID, email, combined)` in `app/chat.go`, undoable through an `internal/undo` command that restores the previous flag. Emit the existing chat-list refresh event.

Acceptance:
- With the flag set, N unthreaded messages from the sender appear as one chat row whose unread count, snippet, latest date and message IDs cover all N. Turning the flag off restores N rows.
- A thread started by the sender that contains a human reply stays whole in the sender chat. A thread started by someone else that contains a message from the sender stays a separate thread chat.
- Sections behave as before: one priority message lifts the sender chat out of Low priority, and snoozed sender chats appear only under Snoozed.
- Pin and snooze set on a sender chat survive new mail from the sender. A due snooze wakes the sender chat and marks its latest message unread.
- Flags are per account: the same address in another account is unaffected.
- Thread chats of senders without the flag behave exactly as before, with the same rows and order on an existing database.

Validation: table-driven tests in `internal/message` for the chat-key mapping (unthreaded messages, threaded with human reply, other starter, two accounts), sections, search, `fillLastFromMe`, due snoozes, orphan cleanup and `ListNewMail`. Test the undo command and migration 48 on a v47 database. Run `go test ./internal/message/... ./internal/undo/... ./internal/database/... ./app/...`, then `make test`.

### M2: Sender chat conversation and replies (backend)

- Store: `GetSenderConversation(accountID, email, folderID)` builds a `Conversation` from all threads mapped to the sender chat in that scope, plus their Sent/Drafts messages, sorted by date. It reuses `GetConversation`'s per-message loading rather than copying it. A scope of `""` (unified inbox) covers that account's inbox.
- Binding `GetSenderChat(accountID, email, folderID)` in `app/message.go` or `app/chat.go`, filling chat text the same way the `GetConversation` binding does.
- Chat replies and drafts: `SendChatReply`/`SaveChatDraft`/`GetChatDraft` accept a sender chat key. The frontend passes the reply target message ID, so `buildChatReply` needs no new threading logic. Verify that the draft link stays keyed by the chat key.
- `make generate`.

Acceptance:
- `GetSenderChat` returns every message shown in the sender chat row and none from other chats, including mail I sent in those threads.
- A chat reply in a sender chat threads onto the targeted message (`In-Reply-To`/`References` match it), and the draft survives a restart.
- Large sender chats (500+ messages) load within the existing conversation load budget. Check with a test fixture and a timing log, and record the result in Surprises & Discoveries.

Validation: table-driven tests for `GetSenderConversation` (scope, Sent merge, cross-account isolation), plus a reply-header test via `buildChatReply`. Run `make test`.

### M3: Combine action and sender chat UI (frontend)

- Action: "Combine messages from this sender" in the chat view overflow menu and the row context menu, next to the low-priority action. Inside a sender chat it becomes "Show as separate threads". Use a `chatTriage.ts` helper with an undo toast, modeled on `setSenderLow`.
- Store (`chat.svelte.ts`): rows with `senderEmail` open via `GetSenderChat`; selection follows the chat key across the combine/split refresh, so the merged chat stays selected after combining.
- `chatThread.svelte.ts`: load sender chats through `GetSenderChat`, with refresh on sync events as for threads. Reply target: the latest received message.
- Rendering: `ChatRow` shows the sender as the title and the latest subject as the secondary line. `ChatBubble` shows a compact subject label when the subject changes between consecutive bubbles; quoted-history toggles and rich cards are unchanged. `ChatViewHeader` shows the sender instead of a thread subject. Keep `ChatView.svelte` (now 496 lines) under 500 lines by putting new logic in the store or a helper.
- Notifications: clicking a new-mail notification for a combined sender opens the sender chat.
- English strings in the English locale only, and keyboard access for the new menu items.
- Docs: describe combining senders in `docs/user-guide/features/email-management.md`, and mention it in README chat mail notes if those list triage features.

Acceptance:
- On a real account, combining `notifications-noreply@cityu.brightspace.com` (or any repeated sender) collapses its chats into one row. The thread view shows every message with subject labels, Undo restores the separate chats, and "Show as separate threads" does the same later.
- Pin, snooze, mark unread, Done and low-priority actions work on a sender chat.
- Replies from the docked composer go to the latest message, and drafts persist when switching chats.
- No regressions in thread chats, search or the narrow layout.

Validation: `cd frontend && npm run lint && npm run check`. Use the `run-aerion` skill to combine a sender, browse, reply, snooze, undo and split, and take screenshots of the row and thread view. If no display is available, record that in Progress.

## Progress

- [x] M1: Sender chat data model and list queries (2026-10-09)
  - [x] Migration 48 and SetSenderChat/IsSenderChat (2026-10-09)
  - [x] Chat-key fragment in chatBaseQuery, fillLastFromMe, SearchChats (2026-10-09)
  - [x] State paths: ListDueSnoozes, orphan cleanup, ListNewMail (2026-10-09)
  - [x] Binding SetSenderChat with undo command (2026-10-09)
  - [x] Tests (message, undo, migration) and make test (2026-10-09)
- [ ] M2: Sender chat conversation and replies
- [ ] M3: Combine action and sender chat UI

## Surprises & Discoveries

- 2026-10-09 (M1): `SearchChats` merges results of the same sender chat after paging, so a merged page can hold fewer results than `limit`. The frontend must not treat a short page as the end of results (M3).
- 2026-10-09 (M1): `ChatSearchResult` gained a `threadKey` (the chat key), so search rows can open sender chats without recomputing the key in the frontend.
- 2026-10-09 (M1): Search merges sender chats only within a page, so a sender chat whose threads span two pages can appear on both. M3 should merge search rows by `threadKey` across loaded pages.
- 2026-10-09 (M1): A thread's starter is its earliest message in scope, so a thread can move between a sender chat and a thread chat when that first message is deleted or moved out of the inbox. This follows from the starter rule.
- 2026-10-09 (M1): The starter window runs over the whole inbox scope. `ListChats`/`CountChats` use a plain thread-key fragment when no account has a combined sender, so databases without sender chats run the old query shape.
- 2026-10-09 (M1): `NewMail.ChatKey` is filled but `app/background.go` still opens notifications by thread ID. M3 must switch notification clicks to the chat key.
- 2026-10-09 (M1): `gofmt -l` already flags `internal/message/store.go` and several `app/*.go` files before this feature; they are left untouched.

## Decision Log

- 2026-10-09: Combining is opt-in per sender and per account, not automatic. Auto-combining every sender, or every bulk sender, would collapse human threads that have different subjects and surprise the owner.
- 2026-10-09: A thread joins a sender chat by its starter, not message by message, so threads are never split across two chats.
- 2026-10-09: Pin/snooze state is not carried between thread chats and a sender chat. Existing per-thread rows remain unused while the flag is set and are still there after splitting. Turning the flag off drops the sender chat's own state.
- 2026-10-09: Replies from a sender chat target the latest received message, and outgoing threading headers are unchanged.
- 2026-10-09: The flag is stored in a new `sender_chat` table instead of a new category value, because combining is independent of priority/low classification.
- 2026-10-09 (M1): The chat key comes from one shared SQL fragment, `chatKeyRows`, which yields `(mid, thread_key, account_id, chat_key)` for messages in scope. The thread starter is found with `FIRST_VALUE(LOWER(from_email)) OVER (PARTITION BY thread_key, account ORDER BY date, id)` and left-joined to `sender_chat`. One window pass replaces the planned two-level GROUP BY and lets list, count, search, due snoozes, cleanup and `ListNewMail` share the mapping. Each chat row also carries the JSON list of its thread keys, so `fillLastFromMe` matches Sent mail through `json_each` without hitting SQLite's variable limit.
- 2026-10-09 (M1): Sender chat keys are matched with a case-sensitive `substr` prefix test (`isSenderKey`), not `LIKE`, which ignores case. Emails are normalized once by `NormalizeSenderEmail`, which the binding also uses before its lookups.
- 2026-10-09 (M1): `SetSenderChat` undo restores the flag and, after a split, the sender chat's previous pin/snooze state. The chat-draft link is released (draft kept), not restored on undo.

## Outcomes & Retrospective

- M1 (2026-10-09): migration 48, the `chatKeyRows` mapping, and sender-chat support in the list, count, search, last-from-me, due-snooze, cleanup and new-mail paths, plus an undoable `SetSenderChat` binding. Covered by 8 table-driven store tests, an undo test and a migration test. Review fixes:
  - a plain thread-key fragment when no sender is combined
  - recipient dedupe in merged search
  - a case-sensitive prefix test
  - email normalization in the binding
  - a shared thread-to-chat-key query

  Declined:
  - SQL-side search merge (logged in Surprises for M3)
  - migrating per-thread state on combine (Decision Log)
  - restoring the draft link on undo (Decision Log)

  The full feature retrospective is written when M3 lands.
