<script lang="ts">
  // ChatSnoozeDateDialog — "Pick date..." from the snooze menu: a date and
  // time input, defaulting to tomorrow morning.
  import { addDays, format, setHours, startOfHour } from 'date-fns'
  import { _ } from '$lib/i18n'
  import * as Dialog from '$lib/components/ui/dialog'
  import { Button } from '$lib/components/ui/button'
  import { dialogGuardOpen, dialogGuardClose } from '$lib/stores/dialogGuard'

  interface Props {
    open: boolean
    onPick: (until: Date) => void
  }

  let { open = $bindable(false), onPick }: Props = $props()

  const INPUT_FORMAT = "yyyy-MM-dd'T'HH:mm"
  let value = $state('')
  let error = $state('')

  // Hold the dialog guard while open so App's shortcuts leave the input alone.
  $effect(() => {
    if (!open) return
    dialogGuardOpen()
    value = format(startOfHour(setHours(addDays(new Date(), 1), 8)), INPUT_FORMAT)
    error = ''
    return () => dialogGuardClose()
  })

  function submit(e: SubmitEvent) {
    e.preventDefault()
    const until = new Date(value)
    if (isNaN(until.getTime()) || until.getTime() <= Date.now()) {
      error = $_('chat.snoozeInPast')
      return
    }
    open = false
    onPick(until)
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="max-w-sm">
    <form class="grid gap-4" onsubmit={submit}>
      <Dialog.Header>
        <Dialog.Title>{$_('chat.snoozePickTitle')}</Dialog.Title>
        <Dialog.Description>{$_('chat.snoozePickDescription')}</Dialog.Description>
      </Dialog.Header>
      <input
        type="datetime-local"
        class="h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm"
        aria-label={$_('chat.snoozePickTitle')}
        aria-invalid={!!error}
        aria-describedby={error ? 'chat-snooze-error' : undefined}
        bind:value
      />
      {#if error}<p id="chat-snooze-error" class="text-sm text-destructive">{error}</p>{/if}
      <Dialog.Footer>
        <Button type="button" variant="outline" onclick={() => (open = false)}>{$_('common.cancel')}</Button>
        <Button type="submit">{$_('chat.snooze')}</Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
