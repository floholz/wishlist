<script lang="ts">
  import { editUrl, shareUrl, type List } from './api'
  import Dialog from './Dialog.svelte'
  import { t } from './i18n/index.svelte'
  import Rich from './Rich.svelte'
  import { copy, session } from './ui.svelte'

  let { open = $bindable(false), list }: { open?: boolean; list: List } = $props()
  const canShare = typeof navigator !== 'undefined' && !!navigator.share

  function nativeShare() {
    navigator.share({ title: list.title, text: t('shareDialog.shareText', { title: list.title }), url: shareUrl(list.share) }).catch(() => {})
  }
</script>

<Dialog bind:open title={t('shareDialog.title')}>
  <div class="stack">
    <section class="link-box share">
      <div class="row"><b>{t('links.share')}</b><span class="tag">{t('shareDialog.forFriends')}</span></div>
      <p class="muted small">{t('shareDialog.shareHint')}</p>
      <div class="row url"><code>{shareUrl(list.share)}</code></div>
      <div class="row">
        <button type="button" class="primary" onclick={() => copy(shareUrl(list.share), t('links.share'))}>{t('shareDialog.copyShare')}</button>
        {#if canShare}<button type="button" onclick={nativeShare}>{t('shareDialog.shareNative')}</button>{/if}
        <a class="btn ghost" href="/l/{list.share}">{t('shareDialog.preview')}</a>
      </div>
    </section>

    <section class="link-box">
      <div class="row"><b>{t('links.edit')}</b><span class="tag secret">{t('shareDialog.keepPrivate')}</span></div>
      <p class="muted small">
        {t('shareDialog.editHint')}
        {#if list.mine}{t('shareDialog.editSaved')}
        {:else}<Rich text={t('shareDialog.editBookmark')} onclick={() => { open = false; session.authOpen = true }} />{/if}
      </p>
      <div class="row url"><code>{editUrl(list.edit!)}</code></div>
      <div class="row"><button type="button" onclick={() => copy(editUrl(list.edit!), t('links.edit'))}>{t('shareDialog.copyEdit')}</button></div>
    </section>
  </div>
</Dialog>

<style>
  .link-box { border: 1px solid var(--line); border-radius: 12px; padding: .9rem 1rem; display: flex; flex-direction: column; gap: .5rem; }
  .link-box.share { border-color: var(--accent); }
  .link-box p { margin: 0; }
  .row { flex-wrap: wrap; }
  .tag { font-size: .75rem; padding: .1rem .5rem; border-radius: 999px; background: var(--accent-soft, var(--band)); color: var(--accent-ink, var(--fg)); }
  .tag.secret { background: var(--band); color: var(--muted); }
  .url code { font-size: .82rem; background: var(--band); padding: .45rem .6rem; border-radius: 8px; word-break: break-all; width: 100%; user-select: all; }
</style>
