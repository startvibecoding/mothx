<script>
  import { onMount } from 'svelte';
  import { currentSession, sessions, setError, setNotice, clearBanners } from '../lib/stores.js';
  import { request, postJSON } from '../lib/api.js';
  import { t } from '../lib/preferences.js';

  const pageSize = 20;
  let market = 'skillhub.cn';
  let view = 'official';
  let query = '';
  let category = '';
  let sort = 'downloads';
  let scope = 'project';
  let selectedSession = '';
  let targetDir = '';
  let targets = [];
  let targetLoading = false;
  let page = 1;
  let cursor = '';
  let cursorHistory = [];
  let nextCursor = '';
  let total = 0;
  let items = [];
  let selected = null;
  let detail = null;
  let categories = [];
  let activeSkills = [];
  let loading = false;
  let detailLoading = false;
  let actionLoading = false;
  let selectedBatch = new Set();
  let resultsRequest = 0;
  let detailRequest = 0;

  $: sessionID = selectedSession || '';
  $: selectedSessionInfo = $sessions.find((item) => item.id === sessionID) || null;
  $: canPrevious = market === 'clawhub.ai' ? cursorHistory.length > 0 : page > 1;
  $: canNext = market === 'clawhub.ai' ? Boolean(nextCursor) : page * pageSize < total;

  onMount(async () => {
    if (!selectedSession && $currentSession) selectedSession = $currentSession;
    await Promise.all([loadCategories(), loadInstalled(), loadTargets()]);
    await loadResults();
  });

  function sessionParams(extra = {}) {
    const params = new URLSearchParams(extra);
    if (sessionID) params.set('sessionId', sessionID);
    return params;
  }

  async function loadInstalled() {
    try {
      const data = await request(`/api/skillhub/installed?${sessionParams()}`);
      activeSkills = data?.session?.activeSkills || [];
    } catch (err) {
      setError(err);
    }
  }

  async function loadCategories() {
    try {
      const data = await request('/api/skillhub/categories?market=skillhub.cn');
      categories = data?.categories || [];
    } catch {
      categories = [];
    }
  }

  async function loadTargets() {
    targets = [];
    targetDir = '';
    if (!sessionID) return;
    targetLoading = true;
    try {
      const data = await request(`/api/skillhub/targets?${sessionParams()}`);
      targets = data?.targets || [];
    } catch (err) {
      setError(err);
    } finally {
      targetLoading = false;
    }
  }

  function selectSession(id) {
    selectedSession = id;
    loadTargets();
    loadInstalled();
  }

  async function loadResults() {
	const requestID = ++resultsRequest;
	detailRequest += 1;
	detail = null;
    loading = true;
    clearBanners();
    try {
      const params = sessionParams({ market, limit: String(pageSize) });
      let endpoint = '/api/skillhub/search';
      if (view === 'official') {
        endpoint = '/api/skillhub/official';
        params.set('page', String(page));
        if (query.trim()) params.set('q', query.trim());
      } else {
        if (view === 'search' && query.trim()) params.set('q', query.trim());
        if (market === 'clawhub.ai') {
          if (cursor) params.set('cursor', cursor);
        } else {
          params.set('page', String(page));
          params.set('sort', sort);
          params.set('order', 'desc');
          if (category) params.set('category', category);
        }
      }
      const data = await request(`${endpoint}?${params}`);
	  if (requestID !== resultsRequest) return;
      items = data?.items || [];
      total = Number(data?.total || items.length);
      nextCursor = data?.nextCursor || '';
      selected = items[0] || null;
      detail = null;
      if (selected) await loadDetail(selected);
    } catch (err) {
	  if (requestID !== resultsRequest) return;
      items = [];
      selected = null;
      detail = null;
      setError(err);
    } finally {
	  if (requestID === resultsRequest) loading = false;
    }
  }

  async function loadDetail(item) {
    if (!item) return;
	const requestID = ++detailRequest;
    selected = item;
    detailLoading = true;
    try {
      const params = sessionParams();
	  const next = await request(`/api/skillhub/skills/${encodeURIComponent(item.market)}/${encodeURIComponent(item.id)}?${params}`);
	  if (requestID !== detailRequest) return;
	  detail = next;
    } catch (err) {
	  if (requestID !== detailRequest) return;
      detail = null;
      setError(err);
    } finally {
	  if (requestID === detailRequest) detailLoading = false;
    }
  }

  function switchMarket(next) {
    if (market === next) return;
    market = next;
    view = market === 'skillhub.cn' ? 'official' : 'browse';
    resetPaging();
    loadResults();
  }

  function switchView(next) {
    if (next === 'official' && market !== 'skillhub.cn') return;
    view = next;
    resetPaging();
    loadResults();
  }

  function search() {
    view = query.trim() ? 'search' : market === 'skillhub.cn' ? 'official' : 'browse';
    resetPaging();
    loadResults();
  }

  function resetPaging() {
    page = 1;
    cursor = '';
    cursorHistory = [];
    nextCursor = '';
  }

  function previousPage() {
    if (!canPrevious) return;
    if (market === 'clawhub.ai') {
      cursor = cursorHistory[cursorHistory.length - 1] || '';
      cursorHistory = cursorHistory.slice(0, -1);
      page = Math.max(1, page - 1);
    } else {
      page -= 1;
    }
    loadResults();
  }

  function nextPage() {
    if (!canNext) return;
    if (market === 'clawhub.ai') {
      cursorHistory = [...cursorHistory, cursor];
      cursor = nextCursor;
    }
    page += 1;
    loadResults();
  }

  async function loadShowcase(kind = 'recommended') {
    loading = true; clearBanners();
    try { const data = await request(`/api/skillhub/showcase/${encodeURIComponent(kind)}?${sessionParams({ market })}`); items = data?.items || []; total = Number(data?.total || items.length); selected = items[0] || null; if (selected) await loadDetail(selected); } catch (err) { setError(err); } finally { loading = false; }
  }
  function toggleBatch(item) { const next = new Set(selectedBatch); const key = `${item.market}:${item.id}`; if (next.has(key)) next.delete(key); else next.add(key); selectedBatch = next; }
  async function installBatch() { const chosen = items.filter((item) => selectedBatch.has(`${item.market}:${item.id}`)); if (!chosen.length || !sessionID || !targetDir) return; actionLoading = true; try { await postJSON('/api/skillhub/skillset', { skills: chosen.map((item) => ({ market: item.market, id: item.id, version: item.version || '', scope, targetDir })), sessionId: sessionID, targetDir, scope }); selectedBatch = new Set(); setNotice('Selected skills installed.'); await loadResults(); } catch (err) { setError(err); } finally { actionLoading = false; } }
  async function loadFileContent(file) { if (!detail || !file?.path) return; try { const data = await request(`/api/skillhub/content/${encodeURIComponent(detail.market)}/${encodeURIComponent(detail.id)}?${sessionParams({ path: file.path, version: detail.version || '' })}`); file.content = data?.content || ''; detail = { ...detail }; } catch (err) { setError(err); } }
  async function install(activate = false, overwrite = false) {
    if (!detail) return;
    actionLoading = true;
    clearBanners();
    try {
      const data = await postJSON('/api/skillhub/install', {
        market: detail.market,
        id: detail.id,
        version: detail.version || '',
        scope,
        sessionId: sessionID,
        targetDir,
        overwrite,
        activate
      });
      activeSkills = data?.session?.activeSkills || activeSkills;
      setNotice(overwrite ? $t('skills.updated', { name: data?.install?.name || detail.name }) : $t('skills.installed', { name: data?.install?.name || detail.name }));
      await loadResults();
    } catch (err) {
      setError(err);
    } finally {
      actionLoading = false;
    }
  }

  async function activate() {
    const name = installedName(detail);
    if (!name) return;
    actionLoading = true;
    clearBanners();
    try {
      const data = await postJSON('/api/skillhub/activate', { name, sessionId: sessionID });
      activeSkills = data?.session?.activeSkills || activeSkills;
      setNotice($t('skills.activated', { name }));
    } catch (err) {
      setError(err);
    } finally {
      actionLoading = false;
    }
  }

  async function uninstall() {
    if (!detail) return;
    actionLoading = true;
    clearBanners();
    try {
      await postJSON('/api/skillhub/uninstall', { market: detail.market, id: detail.id, scope, sessionId: sessionID });
      activeSkills = activeSkills.filter((name) => name !== installedName(detail));
      setNotice(`Uninstalled ${detail.name || detail.slug}`);
      await loadResults();
    } catch (err) {
      setError(err);
    } finally {
      actionLoading = false;
    }
  }

  function installedName(item) {
    const dir = item?.installed?.dir || '';
    if (dir) return dir.split(/[\\/]/).filter(Boolean).pop() || '';
    return item?.slug || '';
  }

  function isActive(item) {
    const name = installedName(item);
    return name && activeSkills.includes(name);
  }

  function badges(item) {
    const values = [];
    if (view === 'official' && item?.market === 'skillhub.cn') values.push('official');
    if (item?.publisherVerified) values.push('certified');
    if (item?.verified) values.push('verified');
    if (item?.suspicious) values.push('risk');
    if (item?.installed?.installed) values.push('installed');
    if (item?.installed?.updateAvailable) values.push('update');
    if (isActive(item)) values.push('active');
    return values;
  }

  function formatCount(value) {
    return new Intl.NumberFormat().format(Number(value || 0));
  }

  function formatReport(value) {
    if (!value || typeof value !== 'object') return $t('skills.notAvailable');
    return JSON.stringify(value, null, 2);
  }
</script>

<section class="page skills-page">
  <div class="skills-controls">
    <div class="segmented" aria-label={$t('skills.market')}>
      <button type="button" class:active={market === 'skillhub.cn'} on:click={() => switchMarket('skillhub.cn')}>SkillHub.cn</button>
      <button type="button" class:active={market === 'clawhub.ai'} on:click={() => switchMarket('clawhub.ai')}>ClawHub.ai</button>
    </div>
    <div class="segmented skills-view-tabs" aria-label={$t('skills.view')}>
      <button type="button" class:active={view === 'browse'} on:click={() => switchView('browse')}>{$t('skills.browse')}</button>
      <button type="button" class:active={view === 'search'} on:click={() => switchView('search')}>{$t('skills.search')}</button>
      {#if market === 'skillhub.cn'}
        <button type="button" class:active={view === 'official'} on:click={() => switchView('official')}>{$t('skills.official')}</button>
        <button type="button" on:click={() => loadShowcase('recommended')}>Showcase</button>
      {/if}
    </div>
    <form class="skills-search" on:submit|preventDefault={search}>
      <input bind:value={query} placeholder={$t('skills.searchPlaceholder')} aria-label={$t('skills.search')} />
      <button type="submit" class="primary">{$t('skills.search')}</button>
    </form>
  </div>

  {#if market === 'skillhub.cn' && view === 'browse'}
    <div class="skills-filters">
      <label>
        <span>{$t('skills.category')}</span>
        <select bind:value={category} on:change={() => { resetPaging(); loadResults(); }}>
          <option value="">{$t('skills.allCategories')}</option>
          {#each categories as item}
            <option value={item.key}>{item.nameEn || item.name || item.key}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>{$t('skills.sort')}</span>
        <select bind:value={sort} on:change={() => { resetPaging(); loadResults(); }}>
          <option value="downloads">{$t('skills.downloads')}</option>
          <option value="stars">Stars</option>
          <option value="installs">{$t('skills.installs')}</option>
          <option value="score">Score</option>
          <option value="updated_at">{$t('skills.updatedAt')}</option>
        </select>
      </label>
    </div>
  {/if}

  <div class="skills-workbench">
    <section class="skills-list" aria-label={$t('skills.results')}>
      <div class="skills-section-head">
        <strong>{$t('skills.results')}</strong>
        <span class="loading-row">{#if loading}<span class="spinner sm"></span>{$t('common.loading')}{:else}{$t('common.items', { count: total })}{/if}</span>
        <button type="button" disabled={selectedBatch.size === 0 || actionLoading || !sessionID || !targetDir} on:click={installBatch}>{#if actionLoading}<span class="spinner sm"></span> {/if}Install selected ({selectedBatch.size})</button>
      </div>
      <div class="skills-rows">
        {#if loading && items.length === 0}
          <div class="spinner-center"><span class="spinner lg"></span><span>{$t('common.loading')}</span></div>
        {:else}
        {#each items as item (item.market + ':' + item.id)}
          <button type="button" class="skill-row" class:active={selected?.id === item.id} on:click={() => loadDetail(item)}>
            <input type="checkbox" checked={selectedBatch.has(`${item.market}:${item.id}`)} on:click|stopPropagation={() => toggleBatch(item)} aria-label="Select skill" />
            <span class="skill-row-main">
              <strong>{item.displayName || item.name || item.slug || item.id}</strong>
              <span>{item.author || item.publisherName || item.market}</span>
            </span>
            <span class="skill-row-meta">
              <span class="download-count">↓ {formatCount(item.downloads)}</span>
              <span>{item.version || ''}</span>
            </span>
            {#if badges(item).length}
              <span class="skill-badges">
                {#each badges(item) as badge}<span class:warning={badge === 'risk'} class:positive={badge === 'active' || badge === 'certified'}>{badge}</span>{/each}
              </span>
            {/if}
            <span class="skill-description">{item.description || ''}</span>
          </button>
        {/each}
        {#if !loading && items.length === 0}<p class="empty">{$t('skills.empty')}</p>{/if}
        {/if}
      </div>
      <div class="skills-pager">
        <button type="button" class="sm" disabled={!canPrevious || loading} on:click={previousPage} aria-label={$t('common.previous')}>‹</button>
        <span>{$t('skills.page', { page })}</span>
        <button type="button" class="sm" disabled={!canNext || loading} on:click={nextPage} aria-label={$t('common.nextPage')}>›</button>
      </div>
    </section>

    <section class="skills-detail" aria-label={$t('skills.detail')}>
      {#if detailLoading}
        <div class="spinner-center"><span class="spinner lg"></span><span>{$t('common.loading')}</span></div>
      {:else if detail}
        <div class="skills-detail-head">
          <div>
            <h2>{detail.displayName || detail.name || detail.slug}</h2>
            <p>{detail.author || detail.publisherName || detail.market} · {detail.version || $t('skills.notAvailable')}</p>
          </div>
          <span class="download-count">↓ {formatCount(detail.downloads)}</span>
        </div>
        <p class="skills-summary">{detail.description || $t('skills.noDescription')}</p>
        <div class="skill-badges detail-badges">
          {#each badges(detail) as badge}<span class:warning={badge === 'risk'} class:positive={badge === 'active' || badge === 'certified'}>{badge}</span>{/each}
        </div>
        <div class="skills-install-target">
          <div class="skills-install-target-head">
            <div>
              <span class="skills-install-target-kicker">INSTALL TARGET</span>
              <h3>安装目标</h3>
            </div>
            <span class:ready={sessionID && targetDir} class="skills-install-target-state">{sessionID && targetDir ? '已就绪' : '未选择'}</span>
          </div>
          <p class="skills-install-target-hint">请选择要刷新技能上下文的 Session，以及技能实际安装的目录。</p>
          <div class="skills-install-target-fields">
            <label>
              <span>Session</span>
              <select bind:value={selectedSession} on:change={(event) => selectSession(event.currentTarget.value)}>
                <option value="">选择 Session</option>
                {#each $sessions as item}<option value={item.id}>{item.title || item.id} · {item.workDir || '未知目录'}</option>{/each}
              </select>
              {#if selectedSessionInfo}<small>{selectedSessionInfo.workDir || '工作目录未知'}</small>{/if}
            </label>
            <label>
              <span>安装目录</span>
              <select bind:value={targetDir} on:change={() => { scope = targets.find((item) => item.path === targetDir)?.scope || 'project'; }} disabled={!sessionID || targetLoading}>
                <option value="">选择安装目录</option>
                {#each targets as target}<option value={target.path}>{target.label} · {target.path}</option>{/each}
              </select>
              {#if targetDir}<small class="skills-target-path">{targetDir}</small>{/if}
            </label>
          </div>
          <div class="skills-install-target-confirmation">
            <span class="skills-target-dot"></span>
            {#if sessionID && targetDir}
              <span>将安装到 <strong>{targetDir}</strong>，并刷新 Session <strong>{sessionID}</strong></span>
            {:else}
              <span>请选择 Session 和安装目录</span>
            {/if}
          </div>
        </div>

        <div class="skills-actions">
          {#if detail.installed?.updateAvailable}
          <button type="button" class="primary" disabled={actionLoading || !sessionID || !targetDir} on:click={() => install(false, true)}>{#if actionLoading}<span class="spinner sm"></span> {/if}{$t('skills.update')}</button>
          {:else if !detail.installed?.installed}
          <button type="button" class="primary" disabled={actionLoading || !sessionID || !targetDir} on:click={() => install(false, false)}>{#if actionLoading}<span class="spinner sm"></span> {/if}{$t('skills.install')}</button>
          <button type="button" disabled={actionLoading || !sessionID || !targetDir} on:click={() => install(true, false)}>{#if actionLoading}<span class="spinner sm"></span> {/if}{$t('skills.installActivate')}</button>
          {:else if !isActive(detail)}
            <button type="button" class="primary" disabled={actionLoading} on:click={activate}>{#if actionLoading}<span class="spinner sm"></span> {/if}{$t('skills.activate')}</button>
            <button type="button" disabled={actionLoading} on:click={uninstall}>{#if actionLoading}<span class="spinner sm"></span> {/if}Uninstall</button>
          {:else}
            <span class="status-tag">{$t('skills.active')}</span>
            <button type="button" disabled={actionLoading} on:click={uninstall}>{#if actionLoading}<span class="spinner sm"></span> {/if}Uninstall</button>
          {/if}
        </div>
        <dl class="skills-metadata">
          <div><dt>{$t('skills.market')}</dt><dd>{detail.market}</dd></div>
          <div><dt>{$t('skills.category')}</dt><dd>{detail.category || $t('skills.notAvailable')}</dd></div>
          <div><dt>{$t('skills.source')}</dt><dd>{detail.source || $t('skills.notAvailable')}</dd></div>
          <div><dt>{$t('skills.installs')}</dt><dd>{formatCount(detail.installs)}</dd></div>
        </dl>
        {#if detail.tags?.length}
          <div class="skills-block"><h3>Tags</h3><p>{detail.tags.join(', ')}</p></div>
        {/if}
        <div class="skills-block">
          <h3>{$t('skills.downloadSources')}</h3>
          {#if detail.downloadSources?.length}
            <ul class="download-sources">
              {#each detail.downloadSources as source}
                <li><span>{source.kind}{source.fallback ? ` · ${$t('skills.fallback')}` : ''}</span><code>{source.url}</code></li>
              {/each}
            </ul>
          {:else}<p>{$t('skills.notAvailable')}</p>{/if}
        </div>
        <div class="skills-block">
          <h3>{$t('skills.files')} ({detail.files?.length || 0})</h3>
          <ul class="skill-files">
            {#each detail.files || [] as file}<li><code>{file.path}</code><span>{formatCount(file.size)} B <button type="button" on:click={() => loadFileContent(file)}>View</button></span>{#if file.content}<pre>{file.content}</pre>{/if}</li>{/each}
          </ul>
        </div>
        <details class="skills-report"><summary>{$t('skills.security')}</summary><pre>{formatReport(detail.securityReports)}</pre></details>
        <details class="skills-report"><summary>{$t('skills.evaluation')}</summary><pre>{formatReport(detail.evaluation)}</pre></details>
      {:else}
        <p class="empty">{$t('skills.select')}</p>
      {/if}
    </section>
  </div>
</section>
