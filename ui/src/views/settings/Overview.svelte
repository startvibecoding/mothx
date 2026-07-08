<script>
  import { navigate } from '../../lib/router.js';
  import { status, health, memoryInfo, cronInfo, features } from '../../lib/stores.js';
  import { t } from '../../lib/preferences.js';

  const groups = [
    { key: 'serve', title: 'settings.tabs.serve', desc: 'settings.overview.serve.desc' },
    { key: 'app', title: 'settings.tabs.app', desc: 'settings.overview.app.desc' },
    { key: 'memory', title: 'settings.tabs.memory', desc: 'settings.overview.memory.desc' },
    { key: 'features', title: 'settings.tabs.features', desc: 'settings.overview.features.desc' }
  ];
</script>

<div class="grid-cards">
  {#each groups as g}
    <button type="button" class="card link" on:click={() => navigate(`/settings/${g.key}`)}>
      <div class="card-head"><h3>{$t(g.title)}</h3></div>
      <p class="card-desc">{$t(g.desc)}</p>
    </button>
  {/each}
</div>

<div class="card">
  <div class="card-head"><h3>{$t('settings.runtime.title')}</h3></div>
  <dl class="kv">
    <dt>{$t('settings.runtime.version')}</dt><dd>{$health?.version || 'dev'}</dd>
    <dt>{$t('settings.runtime.listen')}</dt><dd>{$status?.listen || '—'}</dd>
    <dt>{$t('settings.runtime.sessions')}</dt><dd>{$status?.sessions ?? $health?.sessions ?? 0}</dd>
    <dt>Cron</dt><dd>{$cronInfo?.enabled === false ? $t('common.disabledState') : ($cronInfo?.running ? $t('common.running') : $t('common.idle'))}</dd>
    <dt>Memory</dt><dd>{$memoryInfo?.enabled === false ? $t('common.disabledState') : ($memoryInfo?.path || $t('common.uninitialized'))}</dd>
    <dt>{$t('settings.runtime.api')}</dt><dd>{$features.api ? $t('common.enabled') : $t('common.disabled')}</dd>
  </dl>
</div>
