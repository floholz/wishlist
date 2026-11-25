<script lang="ts">
  import type { Snippet } from 'svelte'
  import { t } from './i18n/index.svelte'
  import LanguagePicker from './LanguagePicker.svelte'
  import Menu from './Menu.svelte'
  import ThemeToggle from './ThemeToggle.svelte'
  import { pb } from './api'
  import { signOut } from './session'
  import { session } from './ui.svelte'

  let { home = '/app', children }: { home?: string; children?: Snippet } = $props()
  let scrolled = $state(false)
  // the picture from Google (if any); falls back to the initial when missing or broken
  let broken = $state('')
  const avatar = $derived(session.user?.avatar ? pb.files.getURL(session.user, session.user.avatar) : '')
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
      {#snippet trigger()}<span class="avatar">{#if avatar && broken !== avatar}<img src={avatar} alt="" referrerpolicy="no-referrer" onerror={() => (broken = avatar)} />{:else}{(session.user?.name || session.user?.email || '?')[0]}{/if}</span>{/snippet}
      <div class="hint">{t('nav.signedInAs')}<br />{#if session.user?.name}<b style="color:var(--fg)">{session.user.name}</b><br />{session.user.email}{:else}<b style="color:var(--fg)">{session.user?.email}</b>{/if}</div>
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
