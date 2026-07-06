<script>
  import { sessions, currentSession, features } from '../lib/stores.js';
  import { route, navigate } from '../lib/router.js';
  import { shortID } from '../lib/format.js';

  let searchTerm = '';

  const primaryNav = [
    { key: 'chat', path: '/chat', label: '新对话', icon: 'edit', accent: true },
    { key: 'sessions', path: '/sessions', label: '历史对话', icon: 'clock' },
    { key: 'cron', path: '/cron', label: '定时任务', icon: 'timer', feature: 'cron' },
    { key: 'channels', path: '/channels', label: '通道', icon: 'plug' },
    { key: 'logs', path: '/logs', label: '日志', icon: 'stream' }
  ];

  const secondaryNav = [
    { key: 'settings', path: '/settings', label: '设置', icon: 'settings' }
  ];

  $: filteredSessions = filterSessions($sessions, searchTerm);
  $: recentSessions = filteredSessions.slice(0, 12);

  function filterSessions(list, term) {
    const t = term.trim().toLowerCase();
    if (!t) return list;
    return list.filter((s) => {
      const hay = `${s.id || ''} ${s.workDir || ''} ${(s.title || '')}`.toLowerCase();
      return hay.includes(t);
    });
  }

  function openSession(id) {
    currentSession.set(id);
    navigate('/chat');
  }

  function openNewChat() {
    currentSession.set('');
    navigate('/chat');
  }

  function isActive(item) {
    return $route.section === item.key;
  }

  function isFeatureEnabled(item) {
    if (!item.feature) return true;
    return $features[item.feature] !== false;
  }
</script>

<aside class="sidebar">
  <div class="side-search">
    <span class="ico" aria-hidden="true">🔍</span>
    <input
      bind:value={searchTerm}
      placeholder="搜索会话..."
      aria-label="搜索会话"
    />
    <kbd>⌘K</kbd>
  </div>

  <button class="new-chat" on:click={openNewChat}>
    <span class="ico" aria-hidden="true">✎</span>
    <span class="label">新对话</span>
    <kbd>⇧ ⌘K</kbd>
  </button>

  <nav class="side-nav" aria-label="主导航">
    {#each primaryNav.slice(1) as item}
      <button
        type="button"
        class="nav-item"
        class:active={isActive(item)}
        disabled={!isFeatureEnabled(item)}
        on:click={() => navigate(item.path)}
      >
        <span class="ico ico-{item.icon}" aria-hidden="true"></span>
        <span class="label">{item.label}</span>
      </button>
    {/each}

    <div class="nav-divider"></div>

    {#each secondaryNav as item}
      <button
        type="button"
        class="nav-item"
        class:active={isActive(item)}
        on:click={() => navigate(item.path)}
      >
        <span class="ico ico-{item.icon}" aria-hidden="true"></span>
        <span class="label">{item.label}</span>
      </button>
    {/each}
  </nav>

  <section class="side-history" aria-label="历史对话">
    <div class="side-history-head">
      <span>历史对话</span>
      <button
        type="button"
        class="link-btn"
        on:click={() => navigate('/sessions')}
      >
        全部
      </button>
    </div>
    <div class="side-history-list">
      <button
        type="button"
        class="history-item"
        class:active={$currentSession === '' && $route.section === 'chat'}
        on:click={() => openSession('')}
      >
        <span class="dot" aria-hidden="true"></span>
        <span class="text">默认会话</span>
      </button>
      {#each recentSessions as session (session.id)}
        <button
          type="button"
          class="history-item"
          class:active={$currentSession === session.id && $route.section === 'chat'}
          title={session.workDir || session.id}
          on:click={() => openSession(session.id)}
        >
          <span class="dot" aria-hidden="true"></span>
          <span class="text">
            <span class="name">{session.title || shortID(session.id)}</span>
            {#if session.workDir}<span class="dir">{session.workDir}</span>{/if}
          </span>
        </button>
      {/each}
      {#if recentSessions.length === 0 && !searchTerm}
        <p class="empty">暂无历史</p>
      {:else if recentSessions.length === 0 && searchTerm}
        <p class="empty">未匹配到会话</p>
      {/if}
    </div>
  </section>

  <div class="side-footer">
    <div class="avatar" aria-hidden="true">Mx</div>
    <div class="who">
      <strong>MothX</strong>
      <span>本地运行</span>
    </div>
  </div>
</aside>
