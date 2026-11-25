// Everything this browser remembers. No account needed: the edit token *is* the key.
const read = <T>(k: string, d: T): T => {
  try { return JSON.parse(localStorage.getItem(k) ?? '') ?? d } catch { return d }
}
const write = (k: string, v: unknown) => { try { localStorage.setItem(k, JSON.stringify(v)) } catch { /* private mode */ } }

const OWNED = 'wishlist_lists', VISITED = 'wishlist_visited', RESERVED = 'wishlist_reserved'

export interface OwnedRef { edit: string; share: string }
export const ownedLists = (): OwnedRef[] => read<OwnedRef[]>(OWNED, [])
export const rememberOwned = (edit: string, share: string) =>
  write(OWNED, [{ edit, share }, ...ownedLists().filter(l => l.edit !== edit)])
export const forgetOwned = (edits: string[]) => write(OWNED, ownedLists().filter(l => !edits.includes(l.edit)))
export const isOwnShare = (share: string) => ownedLists().some(l => l.share === share)

export interface VisitedRef { share: string; title: string; preset: string; at: number }
export const visitedLists = (): VisitedRef[] => read<VisitedRef[]>(VISITED, [])
export const rememberVisit = (v: Omit<VisitedRef, 'at'>) =>
  write(VISITED, [{ ...v, at: Date.now() }, ...visitedLists().filter(x => x.share !== v.share)].slice(0, 30))
export const forgetVisit = (share: string) => write(VISITED, visitedLists().filter(x => x.share !== share))

/** itemId → reservation key (proves "I reserved this" when undoing). */
export const reservations = (): Record<string, string> => read(RESERVED, {})
export const clearGuestData = () => { write(RESERVED, {}); write(VISITED, []) }
export const setReservation = (item: string, key: string | null) => {
  const r = reservations()
  if (key) r[item] = key; else delete r[item]
  write(RESERVED, r)
}
