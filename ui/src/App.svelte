<script>
  let health = null;
  let channels = [];
  let serveConfig = '';
  let settings = '';
  let prompt = '';
  let response = '';
  let busy = false;
  let notice = '';
  let error = '';

  const jsonHeaders = { 'Content-Type': 'application/json' };

  async function request(path, options = {}) {
    const res = await fetch(path, options);
    const text = await res.text();
    const data = text ? JSON.parse(text) : null;
    if (!res.ok) {
      throw new Error(data?.error || data?.message || `${res.status} ${res.statusText}`);
    }
    return data;
  }

  async function refresh() {
    error = '';
    try {
      const [h, c, sc, s] = await Promise.all([
        request('/health'),
        request('/api/channels'),
        request('/api/serve/config'),
        request('/api/settings')
      ]);
      health = h;
      channels = c;
      serveConfig = JSON.stringify(sc, null, 2);
      settings = JSON.stringify(s, null, 2);
    } catch (err) {
      error = err.message;
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

  async function sendPrompt() {
    if (!prompt.trim()) return;
    busy = true;
    response = '';
    error = '';
    try {
      const body = JSON.stringify({
        model: 'default',
        stream: false,
        messages: [{ role: 'user', content: prompt }]
      });
      const data = await request('/v1/chat/completions', { method: 'POST', headers: jsonHeaders, body });
      response = data?.choices?.[0]?.message?.content || JSON.stringify(data, null, 2);
    } catch (err) {
      error = err.message;
    } finally {
      busy = false;
    }
  }

  refresh();
</script>

<main class="shell">
  <aside class="rail">
    <div class="brand">
      <div class="mark">Mx</div>
      <div>
        <h1>MothX Serve</h1>
        <p>OpenAI API, Web UI, channels</p>
      </div>
    </div>

    <nav>
      <a href="#chat">Chat</a>
      <a href="#settings">Settings</a>
      <a href="#channels">Channels</a>
      <a href="#api">API</a>
    </nav>
  </aside>

  <section class="workspace">
    <header class="topbar">
      <div>
        <strong>{health?.status || 'loading'}</strong>
        <span>v{health?.version || 'dev'}</span>
        <span>{health?.sessions ?? 0} sessions</span>
      </div>
      <button on:click={refresh}>Refresh</button>
    </header>

    {#if error}
      <div class="banner error">{error}</div>
    {/if}
    {#if notice}
      <div class="banner">{notice}</div>
    {/if}

    <section id="chat" class="panel chat">
      <div class="panelHead">
        <h2>Chat</h2>
        <span>/v1/chat/completions</span>
      </div>
      <textarea bind:value={prompt} placeholder="Ask MothX to inspect, edit, or explain this workspace"></textarea>
      <div class="actions">
        <button class="primary" disabled={busy} on:click={sendPrompt}>{busy ? 'Running' : 'Send'}</button>
      </div>
      {#if response}
        <pre class="response">{response}</pre>
      {/if}
    </section>

    <section id="channels" class="panel">
      <div class="panelHead">
        <h2>Channels</h2>
        <span>Feishu / WeChat</span>
      </div>
      <div class="channelGrid">
        {#each channels as channel}
          <div class="channel">
            <strong>{channel.name}</strong>
            <span class:online={channel.connected}>{channel.connected ? 'connected' : 'offline'}</span>
          </div>
        {/each}
        {#if channels.length === 0}
          <p class="empty">No channels connected.</p>
        {/if}
      </div>
    </section>

    <section id="settings" class="grid">
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
    </section>

    <section id="api" class="panel">
      <div class="panelHead">
        <h2>API</h2>
        <span>OpenAI compatible</span>
      </div>
      <div class="endpoint">
        <code>POST /v1/chat/completions</code>
        <code>GET /v1/models</code>
        <code>GET /health</code>
      </div>
    </section>
  </section>
</main>

