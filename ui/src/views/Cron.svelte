<script>
  import { cronInfo, refreshCron, setError, setNotice, clearBanners } from '../lib/stores.js';
  import { postJSON, patchJSON, del } from '../lib/api.js';
  import { shortID, scheduleLabel, formatDateTime } from '../lib/format.js';
  import { t } from '../lib/preferences.js';

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
      setNotice($t('cron.created'));
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
      setNotice($t('cron.deleted', { id: shortID(id) }));
      await refreshCron();
    } catch (err) {
      setError(err);
    }
  }
</script>

<section class="page">
  <div class="page-toolbar">
    <div class="status-tag" class:muted={disabled}>
      {disabled ? $t('common.disabledState') : info?.running ? $t('common.running') : $t('common.idle')}
      {#if info?.path}<span class="hint">{info.path}</span>{/if}
    </div>
    <button type="button" class="ghost" on:click={refreshCron}>{$t('common.refresh')}</button>
  </div>

  <div class="page-body">
    <div class="card">
      <div class="card-head"><h3>{$t('cron.newTask')}</h3></div>
      <form class="form-grid" on:submit|preventDefault={create}>
        <label>
          <span>{$t('cron.name')}</span>
          <input bind:value={form.name} disabled={disabled} placeholder={$t('cron.namePlaceholder')} />
        </label>
        <label>
          <span>{$t('cron.schedule')}</span>
          <input
            bind:value={form.schedule}
            disabled={disabled || form.oneshot}
            placeholder={$t('cron.schedulePlaceholder')}
          />
        </label>
        <label>
          <span>{$t('cron.mode')}</span>
          <select bind:value={form.mode} disabled={disabled}>
            <option value="yolo">yolo</option>
            <option value="agent">agent</option>
          </select>
        </label>
        <label class="checkbox">
          <input type="checkbox" bind:checked={form.oneshot} disabled={disabled} />
          <span>{$t('cron.oneshot')}</span>
        </label>
        <label class="full">
          <span>Prompt</span>
          <textarea bind:value={form.prompt} disabled={disabled} rows="4" placeholder={$t('cron.promptPlaceholder')}></textarea>
        </label>
        <div class="form-actions">
          <button
            type="submit"
            class="primary"
            disabled={disabled || !form.name.trim() || !form.prompt.trim()}
          >
            {$t('common.create')}
          </button>
        </div>
      </form>
    </div>

    <div class="card">
      <div class="card-head"><h3>{$t('cron.list')}</h3><span class="hint">{$t('common.items', { count: jobs.length })}</span></div>
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
              <span>{$t('common.times', { count: job.run_count || 0 })}</span>
              {#if job.next_run}<span>{$t('common.next', { time: formatDateTime(job.next_run) })}</span>{/if}
              {#if job.last_status}<span class="tag">{job.last_status}</span>{/if}
            </div>
            <div class="cron-actions">
              {#if job.enabled}
                <button type="button" class="ghost" on:click={() => toggle(job.id, false)}>{$t('common.disable')}</button>
              {:else}
                <button type="button" class="ghost" on:click={() => toggle(job.id, true)}>{$t('common.enable')}</button>
              {/if}
              <button type="button" class="danger" on:click={() => remove(job.id)}>{$t('common.delete')}</button>
            </div>
            {#if job.last_error}
              <code class="cron-error">{job.last_error}</code>
            {/if}
          </div>
        {/each}
        {#if jobs.length === 0}
          <p class="empty">{$t('cron.empty')}</p>
        {/if}
      </div>
    </div>
  </div>
</section>
