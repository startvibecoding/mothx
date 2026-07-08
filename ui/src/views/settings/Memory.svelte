<script>
  import { memory, memoryInfo, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';
  import { t } from '../../lib/preferences.js';

  $: disabled = $memoryInfo?.enabled === false;

  async function save() {
    clearBanners();
    try {
      const saved = await putJSON('/api/memory', { content: $memory });
      memoryInfo.set(saved);
      memory.set(saved?.content || '');
      setNotice($t('settings.memory.saved'));
    } catch (err) {
      setError(err);
    }
  }
</script>

<div class="card editor-card">
  <div class="card-head">
    <div>
      <h3>Memory</h3>
      <span class="hint">
        {disabled ? $t('common.disabledState') : $memoryInfo?.path || $t('settings.memory.notInitialized')}
      </span>
    </div>
    <button type="button" class="primary" on:click={save} disabled={disabled}>{$t('common.save')}</button>
  </div>
  <textarea class="code" bind:value={$memory} disabled={disabled} spellcheck="false"></textarea>
</div>
