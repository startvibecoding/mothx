<script>
  import { cronInfo, refreshCron, setError, setNotice, clearBanners } from '../lib/stores.js';
  import { postJSON, patchJSON, del } from '../lib/api.js';
  import { shortID, scheduleLabel, formatDateTime } from '../lib/format.js';

  let form = { name: '', prompt: '', schedule: '', oneshot: false, mode: 'yolo' };

  $: info = $cronInfo;
  $: disabled = info?.enabled === false;
  $: jobs = info?.jobs || [];

  async function create() {
    if (!form.name.trim() || !form.prompt.trim()) return;
    clearBanners();
    try {
      await postJSON('/api/cron', {
        name: form.name.trim(),
        prompt: form.prompt,
        schedule: form.schedule.trim(),
        oneshot: form.oneshot,
        mode: form.mode
      });
      form = { name: '', prompt: '', schedule: '', oneshot: false, mode: 'yolo' };
      setNotice('已创建定时任务');
      await refreshCron();
    } catch (err) {
      setError(err);
    }
  }

  async function toggle(id, enabled) {
    clearBanners();
    try {
      await patchJSON(`/api/cron/${encodeURIComponent(id)}`, { enabled });
      await refreshCron();
    } catch (err) {
      setError(err);
    }
  }

  async function remove(id) {
    clearBanners();
    try {
      await del(`/api/cron/${encodeURIComponent(id)}`);
      setNotice(`已删除任务 ${shortID(id)}`);
      await refreshCron();
    } catch (err) {
      setError(err);
    }
  }
</script>

<section class="page">
  <div class="page-toolbar">
    <div class="status-tag" class:muted={disabled}>
      {disabled ? '已禁用' : info?.running ? '运行中' : '空闲'}
      {#if info?.path}<span class="hint">{info.path}</span>{/if}
    </div>
    <button type="button" class="ghost" on:click={refreshCron}>刷新</button>
  </div>

  <div class="page-body">
    <div class="card">
      <div class="card-head"><h3>新建任务</h3></div>
      <form class="form-grid" on:submit|preventDefault={create}>
        <label>
          <span>名称</span>
          <input bind:value={form.name} disabled={disabled} placeholder="每日总结" />
        </label>
        <label>
          <span>调度表达式</span>
          <input
            bind:value={form.schedule}
            disabled={disabled || form.oneshot}
            placeholder="@daily 或 0 9 * * *"
          />
        </label>
        <label>
          <span>模式</span>
          <select bind:value={form.mode} disabled={disabled}>
            <option value="yolo">yolo</option>
            <option value="agent">agent</option>
          </select>
        </label>
        <label class="checkbox">
          <input type="checkbox" bind:checked={form.oneshot} disabled={disabled} />
          <span>一次性执行</span>
        </label>
        <label class="full">
          <span>Prompt</span>
          <textarea bind:value={form.prompt} disabled={disabled} rows="4" placeholder="要执行的指令…"></textarea>
        </label>
        <div class="form-actions">
          <button
            type="submit"
            class="primary"
            disabled={disabled || !form.name.trim() || !form.prompt.trim()}
          >
            创建
          </button>
        </div>
      </form>
    </div>

    <div class="card">
      <div class="card-head"><h3>任务列表</h3><span class="hint">共 {jobs.length} 项</span></div>
      <div class="cron-list">
        {#each jobs as job (job.id)}
          <div class="cron-row">
            <div class="cron-main">
              <strong>{job.name}</strong>
              <span class="mono">{shortID(job.id)}</span>
            </div>
            <div class="cron-meta">
              <span class="tag">{scheduleLabel(job)}</span>
              <span class="tag">{job.mode || 'yolo'}</span>
              <span>{job.run_count || 0} 次</span>
              {#if job.next_run}<span>下次 {formatDateTime(job.next_run)}</span>{/if}
              {#if job.last_status}<span class="tag">{job.last_status}</span>{/if}
            </div>
            <div class="cron-actions">
              {#if job.enabled}
                <button type="button" class="ghost" on:click={() => toggle(job.id, false)}>停用</button>
              {:else}
                <button type="button" class="ghost" on:click={() => toggle(job.id, true)}>启用</button>
              {/if}
              <button type="button" class="danger" on:click={() => remove(job.id)}>删除</button>
            </div>
            {#if job.last_error}
              <code class="cron-error">{job.last_error}</code>
            {/if}
          </div>
        {/each}
        {#if jobs.length === 0}
          <p class="empty">暂无任务</p>
        {/if}
      </div>
    </div>
  </div>
</section>
