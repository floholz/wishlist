<script lang="ts">
  import { t } from '../lib/i18n/index.svelte'
  import { ownedLists } from '../lib/local'
  import { PRESETS, presetLabel, presetStyle, preset } from '../lib/presets'
  import Rich from '../lib/Rich.svelte'
  import TopBar from '../lib/TopBar.svelte'
  import { session } from '../lib/ui.svelte'

  const returning = ownedLists().length > 0 || !!session.user
  const xmas = preset('christmas')
  const steps = [
    ['landing.step1Title', 'landing.step1'],
    ['landing.step2Title', 'landing.step2'],
    ['landing.step3Title', 'landing.step3'],
  ] as const
  const features = [
    ['landing.f1Title', 'landing.f1'], ['landing.f2Title', 'landing.f2'], ['landing.f3Title', 'landing.f3'],
    ['landing.f4Title', 'landing.f4'], ['landing.f5Title', 'landing.f5'], ['landing.f6Title', 'landing.f6'],
  ] as const
</script>

<TopBar home="/">
  {#if returning}<a class="btn ghost hide-sm" href="/app">{t('landing.yourLists')}</a>{/if}
</TopBar>

<main>
  <section class="hero-l wrap">
    <div class="copy">
      <p class="eyebrow">{t('landing.eyebrow')}</p>
      <h1>{t('landing.heroLine1')}<br /><Rich text={t('landing.heroLine2')} /></h1>
      <p class="lede">{t('landing.lede')}</p>
      <div class="row cta">
        <a class="btn primary big" href="/app">{returning ? t('landing.ctaReturning') : t('landing.ctaStart')}</a>
        <a class="btn ghost big" href="#how">{t('landing.howLink')}</a>
      </div>
      <p class="small muted">{t('landing.free')}</p>
    </div>

    <!-- a little preview of a shared list -->
    <div class="mock tinted" style={presetStyle('christmas')} aria-hidden="true">
      <div class="mock-hero" style="background-image:url({xmas.image})"><span>{t('landing.mock.title')}</span></div>
      <ul>
        <li><span class="rank">1</span><span class="n">{t('landing.mock.item1')}</span><span class="pill">{t('landing.mock.reserve')}</span></li>
        <li class="taken"><span class="rank">2</span><span class="n">{t('landing.mock.item2')}</span><span class="pill off">{t('landing.mock.taken')}</span></li>
        <li class="mine"><span class="rank">3</span><span class="n">{t('landing.mock.item3')}</span><span class="pill on">{t('landing.mock.mine')}</span></li>
        <li><span class="rank">4</span><span class="n">{t('landing.mock.item4')}</span><span class="pill">{t('landing.mock.reserve')}</span></li>
      </ul>
    </div>
  </section>

  <section id="how" class="band">
    <div class="wrap">
      <h2>{t('landing.howTitle')}</h2>
      <ol class="steps">
        {#each steps as [title, text], i}
          <li>
            <span class="num">{i + 1}</span>
            <h3>{t(title)}</h3>
            <p><Rich text={t(text)} /></p>
          </li>
        {/each}
      </ol>
    </div>
  </section>

  <section class="wrap features">
    {#each features as [title, text]}
      <div><h3>{t(title)}</h3><p class="muted">{t(text)}</p></div>
    {/each}
  </section>

  <section class="wrap themes-l">
    <h2>{t('landing.themesTitle')}</h2>
    <div class="tiles">
      {#each PRESETS as p (p.id)}
        <a class="tile" href="/app" style="{presetStyle(p.id)}background-image:url({p.image})"><span>{presetLabel(p.id)}</span></a>
      {/each}
    </div>
  </section>

  <section class="wrap closing">
    <h2>{t('landing.closing')}</h2>
    <a class="btn primary big" href="/app">{t('landing.ctaStart')}</a>
  </section>
</main>

<footer class="wrap foot small muted">{t('landing.footer')}</footer>

<style>
  .wrap { max-width: 68rem; margin: 0 auto; padding-inline: 1.25rem; }
  h2 { font-family: var(--serif); font-size: clamp(1.6rem, 4vw, 2.2rem); font-weight: 600; letter-spacing: -.01em; }
  .big { min-height: 3rem; padding-inline: 1.5rem; font-size: 1.02rem; }
  @media (max-width: 520px) { :global(.hide-sm) { display: none !important; } }

  .hero-l { display: grid; grid-template-columns: 1.15fr 1fr; gap: 3rem; align-items: center; padding-top: clamp(2rem, 7vw, 5.5rem); padding-bottom: clamp(2.5rem, 7vw, 5.5rem); }
  @media (max-width: 860px) { .hero-l { grid-template-columns: 1fr; gap: 2.5rem; } }
  .eyebrow { text-transform: uppercase; letter-spacing: .12em; font-size: .78rem; font-weight: 600; color: var(--muted); margin: 0 0 1rem; }
  h1 { font-family: var(--serif); font-size: clamp(2.4rem, 7vw, 4rem); font-weight: 600; letter-spacing: -.02em; line-height: 1.05; }
  h1 :global(em) { font-style: italic; color: #b70000; }
  .lede { font-size: 1.15rem; color: var(--muted); max-width: 34rem; margin: 1.25rem 0 1.75rem; }
  .cta { flex-wrap: wrap; margin-bottom: .75rem; }
  .btn.primary { background: var(--fg); color: var(--bg); }
  .btn.primary:hover { background: color-mix(in oklab, var(--fg) 82%, var(--bg)); }

  .mock { background: var(--panel); border: 1px solid var(--line); border-radius: 20px; overflow: hidden; box-shadow: var(--shadow); transform: rotate(1.2deg); max-width: 26rem; justify-self: center; width: 100%; }
  .mock-hero { height: 8.5rem; background: var(--accent) center / cover; display: flex; align-items: flex-end; position: relative; }
  .mock-hero::after { content: ""; position: absolute; inset: 0; background: linear-gradient(to top, rgb(0 0 0 / .5), transparent 70%); }
  .mock-hero span { position: relative; z-index: 1; color: #fff; font-family: var(--serif); font-size: 1.35rem; font-weight: 600; padding: .8rem 1rem; }
  .mock ul { list-style: none; margin: 0; padding: .75rem; display: flex; flex-direction: column; gap: .45rem; }
  .mock li { display: flex; align-items: center; gap: .6rem; padding: .6rem .7rem; border: 1px solid var(--line); border-radius: 10px; font-size: .9rem; }
  .mock li.taken { opacity: .5; } .mock li.taken .n { text-decoration: line-through; }
  .mock li.mine { border-color: var(--accent); }
  .mock .n { flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .pill { font-size: .75rem; padding: .2rem .6rem; border-radius: 999px; background: var(--accent); color: #fff; white-space: nowrap; }
  .pill.off { background: var(--band); color: var(--muted); }
  .pill.on { background: var(--accent-soft); color: var(--accent-ink); font-weight: 600; }

  .band { background: var(--band); padding: 4rem 0; }
  .steps { list-style: none; padding: 0; margin: 2rem 0 0; display: grid; grid-template-columns: repeat(3, 1fr); gap: 2rem; }
  @media (max-width: 760px) { .steps { grid-template-columns: 1fr; gap: 1.5rem; } }
  .num { display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: 999px; background: var(--fg); color: var(--bg); font-weight: 700; margin-bottom: .8rem; }
  .steps h3 { font-size: 1.15rem; margin-bottom: .4rem; }
  .steps p { margin: 0; color: var(--muted); }

  .features { display: grid; grid-template-columns: repeat(3, 1fr); gap: 2rem 2.5rem; padding-top: 4rem; padding-bottom: 4rem; }
  @media (max-width: 760px) { .features { grid-template-columns: 1fr 1fr; gap: 1.5rem; } }
  @media (max-width: 460px) { .features { grid-template-columns: 1fr; } }
  .features h3 { font-size: 1.02rem; }
  .features p { margin: .3rem 0 0; }

  .themes-l { padding-bottom: 4rem; }
  .tiles { display: grid; grid-template-columns: repeat(5, 1fr); gap: .8rem; margin-top: 1.5rem; }
  @media (max-width: 760px) { .tiles { grid-template-columns: repeat(auto-fill, minmax(8.5rem, 1fr)); } }
  .tile { aspect-ratio: 3 / 4; border-radius: 12px; background: var(--accent) center / cover; position: relative; overflow: hidden; display: flex; align-items: flex-end;
    text-decoration: none; color: #fff; font-weight: 600; padding: .7rem .8rem; transition: transform .15s; border: 1px solid var(--line); }
  .tile:hover { transform: translateY(-3px); }
  .tile::after { content: ""; position: absolute; inset: 0; background: linear-gradient(to top, rgb(0 0 0 / .5), transparent 55%); }
  .tile span { position: relative; z-index: 1; }

  .closing { text-align: center; padding-top: 3rem; padding-bottom: 4rem; display: flex; flex-direction: column; align-items: center; gap: 1.5rem; }
  .foot { padding-block: 2rem calc(2rem + env(safe-area-inset-bottom)); border-top: 1px solid var(--line); text-align: center; }
</style>
