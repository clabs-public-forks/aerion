<script lang="ts">
  // Primary "Compose" call to action, shared by the viewer toolbar and the
  // message-list header. Keeps the original sidebar button's colours and label.
  import { _ } from '$lib/i18n'
  import ToolbarButton from './ToolbarButton.svelte'

  interface Props {
    onclick: () => void
    compact?: boolean
  }

  let { onclick, compact = false }: Props = $props()
</script>

<ToolbarButton icon="mdi:pencil" label={$_('sidebar.compose')} primary {compact} {onclick} />
