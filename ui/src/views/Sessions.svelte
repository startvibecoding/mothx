<script>
  import { sessions, currentSession, setError, setNotice, refreshSessions, clearBanners } from '../lib/stores.js';
  import { del } from '../lib/api.js';
  import { navigate } from '../lib/router.js';
  import { shortID } from '../lib/format.js';
  import { t } from '../lib/preferences.js';

  let filter = '';

  $: filtered = filterList($sessions, filter);

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
    <table class="table">
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
        {#each filtered as s (s.id)}
          <tr class:active={$currentSession === s.id}>
            <td>
              <button type="button" class="link-btn" on:click={() => open(s.id)}>
                {s.title || s.preview || shortID(s.id)}
              </button>
              <div class="sub">{s.id}</div>
              {#if s.preview && s.title}
                <div class="sub">{s.preview}</div>
              {/if}
            </td>
            <td class="wd">{s.workDir || '—'}</td>
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
  </div>
</section>
