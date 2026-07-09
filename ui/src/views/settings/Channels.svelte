<script>
  import { onDestroy } from 'svelte';
  import { channels, serveConfig, refreshAll, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { del, postJSON, putJSON, request } from '../../lib/api.js';
  import { t } from '../../lib/preferences.js';
  import ListEditor from './ListEditor.svelte';

  let form = defaultForm();
  let lastRaw = '';
  let parseError = '';
  let saving = false;
  let feishuOpen = false;
  let feishuDraft = defaultForm().feishu;
  let wechatOpen = false;
  let wechatLogin = null;
  let wechatPoll = null;
  let wechatQRDataURL = '';
  let wechatQRSource = '';
  let wechatQRLoading = false;
  let wechatQRError = '';

  $: syncFromStore($serveConfig);

  onDestroy(() => {
    stopWechatPolling();
  });

  function defaultForm() {
    return {
      websocket: { enabled: false },
      wechat: { enabled: false, credPath: '', workDir: '', autoTyping: true, allowedUsers: [] },
      feishu: { enabled: false, appID: '', appSecret: '', workDir: '', allowedUsers: [] }
    };
  }

  function syncFromStore(raw) {
    if (raw === lastRaw) return;
    lastRaw = raw;
    try {
      const cfg = JSON.parse(raw || '{}');
      form = {
        websocket: { enabled: Boolean(cfg.features?.websocket) },
        wechat: {
          enabled: Boolean(cfg.channels?.wechat?.enabled ?? cfg.features?.wechat),
          credPath: stringValue(cfg.channels?.wechat?.credPath),
          workDir: stringValue(cfg.channels?.wechat?.workDir),
          autoTyping: readBool(cfg.channels?.wechat?.autoTyping, true),
          allowedUsers: arrayValue(cfg.channels?.wechat?.allowedUsers)
        },
        feishu: {
          enabled: Boolean(cfg.channels?.feishu?.enabled ?? cfg.features?.feishu),
          appID: stringValue(cfg.channels?.feishu?.appId),
          appSecret: stringValue(cfg.channels?.feishu?.appSecret),
          workDir: stringValue(cfg.channels?.feishu?.workDir),
          allowedUsers: arrayValue(cfg.channels?.feishu?.allowedUsers)
        }
      };
      parseError = '';
    } catch (err) {
      parseError = err instanceof Error ? err.message : String(err);
      form = defaultForm();
    }
  }

  function stringValue(value) {
    return typeof value === 'string' ? value : '';
  }

  function readBool(value, fallback) {
    return typeof value === 'boolean' ? value : fallback;
  }

  function arrayValue(value) {
    return Array.isArray(value) ? value.map((item) => String(item ?? '')) : [];
  }

  function statusFor(name) {
    return $channels.find((item) => item.name === name) || { name, enabled: false, connected: false };
  }

  function statusLabel(name) {
    const status = statusFor(name);
    if (!status.enabled) return $t('common.disabledState');
    return status.connected ? $t('common.connected') : $t('common.disconnected');
  }

  function buildConfig() {
    const cfg = JSON.parse(lastRaw || '{}');
    cfg.features = ensureObject(cfg, 'features');
    cfg.channels = ensureObject(cfg, 'channels');
    cfg.channels.wechat = ensureObject(cfg.channels, 'wechat');
    cfg.channels.feishu = ensureObject(cfg.channels, 'feishu');

    cfg.features.websocket = Boolean(form.websocket.enabled);
    cfg.features.wechat = Boolean(form.wechat.enabled);
    cfg.features.feishu = Boolean(form.feishu.enabled);

    cfg.channels.wechat.enabled = Boolean(form.wechat.enabled);
    cfg.channels.wechat.autoTyping = Boolean(form.wechat.autoTyping);
    writeString(cfg.channels.wechat, 'credPath', form.wechat.credPath);
    writeString(cfg.channels.wechat, 'workDir', form.wechat.workDir);
    writeList(cfg.channels.wechat, 'allowedUsers', form.wechat.allowedUsers);

    cfg.channels.feishu.enabled = Boolean(form.feishu.enabled);
    writeString(cfg.channels.feishu, 'appId', form.feishu.appID);
    writeString(cfg.channels.feishu, 'appSecret', form.feishu.appSecret);
    writeString(cfg.channels.feishu, 'workDir', form.feishu.workDir);
    writeList(cfg.channels.feishu, 'allowedUsers', form.feishu.allowedUsers);
    return cfg;
  }

  function ensureObject(parent, key) {
    if (!parent[key] || typeof parent[key] !== 'object' || Array.isArray(parent[key])) parent[key] = {};
    return parent[key];
  }

  function writeString(target, key, value) {
    const text = String(value || '').trim();
    if (text) target[key] = text;
    else delete target[key];
  }

  function writeList(target, key, values) {
    const list = values.map((item) => String(item || '').trim()).filter(Boolean);
    if (list.length > 0) target[key] = list;
    else delete target[key];
  }

  async function saveConfig(noticeKey = 'settings.channels.saved') {
    clearBanners();
    saving = true;
    try {
      const saved = await putJSON('/api/serve/config', buildConfig());
      serveConfig.set(JSON.stringify(saved, null, 2));
      await refreshAll();
      if (noticeKey) setNotice($t(noticeKey));
      return saved;
    } catch (err) {
      setError(err);
      throw err;
    } finally {
      saving = false;
    }
  }

  async function toggleWebSocket(event) {
    form.websocket.enabled = event.currentTarget.checked;
    form = form;
    try {
      await saveConfig(form.websocket.enabled ? 'settings.channels.websocketEnabled' : 'settings.channels.websocketDisabled');
    } catch {
      syncFromStore($serveConfig);
    }
  }

  function openFeishu() {
    feishuDraft = cloneChannel(form.feishu);
    feishuOpen = true;
  }

  function closeFeishu() {
    feishuOpen = false;
  }

  async function saveFeishu() {
    if (feishuDraft.enabled && (!feishuDraft.appID.trim() || !feishuDraft.appSecret.trim())) {
      setError($t('settings.channels.feishuKeyRequired'));
      return;
    }
    form.feishu = cloneChannel(feishuDraft);
    form = form;
    await saveConfig('settings.channels.feishuSaved');
    feishuOpen = false;
  }

  function cloneChannel(channel) {
    return {
      ...channel,
      allowedUsers: [...(channel.allowedUsers || [])]
    };
  }

  function addDraftUser(target) {
    target.allowedUsers = [...target.allowedUsers, ''];
  }

  function removeDraftUser(target, index) {
    target.allowedUsers = target.allowedUsers.filter((_, i) => i !== index);
  }

  function addListItem(list) {
    list.push('');
    form = form;
  }

  function removeListItem(list, index) {
    list.splice(index, 1);
    form = form;
  }

  async function saveWechatConfig() {
    await saveConfig('settings.channels.wechatSaved');
  }

  async function disableWechat() {
    form.wechat.enabled = false;
    form = form;
    await saveConfig('settings.channels.wechatDisabled');
  }

  async function startWechatLogin() {
    clearBanners();
    wechatOpen = true;
    wechatLogin = { state: 'starting' };
    resetWechatQRData();
    try {
      await saveConfig('');
      wechatLogin = await postJSON('/api/channels/wechat/login', {});
      loadWechatQRData(wechatLogin);
      startWechatPolling();
    } catch (err) {
      setError(err);
    }
  }

  function startWechatPolling() {
    stopWechatPolling();
    wechatPoll = window.setInterval(loadWechatLogin, 1800);
    loadWechatLogin();
  }

  function stopWechatPolling() {
    if (wechatPoll) window.clearInterval(wechatPoll);
    wechatPoll = null;
  }

  async function loadWechatLogin() {
    try {
      wechatLogin = await request('/api/channels/wechat/login');
      loadWechatQRData(wechatLogin);
      if (wechatLogin?.state === 'confirmed') {
        stopWechatPolling();
        form.wechat.enabled = true;
        form = form;
        await refreshAll();
        setNotice($t('settings.channels.wechatEnabled'));
      } else if (wechatLogin?.state === 'error' || wechatLogin?.state === 'cancelled') {
        stopWechatPolling();
      }
    } catch (err) {
      stopWechatPolling();
      setError(err);
    }
  }

  async function closeWechatLogin() {
    stopWechatPolling();
    if (wechatLogin && !['confirmed', 'error', 'cancelled'].includes(wechatLogin.state)) {
      try {
        await del('/api/channels/wechat/login');
      } catch {
        // Closing the modal should not mask the current page state.
      }
    }
    wechatOpen = false;
    resetWechatQRData();
  }

  function resetWechatQRData() {
    wechatQRDataURL = '';
    wechatQRSource = '';
    wechatQRLoading = false;
    wechatQRError = '';
  }

  async function loadWechatQRData(login) {
    const source = login?.qrUrl || '';
    if (!source) {
      resetWechatQRData();
      return;
    }
    if (wechatQRSource === source && (wechatQRDataURL || wechatQRLoading)) return;
    wechatQRSource = source;
    wechatQRDataURL = '';
    wechatQRError = '';
    wechatQRLoading = true;
    try {
      const data = await request(addQueryParam(source, 'format', 'base64'));
      if (wechatQRSource !== source) return;
      wechatQRDataURL = data?.dataUrl || (data?.base64 ? `data:${data?.contentType || 'image/png'};base64,${data.base64}` : '');
      if (!wechatQRDataURL) wechatQRError = $t('settings.channels.wechatNoQr');
    } catch (err) {
      if (wechatQRSource !== source) return;
      wechatQRError = err instanceof Error ? err.message : String(err);
    } finally {
      if (wechatQRSource === source) wechatQRLoading = false;
    }
  }

  function addQueryParam(path, key, value) {
    return `${path}${path.includes('?') ? '&' : '?'}${encodeURIComponent(key)}=${encodeURIComponent(value)}`;
  }

  function qrEmbedSrc(value) {
    const text = String(value || '').trim();
    if (!text || text.startsWith('/') || text.startsWith('http://') || text.startsWith('https://') || text.startsWith('data:')) return text;
    return `data:image/png;base64,${text}`;
  }

  function openWechatQRTab() {
    const url = wechatLogin?.qrOpenUrl || wechatLogin?.qrUrl || '';
    if (!url) return;
    const win = window.open(qrEmbedSrc(url), '_blank');
    if (!win) setError($t('settings.channels.popupBlocked'));
  }
</script>

<div class="page-toolbar embedded">
  <button type="button" class="ghost" on:click={refreshAll}>{$t('common.refresh')}</button>
</div>

{#if parseError}
  <p class="error-text">{$t('settings.app.parseError', { error: parseError })}</p>
{/if}

<div class="channel-settings-grid">
  <div class="card channel-config-card">
    <div class="card-head">
      <div>
        <h3>WebSocket</h3>
        <span class="hint">{$t('settings.channels.websocketHint')}</span>
      </div>
      <span class="pill" class:on={statusFor('websocket').connected} class:off={!statusFor('websocket').connected}>
        {statusLabel('websocket')}
      </span>
    </div>
    <div class="channel-card-body">
      <label class="channel-switch">
        <input type="checkbox" checked={form.websocket.enabled} disabled={saving} on:change={toggleWebSocket} />
        <span>{$t('settings.channels.websocketToggle')}</span>
      </label>
      <p class="hint">/ws</p>
    </div>
  </div>

  <div class="card channel-config-card">
    <div class="card-head">
      <div>
        <h3>Feishu</h3>
        <span class="hint">{$t('settings.channels.feishuHint')}</span>
      </div>
      <span class="pill" class:on={statusFor('feishu').connected} class:off={!statusFor('feishu').connected}>
        {statusLabel('feishu')}
      </span>
    </div>
    <div class="channel-card-body">
      <dl class="kv compact">
        <dt>App ID</dt><dd>{form.feishu.appID || $t('common.uninitialized')}</dd>
        <dt>{$t('sessions.workDir')}</dt><dd>{form.feishu.workDir || $t('common.uninitialized')}</dd>
      </dl>
      <div class="channel-card-actions">
        <button type="button" class="primary" on:click={openFeishu}>{$t('settings.channels.configure')}</button>
      </div>
    </div>
  </div>

  <div class="card channel-config-card">
    <div class="card-head">
      <div>
        <h3>WeChat</h3>
        <span class="hint">{$t('settings.channels.wechatHint')}</span>
      </div>
      <span class="pill" class:on={statusFor('wechat').connected} class:off={!statusFor('wechat').connected}>
        {statusLabel('wechat')}
      </span>
    </div>
    <div class="channel-card-body">
      <div class="form-grid compact-grid">
        <label><span>{$t('settings.serve.wechatCred')}</span><input bind:value={form.wechat.credPath} placeholder="wechat-credentials.json" /></label>
        <label><span>{$t('settings.serve.wechatWorkDir')}</span><input bind:value={form.wechat.workDir} placeholder="/home/user/project" /></label>
        <label class="checkbox"><input type="checkbox" bind:checked={form.wechat.autoTyping} /> <span>{$t('settings.serve.autoTyping')}</span></label>
      </div>
      <ListEditor title={$t('settings.serve.wechatUsers')} list={form.wechat.allowedUsers} onAdd={() => addListItem(form.wechat.allowedUsers)} onRemove={(i) => removeListItem(form.wechat.allowedUsers, i)} />
      <div class="channel-card-actions">
        <button type="button" class="ghost" disabled={saving} on:click={saveWechatConfig}>{$t('common.save')}</button>
        <button type="button" class="primary" disabled={saving} on:click={startWechatLogin}>{$t(form.wechat.enabled ? 'settings.channels.wechatRelogin' : 'settings.channels.wechatScanEnable')}</button>
        {#if form.wechat.enabled}
          <button type="button" class="ghost danger" disabled={saving} on:click={disableWechat}>{$t('common.disable')}</button>
        {/if}
      </div>
    </div>
  </div>
</div>

{#if feishuOpen}
  <div class="channel-modal-overlay" role="dialog" aria-modal="true" aria-label={$t('settings.channels.feishuConfig')}>
    <div class="channel-modal">
      <header>
        <div>
          <h3>{$t('settings.channels.feishuConfig')}</h3>
          <span class="hint">{$t('settings.channels.feishuConfigHint')}</span>
        </div>
        <button type="button" class="ghost sm" on:click={closeFeishu}>{$t('common.close')}</button>
      </header>
      <div class="form-grid">
        <label class="checkbox"><input type="checkbox" bind:checked={feishuDraft.enabled} /> <span>{$t('common.enabled')}</span></label>
        <label><span>{$t('settings.serve.feishuAppID')}</span><input bind:value={feishuDraft.appID} /></label>
        <label><span>{$t('settings.serve.feishuAppSecret')}</span><input type="password" bind:value={feishuDraft.appSecret} /></label>
        <label><span>{$t('settings.serve.feishuWorkDir')}</span><input bind:value={feishuDraft.workDir} placeholder="/home/user/project" /></label>
      </div>
      <div class="form-body">
        <div class="list-editor">
          <div class="list-head">
            <span>{$t('settings.serve.feishuUsers')}</span>
            <button type="button" class="ghost sm" on:click={() => addDraftUser(feishuDraft)}>{$t('common.add')}</button>
          </div>
          {#each feishuDraft.allowedUsers as user, i (i)}
            <div class="inline-row">
              <input bind:value={feishuDraft.allowedUsers[i]} />
              <button type="button" class="ghost sm" on:click={() => removeDraftUser(feishuDraft, i)}>{$t('common.remove')}</button>
            </div>
          {/each}
        </div>
      </div>
      <footer>
        <button type="button" class="ghost" on:click={closeFeishu}>{$t('dirBrowser.cancel')}</button>
        <button type="button" class="primary" disabled={saving} on:click={saveFeishu}>{$t('common.save')}</button>
      </footer>
    </div>
  </div>
{/if}

{#if wechatOpen}
  <div class="channel-modal-overlay" role="dialog" aria-modal="true" aria-label={$t('settings.channels.wechatLogin')}>
    <div class="channel-modal qr-modal">
      <header>
        <div>
          <h3>{$t('settings.channels.wechatLogin')}</h3>
          <span class="hint">{$t('settings.channels.wechatLoginHint')}</span>
        </div>
        <button type="button" class="ghost sm" on:click={closeWechatLogin}>{$t('common.close')}</button>
      </header>
      <div class="qr-panel">
        {#if wechatQRDataURL}
          <img class="qr-image" src={wechatQRDataURL} alt={$t('settings.channels.wechatLogin')} />
        {:else if wechatQRLoading || wechatLogin?.qrUrl || wechatLogin?.state === 'starting'}
          <p class="empty">{$t('common.loading')}</p>
        {:else}
          <p class="empty">{$t('settings.channels.wechatNoQr')}</p>
        {/if}
        <div class="qr-status">
          <span class="pill" class:on={wechatLogin?.state === 'confirmed'} class:off={wechatLogin?.state === 'error' || wechatLogin?.state === 'cancelled'}>
            {$t(`settings.channels.wechatState.${wechatLogin?.state || 'idle'}`)}
          </span>
          {#if wechatLogin?.userId}
            <code>{wechatLogin.userId}</code>
          {/if}
          {#if wechatLogin?.error}
            <p class="error-text">{wechatLogin.error}</p>
          {/if}
          {#if wechatQRError}
            <p class="error-text">{wechatQRError}</p>
          {/if}
        </div>
      </div>
      <footer>
        <button type="button" class="ghost" on:click={closeWechatLogin}>{$t('dirBrowser.cancel')}</button>
        <button type="button" class="ghost" disabled={!wechatLogin?.qrUrl && !wechatLogin?.qrOpenUrl} on:click={openWechatQRTab}>{$t('settings.channels.openQrTab')}</button>
        <button type="button" class="primary" on:click={startWechatLogin}>{$t('settings.channels.refreshQr')}</button>
      </footer>
    </div>
  </div>
{/if}
