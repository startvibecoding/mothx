<script>
  import { serveConfig, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';

  async function save() {
    clearBanners();
    try {
      const parsed = JSON.parse($serveConfig);
      const saved = await putJSON('/api/serve/config', parsed);
      serveConfig.set(JSON.stringify(saved, null, 2));
      setNotice('Serve 配置已保存。监听或通道变更需要重启生效。');
    } catch (err) {
      setError(err);
    }
  }
</script>

<div class="card editor-card">
  <div class="card-head">
    <div>
      <h3>Serve 配置</h3>
      <span class="hint">/api/serve/config</span>
    </div>
    <button type="button" class="primary" on:click={save}>保存</button>
  </div>
  <textarea class="code" bind:value={$serveConfig} spellcheck="false"></textarea>
</div>
