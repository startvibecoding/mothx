<script>
  import { t } from '../../lib/preferences.js';

  export let title = '';
  export let list = [];
  export let onAdd = () => {};
  export let onRemove = () => {};
</script>

<div class="list-editor full">
  <div class="list-head">
    <span>{title}</span>
    <button type="button" class="ghost sm" on:click={onAdd}>{$t('common.add')}</button>
  </div>
  {#each list as item, i (i)}
    <div class="inline-row">
      <input bind:value={list[i]} />
      <button type="button" class="ghost sm" on:click={() => onRemove(i)}>{$t('common.remove')}</button>
    </div>
  {/each}
</div>
