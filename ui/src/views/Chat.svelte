<script>
  import { onDestroy, onMount, tick } from 'svelte';
  import { markdownToHTML } from '../lib/markdown.js';
  import { patchJSON, readSSE } from '../lib/api.js';
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
    refreshStatsSummary,
    getSessionMessages,
    getSessionToolResult,
    getSessionRunEvents,
    getSessionCapabilityEvents,
    sessionToolOptions,
    sessionToolsFor,
    setSessionTools,
    moveSessionTools
  } from '../lib/stores.js';
  import { shortID, toolStateClass, formatArgs } from '../lib/format.js';
  import DirBrowser from '../components/DirBrowser.svelte';
  import { t } from '../lib/preferences.js';

  let prompt = '';
  let messages = [];
  let busy = false;
  let chatAbort = null;
  let chatEvents = [];
  let sessionRunEvents = [];
  let sessionCapabilityEvents = [];
  let workDir = '';
  let sessionCreated = false;
  let showBrowser = false;
  let imageInput;
  let imageUploads = [];
  let chatScroll;
  let shouldFollowOutput = true;
  let scrollFrame = 0;
  let streamUsesTranscript = false;
  let sessionStreamAbort = null;
  let sessionStreamID = '';
  let sessionHistoryLoadedFor = '';
  let sessionStreamCompletedFor = '';
  let sessionStreamCursor = { entrySeq: 0, runSeq: 0, capabilitySeq: 0 };
  let optimisticRunEventID = '';
  let sessionToolKey = '__new__';
  let sessionTools = sessionToolsFor({}, sessionToolKey);

  const suggestions = [
    'chat.suggestion.projectSummary',
    'chat.suggestion.reviewChanges',
    'chat.suggestion.addTests',
    'chat.suggestion.fixTests',
    'chat.suggestion.refactor',
    'chat.suggestion.configAudit',
    'chat.suggestion.readme',
    'chat.suggestion.multiAgent'
  ];

  const toolToggles = [
    { key: 'webSearch', label: 'webSearch' },
    { key: 'browser', label: 'browser' },
    { key: 'a2aMaster', label: 'a2aMaster' },
    { key: 'delegate', label: 'delegate' },
    { key: 'multiAgent', label: 'multi-agent' }
  ];

  // Reset or load state when the selected session changes.
  let prevSession = $currentSession;
  onMount(() => {
    if ($currentSession) {
      loadSessionMessages($currentSession);
    }
  });
  onDestroy(() => {
    stopSessionStream();
  });

  $: {
    const nextSession = $currentSession;
    if (nextSession !== prevSession) {
      stopSessionStream();
      sessionHistoryLoadedFor = '';
      resetSessionStreamCursor();
      if (nextSession === '') {
        sessionCreated = false;
        workDir = '';
        messages = []; // new chat — no history
        chatEvents = []; // reset tool events
        sessionRunEvents = [];
        sessionCapabilityEvents = [];
        shouldFollowOutput = true;
      } else if (busy) {
        // The first streaming chunk can assign the newly-created session ID.
        // Do not reload persisted history mid-stream; it can replace the local
        // assistant placeholder and cause deltas to append to the user message.
        sessionCreated = true;
      } else {
        // Switched to an existing session — load its messages
        loadSessionMessages(nextSession);
      }
      prevSession = nextSession;
    }
  }

  async function loadSessionMessages(id) {
    try {
      const msgs = await getSessionMessages(id);
      if (id !== $currentSession) return;
      if (msgs && msgs.length > 0) {
        messages = msgs.map(normalizeSessionMessage).filter(Boolean);
      } else {
        messages = [];
      }
      chatEvents = []; // reset tool events for new session view
      await loadSessionEvents(id);
      sessionHistoryLoadedFor = id;
      updateSessionStreamCursorFromState();
      scrollChatToBottom({ force: true });
    } catch {
      if (id !== $currentSession) return;
      // Leave messages empty on error
      sessionHistoryLoadedFor = id;
      updateSessionStreamCursorFromState();
    }
    sessionCreated = true; // existing session, not "new"
  }

  $: activeSession = $sessions.find((s) => s.id === $currentSession);
  $: sessionToolKey = $currentSession || '__new__';
  $: sessionTools = sessionToolsFor($sessionToolOptions, sessionToolKey, activeSession || $features);
  $: recentTools = chatEvents.slice(-6).reverse();
  $: sessionEventSummary = buildSessionEventSummary(sessionRunEvents, sessionCapabilityEvents, activeSessionWorkDir, $selectedModel);
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
  $: {
    const tailID = $currentSession;
    const shouldTail = Boolean(
      tailID &&
      !busy &&
      activeSession?.running &&
      sessionHistoryLoadedFor === tailID &&
      sessionStreamCompletedFor !== tailID
    );
    if (shouldTail) {
      startSessionStream(tailID);
    } else if (sessionStreamID && sessionStreamID !== tailID) {
      stopSessionStream();
    } else if (!shouldTail && sessionStreamID) {
      stopSessionStream();
    }
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
      setError($t('chat.error.modelNoImages'));
      return;
    }
    if (isNewSession && !workDir.trim()) {
      setError($t('chat.error.needWorkDir'));
      return;
    }
    stopSessionStream();
    sessionStreamCompletedFor = '';
    busy = true;
    chatEvents = [];
    streamUsesTranscript = false;
    optimisticRunEventID = beginOptimisticRunEvent();
    clearBanners();

    // Add user message
    messages = [...messages, { role: 'user', content: outgoing, images: outgoingImages }];
    scrollChatToBottom({ force: true });
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
        x_tools: sessionTools,
        x_transcript: true,
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
      scrollChatToBottom({ force: true });
      await readSSE(res.body, handleStreamEvent);
      finishOptimisticRunEvent('completed');
      sessionCreated = true;
    } catch (err) {
      if (err?.name === 'AbortError') {
        finishOptimisticRunEvent('canceled');
        setNotice($t('chat.notice.stopped'));
      } else {
        finishOptimisticRunEvent('failed');
        setError(err);
      }
    } finally {
      busy = false;
      chatAbort = null;
      try { await refreshSessions(); } catch {
        // opportunistic
      }
      try { await refreshStatsSummary(); } catch {
        // opportunistic
      }
      if ($currentSession) {
        try { await loadSessionMessages($currentSession); } catch {
          // opportunistic
        }
      }
      optimisticRunEventID = '';
    }
  }

  function stop() {
    if (chatAbort) chatAbort.abort();
  }

  function resetSession() {
    currentSession.set('');
  }

  function handleChatScroll() {
    shouldFollowOutput = isChatNearBottom();
  }

  function isChatNearBottom() {
    if (!chatScroll) return true;
    const distance = chatScroll.scrollHeight - chatScroll.scrollTop - chatScroll.clientHeight;
    return distance < 96;
  }

  async function scrollChatToBottom({ force = false } = {}) {
    if (!chatScroll) return;
    if (!force && !shouldFollowOutput) return;
    await tick();
    if (!chatScroll) return;
    if (scrollFrame) cancelAnimationFrame(scrollFrame);
    scrollFrame = requestAnimationFrame(() => {
      scrollFrame = 0;
      if (!chatScroll) return;
      if (!force && !shouldFollowOutput) return;
      chatScroll.scrollTop = chatScroll.scrollHeight;
      shouldFollowOutput = true;
    });
  }

  async function updateToolOption(key, event) {
    const targetSession = $currentSession;
    const previousTools = sessionTools;
    const nextTools = {
      ...sessionTools,
      [key]: Boolean(event.currentTarget.checked)
    };
    setSessionTools(sessionToolKey, nextTools);
    if (!targetSession) return;
    try {
      const updated = await patchJSON(
        `/api/sessions/${encodeURIComponent(targetSession)}/capabilities`,
        nextTools
      );
      if (targetSession === $currentSession) {
        setSessionTools(targetSession, updated);
        await loadSessionEvents(targetSession);
      }
      await refreshSessions();
    } catch (err) {
      setSessionTools(targetSession, previousTools);
      setError(err);
    }
  }

  function onDirSelect(e) {
    workDir = e.detail.path;
    showBrowser = false;
  }

  async function handleImageSelect(event) {
    const files = Array.from(event.target.files || []);
    if (files.length === 0) return;
    if (!selectedModelSupportsImages) {
      setError($t('chat.error.modelNoImages'));
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
      throw new Error($t('chat.error.unsupportedFileType', { name: file.name }));
    }
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve({
        name: file.name,
        type: file.type,
        size: file.size,
        dataUrl: String(reader.result || '')
      });
      reader.onerror = () => reject(new Error($t('chat.error.imageReadFailed', { name: file.name })));
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

  async function loadSessionEvents(id) {
    if (!id) {
      sessionRunEvents = [];
      sessionCapabilityEvents = [];
      return;
    }
    try {
      const [runs, caps] = await Promise.all([
        getSessionRunEvents(id),
        getSessionCapabilityEvents(id)
      ]);
      if (id !== $currentSession) return;
      sessionRunEvents = runs || [];
      sessionCapabilityEvents = caps || [];
    } catch {
      if (id !== $currentSession) return;
      sessionRunEvents = [];
      sessionCapabilityEvents = [];
    }
  }

  function beginOptimisticRunEvent() {
    const id = `local-run-${Date.now()}`;
    const runID = `local_${Date.now()}`;
    const event = {
      id,
      runId: runID,
      sessionId: $currentSession || '',
      eventType: 'started',
      source: 'webui',
      status: 'running',
      model: $selectedModel || 'default',
      mode: activeSession?.mode || '',
      timestamp: new Date().toISOString(),
      data: {
        workDir: isNewSession ? workDir.trim() : activeSessionWorkDir,
        optimistic: true
      }
    };
    sessionRunEvents = [...sessionRunEvents.filter((item) => item.id !== id), event];
    return id;
  }

  function finishOptimisticRunEvent(status) {
    if (!optimisticRunEventID) return;
    const idx = sessionRunEvents.findIndex((item) => item.id === optimisticRunEventID);
    if (idx < 0) return;
    const eventType = status === 'failed' ? 'failed' : status === 'canceled' ? 'canceled' : 'finished';
    sessionRunEvents[idx] = {
      ...sessionRunEvents[idx],
      eventType,
      status,
      timestamp: new Date().toISOString()
    };
    sessionRunEvents = sessionRunEvents;
  }

  function resetSessionStreamCursor() {
    sessionStreamCursor = { entrySeq: 0, runSeq: 0, capabilitySeq: 0 };
  }

  function updateSessionStreamCursorFromState() {
    sessionStreamCursor = {
      entrySeq: maxSeq(messages),
      runSeq: maxSeq(sessionRunEvents),
      capabilitySeq: maxSeq(sessionCapabilityEvents)
    };
  }

  function maxSeq(items = []) {
    return items.reduce((max, item) => {
      const seq = Number(item?.seq || 0);
      return seq > max ? seq : max;
    }, 0);
  }

  function bumpSessionStreamCursorFromMessage(message) {
    const seq = Number(message?.seq || 0);
    if (seq > sessionStreamCursor.entrySeq) {
      sessionStreamCursor = { ...sessionStreamCursor, entrySeq: seq };
    }
  }

  function upsertSessionRunEvent(event) {
    if (!event?.id) return;
    if (event.eventType === 'started' || event.status === 'running') {
      sessionStreamCompletedFor = '';
    }
    const idx = sessionRunEvents.findIndex((item) => item.id === event.id);
    if (idx >= 0) {
      sessionRunEvents[idx] = { ...sessionRunEvents[idx], ...event };
      sessionRunEvents = sessionRunEvents;
    } else {
      sessionRunEvents = [...sessionRunEvents, event];
    }
    const seq = Number(event.seq || 0);
    if (seq > sessionStreamCursor.runSeq) {
      sessionStreamCursor = { ...sessionStreamCursor, runSeq: seq };
    }
  }

  function upsertSessionCapabilityEvent(event) {
    if (!event?.id) return;
    const idx = sessionCapabilityEvents.findIndex((item) => item.id === event.id);
    if (idx >= 0) {
      sessionCapabilityEvents[idx] = { ...sessionCapabilityEvents[idx], ...event };
      sessionCapabilityEvents = sessionCapabilityEvents;
    } else {
      sessionCapabilityEvents = [...sessionCapabilityEvents, event];
    }
    const seq = Number(event.seq || 0);
    if (seq > sessionStreamCursor.capabilitySeq) {
      sessionStreamCursor = { ...sessionStreamCursor, capabilitySeq: seq };
    }
  }

  function stopSessionStream() {
    if (sessionStreamAbort) {
      sessionStreamAbort.abort();
    }
    sessionStreamAbort = null;
    sessionStreamID = '';
  }

  function startSessionStream(id) {
    if (!id || busy) return;
    if (sessionStreamID === id && sessionStreamAbort) return;
    stopSessionStream();
    const cursor = { ...sessionStreamCursor };
    const abort = new AbortController();
    sessionStreamAbort = abort;
    sessionStreamID = id;
    consumeSessionStream(id, cursor, abort).finally(() => {
      if (sessionStreamAbort === abort) {
        sessionStreamAbort = null;
        sessionStreamID = '';
      }
    });
  }

  async function consumeSessionStream(id, cursor, abort) {
    const params = new URLSearchParams();
    if (cursor.entrySeq > 0) params.set('after_entry_seq', String(cursor.entrySeq));
    if (cursor.runSeq > 0) params.set('after_run_seq', String(cursor.runSeq));
    if (cursor.capabilitySeq > 0) params.set('after_capability_seq', String(cursor.capabilitySeq));
    const query = params.toString();
    try {
      const res = await fetch(`/api/sessions/${encodeURIComponent(id)}/stream${query ? `?${query}` : ''}`, {
        signal: abort.signal
      });
      if (!res.ok || !res.body) {
        const text = await res.text();
        let data = null;
        try { data = text ? JSON.parse(text) : null; } catch { data = null; }
        throw new Error(data?.error?.message || data?.error || data?.message || `${res.status} ${res.statusText}`);
      }
      await readSSE(res.body, (event) => handleSessionStreamEvent(id, event));
    } catch (err) {
      if (err?.name !== 'AbortError') {
        setError(err);
      }
    }
  }

  function handleSessionStreamEvent(id, event) {
    if (id !== $currentSession) return;
    if (event.data === '[DONE]') return;
    if (event.event === 'done') {
      sessionStreamCompletedFor = id;
      refreshSessions().catch(() => {});
      loadSessionMessages(id).catch(() => {});
      refreshStatsSummary().catch(() => {});
      return;
    }
    if (event.event === 'heartbeat') return;
    if (event.event === 'error') {
      try {
        const item = JSON.parse(event.data);
        setError(item?.error || event.data);
      } catch {
        setError(event.data);
      }
      return;
    }
    if (event.event === 'transcript') {
      try {
        const item = JSON.parse(event.data);
        applyTranscriptStreamEvent(item);
      } catch {
        // ignore malformed transcript frames
      }
      return;
    }
    if (event.event === 'run_event') {
      try {
        upsertSessionRunEvent(JSON.parse(event.data));
      } catch {
        // ignore malformed event frames
      }
      return;
    }
    if (event.event === 'capability_event') {
      try {
        upsertSessionCapabilityEvent(JSON.parse(event.data));
      } catch {
        // ignore malformed event frames
      }
    }
  }

  function buildSessionEventSummary(runEvents = [], capabilityEvents = [], workDir = '', model = '') {
    const runs = mergeRunEvents(runEvents);
    const currentModel = model && model !== 'default' ? model : '';
    const matchingRuns = runs.filter((run) => {
      if (!run.usage) return false;
      if (currentModel && run.model && run.model !== currentModel) return false;
      if (workDir && run.workDir && run.workDir !== workDir) return false;
      return true;
    });
    const totals = matchingRuns.reduce((acc, run) => {
      acc.promptTokens += run.usage.promptTokens;
      acc.completionTokens += run.usage.completionTokens;
      acc.totalTokens += run.usage.totalTokens;
      acc.cacheReadTokens += run.usage.cacheReadTokens;
      acc.cacheWriteTokens += run.usage.cacheWriteTokens;
      return acc;
    }, { promptTokens: 0, completionTokens: 0, totalTokens: 0, cacheReadTokens: 0, cacheWriteTokens: 0 });
    return {
      visible: runs.length > 0 || capabilityEvents.length > 0,
      lastRun: runs[0] || null,
      runCount: runs.length,
      capabilityCount: capabilityEvents.length,
      model: currentModel || runs[0]?.model || '',
      workDir: workDir || runs[0]?.workDir || '',
      matchingRuns: matchingRuns.length,
      ...totals
    };
  }

  function mergeRunEvents(events = []) {
    const byRun = new Map();
    for (const event of events) {
      const runId = event.runId || event.id || '';
      if (!runId) continue;
      const run = byRun.get(runId) || {
        runId,
        eventType: '',
        status: '',
        model: '',
        mode: '',
        workDir: '',
        timestamp: '',
        usage: null
      };
      const eventTime = Date.parse(event.timestamp || '') || 0;
      const runTime = Date.parse(run.timestamp || '') || 0;
      if (eventTime >= runTime) {
        run.timestamp = event.timestamp || run.timestamp;
        run.eventType = event.eventType || run.eventType;
        run.status = event.status || run.status;
      }
      if (event.model) run.model = event.model;
      if (event.mode) run.mode = event.mode;
      if (event.data?.workDir) run.workDir = event.data.workDir;
      const usage = normalizeRunUsage(event.data?.usage);
      if (usage) run.usage = usage;
      byRun.set(runId, run);
    }
    return Array.from(byRun.values())
      .sort((a, b) => (Date.parse(b.timestamp || '') || 0) - (Date.parse(a.timestamp || '') || 0));
  }

  function normalizeRunUsage(raw) {
    if (!raw || typeof raw !== 'object') return null;
    const promptTokens = readNumber(raw, ['prompt_tokens', 'promptTokens', 'inputTokens', 'input']);
    const completionTokens = readNumber(raw, ['completion_tokens', 'completionTokens', 'outputTokens', 'output']);
    const cacheReadTokens = readNumber(raw, ['cache_read_tokens', 'cacheReadTokens', 'cacheRead', 'cached_tokens']);
    const cacheWriteTokens = readNumber(raw, ['cache_write_tokens', 'cacheWriteTokens', 'cacheWrite']);
    const explicitTotal = readNumber(raw, ['total_tokens', 'totalTokens']);
    const totalTokens = explicitTotal || promptTokens + completionTokens;
    if (promptTokens === 0 && completionTokens === 0 && totalTokens === 0 && cacheReadTokens === 0 && cacheWriteTokens === 0) return null;
    return { promptTokens, completionTokens, totalTokens, cacheReadTokens, cacheWriteTokens };
  }

  function readNumber(source, keys) {
    for (const key of keys) {
      const value = Number(source?.[key]);
      if (Number.isFinite(value) && value > 0) return value;
    }
    return 0;
  }

  function sessionRunStateClass(run) {
    if (!run) return 'done';
    if (run.status === 'failed' || run.eventType === 'failed') return 'error';
    if (run.status === 'running' || run.eventType === 'started') return 'running';
    return 'done';
  }

  function sessionRunLabel(run) {
    if (!run) return $t('chat.sessionEvents.idle');
    if (run.status === 'running' || run.eventType === 'started') return $t('common.running');
    if (run.status === 'failed' || run.eventType === 'failed') return $t('common.failed');
    if (run.status === 'canceled' || run.eventType === 'canceled') return $t('chat.sessionEvents.canceled');
    return $t('common.completed');
  }

  function formatCompactTokens(value) {
    const n = Number(value) || 0;
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(n >= 10_000_000 ? 0 : 1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(n >= 10_000 ? 0 : 1)}K`;
    return String(n);
  }

  function formatCacheRate(summary) {
    if (!summary || summary.promptTokens <= 0) return '--';
    const pct = Math.min(100, Math.max(0, (summary.cacheReadTokens / summary.promptTokens) * 100));
    return `${Math.round(pct)}%`;
  }

  function compactPath(path) {
    if (!path) return '';
    const normalized = String(path).replace(/\/+$/, '');
    const parts = normalized.split('/').filter(Boolean);
    if (parts.length <= 2) return normalized || '/';
    return `.../${parts.slice(-2).join('/')}`;
  }

  function sessionEventTooltip(summary) {
    if (!summary) return '';
    const parts = [];
    if (summary.workDir) parts.push(summary.workDir);
    if (summary.model) parts.push(summary.model);
    parts.push(`${formatCompactTokens(summary.totalTokens)} tokens`);
    parts.push(`cache ${formatCacheRate(summary)}`);
    if (summary.lastRun?.timestamp) parts.push(formatEventTime(summary.lastRun.timestamp));
    return parts.join(' · ');
  }

  function formatEventTime(value) {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  function normalizeSessionMessage(message) {
    if (message.role === 'toolCall') {
      const plan = normalizePlan(message.plan || (message.toolName === 'plan' ? message.arguments : null));
      if (message.toolName === 'plan' && plan) {
        return {
          id: message.id,
          seq: message.seq,
          role: 'plan',
          toolCallId: message.toolCallId,
          toolName: message.toolName,
          plan
        };
      }
      return {
        id: message.id,
        seq: message.seq,
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
        id: message.id,
        seq: message.seq,
        role: 'toolResult',
        toolCallId: message.toolCallId,
        toolName: message.toolName || 'tool',
        summary: message.summary || $t('chat.tool.result'),
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
      id: message.id,
      seq: message.seq,
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
      case 'done': return $t('chat.plan.done');
      case 'running': return $t('chat.plan.running');
      case 'failed': return $t('chat.plan.failed');
      default: return $t('chat.plan.pending');
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
      msg.detailError = err instanceof Error ? err.message : String(err || $t('chat.tool.detailLoadFailed'));
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
      if (value.offset) details.push($t('chat.tool.read.offset', { offset: value.offset }));
      if (value.limit) details.push($t('chat.tool.read.limit', { limit: value.limit }));
      if (value.imageMode) details.push($t('chat.tool.read.imageMode', { mode: value.imageMode }));
      if (value.maxLongEdge) details.push($t('chat.tool.read.maxLongEdge', { value: value.maxLongEdge }));
      if (value.crop) details.push($t('chat.tool.read.crop', { value: `${value.crop.width || 0}x${value.crop.height || 0}+${value.crop.x || 0}+${value.crop.y || 0}` }));
      return {
        kind: 'read',
        label: $t('chat.tool.read.label'),
        target: value.path || $t('chat.tool.read.missing'),
        details,
        raw: args,
        invalidArguments
      };
    }
    if (name === 'ls') {
      return {
        kind: 'ls',
        label: $t('chat.tool.ls.label'),
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
      if (value.maxResults) details.push($t('chat.tool.grep.maxResults', { count: value.maxResults }));
      return {
        kind: 'grep',
        label: $t('chat.tool.grep.label'),
        target: value.pattern || $t('chat.tool.grep.missing'),
        details,
        raw: args,
        invalidArguments
      };
    }
    if (name === 'find') {
      const details = [];
      if (value.path) details.push($t('chat.tool.find.path', { path: value.path }));
      if (value.maxDepth !== undefined && value.maxDepth !== null) {
        details.push($t('chat.tool.find.maxDepth', { depth: value.maxDepth }));
      }
      if (value.maxResults !== undefined && value.maxResults !== null) {
        details.push($t('chat.tool.find.maxResults', { count: value.maxResults }));
      }
      return {
        kind: 'find',
        label: $t('chat.tool.find.label'),
        target: value.pattern || $t('chat.tool.find.missing'),
        details,
        pattern: value.pattern || '',
        path: value.path || '.',
        maxDepth: value.maxDepth ?? '',
        maxResults: value.maxResults ?? '',
        raw: args,
        invalidArguments
      };
    }
    if (name === 'bash') {
      const details = [];
      if (value.async) details.push($t('chat.tool.bash.async'));
      if (value.timeout !== undefined && value.timeout !== null) {
        details.push(Number(value.timeout) === 0 ? $t('chat.tool.bash.noTimeout') : $t('chat.tool.bash.timeout', { seconds: value.timeout }));
      }
      return {
        kind: 'bash',
        label: $t('chat.tool.bash.label'),
        target: value.command || $t('chat.tool.bash.missing'),
        details,
        raw: args,
        invalidArguments
      };
    }
    if (name === 'edit') {
      const edits = Array.isArray(value.edits)
        ? value.edits
          .filter(isPlainObject)
          .map((item, index) => {
            const oldText = String(item.oldText ?? '');
            const newText = String(item.newText ?? '');
            return {
              index: index + 1,
              oldText,
              newText,
              oldLines: countTextLines(oldText),
              newLines: countTextLines(newText)
            };
          })
        : [];
      return {
        kind: 'edit',
        label: $t('chat.tool.edit.label'),
        target: value.path || $t('chat.tool.edit.missing'),
        details: [edits.length === 1 ? $t('chat.tool.edit.oneEdit') : $t('chat.tool.edit.manyEdits', { count: edits.length })],
        edits,
        raw: args,
        invalidArguments
      };
    }
    if (name === 'write') {
      const content = typeof value.content === 'string' ? value.content : '';
      const lines = countTextLines(content);
      const chars = content.length;
      return {
        kind: 'write',
        label: $t('chat.tool.write.label'),
        target: value.path || $t('chat.tool.write.missing'),
        details: [
          $t('chat.tool.write.lines', { count: lines }),
          $t('chat.tool.write.chars', { count: chars })
        ],
        content,
        lines,
        chars,
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

  function countTextLines(text = '') {
    if (!text) return 0;
    return String(text).split('\n').length;
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

  function handleToolStatusEvent(item) {
    chatEvents = [...chatEvents.slice(-49), { type: 'tool', ...item }];
    if (!item?.tool || !item?.status) {
      scrollChatToBottom();
      return;
    }
    if (item.status === 'running') {
      upsertStreamingToolCall(item);
      scrollChatToBottom();
      return;
    }
    if (item.status === 'completed' || item.status === 'failed') {
      upsertStreamingToolResult(item);
    }
    scrollChatToBottom();
  }

  function upsertTranscriptMessage(next) {
    if (!next) return;
    bumpSessionStreamCursorFromMessage(next);
    if (next.id) {
      const existing = messages.findIndex((m) => m.id === next.id);
      if (existing >= 0) {
        messages[existing] = { ...messages[existing], ...next };
        messages = messages;
        scrollChatToBottom();
        return;
      }
    }
    if (next.role === 'assistant') {
      const last = messages[messages.length - 1];
      if (next.id && last?.role === 'assistant' && !last.id) {
        messages[messages.length - 1] = { ...last, ...next };
        messages = messages;
        scrollChatToBottom();
        return;
      }
      messages = [...messages, next];
      scrollChatToBottom();
      return;
    }
    if (next.role === 'toolResult') {
      upsertTranscriptToolResult(next);
      scrollChatToBottom();
      return;
    }
    if (next.role !== 'toolCall' && next.role !== 'plan') {
      messages = [...messages, next];
      scrollChatToBottom();
      return;
    }
    upsertTranscriptToolCall(next);
    scrollChatToBottom();
  }

  function upsertTranscriptToolCall(next) {
    const toolCallId = next.toolCallId || '';
    const idx = toolCallId ? messages.findIndex((m) => m.toolCallId === toolCallId && (m.role === 'toolCall' || m.role === 'plan')) : -1;
    if (idx >= 0) {
      messages[idx] = { ...messages[idx], ...next };
      messages = messages;
      return;
    }

    const last = messages[messages.length - 1];
    if (last?.role === 'assistant' && !last.content && !last.images?.length) {
      messages = messages.slice(0, -1);
    }
    messages = [...messages, next];
  }

  function upsertTranscriptToolResult(next) {
    const toolCallId = next.toolCallId || '';
    const existing = toolCallId ? messages.findIndex((m) => m.role === 'toolResult' && m.toolCallId === toolCallId) : -1;
    if (existing >= 0) {
      messages[existing] = { ...messages[existing], ...next };
      messages = messages;
      return;
    }

    const callIdx = toolCallId ? messages.findIndex((m) => m.toolCallId === toolCallId && (m.role === 'toolCall' || m.role === 'plan')) : -1;
    if (callIdx >= 0) {
      messages = [
        ...messages.slice(0, callIdx + 1),
        next,
        ...messages.slice(callIdx + 1)
      ];
      return;
    }

    const last = messages[messages.length - 1];
    if (last?.role === 'assistant' && !last.content && !last.images?.length) {
      messages = messages.slice(0, -1);
    }
    messages = [...messages, next];
  }

  function upsertStreamingToolCall(item) {
    const message = {
      role: 'toolCall',
      toolCallId: item.toolCallId || '',
      toolName: item.tool,
      arguments: item.args
    };
    upsertTranscriptMessage(normalizeSessionMessage(message));
  }

  function upsertStreamingToolResult(item) {
    if (item.tool === 'plan' && item.status !== 'failed' && !item.isError) return;
    const message = {
      role: 'toolResult',
      toolCallId: item.toolCallId || '',
      toolName: item.tool,
      summary: item.summary || (item.status === 'failed' ? $t('chat.tool.failed') : $t('chat.tool.completed')),
      isError: item.isError || item.status === 'failed',
      hasDetail: Boolean(item.hasDetail && item.toolCallId)
    };
    upsertTranscriptMessage(normalizeSessionMessage(message));
  }

  function textFromContents(contents = []) {
    return contents
      .filter((block) => block.type === 'text' && block.text)
      .map((block) => block.text)
      .join('\n');
  }

  function handleStreamEvent(event) {
    if (event.data === '[DONE]') return;
    if (event.event === 'transcript') {
      try {
        const item = JSON.parse(event.data);
        streamUsesTranscript = true;
        applyTranscriptStreamEvent(item);
      } catch {
        // ignore malformed transcript frames
      }
      return;
    }
    if (event.event === 'tool_status') {
      if (streamUsesTranscript) return;
      try {
        const item = JSON.parse(event.data);
        handleToolStatusEvent(item);
      } catch {
        chatEvents = [...chatEvents.slice(-49), { type: 'tool', status: 'unknown', raw: event.data }];
        scrollChatToBottom();
      }
      return;
    }
    try {
      const chunk = JSON.parse(event.data);
      if (chunk?.x_session_id) {
        if (!$currentSession) {
          moveSessionTools('__new__', chunk.x_session_id);
        }
        currentSession.set(chunk.x_session_id);
      }
      const delta = chunk?.choices?.[0]?.delta?.content;
      if (delta && !streamUsesTranscript) {
        appendAssistantDelta(delta);
      }
    } catch {
      // ignore malformed frames
    }
  }

  function applyTranscriptStreamEvent(item) {
    if (item?.x_session_id) {
      if (!$currentSession) {
        moveSessionTools('__new__', item.x_session_id);
      }
      currentSession.set(item.x_session_id);
    }
    const message = item?.message;
    if (!message) return;
    if (item.type === 'assistant_delta') {
      appendAssistantDelta(message.content || '');
      return;
    }
    if (item.type === 'message') {
      upsertTranscriptMessage(normalizeSessionMessage(message));
    }
  }

  function appendAssistantDelta(delta) {
    if (!delta) return;
    const last = messages[messages.length - 1];
    if (!last || last.role !== 'assistant') {
      messages = [...messages, { role: 'assistant', content: delta }];
    } else {
      last.content += delta;
      messages = messages;
    }
    scrollChatToBottom();
  }
</script>

<section class="chat-view">
  <div class="chat-scroll" bind:this={chatScroll} on:scroll={handleChatScroll}>
    {#if messages.length === 0 && !busy}
      <div class="welcome">
        <h2>{$t('chat.welcome')}</h2>
        <div class="suggestions">
          {#each suggestions as key}
            <button
              type="button"
              class="chip"
              disabled={!apiEnabled || (isNewSession && !workDir.trim())}
              on:click={() => pick($t(key))}
            >
              {$t(key)}
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
                <strong>{$t('chat.you')}</strong>
                <span>{shortID($currentSession)}</span>
              </div>
              <p>{msg.content}</p>
              {#if msg.images?.length}
                <div class="msg-images">
                  {#each msg.images as image}
                    <img src={image.dataUrl} alt={image.name} on:load={() => scrollChatToBottom()} />
                  {/each}
                </div>
              {/if}
            </article>
          {:else if msg.role === 'assistant'}
            <article class="msg assistant">
              <div class="meta">
                <strong>MothX</strong>
                <span>{busy && idx === messages.length - 1 ? $t('chat.generating') : $t('common.completed')}</span>
              </div>
              {#if msg.content}
                <div class="markdown">{@html markdownToHTML(msg.content)}</div>
              {:else if busy && idx === messages.length - 1}
                <p class="pending-text">{$t('chat.waitingModel')}</p>
              {/if}
            </article>
          {:else if msg.role === 'plan'}
            <article class="msg plan-card">
              <div class="meta">
                <strong>{$t('chat.plan')}</strong>
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
                <strong>{$t('chat.toolCall')}</strong>
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
                {#if msg.callView?.kind === 'edit' && msg.callView.edits?.length}
                  <div class="edit-call">
                    {#each msg.callView.edits as edit}
                      <section class="edit-block">
                        <div class="edit-block-head">
                          <strong>{$t('chat.tool.edit.editNumber', { number: edit.index })}</strong>
                          <span>{$t('chat.tool.edit.lineChange', { old: edit.oldLines, next: edit.newLines })}</span>
                        </div>
                        <div class="edit-columns">
                          <div class="edit-pane old">
                            <span>{$t('chat.tool.edit.oldText')}</span>
                            <pre class:empty={edit.oldText === ''}>{edit.oldText || $t('chat.tool.edit.empty')}</pre>
                          </div>
                          <div class="edit-pane new">
                            <span>{$t('chat.tool.edit.newText')}</span>
                            <pre class:empty={edit.newText === ''}>{edit.newText || $t('chat.tool.edit.empty')}</pre>
                          </div>
                        </div>
                      </section>
                    {/each}
                  </div>
                {:else if msg.callView?.kind === 'write'}
                  <div class="write-call">
                    <div class="write-call-head">
                      <strong>{$t('chat.tool.write.preview')}</strong>
                      <span>{$t('chat.tool.write.summary', { lines: msg.callView.lines, chars: msg.callView.chars })}</span>
                    </div>
                    <span>{$t('chat.tool.write.content')}</span>
                    <pre class:empty={msg.callView.content === ''}>{msg.callView.content || $t('chat.tool.edit.empty')}</pre>
                  </div>
                {:else if msg.callView?.kind === 'find'}
                  <div class="find-call">
                    <div class="find-row">
                      <span>{$t('chat.tool.find.pattern')}</span>
                      <code>{msg.callView.pattern || $t('chat.tool.find.missing')}</code>
                    </div>
                    <div class="find-row">
                      <span>{$t('chat.tool.find.searchPath')}</span>
                      <code>{msg.callView.path}</code>
                    </div>
                    {#if msg.callView.maxDepth !== ''}
                      <div class="find-row">
                        <span>{$t('chat.tool.find.depth')}</span>
                        <code>{msg.callView.maxDepth}</code>
                      </div>
                    {/if}
                    {#if msg.callView.maxResults !== ''}
                      <div class="find-row">
                        <span>{$t('chat.tool.find.resultLimit')}</span>
                        <code>{msg.callView.maxResults}</code>
                      </div>
                    {/if}
                  </div>
                {/if}
                {#if msg.callView?.kind !== 'generic' && msg.arguments}
                  <details class="tool-raw">
                    <summary>{$t('chat.argsJson')}</summary>
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
                  <span>{msg.isError ? $t('common.failed') : $t('common.completed')}</span>
                  <em>{msg.summary}</em>
                </summary>
                {#if msg.detailLoading}
                  <p class="pending-text">{$t('chat.loadingToolResult')}</p>
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
                        <img src={image.dataUrl} alt={image.name} on:load={() => scrollChatToBottom()} />
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
            <div class="tf-head"><span>{$t('chat.toolEvents')}</span><strong>{chatEvents.length}</strong></div>
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
        {#if sessionEventSummary.visible}
          <aside class="session-event-strip" title={sessionEventTooltip(sessionEventSummary)}>
            <span class="dot {sessionRunStateClass(sessionEventSummary.lastRun)}"></span>
            <strong>{sessionRunLabel(sessionEventSummary.lastRun)}</strong>
            {#if sessionEventSummary.workDir}<span class="path">{compactPath(sessionEventSummary.workDir)}</span>{/if}
            {#if sessionEventSummary.model}<span>{sessionEventSummary.model}</span>{/if}
            <span class="metric">{$t('chat.sessionEvents.tokens', { tokens: formatCompactTokens(sessionEventSummary.totalTokens) })}</span>
            <span class="metric">{$t('chat.sessionEvents.cache', { rate: formatCacheRate(sessionEventSummary) })}</span>
            {#if sessionEventSummary.capabilityCount > 0}
              <span>{$t('chat.sessionEvents.capabilities', { count: sessionEventSummary.capabilityCount })}</span>
            {/if}
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
            <button type="button" class="ghost sm" on:click={() => (workDir = '')}>{$t('chat.clearWorkDir')}</button>
          </span>
        {/if}
        <button type="button" class="dir-btn" on:click={() => (showBrowser = true)}>
          <span class="ico">📂</span>
          {workDir ? $t('chat.changeWorkDir') : $t('chat.selectWorkDir')}
        </button>
      </div>
    {:else if $currentSession}
      <div class="composer-session-info">
        <span class="session-badge">{$t('chat.session')}</span>
        <span class="session-id">{shortID($currentSession)}</span>
        {#if activeSessionWorkDir}<span class="session-dir">{activeSessionWorkDir}</span>{/if}
        <button type="button" class="ghost sm" on:click={resetSession}>{$t('chat.newSession')}</button>
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
              <button type="button" aria-label={$t('chat.removeImage')} on:click={() => removeImage(idx)}>×</button>
            </div>
          {/each}
        </div>
      {/if}
      <textarea
        bind:value={prompt}
        on:keydown={handleKeydown}
        placeholder={!apiEnabled ? $t('chat.apiDisabled') : (isNewSession && !workDir.trim()) ? $t('chat.error.needWorkDir') : $t('chat.messagePlaceholder')}
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
            title={$t('chat.uploadImage')}
            aria-label={$t('chat.uploadImage')}
            on:click={() => imageInput?.click()}
          >
            📎
          </button>
        {/if}
        <select
          bind:value={$selectedModel}
          disabled={!apiEnabled || modelOptions.length === 0}
          aria-label={$t('chat.selectModel')}
        >
          {#if modelOptions.length === 0}
            <option value="default">{$t('chat.defaultModel')}</option>
          {:else}
            {#each modelOptions as m}
              <option value={m.id}>{m.id}</option>
            {/each}
          {/if}
        </select>
        <div class="tool-toggles" aria-label={$t('chat.tools')}>
          {#each toolToggles as item}
            <label class="tool-toggle" title={$t(`chat.toolToggle.${item.key}`)}>
              <input
                type="checkbox"
                checked={sessionTools[item.key]}
                disabled={!apiEnabled || busy}
                on:change={(event) => updateToolOption(item.key, event)}
              />
              <span>{item.label}</span>
            </label>
          {/each}
        </div>
      </div>
      <div class="right">
        {#if busy}
          <button type="button" class="ghost" on:click={stop}>{$t('common.stop')}</button>
        {/if}
        <button
          type="button"
          class="primary"
          disabled={busy || (!prompt.trim() && imageUploads.length === 0) || !apiEnabled || (isNewSession && !workDir.trim())}
          on:click={sendPrompt}
        >
          {busy ? $t('chat.sending') : $t('chat.send')}
        </button>
      </div>
    </div>
  </div>
</section>

<DirBrowser bind:open={showBrowser} on:select={onDirSelect} on:close={() => (showBrowser = false)} />
