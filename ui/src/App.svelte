<script>
  import { onDestroy, onMount } from 'svelte';
  import { get } from 'svelte/store';
  import Sidebar from './components/Sidebar.svelte';
  import Topbar from './components/Topbar.svelte';
  import Banners from './components/Banners.svelte';
  import Chat from './views/Chat.svelte';
  import Sessions from './views/Sessions.svelte';
  import Stats from './views/Stats.svelte';
  import Cron from './views/Cron.svelte';
  import Settings from './views/Settings.svelte';
  import { route, navigate } from './lib/router.js';
  import { refreshAll, disconnectLogs, currentSession } from './lib/stores.js';
  import { t } from './lib/preferences.js';

  let stopRouteSync = null;
  let stopSessionSync = null;

  onMount(() => {
    if (!window.location.hash) navigate('/chat');
    stopRouteSync = route.subscribe(syncSessionFromRoute);
    stopSessionSync = currentSession.subscribe(syncRouteFromSession);
    refreshAll();
  });

  onDestroy(() => {
    stopRouteSync?.();
    stopSessionSync?.();
    disconnectLogs();
  });

  function syncSessionFromRoute(nextRoute) {
    if (nextRoute.section !== 'chat') return;
    const routeSession = nextRoute.query?.session || '';
    if (get(currentSession) !== routeSession) currentSession.set(routeSession);
  }

  function syncRouteFromSession(sessionID) {
    const currentRoute = get(route);
    if (currentRoute.section !== 'chat') return;
    const routeSession = currentRoute.query?.session || '';
    const nextSession = sessionID || '';
    if (routeSession === nextSession) return;
    navigate(nextSession ? `/chat?session=${encodeURIComponent(nextSession)}` : '/chat');
  }
</script>

<div class="app-shell">
  <Sidebar />
  <main class="workbench">
    <Topbar />
    <Banners />
    <div class="view-container">
      {#if $route.section === 'chat'}
        <Chat />
      {:else if $route.section === 'sessions'}
        <Sessions />
      {:else if $route.section === 'stats'}
        <Stats />
      {:else if $route.section === 'cron'}
        <Cron />
      {:else if $route.section === 'settings'}
        <Settings />
      {:else}
        <section class="page">
          <p class="empty">{$t('app.unknownPage')}: {$route.path}</p>
        </section>
      {/if}
    </div>
  </main>
</div>
