<script>
  import { channels, refreshAll } from '../lib/stores.js';
</script>

<section class="page">
  <div class="page-toolbar">
    <button type="button" class="ghost" on:click={refreshAll}>刷新</button>
  </div>
  <div class="page-body">
    <div class="card">
      <div class="card-head">
        <h3>消息通道</h3>
        <span class="hint">{$channels.length} 个</span>
      </div>
      <table class="table">
        <thead>
          <tr>
            <th>名称</th>
            <th>启用</th>
            <th>状态</th>
          </tr>
        </thead>
        <tbody>
          {#each $channels as ch (ch.name)}
            <tr>
              <td>{ch.name}</td>
              <td>{ch.enabled ? '是' : '否'}</td>
              <td>
                <span class="pill" class:on={ch.connected} class:off={!ch.connected}>
                  {ch.enabled ? (ch.connected ? '已连接' : '未连接') : '已禁用'}
                </span>
              </td>
            </tr>
          {/each}
          {#if $channels.length === 0}
            <tr><td colspan="3" class="empty-cell">无通道</td></tr>
          {/if}
        </tbody>
      </table>
    </div>
  </div>
</section>
