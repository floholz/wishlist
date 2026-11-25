<script lang="ts">
  import { api, errText, host, isUrl, photoUrl, type Item } from './api'
  import Dialog from './Dialog.svelte'
  import { shrink } from './image'
  import { ask } from './ask.svelte'
  import { t } from './i18n/index.svelte'
  import { say } from './ui.svelte'

  let { open = $bindable(false), token, item, onsaved, ondeleted }:
    { open?: boolean; token: string; item: Item; onsaved: (i: Item) => void; ondeleted: (id: string) => void } = $props()

  const MAX_PHOTOS = 8
  // existing photo | new upload | image (or product page) URL the server will download
  type Photo = { old: string } | { file: File; url: string } | { remote: string; broken?: boolean }
  const key = (p: Photo) => ('old' in p ? 'k' + p.old : 'file' in p ? 'f' + p.url : 'u' + p.remote)

  // the editor is re-created per item ({#key}), so capturing the initial values is intended
  // svelte-ignore state_referenced_locally
  let name = $state(item.name), description = $state(item.description)
  // svelte-ignore state_referenced_locally
  let links = $state<string[]>(item.links.length ? [...item.links] : [''])
  // svelte-ignore state_referenced_locally
  let photos = $state<Photo[]>(item.photos.map(old => ({ old })))
  // svelte-ignore state_referenced_locally
  let linkPhoto = $state(item.linkPhoto)
  let clearLinkPhoto = $state(false)
  let busy = $state(false), error = $state('')
  let urlOpen = $state(false), urlValue = $state('')
  let dropping = $state(0) // dragenter/leave counter (children fire their own events)
  let fileInput: HTMLInputElement, urlInput: HTMLInputElement | undefined = $state()

  const full = $derived(photos.length >= MAX_PHOTOS)

  async function addFiles(files: Iterable<File>) {
    for (const f of files) {
      if (photos.length >= MAX_PHOTOS) { say(t('item.maxPhotos', { n: MAX_PHOTOS })); break }
      if (!f.type.startsWith('image/')) continue
      const small = await shrink(f)
      photos.push({ file: small, url: URL.createObjectURL(small) })
    }
    if (fileInput) fileInput.value = ''
  }
  function addUrl(raw: string) {
    let u = raw.trim()
    if (!u) return false
    if (!/^https?:\/\//i.test(u)) u = 'https://' + u
    if (!isUrl(u)) { say(t('item.notALink')); return false }
    if (full) { say(t('item.maxPhotos', { n: MAX_PHOTOS })); return false }
    if (!photos.some(p => 'remote' in p && p.remote === u)) photos.push({ remote: u })
    return true
  }
  function removePhoto(i: number) {
    const p = photos[i]
    if ('file' in p) URL.revokeObjectURL(p.url)
    photos.splice(i, 1)
  }

  // ---------- paste & drop ----------
  function onPaste(e: ClipboardEvent) {
    if (!open || !e.clipboardData) return
    const files = Array.from(e.clipboardData.files).filter(f => f.type.startsWith('image/'))
    if (files.length) { e.preventDefault(); addFiles(files); return }
    // a pasted URL outside of a text field becomes a photo
    const t = e.target as HTMLElement | null
    if (t?.closest('input, textarea')) return
    const text = e.clipboardData.getData('text/plain')
    if (isUrl(text)) { e.preventDefault(); addUrl(text) }
  }
  function onDrop(e: DragEvent) {
    e.preventDefault()
    dropping = 0
    const dt = e.dataTransfer
    if (!dt) return
    const files = Array.from(dt.files).filter(f => f.type.startsWith('image/'))
    if (files.length) { addFiles(files); return }
    // an image dragged from another tab: take its address (uri-list, or the <img src> in the html)
    const uri = dt.getData('text/uri-list').split('\n').find(l => l && !l.startsWith('#'))
    const src = dt.getData('text/html').match(/<img[^>]+src=["']([^"']+)/i)?.[1]
    const text = dt.getData('text/plain')
    const u = [src, uri, text].find(x => x && isUrl(x))
    if (u) addUrl(u); else say(t('item.dropHint'))
  }
  const hasDropData = (e: DragEvent) => !!e.dataTransfer && Array.from(e.dataTransfer.types).some(t => t === 'Files' || t === 'text/uri-list' || t === 'text/html')

  async function save(e: SubmitEvent) {
    e.preventDefault()
    if (!name.trim()) return
    if (urlOpen && urlValue.trim() && addUrl(urlValue)) { urlValue = ''; urlOpen = false }
    busy = true; error = ''
    const fd = new FormData()
    fd.set('name', name)
    fd.set('description', description)
    fd.set('links', JSON.stringify(links.map(l => l.trim()).filter(Boolean)))
    let n = 0
    const order = photos.map(p => {
      if ('old' in p) return 'k:' + p.old
      if ('file' in p) { fd.append('photos', p.file); return 'f:' + n++ }
      return 'u:' + p.remote
    })
    fd.set('order', JSON.stringify(order))
    if (clearLinkPhoto) fd.set('clear_link_photo', '1')
    try { onsaved(await api.updateItem(token, item.id, fd)); open = false }
    catch (err: any) { error = errText(err) } finally { busy = false }
  }

  async function remove() {
    if (!await ask({ title: t('item.removeTitle'), message: t('item.removeMessage', { name: item.name }), confirm: t('common.remove'), danger: true })) return
    try { await api.deleteItem(token, item.id); ondeleted(item.id); open = false; say(t('item.removed')) }
    catch (err: any) { error = errText(err) }
  }
</script>

<svelte:window onpaste={onPaste} />

<Dialog bind:open title={t('item.title')}>
  <form id="item-form" class="stack" onsubmit={save}>
    <label class="field">{t('item.what')}
      <input bind:value={name} required maxlength="200" placeholder={t('item.namePlaceholder')} />
    </label>
    <label class="field">{t('item.details')} <span class="opt">{t('item.detailsHint')}</span>
      <textarea bind:value={description} maxlength="4000" rows="3"></textarea>
    </label>

    <div class="field-group">
      <span class="lbl">{t('item.links')} <span class="opt">{t('item.linksHint')}</span></span>
      {#each links as _, i}
        <div class="row">
          <input type="url" inputmode="url" bind:value={links[i]} placeholder="https://…" aria-label={t('item.link', { n: i + 1 })} />
          {#if links.length > 1 || links[0]}
            <button type="button" class="icon-btn" aria-label={t('item.removeLink')} onclick={() => { links.splice(i, 1); if (!links.length) links.push('') }}>✕</button>
          {/if}
        </div>
      {/each}
      {#if links.length < 10}
        <button type="button" class="ghost add-link" onclick={() => links.push('')}>{t('item.anotherLink')}</button>
      {/if}
    </div>

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="field-group drop" class:dropping={dropping > 0}
      ondragenter={e => { if (hasDropData(e)) { e.preventDefault(); dropping++ } }}
      ondragover={e => { if (hasDropData(e)) { e.preventDefault(); if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy' } }}
      ondragleave={() => (dropping = Math.max(0, dropping - 1))}
      ondrop={onDrop}>
      <span class="lbl">{t('item.photos')} <span class="opt">{t('item.photosHint')}</span></span>
      <div class="photos">
        {#each photos as p, i (key(p))}
          <div class="ph">
            {#if 'remote' in p && p.broken}
              <span class="ph-link"><span>🔗</span><span class="small">{host(p.remote)}</span></span>
            {:else}
              <img src={'old' in p ? photoUrl(item, p.old) : 'file' in p ? p.url : p.remote} alt=""
                referrerpolicy="no-referrer" onerror={() => { if ('remote' in p) p.broken = true }} />
            {/if}
            <button type="button" class="x" aria-label={t('item.removePhoto')} onclick={() => removePhoto(i)}>✕</button>
            {#if i > 0}
              <button type="button" class="cover" title={t('item.makeCover')} aria-label={t('item.makeCover')}
                onclick={() => photos.unshift(...photos.splice(i, 1))}>★</button>
            {/if}
          </div>
        {/each}
        {#if !photos.length && linkPhoto && !clearLinkPhoto}
          <div class="ph from-link" title={t('item.fromLinkTitle')}>
            <img src={photoUrl(item, linkPhoto)} alt="" />
            <span class="tag">{t('item.fromLink')}</span>
            <button type="button" class="x" aria-label={t('item.dontUseLinkPhoto')} onclick={() => (clearLinkPhoto = true)}>✕</button>
          </div>
        {/if}
        {#if !full}
          <button type="button" class="ph add" onclick={() => fileInput.click()}><span>＋</span><span class="small">{t('item.upload')}</span></button>
          <button type="button" class="ph add" onclick={() => { urlOpen = true; requestAnimationFrame(() => urlInput?.focus()) }}>
            <span>🔗</span><span class="small">{t('item.fromUrl')}</span>
          </button>
        {/if}
      </div>
      {#if urlOpen}
        <div class="row">
          <input bind:this={urlInput} bind:value={urlValue} type="url" inputmode="url" placeholder={t('item.urlPlaceholder')}
            aria-label={t('item.urlLabel')} onkeydown={e => {
              if (e.key === 'Enter') { e.preventDefault(); if (addUrl(urlValue)) { urlValue = ''; urlOpen = false } }
              if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); urlOpen = false }
            }} />
          <button type="button" onclick={() => { if (addUrl(urlValue)) { urlValue = ''; urlOpen = false } }}>{t('common.add')}</button>
        </div>
      {/if}
      <p class="small muted hint">{t('item.pasteHint')}</p>
      {#if dropping > 0}<div class="drop-overlay">{t('item.dropToAdd')}</div>{/if}
      <input bind:this={fileInput} type="file" accept="image/*" multiple hidden onchange={e => addFiles(Array.from(e.currentTarget.files ?? []))} />
    </div>
    {#if error}<div class="error small">{error}</div>{/if}
  </form>

  {#snippet foot()}
    <button type="button" class="ghost danger" onclick={remove}>{t('common.remove')}</button>
    <span class="spacer"></span>
    <button type="button" onclick={() => (open = false)}>{t('common.cancel')}</button>
    <button class="primary" form="item-form" disabled={busy || !name.trim()}>
      {busy ? (photos.some(p => 'remote' in p) ? t('item.fetchingImages') : t('common.saving')) : t('common.save')}
    </button>
  {/snippet}
</Dialog>

<style>
  .opt { font-weight: 400; opacity: .8; }
  .opt::before { content: "· "; }
  .field-group { display: flex; flex-direction: column; gap: .4rem; }
  .lbl { font-size: .85rem; color: var(--muted); font-weight: 500; }
  .hint { margin: 0; }
  .drop { position: relative; border-radius: 12px; }
  .drop-overlay { position: absolute; inset: -.4rem; border: 2px dashed var(--accent); border-radius: 14px; display: grid; place-items: center;
    background: color-mix(in oklab, var(--panel) 80%, transparent); font-weight: 600; color: var(--accent-ink, var(--fg)); pointer-events: none; }
  .photos { display: grid; grid-template-columns: repeat(auto-fill, minmax(5.5rem, 1fr)); gap: .5rem; }
  .ph { position: relative; aspect-ratio: 1; border-radius: 10px; overflow: hidden; background: var(--band); }
  .ph img { width: 100%; height: 100%; object-fit: cover; }
  .ph-link { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: .2rem;
    padding: .3rem; text-align: center; color: var(--muted); word-break: break-all; }
  .ph .x, .ph .cover { position: absolute; top: .25rem; width: 1.7rem; height: 1.7rem; min-height: 0; padding: 0; border: 0; border-radius: 999px;
    background: rgb(0 0 0 / .55); color: #fff; font-size: .75rem; }
  .ph .x { right: .25rem; } .ph .cover { left: .25rem; }
  .from-link img { opacity: .8; }
  .from-link .tag { position: absolute; left: .25rem; bottom: .25rem; font-size: .68rem; padding: .05rem .4rem; border-radius: 999px; background: rgb(0 0 0 / .6); color: #fff; }
  .ph.add { border: 1.5px dashed var(--line); background: transparent; flex-direction: column; gap: 0; border-radius: 10px; color: var(--muted); padding: 0; white-space: normal; }
  .ph.add span:first-child { font-size: 1.3rem; }
  .add-link { align-self: flex-start; padding-inline: .4rem; color: var(--muted); }
</style>
