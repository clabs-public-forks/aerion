<script lang="ts">
  // ChatDaySeparator — centered day label between messages of different days.
  import { formatDay } from './chatFormat'

  let { date }: { date: Date } = $props()
</script>

<div class="flex items-center gap-3 py-1.5 text-xs text-muted-foreground" role="separator">
  <div class="flex-1 h-px bg-border"></div>
  <span class="px-2 py-0.5 rounded-full bg-muted font-medium">{formatDay(date)}</span>
  <div class="flex-1 h-px bg-border"></div>
</div>
