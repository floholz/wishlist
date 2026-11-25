<script lang="ts">
  import { api, errText, gallery, host, photoUrl, type Item, type List } from '../lib/api'
  import { t } from '../lib/i18n/index.svelte'
  import Rich from '../lib/Rich.svelte'
  import Dialog from '../lib/Dialog.svelte'
  import { isOwnShare, ownedLists, rememberVisit, reservations, setReservation } from '../lib/local'
  import { preset, presetStyle } from '../lib/presets'
  import TopBar from '../lib/TopBar.svelte'
  import { say, session } from '../lib/ui.svelte'
  import NotFound from './NotFound.svelte'

  let { share }: { share: string } = $props()

  let list = $state<List | null>(null)
  let items = $state<Item[]>([])
  let mine = $state<Record<string, string>>(reservations()) // itemId → key
  let loading = $state(true), missing = $state(false), error = $state('')
  let busy = $state<string | null>(null)
  let viewer = $state<{ item: Item; i: number } | null>(null), viewerOpen = $state(false)

  // The list owner never gets to see reservations, not even here.
  const owner = $derived(isOwnShare(share) || !!list?.mine)
  const ownEdit = $derived(ownedLists().find(l => l.share === share)?.edit)

  async function load() {
    try {
      const l = await api.getShare(share)
      list = l; items = l.items ?? []
      document.title = `${l.title} · wishlist`
      // signed in, the server keeps visits for the account
      if (!isOwnShare(share) && !l.mine && !session.user) rememberVisit({ share, title: l.title, preset: l.preset })
      // forget stale keys (item deleted or reservation gone)
      mine = reservations()
      for (const id of Object.keys(mine)) {
        const it = items.find(i => i.id === id)
        if (it && !it.reserved) { setReservation(id, null); delete mine[id] }
      }
    } catch (err: any) { if (err.status === 404) missing = true; else error = errText(err) }
    finally { loading = false }
  }
  // reload on sign-in/out: the account may hold reservations made on another device
  $effect(() => { session.user?.id; load() })

  async function reserve(it: Item) {
    busy = it.id
    try {
      const { key } = await api.reserve(share, it.id)
      setReservation(it.id, key); mine[it.id] = key; it.reserved = true
      say(t('share.reserved'))
    } catch (err: any) {
      if (err.status === 409) { it.reserved = true; say(t('share.tooSlow')) } else say(errText(err))
    } finally { busy = null }
  }
  async function release(it: Item) {
    busy = it.id
    try {
      await api.unreserve(share, it.id, mine[it.id])
      setReservation(it.id, null); delete mine[it.id]; it.reserved = false; it.reservedByMe = false
      say(t('share.released'))
    } catch (err: any) { say(errText(err)) } finally { busy = null }
  }

  const statusOf = (it: Item) => owner ? 'open' : mine[it.id] || it.reservedByMe ? 'mine' : it.reserved ? 'taken' : 'open'
  const openCount = $derived(items.filter(i => statusOf(i) === 'open').length)
  const myCount = $derived(items.filter(i => statusOf(i) === 'mine').length)
</script>

{#if missing}
  <NotFound message={t('share.notFound')} />
{:else}
  <div class="tinted" style={presetStyle(list?.preset)}>
    <TopBar home="/" />
    <header class="hero" style={list ? `background-image:url(${preset(list.preset).image})` : ''}>
      <div class="inner">
        <h1>{list?.title ?? ' '}</h1>
        {#if list?.description}<p>{list.description}</p>{/if}
        {#if list && !owner}<p class="small count">{t('common.wishes', { count: items.length })} · {t('share.stillOpen', { count: openCount })}</p>{/if}
      </div>
    </header>

    <main class="page stack">
      {#if error}<p class="error">{error}</p>{/if}
      {#if loading}<p class="muted">{t('common.loading')}</p>{/if}

      {#if list && owner}
        <div class="notice">
          <p><Rich text={t('share.ownerNotice')} /></p>
          {#if ownEdit}<a class="btn" href="/e/{ownEdit}">{t('share.editList')}</a>{/if}
        </div>
      {:else if list && items.length}
        <p class="small muted intro"><Rich text={t('share.intro')} /></p>
      {/if}

      {#if list && !owner && myCount > 0 && !session.user}
        <div class="notice">
          <p><Rich text={t('share.localNotice')} /></p>
          <button type="button" class="primary" onclick={() => (session.authOpen = true)}>{t('common.signIn')}</button>
        </div>
      {/if}

      {#if list && items.length === 0}
        <p class="muted empty">{t('share.empty')}</p>
      {/if}

      <ol class="items">
        {#each items as it, i (it.id)}
          {@const s = statusOf(it)}
          {@const pics = gallery(it)}
          <li class="card" class:taken={s === 'taken'} class:mine={s === 'mine'}>
            {#if pics.length}
              <button type="button" class="pic" aria-label={t('share.viewPhotos', { name: it.name })} onclick={() => { viewer = { item: it, i: 0 }; viewerOpen = true }}>
                <img src={photoUrl(it, pics[0])} alt="" loading="lazy" />
                {#if pics.length > 1}<span class="more">+{pics.length - 1}</span>{/if}
              </button>
            {/if}
            <div class="content">
              <div class="row top">
                <span class="rank" title={t('share.priority', { n: i + 1 })}>{i + 1}</span>
                <h3>{it.name}</h3>
              </div>
              {#if it.description}<p class="desc">{it.description}</p>{/if}
              {#if it.links.length}
                <div class="chips">
                  {#each it.links as l}<a class="link-chip" href={l} target="_blank" rel="noopener noreferrer nofollow">↗ {host(l)}</a>{/each}
                </div>
              {/if}
              {#if !owner}
                <div class="row actions">
                  {#if s === 'open'}
                    <button type="button" class="primary" disabled={busy === it.id} onclick={() => reserve(it)}>{t('share.reserve')}</button>
                  {:else if s === 'mine'}
                    <span class="badge mine-b">{t('share.mine')}</span>
                    <button type="button" class="ghost small" disabled={busy === it.id} onclick={() => release(it)}>{t('share.undo')}</button>
                  {:else}
                    <span class="badge">{t('share.taken')}</span>
                  {/if}
                </div>
              {/if}
            </div>
          </li>
        {/each}
      </ol>

      {#if list}
        <footer class="center small muted">
          {#if session.user}
            <p><Rich text={t('share.footerUser')} href="/app" /></p>
          {:else}
            <p><Rich text={t('share.footerGuest')} onclick={() => (session.authOpen = true)} /></p>
          {/if}
          <a class="btn" href="/">{t('share.makeOwn')}</a>
        </footer>
      {/if}
    </main>
  </div>

  {#if viewer}
    {@const pics = gallery(viewer.item)}
    <Dialog bind:open={viewerOpen} title={viewer.item.name} wide>
      <div class="viewer">
        <img src={photoUrl(viewer.item, pics[viewer.i], '1200x0')} alt={t('share.photoAlt', { name: viewer.item.name, n: viewer.i + 1 })} />
        {#if pics.length > 1}
          <div class="row nav">
            <button type="button" aria-label={t('share.prevPhoto')} onclick={() => viewer && (viewer.i = (viewer.i - 1 + pics.length) % pics.length)}>‹</button>
            <span class="small muted">{viewer.i + 1} / {pics.length}</span>
            <button type="button" aria-label={t('share.nextPhoto')} onclick={() => viewer && (viewer.i = (viewer.i + 1) % pics.length)}>›</button>
          </div>
        {/if}
      </div>
    </Dialog>
  {/if}
{/if}

<style>
  .count { opacity: .85; }
  .intro { margin: 0; }
  .empty { text-align: center; padding: 2rem; }
  .items { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: .8rem; }
  .card { display: flex; background: var(--panel); border: 1px solid var(--line); border-radius: var(--radius); overflow: hidden; transition: opacity .2s; }
  .card.mine { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent); }
  .card.taken { opacity: .55; }
  .card.taken h3 { text-decoration: line-through; text-decoration-thickness: 1px; }
  .pic { all: unset; cursor: zoom-in; position: relative; flex: none; width: 9rem; background: var(--band); }
  .pic img { width: 100%; height: 100%; object-fit: cover; min-height: 9rem; }
  .more { position: absolute; right: .4rem; bottom: .4rem; background: rgb(0 0 0 / .6); color: #fff; font-size: .75rem; padding: .05rem .45rem; border-radius: 999px; }
  .content { flex: 1; min-width: 0; padding: .9rem 1rem; display: flex; flex-direction: column; gap: .45rem; }
  .top { align-items: flex-start; gap: .6rem; }
  h3 { font-size: 1.05rem; font-weight: 600; padding-top: .15rem; overflow-wrap: anywhere; }
  .desc { margin: 0; white-space: pre-wrap; color: var(--muted); font-size: .93rem; overflow-wrap: anywhere; }
  .chips { display: flex; flex-wrap: wrap; gap: .35rem; }
  .actions { margin-top: auto; padding-top: .25rem; flex-wrap: wrap; }
  .badge { font-size: .85rem; padding: .3rem .75rem; border-radius: 999px; background: var(--band); color: var(--muted); }
  .badge.mine-b { background: var(--accent-soft); color: var(--accent-ink); font-weight: 600; }
  @media (max-width: 520px) {
    .card { flex-direction: column; }
    .pic { width: 100%; aspect-ratio: 4 / 3; }
    .pic img { min-height: 0; }
  }
  footer { padding-top: 1.5rem; display: flex; flex-direction: column; align-items: center; gap: .6rem; }
  footer p { margin: 0; }
  .center { text-align: center; }
  .viewer img { width: 100%; max-height: 70dvh; object-fit: contain; border-radius: 10px; background: var(--band); }
  .nav { justify-content: center; margin-top: .75rem; gap: 1rem; }
</style>
