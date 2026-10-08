<script lang="ts">
  // Labelled icon button for primary toolbar actions (Compose, Reply, Archive…).
  // `primary` uses the accent colours; otherwise a bordered neutral style.
  import Icon from '@iconify/svelte'

  interface Props {
    icon: string
    label: string
    title?: string
    primary?: boolean
    onclick: () => void
  }

  let { icon, label, title = label, primary = false, onclick }: Props = $props()
</script>

<button
  class="flex items-center gap-2 px-3 py-2 rounded-md text-sm font-medium whitespace-nowrap flex-shrink-0 transition-colors {primary
    ? 'border border-transparent bg-primary text-primary-foreground hover:bg-primary/90'
    : 'border border-border text-foreground hover:bg-muted'}"
  {title}
  {onclick}
>
  <Icon {icon} class="w-4 h-4" />
  <span>{label}</span>
</button>
