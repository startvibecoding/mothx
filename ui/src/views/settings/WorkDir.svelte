<script>
  import { serveConfig, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';
  import { t } from '../../lib/preferences.js';

  let workingDir = '';
  let allowedWorkDirs = [];

  $: parseConfig($serveConfig);

  function parseConfig(raw) {
    try {
      const cfg = JSON.parse(raw);
      workingDir = cfg?.gateway?.workingDir || '';
      const rawDirs = cfg?.gateway?.allowedWorkDirs;
      allowedWorkDirs = Array.isArray(rawDirs) ? [...rawDirs] : [];
    } catch {
      workingDir = '';
      allowedWorkDirs = [];
    }
  }

  function addAllowed() {
    allowedWorkDirs = [...allowedWorkDirs, ''];
  }

  function removeAllowed(index) {
    allowedWorkDirs = allowedWorkDirs.filter((_, i) => i !== index);
  }

  async function save() {
    clearBanners();
    try {
      const cfg = JSON.parse($serveConfig);
      if (!cfg.gateway) cfg.gateway = {};
      cfg.gateway.workingDir = workingDir.trim();
      const filtered = allowedWorkDirs.map((d) => d.trim()).filter(Boolean);
      cfg.gateway.allowedWorkDirs = filtered.length > 0 ? filtered : undefined;
      const saved = await putJSON('/api/serve/config', cfg);
      serveConfig.set(JSON.stringify(saved, null, 2));
      setNotice($t('settings.workdir.saved'));
    } catch (err) {
      setError(err);
    }
  }

  function handleKeydown(event) {
    if (event.key === 'Enter') {
      event.preventDefault();
      save();
    }
  }
</script>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.workdir.title')}</h3>
      <span class="hint">{$t('settings.workdir.mainHint')}</span>
    </div>
    <button type="button" class="primary" on:click={save}>{$t('common.save')}</button>
  </div>
  <div class="form-body">
    <label>
      <span>{$t('settings.workdir.default')}</span>
      <input
        bind:value={workingDir}
        on:keydown={handleKeydown}
        placeholder="/home/user/projects"
      />
      <span class="hint">{$t('settings.workdir.defaultHint')}</span>
    </label>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.workdir.allowed')}</h3>
      <span class="hint">{$t('settings.workdir.allowedHint')}</span>
    </div>
    <div class="card-head-actions">
      <button type="button" class="sm" on:click={addAllowed}>+ {$t('common.add')}</button>
      <button type="button" class="primary" on:click={save}>{$t('common.save')}</button>
    </div>
  </div>
  <div class="form-body">
    {#if allowedWorkDirs.length === 0}
      <p class="empty">
        {$t('settings.workdir.noWhitelist')}
      </p>
    {:else}
      <div class="dir-list">
        {#each allowedWorkDirs as dir, i (i)}
          <div class="dir-row">
            <input
              bind:value={allowedWorkDirs[i]}
              placeholder="/home/user/projects"
              class="dir-input"
            />
            <button
              type="button"
              class="ghost danger"
              title={$t('common.remove')}
              on:click={() => removeAllowed(i)}
            >
              ×
            </button>
          </div>
        {/each}
      </div>
    {/if}
    <p class="hint">
      {$t('settings.workdir.arrayHint')}
    </p>
  </div>
</div>
