<script>
  import { sessions, currentSession, setError, setNotice, refreshSessions, clearBanners } from '../lib/stores.js';
  import { del } from '../lib/api.js';
  import { navigate } from '../lib/router.js';
  import { shortID } from '../lib/format.js';

  let filter = '';

  $: filtered = filterList($sessions, filter);

  function filterList(list, term) {
    const t = term.trim().toLowerCase();
    if (!t) return list;
    return list.filter((s) =>
      `${s.id || ''} ${s.workDir || ''} ${s.title || ''} ${s.preview || ''}`.toLowerCase().includes(t)
    );
  }

  function open(id) {
    currentSession.set(id);
    navigate('/chat');
  }

  async function remove(id) {
    clearBanners();
    try {
      await del(`/api/sessions/${encodeURIComponent(id)}`);
      if ($currentSession === id) currentSession.set('');
      setNotice(`会话 ${shortID(id)} 已删除`);
      await refreshSessions();
    } catch (err) {
      setError(err);
    }
  }
</script>

<section class="page">
  <div class="page-toolbar">
    <input
      class="filter"
      bind:value={filter}
      placeholder="过滤会话（ID / 工作目录 / 标题）"
    />
    <button type="button" class="ghost" on:click={refreshSessions}>刷新</button>
  </div>

  <div class="page-body">
    <table class="table">
      <thead>
        <tr>
          <th>会话</th>
          <th>工作目录</th>
          <th>状态</th>
          <th class="num">消息数</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each filtered as s (s.id)}
          <tr class:active={$currentSession === s.id}>
            <td>
              <button type="button" class="link-btn" on:click={() => open(s.id)}>
                {s.title || s.preview || shortID(s.id)}
              </button>
              <div class="sub">{s.id}</div>
              {#if s.preview && s.title}
                <div class="sub">{s.preview}</div>
              {/if}
            </td>
            <td class="wd">{s.workDir || '—'}</td>
            <td>{s.active ? '运行中' : '历史'}</td>
            <td class="num">{s.messageCount || 0}</td>
            <td class="actions">
              <button type="button" class="ghost" on:click={() => open(s.id)}>打开</button>
              <button type="button" class="danger" on:click={() => remove(s.id)}>删除</button>
            </td>
          </tr>
        {/each}
        {#if filtered.length === 0}
          <tr>
            <td colspan="5" class="empty-cell">没有可显示的会话</td>
          </tr>
        {/if}
      </tbody>
    </table>
  </div>
</section>
