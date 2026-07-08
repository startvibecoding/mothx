<script>
  import { settings, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';
  import { t } from '../../lib/preferences.js';

  async function save() {
    clearBanners();
    try {
      const parsed = JSON.parse($settings);
      const saved = await putJSON('/api/settings', parsed);
      settings.set(JSON.stringify(saved, null, 2));
      setNotice($t('settings.app.saved'));
    } catch (err) {
      setError(err);
    }
  }
</script>

<div class="card editor-card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.tabs.app')}</h3>
      <span class="hint">{$t('settings.app.hint')}</span>
    </div>
    <button type="button" class="primary" on:click={save}>{$t('common.save')}</button>
  </div>
  <textarea class="code" bind:value={$settings} spellcheck="false"></textarea>
</div>
