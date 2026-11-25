<script lang="ts">
  import { t } from './i18n/index.svelte'
  import { current, toggle } from './theme'
  let theme = $state(current())
</script>

<button class="icon-btn" type="button" title={theme === 'dark' ? t('nav.themeToLight') : t('nav.themeToDark')}
  aria-label={theme === 'dark' ? t('nav.themeToLight') : t('nav.themeToDark')} onclick={() => (theme = toggle())}>{theme === 'dark' ? '☀' : '☾'}</button>
