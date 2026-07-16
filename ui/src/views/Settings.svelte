<script>
  import { route, navigate } from '../lib/router.js';
  import SettingsOverview from './settings/Overview.svelte';
  import SettingsServe from './settings/ServeConfig.svelte';
  import SettingsApp from './settings/AppSettings.svelte';
  import SettingsProviders from './settings/ProviderSettings.svelte';
  import SettingsMemory from './settings/Memory.svelte';
  import SettingsFeatures from './settings/Features.svelte';
  import SettingsWorkDir from './settings/WorkDir.svelte';
  import SettingsChannels from './settings/Channels.svelte';
  import SettingsLogs from './settings/Logs.svelte';
  import SettingsSkillHub from './settings/SkillHub.svelte';
  import { t } from '../lib/preferences.js';

  const tabs = [
    { key: '', label: 'settings.tabs.overview' },
    { key: 'serve', label: 'settings.tabs.serve' },
    { key: 'workdir', label: 'settings.tabs.workdir' },
    { key: 'providers', label: 'settings.tabs.providers' },
    { key: 'app', label: 'settings.tabs.app' },
    { key: 'memory', label: 'settings.tabs.memory' },
    { key: 'features', label: 'settings.tabs.features' },
    { key: 'channels', label: 'settings.tabs.channels' },
    { key: 'logs', label: 'settings.tabs.logs' },
    { key: 'skillhub', label: 'SkillHub' }
  ];

  $: activeTab = $route.sub || '';

  function open(sub) {
    navigate(sub ? `/settings/${sub}` : '/settings');
  }
</script>

<section class="page settings-page">
  <nav class="sub-tabs" aria-label={$t('nav.settings')}>
    {#each tabs as tab}
      <button
        type="button"
        class:active={activeTab === tab.key}
        on:click={() => open(tab.key)}
      >
        {$t(tab.label)}
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
    {:else if activeTab === 'providers'}
      <SettingsProviders />
    {:else if activeTab === 'app'}
      <SettingsApp />
    {:else if activeTab === 'memory'}
      <SettingsMemory />
    {:else if activeTab === 'features'}
      <SettingsFeatures />
    {:else if activeTab === 'channels'}
      <SettingsChannels />
    {:else if activeTab === 'logs'}
      <SettingsLogs />
    {:else if activeTab === 'skillhub'}
      <SettingsSkillHub />
    {:else}
      <p class="empty">{$t('settings.unknown')}</p>
    {/if}
  </div>
</section>
