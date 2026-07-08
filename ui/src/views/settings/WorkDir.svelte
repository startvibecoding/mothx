<script>
  import { serveConfig, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';
  import { t } from '../../lib/preferences.js';

  let defaultWorkDir = '';
  let restrictWorkDirs = false;
  let allowedWorkDirs = [];

  $: parseConfig($serveConfig);

  function parseConfig(raw) {
    try {
      const cfg = JSON.parse(raw);
      defaultWorkDir = readDefaultWorkDir(cfg);
      const rawDirs = readAllowedWorkDirs(cfg);
      restrictWorkDirs = Array.isArray(rawDirs);
      allowedWorkDirs = Array.isArray(rawDirs) ? [...rawDirs] : [];
    } catch {
      defaultWorkDir = '';
      restrictWorkDirs = false;
      allowedWorkDirs = [];
    }
  }

  function readDefaultWorkDir(cfg) {
    for (const source of [cfg, cfg?.api, cfg?.gateway]) {
      if (!source || typeof source !== 'object') continue;
      const value = source.defaultWorkDir || source.workDir || source.workingDir;
      if (value) return value;
    }
    return '';
  }

  function readAllowedWorkDirs(cfg) {
    for (const source of [cfg, cfg?.api, cfg?.gateway]) {
      if (!source || typeof source !== 'object') continue;
      if (Object.prototype.hasOwnProperty.call(source, 'allowedWorkDirs')) {
        return source.allowedWorkDirs;
      }
    }
    return undefined;
  }

  function addAllowed() {
    restrictWorkDirs = true;
    allowedWorkDirs = [...allowedWorkDirs, ''];
  }

  function removeAllowed(index) {
    allowedWorkDirs = allowedWorkDirs.filter((_, i) => i !== index);
  }

  async function save() {
    clearBanners();
    try {
      const cfg = JSON.parse($serveConfig);
      const nextDefaultWorkDir = defaultWorkDir.trim();
      if (nextDefaultWorkDir) cfg.defaultWorkDir = nextDefaultWorkDir;
      else delete cfg.defaultWorkDir;
      delete cfg.workDir;
      clearLegacyWorkDirFields(cfg.api);
      clearLegacyWorkDirFields(cfg.gateway);

      const filtered = allowedWorkDirs.map((d) => d.trim()).filter(Boolean);
      if (restrictWorkDirs) cfg.allowedWorkDirs = filtered;
      else delete cfg.allowedWorkDirs;
      clearAllowedWorkDirs(cfg.api);
      clearAllowedWorkDirs(cfg.gateway);

      const saved = await putJSON('/api/serve/config', cfg);
      serveConfig.set(JSON.stringify(saved, null, 2));
      setNotice($t('settings.workdir.saved'));
    } catch (err) {
      setError(err);
    }
  }

  function clearLegacyWorkDirFields(source) {
    if (!source || typeof source !== 'object') return;
    delete source.defaultWorkDir;
    delete source.workDir;
    delete source.workingDir;
  }

  function clearAllowedWorkDirs(source) {
    if (!source || typeof source !== 'object') return;
    delete source.allowedWorkDirs;
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
        bind:value={defaultWorkDir}
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
    <label class="checkbox-row">
      <input type="checkbox" bind:checked={restrictWorkDirs} />
      <span>{$t('settings.workdir.restrict')}</span>
    </label>
    <p class="hint">
      {$t('settings.workdir.restrictHint')}
    </p>

    {#if !restrictWorkDirs}
      <p class="empty">
        {$t('settings.workdir.noWhitelist')}
      </p>
    {:else if allowedWorkDirs.length === 0}
      <p class="empty">
        {$t('settings.workdir.denyAll')}
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
