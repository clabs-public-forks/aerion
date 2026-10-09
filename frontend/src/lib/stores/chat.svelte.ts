// Chat list state: one paged stream of chats (or search results) for the
// current scope and filter, refreshed on sync and chat-state events.
// Scope (folder selection) stays in App; ChatList passes it in via setScope.

import { GetChats, GetChatCount, SearchConversations, SearchUnifiedInbox, GetSearchCount, GetSearchCountUnifiedInbox } from '../../../wailsjs/go/app/App'
import { message } from '../../../wailsjs/go/models'
// @ts-ignore - wailsjs runtime
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import { accountStore } from './accounts.svelte'
import { isDialogGuardActive } from './dialogGuard'

export type ChatFilter = 'all' | 'unread' | 'low' | 'snoozed'

// Search results carry no pin/snooze/low state, so only these filters apply.
export function searchSupports(filter: ChatFilter): boolean {
  return filter === 'all' || filter === 'unread'
}

export interface ChatPerson {
  name: string
  email: string
}

// ChatItem is the row model for both chats and search results.
export interface ChatItem {
  key: string // accountId|threadId; unique across the unified inbox
  threadId: string
  threadKey: string
  accountId: string
  accountName: string
  accountColor: string
  folderId: string
  subject: string
  snippet: string
  latestDate: Date
  participants: ChatPerson[]
  recipients: ChatPerson[]
  messageIds: string[]
  messageCount: number
  unreadCount: number
  hasAttachments: boolean
  isStarred: boolean
  isEncrypted: boolean
  isPinned: boolean
  snoozedUntil: Date | null
  isLowPriority: boolean
  lastFromMe: boolean
  highlightedSubject?: string
  highlightedSnippet?: string
}

const PAGE_SIZE = 50
const RELOAD_DELAY_MS = 300

function threadKeyOf(threadId: string): string {
  return threadId.replace(/[<>]/g, '')
}

// Sync stores snippets with HTML entities (&amp;) left in. Escaping "<"
// first keeps literal "<tag>" text; DOMParser documents are inert.
let entityParser: DOMParser | undefined
function decodeEntities(text: string): string {
  if (!text.includes('&')) return text
  entityParser ??= new DOMParser()
  return entityParser.parseFromString(text.replaceAll('<', '&lt;'), 'text/html').documentElement.textContent || ''
}

function toItem(c: message.Chat | message.ConversationSearchResult, fallbackAccountId: string, fallbackFolderId: string): ChatItem {
  const chat = c as Partial<message.Chat>
  const search = c as Partial<message.ConversationSearchResult>
  const accountId = c.accountId || fallbackAccountId
  return {
    key: `${accountId}|${c.threadId}`,
    threadId: c.threadId,
    threadKey: chat.threadKey || threadKeyOf(c.threadId),
    accountId,
    accountName: c.accountName || '',
    accountColor: c.accountColor || '',
    folderId: c.folderId || fallbackFolderId,
    subject: c.subject || '',
    snippet: decodeEntities(c.snippet || ''),
    latestDate: new Date(c.latestDate),
    participants: (c.participants || []).map((p) => ({ name: p.name || '', email: p.email || '' })),
    recipients: (chat.recipients || []).map((p) => ({ name: p.name || '', email: p.email || '' })),
    messageIds: c.messageIds || [],
    messageCount: c.messageCount || 0,
    unreadCount: c.unreadCount || 0,
    hasAttachments: !!c.hasAttachments,
    isStarred: !!c.isStarred,
    isEncrypted: !!c.isEncrypted,
    isPinned: !!chat.isPinned,
    snoozedUntil: chat.snoozedUntil ? new Date(chat.snoozedUntil) : null,
    isLowPriority: !!chat.isLowPriority,
    lastFromMe: !!chat.lastFromMe,
    highlightedSubject: search.highlightedSubject || undefined,
    highlightedSnippet: search.highlightedSnippet || undefined,
  }
}

function isInboxFolder(accountId: string, folderId: string): boolean {
  const acct = accountStore.accounts.find((a) => a.account.id === accountId)
  const stack = [...(acct?.folders || [])]
  while (stack.length > 0) {
    const tree = stack.pop()!
    if (tree.folder?.id === folderId) return tree.folder.type === 'inbox'
    stack.push(...(tree.children || []))
  }
  return false
}

class ChatListStore {
  accountId = $state<string | null>(null)
  folderId = $state<string | null>(null)
  filter = $state<ChatFilter>('all')
  query = $state('')
  items = $state<ChatItem[]>([])
  total = $state(0)
  lowCount = $state(0)
  loading = $state(false)
  error = $state<string | null>(null)
  lowExpanded = $state(false)
  // Settings → Chat: when off, All loads only priority chats (no Low group).
  showLowGroup = $state(true)

  // Bumped on every reload; responses from older requests are dropped.
  private generation = 0
  private reloadTimer: ReturnType<typeof setTimeout> | null = null
  // Set while a reload waits for a dialog to close; holds its keepWindow.
  private pendingKeepWindow: boolean | null = null

  readonly isUnified = $derived(this.accountId === 'unified' && this.folderId === 'inbox')
  readonly isSearch = $derived(this.query.trim().length > 0)
  readonly hasMore = $derived(this.items.length < this.total)
  readonly effectiveFilter = $derived<ChatFilter>(this.isSearch && !searchSupports(this.filter) ? 'all' : this.filter)

  // GetChats scope: "" for the unified inbox, otherwise the folder ID.
  private get scopeId(): string {
    return this.isUnified ? '' : this.folderId || ''
  }

  // GetChats section for the current filter.
  private get section(): string {
    return this.filter === 'all' && !this.showLowGroup ? 'priority' : this.filter
  }

  // lowGroupChats loads every chat of the collapsed Low priority group (low,
  // not pinned) in the current scope, for its bulk actions.
  async lowGroupChats(): Promise<ChatItem[]> {
    const accountId = this.accountId || ''
    const folderId = this.folderId || ''
    const count = await GetChatCount(this.scopeId, 'low')
    if (count === 0) return []
    const chats = await GetChats(this.scopeId, 'low', 0, count)
    return (chats || []).map((c) => toItem(c, accountId, folderId)).filter((c) => !c.isPinned)
  }

  setScope(accountId: string | null, folderId: string | null) {
    if (accountId === this.accountId && folderId === this.folderId) return
    this.accountId = accountId
    this.folderId = folderId
    this.clear()
    void this.reload()
  }

  setFilter(filter: ChatFilter) {
    if (filter === this.filter) return
    this.filter = filter
    this.clear()
    void this.reload()
  }

  setShowLowGroup(show: boolean) {
    if (show === this.showLowGroup) return
    this.showLowGroup = show
    if (this.filter !== 'all') return
    this.clear()
    void this.reload()
  }

  setQuery(query: string) {
    this.query = query
    // Drop in-flight pages for the old query; loadMore waits for the reload.
    this.invalidate()
    this.scheduleReload(false)
  }

  // invalidate drops the results of in-flight requests.
  private invalidate() {
    this.generation++
    this.loading = false
  }

  private clear() {
    this.invalidate()
    this.items = []
    this.total = 0
    this.error = null
  }

  private async fetchPage(offset: number, limit: number): Promise<{ items: ChatItem[]; total: number }> {
    const accountId = this.accountId || ''
    const folderId = this.folderId || ''
    if (this.isSearch) {
      const q = this.query.trim()
      const mode = this.effectiveFilter === 'unread' ? 'unread' : ''
      const [results, total] = await Promise.all(this.isUnified
        ? [SearchUnifiedInbox(q, offset, limit, mode), GetSearchCountUnifiedInbox(q, mode)] as const
        : [SearchConversations(accountId, folderId, q, offset, limit, mode), GetSearchCount(accountId, folderId, q, mode)] as const)
      return { items: (results || []).map((r) => toItem(r, accountId, folderId)), total }
    }
    const [chats, total] = await Promise.all([GetChats(this.scopeId, this.section, offset, limit), GetChatCount(this.scopeId, this.section)])
    return { items: (chats || []).map((c) => toItem(c, accountId, folderId)), total }
  }

  // reload fetches the first page again. keepWindow keeps as many rows as
  // are loaded now, so a sync refresh doesn't collapse the scrolled list.
  async reload(keepWindow = false): Promise<void> {
    if (!this.accountId || !this.folderId) return
    const gen = ++this.generation
    const limit = keepWindow ? Math.max(this.items.length, PAGE_SIZE) : PAGE_SIZE
    this.loading = true
    this.error = null
    try {
      const wantLow = this.section === 'all' && !this.isSearch
      const [page, lowCount] = await Promise.all([
        this.fetchPage(0, limit),
        wantLow ? GetChatCount(this.scopeId, 'low') : Promise.resolve(0),
      ])
      if (gen !== this.generation) return
      this.items = page.items
      this.total = page.total
      this.lowCount = lowCount
    } catch (err) {
      if (gen !== this.generation) return
      console.error('Failed to load chats:', err)
      this.error = String(err)
    } finally {
      if (gen === this.generation) this.loading = false
    }
  }

  async loadMore(): Promise<void> {
    if (this.loading || !this.hasMore || this.error || this.reloadTimer) return
    const gen = this.generation
    this.loading = true
    try {
      const page = await this.fetchPage(this.items.length, PAGE_SIZE)
      if (gen !== this.generation) return
      const seen = new Set(this.items.map((c) => c.key))
      this.items = [...this.items, ...page.items.filter((c) => !seen.has(c.key))]
      this.total = page.total
    } catch (err) {
      if (gen !== this.generation) return
      console.error('Failed to load more chats:', err)
      this.error = String(err)
    } finally {
      if (gen === this.generation) this.loading = false
    }
  }

  // scheduleReload coalesces bursts of events into one reload, and waits
  // while a dialog (e.g. the folder picker) is open.
  scheduleReload(keepWindow = true) {
    if (this.reloadTimer) clearTimeout(this.reloadTimer)
    this.reloadTimer = setTimeout(() => {
      this.reloadTimer = null
      if (isDialogGuardActive()) {
        this.pendingKeepWindow = (this.pendingKeepWindow ?? true) && keepWindow
        return
      }
      void this.reload(keepWindow)
    }, RELOAD_DELAY_MS)
  }

  private inScope(accountId: string, folderId?: string): boolean {
    if (this.isUnified) return folderId === undefined || isInboxFolder(accountId, folderId)
    return accountId === this.accountId && (folderId === undefined || folderId === this.folderId)
  }

  private applyReadChange(messageIds: string[], isRead: boolean) {
    const changed = new Set(messageIds)
    for (const c of this.items) {
      let n = 0
      for (const id of c.messageIds) if (changed.has(id)) n++
      if (n === 0) continue
      c.unreadCount = Math.max(0, c.unreadCount + (isRead ? -n : n))
    }
    // Rows read under the Unread filter stay until the next reload, so the
    // chat being read doesn't vanish from under the selection.
  }

  // start subscribes to backend events; returns the unsubscribe function.
  start(): () => void {
    const offs = [
      EventsOn('folder:synced', (d: { accountId: string; folderId: string }) => {
        if (this.inScope(d.accountId, d.folderId)) this.scheduleReload()
      }),
      EventsOn('messages:updated', (d: { accountId: string; folderId: string }) => {
        if (this.inScope(d.accountId, d.folderId)) this.scheduleReload()
      }),
      EventsOn('chats:changed', (d: { accountId: string }) => {
        if (this.inScope(d.accountId)) this.scheduleReload()
      }),
      EventsOn('messages:readChanged', (d: { messageIds: string[]; isRead: boolean }) => {
        this.applyReadChange(d.messageIds || [], d.isRead)
      }),
    ]
    const guardTimer = setInterval(() => {
      if (this.pendingKeepWindow !== null && !isDialogGuardActive()) {
        const keepWindow = this.pendingKeepWindow
        this.pendingKeepWindow = null
        void this.reload(keepWindow)
      }
    }, 500)
    return () => {
      offs.forEach((off) => off())
      clearInterval(guardTimer)
      if (this.reloadTimer) clearTimeout(this.reloadTimer)
      this.reloadTimer = null
    }
  }
}

export const chatList = new ChatListStore()
