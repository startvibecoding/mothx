<script>
  import { route } from '../lib/router.js';
  import { t } from '../lib/preferences.js';

  const titles = {
    chat: 'nav.newChat',
    sessions: 'nav.sessions',
    skills: 'nav.skills',
    stats: 'nav.stats',
    cron: 'nav.cron',
    settings: 'nav.settings'
  };

  const subtitles = {
    chat: 'topbar.chat.subtitle',
    sessions: 'topbar.sessions.subtitle',
    skills: 'topbar.skills.subtitle',
    stats: 'topbar.stats.subtitle',
    cron: 'topbar.cron.subtitle',
    settings: 'topbar.settings.subtitle'
  };

  $: title = titles[$route.section] ? $t(titles[$route.section]) : $route.section;
  $: subtitle = subtitles[$route.section] ? $t(subtitles[$route.section]) : '';
</script>

<header class="topbar">
  <div class="tb-title">
    <h1>{title}</h1>
    {#if subtitle}<span>{subtitle}</span>{/if}
  </div>
</header>
