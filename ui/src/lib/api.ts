import PocketBase from 'pocketbase'
import { hasKey, i18n, t, type Key } from './i18n/index.svelte'

export const pb = new PocketBase(window.location.origin)
pb.autoCancellation(false)

export interface Item {
  id: string; name: string; description: string; links: string[]; photos: string[]; position: number
  linkPhoto: string // picture fetched from the links; only used when there are no own photos
  reserved?: boolean
  reservedByMe?: boolean // reserved by the signed-in account (share view)
}
export interface List {
  title: string; description: string; preset: string; share: string; edit?: string
  owned: boolean; mine: boolean; updated: string; count: number; preview?: string[]; items?: Item[]
}

export interface Gift { item: Item; title: string; preset: string; share: string }
export interface Visit { share: string; title: string; preset: string; synced?: boolean /* kept in the account */ }
/** What this browser knows without an account (sent to dashboard / claim). */
export interface LocalState { tokens: string[]; reservations: Record<string, string>; visits: string[] }

/** Errors carry PocketBase's message so they can be shown as-is. */
export class ApiError extends Error {
  constructor(public status: number, message: string, public code?: string, public params: Record<string, string | number> = {}) { super(message) }
}

/** A translated, user-facing message for any error (API code → messages, else the server's text). */
export function errText(err: any): string {
  const code: string | undefined = err?.code
  if (code && hasKey(`errors.${code}`)) return t(`errors.${code}` as Key, err.params ?? {})
  return err?.message || t('common.error')
}

async function call<T>(method: string, path: string, body?: unknown, headers: Record<string, string> = {}): Promise<T> {
  const h: Record<string, string> = { ...headers }
  if (pb.authStore.isValid) h.Authorization = pb.authStore.token
  let payload: BodyInit | undefined
  if (body instanceof FormData) payload = body
  else if (body !== undefined) { payload = JSON.stringify(body); h['Content-Type'] = 'application/json' }
  let res: Response
  try { res = await fetch('/api/wl' + path, { method, headers: h, body: payload }) }
  catch { throw new ApiError(0, t('common.offline')) }
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const e = data?.data?.error
    throw new ApiError(res.status, data?.message || `Request failed (${res.status})`, e?.code, e?.params)
  }
  return data as T
}

export const api = {
  createList: (preset: string, title: string) => call<List>('POST', '/lists', { preset, title }),
  dashboard: (b: LocalState) => call<{ lists: List[]; missing: string[]; gifts: Gift[]; visited: Visit[] }>('POST', '/dashboard', b),
  claim: (b: LocalState) => call<{ claimed: number; reservations: number; visits: number }>('POST', '/claim', b),

  getEdit: (t: string) => call<List>('GET', `/e/${t}`),
  patchList: (t: string, p: Partial<Pick<List, 'title' | 'description' | 'preset'>>) => call<List>('PATCH', `/e/${t}`, p),
  deleteList: (t: string) => call<void>('DELETE', `/e/${t}`),
  createItem: (t: string, f: FormData) => call<Item>('POST', `/e/${t}/items`, f),
  updateItem: (t: string, id: string, f: FormData) => call<Item>('PATCH', `/e/${t}/items/${id}`, f),
  deleteItem: (t: string, id: string) => call<void>('DELETE', `/e/${t}/items/${id}`),
  reorder: (t: string, ids: string[]) => call<void>('PUT', `/e/${t}/order`, { ids }),
  linkPreview: (t: string, id: string, autoname = false) => call<Item>('POST', `/e/${t}/items/${id}/preview`, { autoname }),

  getShare: (s: string) => call<List>('GET', `/s/${s}`),
  reserve: (s: string, id: string) => call<{ key: string }>('POST', `/s/${s}/items/${id}/reserve`),
  /** Without a key the signed-in account has to be the one that reserved. */
  unreserve: (s: string, id: string, key?: string) =>
    call<void>('DELETE', `/s/${s}/items/${id}/reserve`, undefined, key ? { 'X-Reservation-Key': key } : {}),

  status: () => call<{ registration: boolean; version: string }>('GET', '/status'),
}

export const photoUrl = (item: Item, name: string, thumb: '480x480' | '1200x0' | '' = '480x480') =>
  `/api/files/items/${item.id}/${encodeURIComponent(name)}${thumb ? `?thumb=${thumb}` : ''}`

/** Photos to show for an item: its own, or else the one fetched from its links. */
export const gallery = (item: Item): string[] => (item.photos.length ? item.photos : item.linkPhoto ? [item.linkPhoto] : [])
export const cover = (item: Item): string | undefined => gallery(item)[0]
/** True when a save should be followed by a link preview fetch. */
export const wantsLinkPreview = (item: Item) => item.links.length > 0 && item.photos.length === 0

export const isUrl = (s: string) => /^https?:\/\/\S+$/i.test(s.trim())

export const editUrl = (edit: string) => `${location.origin}/e/${edit}`
export const shareUrl = (share: string) => `${location.origin}/l/${share}`

export const host = (url: string) => { try { return new URL(url).hostname.replace(/^www\./, '') } catch { return url } }

export function ago(pbDate: string): string {
  const rtf = new Intl.RelativeTimeFormat(i18n.lang, { numeric: 'auto' })
  const t = new Date(pbDate.replace(' ', 'T')).getTime()
  const s = (t - Date.now()) / 1000
  const units: [Intl.RelativeTimeFormatUnit, number][] = [['year', 31536000], ['month', 2592000], ['week', 604800], ['day', 86400], ['hour', 3600], ['minute', 60]]
  for (const [u, n] of units) if (Math.abs(s) >= n) return rtf.format(Math.round(s / n), u)
  return rtf.format(0, 'second')
}
