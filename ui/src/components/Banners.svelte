<script>
  import { error, notice } from '../lib/stores.js';
  import { t } from '../lib/preferences.js';
</script>

{#if $error}
  <div class="banner banner-error" role="alert">
    <span>{$error}</span>
    <button type="button" class="link-btn" on:click={() => error.set('')}>{$t('common.close')}</button>
  </div>
{/if}
{#if $notice}
  <div class="banner banner-info" role="status">
    <span>{$notice}</span>
    <button type="button" class="link-btn" on:click={() => notice.set('')}>{$t('common.close')}</button>
  </div>
{/if}
