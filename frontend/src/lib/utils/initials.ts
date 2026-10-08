/**
 * Up to two uppercase initials from a display name or email. Each
 * space-separated word contributes its first letter or digit, so names
 * like "[VHS] Donna" or "Alanna (Allie)" skip the punctuation. Returns
 * '?' when nothing usable remains.
 */
export function getInitials(name: string | undefined): string {
  const letters = (name ?? '')
    .split(/\s+/)
    .map((word) => word.match(/[\p{L}\p{N}]/u)?.[0] ?? '')
    .join('')
  return letters.slice(0, 2).toUpperCase() || '?'
}
