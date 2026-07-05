<script>
  import { onDestroy } from 'svelte';
  import { markdownToHTML } from './markdown.js';

  let health = null;
  let status = null;
  let channels = [];
  let sessions = [];
  let models = [];
  let cronInfo = null;
  let cronForm = { name: '', prompt: '', schedule: '', oneshot: false, mode: 'yolo' };
  let logs = [];
  let chatEvents = [];
  let serveConfig = '';
  let settings = '';
  let memory = '';
  let memoryInfo = null;
  let prompt = '';
  let lastPrompt = '';
  let response = '';
  let currentSession = '';
  let responseSession = '';
  let selectedModel = 'default';
  let logsConnected = false;
  let busy = false;
  let notice = '';
  let error = '';
  let logsSocket = null;
  let chatAbort = null;

  const jsonHeaders = { 'Content-Type': 'application/json' };

  $: featureList = [
    ['API', status?.features?.openaiAPI !== false],
    ['Web UI', status?.features?.webUI !== false],
    ['WS', status?.features?.websocket === true],
    ['Cron', status?.features?.cron === true],
    ['Memory', status?.features?.memory === true],
    ['Agents', status?.features?.multiAgent === true]
  ];
  $: connectedCount = channels.filter((channel) => channel.connected).length;
  $: activeSession = sessions.find((session) => session.id === currentSession);
  $: recentLogs = logs.slice(-8).reverse();
  $: recentTools = chatEvents.slice(-6).reverse();

  async function request(path, options = {}) {
    const res = await fetch(path, options);
    const text = await res.text();
    const data = text ? JSON.parse(text) : null;
    if (!res.ok) {
      throw new Error(data?.error?.message || data?.error || data?.message || `${res.status} ${res.statusText}`);
    }
    return data;
  }

  async function refresh() {
    error = '';
    try {
      const [h, st, c, sess, cron, sc, s, mem] = await Promise.all([
        request('/health'),
        request('/api/status'),
        request('/api/channels'),
        request('/api/sessions'),
        request('/api/cron'),
        request('/api/serve/config'),
        request('/api/settings'),
        request('/api/memory')
      ]);
      health = h;
      status = st;
      channels = c;
      sessions = sess?.sessions || [];
      cronInfo = cron;
      serveConfig = JSON.stringify(sc, null, 2);
      settings = JSON.stringify(s, null, 2);
      memoryInfo = mem;
      memory = mem?.content || '';
      await refreshModels(st);
    } catch (err) {
      error = err.message;
    }
  }

  async function refreshModels(runtimeStatus = status) {
    if (runtimeStatus?.features?.openaiAPI === false) {
      models = [];
      selectedModel = 'default';
      return;
    }
    try {
      const data = await request('/v1/models');
      models = data?.data || [];
      if (models.length > 0 && !models.some((model) => model.id === selectedModel)) {
        selectedModel = models[0].id;
      }
    } catch {
      models = [];
      selectedModel = 'default';
    }
  }

  async function saveServeConfig() {
    error = '';
    notice = '';
    try {
      const body = JSON.stringify(JSON.parse(serveConfig));
      const saved = await request('/api/serve/config', { method: 'PUT', headers: jsonHeaders, body });
      serveConfig = JSON.stringify(saved, null, 2);
      notice = 'Serve config saved. Restart may be required for listener and channel changes.';
    } catch (err) {
      error = err.message;
    }
  }

  async function saveSettings() {
    error = '';
    notice = '';
    try {
      const body = JSON.stringify(JSON.parse(settings));
      const saved = await request('/api/settings', { method: 'PUT', headers: jsonHeaders, body });
      settings = JSON.stringify(saved, null, 2);
      notice = 'Settings saved.';
    } catch (err) {
      error = err.message;
    }
  }

  async function saveMemory() {
    error = '';
    notice = '';
    try {
      const saved = await request('/api/memory', {
        method: 'PUT',
        headers: jsonHeaders,
        body: JSON.stringify({ content: memory })
      });
      memoryInfo = saved;
      memory = saved?.content || '';
      notice = 'Memory saved.';
    } catch (err) {
      error = err.message;
    }
  }

  async function sendPrompt() {
    const outgoing = prompt.trim();
    if (!outgoing || status?.features?.openaiAPI === false) return;
    busy = true;
    response = '';
    lastPrompt = outgoing;
    responseSession = currentSession;
    chatEvents = [];
    error = '';
    notice = '';
    chatAbort = new AbortController();
    try {
      const body = JSON.stringify({
        model: selectedModel || 'default',
        stream: true,
        x_session_id: currentSession || undefined,
        messages: [{ role: 'user', content: outgoing }]
      });
      prompt = '';
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: jsonHeaders,
        body,
        signal: chatAbort.signal
      });
      if (!res.ok || !res.body) {
        const text = await res.text();
        let data = null;
        try {
          data = text ? JSON.parse(text) : null;
        } catch {
          data = null;
        }
        throw new Error(data?.error?.message || data?.error || data?.message || `${res.status} ${res.statusText}`);
      }
      await readSSE(res.body, handleChatStreamEvent);
    } catch (err) {
      if (err.name === 'AbortError') {
        notice = 'Request stopped.';
      } else {
        error = err.message;
      }
    } finally {
      busy = false;
      chatAbort = null;
      try {
        await refreshSessions();
      } catch {
        // Session refresh is opportunistic after a chat turn.
      }
    }
  }

  function stopChat() {
    if (chatAbort) {
      chatAbort.abort();
    }
  }

  function handlePromptKeydown(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      sendPrompt();
    }
  }

  async function readSSE(body, onEvent) {
    const reader = body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    function flush(final = false) {
      buffer = buffer.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
      let splitAt = buffer.indexOf('\n\n');
      while (splitAt !== -1) {
        dispatchSSEBlock(buffer.slice(0, splitAt), onEvent);
        buffer = buffer.slice(splitAt + 2);
        splitAt = buffer.indexOf('\n\n');
      }
      if (final && buffer.trim()) {
        dispatchSSEBlock(buffer, onEvent);
        buffer = '';
      }
    }

    try {
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        flush();
      }
      buffer += decoder.decode();
      flush(true);
    } finally {
      reader.releaseLock();
    }
  }

  function dispatchSSEBlock(raw, onEvent) {
    const event = parseSSEBlock(raw);
    if (!event || event.data === '') return;
    onEvent(event);
  }

  function parseSSEBlock(raw) {
    const lines = raw.split('\n');
    const data = [];
    let event = 'message';
    for (const line of lines) {
      if (!line || line.startsWith(':')) continue;
      if (line.startsWith('event:')) {
        event = line.slice(6).trim();
      } else if (line.startsWith('data:')) {
        data.push(line.slice(5).trimStart());
      }
    }
    return { event, data: data.join('\n') };
  }

  function handleChatStreamEvent(event) {
    if (event.data === '[DONE]') return;
    if (event.event === 'tool_status') {
      try {
        const item = JSON.parse(event.data);
        chatEvents = [...chatEvents.slice(-49), { type: 'tool', ...item }];
      } catch {
        chatEvents = [...chatEvents.slice(-49), { type: 'tool', status: 'unknown', raw: event.data }];
      }
      return;
    }

    const chunk = JSON.parse(event.data);
    if (chunk?.x_session_id) {
      currentSession = chunk.x_session_id;
      responseSession = chunk.x_session_id;
    }
    const delta = chunk?.choices?.[0]?.delta?.content;
    if (delta) {
      response += delta;
    }
  }

  async function refreshSessions() {
    const data = await request('/api/sessions');
    sessions = data?.sessions || [];
  }

  async function deleteSession(id) {
    if (!id) return;
    error = '';
    notice = '';
    try {
      await request(`/api/sessions/${encodeURIComponent(id)}`, { method: 'DELETE' });
      if (currentSession === id) currentSession = '';
      notice = `Session ${id} deleted.`;
      await refreshSessions();
    } catch (err) {
      error = err.message;
    }
  }

  async function refreshCron() {
    cronInfo = await request('/api/cron');
  }

  async function createCronJob() {
    if (!cronForm.name.trim() || !cronForm.prompt.trim()) return;
    error = '';
    notice = '';
    try {
      await request('/api/cron', {
        method: 'POST',
        headers: jsonHeaders,
        body: JSON.stringify({
          name: cronForm.name.trim(),
          prompt: cronForm.prompt,
          schedule: cronForm.schedule.trim(),
          oneshot: cronForm.oneshot,
          mode: cronForm.mode
        })
      });
      cronForm = { name: '', prompt: '', schedule: '', oneshot: false, mode: 'yolo' };
      notice = 'Cron job created.';
      await refreshCron();
    } catch (err) {
      error = err.message;
    }
  }

  async function setCronEnabled(id, enabled) {
    error = '';
    notice = '';
    try {
      await request(`/api/cron/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        headers: jsonHeaders,
        body: JSON.stringify({ enabled })
      });
      await refreshCron();
    } catch (err) {
      error = err.message;
    }
  }

  async function deleteCronJob(id) {
    if (!id) return;
    error = '';
    notice = '';
    try {
      await request(`/api/cron/${encodeURIComponent(id)}`, { method: 'DELETE' });
      notice = `Cron job ${id} deleted.`;
      await refreshCron();
    } catch (err) {
      error = err.message;
    }
  }

  function connectLogs() {
    if (logsSocket) return;
    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
    logsSocket = new WebSocket(`${scheme}://${window.location.host}/ws/logs`);
    logsSocket.onopen = () => {
      logsConnected = true;
    };
    logsSocket.onmessage = (event) => {
      try {
        const item = JSON.parse(event.data);
        if (item.type === 'heartbeat') return;
        logs = [...logs.slice(-199), item];
        if (item.type === 'connected' && item.status) {
          status = item.status;
        }
      } catch {
        logs = [...logs.slice(-199), { type: 'log', message: event.data }];
      }
    };
    logsSocket.onclose = () => {
      logsConnected = false;
      logsSocket = null;
    };
    logsSocket.onerror = () => {
      logsConnected = false;
    };
  }

  function disconnectLogs() {
    if (logsSocket) {
      logsSocket.close();
      logsSocket = null;
    }
    logsConnected = false;
  }

  function formatTime(value) {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    return date.toLocaleTimeString();
  }

  function formatDateTime(value) {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    return date.toLocaleString();
  }

  function scheduleLabel(job) {
    if (job?.oneshot || !job?.schedule) return 'one-shot';
    return job.schedule;
  }

  function shortID(value) {
    if (!value) return 'default';
    if (value.length <= 18) return value;
    return `${value.slice(0, 8)}...${value.slice(-6)}`;
  }

  function featureClass(enabled) {
    return enabled ? 'on' : 'off';
  }

  function toolStateClass(item) {
    if (item?.status === 'running') return 'running';
    if (item?.status === 'error' || item?.status === 'failed') return 'error';
    return 'done';
  }

  function formatLogMessage(item) {
    if (!item) return '';
    if (item.message) return item.message;
    return JSON.stringify(item.status || item);
  }

  function formatArgs(value) {
    if (!value) return '';
    return JSON.stringify(value, null, 2);
  }

  onDestroy(disconnectLogs);

  refresh();
</script>

<main class="shell">
  <aside class="sidebar">
    <div class="brand">
      <div class="mark">Mx</div>
      <div>
        <h1>MothX</h1>
        <p>{status?.listen || 'serve runtime'}</p>
      </div>
    </div>

    <nav class="nav">
      <a href="#chat">Chat</a>
      <a href="#cron">Cron</a>
      <a href="#settings">Config</a>
      <a href="#logs">Logs</a>
    </nav>

    <section class="sideBlock" id="sessions">
      <div class="blockHead">
        <span>Sessions</span>
        <button class="ghost sm" on:click={refreshSessions}>Refresh</button>
      </div>
      <button class:active={currentSession === ''} class="sessionButton" on:click={() => (currentSession = '')}>
        <strong>Default session</strong>
        <span>shared context</span>
      </button>
      <div class="sessionStack">
        {#each sessions as session}
          <div class="sessionItem">
            <button class:active={currentSession === session.id} class="sessionButton" on:click={() => (currentSession = session.id)}>
              <strong>{shortID(session.id)}</strong>
              <span>{session.messageCount || 0} messages</span>
              <span>{session.workDir}</span>
            </button>
            <button class="iconDanger" title="Delete session" on:click={() => deleteSession(session.id)}>×</button>
          </div>
        {/each}
        {#if sessions.length === 0}
          <p class="empty">No active sessions.</p>
        {/if}
      </div>
    </section>

    <section class="sideBlock">
      <div class="blockHead">
        <span>Features</span>
      </div>
      <div class="featureStack">
        {#each featureList as item}
          <div class="featureLine">
            <span>{item[0]}</span>
            <strong class={featureClass(item[1])}>{item[1] ? 'on' : 'off'}</strong>
          </div>
        {/each}
      </div>
    </section>
  </aside>

  <section class="workbench">
    <header class="topbar">
      <div class="runtimeTitle">
        <strong>{status?.status || health?.status || 'loading'}</strong>
        <span>v{health?.version || 'dev'}</span>
        <span>{status?.sessions ?? health?.sessions ?? 0} sessions</span>
        <span>{connectedCount} channels</span>
      </div>
      <div class="topActions">
        {#if logsConnected}
          <button class="ghost" on:click={disconnectLogs}>Logs off</button>
        {:else}
          <button class="ghost" on:click={connectLogs}>Logs on</button>
        {/if}
        <button on:click={refresh}>Refresh</button>
      </div>
    </header>

    {#if error}
      <div class="banner error">{error}</div>
    {/if}
    {#if notice}
      <div class="banner">{notice}</div>
    {/if}

    <section id="chat" class="primaryGrid">
      <section class="chatSurface">
        <div class="surfaceHead">
          <div>
            <h2>Chat</h2>
            <p>{activeSession?.workDir || status?.listen || 'default workspace'}</p>
          </div>
          <div class="chatControls">
            <select bind:value={selectedModel} disabled={status?.features?.openaiAPI === false || models.length === 0}>
              {#if models.length === 0}
                <option value="default">Default model</option>
              {:else}
                {#each models as model}
                  <option value={model.id}>{model.id}</option>
                {/each}
              {/if}
            </select>
            <select bind:value={currentSession}>
              <option value="">Default session</option>
              {#each sessions as session}
                <option value={session.id}>{shortID(session.id)}</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="transcript">
          {#if lastPrompt}
            <article class="message user">
              <div class="messageMeta">
                <strong>You</strong>
                <span>{responseSession ? shortID(responseSession) : shortID(currentSession)}</span>
              </div>
              <p>{lastPrompt}</p>
            </article>
          {/if}

          {#if response}
            <article class="message assistant">
              <div class="messageMeta">
                <strong>MothX</strong>
                <span>{busy ? 'streaming' : 'complete'}</span>
              </div>
              <div class="markdown">{@html markdownToHTML(response)}</div>
            </article>
          {:else if busy}
            <article class="message assistant pending">
              <div class="messageMeta">
                <strong>MothX</strong>
                <span>streaming</span>
              </div>
              <p>Waiting for model...</p>
            </article>
          {:else if !lastPrompt}
            <div class="emptyState">
              <strong>Ready</strong>
              <span>{status?.features?.openaiAPI === false ? 'OpenAI API is disabled.' : 'Start a session or continue the default context.'}</span>
            </div>
          {/if}
        </div>

        <div class="composer">
          <textarea bind:value={prompt} on:keydown={handlePromptKeydown} placeholder="Message MothX"></textarea>
          <div class="composerBar">
            <span>{status?.features?.openaiAPI === false ? 'API disabled' : selectedModel || 'default model'}</span>
            <div>
              {#if busy}
                <button on:click={stopChat}>Stop</button>
              {/if}
              <button class="primary" disabled={busy || !prompt.trim() || status?.features?.openaiAPI === false} on:click={sendPrompt}>
                {busy ? 'Running' : 'Send'}
              </button>
            </div>
          </div>
        </div>
      </section>

      <aside class="inspector">
        <section class="inspectBlock">
          <div class="blockHead">
            <span>Tools</span>
            <strong>{chatEvents.length}</strong>
          </div>
          <div class="toolStack">
            {#each recentTools as item}
              <details class="toolItem" open={item.status === 'running'}>
                <summary>
                  <span class:running={toolStateClass(item) === 'running'} class:error={toolStateClass(item) === 'error'}></span>
                  <strong>{item.tool || item.type}</strong>
                  <em>{item.status || 'event'}</em>
                </summary>
                {#if item.args}
                  <pre>{formatArgs(item.args)}</pre>
                {:else if item.raw}
                  <pre>{item.raw}</pre>
                {/if}
              </details>
            {/each}
            {#if recentTools.length === 0}
              <p class="empty">No tool events.</p>
            {/if}
          </div>
        </section>

        <section class="inspectBlock">
          <div class="blockHead">
            <span>Channels</span>
            <strong>{connectedCount}/{channels.length}</strong>
          </div>
          <div class="channelStack" id="channels">
            {#each channels as channel}
              <div class="channelLine">
                <span>{channel.name}</span>
                <strong class:online={channel.connected}>{channel.enabled ? (channel.connected ? 'connected' : 'offline') : 'disabled'}</strong>
              </div>
            {/each}
            {#if channels.length === 0}
              <p class="empty">No channels.</p>
            {/if}
          </div>
        </section>

        <section class="inspectBlock" id="logs">
          <div class="blockHead">
            <span>Logs</span>
            <button class="ghost sm" on:click={() => (logs = [])}>Clear</button>
          </div>
          <div class="logStream">
            {#each recentLogs as item}
              <div class="logLine">
                <span>{formatTime(item.timestamp)}</span>
                <strong>{item.type}</strong>
                <code>{formatLogMessage(item)}</code>
              </div>
            {/each}
            {#if recentLogs.length === 0}
              <p class="empty">No log events.</p>
            {/if}
          </div>
        </section>
      </aside>
    </section>

    <section id="cron" class="panel">
      <div class="panelHead">
        <div>
          <h2>Cron</h2>
          <span>{cronInfo?.enabled === false ? 'disabled' : cronInfo?.running ? 'running' : 'idle'} {cronInfo?.path || ''}</span>
        </div>
        <button on:click={refreshCron}>Refresh</button>
      </div>
      <form class="cronCreate" on:submit|preventDefault={createCronJob}>
        <input disabled={cronInfo?.enabled === false} bind:value={cronForm.name} placeholder="Name" />
        <input disabled={cronInfo?.enabled === false || cronForm.oneshot} bind:value={cronForm.schedule} placeholder="@daily" />
        <select disabled={cronInfo?.enabled === false} bind:value={cronForm.mode}>
          <option value="yolo">yolo</option>
          <option value="agent">agent</option>
        </select>
        <label>
          <input disabled={cronInfo?.enabled === false} type="checkbox" bind:checked={cronForm.oneshot} />
          <span>one-shot</span>
        </label>
        <button class="primary" disabled={cronInfo?.enabled === false || !cronForm.name.trim() || !cronForm.prompt.trim()} type="submit">Create</button>
        <textarea disabled={cronInfo?.enabled === false} bind:value={cronForm.prompt} placeholder="Prompt"></textarea>
      </form>
      <div class="cronList">
        {#each cronInfo?.jobs || [] as job}
          <div class="cronRow">
            <div>
              <strong>{job.name}</strong>
              <span>{shortID(job.id)}</span>
            </div>
            <div>
              <span>{scheduleLabel(job)}</span>
              <span>{job.mode || 'yolo'}</span>
              <span>{job.run_count || 0} runs</span>
              {#if job.next_run}
                <span>{formatDateTime(job.next_run)}</span>
              {/if}
              {#if job.last_status}
                <span>{job.last_status}</span>
              {/if}
            </div>
            <div class="actions compact">
              {#if job.enabled}
                <button on:click={() => setCronEnabled(job.id, false)}>Disable</button>
              {:else}
                <button on:click={() => setCronEnabled(job.id, true)}>Enable</button>
              {/if}
              <button class="danger" on:click={() => deleteCronJob(job.id)}>Delete</button>
            </div>
            {#if job.last_error}
              <code>{job.last_error}</code>
            {/if}
          </div>
        {/each}
        {#if (cronInfo?.jobs || []).length === 0}
          <p class="empty">No cron jobs.</p>
        {/if}
      </div>
    </section>

    <section id="settings" class="configGrid">
      <div class="panel editor">
        <div class="panelHead">
          <h2>Serve Config</h2>
          <button on:click={saveServeConfig}>Save</button>
        </div>
        <textarea class="code" bind:value={serveConfig}></textarea>
      </div>

      <div class="panel editor">
        <div class="panelHead">
          <h2>Settings</h2>
          <button on:click={saveSettings}>Save</button>
        </div>
        <textarea class="code" bind:value={settings}></textarea>
      </div>

      <div class="panel editor">
        <div class="panelHead">
          <div>
            <h2>Memory</h2>
            <span>{memoryInfo?.enabled === false ? 'disabled' : memoryInfo?.path || 'not created'}</span>
          </div>
          <button disabled={memoryInfo?.enabled === false} on:click={saveMemory}>Save</button>
        </div>
        <textarea class="code" disabled={memoryInfo?.enabled === false} bind:value={memory}></textarea>
      </div>
    </section>
  </section>
</main>
