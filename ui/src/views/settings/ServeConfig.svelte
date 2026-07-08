<script>
  import { serveConfig, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';
  import { t } from '../../lib/preferences.js';

  async function save() {
    clearBanners();
    try {
      const parsed = JSON.parse($serveConfig);
      const saved = await putJSON('/api/serve/config', parsed);
      serveConfig.set(JSON.stringify(saved, null, 2));
      setNotice($t('settings.serve.saved'));
    } catch (err) {
      setError(err);
    }
  }
</script>

<div class="card editor-card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.tabs.serve')}</h3>
      <span class="hint">/api/serve/config</span>
    </div>
    <button type="button" class="primary" on:click={save}>{$t('common.save')}</button>
  </div>
  <textarea class="code" bind:value={$serveConfig} spellcheck="false"></textarea>
</div>
