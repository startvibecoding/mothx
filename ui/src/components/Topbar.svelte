<script>
  import { route } from '../lib/router.js';
  import { refreshAll, connectLogs, disconnectLogs, logsConnected } from '../lib/stores.js';
  import { t } from '../lib/preferences.js';

  const titles = {
    chat: 'nav.newChat',
    sessions: 'nav.sessions',
    cron: 'nav.cron',
    channels: 'nav.channels',
    logs: 'nav.logs',
    settings: 'nav.settings'
  };

  const subtitles = {
    chat: 'topbar.chat.subtitle',
    sessions: 'topbar.sessions.subtitle',
    cron: 'topbar.cron.subtitle',
    channels: 'topbar.channels.subtitle',
    logs: 'topbar.logs.subtitle',
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
  <div class="tb-actions">
    {#if $logsConnected}
      <button type="button" class="ghost" on:click={disconnectLogs}>{$t('topbar.closeLogs')}</button>
    {:else}
      <button type="button" class="ghost" on:click={connectLogs}>{$t('topbar.openLogs')}</button>
    {/if}
    <button type="button" class="ghost" on:click={refreshAll}>{$t('topbar.refresh')}</button>
  </div>
</header>
