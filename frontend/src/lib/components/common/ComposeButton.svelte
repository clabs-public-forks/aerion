<script lang="ts">
  // Primary "Compose" call to action, shared by the viewer toolbar and the
  // message-list header. Keeps the original sidebar button's colours and label.
  import Icon from '@iconify/svelte'
  import { _ } from '$lib/i18n'

  interface Props {
    onclick: () => void
  }

  let { onclick }: Props = $props()
</script>

<button
  class="flex items-center gap-2 px-3 py-2 bg-primary text-primary-foreground rounded-md text-sm font-medium whitespace-nowrap flex-shrink-0 hover:bg-primary/90 transition-colors"
  title={$_('sidebar.compose')}
  {onclick}
>
  <Icon icon="mdi:pencil" class="w-4 h-4" />
  <span>{$_('sidebar.compose')}</span>
</button>
