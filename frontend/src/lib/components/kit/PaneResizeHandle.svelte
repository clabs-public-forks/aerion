<script lang="ts">
  // PaneResizeHandle — vertical drag handle placed after a fixed-width pane.
  // Mail's App.svelte and the kit panes (SidebarFrame, ContactList) share it so
  // every column boundary drags the same way. The new width is derived from the
  // drag delta (startWidth + dx), so the pane's position on screen (extension
  // rail, collapsed sidebar) never skews it. Resized panes must be
  // flex-shrink-0 so their rendered width matches `width`.

  import { clamp, paneConstraints, type PaneKind } from '$lib/stores/uiState.svelte'

  interface Props {
    /** Current pane width in px. */
    width: number
    /** Selects the min/max bounds from paneConstraints. */
    kind: PaneKind
    /** ARIA label for the handle button. */
    label: string
    /** Fired on every pointer move while dragging with the clamped width. */
    onresize: (width: number) => void
    /** Fired once when the drag ends, e.g. to persist the final width. */
    oncommit?: (width: number) => void
  }

  const { width, kind, label, onresize, oncommit }: Props = $props()

  let dragging = $state(false)
  let startX = 0
  let startWidth = 0
  let lastWidth = 0

  // Pointer capture keeps move/up events coming to the handle even when the
  // pointer leaves the window, and lostpointercapture ends a drag that the
  // system interrupts, so the drag can't get stuck.
  function start(e: PointerEvent) {
    if (e.button !== 0) return
    startX = e.clientX
    startWidth = lastWidth = width
    dragging = true
    const handle = e.currentTarget as HTMLElement
    handle.setPointerCapture(e.pointerId)
    e.preventDefault()
  }

  function move(e: PointerEvent) {
    if (!dragging) return
    const { min, max } = paneConstraints[kind]
    lastWidth = clamp(startWidth + e.clientX - startX, min, max)
    onresize(lastWidth)
  }

  function end() {
    if (!dragging) return
    dragging = false
    oncommit?.(lastWidth)
  }
</script>

<button
  type="button"
  class="w-1 flex-shrink-0 cursor-col-resize hover:bg-primary/20 active:bg-primary/40 transition-colors border-0 p-0 {dragging
    ? 'bg-primary/40'
    : ''}"
  onpointerdown={start}
  onpointermove={move}
  onpointerup={end}
  onlostpointercapture={end}
  aria-label={label}
></button>

<!-- Keeps the resize cursor and blocks hover effects while dragging over other panes. -->
{#if dragging}
  <div class="fixed inset-0 cursor-col-resize z-50"></div>
{/if}
