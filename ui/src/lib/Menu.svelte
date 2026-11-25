<script lang="ts">
  import type { Snippet } from 'svelte'
  import { t } from './i18n/index.svelte'
  let { label = t('common.more'), trigger, children, class: cls = 'icon-btn' }:
    { label?: string; trigger?: Snippet; children: Snippet; class?: string } = $props()
  let open = $state(false)
  let root: HTMLElement
</script>

<svelte:window onclick={e => open && !root.contains(e.target as Node) && (open = false)}
  onkeydown={e => e.key === 'Escape' && (open = false)} />

<div class="wrap" bind:this={root}>
  <button type="button" class={cls} aria-label={label} aria-haspopup="menu" aria-expanded={open}
    onclick={e => { e.preventDefault(); e.stopPropagation(); open = !open }}>
    {#if trigger}{@render trigger()}{:else}⋮{/if}
  </button>
  {#if open}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div class="menu" role="menu" tabindex="-1" onclick={e => { e.stopPropagation(); if ((e.target as Element).closest('button,a')) open = false }}>
      {@render children()}
    </div>
  {/if}
</div>

<style>.wrap { position: relative; display: inline-flex; }</style>
