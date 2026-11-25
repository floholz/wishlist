export type Theme = 'light' | 'dark'
const KEY = 'wishlist_theme'

const system = (): Theme => (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
const stored = (): Theme | null => { try { return localStorage.getItem(KEY) as Theme | null } catch { return null } }
export const current = (): Theme => stored() ?? system()

export function apply(t: Theme | null) {
  try {
    if (t) { document.documentElement.dataset.theme = t; localStorage.setItem(KEY, t) }
    else { delete document.documentElement.dataset.theme; localStorage.removeItem(KEY) }
  } catch { /* ignore */ }
}
export const toggle = () => { const next: Theme = current() === 'dark' ? 'light' : 'dark'; apply(next); return next }

apply(stored())
