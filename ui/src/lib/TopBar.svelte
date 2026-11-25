<script lang="ts">
  import type { Snippet } from 'svelte'
  import { t } from './i18n/index.svelte'
  import LanguagePicker from './LanguagePicker.svelte'
  import Menu from './Menu.svelte'
  import ThemeToggle from './ThemeToggle.svelte'
  import { signOut } from './session'
  import { session } from './ui.svelte'

  let { home = '/app', children }: { home?: string; children?: Snippet } = $props()
  let scrolled = $state(false)
</script>

<svelte:window onscroll={() => (scrolled = scrollY > 4)} />

<header class="bar" class:scrolled>
  <a class="brand" href={home}><img src="/favicon.svg" alt="" /><span class="brand-name">wishlist</span></a>
  <span class="spacer"></span>
  {@render children?.()}
  <LanguagePicker />
  <ThemeToggle />
  {#if session.user}
    <Menu label={t('nav.account')} class="icon-btn">
      {#snippet trigger()}<span class="avatar">{(session.user?.email ?? '?')[0]}</span>{/snippet}
      <div class="hint">{t('nav.signedInAs')}<br /><b style="color:var(--fg)">{session.user?.email}</b></div>
      <hr />
      <a href="/app">{t('nav.yourLists')}</a>
      <button type="button" onclick={() => signOut()}>{t('nav.signOut')}</button>
    </Menu>
  {:else}
    <button type="button" onclick={() => (session.authOpen = true)}>{t('common.signIn')}</button>
  {/if}
</header>

<style>
  /* the bar gets crowded on small phones: keep the logo, drop the word */
  @media (max-width: 420px) { .brand-name { display: none; } }
</style>
