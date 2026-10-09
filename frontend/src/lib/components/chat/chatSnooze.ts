// Snooze presets shared by the chat header (M4) and the snooze picker (M6).

import { addDays, nextMonday, nextSaturday, isWeekend, setHours, startOfHour, addHours, isSameDay } from 'date-fns'

interface SnoozePreset {
  id: 'laterToday' | 'tomorrow' | 'weekend' | 'nextWeek'
  // i18n key under chat.*
  labelKey: string
  until: Date
}

const MORNING_HOUR = 8
const WEEKEND_HOUR = 9
const LATER_TODAY_HOURS = 3

function at(d: Date, hour: number): Date {
  return startOfHour(setHours(d, hour))
}

// snoozePresets lists the presets that make sense right now: "Later today"
// only when it still lands today, "This weekend" only on weekdays.
export function snoozePresets(now = new Date()): SnoozePreset[] {
  const presets: SnoozePreset[] = []
  const later = startOfHour(addHours(now, LATER_TODAY_HOURS + 1))
  if (isSameDay(later, now)) presets.push({ id: 'laterToday', labelKey: 'chat.snoozeLaterToday', until: later })
  presets.push({ id: 'tomorrow', labelKey: 'chat.snoozeTomorrow', until: at(addDays(now, 1), MORNING_HOUR) })
  if (!isWeekend(now)) presets.push({ id: 'weekend', labelKey: 'chat.snoozeWeekend', until: at(nextSaturday(now), WEEKEND_HOUR) })
  // On Sundays "Next week" would repeat "Tomorrow".
  if (now.getDay() !== 0) presets.push({ id: 'nextWeek', labelKey: 'chat.snoozeNextWeek', until: at(nextMonday(now), MORNING_HOUR) })
  return presets
}

// formatSnoozePreset is the short form shown beside presets and in toasts.
export function formatSnoozePreset(d: Date): string {
  return d.toLocaleString([], { weekday: 'short', hour: '2-digit', minute: '2-digit' })
}

// formatSnoozedUntil is the full form for "Snoozed until …" labels.
export function formatSnoozedUntil(d: Date): string {
  return d.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' })
}
