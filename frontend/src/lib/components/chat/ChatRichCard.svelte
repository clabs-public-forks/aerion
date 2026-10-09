<script lang="ts">
  // ChatRichCard — a newsletter, receipt or other layout-heavy message shown
  // as a compact card; expanding it renders the full sandboxed body.
  import Icon from '@iconify/svelte'
  import type { Snippet } from 'svelte'
  import { _ } from '$lib/i18n'

  interface Props {
    subject: string
    preview: string
    expanded: boolean
    onToggle: () => void
    body: Snippet
  }

  let { subject, preview, expanded, onToggle, body }: Props = $props()

  const PREVIEW_CHARS = 280
  const shortPreview = $derived(preview.length > PREVIEW_CHARS ? `${preview.slice(0, PREVIEW_CHARS).trimEnd()}…` : preview)
</script>

<div class="rounded-xl border border-border bg-card overflow-hidden">
  <button
    class="w-full flex items-start gap-3 p-3 pr-20 text-left hover:bg-muted/50 transition-colors"
    aria-expanded={expanded}
    onclick={onToggle}
  >
    <Icon icon="mdi:newspaper-variant-outline" class="w-5 h-5 mt-0.5 shrink-0 text-muted-foreground" />
    <span class="flex-1 min-w-0">
      <span class="block text-sm font-medium text-foreground truncate">{subject || $_('viewer.noSubject')}</span>
      {#if !expanded && shortPreview}
        <span class="block text-sm text-muted-foreground line-clamp-3 whitespace-pre-line">{shortPreview}</span>
      {/if}
    </span>
    <span class="shrink-0 flex items-center gap-1 text-xs text-primary">
      {expanded ? $_('chat.collapseMessage') : $_('chat.expandMessage')}
      <Icon icon={expanded ? 'mdi:chevron-up' : 'mdi:chevron-down'} class="w-4 h-4" />
    </span>
  </button>
  {#if expanded}
    <div class="border-t border-border">{@render body()}</div>
  {/if}
</div>
