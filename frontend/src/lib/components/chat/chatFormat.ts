// Pure helpers for the chat thread view: ownership, grouping, day labels,
// recipient parsing, and URL linkification (rendered as text segments, never HTML).

import { get } from 'svelte/store'
import { format, isThisYear, isToday, isYesterday } from 'date-fns'
import { _ } from '$lib/i18n'
import type { message as messageModels, folder } from '../../../../wailsjs/go/models'
import type { AccountWithFolders } from '$lib/stores/accounts.svelte'

export type ReplyMode = 'reply' | 'reply-all' | 'forward'

export interface Person {
  name: string
  email: string
}

// Consecutive messages from one sender within this window share a group.
const GROUP_WINDOW_MS = 5 * 60 * 1000

export function parseRecipients(raw: string | undefined): Person[] {
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.map((r: { name?: string; email?: string }) => ({ name: r.name || '', email: r.email || '' }))
  } catch {
    return []
  }
}

export function displayName(p: Person): string {
  return p.name || p.email.split('@')[0] || p.email
}

// chatPeople lists a chat's participants once each, latest sender first,
// without my own addresses. When I sent every message (Sent folder rows) it
// falls back to the recipients, and to me only for a note to self. allMine
// says every sender is me even when the address is an unknown alias.
export function chatPeople(participants: Person[], myEmails: Set<string>, recipients: Person[] = [], allMine = false): Person[] {
  const seen = new Set<string>()
  const all: Person[] = []
  for (const p of participants) {
    const key = p.email.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    all.push(p)
  }
  const notMe = (p: Person) => !myEmails.has(p.email.toLowerCase())
  const others = allMine ? [] : all.filter(notMe)
  if (others.length > 0) return others
  const to = recipients.filter(notMe)
  return to.length > 0 ? to : all
}

// Identity used to recognize my own messages: account addresses plus each
// account's Sent folders (covers identities and aliases sent from them).
export interface Me {
  emails: Set<string>
  sentFolderIds: Set<string>
}

export function buildMe(accounts: AccountWithFolders[]): Me {
  const emails = new Set<string>()
  const sentFolderIds = new Set<string>()
  const walk = (trees: folder.FolderTree[] | undefined) => {
    for (const t of trees ?? []) {
      if (t.folder?.type === 'sent') sentFolderIds.add(t.folder.id)
      walk(t.children)
    }
  }
  for (const a of accounts) {
    if (a.account.email) emails.add(a.account.email.toLowerCase())
    walk(a.folders)
  }
  return { emails, sentFolderIds }
}

export function isMine(msg: messageModels.Message, me: Me): boolean {
  return me.sentFolderIds.has(msg.folderId) || me.emails.has((msg.fromEmail || '').toLowerCase())
}

// People in the thread other than me, in first-appearance order. Falls back
// to recipients when every message is mine (e.g. a Sent-only thread).
export function threadPeople(messages: messageModels.Message[], me: Me): Person[] {
  const seen = new Set<string>()
  const people: Person[] = []
  const add = (p: Person) => {
    const key = p.email.toLowerCase()
    if (!key || seen.has(key) || me.emails.has(key)) return
    seen.add(key)
    people.push(p)
  }
  for (const m of messages) {
    if (!isMine(m, me)) add({ name: m.fromName || '', email: m.fromEmail || '' })
  }
  if (people.length === 0) {
    for (const m of messages) {
      parseRecipients(m.toList).forEach(add)
      parseRecipients(m.ccList).forEach(add)
    }
  }
  return people
}

// replyTarget picks the message a chat reply answers: the latest message
// from someone else, else my latest one (replying to its recipients). Reply
// all is the default when the answered message involves more than one other
// person. Drafts never count.
export function replyTarget(messages: messageModels.Message[], me: Me): { messageId: string; defaultReplyAll: boolean } | null {
  let theirs: messageModels.Message | null = null
  let last: messageModels.Message | null = null
  for (const m of messages) {
    if (m.isDraft) continue
    last = m
    if (!isMine(m, me)) theirs = m
  }
  if (!theirs) return last ? { messageId: last.id, defaultReplyAll: true } : null
  const others = new Set<string>()
  for (const p of [{ name: '', email: theirs.fromEmail || '' }, ...parseRecipients(theirs.toList), ...parseRecipients(theirs.ccList)]) {
    const email = p.email.toLowerCase()
    if (email && !me.emails.has(email)) others.add(email)
  }
  return { messageId: theirs.id, defaultReplyAll: others.size > 1 }
}

function dayKey(d: Date): string {
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
}

export function formatDay(d: Date): string {
  const t = get(_)
  if (isToday(d)) return t('chat.today')
  if (isYesterday(d)) return t('date.yesterday')
  return format(d, isThisYear(d) ? 'EEE d MMM' : 'EEE d MMM yyyy')
}

export function formatTime(d: Date): string {
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export interface ThreadItem {
  msg: messageModels.Message
  date: Date
  mine: boolean
  // First message of a new day: render a separator before it.
  newDay: boolean
  // First message of a same-sender run: show avatar and name.
  groupStart: boolean
}

export function buildThreadItems(messages: messageModels.Message[], me: Me): ThreadItem[] {
  const items: ThreadItem[] = []
  let prev: ThreadItem | null = null
  for (const msg of messages) {
    const date = new Date(msg.date)
    const mine = isMine(msg, me)
    const newDay = !prev || dayKey(prev.date) !== dayKey(date)
    const sameSender = !!prev && prev.mine === mine && (prev.msg.fromEmail || '').toLowerCase() === (msg.fromEmail || '').toLowerCase()
    const groupStart = newDay || !sameSender || date.getTime() - prev!.date.getTime() > GROUP_WINDOW_MS
    const item: ThreadItem = { msg, date, mine, newDay, groupStart }
    items.push(item)
    prev = item
  }
  return items
}

export interface TextSegment {
  text: string
  url?: string
}

const URL_RE = /\b(?:https?:\/\/|mailto:|www\.)[^\s<>"']+/gi
const TRAILING_PUNCT_RE = /[.,;:!?)\]}'"]+$/

// linkify splits text into plain and link segments. Trailing punctuation
// stays outside the link; a closing ")" is kept when the URL opened one.
export function linkify(text: string): TextSegment[] {
  const out: TextSegment[] = []
  let last = 0
  for (const m of text.matchAll(URL_RE)) {
    let raw = m[0]
    const trail = raw.match(TRAILING_PUNCT_RE)?.[0] ?? ''
    raw = raw.slice(0, raw.length - trail.length)
    let keep = ''
    if (trail.startsWith(')') && raw.includes('(')) keep = ')'
    raw += keep
    const start = m.index!
    if (start > last) out.push({ text: text.slice(last, start) })
    const url = raw.toLowerCase().startsWith('www.') ? `https://${raw}` : raw
    out.push({ text: raw, url })
    last = start + raw.length
  }
  if (last < text.length) out.push({ text: text.slice(last) })
  return out
}
