<script>
  import { logs, logsConnected, connectLogs, disconnectLogs } from '../lib/stores.js';
  import { formatTime, formatLogMessage } from '../lib/format.js';

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

<section class="page">
  <div class="page-toolbar">
    <input class="filter" bind:value={filter} placeholder="过滤日志内容" />
    {#if $logsConnected}
      <button type="button" class="ghost" on:click={disconnectLogs}>断开</button>
    {:else}
      <button type="button" class="ghost" on:click={connectLogs}>连接</button>
    {/if}
    <button type="button" class="ghost" on:click={clearLogs}>清空</button>
  </div>

  <div class="page-body">
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
          <p class="empty">暂无日志</p>
        {/if}
      </div>
    </div>
  </div>
</section>
