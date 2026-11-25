<script lang="ts">
  import { route } from './lib/router.svelte'
  import { toast } from './lib/ui.svelte'
  import AuthDialog from './lib/AuthDialog.svelte'
  import AskDialog from './lib/AskDialog.svelte'
  import Landing from './pages/Landing.svelte'
  import Dashboard from './pages/Dashboard.svelte'
  import Edit from './pages/Edit.svelte'
  import Share from './pages/Share.svelte'
  import NotFound from './pages/NotFound.svelte'
  import './lib/theme'

  const m = $derived(route.path.match(/^\/(e|l)\/([A-Za-z0-9]+)\/?$/))
</script>

{#if route.path === '/'}
  <Landing />
{:else if route.path === '/app' || route.path === '/app/'}
  <Dashboard />
{:else if m?.[1] === 'e'}
  {#key m[2]}<Edit token={m[2]} />{/key}
{:else if m?.[1] === 'l'}
  {#key m[2]}<Share share={m[2]} />{/key}
{:else}
  <NotFound />
{/if}

<AuthDialog />
<AskDialog />
{#if toast.text}{#key toast.n}<div class="toast" role="status">{toast.text}</div>{/key}{/if}
