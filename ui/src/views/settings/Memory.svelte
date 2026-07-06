<script>
  import { memory, memoryInfo, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';

  $: disabled = $memoryInfo?.enabled === false;

  async function save() {
    clearBanners();
    try {
      const saved = await putJSON('/api/memory', { content: $memory });
      memoryInfo.set(saved);
      memory.set(saved?.content || '');
      setNotice('Memory 已保存。');
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
        {disabled ? '已禁用' : $memoryInfo?.path || '尚未初始化'}
      </span>
    </div>
    <button type="button" class="primary" on:click={save} disabled={disabled}>保存</button>
  </div>
  <textarea class="code" bind:value={$memory} disabled={disabled} spellcheck="false"></textarea>
</div>
