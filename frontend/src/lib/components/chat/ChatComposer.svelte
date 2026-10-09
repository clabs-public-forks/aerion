<script lang="ts">
  // ChatComposer — the reply box docked under a chat: autosizing text,
  // reply / reply-all chip, attachments, Expand to the full composer, Send and
  // Send & Done. State and saving live in the ChatComposer controller.
  import Icon from '@iconify/svelte'
  import { _ } from '$lib/i18n'
  import { toasts } from '$lib/stores/toast'
  import { getChatSendKey } from '$lib/stores/settings.svelte'
  import type { ChatComposer } from './chatComposer.svelte'

  interface Props {
    composer: ChatComposer
    disabled?: boolean
    onExpand: () => void
    onEscape?: () => void
  }

  let { composer, disabled = false, onExpand, onEscape }: Props = $props()

  const MAX_HEIGHT_PX = 200

  let textarea = $state<HTMLTextAreaElement | null>(null)
  const ctrlSends = $derived(getChatSendKey() === 'ctrl-enter')
  const sendHint = $derived(ctrlSends ? $_('chat.sendHintCtrl') : $_('chat.sendHintEnter'))
  const canSend = $derived(!disabled && !composer.isEmpty)

  // Grow with the text up to MAX_HEIGHT_PX, then scroll.
  $effect(() => {
    void composer.text
    if (!textarea) return
    textarea.style.height = 'auto'
    textarea.style.height = `${Math.min(textarea.scrollHeight, MAX_HEIGHT_PX)}px`
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
    if (canSend) void composer.send(false)
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
    <button
      class="shrink-0 h-8 px-2 rounded-full text-xs font-medium border border-border hover:bg-muted flex items-center gap-1"
      title={$_('chat.replyModeHint')}
      aria-pressed={composer.replyAll}
      {disabled}
      onclick={() => composer.toggleReplyAll()}
    >
      <Icon icon={composer.replyAll ? 'mdi:reply-all' : 'mdi:reply'} class="w-4 h-4" />
      {composer.replyAll ? $_('viewer.replyAll') : $_('viewer.reply')}
    </button>

    <button class="shrink-0 p-1.5 rounded-full hover:bg-muted" title={$_('chat.attach')} aria-label={$_('chat.attach')} {disabled} onclick={attach}>
      <Icon icon="mdi:paperclip" class="w-5 h-5 text-muted-foreground" />
    </button>

    <textarea
      bind:this={textarea}
      class="flex-1 min-w-0 resize-none rounded-2xl border border-border bg-muted/40 px-3.5 py-1.5 text-sm leading-5 focus:outline-none focus:ring-2 focus:ring-primary/40 scrollbar-thin"
      rows="1"
      placeholder={$_('chat.typeReply')}
      aria-label={$_('chat.typeReply')}
      title={sendHint}
      spellcheck="true"
      {disabled}
      value={composer.text}
      oninput={(e) => composer.setText(e.currentTarget.value)}
      onkeydown={handleKeyDown}
      onblur={() => void composer.flush()}
    ></textarea>

    <button class="shrink-0 p-1.5 rounded-full hover:bg-muted" title={$_('chat.expand')} aria-label={$_('chat.expand')} {disabled} onclick={onExpand}>
      <Icon icon="mdi:arrow-expand" class="w-5 h-5 text-muted-foreground" />
    </button>

    <button
      class="shrink-0 h-8 px-3 rounded-full text-xs font-medium border border-border hover:bg-muted disabled:opacity-50 disabled:pointer-events-none"
      title={$_('chat.sendAndDoneHint')}
      disabled={!canSend}
      onclick={() => void composer.send(true)}
    >
      {$_('chat.sendAndDone')}
    </button>

    <button
      class="shrink-0 h-8 w-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center hover:bg-primary/90 disabled:opacity-50 disabled:pointer-events-none"
      title={`${$_('chat.send')} (${sendHint})`}
      aria-label={$_('chat.send')}
      disabled={!canSend}
      onclick={() => void composer.send(false)}
    >
      <Icon icon="mdi:send" class="w-4 h-4" />
    </button>
  </div>
</div>
