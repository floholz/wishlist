// In-app replacement for confirm() / prompt(): `if (await ask({...})) …`
export interface AskOptions {
  title: string
  message?: string
  confirm?: string        // confirm button label
  cancel?: string | null  // null = single-button dialog
  danger?: boolean        // destructive action → red button, Cancel focused
  copyText?: string       // shows a selectable text box (clipboard fallback)
}

export const asking = $state<{ open: boolean; opts: AskOptions }>({ open: false, opts: { title: '' } })
let resolve: ((ok: boolean) => void) | null = null

export function ask(opts: AskOptions): Promise<boolean> {
  resolve?.(false) // a newer question replaces a pending one
  asking.opts = opts
  asking.open = true
  return new Promise(r => (resolve = r))
}

export function answer(ok: boolean) {
  asking.open = false
  resolve?.(ok)
  resolve = null
}
