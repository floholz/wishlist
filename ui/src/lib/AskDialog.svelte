<script lang="ts">
  import { answer, asking } from './ask.svelte'
  import Dialog from './Dialog.svelte'
  import { t } from './i18n/index.svelte'

  let cancelBtn: HTMLButtonElement | undefined = $state()
  let confirmBtn: HTMLButtonElement | undefined = $state()
  let copyBox: HTMLInputElement | undefined = $state()

  // closing via Esc / backdrop / ✕ counts as "cancel"
  $effect(() => { if (!asking.open) answer(false) })

  // focus the safe choice for destructive questions, the main one otherwise
  $effect(() => {
    if (!asking.open) return
    requestAnimationFrame(() => {
      if (asking.opts.copyText) copyBox?.select()
      else (asking.opts.danger && cancelBtn ? cancelBtn : confirmBtn)?.focus()
    })
  })
</script>

<Dialog bind:open={asking.open} title={asking.opts.title}>
  {#if asking.opts.message}<p class="msg">{asking.opts.message}</p>{/if}
  {#if asking.opts.copyText}
    <input bind:this={copyBox} class="copy" readonly value={asking.opts.copyText} aria-label={t('common.textToCopy')} onfocus={e => e.currentTarget.select()} />
  {/if}
  {#snippet foot()}
    <span class="spacer"></span>
    {#if asking.opts.cancel !== null}
      <button bind:this={cancelBtn} type="button" onclick={() => answer(false)}>{asking.opts.cancel ?? t('common.cancel')}</button>
    {/if}
    <button bind:this={confirmBtn} type="button" class="primary" class:danger-fill={asking.opts.danger} onclick={() => answer(true)}>
      {asking.opts.confirm ?? t('common.ok')}
    </button>
  {/snippet}
</Dialog>

<style>
  .msg { margin: 0; color: var(--muted); }
  .copy { margin-top: .75rem; font-size: .88rem; }
  .danger-fill { background: var(--danger); color: #fff; }
  .danger-fill:hover:not(:disabled) { background: color-mix(in oklab, var(--danger) 85%, #000); }
</style>
