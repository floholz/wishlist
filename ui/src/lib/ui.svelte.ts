// Small app-wide UI state: toast + auth dialog + signed-in user.
import { pb } from './api'
import { ask } from './ask.svelte'
import { t } from './i18n/index.svelte'

export const toast = $state({ text: '', n: 0 })
let timer: ReturnType<typeof setTimeout>
export function say(text: string) {
  toast.text = text; toast.n++
  clearTimeout(timer); timer = setTimeout(() => (toast.text = ''), 2600)
}

/** `what` is the already translated name of the thing, e.g. t('links.share'). */
export async function copy(text: string, what: string) {
  try { await navigator.clipboard.writeText(text); say(t('common.copied', { what })) }
  catch { ask({ title: t('common.copyTitle'), message: t('common.copyMessage'), copyText: text, confirm: t('common.done'), cancel: null }) }
}

export const session = $state({
  user: pb.authStore.isValid ? pb.authStore.record : null,
  authOpen: false,
})
pb.authStore.onChange(() => (session.user = pb.authStore.isValid ? pb.authStore.record : null))
