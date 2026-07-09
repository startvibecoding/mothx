<script>
  import { onDestroy, onMount, tick } from 'svelte';
  import { sessions, currentSession, features, statsSummary, refreshStatsSummary } from '../lib/stores.js';
  import { route, navigate } from '../lib/router.js';
  import { shortID } from '../lib/format.js';
  import { t } from '../lib/preferences.js';
  import PreferenceControls from './PreferenceControls.svelte';

  let searchTerm = '';
  let searchInput;
  let isMac = false;
  let searchShortcut = 'Ctrl K';
  let newChatShortcut = 'Ctrl⇧K';
  let removeShortcutListener = null;
  let historyScrollbarVisible = false;
  let hideHistoryScrollbarTimer = null;

  const primaryNav = [
    { key: 'chat', path: '/chat', label: 'nav.newChat', icon: 'edit', accent: true },
    { key: 'sessions', path: '/sessions', label: 'nav.sessions', icon: 'clock' },
    { key: 'stats', path: '/stats', label: 'nav.stats', icon: 'chart' },
    { key: 'cron', path: '/cron', label: 'nav.cron', icon: 'timer' }
  ];

  const secondaryNav = [
    { key: 'settings', path: '/settings', label: 'nav.settings', icon: 'settings' }
  ];

  $: filteredSessions = filterSessions($sessions, searchTerm);
  $: recentSessions = filteredSessions.slice(0, 12);
  $: summaryStats = $statsSummary || {};
  $: searchAriaShortcut = isMac ? 'Meta+K' : 'Control+K';
  $: newChatAriaShortcut = isMac ? 'Shift+Meta+K' : 'Shift+Control+K';

  onMount(() => {
    isMac = /Mac|iPhone|iPad|iPod/.test(navigator.platform || '');
    searchShortcut = isMac ? '⌘K' : 'Ctrl K';
    newChatShortcut = isMac ? '⇧⌘K' : 'Ctrl⇧K';
    const onKeydown = (event) => handleGlobalShortcut(event);
    window.addEventListener('keydown', onKeydown);
    removeShortcutListener = () => window.removeEventListener('keydown', onKeydown);
    refreshStatsSummary();
  });

  onDestroy(() => {
    removeShortcutListener?.();
    if (hideHistoryScrollbarTimer) clearTimeout(hideHistoryScrollbarTimer);
  });

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
    navigate(id ? `/chat?session=${encodeURIComponent(id)}` : '/chat');
  }

  function openNewChat() {
    currentSession.set('');
    navigate('/chat');
  }

  async function focusSearch() {
    await tick();
    searchInput?.focus();
    searchInput?.select();
  }

  function handleGlobalShortcut(event) {
    const key = (event.key || '').toLowerCase();
    const mod = isMac ? event.metaKey : event.ctrlKey;
    if (!mod || key !== 'k' || event.altKey) return;
    event.preventDefault();
    event.stopPropagation();
    if (event.shiftKey) {
      openNewChat();
      return;
    }
    focusSearch();
  }

  function handleSearchKeydown(event) {
    if (event.key === 'Escape' && searchTerm) {
      event.preventDefault();
      searchTerm = '';
    }
  }

  function showHistoryScrollbar() {
    historyScrollbarVisible = true;
    if (hideHistoryScrollbarTimer) clearTimeout(hideHistoryScrollbarTimer);
    hideHistoryScrollbarTimer = setTimeout(() => {
      historyScrollbarVisible = false;
      hideHistoryScrollbarTimer = null;
    }, 900);
  }

  function isActive(item) {
    return $route.section === item.key;
  }

  function isFeatureEnabled(item) {
    if (!item.feature) return true;
    return $features[item.feature] !== false;
  }

  function formatStat(value) {
    const n = Number(value || 0);
    if (!Number.isFinite(n)) return '0';
    if (Math.abs(n) < 10000) return new Intl.NumberFormat().format(n);
    return new Intl.NumberFormat(undefined, {
      notation: 'compact',
      maximumFractionDigits: 1
    }).format(n);
  }
</script>

<aside class="sidebar">
  <div class="side-search">
    <span class="ico" aria-hidden="true">🔍</span>
    <input
      bind:this={searchInput}
      bind:value={searchTerm}
      placeholder={$t('sidebar.search')}
      aria-label={$t('sidebar.search')}
      aria-keyshortcuts={searchAriaShortcut}
      on:keydown={handleSearchKeydown}
    />
    <kbd>{searchShortcut}</kbd>
  </div>

  <button
    type="button"
    class="new-chat"
    on:click={openNewChat}
    aria-keyshortcuts={newChatAriaShortcut}
    title={`${$t('nav.newChat')} (${newChatShortcut})`}
  >
    <span class="ico" aria-hidden="true">✎</span>
    <span class="label">{$t('nav.newChat')}</span>
    <kbd>{newChatShortcut}</kbd>
  </button>

  <nav class="side-nav" aria-label={$t('nav.sessions')}>
    {#each primaryNav.slice(1) as item}
      <button
        type="button"
        class="nav-item"
        class:active={isActive(item)}
        disabled={!isFeatureEnabled(item)}
        on:click={() => navigate(item.path)}
      >
        <span class="ico ico-{item.icon}" aria-hidden="true"></span>
        <span class="label">{$t(item.label)}</span>
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
        <span class="label">{$t(item.label)}</span>
      </button>
    {/each}
  </nav>

  <section class="side-history" aria-label={$t('sidebar.history')}>
    <div class="side-history-head">
      <span>{$t('sidebar.history')}</span>
      <button
        type="button"
        class="link-btn"
        on:click={() => navigate('/sessions')}
      >
        {$t('sidebar.all')}
      </button>
    </div>
    <div
      class="side-history-list"
      class:scrolling={historyScrollbarVisible}
      on:wheel={showHistoryScrollbar}
      on:scroll={showHistoryScrollbar}
    >
      <button
        type="button"
        class="history-item"
        class:active={$currentSession === '' && $route.section === 'chat'}
        on:click={() => openSession('')}
      >
        <span class="dot" aria-hidden="true"></span>
        <span class="text">{$t('sidebar.defaultSession')}</span>
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
        <p class="empty">{$t('sidebar.noHistory')}</p>
      {:else if recentSessions.length === 0 && searchTerm}
        <p class="empty">{$t('sidebar.noMatches')}</p>
      {/if}
    </div>
  </section>

  <button type="button" class="side-stats" aria-label={$t('sidebar.stats')} on:click={() => navigate('/stats')}>
    <span>{$t('sidebar.stats')}</span>
    <div>
      <div>
        <strong>{formatStat(summaryStats.totalRequests)}</strong>
        <span>{$t('sidebar.stats.requests')}</span>
      </div>
      <div>
        <strong>{formatStat(summaryStats.totalTokens)}</strong>
        <span>{$t('sidebar.stats.tokens')}</span>
      </div>
    </div>
  </button>

  <div class="side-footer">
    <PreferenceControls />
  </div>
</aside>
