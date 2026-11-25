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
  api.status().then(s => (registration = s.registration)).catch(() => {})

  async function submit(e: SubmitEvent) {
    e.preventDefault()
    busy = true; error = ''
    try {
      if (mode === 'register') await pb.collection('users').create({ email, password, passwordConfirm: password })
      await pb.collection('users').authWithPassword(email, password)
      const c = await claimLocal().catch(() => null)
      session.authOpen = false
      password = ''
      const moved: string[] = []
      if (c?.claimed) moved.push(t('auth.lists', { count: c.claimed }))
      if (c?.reservations) moved.push(t('auth.reservations', { count: c.reservations }))
      say(moved.length ? t('auth.signedInMoved', { what: joinList(moved) }) : t('auth.signedIn'))
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
