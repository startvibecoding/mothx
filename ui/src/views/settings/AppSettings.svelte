<script>
  import { settings, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';
  import { t } from '../../lib/preferences.js';
  import ListEditor from './ListEditor.svelte';

  export let section = 'app';

  let form = defaultForm();
  let jsonDraft = '{}';
  let parseError = '';
  let lastRaw = '';
  let selectedProviderID = '';

  $: isProviderSettings = section === 'providers';
  $: syncFromStore($settings);
  $: currentProvider = form.providers.find((item) => item.id === selectedProviderID) || form.providers[0] || null;
  $: defaultProvider = form.providers.find((item) => item.id === form.defaults.defaultProvider) || null;
  $: defaultModelOptions = modelOptionsForProvider(defaultProvider);
  $: defaultProviderMissing = Boolean(form.defaults.defaultProvider) && !form.providers.some((item) => item.id === form.defaults.defaultProvider);
  $: defaultModelMissing = Boolean(form.defaults.defaultModel) && !defaultModelOptions.some((model) => model.id === form.defaults.defaultModel);

  function defaultForm() {
    return {
      defaults: {
        defaultProvider: '',
        defaultModel: '',
        defaultMode: 'agent',
        defaultThinkingLevel: 'medium',
        theme: 'dark',
        enablePlanTool: '',
        updateCheck: '',
        maxContextTokens: '',
        maxOutputTokens: '',
        skillsDir: '',
        sessionDir: '',
        shellPath: '',
        shellCommandPrefix: ''
      },
      webSearch: { enabled: '', provider: '', providerType: '', model: '' },
      statusLine: { enabled: false, type: 'command', command: '', padding: 0, refreshInterval: '', timeoutMs: '', fallback: '' },
      contextFiles: { enabled: true, extraFiles: [] },
      compaction: {
        enabled: true,
        reserveTokens: 16384,
        keepRecentTokens: 20000,
        tokenizer: '',
        tokenizerModel: '',
        template: '',
        idleCompressionEnabled: false,
        idleTimeoutSeconds: '',
        idleMinTokensForCompress: ''
      },
      sandbox: {
        enabled: false,
        level: 'none',
        bwrapPath: '',
        allowNetwork: false,
        allowedRead: [],
        allowedWrite: [],
        deniedPaths: [],
        passEnv: [],
        tmpSize: ''
      },
      retry: { enabled: true, maxRetries: 5, baseDelayMs: 3000 },
      approval: { bashWhitelist: [], bashBlacklist: [], confirmBeforeWrite: '' },
      providers: []
    };
  }

  function syncFromStore(raw) {
    if (raw === lastRaw) return;
    lastRaw = raw;
    jsonDraft = raw || '{}';
    try {
      const cfg = JSON.parse(raw || '{}');
      form = formFromConfig(cfg);
      parseError = '';
      if (!selectedProviderID || !form.providers.some((item) => item.id === selectedProviderID)) {
        selectedProviderID = cfg.defaultProvider || form.providers[0]?.id || '';
      }
    } catch (err) {
      parseError = err instanceof Error ? err.message : String(err);
      form = defaultForm();
      selectedProviderID = '';
    }
  }

  function formFromConfig(cfg = {}) {
    const base = defaultForm();
    return {
      defaults: {
        defaultProvider: stringValue(cfg.defaultProvider, ''),
        defaultModel: stringValue(cfg.defaultModel, ''),
        defaultMode: stringValue(cfg.defaultMode, base.defaults.defaultMode),
        defaultThinkingLevel: stringValue(cfg.defaultThinkingLevel, base.defaults.defaultThinkingLevel),
        theme: stringValue(cfg.theme, base.defaults.theme),
        enablePlanTool: triBool(cfg.enablePlanTool),
        updateCheck: triBool(cfg.updateCheck),
        maxContextTokens: optionalNumber(cfg.maxContextTokens),
        maxOutputTokens: optionalNumber(cfg.maxOutputTokens),
        skillsDir: stringValue(cfg.skillsDir, ''),
        sessionDir: stringValue(cfg.sessionDir, ''),
        shellPath: stringValue(cfg.shellPath, ''),
        shellCommandPrefix: stringValue(cfg.shellCommandPrefix, '')
      },
      webSearch: {
        enabled: triBool(cfg.webSearch?.enabled),
        provider: stringValue(cfg.webSearch?.provider, ''),
        providerType: stringValue(cfg.webSearch?.providerType, ''),
        model: stringValue(cfg.webSearch?.model, '')
      },
      statusLine: {
        enabled: Boolean(cfg.statusLine?.enabled),
        type: stringValue(cfg.statusLine?.type, base.statusLine.type),
        command: stringValue(cfg.statusLine?.command, ''),
        padding: numberValue(cfg.statusLine?.padding, base.statusLine.padding),
        refreshInterval: optionalNumber(cfg.statusLine?.refreshInterval),
        timeoutMs: optionalNumber(cfg.statusLine?.timeoutMs),
        fallback: stringValue(cfg.statusLine?.fallback, '')
      },
      contextFiles: {
        enabled: readBool(cfg.contextFiles?.enabled, base.contextFiles.enabled),
        extraFiles: arrayValue(cfg.contextFiles?.extraFiles)
      },
      compaction: {
        enabled: readBool(cfg.compaction?.enabled, base.compaction.enabled),
        reserveTokens: numberValue(cfg.compaction?.reserveTokens, base.compaction.reserveTokens),
        keepRecentTokens: numberValue(cfg.compaction?.keepRecentTokens, base.compaction.keepRecentTokens),
        tokenizer: stringValue(cfg.compaction?.tokenizer, ''),
        tokenizerModel: stringValue(cfg.compaction?.tokenizerModel, ''),
        template: stringValue(cfg.compaction?.template, ''),
        idleCompressionEnabled: Boolean(cfg.compaction?.idleCompressionEnabled),
        idleTimeoutSeconds: optionalNumber(cfg.compaction?.idleTimeoutSeconds),
        idleMinTokensForCompress: optionalNumber(cfg.compaction?.idleMinTokensForCompress)
      },
      sandbox: {
        enabled: readBool(cfg.sandbox?.enabled, base.sandbox.enabled),
        level: stringValue(cfg.sandbox?.level, base.sandbox.level),
        bwrapPath: stringValue(cfg.sandbox?.bwrapPath, ''),
        allowNetwork: Boolean(cfg.sandbox?.allowNetwork),
        allowedRead: arrayValue(cfg.sandbox?.allowedRead),
        allowedWrite: arrayValue(cfg.sandbox?.allowedWrite),
        deniedPaths: arrayValue(cfg.sandbox?.deniedPaths),
        passEnv: arrayValue(cfg.sandbox?.passEnv),
        tmpSize: stringValue(cfg.sandbox?.tmpSize, '')
      },
      retry: {
        enabled: readBool(cfg.retry?.enabled, base.retry.enabled),
        maxRetries: numberValue(cfg.retry?.maxRetries, base.retry.maxRetries),
        baseDelayMs: numberValue(cfg.retry?.baseDelayMs, base.retry.baseDelayMs)
      },
      approval: {
        bashWhitelist: arrayValue(cfg.approval?.bashWhitelist),
        bashBlacklist: arrayValue(cfg.approval?.bashBlacklist),
        confirmBeforeWrite: triBool(cfg.approval?.confirmBeforeWrite)
      },
      providers: providersFromConfig(cfg.providers || {})
    };
  }

  function providersFromConfig(providers = {}) {
    return Object.entries(providers)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([id, provider]) => ({
        id,
        raw: { ...(provider || {}) },
        vendor: stringValue(provider?.vendor, ''),
        apiKey: stringValue(provider?.apiKey, ''),
        baseUrl: stringValue(provider?.baseUrl, ''),
        httpProxy: stringValue(provider?.httpProxy, ''),
        forceHTTP11: Boolean(provider?.forceHTTP11),
        api: stringValue(provider?.api, ''),
        thinkingFormat: stringValue(provider?.thinkingFormat, ''),
        cacheControl: triBool(provider?.cacheControl),
        headers: mapToPairs(provider?.headers),
        responses: {
          reasoningSummary: stringValue(provider?.responses?.reasoningSummary, ''),
          promptCacheEnabled: triBool(provider?.responses?.promptCacheEnabled),
          promptCacheKey: stringValue(provider?.responses?.promptCacheKey, ''),
          promptCacheRetention: stringValue(provider?.responses?.promptCacheRetention, '')
        },
        models: arrayValue(provider?.models).map((model) => modelFromConfig(model)).filter((model) => model.id || model.name)
      }));
  }

  function modelFromConfig(model = {}) {
    return {
      raw: { ...(model || {}) },
      id: stringValue(model?.id, ''),
      name: stringValue(model?.name, ''),
      reasoning: Boolean(model?.reasoning),
      contextWindow: optionalNumber(model?.contextWindow),
      maxTokens: optionalNumber(model?.maxTokens),
      input: arrayValue(model?.input).join(', '),
      temperature: optionalNumber(model?.temperature),
      topP: optionalNumber(model?.top_p)
    };
  }

  function buildConfigForSave() {
    const cfg = JSON.parse(jsonDraft || '{}');

    if (isProviderSettings) {
      cfg.defaultProvider = form.defaults.defaultProvider.trim();
      cfg.defaultModel = form.defaults.defaultModel.trim();
      cfg.defaultThinkingLevel = form.defaults.defaultThinkingLevel || 'medium';
      writeOptionalNumber(cfg, 'maxContextTokens', form.defaults.maxContextTokens);
      writeOptionalNumber(cfg, 'maxOutputTokens', form.defaults.maxOutputTokens);
      cfg.providers = providersToConfig(form.providers);
      return cfg;
    }

    cfg.defaultMode = form.defaults.defaultMode || 'agent';
    cfg.theme = form.defaults.theme || 'dark';
    writeTriBool(cfg, 'enablePlanTool', form.defaults.enablePlanTool);
    writeTriBool(cfg, 'updateCheck', form.defaults.updateCheck);
    writeString(cfg, 'skillsDir', form.defaults.skillsDir);
    writeString(cfg, 'sessionDir', form.defaults.sessionDir);
    writeString(cfg, 'shellPath', form.defaults.shellPath);
    writeString(cfg, 'shellCommandPrefix', form.defaults.shellCommandPrefix);

    cfg.webSearch = ensureObject(cfg, 'webSearch');
    writeTriBool(cfg.webSearch, 'enabled', form.webSearch.enabled);
    writeString(cfg.webSearch, 'provider', form.webSearch.provider);
    writeString(cfg.webSearch, 'providerType', form.webSearch.providerType);
    writeString(cfg.webSearch, 'model', form.webSearch.model);

    cfg.statusLine = ensureObject(cfg, 'statusLine');
    cfg.statusLine.enabled = Boolean(form.statusLine.enabled);
    writeString(cfg.statusLine, 'type', form.statusLine.type);
    writeString(cfg.statusLine, 'command', form.statusLine.command);
    cfg.statusLine.padding = numberValue(form.statusLine.padding, 0);
    writeOptionalNumber(cfg.statusLine, 'refreshInterval', form.statusLine.refreshInterval);
    writeOptionalNumber(cfg.statusLine, 'timeoutMs', form.statusLine.timeoutMs);
    writeString(cfg.statusLine, 'fallback', form.statusLine.fallback);

    cfg.contextFiles = ensureObject(cfg, 'contextFiles');
    cfg.contextFiles.enabled = Boolean(form.contextFiles.enabled);
    writeList(cfg.contextFiles, 'extraFiles', form.contextFiles.extraFiles);

    cfg.compaction = ensureObject(cfg, 'compaction');
    cfg.compaction.enabled = Boolean(form.compaction.enabled);
    cfg.compaction.reserveTokens = numberValue(form.compaction.reserveTokens, 0);
    cfg.compaction.keepRecentTokens = numberValue(form.compaction.keepRecentTokens, 0);
    writeString(cfg.compaction, 'tokenizer', form.compaction.tokenizer);
    writeString(cfg.compaction, 'tokenizerModel', form.compaction.tokenizerModel);
    writeString(cfg.compaction, 'template', form.compaction.template);
    cfg.compaction.idleCompressionEnabled = Boolean(form.compaction.idleCompressionEnabled);
    writeOptionalNumber(cfg.compaction, 'idleTimeoutSeconds', form.compaction.idleTimeoutSeconds);
    writeOptionalNumber(cfg.compaction, 'idleMinTokensForCompress', form.compaction.idleMinTokensForCompress);

    cfg.sandbox = ensureObject(cfg, 'sandbox');
    cfg.sandbox.enabled = Boolean(form.sandbox.enabled);
    cfg.sandbox.level = form.sandbox.level || 'none';
    cfg.sandbox.allowNetwork = Boolean(form.sandbox.allowNetwork);
    writeString(cfg.sandbox, 'bwrapPath', form.sandbox.bwrapPath);
    writeString(cfg.sandbox, 'tmpSize', form.sandbox.tmpSize);
    writeList(cfg.sandbox, 'allowedRead', form.sandbox.allowedRead);
    writeList(cfg.sandbox, 'allowedWrite', form.sandbox.allowedWrite);
    writeList(cfg.sandbox, 'deniedPaths', form.sandbox.deniedPaths);
    writeList(cfg.sandbox, 'passEnv', form.sandbox.passEnv);

    cfg.retry = ensureObject(cfg, 'retry');
    cfg.retry.enabled = Boolean(form.retry.enabled);
    cfg.retry.maxRetries = numberValue(form.retry.maxRetries, 0);
    cfg.retry.baseDelayMs = numberValue(form.retry.baseDelayMs, 0);

    cfg.approval = ensureObject(cfg, 'approval');
    writeList(cfg.approval, 'bashWhitelist', form.approval.bashWhitelist);
    writeList(cfg.approval, 'bashBlacklist', form.approval.bashBlacklist);
    writeTriBool(cfg.approval, 'confirmBeforeWrite', form.approval.confirmBeforeWrite);

    return cfg;
  }

  function providersToConfig(providers = []) {
    const out = {};
    for (const provider of providers) {
      const id = provider.id.trim();
      if (!id) continue;
      const raw = { ...(provider.raw || {}) };
      writeString(raw, 'vendor', provider.vendor);
      writeString(raw, 'apiKey', provider.apiKey);
      writeString(raw, 'baseUrl', provider.baseUrl);
      writeString(raw, 'httpProxy', provider.httpProxy);
      if (provider.forceHTTP11) raw.forceHTTP11 = true;
      else delete raw.forceHTTP11;
      writeString(raw, 'api', provider.api);
      writeString(raw, 'thinkingFormat', provider.thinkingFormat);
      writeTriBool(raw, 'cacheControl', provider.cacheControl);
      writeMap(raw, 'headers', provider.headers);
      raw.responses = ensureObject(raw, 'responses');
      writeString(raw.responses, 'reasoningSummary', provider.responses.reasoningSummary);
      writeTriBool(raw.responses, 'promptCacheEnabled', provider.responses.promptCacheEnabled);
      writeString(raw.responses, 'promptCacheKey', provider.responses.promptCacheKey);
      writeString(raw.responses, 'promptCacheRetention', provider.responses.promptCacheRetention);
      if (Object.keys(raw.responses).length === 0) delete raw.responses;
      raw.models = provider.models.map(modelToConfig).filter((model) => model.id);
      out[id] = raw;
    }
    return out;
  }

  function modelToConfig(model) {
    const raw = { ...(model.raw || {}) };
    raw.id = model.id.trim();
    raw.name = model.name.trim() || raw.id;
    if (model.reasoning) raw.reasoning = true;
    else delete raw.reasoning;
    writeOptionalNumber(raw, 'contextWindow', model.contextWindow);
    writeOptionalNumber(raw, 'maxTokens', model.maxTokens);
    const input = csvList(model.input);
    if (input.length > 0) raw.input = input;
    else delete raw.input;
    writeOptionalFloat(raw, 'temperature', model.temperature);
    writeOptionalFloat(raw, 'top_p', model.topP);
    return raw;
  }

  async function save() {
    clearBanners();
    try {
      const next = buildConfigForSave();
      const saved = await putJSON('/api/settings', next);
      settings.set(JSON.stringify(saved, null, 2));
      setNotice($t(isProviderSettings ? 'settings.providers.saved' : 'settings.app.saved'));
    } catch (err) {
      setError(err);
    }
  }

  function addProvider() {
    const id = uniqueProviderID('provider');
    form.providers = [...form.providers, {
      id,
      raw: {},
      vendor: '',
      apiKey: '',
      baseUrl: '',
      httpProxy: '',
      forceHTTP11: false,
      api: 'openai-chat',
      thinkingFormat: '',
      cacheControl: '',
      headers: [],
      responses: { reasoningSummary: '', promptCacheEnabled: '', promptCacheKey: '', promptCacheRetention: '' },
      models: []
    }];
    selectedProviderID = id;
  }

  function removeProvider(provider) {
    form.providers = form.providers.filter((item) => item !== provider);
    if (selectedProviderID === provider.id) selectedProviderID = form.providers[0]?.id || '';
  }

  function selectDefaultProvider(value) {
    form.defaults.defaultProvider = value;
    const provider = form.providers.find((item) => item.id === value) || null;
    const models = modelOptionsForProvider(provider);
    if (!models.some((model) => model.id === form.defaults.defaultModel)) {
      form.defaults.defaultModel = models[0]?.id || '';
    }
    form = form;
  }

  function selectDefaultModel(value) {
    form.defaults.defaultModel = value;
    form = form;
  }

  function renameProvider(provider, value) {
    provider.id = value.trim();
    selectedProviderID = provider.id;
    form = form;
  }

  function addModel(provider) {
    provider.models = [...provider.models, {
      raw: {},
      id: '',
      name: '',
      reasoning: false,
      contextWindow: '',
      maxTokens: '',
      input: 'text',
      temperature: '',
      topP: ''
    }];
    form = form;
  }

  function removeModel(provider, index) {
    provider.models = provider.models.filter((_, i) => i !== index);
    form = form;
  }

  function addHeader(provider) {
    provider.headers = [...provider.headers, { key: '', value: '' }];
    form = form;
  }

  function removeHeader(provider, index) {
    provider.headers = provider.headers.filter((_, i) => i !== index);
    form = form;
  }

  function addListItem(list) {
    list.push('');
    form = form;
  }

  function removeListItem(list, index) {
    list.splice(index, 1);
    form = form;
  }

  function modelOptionsForProvider(provider) {
    const seen = new Set();
    const models = [];
    for (const model of provider?.models || []) {
      const id = String(model?.id || '').trim();
      if (!id || seen.has(id)) continue;
      seen.add(id);
      models.push({ id, name: String(model?.name || '').trim() });
    }
    return models;
  }

  function uniqueProviderID(prefix) {
    let n = 1;
    let id = prefix;
    const used = new Set(form.providers.map((item) => item.id));
    while (used.has(id)) {
      n += 1;
      id = `${prefix}-${n}`;
    }
    return id;
  }

  function triBool(value) {
    if (typeof value !== 'boolean') return '';
    return value ? 'true' : 'false';
  }

  function boolFromTri(value) {
    if (value === 'true') return true;
    if (value === 'false') return false;
    return undefined;
  }

  function stringValue(value, fallback = '') {
    return typeof value === 'string' ? value : fallback;
  }

  function numberValue(value, fallback = 0) {
    const n = Number(value);
    return Number.isFinite(n) ? n : fallback;
  }

  function optionalNumber(value) {
    const n = Number(value);
    return Number.isFinite(n) && n > 0 ? n : '';
  }

  function readBool(...values) {
    for (const value of values) {
      if (typeof value === 'boolean') return value;
    }
    return Boolean(values[values.length - 1]);
  }

  function arrayValue(value, fallback = []) {
    if (!Array.isArray(value)) return [...fallback];
    return value.map((item) => item && typeof item === 'object' ? item : String(item ?? ''));
  }

  function mapToPairs(value) {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return [];
    return Object.entries(value).map(([key, val]) => ({ key, value: String(val ?? '') }));
  }

  function cleanList(values = []) {
    return values.map((item) => String(item || '').trim()).filter(Boolean);
  }

  function csvList(value = '') {
    return String(value || '').split(',').map((item) => item.trim()).filter(Boolean);
  }

  function ensureObject(parent, key) {
    if (!parent[key] || typeof parent[key] !== 'object' || Array.isArray(parent[key])) {
      parent[key] = {};
    }
    return parent[key];
  }

  function writeString(target, key, value) {
    const text = String(value || '').trim();
    if (text) target[key] = text;
    else delete target[key];
  }

  function writeTriBool(target, key, value) {
    const bool = boolFromTri(value);
    if (bool === undefined) delete target[key];
    else target[key] = bool;
  }

  function writeOptionalNumber(target, key, value) {
    const n = Number(value);
    if (Number.isFinite(n) && n > 0) target[key] = n;
    else delete target[key];
  }

  function writeOptionalFloat(target, key, value) {
    const n = Number(value);
    if (Number.isFinite(n)) target[key] = n;
    else delete target[key];
  }

  function writeList(target, key, values) {
    const list = cleanList(values);
    if (list.length > 0) target[key] = list;
    else delete target[key];
  }

  function writeMap(target, key, pairs) {
    const out = {};
    for (const pair of pairs || []) {
      const k = String(pair.key || '').trim();
      if (!k) continue;
      out[k] = String(pair.value || '');
    }
    if (Object.keys(out).length > 0) target[key] = out;
    else delete target[key];
  }
</script>

<div class="card editor-card">
  <div class="card-head">
    <div>
      <h3>{$t(isProviderSettings ? 'settings.tabs.providers' : 'settings.tabs.app')}</h3>
      <span class="hint">{$t(isProviderSettings ? 'settings.providers.hint' : 'settings.app.hint')}</span>
    </div>
    <button type="button" class="primary" on:click={save}>{$t('common.save')}</button>
  </div>
  {#if parseError}
    <p class="error-text">{$t('settings.app.parseError', { error: parseError })}</p>
  {/if}
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t(isProviderSettings ? 'settings.providers.sections.defaults' : 'settings.app.sections.defaults')}</h3>
      <span class="hint">{$t(isProviderSettings ? 'settings.providers.defaultsHint' : 'settings.app.defaultsHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    {#if isProviderSettings}
      <label>
        <span>{$t('settings.app.defaultProvider')}</span>
        <select value={form.defaults.defaultProvider} on:change={(event) => selectDefaultProvider(event.currentTarget.value)}>
          <option value="">{$t('common.uninitialized')}</option>
          {#if defaultProviderMissing}
            <option value={form.defaults.defaultProvider}>{form.defaults.defaultProvider}</option>
          {/if}
          {#each form.providers as provider}
            <option value={provider.id}>{provider.id}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>{$t('settings.app.defaultModel')}</span>
        <select
          value={form.defaults.defaultModel}
          disabled={defaultModelOptions.length === 0 && !form.defaults.defaultModel}
          on:change={(event) => selectDefaultModel(event.currentTarget.value)}
        >
          <option value="">{$t('common.uninitialized')}</option>
          {#if defaultModelMissing}
            <option value={form.defaults.defaultModel}>{form.defaults.defaultModel}</option>
          {/if}
          {#each defaultModelOptions as model}
            <option value={model.id}>{model.name && model.name !== model.id ? `${model.id} - ${model.name}` : model.id}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>{$t('settings.app.thinking')}</span>
        <select bind:value={form.defaults.defaultThinkingLevel}>
          <option value="low">low</option>
          <option value="medium">medium</option>
          <option value="high">high</option>
        </select>
      </label>
      <label><span>{$t('settings.app.maxContextTokens')}</span><input type="number" min="0" bind:value={form.defaults.maxContextTokens} /></label>
      <label><span>{$t('settings.app.maxOutputTokens')}</span><input type="number" min="0" bind:value={form.defaults.maxOutputTokens} /></label>
    {:else}
      <label>
        <span>{$t('settings.app.defaultMode')}</span>
        <select bind:value={form.defaults.defaultMode}>
          <option value="plan">plan</option>
          <option value="agent">agent</option>
          <option value="yolo">yolo</option>
        </select>
      </label>
      <label><span>{$t('settings.app.skillsDir')}</span><input bind:value={form.defaults.skillsDir} /></label>
      <label><span>{$t('settings.app.sessionDir')}</span><input bind:value={form.defaults.sessionDir} /></label>
      <label><span>{$t('settings.app.shellPath')}</span><input bind:value={form.defaults.shellPath} /></label>
      <label><span>{$t('settings.app.shellPrefix')}</span><input bind:value={form.defaults.shellCommandPrefix} /></label>
      <label>
        <span>{$t('settings.app.enablePlanTool')}</span>
        <select bind:value={form.defaults.enablePlanTool}>
          <option value="">{$t('common.uninitialized')}</option>
          <option value="true">{$t('common.enabled')}</option>
          <option value="false">{$t('common.disabled')}</option>
        </select>
      </label>
      <label>
        <span>{$t('settings.app.updateCheck')}</span>
        <select bind:value={form.defaults.updateCheck}>
          <option value="">{$t('common.uninitialized')}</option>
          <option value="true">{$t('common.enabled')}</option>
          <option value="false">{$t('common.disabled')}</option>
        </select>
      </label>
    {/if}
  </div>
</div>

{#if !isProviderSettings}
<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.app.sections.context')}</h3>
      <span class="hint">{$t('settings.app.contextHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label class="checkbox"><input type="checkbox" bind:checked={form.contextFiles.enabled} /> {$t('settings.app.contextFiles')}</label>
    <label class="checkbox"><input type="checkbox" bind:checked={form.compaction.enabled} /> {$t('settings.app.compaction')}</label>
    <label><span>{$t('settings.app.reserveTokens')}</span><input type="number" min="0" bind:value={form.compaction.reserveTokens} /></label>
    <label><span>{$t('settings.app.keepRecentTokens')}</span><input type="number" min="0" bind:value={form.compaction.keepRecentTokens} /></label>
    <label><span>{$t('settings.app.tokenizer')}</span><input bind:value={form.compaction.tokenizer} /></label>
    <label><span>{$t('settings.app.tokenizerModel')}</span><input bind:value={form.compaction.tokenizerModel} /></label>
    <label class="checkbox"><input type="checkbox" bind:checked={form.compaction.idleCompressionEnabled} /> {$t('settings.app.idleCompression')}</label>
    <label><span>{$t('settings.app.idleTimeout')}</span><input type="number" min="0" bind:value={form.compaction.idleTimeoutSeconds} /></label>
    <label><span>{$t('settings.app.idleMinTokens')}</span><input type="number" min="0" bind:value={form.compaction.idleMinTokensForCompress} /></label>
    <label class="full"><span>{$t('settings.app.compactionTemplate')}</span><textarea bind:value={form.compaction.template}></textarea></label>
    <div class="list-editor full">
      <div class="list-head">
        <span>{$t('settings.app.extraFiles')}</span>
        <button type="button" class="ghost sm" on:click={() => addListItem(form.contextFiles.extraFiles)}>{$t('common.add')}</button>
      </div>
      {#each form.contextFiles.extraFiles as item, i (i)}
        <div class="inline-row">
          <input bind:value={form.contextFiles.extraFiles[i]} />
          <button type="button" class="ghost sm" on:click={() => removeListItem(form.contextFiles.extraFiles, i)}>{$t('common.remove')}</button>
        </div>
      {/each}
    </div>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.app.sections.tools')}</h3>
      <span class="hint">{$t('settings.app.toolsHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label>
      <span>{$t('settings.app.webSearch')}</span>
      <select bind:value={form.webSearch.enabled}>
        <option value="">{$t('common.uninitialized')}</option>
        <option value="true">{$t('common.enabled')}</option>
        <option value="false">{$t('common.disabled')}</option>
      </select>
    </label>
    <label><span>{$t('settings.app.webSearchProvider')}</span><input bind:value={form.webSearch.provider} /></label>
    <label><span>{$t('settings.app.webSearchType')}</span><input bind:value={form.webSearch.providerType} /></label>
    <label><span>{$t('settings.app.webSearchModel')}</span><input bind:value={form.webSearch.model} /></label>
    <label class="checkbox"><input type="checkbox" bind:checked={form.retry.enabled} /> {$t('settings.app.retry')}</label>
    <label><span>{$t('settings.app.maxRetries')}</span><input type="number" min="0" bind:value={form.retry.maxRetries} /></label>
    <label><span>{$t('settings.app.baseDelay')}</span><input type="number" min="0" bind:value={form.retry.baseDelayMs} /></label>
    <label class="checkbox"><input type="checkbox" bind:checked={form.statusLine.enabled} /> {$t('settings.app.statusLine')}</label>
    <label><span>{$t('settings.app.statusLineType')}</span><input bind:value={form.statusLine.type} /></label>
    <label><span>{$t('settings.app.statusLineCommand')}</span><input bind:value={form.statusLine.command} /></label>
    <label><span>{$t('settings.app.statusLineTimeout')}</span><input type="number" min="0" bind:value={form.statusLine.timeoutMs} /></label>
    <label><span>{$t('settings.app.statusLineFallback')}</span><input bind:value={form.statusLine.fallback} /></label>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.app.sections.safety')}</h3>
      <span class="hint">{$t('settings.app.safetyHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label class="checkbox"><input type="checkbox" bind:checked={form.sandbox.enabled} /> {$t('settings.app.sandbox')}</label>
    <label class="checkbox"><input type="checkbox" bind:checked={form.sandbox.allowNetwork} /> {$t('settings.app.allowNetwork')}</label>
    <label><span>{$t('settings.app.sandboxLevel')}</span><input bind:value={form.sandbox.level} /></label>
    <label><span>{$t('settings.app.bwrapPath')}</span><input bind:value={form.sandbox.bwrapPath} /></label>
    <label><span>{$t('settings.app.tmpSize')}</span><input bind:value={form.sandbox.tmpSize} /></label>
    <label>
      <span>{$t('settings.app.confirmBeforeWrite')}</span>
      <select bind:value={form.approval.confirmBeforeWrite}>
        <option value="">{$t('common.uninitialized')}</option>
        <option value="true">{$t('common.enabled')}</option>
        <option value="false">{$t('common.disabled')}</option>
      </select>
    </label>
  </div>
  <div class="form-grid two-lists">
    <ListEditor title={$t('settings.app.allowedRead')} list={form.sandbox.allowedRead} onAdd={() => addListItem(form.sandbox.allowedRead)} onRemove={(i) => removeListItem(form.sandbox.allowedRead, i)} />
    <ListEditor title={$t('settings.app.allowedWrite')} list={form.sandbox.allowedWrite} onAdd={() => addListItem(form.sandbox.allowedWrite)} onRemove={(i) => removeListItem(form.sandbox.allowedWrite, i)} />
    <ListEditor title={$t('settings.app.deniedPaths')} list={form.sandbox.deniedPaths} onAdd={() => addListItem(form.sandbox.deniedPaths)} onRemove={(i) => removeListItem(form.sandbox.deniedPaths, i)} />
    <ListEditor title={$t('settings.app.passEnv')} list={form.sandbox.passEnv} onAdd={() => addListItem(form.sandbox.passEnv)} onRemove={(i) => removeListItem(form.sandbox.passEnv, i)} />
    <ListEditor title={$t('settings.app.bashWhitelist')} list={form.approval.bashWhitelist} onAdd={() => addListItem(form.approval.bashWhitelist)} onRemove={(i) => removeListItem(form.approval.bashWhitelist, i)} />
    <ListEditor title={$t('settings.app.bashBlacklist')} list={form.approval.bashBlacklist} onAdd={() => addListItem(form.approval.bashBlacklist)} onRemove={(i) => removeListItem(form.approval.bashBlacklist, i)} />
  </div>
</div>
{/if}

{#if isProviderSettings}
<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.app.sections.providers')}</h3>
      <span class="hint">{$t('settings.app.providersHint', { count: form.providers.length })}</span>
    </div>
    <button type="button" class="ghost" on:click={addProvider}>{$t('common.add')}</button>
  </div>
  {#if form.providers.length === 0}
    <p class="empty">{$t('settings.app.noProviders')}</p>
  {:else}
    <div class="provider-editor">
      <aside class="provider-list">
        {#each form.providers as provider (provider.id)}
          <button type="button" class:active={provider.id === selectedProviderID} on:click={() => (selectedProviderID = provider.id)}>
            <strong>{provider.id || $t('settings.app.unnamedProvider')}</strong>
            <span>{provider.models.length} models</span>
          </button>
        {/each}
      </aside>
      {#if currentProvider}
        <section class="provider-detail">
          <div class="form-grid">
            <label><span>{$t('settings.app.providerID')}</span><input value={currentProvider.id} on:input={(event) => renameProvider(currentProvider, event.currentTarget.value)} /></label>
            <label><span>{$t('settings.app.providerVendor')}</span><input bind:value={currentProvider.vendor} /></label>
            <label><span>{$t('settings.app.providerAPI')}</span><input bind:value={currentProvider.api} /></label>
            <label><span>{$t('settings.app.providerThinkingFormat')}</span><input bind:value={currentProvider.thinkingFormat} /></label>
            <label class="full"><span>{$t('settings.app.providerBaseURL')}</span><input bind:value={currentProvider.baseUrl} /></label>
            <label class="full"><span>{$t('settings.app.providerAPIKey')}</span><input bind:value={currentProvider.apiKey} /></label>
            <label><span>{$t('settings.app.httpProxy')}</span><input bind:value={currentProvider.httpProxy} /></label>
            <label class="checkbox"><input type="checkbox" bind:checked={currentProvider.forceHTTP11} /> {$t('settings.app.forceHTTP11')}</label>
            <label>
              <span>{$t('settings.app.cacheControl')}</span>
              <select bind:value={currentProvider.cacheControl}>
                <option value="">{$t('common.uninitialized')}</option>
                <option value="true">{$t('common.enabled')}</option>
                <option value="false">{$t('common.disabled')}</option>
              </select>
            </label>
            <label><span>{$t('settings.app.reasoningSummary')}</span><input bind:value={currentProvider.responses.reasoningSummary} /></label>
            <label><span>{$t('settings.app.promptCacheKey')}</span><input bind:value={currentProvider.responses.promptCacheKey} /></label>
            <label><span>{$t('settings.app.promptCacheRetention')}</span><input bind:value={currentProvider.responses.promptCacheRetention} /></label>
          </div>
          <div class="provider-actions">
            <button type="button" class="ghost sm" on:click={() => addHeader(currentProvider)}>{$t('settings.app.addHeader')}</button>
            <button type="button" class="ghost sm" on:click={() => addModel(currentProvider)}>{$t('settings.app.addModel')}</button>
            <button type="button" class="ghost danger sm" on:click={() => removeProvider(currentProvider)}>{$t('common.remove')}</button>
          </div>
          {#if currentProvider.headers.length > 0}
            <div class="model-list">
              <div class="list-head"><span>{$t('settings.app.headers')}</span></div>
              {#each currentProvider.headers as header, i (i)}
                <div class="provider-header-row">
                  <input bind:value={header.key} placeholder="Header" />
                  <input bind:value={header.value} placeholder="Value" />
                  <button type="button" class="ghost sm" on:click={() => removeHeader(currentProvider, i)}>{$t('common.remove')}</button>
                </div>
              {/each}
            </div>
          {/if}
          <div class="model-list">
            <div class="list-head">
              <span>{$t('settings.app.models')}</span>
              <button type="button" class="ghost sm" on:click={() => addModel(currentProvider)}>{$t('common.add')}</button>
            </div>
            <div class="model-row model-row-head">
              <span>{$t('settings.app.modelID')}</span>
              <span>{$t('settings.app.modelName')}</span>
              <span>{$t('settings.app.modelContext')}</span>
              <span>{$t('settings.app.modelMaxTokens')}</span>
              <span>{$t('settings.app.modelTemperature')}</span>
              <span>{$t('settings.app.modelTopP')}</span>
              <span>{$t('settings.app.modelInput')}</span>
              <span>{$t('settings.app.modelReasoning')}</span>
              <span>{$t('settings.app.modelActions')}</span>
            </div>
            {#each currentProvider.models as model, i (i)}
              <div class="model-row">
                <input bind:value={model.id} placeholder="model-id" />
                <input bind:value={model.name} placeholder="Display name" />
                <input type="number" min="0" bind:value={model.contextWindow} placeholder="context" />
                <input type="number" min="0" bind:value={model.maxTokens} placeholder="max" />
                <input type="number" step="0.1" bind:value={model.temperature} placeholder="temp" />
                <input type="number" step="0.1" bind:value={model.topP} placeholder="top_p" />
                <input bind:value={model.input} placeholder="text, image" />
                <label class="model-reasoning-toggle"><input type="checkbox" bind:checked={model.reasoning} /> {$t('settings.app.modelReasoning')}</label>
                <button type="button" class="ghost sm" on:click={() => removeModel(currentProvider, i)}>{$t('common.remove')}</button>
              </div>
            {/each}
          </div>
        </section>
      {/if}
    </div>
  {/if}
</div>
{/if}

<details class="card editor-card advanced-json">
  <summary>
    <div>
      <h3>{$t('settings.app.advancedJson')}</h3>
      <span class="hint">{$t('settings.app.advancedJsonHint')}</span>
    </div>
  </summary>
  <textarea class="code" bind:value={jsonDraft} spellcheck="false"></textarea>
</details>
