<script lang="ts">
  import { i18n, LANGUAGES, setLanguage, t } from './i18n/index.svelte'
  import Menu from './Menu.svelte'
</script>

<Menu label={t('nav.language')} class="icon-btn lang-btn">
  {#snippet trigger()}<span class="code">{i18n.lang}</span>{/snippet}
  <div class="hint">{t('nav.language')}</div>
  <button type="button" role="menuitemradio" aria-checked={i18n.choice === 'auto'} onclick={() => setLanguage('auto')}>
    <span class="tick">{i18n.choice === 'auto' ? '✓' : ''}</span>{t('nav.languageAuto')}
  </button>
  {#each LANGUAGES as l (l.code)}
    <!-- language names stay in their own language, so people can always find theirs -->
    <button type="button" role="menuitemradio" aria-checked={i18n.choice === l.code} lang={l.code} onclick={() => setLanguage(l.code)}>
      <span class="tick">{i18n.choice === l.code ? '✓' : ''}</span>{l.name}
    </button>
  {/each}
</Menu>

<style>
  .code { font-size: .78rem; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; }
  .tick { display: inline-block; width: 1.2rem; }
</style>
