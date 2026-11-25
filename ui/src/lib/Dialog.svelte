<script lang="ts">
  import type { Snippet } from 'svelte'
  let { open = $bindable(false), title = '', children, foot, wide = false }:
    { open?: boolean; title?: string; children: Snippet; foot?: Snippet; wide?: boolean } = $props()
  import { t } from './i18n/index.svelte'
  let el: HTMLDialogElement
  $effect(() => {
    if (open && !el.open) el.showModal()
    else if (!open && el.open) el.close()
  })
</script>

<!-- clicks that land on the <dialog> itself (not its content) are backdrop clicks -->
<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
<dialog bind:this={el} class:wide onclose={() => (open = false)} onclick={e => e.target === el && (open = false)}>
  {#if open}
    <div class="dlg">
      {#if title}
        <div class="head"><h2>{title}</h2><button class="icon-btn" type="button" aria-label={t('common.close')} onclick={() => (open = false)}>✕</button></div>
      {/if}
      <div class="body">{@render children()}</div>
      {#if foot}<div class="foot">{@render foot()}</div>{/if}
    </div>
  {/if}
</dialog>

<style>
  .dlg { display: flex; flex-direction: column; max-height: inherit; }
  .body { overflow: auto; }
  dialog.wide { width: min(46rem, calc(100vw - 2rem)); }
  @media (max-width: 600px) { dialog.wide { width: 100vw; } }
</style>
