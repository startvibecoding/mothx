<script>
  import { route } from '../lib/router.js';
  import { refreshAll, connectLogs, disconnectLogs, logsConnected } from '../lib/stores.js';

  const titles = {
    chat: '新对话',
    sessions: '历史对话',
    cron: '定时任务',
    channels: '通道',
    logs: '日志',
    settings: '设置'
  };

  const subtitles = {
    chat: 'AI 生成可能有误，请核实',
    sessions: '管理所有会话历史',
    cron: '调度自动运行的任务',
    channels: '外部消息通道状态',
    logs: '实时日志流',
    settings: 'Serve / App / Memory 配置'
  };

  $: title = titles[$route.section] || $route.section;
  $: subtitle = subtitles[$route.section] || '';
</script>

<header class="topbar">
  <div class="tb-title">
    <h1>{title}</h1>
    {#if subtitle}<span>{subtitle}</span>{/if}
  </div>
  <div class="tb-actions">
    {#if $logsConnected}
      <button type="button" class="ghost" on:click={disconnectLogs}>关闭日志</button>
    {:else}
      <button type="button" class="ghost" on:click={connectLogs}>开启日志</button>
    {/if}
    <button type="button" class="ghost" on:click={refreshAll}>刷新</button>
  </div>
</header>
