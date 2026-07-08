<script>
  import { language, themeMode, t, setLanguage, setThemeMode } from '../lib/preferences.js';

  const languages = [
    { value: 'zh', label: 'prefs.zh' },
    { value: 'en', label: 'prefs.en' }
  ];

  const themes = [
    { value: 'light', label: 'prefs.light' },
    { value: 'dark', label: 'prefs.dark' },
    { value: 'auto', label: 'prefs.auto' }
  ];
</script>

<div class="pref-controls" aria-label={$t('prefs.preferences')}>
  <div class="pref-row">
    <span>{$t('prefs.language')}</span>
    <div class="segmented">
      {#each languages as item}
        <button
          type="button"
          class:active={$language === item.value}
          aria-pressed={$language === item.value}
          on:click={() => setLanguage(item.value)}
        >
          {$t(item.label)}
        </button>
      {/each}
    </div>
  </div>

  <div class="pref-row">
    <span>{$t('prefs.theme')}</span>
    <div class="segmented theme">
      {#each themes as item}
        <button
          type="button"
          class:active={$themeMode === item.value}
          aria-pressed={$themeMode === item.value}
          on:click={() => setThemeMode(item.value)}
        >
          {$t(item.label)}
        </button>
      {/each}
    </div>
  </div>
</div>
