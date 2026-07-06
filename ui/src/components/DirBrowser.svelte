<script>
  import { createEventDispatcher } from 'svelte';
  import { request } from '../lib/api.js';

  export let open = false;

  const dispatch = createEventDispatcher();

  let currentPath = '';
  let parentPath = '';
  let entries = [];
  let loading = false;
  let error = '';

  $: if (open) load();

  async function load(dir) {
    loading = true;
    error = '';
    try {
      const params = currentPath ? `?path=${encodeURIComponent(currentPath)}` : '';
      const data = await request(`/api/browse${params}`);
      currentPath = data.path;
      parentPath = data.parent;
      entries = data.entries || [];
    } catch (err) {
      error = err.message;
      entries = [];
    } finally {
      loading = false;
    }
  }

  function enter(dirPath) {
    currentPath = dirPath;
    load();
  }

  function goUp() {
    if (parentPath && parentPath !== currentPath) {
      currentPath = parentPath;
      load();
    }
  }

  function select() {
    dispatch('select', { path: currentPath });
    open = false;
  }

  function close() {
    open = false;
    dispatch('close');
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') close();
  }
</script>

{#if open}
  <div class="dir-overlay" on:keydown={handleKeydown} role="dialog" aria-modal="true" aria-label="选择工作目录" tabindex="-1">
    <div class="dir-modal">
      <div class="dir-header">
        <h3>选择工作目录</h3>
        <button type="button" class="ghost" on:click={close}>✕</button>
      </div>

      <div class="dir-nav">
        <button type="button" class="ghost" on:click={goUp} disabled={parentPath === currentPath || !parentPath}>↑ 上级</button>
        <div class="dir-path">
          <span class="ico">📁</span>
          <span class="path-text">{currentPath}</span>
        </div>
      </div>

      <div class="dir-list">
        {#if loading}
          <p class="empty">加载中…</p>
        {:else if error}
          <p class="empty dir-error">{error}</p>
        {:else if entries.length === 0}
          <p class="empty">无子目录</p>
        {:else}
          {#each entries as entry (entry.path)}
            <button
              type="button"
              class="dir-entry"
              on:click={() => enter(entry.path)}
            >
              <span class="ico">📂</span>
              <span class="name">{entry.name}</span>
            </button>
          {/each}
        {/if}
      </div>

      <div class="dir-footer">
        <button type="button" class="ghost" on:click={close}>取消</button>
        <button type="button" class="primary" on:click={select}>选择此目录</button>
      </div>
    </div>
  </div>
{/if}
