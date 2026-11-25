<script lang="ts">
  import { api, ago, cover, editUrl, errText, photoUrl, shareUrl, type Gift, type List, type Visit } from '../lib/api'
  import { t } from '../lib/i18n/index.svelte'
  import Rich from '../lib/Rich.svelte'
  import { forgetOwned, forgetVisit, rememberOwned, visitedLists } from '../lib/local'
  import { localState } from '../lib/session'
  import Menu from '../lib/Menu.svelte'
  import { PRESETS, preset, presetLabel, presetStyle, presetTitle, type Preset } from '../lib/presets'
  import { ask } from '../lib/ask.svelte'
  import { navigate } from '../lib/router.svelte'
  import TopBar from '../lib/TopBar.svelte'
  import { copy, say, session } from '../lib/ui.svelte'

  let lists = $state<List[]>([])
  const localVisits = (): Visit[] => visitedLists().map(({ share, title, preset }) => ({ share, title, preset }))
  let visited = $state<Visit[]>(localVisits())
  let gifts = $state<Gift[]>([])
  let loading = $state(true), error = $state('')
  let creating = $state<string | null>(null)

  async function load() {
    error = ''
    try {
      const r = await api.dashboard(localState())
      if (r.missing.length) forgetOwned(r.missing) // deleted elsewhere
      lists = r.lists
      gifts = r.gifts
      // account visits first, then this browser's; own lists don't belong in "visited"
      const seen = new Set(r.lists.map(l => l.share))
      visited = [...r.visited.map(v => ({ ...v, synced: true })), ...localVisits()].filter(v => !seen.has(v.share) && (seen.add(v.share), true))
    } catch (err: any) { error = errText(err) } finally { loading = false }
  }
  // reload when signing in/out
  $effect(() => { session.user?.id; load() })

  async function create(p: Preset) {
    if (creating) return
    creating = p.id
    try {
      const l = await api.createList(p.id, presetTitle(p.id))
      rememberOwned(l.edit!, l.share)
      navigate(`/e/${l.edit}`)
    } catch (err: any) { say(errText(err)); creating = null }
  }

  async function remove(l: List) {
    if (!await ask({ title: t('deleteList.title'), message: t('deleteList.message', { title: l.title }), confirm: t('deleteList.confirm'), danger: true })) return
    try { await api.deleteList(l.edit!); forgetOwned([l.edit!]); lists = lists.filter(x => x !== l); say(t('deleteList.done')) }
    catch (err: any) { say(errText(err)) }
  }
  function forget(l: List) { forgetOwned([l.edit!]); lists = lists.filter(x => x !== l); say(t('dashboard.forgotten')) }

  const anonymous = $derived(!session.user && lists.length > 0)
  const showLists = $derived(lists.length > 0 || gifts.length === 0)
</script>

<TopBar home="/" />

<section class="band">
  <div class="wrap">
    <h2 class="label">{t('dashboard.startNew')}</h2>
    <div class="templates">
      {#each PRESETS as p (p.id)}
        <button type="button" class="tpl" style={presetStyle(p.id)} onclick={() => create(p)} disabled={!!creating}>
          <span class="tpl-img" class:blank={p.id === 'blank'} style={p.id === 'blank' ? '' : `background-image:url(${p.image})`}>
            {#if p.id === 'blank'}<span class="plus" aria-hidden="true"></span>{/if}
            {#if creating === p.id}<span class="busy">{t('dashboard.creating')}</span>{/if}
          </span>
          <span class="tpl-name">{presetLabel(p.id)}</span>
        </button>
      {/each}
    </div>
  </div>
</section>

<main class="wrap lists">
  {#if gifts.length}
    <div class="row head"><h2 class="label">{t('dashboard.gifts')}</h2></div>
    <div class="gifts">
      {#each gifts as g (g.item.id)}
        <a class="gift tinted" style={presetStyle(g.preset)} href="/l/{g.share}">
          {#if cover(g.item)}
            <img src={photoUrl(g.item, cover(g.item)!)} alt="" loading="lazy" />
          {:else}
            <span class="gimg" style="background-image:url({preset(g.preset).image})"></span>
          {/if}
          <span class="gtxt">
            <span class="gname">{g.item.name}</span>
            <span class="small muted">{t('dashboard.giftFor', { title: g.title })}</span>
          </span>
          <span class="check" aria-label={t('dashboard.reservedByYou')}>✓</span>
        </a>
      {/each}
    </div>
    {#if !session.user}
      <p class="small muted" style="margin:0"><Rich text={t('dashboard.giftsLocal')} onclick={() => (session.authOpen = true)} /></p>
    {/if}
  {/if}

  {#if showLists}
    <div class="row head" style={gifts.length ? 'margin-top:1.5rem' : ''}>
      <h2 class="label">{session.user ? t('dashboard.yourLists') : t('dashboard.deviceLists')}</h2>
    </div>
  {/if}

  {#if anonymous}
    <div class="notice">
      <p><Rich text={t('dashboard.anonymous')} /></p>
      <button type="button" class="primary" onclick={() => (session.authOpen = true)}>{t('common.signIn')}</button>
    </div>
  {/if}
  {#if error}<p class="error">{error}</p>{/if}

  {#if loading}
    <p class="muted">{t('common.loading')}</p>
  {:else if lists.length === 0 && showLists}
    <div class="empty">
      <p><b>{t('dashboard.emptyTitle')}</b></p>
      <p class="muted">{t('dashboard.emptyHint')}</p>
    </div>
  {:else}
    <div class="grid">
      {#each lists as l (l.share)}
        {@const p = preset(l.preset)}
        <article class="card tinted" style={presetStyle(l.preset)}>
          <a class="paper" href="/e/{l.edit}" aria-label={t('dashboard.open', { title: l.title })}>
            <span class="paper-hero" style="background-image:url({p.image})"><span>{l.title}</span></span>
            <ol>
              {#each l.preview ?? [] as name}<li>{name}</li>{/each}
              {#if !l.preview?.length}<li class="muted">{t('dashboard.noWishes')}</li>{/if}
            </ol>
          </a>
          <div class="meta">
            <a class="t" href="/e/{l.edit}">{l.title}</a>
            <div class="row sub">
              <span class="dot" aria-hidden="true"></span>
              <span class="small muted grow">{t('common.wishes', { count: l.count })} · {ago(l.updated)}</span>
              <Menu label={t('dashboard.actions')}>
                <a href="/e/{l.edit}">{t('dashboard.edit')}</a>
                <a href="/l/{l.share}">{t('dashboard.openShare')}</a>
                <button type="button" onclick={() => copy(shareUrl(l.share), t('links.share'))}>{t('dashboard.copyShare')}</button>
                <button type="button" onclick={() => copy(editUrl(l.edit!), t('links.edit'))}>{t('dashboard.copyEdit')}</button>
                <hr />
                {#if !l.mine}<button type="button" onclick={() => forget(l)}>{t('dashboard.forget')}</button>{/if}
                <button type="button" class="danger" onclick={() => remove(l)}>{t('deleteList.confirm')}</button>
              </Menu>
            </div>
          </div>
        </article>
      {/each}
    </div>
  {/if}

  {#if visited.length}
    <div class="row head" style="margin-top:2.5rem"><h2 class="label">{t('dashboard.visited')}</h2></div>
    <div class="visited">
      {#each visited as v (v.share)}
        <div class="vrow tinted" style={presetStyle(v.preset)}>
          <a href="/l/{v.share}" class="vlink">
            <span class="vimg" style="background-image:url({preset(v.preset).image})"></span>
            <span class="grow">{v.title}</span>
          </a>
          {#if !v.synced}<!-- this browser's visits can be forgotten; the account's stay -->
            <button type="button" class="icon-btn small" aria-label={t('dashboard.forgetVisit', { title: v.title })} title={t('dashboard.forgetShort')}
              onclick={() => { forgetVisit(v.share); visited = visited.filter(x => x !== v) }}>✕</button>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</main>

<style>
  .wrap { max-width: 72rem; margin: 0 auto; padding: 0 1rem; }
  .label { font-size: 1rem; font-weight: 500; }
  .band { background: var(--band); padding: 1rem 0 1.5rem; }
  .band .label { margin: .25rem 0 1rem; }
  .templates { display: flex; gap: 1rem; overflow-x: auto; scroll-snap-type: x mandatory; padding-bottom: .25rem; margin: 0 -1rem; padding-inline: 1rem; scrollbar-width: none; }
  .tpl { all: unset; cursor: pointer; display: flex; flex-direction: column; gap: .55rem; flex: none; width: 9.25rem; scroll-snap-align: start; }
  .tpl:focus-visible .tpl-img { outline: 2px solid var(--accent); outline-offset: 2px; }
  .tpl[disabled] { cursor: progress; }
  .tpl-img { position: relative; aspect-ratio: 3 / 4; border-radius: 8px; border: 1px solid var(--line); background: var(--accent) center / cover;
    transition: border-color .15s, transform .15s; overflow: hidden; }
  .tpl:hover .tpl-img { border-color: var(--accent); transform: translateY(-2px); }
  .tpl-img.blank { background: var(--panel); display: grid; place-items: center; }
  .plus { width: 3rem; height: 3rem; position: relative; }
  .plus::before, .plus::after { content: ""; position: absolute; background: var(--fg); border-radius: 2px; }
  .plus::before { left: 50%; top: 0; bottom: 0; width: 5px; transform: translateX(-50%); }
  .plus::after { top: 50%; left: 0; right: 0; height: 5px; transform: translateY(-50%); }
  .busy { position: absolute; inset: 0; display: grid; place-items: center; background: rgb(0 0 0 / .45); color: #fff; font-size: .85rem; }
  .tpl-name { font-size: .92rem; font-weight: 500; }

  .lists { padding-top: 1.5rem; padding-bottom: calc(3rem + env(safe-area-inset-bottom)); display: flex; flex-direction: column; gap: 1rem; }
  .head { justify-content: space-between; }
  .empty { text-align: center; padding: 3rem 1rem; }
  .empty p { margin: .25rem 0; }
  .grid { display: grid; gap: 1.25rem; grid-template-columns: repeat(auto-fill, minmax(13rem, 1fr)); }
  @media (max-width: 520px) { .grid { grid-template-columns: 1fr 1fr; gap: .75rem; } }

  .card { border: 1px solid var(--line); border-radius: 8px; background: var(--panel); overflow: hidden; display: flex; flex-direction: column; transition: border-color .15s; }
  .card:hover { border-color: var(--accent); }
  .paper { display: block; aspect-ratio: 4 / 3.6; overflow: hidden; text-decoration: none; border-bottom: 1px solid var(--line); background: var(--panel); }
  .paper-hero { display: flex; align-items: flex-end; height: 45%; background: var(--accent) center / cover; position: relative; }
  .paper-hero::after { content: ""; position: absolute; inset: 0; background: linear-gradient(to top, rgb(0 0 0 / .5), transparent 70%); }
  .paper-hero span { position: relative; z-index: 1; color: #fff; font-family: var(--serif); font-weight: 600; padding: .5rem .7rem; font-size: .95rem;
    line-height: 1.2; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; }
  .paper ol { margin: 0; padding: .55rem .7rem 0 1.9rem; font-size: .72rem; color: var(--muted); }
  .paper li { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; padding: .05rem 0; }
  .paper li::marker { color: var(--accent-ink); font-weight: 700; }
  .meta { padding: .6rem .5rem .4rem .8rem; }
  .t { display: block; font-weight: 500; text-decoration: none; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; padding-right: .3rem; }
  .sub { gap: .45rem; }
  .grow { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .dot { width: .7rem; height: .7rem; border-radius: 3px; background: var(--accent); flex: none; }

  .gifts { display: grid; grid-template-columns: repeat(auto-fill, minmax(17rem, 1fr)); gap: .6rem; }
  .gift { display: flex; align-items: center; gap: .75rem; padding: .5rem .7rem .5rem .5rem; border: 1px solid var(--line); border-radius: 12px;
    background: var(--panel); text-decoration: none; transition: border-color .15s; }
  .gift:hover { border-color: var(--accent); }
  .gift img, .gimg { width: 3.25rem; height: 3.25rem; border-radius: 8px; object-fit: cover; flex: none; background: var(--accent) center / cover; }
  .gtxt { flex: 1; min-width: 0; display: flex; flex-direction: column; }
  .gname { font-weight: 550; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .gtxt .muted { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .check { width: 1.6rem; height: 1.6rem; border-radius: 999px; display: grid; place-items: center; background: var(--accent-soft); color: var(--accent-ink); font-weight: 700; font-size: .85rem; flex: none; }
  .visited { display: grid; grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr)); gap: .5rem; }
  .vrow { display: flex; align-items: center; border: 1px solid var(--line); border-radius: 10px; background: var(--panel); padding-right: .25rem; }
  .vlink { display: flex; align-items: center; gap: .7rem; text-decoration: none; flex: 1; min-width: 0; padding: .45rem .6rem; }
  .vimg { width: 2.2rem; height: 2.2rem; border-radius: 6px; background: var(--accent) center / cover; flex: none; }
</style>
