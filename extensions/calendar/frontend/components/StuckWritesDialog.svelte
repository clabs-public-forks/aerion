<script lang="ts">
  // StuckWritesDialog — lists calendar writes that keep failing on the
  // server (retry budget used up) with per-row Retry / Discard. Discard
  // asks for an inline confirmation because it drops the user's change.

  import { _ } from 'svelte-i18n'
  import * as Dialog from '$lib/components/ui/dialog'
  import { Button } from '$lib/components/ui/button'
  import { toasts } from '$lib/stores/toast'
  import { dialogGuardOpen, dialogGuardClose } from '$lib/stores/dialogGuard'
  import { stuckWrites } from '$extensions/calendar/frontend/stores/stuckWrites.svelte'

  interface Props {
    open: boolean
  }

  let { open = $bindable(false) }: Props = $props()

  let busyId = $state('')
  let confirmingId = $state('')

  $effect(() => {
    if (!open) return
    dialogGuardOpen()
    return () => dialogGuardClose()
  })

  // Close once the last row is resolved.
  $effect(() => {
    if (open && stuckWrites.items.length === 0 && busyId === '') open = false
  })

  function opLabel(op: string): string {
    return $_(`calendar.stuckWrites.op.${op}`, { default: op })
  }

  function formatTime(unix: number): string {
    return unix > 0 ? new Date(unix * 1000).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }) : ''
  }

  async function run<T>(id: string, action: (id: string) => Promise<T>, onDone: (result: T) => void) {
    if (busyId !== '') return
    busyId = id
    confirmingId = ''
    try {
      onDone(await action(id))
    } catch (err) {
      toasts.error((err as Error)?.message ?? String(err))
    } finally {
      busyId = ''
    }
  }

  function retry(id: string) {
    void run(id, stuckWrites.retry, (status) => {
      if (status === 'synced') toasts.success($_('calendar.stuckWrites.retried'))
      else if (status === 'failed') toasts.error($_('calendar.stuckWrites.stillFailing'))
      else toasts.info($_('calendar.stuckWrites.pending'))
    })
  }

  function discard(id: string) {
    void run(id, stuckWrites.discard, () => toasts.success($_('calendar.stuckWrites.discarded')))
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="max-w-lg">
    <Dialog.Header>
      <Dialog.Title>{$_('calendar.stuckWrites.title')}</Dialog.Title>
      <Dialog.Description>{$_('calendar.stuckWrites.description')}</Dialog.Description>
    </Dialog.Header>

    <ul class="mt-2 max-h-[60vh] overflow-y-auto divide-y divide-border">
      {#each stuckWrites.items as w (w.id)}
        <li class="py-3 space-y-1">
          <div class="flex items-baseline gap-2 min-w-0">
            <span class="font-medium truncate">{w.summary || $_('calendar.stuckWrites.untitled')}</span>
            <span class="text-xs text-muted-foreground shrink-0">{opLabel(w.op)}</span>
          </div>
          <div class="text-xs text-muted-foreground">
            {w.sourceName}{#if w.dtstartUnix > 0}{' · '}{formatTime(w.dtstartUnix)}{/if}
          </div>
          {#if w.lastError}
            <p class="text-xs text-destructive wrap-break-word">{w.lastError}</p>
          {/if}
          {#if w.lastAttemptUnix > 0}
            <div class="text-xs text-muted-foreground">
              {$_('calendar.stuckWrites.lastAttempt', { values: { time: formatTime(w.lastAttemptUnix) } })}
            </div>
          {/if}
          <div class="flex justify-end gap-2 pt-1">
            {#if confirmingId === w.id}
              <span class="text-xs text-muted-foreground self-center mr-auto">{$_('calendar.stuckWrites.confirmDiscard')}</span>
              <Button variant="outline" size="sm" onclick={() => { confirmingId = '' }}>
                {$_('calendar.common.cancel')}
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onclick={() => discard(w.id)}
              >
                {$_('calendar.stuckWrites.discard')}
              </Button>
            {:else}
              <Button
                variant="outline"
                size="sm"
                disabled={busyId !== ''}
                onclick={() => { confirmingId = w.id }}
              >
                {$_('calendar.stuckWrites.discard')}
              </Button>
              <Button
                size="sm"
                disabled={busyId !== ''}
                onclick={() => retry(w.id)}
              >
                {busyId === w.id ? $_('calendar.stuckWrites.working') : $_('calendar.stuckWrites.retry')}
              </Button>
            {/if}
          </div>
        </li>
      {/each}
    </ul>
  </Dialog.Content>
</Dialog.Root>
