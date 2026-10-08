// Stuck calendar writes: queued writes that keep failing with non-transport
// errors and have used up their retry budget (see pending_writes.go). The
// sidebar footer shows a count; StuckWritesDialog lists them with
// retry / discard actions.

// @ts-ignore - wailsjs bindings
import {
  Calendar_ListStuckWrites,
  Calendar_RetryStuckWrite,
  Calendar_DiscardStuckWrite,
} from '$wailsjs/go/app/App.js'
// @ts-ignore - wailsjs bindings
import { EventsOn } from '$wailsjs/runtime/runtime.js'
// @ts-ignore - wailsjs bindings
import type { backend } from '$wailsjs/go/models'
import { logger } from '$extensions/calendar/frontend/lib/logger'

let items = $state<backend.StuckWrite[]>([])
let subscribed = false

async function load(): Promise<void> {
  try {
    items = (await Calendar_ListStuckWrites()) ?? []
  } catch (err) {
    logger.warn(`list stuck writes failed: ${err}`)
  }
}

// Every drain runs at the end of a sync pass, so a finished pass (clean or
// failed) is when rows can become stuck or clear.
function init(): void {
  void load()
  if (subscribed) return
  subscribed = true
  EventsOn('calendar:sync-complete', () => void load())
  EventsOn('calendar:source-error', () => void load())
}

export type RetryStatus = 'synced' | 'failed' | 'pending'

async function retry(id: string): Promise<RetryStatus> {
  try {
    return (await Calendar_RetryStuckWrite(id)) as RetryStatus
  } finally {
    await load()
  }
}

async function discard(id: string): Promise<void> {
  try {
    await Calendar_DiscardStuckWrite(id)
  } finally {
    await load()
  }
}

export const stuckWrites = {
  get items() { return items },
  init,
  retry,
  discard,
}
