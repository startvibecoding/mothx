<script>
  import { serveConfig, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';
  import { navigate } from '../../lib/router.js';

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
      setNotice('工作目录配置已保存。');
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
      <h3>工作目录</h3>
      <span class="hint">gateway.workingDir — Agent 的默认工作目录</span>
    </div>
    <button type="button" class="primary" on:click={save}>保存</button>
  </div>
  <div class="form-body">
    <label>
      <span>默认工作目录</span>
      <input
        bind:value={workingDir}
        on:keydown={handleKeydown}
        placeholder="/home/user/projects"
      />
      <span class="hint">Agent 执行命令时的默认 cwd。留空则使用进程当前目录。</span>
    </label>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>允许的工作目录</h3>
      <span class="hint">gateway.allowedWorkDirs — 安全白名单</span>
    </div>
    <div class="card-head-actions">
      <button type="button" class="sm" on:click={addAllowed}>+ 添加</button>
      <button type="button" class="primary" on:click={save}>保存</button>
    </div>
  </div>
  <div class="form-body">
    {#if allowedWorkDirs.length === 0}
      <p class="empty">
        未配置白名单。未配置时，Agent 可以切换到任意目录（仅受 sandbox 限制）。
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
              title="移除"
              on:click={() => removeAllowed(i)}
            >
              ×
            </button>
          </div>
        {/each}
      </div>
    {/if}
    <p class="hint">
      设为空数组 <code>[]</code> 表示禁止 Agent 覆盖工作目录。设为 <code>null</code>（删除整个字段）表示不限制。
    </p>
  </div>
</div>
