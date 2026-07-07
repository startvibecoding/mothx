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
    getSessionMessages,
    getSessionToolResult
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
  let imageInput;
  let imageUploads = [];

  const suggestions = [
    '阅读当前项目并总结核心模块和调用链',
    '审查最近的代码改动，列出潜在回归风险',
    '帮我为这个 Go 包补充关键单元测试',
    '定位测试失败原因并给出最小修复方案',
    '重构这段代码，保持行为不变并降低复杂度',
    '检查配置文件和启动参数是否存在冲突',
    '为新增功能整理一份简洁的 README 说明',
    '生成一个安全的多 Agent 任务拆解方案'
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
        messages = msgs.map(normalizeSessionMessage).filter(Boolean);
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
  $: activeModel = modelOptions.find((m) => m.id === $selectedModel);
  $: selectedModelSupportsImages = (activeModel?.input || []).includes('image');
  $: apiEnabled = $features.api;
  $: isNewSession = !$currentSession && !sessionCreated;
  $: activeSessionWorkDir = activeSession?.workDir || workDir.trim();
  $: if ($currentSession && activeSession?.workDir && workDir !== activeSession.workDir) {
    workDir = activeSession.workDir;
  }
  $: if (!selectedModelSupportsImages && imageUploads.length > 0) {
    clearImages();
  }

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
    const outgoingImages = imageUploads;
    if ((!outgoing && outgoingImages.length === 0) || !apiEnabled) return;
    if (outgoingImages.length > 0 && !selectedModelSupportsImages) {
      setError('当前模型不支持图片输入');
      return;
    }
    if (isNewSession && !workDir.trim()) {
      setError('请先填写工作目录');
      return;
    }
    busy = true;
    chatEvents = [];
    clearBanners();

    // Add user message
    messages = [...messages, { role: 'user', content: outgoing, images: outgoingImages }];
    prompt = '';
    imageUploads = [];
    if (imageInput) imageInput.value = '';

    chatAbort = new AbortController();
    try {
      const requestMessages = messages.map((m, idx) => {
        if (idx === messages.length - 1 && outgoingImages.length > 0) {
          return { role: m.role, content: buildOutgoingContent(outgoing, outgoingImages) };
        }
        return { role: m.role, content: m.content || '' };
      });
      const body = JSON.stringify({
        model: $selectedModel || 'default',
        stream: true,
        x_session_id: $currentSession || undefined,
        x_working_dir: isNewSession ? workDir.trim() : activeSessionWorkDir,
        messages: requestMessages
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
      if ($currentSession) {
        try { await loadSessionMessages($currentSession); } catch {
          // opportunistic
        }
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

  async function handleImageSelect(event) {
    const files = Array.from(event.target.files || []);
    if (files.length === 0) return;
    if (!selectedModelSupportsImages) {
      setError('当前模型不支持图片输入');
      event.target.value = '';
      return;
    }
    try {
      const next = await Promise.all(files.map(readImageFile));
      imageUploads = [...imageUploads, ...next].slice(0, 6);
    } catch (err) {
      setError(err);
    } finally {
      event.target.value = '';
    }
  }

  function readImageFile(file) {
    if (!file.type.startsWith('image/')) {
      throw new Error(`不支持的文件类型：${file.name}`);
    }
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve({
        name: file.name,
        type: file.type,
        size: file.size,
        dataUrl: String(reader.result || '')
      });
      reader.onerror = () => reject(new Error(`读取图片失败：${file.name}`));
      reader.readAsDataURL(file);
    });
  }

  function removeImage(index) {
    imageUploads = imageUploads.filter((_, i) => i !== index);
  }

  function clearImages() {
    imageUploads = [];
    if (imageInput) imageInput.value = '';
  }

  function buildOutgoingContent(text, images) {
    const parts = [];
    if (text) parts.push({ type: 'text', text });
    for (const image of images) {
      parts.push({
        type: 'image_url',
        image_url: { url: image.dataUrl, detail: 'auto' }
      });
    }
    return parts;
  }

  function formatImageSize(bytes) {
    if (!bytes) return '';
    if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  }

  function normalizeSessionMessage(message) {
    if (message.role === 'toolCall') {
      const plan = normalizePlan(message.plan || (message.toolName === 'plan' ? message.arguments : null));
      if (message.toolName === 'plan' && plan) {
        return {
          role: 'plan',
          toolCallId: message.toolCallId,
          toolName: message.toolName,
          plan
        };
      }
      return {
        role: 'toolCall',
        toolCallId: message.toolCallId,
        toolName: message.toolName || 'tool',
        arguments: message.arguments,
        invalidArguments: message.invalidArguments,
        callView: buildToolCallView(message.toolName || 'tool', message.arguments, message.invalidArguments)
      };
    }
    if (message.role === 'toolResult') {
      if (message.toolName === 'plan' && !message.isError) return null;
      return {
        role: 'toolResult',
        toolCallId: message.toolCallId,
        toolName: message.toolName || 'tool',
        summary: message.summary || '工具结果',
        isError: message.isError,
        hasDetail: message.hasDetail,
        detailLoaded: false,
        detailLoading: false,
        detailError: '',
        detail: null
      };
    }
    const images = [];
    for (const block of message.contents || []) {
      if (block.type !== 'image' || !block.image?.data || !block.image?.mimeType) continue;
      images.push({
        name: block.image.mimeType,
        type: block.image.mimeType,
        size: block.image.bytes || block.image.originalBytes || 0,
        dataUrl: `data:${block.image.mimeType};base64,${block.image.data}`
      });
    }
    return {
      role: message.role,
      content: message.content || textFromContents(message.contents),
      images
    };
  }

  function normalizePlan(value) {
    if (!value || !Array.isArray(value.steps) || value.steps.length === 0) return null;
    const steps = value.steps
      .map((step) => ({
        title: String(step?.title || '').trim(),
        status: normalizePlanStatus(step?.status)
      }))
      .filter((step) => step.title);
    if (steps.length === 0) return null;
    return {
      title: String(value.title || '').trim(),
      note: String(value.note || '').trim(),
      steps
    };
  }

  function normalizePlanStatus(status) {
    const s = String(status || '').trim().toLowerCase();
    if (['pending', 'running', 'done', 'failed'].includes(s)) return s;
    return 'pending';
  }

  function planStatusLabel(status) {
    switch (status) {
      case 'done': return '完成';
      case 'running': return '进行中';
      case 'failed': return '失败';
      default: return '待处理';
    }
  }

  async function loadToolResultDetail(msg, event) {
    if (!event.currentTarget.open || !msg.hasDetail || msg.detailLoaded || msg.detailLoading) return;
    if (!$currentSession || !msg.toolCallId) return;
    msg.detailLoading = true;
    msg.detailError = '';
    messages = messages;
    try {
      const detail = await getSessionToolResult($currentSession, msg.toolCallId);
      msg.detail = normalizeToolResultDetail(detail);
      msg.detailLoaded = true;
    } catch (err) {
      msg.detailError = err instanceof Error ? err.message : String(err || '加载失败');
    } finally {
      msg.detailLoading = false;
      messages = messages;
    }
  }

  function normalizeToolResultDetail(detail) {
    if (!detail) return { content: '', images: [] };
    const images = [];
    for (const block of detail.contents || []) {
      if (block.type !== 'image' || !block.image?.data || !block.image?.mimeType) continue;
      images.push({
        name: block.image.mimeType,
        type: block.image.mimeType,
        size: block.image.bytes || block.image.originalBytes || 0,
        dataUrl: `data:${block.image.mimeType};base64,${block.image.data}`
      });
    }
    const content = detail.content || textFromContents(detail.contents);
    return {
      toolName: detail.toolName || '',
      kind: toolResultKind(detail.toolName, content),
      content: detail.content || textFromContents(detail.contents),
      images,
      readLines: parseReadResult(content),
      lsEntries: parseLsResult(content),
      grepMatches: parseGrepResult(content),
      bashResult: parseBashResult(content)
    };
  }

  function buildToolCallView(toolName, args, invalidArguments = '') {
    const name = toolName || 'tool';
    const value = isPlainObject(args) ? args : {};
    if (name === 'read') {
      const details = [];
      if (value.offset) details.push(`从第 ${value.offset} 行`);
      if (value.limit) details.push(`最多 ${value.limit} 行`);
      if (value.imageMode) details.push(`图片模式 ${value.imageMode}`);
      if (value.maxLongEdge) details.push(`最长边 ${value.maxLongEdge}px`);
      if (value.crop) details.push(`裁剪 ${value.crop.width || 0}x${value.crop.height || 0}+${value.crop.x || 0}+${value.crop.y || 0}`);
      return {
        kind: 'read',
        label: '读取文件',
        target: value.path || '(未指定文件)',
        details,
        raw: args,
        invalidArguments
      };
    }
    if (name === 'ls') {
      return {
        kind: 'ls',
        label: '列出目录',
        target: value.path || '.',
        details: [],
        raw: args,
        invalidArguments
      };
    }
    if (name === 'grep') {
      const details = [];
      if (value.path) details.push(value.path);
      if (value.include) details.push(`include ${value.include}`);
      if (value.maxResults) details.push(`最多 ${value.maxResults} 条`);
      return {
        kind: 'grep',
        label: '搜索文本',
        target: value.pattern || '(未指定 pattern)',
        details,
        raw: args,
        invalidArguments
      };
    }
    if (name === 'bash') {
      const details = [];
      if (value.async) details.push('后台运行');
      if (value.timeout !== undefined && value.timeout !== null) {
        details.push(Number(value.timeout) === 0 ? '无工具超时' : `超时 ${value.timeout}s`);
      }
      return {
        kind: 'bash',
        label: '执行命令',
        target: value.command || '(未指定命令)',
        details,
        raw: args,
        invalidArguments
      };
    }
    return {
      kind: 'generic',
      label: name,
      target: '',
      details: [],
      raw: args,
      invalidArguments
    };
  }

  function isPlainObject(value) {
    return value && typeof value === 'object' && !Array.isArray(value);
  }

  function toolResultKind(toolName, content) {
    if (toolName === 'read' && parseReadResult(content).length > 0) return 'read';
    if (toolName === 'ls' && (parseLsResult(content).length > 0 || content === '(empty directory)')) return 'ls';
    if (toolName === 'grep' && (parseGrepResult(content).matches.length > 0 || content === '(no matches found)')) return 'grep';
    if (toolName === 'bash' && parseBashResult(content)) return 'bash';
    return 'text';
  }

  function parseReadResult(content = '') {
    if (!content) return [];
    const lines = content.split('\n').filter((line) => line.length > 0);
    const parsed = [];
    for (const line of lines) {
      const match = line.match(/^(\d+)\t(.*)$/);
      if (!match) return [];
      parsed.push({ number: match[1], text: match[2] });
    }
    return parsed;
  }

  function parseLsResult(content = '') {
    if (!content || content === '(empty directory)') return [];
    const entries = [];
    for (const line of content.split('\n')) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      const dir = trimmed.match(/^📁\s+(.+)\/$/);
      if (dir) {
        entries.push({ type: 'dir', name: dir[1], size: '' });
        continue;
      }
      const file = trimmed.match(/^📄\s+(.+)\s+\(([^)]+)\)$/);
      if (file) {
        entries.push({ type: 'file', name: file[1], size: file[2] });
        continue;
      }
      return [];
    }
    return entries;
  }

  function parseGrepResult(content = '') {
    const result = { matches: [], note: '' };
    if (!content || content === '(no matches found)') return result;
    for (const line of content.split('\n')) {
      if (!line) continue;
      if (line.startsWith('... (truncated')) {
        result.note = line;
        continue;
      }
      const match = line.match(/^(.+):(\d+):(.*)$/);
      if (!match) return { matches: [], note: '' };
      result.matches.push({ path: match[1], line: match[2], text: match[3] });
    }
    return result;
  }

  function parseBashResult(content = '') {
    if (!content) return null;
    const sections = parseTaggedSections(content);
    if (!sections.runtime && !sections.command && !sections.stdout && !sections.stderr && !sections.exit_code) {
      return null;
    }
    let command = sections.command || '';
    let note = '';
    const noteIndex = command.indexOf("\nUse 'jobs' tool");
    if (noteIndex >= 0) {
      note = command.slice(noteIndex + 1).trim();
      command = command.slice(0, noteIndex).trimEnd();
    }
    return {
      runtime: sections.runtime || '',
      command,
      cwd: sections.cwd || '',
      stdout: sections.stdout || '',
      stderr: sections.stderr || '',
      exitCode: sections.exit_code || '',
      note,
      prefix: sections.__prefix || ''
    };
  }

  function parseTaggedSections(content = '') {
    const sections = { __prefix: [] };
    let current = '__prefix';
    for (const line of content.split('\n')) {
      const match = line.match(/^\[([a-z_]+)\]$/);
      if (match) {
        current = match[1];
        if (!sections[current]) sections[current] = [];
        continue;
      }
      sections[current].push(line);
    }
    const out = {};
    for (const [key, lines] of Object.entries(sections)) {
      out[key] = lines.join('\n').trim();
    }
    return out;
  }

  function textFromContents(contents = []) {
    return contents
      .filter((block) => block.type === 'text' && block.text)
      .map((block) => block.text)
      .join('\n');
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
              {#if msg.images?.length}
                <div class="msg-images">
                  {#each msg.images as image}
                    <img src={image.dataUrl} alt={image.name} />
                  {/each}
                </div>
              {/if}
            </article>
          {:else if msg.role === 'assistant'}
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
          {:else if msg.role === 'plan'}
            <article class="msg plan-card">
              <div class="meta">
                <strong>任务计划</strong>
                {#if msg.toolCallId}<span>{shortID(msg.toolCallId)}</span>{/if}
              </div>
              <section class="todo-plan">
                {#if msg.plan.title}
                  <h3>{msg.plan.title}</h3>
                {/if}
                <ol>
                  {#each msg.plan.steps as step}
                    <li class:done={step.status === 'done'} class:running={step.status === 'running'} class:failed={step.status === 'failed'}>
                      <span class="todo-mark" aria-hidden="true"></span>
                      <span class="todo-title">{step.title}</span>
                      <em>{planStatusLabel(step.status)}</em>
                    </li>
                  {/each}
                </ol>
                {#if msg.plan.note}
                  <p>{msg.plan.note}</p>
                {/if}
              </section>
            </article>
          {:else if msg.role === 'toolCall'}
            <article class="msg tool-call">
              <div class="meta">
                <strong>工具调用</strong>
                <span>{msg.toolName}</span>
              </div>
              <div class="tool-call-body">
                <div class="tool-title">
                  <span class="dot running"></span>
                  <strong>{msg.callView?.label || msg.toolName}</strong>
                  {#if msg.callView?.target}
                    <span class="tool-target">{msg.callView.target}</span>
                  {/if}
                  {#if msg.toolCallId}<em>{shortID(msg.toolCallId)}</em>{/if}
                </div>
                {#if msg.callView?.details?.length}
                  <div class="tool-call-tags">
                    {#each msg.callView.details as item}
                      <span>{item}</span>
                    {/each}
                  </div>
                {/if}
                {#if msg.callView?.kind !== 'generic' && msg.arguments}
                  <details class="tool-raw">
                    <summary>参数 JSON</summary>
                    <pre>{formatArgs(msg.arguments)}</pre>
                  </details>
                {:else if msg.arguments}
                  <pre>{formatArgs(msg.arguments)}</pre>
                {:else if msg.invalidArguments}
                  <pre>{msg.invalidArguments}</pre>
                {/if}
              </div>
            </article>
          {:else if msg.role === 'toolResult'}
            <article class="msg tool-result">
              <details on:toggle={(event) => loadToolResultDetail(msg, event)}>
                <summary>
                  <span class="dot {msg.isError ? 'error' : 'done'}"></span>
                  <strong>{msg.toolName}</strong>
                  <span>{msg.isError ? '失败' : '完成'}</span>
                  <em>{msg.summary}</em>
                </summary>
                {#if msg.detailLoading}
                  <p class="pending-text">正在加载工具结果…</p>
                {:else if msg.detailError}
                  <p class="error-text">{msg.detailError}</p>
                {:else if msg.detailLoaded}
                  {#if msg.detail?.kind === 'bash' && msg.detail.bashResult}
                    <div class="bash-result">
                      <div class="bash-meta">
                        {#if msg.detail.bashResult.runtime}<span>{msg.detail.bashResult.runtime}</span>{/if}
                        {#if msg.detail.bashResult.cwd}<span>{msg.detail.bashResult.cwd}</span>{/if}
                        {#if msg.detail.bashResult.exitCode}
                          <strong class:failed={msg.detail.bashResult.exitCode !== '0'}>exit {msg.detail.bashResult.exitCode}</strong>
                        {/if}
                      </div>
                      {#if msg.detail.bashResult.prefix}
                        <p class="bash-note">{msg.detail.bashResult.prefix}</p>
                      {/if}
                      {#if msg.detail.bashResult.command}
                        <div class="bash-block">
                          <span>command</span>
                          <pre>{msg.detail.bashResult.command}</pre>
                        </div>
                      {/if}
                      {#if msg.detail.bashResult.stdout}
                        <div class="bash-block">
                          <span>stdout</span>
                          <pre class:empty={msg.detail.bashResult.stdout === '(no output)'}>{msg.detail.bashResult.stdout}</pre>
                        </div>
                      {/if}
                      {#if msg.detail.bashResult.stderr}
                        <div class="bash-block">
                          <span>stderr</span>
                          <pre class:empty={msg.detail.bashResult.stderr === '(no output)'}>{msg.detail.bashResult.stderr}</pre>
                        </div>
                      {/if}
                      {#if msg.detail.bashResult.note}
                        <p class="bash-note">{msg.detail.bashResult.note}</p>
                      {/if}
                    </div>
                  {:else if msg.detail?.kind === 'read' && msg.detail.readLines?.length}
                    <div class="read-result">
                      {#each msg.detail.readLines as line}
                        <div class="code-line">
                          <span>{line.number}</span>
                          <code>{line.text}</code>
                        </div>
                      {/each}
                    </div>
                  {:else if msg.detail?.kind === 'ls' && msg.detail.lsEntries?.length}
                    <div class="ls-result">
                      {#each msg.detail.lsEntries as entry}
                        <div class="ls-entry {entry.type}">
                          <span>{entry.type === 'dir' ? 'dir' : 'file'}</span>
                          <strong>{entry.name}</strong>
                          {#if entry.size}<em>{entry.size}</em>{/if}
                        </div>
                      {/each}
                    </div>
                  {:else if msg.detail?.kind === 'grep' && msg.detail.grepMatches?.matches?.length}
                    <div class="grep-result">
                      {#each msg.detail.grepMatches.matches as match}
                        <div class="grep-match">
                          <div><strong>{match.path}</strong><span>:{match.line}</span></div>
                          <code>{match.text}</code>
                        </div>
                      {/each}
                      {#if msg.detail.grepMatches.note}
                        <p>{msg.detail.grepMatches.note}</p>
                      {/if}
                    </div>
                  {:else if msg.detail?.content}
                    <pre>{msg.detail.content}</pre>
                  {/if}
                  {#if msg.detail?.images?.length}
                    <div class="msg-images">
                      {#each msg.detail.images as image}
                        <img src={image.dataUrl} alt={image.name} />
                      {/each}
                    </div>
                  {/if}
                {/if}
              </details>
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
      {#if imageUploads.length > 0}
        <div class="image-preview-row">
          {#each imageUploads as image, idx}
            <div class="image-preview">
              <img src={image.dataUrl} alt={image.name} />
              <span title={image.name}>{image.name}</span>
              <em>{formatImageSize(image.size)}</em>
              <button type="button" aria-label="移除图片" on:click={() => removeImage(idx)}>×</button>
            </div>
          {/each}
        </div>
      {/if}
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
        <input
          bind:this={imageInput}
          class="file-input"
          type="file"
          accept="image/png,image/jpeg,image/gif,image/webp"
          multiple
          on:change={handleImageSelect}
        />
        {#if selectedModelSupportsImages}
          <button
            type="button"
            class="icon-btn"
            disabled={!apiEnabled || busy}
            title="上传图片"
            aria-label="上传图片"
            on:click={() => imageInput?.click()}
          >
            📎
          </button>
        {/if}
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
            <option value={s.id}>{shortID(s.id)} · {s.workDir || '未知目录'}</option>
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
          disabled={busy || (!prompt.trim() && imageUploads.length === 0) || !apiEnabled || (isNewSession && !workDir.trim())}
          on:click={sendPrompt}
        >
          {busy ? '发送中' : '发送'}
        </button>
      </div>
    </div>
  </div>
</section>

<DirBrowser bind:open={showBrowser} on:select={onDirSelect} on:close={() => (showBrowser = false)} />
