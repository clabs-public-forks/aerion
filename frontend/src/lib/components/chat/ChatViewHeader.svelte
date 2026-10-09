<script lang="ts">
  // ChatViewHeader — the open chat's top bar: Compose, Reply, Archive, Delete
  // and Spam on the left; people and subject centered; Snooze, Pin, Unread and
  // a "⋯" button that opens the shared thread context menu on the right.
  import Icon from '@iconify/svelte'
  import { _ } from '$lib/i18n'
  import Avatar from '$lib/components/kit/Avatar.svelte'
  import ComposeButton from '$lib/components/common/ComposeButton.svelte'
  import ToolbarButton from '$lib/components/common/ToolbarButton.svelte'
  import MessageContextMenu from '$lib/components/common/MessageContextMenu.svelte'
  import { ContextMenuItem } from '$lib/components/ui/context-menu'
  import { contactPhotos } from '$lib/stores/contactPhotos.svelte'
  import { displayName, type Person, type ReplyMode } from './chatFormat'
  import ChatSnoozeMenu from './ChatSnoozeMenu.svelte'
  import { formatSnoozedUntil } from './chatSnooze'

  interface Props {
    subject: string
    people: Person[]
    messageIds: string[]
    accountId: string
    folderId: string
    folderType: string
    isStarred: boolean
    allRead: boolean
    isPinned: boolean
    snoozedUntil: Date | null
    // null when the chat's priority is unknown (not in the loaded list).
    isLowPriority: boolean | null
    canTriage: boolean
    // False when the chat has no reply target yet; disables Reply buttons.
    canReply: boolean
    isTrash: boolean
    isSpam: boolean
    showBackButton: boolean
    onBack?: () => void
    onCompose?: () => void
    onArchive: () => void
    onReplyChat: (replyAll: boolean) => void
    onForward: () => void
    onDelete: () => void
    onSpam: () => void
    onPin: () => void
    onSnooze: (until: Date) => void
    onUnsnooze: () => void
    onToggleRead: () => void
    onToggleSenderLow: () => void
    onActionComplete: (autoSelectNext?: boolean) => void
    onReply?: (mode: ReplyMode, messageId: string) => void
  }

  let {
    subject, people, messageIds, accountId, folderId, folderType, isStarred, allRead, isPinned, snoozedUntil, isLowPriority, canTriage, canReply, isTrash, isSpam,
    showBackButton, onBack, onCompose, onArchive, onReplyChat, onForward, onDelete, onSpam, onPin, onSnooze, onUnsnooze, onToggleRead, onToggleSenderLow, onActionComplete, onReply,
  }: Props = $props()

  let menuAnchor = $state<HTMLElement | null>(null)
  let snoozeMenu = $state<ChatSnoozeMenu | null>(null)
  const COMPACT_WIDTH = 860
  const STACKED_WIDTH = 600
  let width = $state(0)
  // Icon-only toolbar buttons when the pane is too narrow for labels; on
  // phone widths the people move to their own row above the buttons.
  const compact = $derived(width > 0 && width < COMPACT_WIDTH)
  const stacked = $derived(width > 0 && width < STACKED_WIDTH)

  const MAX_AVATARS = 3
  const title = $derived.by(() => {
    if (people.length === 0) return $_('viewer.unknown')
    const names = people.slice(0, 3).map(displayName).join(', ')
    return people.length > 3 ? `${names} ${$_('chat.andMore', { values: { count: people.length - 3 } })}` : names
  })
  const snoozeLabel = $derived(snoozedUntil ? formatSnoozedUntil(snoozedUntil) : '')

  export function openSnooze() { snoozeMenu?.open() }

  // Open the shared context menu (bits-ui ContextMenu) under the "⋯" button.
  export function openMenu() {
    if (!menuAnchor) return
    const r = menuAnchor.getBoundingClientRect()
    menuAnchor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, clientX: r.right, clientY: r.bottom }))
  }
</script>

{#snippet iconButton(icon: string, label: string, onclick: () => void, active?: boolean)}
  <!-- aria-pressed only for toggles (active passed); plain actions omit it. -->
  <button class="p-2 rounded-md hover:bg-muted transition-colors" title={label} aria-label={label} aria-pressed={active} {onclick}>
    <Icon {icon} class="w-5 h-5 {active ? 'text-primary' : 'text-muted-foreground'}" />
  </button>
{/snippet}

{#snippet senderItem()}
  <ContextMenuItem onSelect={onToggleSenderLow}>
    <Icon icon={isLowPriority ? 'mdi:account-arrow-up-outline' : 'mdi:newspaper-variant-outline'} class="mr-2 h-4 w-4" />
    {isLowPriority ? $_('chat.moveSenderToPriority') : $_('chat.moveSenderToLow')}
  </ContextMenuItem>
{/snippet}

<!-- Side columns are at least their content and otherwise equal, so the
     people stay centered on the bar until the buttons need the room. -->
<header
  bind:clientWidth={width}
  class="{stacked ? 'flex flex-wrap justify-between gap-y-2' : 'grid grid-cols-[minmax(max-content,1fr)_minmax(0,auto)_minmax(max-content,1fr)]'} items-center gap-x-3 px-3 py-2 border-b border-border min-h-14"
>
  <div class="flex items-center gap-2">
    {#if showBackButton}
      {@render iconButton('mdi:arrow-left', $_('responsive.back'), () => onBack?.())}
      <div class="w-px h-5 bg-border"></div>
    {/if}
    {#if onCompose}<ComposeButton {compact} onclick={onCompose} />{/if}
    <ToolbarButton icon="mdi:reply" label={$_('viewer.reply')} {compact} disabled={!canReply} onclick={() => onReplyChat(false)} />
    <ToolbarButton icon="mdi:archive-outline" label={$_('viewer.archive')} {compact} onclick={onArchive} />
    <ToolbarButton
      icon={isTrash ? 'mdi:delete-forever' : 'mdi:delete-outline'}
      label={$_('viewer.delete')}
      title={$_(isTrash ? 'viewer.deletePermanently' : 'viewer.delete')}
      {compact}
      onclick={onDelete}
    />
    <ToolbarButton
      icon={isSpam ? 'mdi:email-check-outline' : 'mdi:alert-octagon-outline'}
      label={$_(isSpam ? 'viewer.notSpam' : 'viewer.spam')}
      title={$_(isSpam ? 'viewer.markAsNotSpam' : 'viewer.markAsSpam')}
      {compact}
      onclick={onSpam}
    />
    <div class="w-px h-5 bg-border"></div>
    <div class="flex items-center gap-0.5">
      <button
        class="p-2 rounded-md hover:bg-muted transition-colors disabled:opacity-50 disabled:pointer-events-none"
        title={$_('viewer.replyAll')}
        aria-label={$_('viewer.replyAll')}
        disabled={!canReply}
        onclick={() => onReplyChat(true)}
      >
        <Icon icon="mdi:reply-all" class="w-5 h-5 text-muted-foreground" />
      </button>
      {@render iconButton('mdi:share', $_('viewer.forward'), onForward)}
    </div>
  </div>

  <div class="flex items-center gap-3 min-w-0 {stacked ? 'order-first basis-full' : 'justify-center'}">
    <div class="flex -space-x-2 shrink-0" aria-hidden="true">
      {#each people.slice(0, MAX_AVATARS) as p (p.email)}
        {@const photo = contactPhotos.get(p.email)}
        <div class="rounded-full ring-2 ring-background">
          <Avatar email={p.email} name={p.name} size={32} photoData={photo?.data} photoMediaType={photo?.mediaType} />
        </div>
      {/each}
    </div>

    <div class="min-w-0">
      <h2 class="text-sm font-semibold text-foreground truncate" title={people.map((p) => p.email).join(', ')}>{title}</h2>
      <p class="text-xs text-muted-foreground truncate" title={subject}>
        {#if snoozeLabel}<Icon icon="mdi:alarm-snooze" class="inline w-3.5 h-3.5 -mt-0.5" /> {$_('chat.snoozedUntil', { values: { time: snoozeLabel } })} · {/if}{subject || $_('viewer.noSubject')}
      </p>
    </div>
  </div>

  <div class="flex items-center justify-end gap-0.5">
    {#if canTriage}
      <ChatSnoozeMenu bind:this={snoozeMenu} {snoozedUntil} {onSnooze} {onUnsnooze} />
      {@render iconButton(isPinned ? 'mdi:pin' : 'mdi:pin-outline', isPinned ? $_('chat.unpin') : $_('chat.pin'), onPin, isPinned)}
    {/if}
    {@render iconButton(allRead ? 'mdi:email-outline' : 'mdi:email-open-outline', allRead ? $_('viewer.markAsUnread') : $_('viewer.markAsRead'), onToggleRead)}

    <MessageContextMenu
      {messageIds}
      {accountId}
      currentFolderId={folderId}
      {folderType}
      {isStarred}
      isRead={allRead}
      {onActionComplete}
      {onReply}
      extraItems={canTriage && isLowPriority !== null && people.length > 0 ? senderItem : undefined}
    >
      <span bind:this={menuAnchor} class="inline-flex">
        {@render iconButton('mdi:dots-vertical', $_('chat.moreActions'), openMenu)}
      </span>
    </MessageContextMenu>
  </div>
</header>
