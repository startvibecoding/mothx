<script>
  import { onDestroy, onMount } from 'svelte';
  import Sidebar from './components/Sidebar.svelte';
  import Topbar from './components/Topbar.svelte';
  import Banners from './components/Banners.svelte';
  import Chat from './views/Chat.svelte';
  import Sessions from './views/Sessions.svelte';
  import Cron from './views/Cron.svelte';
  import Channels from './views/Channels.svelte';
  import Logs from './views/Logs.svelte';
  import Settings from './views/Settings.svelte';
  import { route, navigate } from './lib/router.js';
  import { refreshAll, disconnectLogs } from './lib/stores.js';

  onMount(() => {
    refreshAll();
    if (!window.location.hash) navigate('/chat');
  });

  onDestroy(disconnectLogs);
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
      {:else if $route.section === 'cron'}
        <Cron />
      {:else if $route.section === 'channels'}
        <Channels />
      {:else if $route.section === 'logs'}
        <Logs />
      {:else if $route.section === 'settings'}
        <Settings />
      {:else}
        <section class="page">
          <p class="empty">未知的页面：{$route.path}</p>
        </section>
      {/if}
    </div>
  </main>
</div>
