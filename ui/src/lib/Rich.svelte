<script lang="ts">
  // Renders a translated sentence with **bold**, _emphasis_ and one [action]
  // (a link with `href`, or a button with `onclick`). Everything is text — no HTML.
  let { text, href, onclick }: { text: string; href?: string; onclick?: () => void } = $props()

  type Part = { kind: 'text' | 'b' | 'em' | 'action'; value: string }
  const parts = $derived.by(() => {
    const out: Part[] = []
    const re = /\*\*(.+?)\*\*|_(.+?)_|\[(.+?)\]/g
    let last = 0
    for (const m of text.matchAll(re)) {
      if (m.index! > last) out.push({ kind: 'text', value: text.slice(last, m.index) })
      out.push(m[1] ? { kind: 'b', value: m[1] } : m[2] ? { kind: 'em', value: m[2] } : { kind: 'action', value: m[3] })
      last = m.index! + m[0].length
    }
    if (last < text.length) out.push({ kind: 'text', value: text.slice(last) })
    return out
  })
</script>

{#each parts as p}{#if p.kind === 'b'}<b>{p.value}</b>{:else if p.kind === 'em'}<em>{p.value}</em>{:else if p.kind === 'action'}{#if href}<a {href}>{p.value}</a>{:else}<button type="button" class="rich-action" {onclick}>{p.value}</button>{/if}{:else}{p.value}{/if}{/each}

<style>
  .rich-action { all: unset; cursor: pointer; text-decoration: underline; color: var(--fg); }
  .rich-action:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; border-radius: 3px; }
</style>
