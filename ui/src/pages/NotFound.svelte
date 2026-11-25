<script lang="ts">
  import { t } from '../lib/i18n/index.svelte'
  import TopBar from '../lib/TopBar.svelte'
  let { message }: { message?: string } = $props()
</script>

<TopBar />
<main class="page" style="text-align:center;padding-top:4rem">
  <h1 style="font-family:var(--serif);font-size:2rem">{t('notFound.title')}</h1>
  <p class="muted">{message ?? t('notFound.message')}</p>
  <a class="btn primary" href="/app">{t('notFound.back')}</a>
</main>
