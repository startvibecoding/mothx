<script>
  import { channels, refreshAll } from '../../lib/stores.js';
  import { t } from '../../lib/preferences.js';
</script>

<div class="page-toolbar embedded">
  <button type="button" class="ghost" on:click={refreshAll}>{$t('common.refresh')}</button>
</div>

<div class="card">
  <div class="card-head">
    <h3>{$t('channels.title')}</h3>
    <span class="hint">{$t('common.count', { count: $channels.length })}</span>
  </div>
  <table class="table">
    <thead>
      <tr>
        <th>{$t('channels.name')}</th>
        <th>{$t('channels.enabled')}</th>
        <th>{$t('channels.status')}</th>
      </tr>
    </thead>
    <tbody>
      {#each $channels as ch (ch.name)}
        <tr>
          <td>{ch.name}</td>
          <td>{ch.enabled ? $t('common.yes') : $t('common.no')}</td>
          <td>
            <span class="pill" class:on={ch.connected} class:off={!ch.connected}>
              {ch.enabled ? (ch.connected ? $t('common.connected') : $t('common.disconnected')) : $t('common.disabledState')}
            </span>
          </td>
        </tr>
      {/each}
      {#if $channels.length === 0}
        <tr><td colspan="3" class="empty-cell">{$t('channels.empty')}</td></tr>
      {/if}
    </tbody>
  </table>
</div>
