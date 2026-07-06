<script>
  import { route, navigate } from '../lib/router.js';
  import SettingsOverview from './settings/Overview.svelte';
  import SettingsServe from './settings/ServeConfig.svelte';
  import SettingsApp from './settings/AppSettings.svelte';
  import SettingsMemory from './settings/Memory.svelte';
  import SettingsFeatures from './settings/Features.svelte';
  import SettingsWorkDir from './settings/WorkDir.svelte';

  const tabs = [
    { key: '', label: '概览' },
    { key: 'serve', label: 'Serve 配置' },
    { key: 'workdir', label: '工作目录' },
    { key: 'app', label: '应用设置' },
    { key: 'memory', label: 'Memory' },
    { key: 'features', label: '功能开关' }
  ];

  $: activeTab = $route.sub || '';

  function open(sub) {
    navigate(sub ? `/settings/${sub}` : '/settings');
  }
</script>

<section class="page settings-page">
  <nav class="sub-tabs" aria-label="设置分组">
    {#each tabs as tab}
      <button
        type="button"
        class:active={activeTab === tab.key}
        on:click={() => open(tab.key)}
      >
        {tab.label}
      </button>
    {/each}
  </nav>

  <div class="sub-body">
    {#if activeTab === ''}
      <SettingsOverview />
    {:else if activeTab === 'serve'}
      <SettingsServe />
    {:else if activeTab === 'workdir'}
      <SettingsWorkDir />
    {:else if activeTab === 'app'}
      <SettingsApp />
    {:else if activeTab === 'memory'}
      <SettingsMemory />
    {:else if activeTab === 'features'}
      <SettingsFeatures />
    {:else}
      <p class="empty">未知的设置分组</p>
    {/if}
  </div>
</section>
