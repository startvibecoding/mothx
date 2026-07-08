<script>
  import { logs, logsConnected, connectLogs, disconnectLogs, refreshAll } from '../../lib/stores.js';
  import { formatTime, formatLogMessage } from '../../lib/format.js';
  import { t } from '../../lib/preferences.js';

  let filter = '';
  $: filtered = filterLogs($logs, filter).slice(-500).reverse();

  function filterLogs(list, term) {
    const t = term.trim().toLowerCase();
    if (!t) return list;
    return list.filter((item) =>
      `${item.type || ''} ${formatLogMessage(item)}`.toLowerCase().includes(t)
    );
  }

  function clearLogs() {
    logs.set([]);
  }
</script>

<div class="page-toolbar embedded">
  <input class="filter" bind:value={filter} placeholder={$t('logs.filter')} />
  {#if $logsConnected}
    <button type="button" class="ghost" on:click={disconnectLogs}>{$t('topbar.closeLogs')}</button>
  {:else}
    <button type="button" class="ghost" on:click={connectLogs}>{$t('topbar.openLogs')}</button>
  {/if}
  <button type="button" class="ghost" on:click={refreshAll}>{$t('common.refresh')}</button>
  <button type="button" class="ghost" on:click={clearLogs}>{$t('common.clear')}</button>
</div>

<div class="card log-card">
  <div class="log-list">
    {#each filtered as item, idx (idx)}
      <div class="log-line">
        <span class="ts">{formatTime(item.timestamp)}</span>
        <strong class="type">{item.type}</strong>
        <code>{formatLogMessage(item)}</code>
      </div>
    {/each}
    {#if filtered.length === 0}
      <p class="empty">{$t('logs.empty')}</p>
    {/if}
  </div>
</div>
