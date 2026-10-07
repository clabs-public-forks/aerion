---
sidebar_position: 2
---

# Email Management

Aerion provides essential tools to manage your email efficiently.

## Unified Inbox

The Unified Inbox shows all incoming mail from all your accounts in one view.

- Each account is assigned a color for easy identification
- Messages are sorted by date across all accounts
- Click on an individual account's inbox to view only that account

### Account Colors

Set account colors in **Settings > Accounts > Edit Account**. The color appears as a subtle indicator on each message in the unified inbox.

## Folder Navigation

The sidebar shows all folders for each account:

- **Unified Inbox** - All inboxes combined
- **Individual Account Inboxes** - Nested under Unified Inbox
- **Account Folders** - Each account's folder tree

### Navigating Folders

- Click a folder to view its contents
- Click an account header to expand/collapse its folders
- Use `Alt+Up/Down` or `Alt+J/K` to navigate folders with keyboard

## Conversation View

Related messages are grouped into conversations (threads):

- Open a conversation to see all messages in the thread
- Messages are displayed chronologically
- Reply context is preserved for easier reading

Delete single message in a conversation:

- Right click on the header of a message and select delete from the context menu
- Focus (Alt+L or ALt+Right) on the conversation viewer, press tab to navigate to the message and press delete.

Focus mode:

Any thread or specific message can be toggled into focus mode which stretches the thread or message across the full size of the window. This is particularly useful if you would like to share your screen in an online meeting without exposing your folders and message list. It's of course also good for those who just want to focus on an e-mail and not be distracted by other elements of the app. While in focus mode, reply, reply all, and forward actions will launch a detached composer instead of the default composer. To reply, reply all, or forward with the default in window composer, exit focus mode before you toggle the action.

To toggle a thread into focus mode, click the focus mode icon in the top right corner of the conversation viewer next to the print icon.

To toggle a message into focus mode, click the focus mode icon in the top right corner of the message's header.

You can also toggle a thread into focus mode by just pressing **f** and toggle a message you are focused on into focus mode by pressing **Shift+f**.


## Message Actions

### Single Message Actions

When viewing a message or with a message selected:

| Action | Method |
|--------|--------|
| Reply | Click Reply button or `Ctrl+R` |
| Reply All | Click Reply All button or `Ctrl+Shift+R` |
| Forward | Click Forward button or `Ctrl+F` |
| Star/Unstar | Click star icon or `S` |
| Mark Read | Right-click menu or `Ctrl+U` |
| Mark Unread | Right-click menu or `Ctrl+Shift+U` |
| Archive | Click Archive button or `Ctrl+K` |
| Move to Trash | Click Trash button or `Delete` |
| Mark as Spam | Click Spam button or `Ctrl+J` |
| Copy to Folder | Right-click > Copy to or Right Alt > Copy to or `Alt+C` |
| Move to Folder | Right-click > Move to or Right Alt > Move to or `Alt+M` |
| Thread Focus Mode | `F` |
| Message Focus Mode | `Shift+F` |

### Copy to Folder and Move to Folder

Toggling either copy or move to folder will bring up a dialog with a scrollable list of folders to choose as destination. There's also a search bar on top of the folder list so that users can use it to find the target folder in the case that the account has a substantial amount of folders.

### Bulk Actions

Select multiple messages to apply actions in bulk:

1. Check the checkbox on each message, or
2. Use `Space` to toggle the checkbox on the focused message, or
3. Use `Shift+Up/Down` or `Shift+J/K` to select while navigating

Then apply any action - it will apply to all selected messages.

**Tip:** Press `Escape` once to clear all checkboxes.

## Swipe Gestures

Aerion has some basic swipe gestures:

- Right: select message
- Left: delete message

For laptop trackpads, swipe with 2 fingers.
For mobile layout, swipe with 1 finger.

## Search

Find emails quickly with the search bar:

1. Click the search bar or press `Ctrl+S`
2. Type your search query
3. Press `Enter` to switch focus to results

Search looks through:
- Subject lines
- Sender and recipient addresses
- Message body content

Search results are highlighted to show matching terms.

### Search Scope

- Search in the currently selected folder, or
- Search across all folders (depending on your email server's capabilities)

### Server Side Search

If the search results don't yield what you are looking for, you can perform a more comprehensive server side IMAP search by clicking the **Search Server** link located on the bar above the first search result or at the center of the message list pane when basic search yields no results. This searches all messages on the server, including older messages that haven't been downloaded locally. It is slower but much more comprehensive. By default, server searches return a maximum of 200 results. If this still does not return what you are looking for, scroll down to the bar below the last search result and click the **Load More** link to return all results.

Alternatively, if you prefer to always use server side search or know a server side search is needed, simply press **Shift+Enter** after typing your search phrase in the search bar.

## Sorting

There are only 2 types of sort. **Newest on top** or **Oldest on top**. You can change between the two by clicking the farthest right button on top of the message list.

## Filtering

If you need to see only messages that are either unread, starred, or with attachments in any given folder, there's a **filter** icon on top of the message list between the **search** and **sort** icons. 

## Sync Options

### Automatic Sync

Aerion periodically checks for new mail. The sync interval can be adjusted in settings.

Aerion also holds IDLE connections for push e-mail. When new e-mail arrive, it triggers a sync of that account's core folders.

### Manual Sync

Select the folder you want to sync, click the sync icon, select **Sync Folder** or press `Ctrl + Shift + S`. Clicking the sync button again or pressing `Ctrl + Shift + S` again will stop the sync.

### Sync All Accounts

Use the sync button in the bottom left corner of the app or press `Ctrl + Shift + A` to sync all accounts at once. Clicking it or press `Ctrl + Shift + A` again while a sync is in progress will stop the sync.

### Force Re-sync

If messages appear to be missing or out of date:

1. Select the folder that seems to be out of sync
2. Click the Sync icon
3. Click **Force Re-sync**

This will re-download messages from the server.

## Message Density

Adjust how much space each message takes in the list:

1. Go to **Settings > General**
2. Choose from:
   - **Micro** - Minimal spacing, more messages visible
   - **Compact** - Reduced spacing
   - **Standard** - Default spacing
   - **Large** - More spacing, easier to read

## Remote Images

For privacy, Aerion blocks remote images in emails by default. These images can be used to track when you open an email.

To load images in a specific message:
- Click **Load Images** or press `Ctrl+L`

To always load images from a sender:
- Click the dropdown arrow next to Load Images or pres `Ctrl-Shift-L`
- Select:
    - **For this domain** - Trusting all e-mails from this domain from now on
    - **For this e-mail address**) - Trusting all e-mails from this e-mail address from now on

## Tracking Element Removal

Aerion automatically removes common tracking elements from emails:
- Tracking pixels (1x1 invisible images)
- Known tracking parameters in links

This helps protect your privacy without requiring any configuration.
