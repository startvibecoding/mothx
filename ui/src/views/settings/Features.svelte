<script>
  import { features } from '../../lib/stores.js';

  const rows = [
    { key: 'api', label: 'OpenAI 兼容 API', desc: '开启后可通过 /v1/chat/completions 调用。' },
    { key: 'webUI', label: 'Web UI', desc: '当前界面本身，来自 Serve 静态资源目录。' },
    { key: 'websocket', label: 'WebSocket 日志', desc: '/ws/logs 实时日志推送。' },
    { key: 'cron', label: 'Cron 定时任务', desc: '基于本地任务清单的调度器。' },
    { key: 'memory', label: 'Memory', desc: 'memory.md 长期记忆。' },
    { key: 'multiAgent', label: '多 Agent', desc: '允许 spawn / send / destroy 子 Agent 工具。' }
  ];
</script>

<div class="card">
  <div class="card-head">
    <h3>功能开关</h3>
    <span class="hint">只读展示。修改请在「Serve 配置」或「应用设置」中调整。</span>
  </div>
  <table class="table">
    <thead>
      <tr>
        <th>功能</th>
        <th>状态</th>
        <th>说明</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as row}
        <tr>
          <td>{row.label}</td>
          <td>
            <span class="pill" class:on={$features[row.key]} class:off={!$features[row.key]}>
              {$features[row.key] ? '开启' : '关闭'}
            </span>
          </td>
          <td class="hint">{row.desc}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
