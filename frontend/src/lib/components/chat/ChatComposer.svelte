<script lang="ts">
  // ChatComposer — the reply box docked under a chat: attachments, autosizing
  // text, the reply / reply-all chip and Send. Expand to the full composer
  // lives on the chat's top bar. State and saving live in the ChatComposer
  // controller.
  import Icon from '@iconify/svelte'
  import { _ } from '$lib/i18n'
  import { toasts } from '$lib/stores/toast'
  import { getChatSendKey } from '$lib/stores/settings.svelte'
  import type { ChatComposer } from './chatComposer.svelte'
  import { displayName } from './chatFormat'

  interface Props {
    composer: ChatComposer
    disabled?: boolean
    onEscape?: () => void
  }

  let { composer, disabled = false, onEscape }: Props = $props()

  const MAX_HEIGHT_PX = 200

  let textarea = $state<HTMLTextAreaElement | null>(null)
  const ctrlSends = $derived(getChatSendKey() === 'ctrl-enter')
  const sendHint = $derived(ctrlSends ? $_('chat.sendHintCtrl') : $_('chat.sendHintEnter'))
  const canSend = $derived(!disabled && !composer.isEmpty)

  // Name who the reply goes to, so the reply mode is never a guess.
  const MAX_NAMES = 3
  const MAX_TITLE_EMAILS = 10
  const listed = (items: string[], max: number) =>
    items.length > max ? `${items.slice(0, max).join(', ')} ${$_('chat.andMore', { values: { count: items.length - max } })}` : items.join(', ')
  const people = $derived(composer.recipients)
  const placeholder = $derived(people.length > 0 ? $_('chat.replyTo', { values: { names: listed(people.map(displayName), MAX_NAMES) } }) : $_('chat.typeReply'))
  // The textarea's title doubles as its accessible description.
  const title = $derived(people.length > 0 ? `${listed(people.map((p) => p.email), MAX_TITLE_EMAILS)}\n${sendHint}` : sendHint)

  // Grow with the text up to MAX_HEIGHT_PX, then scroll.
  $effect(() => {
    void composer.text
    if (!textarea) return
    textarea.style.height = 'auto'
    // border-box sizing: scrollHeight excludes the border, so add it back.
    const border = textarea.offsetHeight - textarea.clientHeight
    textarea.style.height = `${Math.min(textarea.scrollHeight + border, MAX_HEIGHT_PX)}px`
  })

  export function focus() {
    textarea?.focus()
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      textarea?.blur()
      onEscape?.()
      return
    }
    if (e.key !== 'Enter' || e.isComposing) return
    const modifier = e.ctrlKey || e.metaKey
    if (ctrlSends ? !modifier : e.shiftKey || modifier) return
    e.preventDefault()
    if (canSend) void composer.send()
  }

  async function attach() {
    try {
      await composer.attach()
    } catch {
      toasts.error($_('chat.attachFailed'))
    }
  }

  function sizeLabel(a: { content_base64?: string; content?: unknown }): string {
    const b64 = a.content_base64 || (typeof a.content === 'string' ? a.content : '')
    const kb = Math.max(1, Math.round((b64.length * 3) / 4 / 1024))
    return kb >= 1024 ? `${(kb / 1024).toFixed(1)} MB` : `${kb} KB`
  }
</script>

<div class="border-t border-border px-3 py-2 bg-background">
  {#if composer.attachments.length > 0}
    <ul class="flex flex-wrap gap-1.5 mb-2" aria-label={$_('chat.attachments')}>
      {#each composer.attachments as att, i (i)}
        <li class="flex items-center gap-1 pl-2 pr-1 py-0.5 rounded-full bg-muted text-xs max-w-60">
          <Icon icon="mdi:paperclip" class="w-3.5 h-3.5 shrink-0 text-muted-foreground" />
          <span class="truncate">{att.filename}</span>
          <span class="text-muted-foreground shrink-0">{sizeLabel(att)}</span>
          <button
            class="p-0.5 rounded-full hover:bg-foreground/10"
            title={$_('chat.removeAttachment')}
            aria-label={$_('chat.removeAttachmentNamed', { values: { name: att.filename } })}
            onclick={() => composer.removeAttachment(i)}
          >
            <Icon icon="mdi:close" class="w-3.5 h-3.5" />
          </button>
        </li>
      {/each}
    </ul>
  {/if}

  <div class="flex items-end gap-1.5">
    <button class="shrink-0 p-1.5 rounded-full hover:bg-muted" title={$_('chat.attach')} aria-label={$_('chat.attach')} {disabled} onclick={attach}>
      <Icon icon="mdi:paperclip" class="w-5 h-5 text-muted-foreground" />
    </button>

    <textarea
      bind:this={textarea}
      class="flex-1 min-w-0 resize-none rounded-2xl border border-border bg-muted/40 px-3.5 py-1.5 text-sm leading-5 focus:outline-none focus:ring-2 focus:ring-primary/40 scrollbar-thin"
      rows="1"
      {placeholder}
      aria-label={$_('viewer.reply')}
      {title}
      spellcheck="true"
      {disabled}
      value={composer.text}
      oninput={(e) => composer.setText(e.currentTarget.value)}
      onkeydown={handleKeyDown}
      onblur={() => void composer.flush()}
    ></textarea>

    <button
      class="shrink-0 h-8 px-3 rounded-full text-xs font-medium border disabled:opacity-50 disabled:pointer-events-none flex items-center gap-1 {composer.replyAll
        ? 'border-primary/60 bg-primary/15 text-primary hover:bg-primary/25'
        : 'border-border hover:bg-muted'}"
      title={$_('chat.replyModeHint')}
      aria-pressed={composer.replyAll}
      {disabled}
      onclick={() => composer.toggleReplyAll()}
    >
      <Icon icon={composer.replyAll ? 'mdi:reply-all' : 'mdi:reply'} class="w-4 h-4" />
      {composer.replyAll ? $_('viewer.replyAll') : $_('viewer.reply')}
    </button>

    <!-- The wrapper carries the tooltip, which a disabled button may not show. -->
    <span class="shrink-0" title={`${$_('chat.send')} (${sendHint})`}>
      <button
        class="h-8 w-8 rounded-full border flex items-center justify-center transition-colors border-primary bg-primary text-primary-foreground enabled:hover:bg-primary/90 disabled:border-border disabled:bg-transparent disabled:text-muted-foreground disabled:cursor-not-allowed"
        aria-label={$_('chat.send')}
        disabled={!canSend}
        onclick={() => void composer.send()}
      >
        <Icon icon="mdi:send" class="w-4 h-4" />
      </button>
    </span>
  </div>
</div>
