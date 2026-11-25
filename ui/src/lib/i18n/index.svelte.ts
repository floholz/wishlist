// Tiny typed i18n. To add a language: copy en.ts → xx.ts, translate, add one line to LANGUAGES.
//
// Message syntax (plain text, never HTML):
//   {name}             parameter
//   **bold**           bold text            (rendered by <Rich>)
//   _emphasis_         italic/emphasis      (rendered by <Rich>)
//   [link text]        the one action/link of the sentence (rendered by <Rich>)
//   { one, other, … }  plural forms, chosen by the `count` param (Intl.PluralRules)
import de from './de'
import en from './en'
import type { Key, Messages, Plural } from './types'

export type { Key } from './types'

export const LANGUAGES = [
  { code: 'en', name: 'English', messages: en as Messages },
  { code: 'de', name: 'Deutsch', messages: de },
] as const satisfies readonly { code: string; name: string; messages: Messages }[]

export type Lang = (typeof LANGUAGES)[number]['code']
export type Choice = Lang | 'auto'
const KEY = 'wishlist_lang'

const supported = (code: string | null | undefined): Lang | undefined =>
  LANGUAGES.find(l => l.code === code?.toLowerCase().split('-')[0])?.code

/** First browser/OS language we support, else English. */
function detect(): Lang {
  for (const l of navigator.languages ?? [navigator.language]) {
    const hit = supported(l)
    if (hit) return hit
  }
  return 'en'
}

function stored(): Choice {
  try { return supported(localStorage.getItem(KEY)) ?? 'auto' } catch { return 'auto' }
}

export const i18n = $state<{ choice: Choice; lang: Lang }>({ choice: stored(), lang: 'en' })

function apply() {
  i18n.lang = i18n.choice === 'auto' ? detect() : i18n.choice
  document.documentElement.lang = i18n.lang
}

export function setLanguage(choice: Choice) {
  i18n.choice = choice
  try { if (choice === 'auto') localStorage.removeItem(KEY); else localStorage.setItem(KEY, choice) } catch { /* private mode */ }
  apply()
}

apply()
addEventListener('languagechange', () => { if (i18n.choice === 'auto') apply() })

// ---------- lookup ----------
type Params = Record<string, string | number>

const messagesOf = (lang: Lang): Messages => LANGUAGES.find(l => l.code === lang)!.messages

function lookup(msgs: unknown, key: string): string | Plural | undefined {
  let cur: any = msgs
  for (const part of key.split('.')) cur = cur?.[part]
  return typeof cur === 'string' || (cur && typeof cur.other === 'string') ? cur : undefined
}

export const hasKey = (key: string) => lookup(en, key) !== undefined

/** Translate `key` into the current language (falls back to English). Reactive in templates. */
export function t(key: Key, params: Params = {}): string {
  const lang = i18n.lang
  const msg = lookup(messagesOf(lang), key) ?? lookup(en, key) ?? key
  let text: string
  if (typeof msg === 'string') text = msg
  else {
    const n = Number(params.count ?? 0)
    text = msg[new Intl.PluralRules(lang).select(n)] ?? msg.other
  }
  return text.replace(/\{(\w+)\}/g, (m, p) => (p in params ? String(params[p]) : m))
}

/** Join items the way the language does: "a, b and c" / "a, b und c". */
export const joinList = (items: string[]) => new Intl.ListFormat(i18n.lang, { type: 'conjunction' }).format(items)
