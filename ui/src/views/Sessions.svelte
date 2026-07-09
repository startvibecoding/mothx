<script>
  import { sessions, currentSession, setError, setNotice, refreshSessions, clearBanners } from '../lib/stores.js';
  import { del } from '../lib/api.js';
  import { navigate } from '../lib/router.js';
  import { shortID } from '../lib/format.js';
  import { t } from '../lib/preferences.js';

  const pageSize = 25;

  let filter = '';
  let page = 1;
  let previousFilter = '';

  $: filtered = filterList($sessions, filter);
  $: totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  $: if (filter !== previousFilter) {
    page = 1;
    previousFilter = filter;
  }
  $: if (page > totalPages) page = totalPages;
  $: if (page < 1) page = 1;
  $: pageStart = filtered.length === 0 ? 0 : (page - 1) * pageSize + 1;
  $: pageEnd = Math.min(filtered.length, page * pageSize);
  $: pageItems = filtered.slice((page - 1) * pageSize, page * pageSize);
  $: pageNumbers = buildPageNumbers(page, totalPages);

  function filterList(list, term) {
    const t = term.trim().toLowerCase();
    if (!t) return list;
    return list.filter((s) =>
      `${s.id || ''} ${s.workDir || ''} ${s.title || ''} ${s.preview || ''}`.toLowerCase().includes(t)
    );
  }

  function open(id) {
    currentSession.set(id);
    navigate(id ? `/chat?session=${encodeURIComponent(id)}` : '/chat');
  }

  function buildPageNumbers(current, total) {
    if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
    const pages = [1];
    const start = Math.max(2, current - 1);
    const end = Math.min(total - 1, current + 1);
    if (start > 2) pages.push('gap-start');
    for (let n = start; n <= end; n += 1) pages.push(n);
    if (end < total - 1) pages.push('gap-end');
    pages.push(total);
    return pages;
  }

  function goToPage(next) {
    page = Math.min(totalPages, Math.max(1, Number(next) || 1));
  }

  async function remove(id) {
    clearBanners();
    try {
      await del(`/api/sessions/${encodeURIComponent(id)}`);
      if ($currentSession === id) currentSession.set('');
      setNotice($t('sessions.deleted', { id: shortID(id) }));
      await refreshSessions();
    } catch (err) {
      setError(err);
    }
  }
</script>

<section class="page">
  <div class="page-toolbar">
    <input
      class="filter"
      bind:value={filter}
      placeholder={$t('sessions.filter')}
    />
    <button type="button" class="ghost" on:click={refreshSessions}>{$t('common.refresh')}</button>
  </div>

  <div class="page-body">
    <table class="table sessions-table">
      <colgroup>
        <col class="session-col" />
        <col class="workdir-col" />
        <col class="status-col" />
        <col class="count-col" />
        <col class="actions-col" />
      </colgroup>
      <thead>
        <tr>
          <th>{$t('sessions.session')}</th>
          <th>{$t('sessions.workDir')}</th>
          <th>{$t('sessions.status')}</th>
          <th class="num">{$t('sessions.messageCount')}</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each pageItems as s (s.id)}
          <tr class:active={$currentSession === s.id}>
            <td class="session-cell">
              <button
                type="button"
                class="link-btn session-title"
                title={s.title || s.preview || shortID(s.id)}
                on:click={() => open(s.id)}
              >
                {s.title || s.preview || shortID(s.id)}
              </button>
              <div class="sub session-id-line" title={s.id}>{s.id}</div>
              {#if s.preview && s.title}
                <div class="sub session-preview" title={s.preview}>{s.preview}</div>
              {/if}
            </td>
            <td class="wd" title={s.workDir || ''}>{s.workDir || '—'}</td>
            <td>{s.active ? $t('sessions.active') : $t('sessions.history')}</td>
            <td class="num">{s.messageCount || 0}</td>
            <td class="actions">
              <button type="button" class="ghost" on:click={() => open(s.id)}>{$t('common.open')}</button>
              <button type="button" class="danger" on:click={() => remove(s.id)}>{$t('common.delete')}</button>
            </td>
          </tr>
        {/each}
        {#if filtered.length === 0}
          <tr>
            <td colspan="5" class="empty-cell">{$t('sessions.empty')}</td>
          </tr>
        {/if}
      </tbody>
    </table>
    {#if filtered.length > pageSize}
      <div class="stats-pagination sessions-pagination">
        <button type="button" class="page-btn" disabled={page <= 1} on:click={() => goToPage(1)}>{$t('common.first')}</button>
        <button type="button" class="page-btn" disabled={page <= 1} on:click={() => goToPage(page - 1)}>{$t('common.previous')}</button>
        {#each pageNumbers as item}
          {#if typeof item === 'number'}
            <button
              type="button"
              class="page-btn"
              class:active={item === page}
              aria-current={item === page ? 'page' : undefined}
              on:click={() => goToPage(item)}
            >
              {item}
            </button>
          {:else}
            <span class="page-gap" aria-hidden="true">...</span>
          {/if}
        {/each}
        <button type="button" class="page-btn" disabled={page >= totalPages} on:click={() => goToPage(page + 1)}>{$t('common.nextPage')}</button>
        <button type="button" class="page-btn" disabled={page >= totalPages} on:click={() => goToPage(totalPages)}>{$t('common.last')}</button>
        <span class="page-info">{$t('sessions.pageRange', { start: pageStart, end: pageEnd, total: filtered.length })}</span>
      </div>
    {/if}
  </div>
</section>
