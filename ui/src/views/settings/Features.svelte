<script>
  import { features } from '../../lib/stores.js';
  import { t } from '../../lib/preferences.js';

  const rows = [
    { key: 'api', label: 'settings.features.api.label', desc: 'settings.features.api.desc' },
    { key: 'webUI', label: 'settings.features.webUI.label', desc: 'settings.features.webUI.desc' },
    { key: 'websocket', label: 'settings.features.websocket.label', desc: 'settings.features.websocket.desc' },
    { key: 'cron', label: 'settings.features.cron.label', desc: 'settings.features.cron.desc' },
    { key: 'memory', label: 'settings.features.memory.label', desc: 'settings.features.memory.desc' },
    { key: 'multiAgent', label: 'settings.features.multiAgent.label', desc: 'settings.features.multiAgent.desc' }
  ];
</script>

<div class="card">
  <div class="card-head">
    <h3>{$t('settings.tabs.features')}</h3>
    <span class="hint">{$t('settings.features.hint')}</span>
  </div>
  <table class="table">
    <thead>
      <tr>
        <th>{$t('settings.features.feature')}</th>
        <th>{$t('settings.features.status')}</th>
        <th>{$t('settings.features.description')}</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as row}
        <tr>
          <td>{$t(row.label)}</td>
          <td>
            <span class="pill" class:on={$features[row.key]} class:off={!$features[row.key]}>
              {$features[row.key] ? $t('common.enabled') : $t('common.disabled')}
            </span>
          </td>
          <td class="hint">{$t(row.desc)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
