<script>
  import { settings, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';

  async function save() {
    clearBanners();
    try {
      const parsed = JSON.parse($settings);
      const saved = await putJSON('/api/settings', parsed);
      settings.set(JSON.stringify(saved, null, 2));
      setNotice('应用设置已保存。');
    } catch (err) {
      setError(err);
    }
  }
</script>

<div class="card editor-card">
  <div class="card-head">
    <div>
      <h3>应用设置</h3>
      <span class="hint">/api/settings — Providers / Approval / Context 等</span>
    </div>
    <button type="button" class="primary" on:click={save}>保存</button>
  </div>
  <textarea class="code" bind:value={$settings} spellcheck="false"></textarea>
</div>
