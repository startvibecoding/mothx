<script>
  import { navigate } from '../../lib/router.js';
  import { status, health, memoryInfo, cronInfo, features } from '../../lib/stores.js';

  const groups = [
    { key: 'serve', title: 'Serve 配置', desc: 'HTTP 监听、通道、WebUI 静态目录等运行时配置。' },
    { key: 'app', title: '应用设置', desc: '模型、Provider、审批策略、上下文压缩等应用层参数。' },
    { key: 'memory', title: 'Memory', desc: '长期记忆 memory.md 的启用与内容编辑。' },
    { key: 'features', title: '功能开关', desc: '开关 API、WebSocket、Cron、Memory 等模块。' }
  ];
</script>

<div class="grid-cards">
  {#each groups as g}
    <button type="button" class="card link" on:click={() => navigate(`/settings/${g.key}`)}>
      <div class="card-head"><h3>{g.title}</h3></div>
      <p class="card-desc">{g.desc}</p>
    </button>
  {/each}
</div>

<div class="card">
  <div class="card-head"><h3>运行时信息</h3></div>
  <dl class="kv">
    <dt>版本</dt><dd>{$health?.version || 'dev'}</dd>
    <dt>监听</dt><dd>{$status?.listen || '—'}</dd>
    <dt>会话总数</dt><dd>{$status?.sessions ?? $health?.sessions ?? 0}</dd>
    <dt>Cron</dt><dd>{$cronInfo?.enabled === false ? '禁用' : ($cronInfo?.running ? '运行中' : '空闲')}</dd>
    <dt>Memory</dt><dd>{$memoryInfo?.enabled === false ? '禁用' : ($memoryInfo?.path || '未初始化')}</dd>
    <dt>OpenAI API</dt><dd>{$features.api ? '开启' : '关闭'}</dd>
  </dl>
</div>
