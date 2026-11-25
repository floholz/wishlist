<script lang="ts">
  import { api, pb } from './api'
  import Dialog from './Dialog.svelte'
  import { hasKey, joinList, t, type Key } from './i18n/index.svelte'
  import { claimLocal } from './session'
  import { say, session } from './ui.svelte'

  let mode = $state<'signin' | 'register'>('signin')
  let email = $state(''), password = $state('')
  let busy = $state(false), error = $state('')
  let registration = $state(true)
  let google = $state(false)
  api.status().then(s => (registration = s.registration)).catch(() => {})
  pb.collection('users').listAuthMethods()
    .then(m => (google = m.oauth2.enabled && m.oauth2.providers.some(p => p.name === 'google'))).catch(() => {})

  /** After any sign-in: move this browser's lists & reservations into the account. */
  async function signedIn() {
    const c = await claimLocal().catch(() => null)
    session.authOpen = false
    password = ''
    const moved: string[] = []
    if (c?.claimed) moved.push(t('auth.lists', { count: c.claimed }))
    if (c?.reservations) moved.push(t('auth.reservations', { count: c.reservations }))
    say(moved.length ? t('auth.signedInMoved', { what: joinList(moved) }) : t('auth.signedIn'))
  }

  async function withGoogle() {
    busy = true; error = ''
    try {
      // opens a popup; PocketBase handles the redirect at /api/oauth2-redirect
      await pb.collection('users').authWithOAuth2({ provider: 'google' })
      await signedIn()
    } catch (err: any) {
      if (!err?.isAbort) error = err?.status === 403 ? t('errors.registration_closed') : t('auth.googleFailed')
    } finally { busy = false }
  }

  async function submit(e: SubmitEvent) {
    e.preventDefault()
    busy = true; error = ''
    try {
      if (mode === 'register') await pb.collection('users').create({ email, password, passwordConfirm: password })
      await pb.collection('users').authWithPassword(email, password)
      await signedIn()
    } catch (err: any) {
      const data = err?.response?.data
      // PocketBase field errors carry codes (validation_not_unique, …) → translated per field
      const fieldLabel = (k: string) => (k === 'email' ? t('auth.email') : k.startsWith('password') ? t('auth.password') : k)
      error = data && Object.keys(data).length
        ? Object.entries(data).map(([k, v]: any) =>
            `${fieldLabel(k)}: ${hasKey(`errors.${v.code}`) ? t(`errors.${v.code}` as Key) : v.message}`).join('\n')
        : mode === 'signin' ? t('auth.wrongPassword') : err?.status === 403 ? t('errors.registration_closed') : t('common.error')
    } finally { busy = false }
  }
</script>

<Dialog bind:open={session.authOpen} title={mode === 'signin' ? t('auth.signInTitle') : t('auth.registerTitle')}>
  <form class="stack" onsubmit={submit}>
    <p class="muted small" style="margin:0">{t('auth.intro')}</p>
    {#if google}
      <button type="button" class="google" disabled={busy} onclick={withGoogle}>
        <svg viewBox="0 0 48 48" width="18" height="18" aria-hidden="true">
          <path fill="#EA4335" d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"/>
          <path fill="#4285F4" d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"/>
          <path fill="#FBBC05" d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"/>
          <path fill="#34A853" d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"/>
        </svg>
        {t('auth.google')}
      </button>
      <div class="or small muted"><span>{t('auth.or')}</span></div>
    {/if}
    <label class="field">{t('auth.email')}<input type="email" bind:value={email} required autocomplete="email" /></label>
    <label class="field">{t('auth.password')}
      <input type="password" bind:value={password} required minlength={mode === 'register' ? 8 : 1}
        autocomplete={mode === 'signin' ? 'current-password' : 'new-password'} />
    </label>
    {#if error}<div class="error small">{error}</div>{/if}
    <button class="primary" disabled={busy}>{busy ? '…' : mode === 'signin' ? t('auth.submitSignIn') : t('auth.submitRegister')}</button>
    {#if registration}
      <p class="small muted" style="margin:0;text-align:center">
        {mode === 'signin' ? t('auth.noAccount') : t('auth.haveAccount')}
        <button type="button" class="ghost" style="padding:0 .3rem;min-height:0" onclick={() => { mode = mode === 'signin' ? 'register' : 'signin'; error = '' }}>
          <u>{mode === 'signin' ? t('auth.createOne') : t('auth.submitSignIn')}</u>
        </button>
      </p>
    {/if}
  </form>
</Dialog>

<style>
  .google { display: flex; align-items: center; justify-content: center; gap: .6rem; }
  .or { display: flex; align-items: center; gap: .75rem; }
  .or::before, .or::after { content: ''; flex: 1; border-top: 1px solid currentColor; opacity: .3; }
</style>
