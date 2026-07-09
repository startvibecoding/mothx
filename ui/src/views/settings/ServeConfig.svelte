<script>
  import { serveConfig, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';
  import { t } from '../../lib/preferences.js';

  let form = defaultForm();
  let jsonDraft = '';
  let parseError = '';
  let advancedOpen = false;
  let lastRaw = '';

  $: syncFromStore($serveConfig);

  function defaultForm() {
    return {
      features: {
        webUI: true,
        openAIAPI: true,
        websocket: false,
        cron: true,
        memory: true,
        multiAgent: false,
        wechat: false,
        feishu: false
      },
      webUI: { enabled: true, dir: 'ui/dist' },
      api: {
        listen: ':8080',
        provider: '',
        model: '',
        defaultMode: 'yolo',
        defaultThinkingLevel: 'medium',
        systemPromptMode: 'append',
        requestTimeoutSeconds: 1800,
        maxConcurrentRequests: '',
        logLevel: 'info',
        enableWebSearch: false,
        enableBrowser: false,
        enableA2AMaster: false,
        enableDelegate: false,
        enableWorkflows: false,
        enableSubAgents: false,
        auth: { enabled: false, tokens: [] },
        sandbox: { enabled: false, level: '' },
        session: { idleTimeoutSeconds: 1800, maxSessions: '' },
        cors: { enabled: false, allowOrigins: ['*'] },
        toolVisibility: { mode: 'content', detail: 'collapsed' }
      },
      cron: { enabled: true, interval: 30 },
      memory: { enabled: true, path: '' },
      security: { smartApprovals: true },
      agent: {
        maxTurns: 90,
        budgetPressure: true,
        contextPressure: true,
        budgetPressureThreshold: 0.2,
        contextPressureThreshold: 0.55
      },
      hooks: { preToolCall: '', postToolCall: '' },
      channels: {
        wechat: { enabled: false, credPath: '', workDir: '', allowedUsers: [], autoTyping: true },
        feishu: { enabled: false, appID: '', appSecret: '', workDir: '', allowedUsers: [] }
      },
      lobsterMode: false
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
    } catch (err) {
      parseError = err instanceof Error ? err.message : String(err);
      form = defaultForm();
    }
  }

  function formFromConfig(cfg = {}) {
    const base = defaultForm();
    const api = cfg.api || {};
    const features = cfg.features || {};
    const webUI = cfg.webUI || {};
    const cron = cfg.cron || {};
    const memory = cfg.memory || {};
    const security = cfg.security || {};
    const agent = cfg.agent || {};
    const hooks = cfg.hooks || {};
    const channels = cfg.channels || {};
    const wechat = channels.wechat || {};
    const feishu = channels.feishu || {};

    return {
      features: {
        webUI: readBool(features.webUI, webUI.enabled, base.features.webUI),
        openAIAPI: readBool(features.openAIAPI, base.features.openAIAPI),
        websocket: readBool(features.websocket, base.features.websocket),
        cron: readBool(features.cron, cron.enabled, base.features.cron),
        memory: readBool(features.memory, memory.enabled, base.features.memory),
        multiAgent: readBool(features.multiAgent, api.enableSubAgents, base.features.multiAgent),
        wechat: readBool(features.wechat, wechat.enabled, base.features.wechat),
        feishu: readBool(features.feishu, feishu.enabled, base.features.feishu)
      },
      webUI: {
        enabled: readBool(webUI.enabled, features.webUI, base.webUI.enabled),
        dir: stringValue(webUI.dir, base.webUI.dir)
      },
      api: {
        listen: stringValue(api.listen, base.api.listen),
        provider: stringValue(api.provider, ''),
        model: stringValue(api.model, ''),
        defaultMode: stringValue(api.defaultMode, base.api.defaultMode),
        defaultThinkingLevel: stringValue(api.defaultThinkingLevel, base.api.defaultThinkingLevel),
        systemPromptMode: stringValue(api.systemPromptMode, base.api.systemPromptMode),
        requestTimeoutSeconds: numberValue(api.requestTimeoutSeconds, base.api.requestTimeoutSeconds),
        maxConcurrentRequests: optionalNumber(api.maxConcurrentRequests),
        logLevel: stringValue(api.logLevel, base.api.logLevel),
        enableWebSearch: readBool(api.enableWebSearch, false),
        enableBrowser: readBool(api.enableBrowser, false),
        enableA2AMaster: readBool(api.enableA2AMaster, false),
        enableDelegate: readBool(api.enableDelegate, false),
        enableWorkflows: readBool(api.enableWorkflows, false),
        enableSubAgents: readBool(api.enableSubAgents, features.multiAgent, false),
        auth: {
          enabled: readBool(api.auth?.enabled, false),
          tokens: arrayValue(api.auth?.tokens)
        },
        sandbox: {
          enabled: readBool(api.sandbox?.enabled, false),
          level: stringValue(api.sandbox?.level, '')
        },
        session: {
          idleTimeoutSeconds: numberValue(api.session?.idleTimeoutSeconds, base.api.session.idleTimeoutSeconds),
          maxSessions: optionalNumber(api.session?.maxSessions)
        },
        cors: {
          enabled: readBool(api.cors?.enabled, false),
          allowOrigins: arrayValue(api.cors?.allowOrigins, ['*'])
        },
        toolVisibility: {
          mode: stringValue(api.toolVisibility?.mode, base.api.toolVisibility.mode),
          detail: stringValue(api.toolVisibility?.detail, base.api.toolVisibility.detail)
        }
      },
      cron: {
        enabled: readBool(cron.enabled, features.cron, base.cron.enabled),
        interval: numberValue(cron.interval, base.cron.interval)
      },
      memory: {
        enabled: readBool(memory.enabled, features.memory, base.memory.enabled),
        path: stringValue(memory.path, '')
      },
      security: {
        smartApprovals: readBool(security.smart_approvals, base.security.smartApprovals)
      },
      agent: {
        maxTurns: numberValue(agent.max_turns, base.agent.maxTurns),
        budgetPressure: readBool(agent.budget_pressure, base.agent.budgetPressure),
        contextPressure: readBool(agent.context_pressure, base.agent.contextPressure),
        budgetPressureThreshold: numberValue(agent.budget_pressure_threshold, base.agent.budgetPressureThreshold),
        contextPressureThreshold: numberValue(agent.context_pressure_threshold, base.agent.contextPressureThreshold)
      },
      hooks: {
        preToolCall: stringValue(hooks.pre_tool_call, ''),
        postToolCall: stringValue(hooks.post_tool_call, '')
      },
      channels: {
        wechat: {
          enabled: readBool(wechat.enabled, features.wechat, false),
          credPath: stringValue(wechat.cred_path, ''),
          workDir: stringValue(wechat.work_dir, ''),
          allowedUsers: arrayValue(wechat.allowed_users),
          autoTyping: readBool(wechat.auto_typing, true)
        },
        feishu: {
          enabled: readBool(feishu.enabled, features.feishu, false),
          appID: stringValue(feishu.app_id, ''),
          appSecret: stringValue(feishu.app_secret, ''),
          workDir: stringValue(feishu.work_dir, ''),
          allowedUsers: arrayValue(feishu.allowed_users)
        }
      },
      lobsterMode: readBool(cfg.lobsterMode, false)
    };
  }

  function readBool(...values) {
    for (const value of values) {
      if (typeof value === 'boolean') return value;
    }
    return Boolean(values[values.length - 1]);
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

  function arrayValue(value, fallback = []) {
    if (!Array.isArray(value)) return [...fallback];
    return value.map((item) => String(item ?? ''));
  }

  function ensureObject(parent, key) {
    if (!parent[key] || typeof parent[key] !== 'object' || Array.isArray(parent[key])) {
      parent[key] = {};
    }
    return parent[key];
  }

  function cleanList(values = []) {
    return values.map((item) => String(item || '').trim()).filter(Boolean);
  }

  function writeOptionalNumber(target, key, value) {
    const n = Number(value);
    if (Number.isFinite(n) && n > 0) target[key] = n;
    else delete target[key];
  }

  function buildConfigForSave() {
    const cfg = JSON.parse(jsonDraft || '{}');
    const features = ensureObject(cfg, 'features');
    const api = ensureObject(cfg, 'api');
    const webUI = ensureObject(cfg, 'webUI');
    const auth = ensureObject(api, 'auth');
    const sandbox = ensureObject(api, 'sandbox');
    const session = ensureObject(api, 'session');
    const cors = ensureObject(api, 'cors');
    const toolVisibility = ensureObject(api, 'toolVisibility');
    const cron = ensureObject(cfg, 'cron');
    const memory = ensureObject(cfg, 'memory');
    const security = ensureObject(cfg, 'security');
    const agent = ensureObject(cfg, 'agent');
    const hooks = ensureObject(cfg, 'hooks');
    const channels = ensureObject(cfg, 'channels');
    const wechat = ensureObject(channels, 'wechat');
    const feishu = ensureObject(channels, 'feishu');

    features.webUI = Boolean(form.features.webUI);
    features.openAIAPI = Boolean(form.features.openAIAPI);
    features.websocket = Boolean(form.features.websocket);
    features.cron = Boolean(form.features.cron);
    features.memory = Boolean(form.features.memory);
    features.multiAgent = Boolean(form.features.multiAgent);
    features.wechat = Boolean(form.features.wechat);
    features.feishu = Boolean(form.features.feishu);

    webUI.enabled = Boolean(form.features.webUI);
    webUI.dir = form.webUI.dir.trim() || 'ui/dist';

    api.listen = form.api.listen.trim() || ':8080';
    api.defaultMode = form.api.defaultMode || 'yolo';
    api.defaultThinkingLevel = form.api.defaultThinkingLevel || 'medium';
    api.systemPromptMode = form.api.systemPromptMode || 'append';
    api.requestTimeoutSeconds = numberValue(form.api.requestTimeoutSeconds, 1800);
    writeOptionalNumber(api, 'maxConcurrentRequests', form.api.maxConcurrentRequests);
    api.logLevel = form.api.logLevel || 'info';
    api.enableWebSearch = Boolean(form.api.enableWebSearch);
    api.enableBrowser = Boolean(form.api.enableBrowser);
    api.enableA2AMaster = Boolean(form.api.enableA2AMaster);
    api.enableDelegate = Boolean(form.api.enableDelegate);
    api.enableWorkflows = Boolean(form.api.enableWorkflows);
    api.enableSubAgents = Boolean(form.features.multiAgent || form.api.enableSubAgents);
    if (form.api.provider.trim()) api.provider = form.api.provider.trim();
    else delete api.provider;
    if (form.api.model.trim()) api.model = form.api.model.trim();
    else delete api.model;

    auth.enabled = Boolean(form.api.auth.enabled);
    auth.tokens = cleanList(form.api.auth.tokens);
    sandbox.enabled = Boolean(form.api.sandbox.enabled);
    if (form.api.sandbox.level) sandbox.level = form.api.sandbox.level;
    else delete sandbox.level;
    session.idleTimeoutSeconds = numberValue(form.api.session.idleTimeoutSeconds, 1800);
    writeOptionalNumber(session, 'maxSessions', form.api.session.maxSessions);
    cors.enabled = Boolean(form.api.cors.enabled);
    cors.allowOrigins = cleanList(form.api.cors.allowOrigins);
    toolVisibility.mode = form.api.toolVisibility.mode || 'content';
    toolVisibility.detail = form.api.toolVisibility.detail || 'collapsed';

    cron.enabled = Boolean(form.features.cron);
    cron.interval = numberValue(form.cron.interval, 30);
    memory.enabled = Boolean(form.features.memory);
    if (form.memory.path.trim()) memory.path = form.memory.path.trim();
    else delete memory.path;

    security.smart_approvals = Boolean(form.security.smartApprovals);

    agent.max_turns = numberValue(form.agent.maxTurns, 90);
    agent.budget_pressure = Boolean(form.agent.budgetPressure);
    agent.context_pressure = Boolean(form.agent.contextPressure);
    agent.budget_pressure_threshold = numberValue(form.agent.budgetPressureThreshold, 0.2);
    agent.context_pressure_threshold = numberValue(form.agent.contextPressureThreshold, 0.55);

    hooks.pre_tool_call = form.hooks.preToolCall.trim();
    hooks.post_tool_call = form.hooks.postToolCall.trim();

    wechat.enabled = Boolean(form.features.wechat);
    wechat.cred_path = form.channels.wechat.credPath.trim();
    wechat.work_dir = form.channels.wechat.workDir.trim();
    wechat.allowed_users = cleanList(form.channels.wechat.allowedUsers);
    wechat.auto_typing = Boolean(form.channels.wechat.autoTyping);

    feishu.enabled = Boolean(form.features.feishu);
    feishu.app_id = form.channels.feishu.appID.trim();
    feishu.app_secret = form.channels.feishu.appSecret.trim();
    feishu.work_dir = form.channels.feishu.workDir.trim();
    feishu.allowed_users = cleanList(form.channels.feishu.allowedUsers);

    cfg.lobsterMode = Boolean(form.lobsterMode);
    return cfg;
  }

  async function save() {
    clearBanners();
    try {
      const next = buildConfigForSave();
      const saved = await putJSON('/api/serve/config', next);
      const text = JSON.stringify(saved, null, 2);
      lastRaw = text;
      jsonDraft = text;
      serveConfig.set(text);
      form = formFromConfig(saved);
      parseError = '';
      setNotice($t('settings.serve.saved'));
    } catch (err) {
      setError(err);
    }
  }

  function addList(path) {
    const list = listForPath(path);
    list.push('');
    form = form;
  }

  function removeList(path, index) {
    const list = listForPath(path);
    list.splice(index, 1);
    form = form;
  }

  function listForPath(path) {
    switch (path) {
      case 'tokens': return form.api.auth.tokens;
      case 'origins': return form.api.cors.allowOrigins;
      case 'wechatUsers': return form.channels.wechat.allowedUsers;
      case 'feishuUsers': return form.channels.feishu.allowedUsers;
      default: return [];
    }
  }
</script>

{#if parseError}
  <div class="card">
    <div class="form-body">
      <p class="error-text">{parseError}</p>
    </div>
  </div>
{/if}

<div class="page-toolbar embedded">
  <button type="button" class="primary" on:click={save}>{$t('common.save')}</button>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.serve.sections.runtime')}</h3>
      <span class="hint">{$t('settings.serve.runtimeHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label>
      <span>{$t('settings.serve.listen')}</span>
      <input bind:value={form.api.listen} placeholder=":8080" />
    </label>
    <label>
      <span>{$t('settings.serve.webuiDir')}</span>
      <input bind:value={form.webUI.dir} placeholder="ui/dist" />
    </label>
    <label>
      <span>{$t('settings.serve.provider')}</span>
      <input bind:value={form.api.provider} placeholder="default" />
    </label>
    <label>
      <span>{$t('settings.serve.model')}</span>
      <input bind:value={form.api.model} placeholder="default" />
    </label>
    <label>
      <span>{$t('settings.serve.defaultMode')}</span>
      <select bind:value={form.api.defaultMode}>
        <option value="plan">plan</option>
        <option value="agent">agent</option>
        <option value="yolo">yolo</option>
      </select>
    </label>
    <label>
      <span>{$t('settings.serve.thinking')}</span>
      <select bind:value={form.api.defaultThinkingLevel}>
        <option value="low">low</option>
        <option value="medium">medium</option>
        <option value="high">high</option>
      </select>
    </label>
    <label>
      <span>{$t('settings.serve.timeout')}</span>
      <input type="number" min="1" bind:value={form.api.requestTimeoutSeconds} />
    </label>
    <label>
      <span>{$t('settings.serve.maxConcurrent')}</span>
      <input type="number" min="0" bind:value={form.api.maxConcurrentRequests} placeholder="unlimited" />
    </label>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.serve.sections.features')}</h3>
      <span class="hint">{$t('settings.serve.featuresHint')}</span>
    </div>
  </div>
  <div class="form-grid toggle-grid">
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.webUI} />
      <span>Web UI</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.openAIAPI} />
      <span>OpenAI API</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.websocket} />
      <span>WebSocket</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.cron} />
      <span>Cron</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.memory} />
      <span>Memory</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.multiAgent} />
      <span>Multi-agent</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.api.enableDelegate} />
      <span>Delegate</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.api.enableWebSearch} />
      <span>Web Search</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.api.enableBrowser} />
      <span>Browser</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.api.enableA2AMaster} />
      <span>A2A Master</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.api.enableWorkflows} />
      <span>Workflows</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.lobsterMode} />
      <span>Lobster mode</span>
    </label>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.serve.sections.output')}</h3>
      <span class="hint">{$t('settings.serve.outputHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label>
      <span>{$t('settings.serve.toolMode')}</span>
      <select bind:value={form.api.toolVisibility.mode}>
        <option value="content">content</option>
        <option value="sse_event">sse_event</option>
        <option value="none">none</option>
      </select>
    </label>
    <label>
      <span>{$t('settings.serve.toolDetail')}</span>
      <select bind:value={form.api.toolVisibility.detail}>
        <option value="collapsed">collapsed</option>
        <option value="expanded">expanded</option>
      </select>
    </label>
    <label>
      <span>{$t('settings.serve.systemPromptMode')}</span>
      <select bind:value={form.api.systemPromptMode}>
        <option value="append">append</option>
        <option value="ignore">ignore</option>
      </select>
    </label>
    <label>
      <span>{$t('settings.serve.logLevel')}</span>
      <select bind:value={form.api.logLevel}>
        <option value="debug">debug</option>
        <option value="info">info</option>
        <option value="warn">warn</option>
        <option value="error">error</option>
      </select>
    </label>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.serve.sections.security')}</h3>
      <span class="hint">{$t('settings.serve.securityHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.api.auth.enabled} />
      <span>{$t('settings.serve.auth')}</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.api.sandbox.enabled} />
      <span>{$t('settings.serve.sandbox')}</span>
    </label>
    <label>
      <span>{$t('settings.serve.sandboxLevel')}</span>
      <select bind:value={form.api.sandbox.level}>
        <option value="">auto</option>
        <option value="none">none</option>
        <option value="standard">standard</option>
        <option value="strict">strict</option>
      </select>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.security.smartApprovals} />
      <span>{$t('settings.serve.smartApprovals')}</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.api.cors.enabled} />
      <span>CORS</span>
    </label>
  </div>
  <div class="form-body">
    <div class="list-editor">
      <div class="list-head">
        <span>{$t('settings.serve.tokens')}</span>
        <button type="button" class="sm" on:click={() => addList('tokens')}>+ {$t('common.add')}</button>
      </div>
      {#each form.api.auth.tokens as token, i (i)}
        <div class="dir-row">
          <input bind:value={form.api.auth.tokens[i]} class="dir-input" placeholder="sk-..." />
          <button type="button" class="ghost danger" on:click={() => removeList('tokens', i)}>×</button>
        </div>
      {/each}
    </div>
    <div class="list-editor">
      <div class="list-head">
        <span>{$t('settings.serve.corsOrigins')}</span>
        <button type="button" class="sm" on:click={() => addList('origins')}>+ {$t('common.add')}</button>
      </div>
      {#each form.api.cors.allowOrigins as origin, i (i)}
        <div class="dir-row">
          <input bind:value={form.api.cors.allowOrigins[i]} class="dir-input" placeholder="*" />
          <button type="button" class="ghost danger" on:click={() => removeList('origins', i)}>×</button>
        </div>
      {/each}
    </div>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.serve.sections.sessions')}</h3>
      <span class="hint">{$t('settings.serve.sessionsHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label>
      <span>{$t('settings.serve.idleTimeout')}</span>
      <input type="number" min="1" bind:value={form.api.session.idleTimeoutSeconds} />
    </label>
    <label>
      <span>{$t('settings.serve.maxSessions')}</span>
      <input type="number" min="0" bind:value={form.api.session.maxSessions} placeholder="unlimited" />
    </label>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.serve.sections.automation')}</h3>
      <span class="hint">{$t('settings.serve.automationHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.cron} />
      <span>Cron</span>
    </label>
    <label>
      <span>{$t('settings.serve.cronInterval')}</span>
      <input type="number" min="1" bind:value={form.cron.interval} />
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.memory} />
      <span>Memory</span>
    </label>
    <label>
      <span>{$t('settings.serve.memoryPath')}</span>
      <input bind:value={form.memory.path} placeholder=".mothx/memory.md" />
    </label>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.serve.sections.agent')}</h3>
      <span class="hint">{$t('settings.serve.agentHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label>
      <span>{$t('settings.serve.maxTurns')}</span>
      <input type="number" min="1" bind:value={form.agent.maxTurns} />
    </label>
    <label>
      <span>{$t('settings.serve.budgetThreshold')}</span>
      <input type="number" min="0" max="1" step="0.01" bind:value={form.agent.budgetPressureThreshold} />
    </label>
    <label>
      <span>{$t('settings.serve.contextThreshold')}</span>
      <input type="number" min="0" max="1" step="0.01" bind:value={form.agent.contextPressureThreshold} />
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.agent.budgetPressure} />
      <span>{$t('settings.serve.budgetPressure')}</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.agent.contextPressure} />
      <span>{$t('settings.serve.contextPressure')}</span>
    </label>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.serve.sections.channels')}</h3>
      <span class="hint">{$t('settings.serve.channelsHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.wechat} />
      <span>WeChat</span>
    </label>
    <label>
      <span>{$t('settings.serve.wechatCred')}</span>
      <input bind:value={form.channels.wechat.credPath} placeholder="wechat-cred.json" />
    </label>
    <label>
      <span>{$t('settings.serve.wechatWorkDir')}</span>
      <input bind:value={form.channels.wechat.workDir} placeholder="/home/user/project" />
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.channels.wechat.autoTyping} />
      <span>{$t('settings.serve.autoTyping')}</span>
    </label>
    <label class="checkbox">
      <input type="checkbox" bind:checked={form.features.feishu} />
      <span>Feishu</span>
    </label>
    <label>
      <span>{$t('settings.serve.feishuAppID')}</span>
      <input bind:value={form.channels.feishu.appID} />
    </label>
    <label>
      <span>{$t('settings.serve.feishuAppSecret')}</span>
      <input type="password" bind:value={form.channels.feishu.appSecret} />
    </label>
    <label>
      <span>{$t('settings.serve.feishuWorkDir')}</span>
      <input bind:value={form.channels.feishu.workDir} placeholder="/home/user/project" />
    </label>
  </div>
  <div class="form-body two-lists">
    <div class="list-editor">
      <div class="list-head">
        <span>{$t('settings.serve.wechatUsers')}</span>
        <button type="button" class="sm" on:click={() => addList('wechatUsers')}>+ {$t('common.add')}</button>
      </div>
      {#each form.channels.wechat.allowedUsers as user, i (i)}
        <div class="dir-row">
          <input bind:value={form.channels.wechat.allowedUsers[i]} class="dir-input" />
          <button type="button" class="ghost danger" on:click={() => removeList('wechatUsers', i)}>×</button>
        </div>
      {/each}
    </div>
    <div class="list-editor">
      <div class="list-head">
        <span>{$t('settings.serve.feishuUsers')}</span>
        <button type="button" class="sm" on:click={() => addList('feishuUsers')}>+ {$t('common.add')}</button>
      </div>
      {#each form.channels.feishu.allowedUsers as user, i (i)}
        <div class="dir-row">
          <input bind:value={form.channels.feishu.allowedUsers[i]} class="dir-input" />
          <button type="button" class="ghost danger" on:click={() => removeList('feishuUsers', i)}>×</button>
        </div>
      {/each}
    </div>
  </div>
</div>

<div class="card">
  <div class="card-head">
    <div>
      <h3>{$t('settings.serve.sections.hooks')}</h3>
      <span class="hint">{$t('settings.serve.hooksHint')}</span>
    </div>
  </div>
  <div class="form-grid">
    <label>
      <span>{$t('settings.serve.preToolCall')}</span>
      <input bind:value={form.hooks.preToolCall} placeholder="/path/to/pre-hook.sh" />
    </label>
    <label>
      <span>{$t('settings.serve.postToolCall')}</span>
      <input bind:value={form.hooks.postToolCall} placeholder="/path/to/post-hook.sh" />
    </label>
  </div>
</div>

<details class="card editor-card advanced-json" bind:open={advancedOpen}>
  <summary>
    <div>
      <h3>{$t('settings.serve.advancedJson')}</h3>
      <span class="hint">{$t('settings.serve.advancedJsonHint')}</span>
    </div>
  </summary>
  <textarea class="code" bind:value={jsonDraft} spellcheck="false"></textarea>
</details>
