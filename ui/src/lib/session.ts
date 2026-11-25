import { api, pb, type LocalState } from './api'
import { clearGuestData, forgetOwned, ownedLists, reservations, visitedLists } from './local'

/** Everything this browser knows — the proof needed to claim it for an account. */
export const localState = (): LocalState => ({
  tokens: ownedLists().map(l => l.edit),
  reservations: reservations(),
  visits: visitedLists().map(v => v.share),
})

/** Move lists, reservations and visited lists of this browser into the signed-in account. */
export async function claimLocal() {
  if (!pb.authStore.isValid) return null
  const s = localState()
  if (!s.tokens.length && !Object.keys(s.reservations).length && !s.visits.length) return null
  return api.claim(s)
}

/** Sign out; whatever is safe in the account is removed from this device. */
export async function signOut() {
  try {
    const { lists } = await api.dashboard({ tokens: ownedLists().map(l => l.edit), reservations: {}, visits: [] })
    forgetOwned(lists.filter(l => l.mine && l.edit).map(l => l.edit!))
    clearGuestData() // reservations & visits were claimed on sign-in
  } catch { /* offline — keep everything */ }
  pb.authStore.clear()
}
