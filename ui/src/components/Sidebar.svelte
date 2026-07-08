<script>
  import { sessions, currentSession, features } from '../lib/stores.js';
  import { route, navigate } from '../lib/router.js';
  import { shortID } from '../lib/format.js';
  import { t } from '../lib/preferences.js';
  import PreferenceControls from './PreferenceControls.svelte';

  let searchTerm = '';

  const primaryNav = [
    { key: 'chat', path: '/chat', label: 'nav.newChat', icon: 'edit', accent: true },
    { key: 'sessions', path: '/sessions', label: 'nav.sessions', icon: 'clock' },
    { key: 'cron', path: '/cron', label: 'nav.cron', icon: 'timer', feature: 'cron' },
    { key: 'channels', path: '/channels', label: 'nav.channels', icon: 'plug' },
    { key: 'logs', path: '/logs', label: 'nav.logs', icon: 'stream' }
  ];

  const secondaryNav = [
    { key: 'settings', path: '/settings', label: 'nav.settings', icon: 'settings' }
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
      placeholder={$t('sidebar.search')}
      aria-label={$t('sidebar.search')}
    />
    <kbd>⌘K</kbd>
  </div>

  <button class="new-chat" on:click={openNewChat}>
    <span class="ico" aria-hidden="true">✎</span>
    <span class="label">{$t('nav.newChat')}</span>
    <kbd>⇧ ⌘K</kbd>
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
    <div class="side-history-list">
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

  <div class="side-footer">
    <PreferenceControls />
    <div class="side-identity">
      <div class="avatar" aria-hidden="true">Mx</div>
      <div class="who">
        <strong>{$t('app.name')}</strong>
        <span>{$t('app.local')}</span>
      </div>
    </div>
  </div>
</aside>
