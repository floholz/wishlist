<script lang="ts">
  import { api, cover, errText, host, isUrl, photoUrl, wantsLinkPreview, type Item, type List } from '../lib/api'
  import { t } from '../lib/i18n/index.svelte'
  import Rich from '../lib/Rich.svelte'
  import Dialog from '../lib/Dialog.svelte'
  import ItemEditor from '../lib/ItemEditor.svelte'
  import { forgetOwned, rememberOwned } from '../lib/local'
  import Menu from '../lib/Menu.svelte'
  import { PRESETS, preset, presetLabel, presetStyle } from '../lib/presets'
  import { ask } from '../lib/ask.svelte'
  import { navigate } from '../lib/router.svelte'
  import ShareDialog from '../lib/ShareDialog.svelte'
  import TopBar from '../lib/TopBar.svelte'
  import { say, session } from '../lib/ui.svelte'
  import NotFound from './NotFound.svelte'

  let { token }: { token: string } = $props()

  let list = $state<List | null>(null)
  let items = $state<Item[]>([])
  let loading = $state(true), missing = $state(false), error = $state('')
  let title = $state(''), description = $state('')
  let newName = $state(''), adding = $state(false)
  let editing = $state<Item | null>(null), editorOpen = $state(false)
  let shareOpen = $state(false), themeOpen = $state(false)
  let addInput: HTMLInputElement | undefined = $state()
  let fetching = $state<Record<string, boolean>>({}) // item ids waiting for a link picture

  async function load() {
    try {
      const l = await api.getEdit(token)
      items = l.items ?? []
      list = l; title = l.title; description = l.description
      rememberOwned(token, l.share) // opening an edit link adds it to this device's lists
      document.title = `${l.title} · wishlist`
    } catch (err: any) {
      if (err.status === 404) { missing = true; forgetOwned([token]) } else error = errText(err)
    } finally { loading = false }
  }
  // owned/mine change when signing in
  $effect(() => { session.user?.id; load() })

  async function patch(p: Partial<Pick<List, 'title' | 'description' | 'preset'>>) {
    try { const l = await api.patchList(token, p); list = { ...list!, ...l, items: undefined }; title = l.title; description = l.description }
    catch (err: any) { say(errText(err)) }
  }
  function saveTitle() {
    if (!list) return
    if (!title.trim()) { title = list.title; return }
    if (title.trim() !== list.title) patch({ title })
  }
  function saveDescription() { if (list && description.trim() !== list.description) patch({ description }) }

  async function add(e: SubmitEvent) {
    e.preventDefault()
    if (!newName.trim() || adding) return
    adding = true
    // pasting a product link creates the wish from the page: title + picture
    const url = isUrl(newName) ? newName.trim() : ''
    const fd = new FormData()
    fd.set('name', url ? host(url) : newName)
    if (url) fd.set('links', JSON.stringify([url]))
    try {
      const it = await api.createItem(token, fd)
      items.push(it); newName = ''
      if (url) fetchPreview(it, true)
    }
    catch (err: any) { say(errText(err)) } finally { adding = false; addInput?.focus() }
  }

  function open(it: Item) { editing = it; editorOpen = true }
  function replace(it: Item) { const i = items.findIndex(x => x.id === it.id); if (i >= 0) items[i] = it }
  function saved(it: Item) { replace(it); if (wantsLinkPreview(it)) fetchPreview(it) }
  async function fetchPreview(it: Item, autoname = false) {
    fetching[it.id] = true
    try {
      const fresh = await api.linkPreview(token, it.id, autoname)
      // keep local edits that happened meanwhile; only take the fetched bits
      const cur = items.find(x => x.id === it.id)
      if (cur) replace({ ...cur, linkPhoto: fresh.linkPhoto, name: autoname ? fresh.name : cur.name })
    } catch { /* no picture — fine */ } finally { delete fetching[it.id] }
  }
  function deleted(id: string) { items = items.filter(x => x.id !== id) }

  async function removeList() {
    if (!list || !await ask({ title: t('deleteList.title'), message: t('deleteList.message', { title: list.title }), confirm: t('deleteList.confirm'), danger: true })) return
    try { await api.deleteList(token); forgetOwned([token]); say(t('deleteList.done')); navigate('/app', true) }
    catch (err: any) { say(errText(err)) }
  }

  // ---------- priority: drag & drop (pointer events → mouse + touch) and keyboard ----------
  let dragId = $state<string | null>(null)
  let before: string[] = []
  let lastY = 0, scrollTimer = 0

  const order = () => items.map(i => i.id)
  async function persist(prev: string[]) {
    if (prev.join() === order().join()) return
    try { await api.reorder(token, order()) }
    catch (err: any) { say(errText(err)); items = prev.map(id => items.find(i => i.id === id)!) }
  }

  function dragStart(e: PointerEvent, it: Item) {
    if (e.pointerType === 'mouse' && e.button !== 0) return
    e.preventDefault()
    dragId = it.id; before = order(); lastY = e.clientY
    scrollTimer = window.setInterval(autoScroll, 16)
  }
  function autoScroll() {
    const edge = 70
    if (lastY < edge) scrollBy(0, -10)
    else if (lastY > innerHeight - edge) scrollBy(0, 10)
  }
  function dragMove(e: PointerEvent) {
    if (!dragId) return
    lastY = e.clientY
    const el = document.elementFromPoint(e.clientX, e.clientY)?.closest<HTMLElement>('[data-item]')
    if (!el || el.dataset.item === dragId) return
    const from = items.findIndex(i => i.id === dragId), to = items.findIndex(i => i.id === el.dataset.item)
    const r = el.getBoundingClientRect()
    const upperHalf = e.clientY < r.top + r.height / 2
    if ((to > from && upperHalf) || (to < from && !upperHalf)) return // wait until the midpoint is crossed
    const [moved] = items.splice(from, 1)
    items.splice(to, 0, moved)
  }
  function dragEnd() {
    if (!dragId) return
    dragId = null; clearInterval(scrollTimer)
    persist(before)
  }
  function keyMove(e: KeyboardEvent, it: Item) {
    const d = e.key === 'ArrowUp' ? -1 : e.key === 'ArrowDown' ? 1 : 0
    if (!d) return
    e.preventDefault()
    const from = items.indexOf(it), to = from + d
    if (to < 0 || to >= items.length) return
    const prev = order()
    items.splice(to, 0, ...items.splice(from, 1))
    persist(prev)
    requestAnimationFrame(() => document.querySelector<HTMLElement>(`[data-item="${it.id}"] .handle`)?.focus())
  }
</script>

<svelte:window onpointermove={dragMove} onpointerup={dragEnd} onpointercancel={dragEnd} />

{#if missing}
  <NotFound message={t('edit.notFound')} />
{:else}
  <div class="tinted" style={presetStyle(list?.preset)}>
    <TopBar>
      {#if list}
        <button type="button" class="primary share-btn" onclick={() => (shareOpen = true)}>{t('edit.share')}</button>
        <Menu label={t('edit.options')}>
          <a href="/l/{list.share}">{t('edit.preview')}</a>
          <button type="button" onclick={() => (themeOpen = true)}>{t('edit.changeTheme')}</button>
          <button type="button" onclick={() => (shareOpen = true)}>{t('edit.showLinks')}</button>
          <hr />
          <button type="button" class="danger" onclick={removeList}>{t('deleteList.confirm')}</button>
        </Menu>
      {/if}
    </TopBar>

    <header class="hero" style={list ? `background-image:url(${preset(list.preset).image})` : ''}>
      <div class="inner">
        {#if list}
          <input class="title-input" bind:value={title} maxlength="120" aria-label={t('edit.titleLabel')}
            onblur={saveTitle} onkeydown={e => e.key === 'Enter' && e.currentTarget.blur()} />
          <textarea class="desc-input" bind:value={description} maxlength="2000" rows="1" placeholder={t('edit.descriptionPlaceholder')}
            aria-label={t('edit.descriptionLabel')} onblur={saveDescription}></textarea>
        {:else}
          <h1>{loading ? ' ' : t('common.error')}</h1>
        {/if}
      </div>
    </header>

    <main class="page stack">
      {#if error}<p class="error">{error}</p>{/if}
      {#if list}
        {#if !list.owned}
          <div class="notice">
            <p><Rich text={t('edit.keepLink')} /></p>
            <button type="button" onclick={() => (session.authOpen = true)}>{t('common.signIn')}</button>
          </div>
        {/if}

        <form class="add" onsubmit={add}>
          <input bind:this={addInput} bind:value={newName} placeholder={t('edit.addPlaceholder')} maxlength="2000" aria-label={t('edit.addLabel')} enterkeyhint="done" autocomplete="off" />
          <button class="primary" disabled={adding || !newName.trim()}>{t('common.add')}</button>
        </form>

        {#if items.length === 0}
          <div class="empty muted">
            <p>{t('edit.emptyTitle')}</p>
            <p class="small">{t('edit.emptyHint')}</p>
          </div>
        {:else}
          <p class="small muted hint">{t('edit.dragHint')}</p>
          <ol class="items" class:dragging={dragId !== null}>
            {#each items as it, i (it.id)}
              <li data-item={it.id} class:lifted={dragId === it.id}>
                <button type="button" class="handle" aria-label={t('edit.reorder', { name: it.name })} title={t('edit.dragTitle')}
                  onpointerdown={e => dragStart(e, it)} onkeydown={e => keyMove(e, it)}>⠿</button>
                <span class="rank">{i + 1}</span>
                <button type="button" class="body" onclick={() => open(it)}>
                  {#if cover(it)}<img class="thumb" src={photoUrl(it, cover(it)!)} alt="" loading="lazy" />
                  {:else if fetching[it.id]}<span class="thumb loading" aria-label={t('edit.fetchingPicture')}></span>{/if}
                  <span class="txt">
                    <span class="name">{it.name}</span>
                    {#if it.description}<span class="desc small muted">{it.description}</span>{/if}
                    {#if it.links.length}
                      <span class="chips">{#each it.links as l}<span class="link-chip">↗ {host(l)}</span>{/each}</span>
                    {/if}
                  </span>
                  <span class="chev muted" aria-hidden="true">›</span>
                </button>
              </li>
            {/each}
          </ol>
          <p class="small muted center">{t('edit.hiddenNote')}</p>
        {/if}
      {/if}
    </main>
  </div>

  {#if list}
    <ShareDialog bind:open={shareOpen} {list} />
    <Dialog bind:open={themeOpen} title={t('edit.theme')}>
      <div class="themes">
        {#each PRESETS as p (p.id)}
          <button type="button" class="theme" class:on={list.preset === p.id} style="{presetStyle(p.id)}background-image:url({p.image})"
            onclick={() => { patch({ preset: p.id }); themeOpen = false }}><span>{presetLabel(p.id)}</span></button>
        {/each}
      </div>
    </Dialog>
  {/if}
  {#if editing}
    {#key editing.id}
      <ItemEditor bind:open={editorOpen} {token} item={editing} onsaved={saved} ondeleted={deleted} />
    {/key}
  {/if}
{/if}

<style>
  .share-btn { padding-inline: 1.2rem; }
  .title-input, .desc-input { all: unset; display: block; width: 100%; box-sizing: border-box; color: #fff; border-radius: 8px; transition: background .15s; }
  .title-input { font-family: var(--serif); font-size: clamp(1.8rem, 6vw, 2.6rem); font-weight: 600; line-height: 1.2; text-shadow: 0 1px 12px rgb(0 0 0 / .3); padding: .1rem .3rem; margin-left: -.3rem; }
  .desc-input { margin: .3rem 0 0 -.3rem; padding: .2rem .3rem; field-sizing: content; min-height: 1.5em; opacity: .95; white-space: pre-wrap; }
  .desc-input::placeholder { color: rgb(255 255 255 / .75); }
  .title-input:hover, .desc-input:hover, .title-input:focus, .desc-input:focus { background: rgb(255 255 255 / .14); }

  .add { display: flex; gap: .5rem; }
  .add input { border-radius: 999px; padding-inline: 1.1rem; }
  .empty { text-align: center; padding: 2.5rem 1rem; }
  .empty p { margin: .2rem 0; }
  .hint { margin: 0; }
  .center { text-align: center; }

  .items { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: .5rem; }
  .items li { display: flex; align-items: center; gap: .35rem; background: var(--panel); border: 1px solid var(--line); border-radius: var(--radius);
    padding: .4rem .4rem .4rem .2rem; transition: box-shadow .15s, transform .15s; }
  .items.dragging { user-select: none; -webkit-user-select: none; }
  .items.dragging li:not(.lifted) .body { pointer-events: none; }
  .items li.lifted { box-shadow: var(--shadow); transform: scale(1.015); border-color: var(--accent); position: relative; z-index: 2; }
  .handle { all: unset; cursor: grab; touch-action: none; color: var(--muted); padding: .6rem .35rem; font-size: 1.1rem; border-radius: 6px; }
  .handle:focus-visible { outline: 2px solid var(--accent); }
  .items.dragging .handle { cursor: grabbing; }
  .body { all: unset; cursor: pointer; flex: 1; min-width: 0; display: flex; align-items: center; gap: .75rem; padding: .2rem .4rem; border-radius: 8px; }
  .body:focus-visible { outline: 2px solid var(--accent); }
  .thumb { width: 3.25rem; height: 3.25rem; }
  .txt { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: .15rem; }
  .name { font-weight: 550; overflow-wrap: anywhere; }
  .desc { overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; }
  .chips { display: flex; flex-wrap: wrap; gap: .3rem; margin-top: .1rem; }
  .thumb.loading { animation: pulse 1.1s ease-in-out infinite alternate; }
  @keyframes pulse { to { opacity: .4; } }
  .chev { font-size: 1.4rem; padding-right: .2rem; }

  .themes { display: grid; grid-template-columns: repeat(auto-fill, minmax(8rem, 1fr)); gap: .6rem; }
  .theme { aspect-ratio: 4 / 3; border-radius: 12px; background: var(--accent) center / cover; color: #fff; align-items: flex-end; justify-content: flex-start;
    padding: .5rem .7rem; border: 2px solid transparent; position: relative; overflow: hidden; }
  .theme::after { content: ""; position: absolute; inset: 0; background: linear-gradient(to top, rgb(0 0 0 / .5), transparent 60%); }
  .theme span { position: relative; z-index: 1; font-weight: 600; }
  .theme.on { border-color: var(--fg); }
</style>
