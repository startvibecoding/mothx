<script>
  import { markdownToHTML } from '../lib/markdown.js';
  import { request, readSSE } from '../lib/api.js';
  import {
    sessions,
    currentSession,
    selectedModel,
    models,
    features,
    setError,
    setNotice,
    clearBanners,
    refreshSessions,
    getSessionMessages
  } from '../lib/stores.js';
  import { shortID, toolStateClass, formatArgs } from '../lib/format.js';
  import DirBrowser from '../components/DirBrowser.svelte';

  let prompt = '';
  let messages = [];
  let busy = false;
  let chatAbort = null;
  let chatEvents = [];
  let workDir = '';
  let sessionCreated = false;
  let showBrowser = false;

  const suggestions = [
    '资讯：Fable 5 用 1600 行代码生成水下曼哈顿获赞',
    '教我如何判断电脑是否需要清灰',
    '推荐几本末日求生题材的小说',
    '帮我分析线稿常见错误及改进方法',
    '资讯：前端工具链正从 JavaScript 向 Rust 逐步迁移',
    '分析机械盘选购注意事项',
    '创作一段末世废土风格的环境描写',
    '分析振荡器在电子乐器中的应用案例'
  ];

  // Reset state when session becomes empty (new chat)
  let prevSession = $currentSession;
  $: if ($currentSession === '' && prevSession !== '') {
    sessionCreated = false;
    workDir = '';
    messages = []; // new chat — no history
    chatEvents = []; // reset tool events
  } else if ($currentSession && prevSession !== $currentSession) {
    // Switched to an existing session — load its messages
    loadSessionMessages($currentSession);
  }
  prevSession = $currentSession;

  async function loadSessionMessages(id) {
    try {
      const msgs = await getSessionMessages(id);
      if (msgs && msgs.length > 0) {
        messages = msgs.map((m) => ({ role: m.role, content: m.content }));
      } else {
        messages = [];
      }
      chatEvents = []; // reset tool events for new session view
    } catch {
      // Leave messages empty on error
    }
    sessionCreated = true; // existing session, not "new"
  }

  $: activeSession = $sessions.find((s) => s.id === $currentSession);
  $: recentTools = chatEvents.slice(-6).reverse();
  $: modelOptions = $models;
  $: apiEnabled = $features.api;
  $: isNewSession = !$currentSession && !sessionCreated;
  $: activeSessionWorkDir = activeSession?.workDir || workDir.trim();

  function pick(text) {
    if (busy) return;
    prompt = text;
    sendPrompt();
  }

  function handleKeydown(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      sendPrompt();
    }
  }

  async function sendPrompt() {
    const outgoing = prompt.trim();
    if (!outgoing || !apiEnabled) return;
    if (isNewSession && !workDir.trim()) {
      setError('请先填写工作目录');
      return;
    }
    busy = true;
    chatEvents = [];
    clearBanners();

    // Add user message
    messages = [...messages, { role: 'user', content: outgoing }];
    prompt = '';

    chatAbort = new AbortController();
    try {
      const body = JSON.stringify({
        model: $selectedModel || 'default',
        stream: true,
        x_session_id: $currentSession || undefined,
        x_working_dir: workDir.trim(),
        messages: messages.map((m) => ({ role: m.role, content: m.content }))
      });
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
        signal: chatAbort.signal
      });
      if (!res.ok || !res.body) {
        const text = await res.text();
        let data = null;
        try { data = text ? JSON.parse(text) : null; } catch { data = null; }
        throw new Error(data?.error?.message || data?.error || data?.message || `${res.status} ${res.statusText}`);
      }
      // Add placeholder assistant message
      messages = [...messages, { role: 'assistant', content: '' }];
      await readSSE(res.body, handleStreamEvent);
      sessionCreated = true;
    } catch (err) {
      if (err?.name === 'AbortError') setNotice('已停止请求。');
      else setError(err);
    } finally {
      busy = false;
      chatAbort = null;
      try { await refreshSessions(); } catch {
        // opportunistic
      }
    }
  }

  function stop() {
    if (chatAbort) chatAbort.abort();
  }

  function resetSession() {
    currentSession.set('');
  }

  function onDirSelect(e) {
    workDir = e.detail.path;
    showBrowser = false;
  }

  function handleStreamEvent(event) {
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
    try {
      const chunk = JSON.parse(event.data);
      if (chunk?.x_session_id) {
        currentSession.set(chunk.x_session_id);
      }
      const delta = chunk?.choices?.[0]?.delta?.content;
      if (delta) {
        messages[messages.length - 1].content += delta;
        messages = messages;
      }
    } catch {
      // ignore malformed frames
    }
  }
</script>

<section class="chat-view">
  <div class="chat-scroll">
    {#if messages.length === 0 && !busy}
      <div class="welcome">
        <h2>有什么我能帮你的吗？</h2>
        <div class="suggestions">
          {#each suggestions as text}
            <button
              type="button"
              class="chip"
              disabled={!apiEnabled || (isNewSession && !workDir.trim())}
              on:click={() => pick(text)}
            >
              {text}
            </button>
          {/each}
        </div>
      </div>
    {:else}
      <div class="transcript">
        {#each messages as msg, idx}
          {#if msg.role === 'user'}
            <article class="msg user">
              <div class="meta">
                <strong>你</strong>
                <span>{shortID($currentSession)}</span>
              </div>
              <p>{msg.content}</p>
            </article>
          {:else}
            <article class="msg assistant">
              <div class="meta">
                <strong>MothX</strong>
                <span>{busy && idx === messages.length - 1 ? '生成中' : '完成'}</span>
              </div>
              {#if msg.content}
                <div class="markdown">{@html markdownToHTML(msg.content)}</div>
              {:else if busy && idx === messages.length - 1}
                <p class="pending-text">正在等待模型响应…</p>
              {/if}
            </article>
          {/if}
        {/each}
        {#if recentTools.length > 0}
          <aside class="tool-feed">
            <div class="tf-head"><span>工具事件</span><strong>{chatEvents.length}</strong></div>
            {#each recentTools as item}
              <details class="tool-item" open={item.status === 'running'}>
                <summary>
                  <span class="dot {toolStateClass(item)}"></span>
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
          </aside>
        {/if}
      </div>
    {/if}
  </div>

  <div class="composer">
    {#if isNewSession}
      <div class="composer-workdir">
        {#if workDir}
          <span class="dir-display">
            <span class="ico">📁</span>
            <span class="dir-path-text">{workDir}</span>
            <button type="button" class="ghost sm" on:click={() => (workDir = '')}>清除</button>
          </span>
        {/if}
        <button type="button" class="dir-btn" on:click={() => (showBrowser = true)}>
          <span class="ico">📂</span>
          {workDir ? '更换目录' : '选择工作目录'}
        </button>
      </div>
    {:else if $currentSession}
      <div class="composer-session-info">
        <span class="session-badge">会话</span>
        <span class="session-id">{shortID($currentSession)}</span>
        {#if activeSessionWorkDir}<span class="session-dir">{activeSessionWorkDir}</span>{/if}
        <button type="button" class="ghost sm" on:click={resetSession}>新建会话</button>
      </div>
    {/if}
    <div class="composer-row">
      <textarea
        bind:value={prompt}
        on:keydown={handleKeydown}
        placeholder={!apiEnabled ? 'OpenAI API 已禁用' : (isNewSession && !workDir.trim()) ? '请先填写工作目录…' : '发消息…'}
        disabled={!apiEnabled}
        rows="1"
      ></textarea>
    </div>
    <div class="composer-bar">
      <div class="left">
        <select
          bind:value={$selectedModel}
          disabled={!apiEnabled || modelOptions.length === 0}
          aria-label="选择模型"
        >
          {#if modelOptions.length === 0}
            <option value="default">默认模型</option>
          {:else}
            {#each modelOptions as m}
              <option value={m.id}>{m.id}</option>
            {/each}
          {/if}
        </select>
        <select bind:value={$currentSession} aria-label="选择会话">
          <option value="">默认会话</option>
          {#each $sessions as s}
            <option value={s.id}>{shortID(s.id)}</option>
          {/each}
        </select>
      </div>
      <div class="right">
        {#if busy}
          <button type="button" class="ghost" on:click={stop}>停止</button>
        {/if}
        <button
          type="button"
          class="primary"
          disabled={busy || !prompt.trim() || !apiEnabled || (isNewSession && !workDir.trim())}
          on:click={sendPrompt}
        >
          {busy ? '发送中' : '发送'}
        </button>
      </div>
    </div>
  </div>
</section>

<DirBrowser bind:open={showBrowser} on:select={onDirSelect} on:close={() => (showBrowser = false)} />
