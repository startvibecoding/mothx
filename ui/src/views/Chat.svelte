<script>
  import { onDestroy, onMount, tick } from 'svelte';
  import { get } from 'svelte/store';
  import { markdownToHTML } from '../lib/markdown.js';
  import { readSSE, postJSON } from '../lib/api.js';
  import { approvalSessionID, approvalRequestOwnership, approvalHistoryFromRunEvents, applyApprovalRequestToRuntime } from '../lib/approval.js';
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
    resetSelectedModelToDefault,
    getSessionMessages,
    getSessionToolResult,
    getSessionSubAgents,
    getSessionSubAgentMessages,
    getSessionRunEvents,
    getSessionCapabilityEvents,
    getSessionRuntime,
    patchSessionRuntime,
    sessionRuntime,
    activeApproval,
    toolEvents,
    sessionToolOptions,
    sessionToolsFor,
    setSessionTools,
    moveSessionTools
  } from '../lib/stores.js';
  import { shortID, toolStateClass, formatArgs } from '../lib/format.js';
  import {
    sessionRunStates,
    ensureSessionState,
    getSessionState,
    updateSessionState,
    isCompletionActive,
    registerCompletion,
    markCompletion,
    clearCompletion,
    abortCompletion,
    registerObserver,
    clearObserver,
    stopObserver,
    eventBelongsToSession
  } from '../lib/session-runs.js';
  import DirBrowser from '../components/DirBrowser.svelte';
  import { t } from '../lib/preferences.js';

  let prompt = '';
  let messages = [];
  let busy = false;
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
  let sessionHistoryLoadedFor = '';
  let sessionStreamCompletedFor = '';
  let sessionStreamCursor = { entrySeq: 0, runSeq: 0, capabilitySeq: 0 };
  let optimisticRunEventID = '';
  let sessionToolKey = '__new__';
  let sessionTools = sessionToolsFor({}, sessionToolKey);
  let subAgents = [];
  let subAgentTranscripts = {};
  let showSubAgentModal = false;
  let selectedSubAgentID = '';
  let subAgentModalMessages = [];
  let subAgentModalLoading = false;
  let subAgentModalError = '';
  let subAgentRefreshTimer = 0;
  let sessionRuntimeValue = null;
  let newSessionMode = 'yolo';
  let runtimeUpdating = false;
  let runtimeControls;
  let showRuntimePanel = false;
  let showApprovalCenter = false;
  let selectedApprovalID = '';
  let approvalSubmitting = false;
  let approvalHistory = [];

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
    { key: 'multiAgent', label: 'multi-agent' },
    { key: 'workflows', label: 'workflow' }
  ];

  // Reset or load state when the selected session changes.
  let prevSession = $currentSession;
  onMount(() => {
    const handleRuntimeOutsidePointer = (event) => {
      if (showRuntimePanel && runtimeControls && !runtimeControls.contains(event.target)) {
        showRuntimePanel = false;
      }
    };
    document.addEventListener('pointerdown', handleRuntimeOutsidePointer);
    if ($currentSession) {
      loadSessionMessages($currentSession);
    }
    return () => document.removeEventListener('pointerdown', handleRuntimeOutsidePointer);
  });
  onDestroy(() => {
    for (const state of Object.values(get(sessionRunStates))) {
      state.observer?.controller?.abort();
      state.completion?.controller?.abort();
    }
    if (subAgentRefreshTimer) clearTimeout(subAgentRefreshTimer);
  });

  $: {
    const nextSession = $currentSession;
    if (nextSession !== prevSession) {
      if (prevSession) persistLocalSessionState(prevSession);
      if (prevSession && prevSession !== nextSession) stopObserver(prevSession);
      sessionHistoryLoadedFor = '';
      subAgents = [];
      subAgentTranscripts = {};
      closeSubAgentModal();
      activeApproval.set(null);
      selectedApprovalID = '';
      if (nextSession === '') {
        sessionCreated = false;
        workDir = '';
        messages = []; // new chat — no history
        chatEvents = []; // reset tool events
        sessionRunEvents = [];
        sessionCapabilityEvents = [];
        resetSelectedModelToDefault();
        shouldFollowOutput = true;
      } else {
        const cached = getSessionState(nextSession);
        if (cached.historyLoaded || isCompletionActive(cached)) {
          restoreLocalSessionState(cached);
          sessionCreated = true;
          scrollChatToBottom({ force: true });
        } else {
          loadSessionMessages(nextSession);
        }
      }
      prevSession = nextSession;
    }
  }

  function persistLocalSessionState(id) {
    if (!id) return;
    updateSessionState(id, (state) => ({
      ...state,
      messages,
      toolEvents: chatEvents,
      runEvents: sessionRunEvents,
      capabilityEvents: sessionCapabilityEvents,
      runtime: sessionRuntimeValue,
      pendingApprovals: sessionRuntimeValue?.pendingApprovals || [],
      cursor: sessionStreamCursor,
      historyLoaded: sessionHistoryLoadedFor === id || state.historyLoaded,
      streamCompleted: sessionStreamCompletedFor === id,
      streamUsesTranscript,
      optimisticRunEventID,
      subAgents,
      subAgentTranscripts
    }));
  }

  function restoreLocalSessionState(state) {
    messages = state?.messages || [];
    chatEvents = state?.toolEvents || [];
    sessionRunEvents = state?.runEvents || [];
    sessionCapabilityEvents = state?.capabilityEvents || [];
    sessionRuntimeValue = state?.runtime || null;
    sessionRuntime.set(sessionRuntimeValue);
    sessionStreamCursor = state?.cursor || { entrySeq: 0, runSeq: 0, capabilitySeq: 0 };
    sessionHistoryLoadedFor = state?.historyLoaded ? state.sessionId : '';
    sessionStreamCompletedFor = state?.streamCompleted ? state.sessionId : '';
    streamUsesTranscript = Boolean(state?.streamUsesTranscript);
    optimisticRunEventID = state?.optimisticRunEventID || '';
    subAgents = state?.subAgents || [];
    subAgentTranscripts = state?.subAgentTranscripts || {};
    approvalHistory = approvalHistoryFromRunEvents(sessionRunEvents);
  }

  function withSessionProjection(id, callback) {
    if (!id || typeof callback !== 'function') return;
    const selectedID = $currentSession;
    if (selectedID === id) {
      callback();
      persistLocalSessionState(id);
      return;
    }
    const selectedSnapshot = selectedID ? getSessionState(selectedID) : null;
    restoreLocalSessionState(getSessionState(id));
    callback();
    persistLocalSessionState(id);
    if (selectedSnapshot) restoreLocalSessionState(selectedSnapshot);
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
      await loadSessionRuntime(id);
      sessionHistoryLoadedFor = id;
      updateSessionStreamCursorFromState();
      persistLocalSessionState(id);
      scrollChatToBottom({ force: true });
    } catch {
      if (id !== $currentSession) return;
      // Leave messages empty on error
      sessionHistoryLoadedFor = id;
      updateSessionStreamCursorFromState();
      persistLocalSessionState(id);
    }
    sessionCreated = true; // existing session, not "new"
  }

  $: activeSession = $sessions.find((s) => s.id === $currentSession);
  $: selectedRunState = $currentSession ? $sessionRunStates[$currentSession] : null;
  $: busy = isCompletionActive(selectedRunState) || selectedRunState?.runtime?.activeRun?.status === 'running';
  $: runtimeMode = sessionRuntimeValue?.mode || activeSession?.mode || (!$currentSession ? newSessionMode : 'yolo');
  $: pendingApprovalCount = (sessionRuntimeValue?.pendingApprovals || []).length;
  $: approvalToolViewValue = approvalToolView(selectedApproval);
  $: selectedApproval = (sessionRuntimeValue?.pendingApprovals || []).find((approval) => approval.approvalId === selectedApprovalID) || $activeApproval || null;
  $: runtimeActiveRun = sessionRuntimeValue?.activeRun;
  $: sessionToolKey = $currentSession || '__new__';
  $: sessionTools = sessionToolsFor($sessionToolOptions, sessionToolKey, activeSession || $features);
  $: availableToolToggles = toolToggles.filter(isToolToggleVisible);
  $: visibleSessionTools = filterHiddenSessionTools(sessionTools, $features);
  $: recentTools = chatEvents.slice(-6).reverse();
  $: sessionEventSummary = buildSessionEventSummary(sessionRunEvents, sessionCapabilityEvents, activeSessionWorkDir, $selectedModel);
  $: subAgentSummary = buildSubAgentSummary(subAgents);
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
      !isCompletionActive($sessionRunStates[tailID]) &&
      activeSession?.running &&
      sessionHistoryLoadedFor === tailID &&
      sessionStreamCompletedFor !== tailID
    );
    if (shouldTail) {
      startSessionStream(tailID);
    } else if (!shouldTail && tailID) {
      stopObserver(tailID);
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
    const creatingSession = isNewSession;
    if (creatingSession && !workDir.trim()) {
      setError($t('chat.error.needWorkDir'));
      return;
    }

    const sessionID = $currentSession || newWebUISessionID();
    const existingState = getSessionState(sessionID);
    if (isCompletionActive(existingState) || existingState.runtime?.activeRun?.status === 'running') {
      setError('This session already has an active run.');
      return;
    }
    if (!$currentSession) {
      ensureSessionState(sessionID);
      moveSessionTools('__new__', sessionID);
      currentSession.set(sessionID);
    }
    stopObserver(sessionID);
    sessionStreamCompletedFor = '';
    chatEvents = [];
    streamUsesTranscript = false;
    clearBanners();

    messages = [...messages, { role: 'user', content: outgoing, images: outgoingImages }];
    scrollChatToBottom({ force: true });
    prompt = '';
    imageUploads = [];
    if (imageInput) imageInput.value = '';

    const controller = new AbortController();
    registerCompletion(sessionID, controller);
    optimisticRunEventID = beginOptimisticRunEvent(sessionID);
    persistLocalSessionState(sessionID);
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
        x_session_id: sessionID,
        x_working_dir: creatingSession ? workDir.trim() : activeSessionWorkDir,
        x_mode: creatingSession ? newSessionMode : undefined,
        x_tools: visibleSessionTools,
        x_transcript: true,
        messages: requestMessages
      });
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
        signal: controller.signal
      });
      if (!res.ok || !res.body) {
        const text = await res.text();
        let data = null;
        try { data = text ? JSON.parse(text) : null; } catch { data = null; }
        throw new Error(data?.error?.message || data?.error || data?.message || `${res.status} ${res.statusText}`);
      }
      markCompletion(sessionID, 'running');
      withSessionProjection(sessionID, () => {
        messages = [...messages, { role: 'assistant', content: '' }];
        scrollChatToBottom({ force: true });
      });
      await readSSE(res.body, (event) => handleStreamEvent(sessionID, event));
      withSessionProjection(sessionID, () => finishOptimisticRunEvent('completed'));
      markCompletion(sessionID, 'completed');
      sessionCreated = true;
    } catch (err) {
      const canceled = err?.name === 'AbortError';
      withSessionProjection(sessionID, () => finishOptimisticRunEvent(canceled ? 'canceled' : 'failed'));
      markCompletion(sessionID, canceled ? 'canceled' : 'failed', canceled ? '' : err);
      if (sessionID === $currentSession) {
        if (canceled) setNotice($t('chat.notice.stopped'));
        else setError(err);
      }
    } finally {
      clearCompletion(sessionID, controller);
      try { await refreshSessions(); } catch {
        // opportunistic
      }
      try { await refreshStatsSummary(); } catch {
        // opportunistic
      }
      if (sessionID === $currentSession) {
        try { await loadSessionMessages(sessionID); } catch {
          // opportunistic
        }
        try { await loadSubAgents(sessionID); } catch {
          // opportunistic
        }
      }
      updateSessionState(sessionID, (state) => ({ ...state, optimisticRunEventID: '' }));
      if (sessionID === $currentSession) optimisticRunEventID = '';
    }
  }

  function newWebUISessionID() {
    if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID();
    return `webui-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  }

  function stop() {
    if ($currentSession) abortCompletion($currentSession);
  }

  function resetSession() {
    resetSelectedModelToDefault();
    newSessionMode = 'yolo';
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
    const nextTools = filterHiddenSessionTools({
      ...sessionTools,
      [key]: Boolean(event.currentTarget.checked)
    }, $features);
    setSessionTools(sessionToolKey, nextTools);
    if (!targetSession) return;
    try {
      const updated = await patchSessionRuntime(
        targetSession,
        { capabilities: { [key]: Boolean(event.currentTarget.checked) } }
      );
      if (targetSession === $currentSession) {
        sessionRuntime.set(updated);
        sessionRuntimeValue = updated;
        setSessionTools(targetSession, { ...nextTools, [key]: Boolean(updated?.capabilities?.[key]?.enabled) });
        await loadSessionEvents(targetSession);
      }
      await refreshSessions();
    } catch (err) {
      setSessionTools(targetSession, previousTools);
      setError(err);
    }
  }

  function isToolToggleVisible(item) {
    if (item?.key === 'webSearch' || item?.key === 'a2aMaster') {
      return $features[item.key] === true;
    }
    return true;
  }

  function filterHiddenSessionTools(tools = {}, featureState = {}) {
    return {
      ...tools,
      webSearch: featureState.webSearch === true && tools.webSearch === true,
      a2aMaster: featureState.a2aMaster === true && tools.a2aMaster === true
    };
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


  function handleApprovalRequest(data, boundSessionID = $currentSession) {
    try {
      const item = typeof data === 'string' ? JSON.parse(data) : data;
      if (!item?.approvalId || !eventBelongsToSession(boundSessionID, item)) return;
      const ownership = approvalRequestOwnership(boundSessionID, item);
      const next = applyApprovalRequestToRuntime(sessionRuntimeValue, boundSessionID, item);
      if (!ownership.belongs) return;
      sessionRuntimeValue = next;
      sessionRuntime.set(sessionRuntimeValue);
      if (boundSessionID === $currentSession) {
        activeApproval.set(item);
        selectedApprovalID = item.approvalId;
        showApprovalCenter = true;
      }
    } catch {
      // ignore malformed approval frames
    }
  }

  function recordApprovalResolution(resolution, sessionID) {
    if (!resolution?.approvalId || sessionID !== $currentSession) return;
    const id = `approval-resolution-${resolution.approvalId}-${resolution.status || 'resolved'}`;
    upsertSessionRunEvent({
      id,
      sessionId: sessionID,
      eventType: 'approval_resolved',
      status: resolution.status || 'resolved',
      timestamp: resolution.timestamp || new Date().toISOString(),
      data: { resolution }
    });
  }

  function approvalToolView(approval) {
    const tool = approval?.tool || {};
    return buildToolCallView(tool.name || '', tool.args || tool.details || {});
  }

  function approvalBashCommand(approval) {
    const args = approval?.tool?.args || {};
    return approval?.tool?.details?.command || args.command || args.cmd || '';
  }

  function approvalBashWorkDir(approval) {
    return approval?.tool?.details?.workDir || approval?.context?.workDir || '';
  }

  async function respondApproval(approval, action) {
    const sessionID = approvalSessionID(approval, $currentSession);
    if (!approval?.approvalId || !sessionID || approvalSubmitting) return;
    approvalSubmitting = true;
    try {
      const resolved = await postJSON(`/api/sessions/${encodeURIComponent(sessionID)}/approvals/${encodeURIComponent(approval.approvalId)}`, { action });
      recordApprovalResolution(resolved, sessionID);
      if (sessionID === $currentSession) {
        activeApproval.set(null);
        selectedApprovalID = '';
        showApprovalCenter = false;
        sessionRuntimeValue = { ...sessionRuntimeValue, pendingApprovals: (sessionRuntimeValue?.pendingApprovals || []).filter((item) => item.approvalId !== approval.approvalId) };
        sessionRuntime.set(sessionRuntimeValue);
      }
    } catch (err) { setError(err); }
    finally { approvalSubmitting = false; }
  }

  async function loadSessionRuntime(id) {
    if (!id) {
      sessionRuntime.set(null);
      sessionRuntimeValue = null;
      return;
    }
    try {
      const snapshot = await getSessionRuntime(id);
      if (id !== $currentSession) return;
      sessionRuntime.set(snapshot);
      sessionRuntimeValue = snapshot;
      const enabledTools = Object.fromEntries(Object.entries(snapshot?.capabilities || {}).map(([key, state]) => [key, Boolean(state?.enabled)]));
      setSessionTools(id, { ...sessionTools, ...enabledTools });
    } catch (err) {
      if (id === $currentSession) setError(err);
    }
  }

  async function updateRuntime(patch) {
    const id = $currentSession;
    if (!id || runtimeUpdating) return;
    const previous = sessionRuntimeValue;
    runtimeUpdating = true;
    try {
      const snapshot = await patchSessionRuntime(id, patch);
      if (id === $currentSession) {
        sessionRuntime.set(snapshot);
        sessionRuntimeValue = snapshot;
        const enabledTools = Object.fromEntries(Object.entries(snapshot?.capabilities || {}).map(([key, state]) => [key, Boolean(state?.enabled)]));
        setSessionTools(id, { ...sessionTools, ...enabledTools });
      }
      await refreshSessions();
    } catch (err) {
      sessionRuntime.set(previous);
      sessionRuntimeValue = previous;
      setError(err);
    } finally {
      runtimeUpdating = false;
    }
  }

  async function setMode(mode) {
    if (!$currentSession) {
      newSessionMode = mode;
      return;
    }
    await updateRuntime({ mode });
  }

  async function loadSessionEvents(id) {
    if (!id) {
      sessionRunEvents = [];
      sessionCapabilityEvents = [];
      approvalHistory = [];
      return;
    }
    try {
      const [runs, caps] = await Promise.all([
        getSessionRunEvents(id),
        getSessionCapabilityEvents(id)
      ]);
      if (id !== $currentSession) return;
      sessionRunEvents = runs || [];
      approvalHistory = approvalHistoryFromRunEvents(sessionRunEvents);
      sessionCapabilityEvents = caps || [];
    } catch {
      if (id !== $currentSession) return;
      sessionRunEvents = [];
      sessionCapabilityEvents = [];
      approvalHistory = [];
    }
  }

  async function loadSubAgents(id) {
    if (!id) {
      subAgents = [];
      return;
    }
    const agents = await getSessionSubAgents(id);
    if (id !== $currentSession) return;
    subAgents = mergeSubAgents(subAgents, agents || []);
    if (showSubAgentModal) {
      if (!selectedSubAgentID && subAgents.length > 0) {
        selectedSubAgentID = subAgents[0].id;
      }
      if (selectedSubAgentID) {
        await loadSubAgentMessages(selectedSubAgentID);
      }
    }
  }

  function scheduleSubAgentRefresh(delay = 250) {
    if (!$currentSession) return;
    if (subAgentRefreshTimer) clearTimeout(subAgentRefreshTimer);
    const targetSession = $currentSession;
    subAgentRefreshTimer = setTimeout(() => {
      subAgentRefreshTimer = 0;
      if (targetSession === $currentSession) {
        loadSubAgents(targetSession).catch(() => {});
      }
    }, delay);
  }

  function mergeSubAgents(existing = [], incoming = []) {
    const byID = new Map();
    for (const item of existing) {
      if (item?.id) byID.set(item.id, item);
    }
    for (const item of incoming) {
      if (!item?.id) continue;
      byID.set(item.id, { ...byID.get(item.id), ...item });
    }
    return Array.from(byID.values()).sort((a, b) => {
      const left = Date.parse(a.startedAt || a.updatedAt || '') || 0;
      const right = Date.parse(b.startedAt || b.updatedAt || '') || 0;
      if (left !== right) return left - right;
      return String(a.id).localeCompare(String(b.id));
    });
  }

  async function loadSubAgentMessages(agentID) {
    if (!$currentSession || !agentID) {
      subAgentModalMessages = [];
      return;
    }
    subAgentModalLoading = true;
    subAgentModalError = '';
    try {
      const msgs = await getSessionSubAgentMessages($currentSession, agentID);
      if (agentID !== selectedSubAgentID) return;
      const normalized = (msgs || []).map(normalizeSessionMessage).filter(Boolean);
      const live = subAgentTranscripts[agentID] || [];
      subAgentModalMessages = mergeMessageLists(normalized, live);
    } catch (err) {
      subAgentModalError = err instanceof Error ? err.message : String(err || '');
      subAgentModalMessages = subAgentTranscripts[agentID] || [];
    } finally {
      subAgentModalLoading = false;
    }
  }

  function mergeMessageLists(base = [], live = []) {
    let out = [...base];
    for (const item of live) {
      out = upsertMessageInList(out, item);
    }
    return out;
  }

  function openSubAgentModal(agentID = '') {
    selectedSubAgentID = agentID || selectedSubAgentID || subAgents[0]?.id || '';
    showSubAgentModal = true;
    if ($currentSession) {
      loadSubAgents($currentSession).catch(() => {});
    }
    if (selectedSubAgentID) {
      loadSubAgentMessages(selectedSubAgentID).catch(() => {});
    }
  }

  function closeSubAgentModal() {
    showSubAgentModal = false;
    subAgentModalError = '';
  }

  function selectSubAgent(agentID) {
    selectedSubAgentID = agentID;
    subAgentModalMessages = subAgentTranscripts[agentID] || [];
    loadSubAgentMessages(agentID).catch(() => {});
  }

  function beginOptimisticRunEvent(sessionID = $currentSession) {
    const id = `local-run-${Date.now()}`;
    const runID = `local_${Date.now()}`;
    const event = {
      id,
      runId: runID,
      sessionId: sessionID || '',
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
    approvalHistory = approvalHistoryFromRunEvents(sessionRunEvents);
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

  function startSessionStream(id) {
    if (!id || isCompletionActive(getSessionState(id))) return;
    const state = getSessionState(id);
    if (state.observer?.controller) return;
    const cursor = { ...(state.cursor || sessionStreamCursor) };
    const abort = new AbortController();
    registerObserver(id, abort);
    consumeSessionStream(id, cursor, abort).finally(() => {
      clearObserver(id, abort);
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
    if (event.data === '[DONE]') return;
    withSessionProjection(id, () => handleProjectedSessionStreamEvent(id, event));
  }

  function handleProjectedSessionStreamEvent(id, event) {
    if (event.event === 'done') {
      sessionStreamCompletedFor = id;
      refreshSessions().catch(() => {});
      loadSessionMessages(id).catch(() => {});
      loadSubAgents(id).catch(() => {});
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
        applyTranscriptStreamEvent(item, id);
      } catch {
        // ignore malformed transcript frames
      }
      return;
    }
    if (event.event === 'run_event') {
      try {
        const item = JSON.parse(event.data);
        if (!eventBelongsToSession(id, item)) return;
        upsertSessionRunEvent(item);
      } catch {
        // ignore malformed event frames
      }
      return;
    }
    if (event.event === 'runtime_event') {
      try {
        const snapshot = JSON.parse(event.data);
        if (!eventBelongsToSession(id, snapshot)) return;
        sessionRuntime.set(snapshot);
        sessionRuntimeValue = snapshot;
      } catch {
        // ignore malformed runtime frames
      }
      return;
    }
    if (event.event === 'approval_request') {
      handleApprovalRequest(event.data, id);
      return;
    }
    if (event.event === 'approval_resolved') {
      try {
        const item = JSON.parse(event.data);
        const resolvedSessionID = approvalSessionID(item, id);
        recordApprovalResolution(item, resolvedSessionID);
        if (resolvedSessionID !== id) return;
        if (selectedApprovalID === item.approvalId) {
          activeApproval.set(null);
          selectedApprovalID = '';
        }
        sessionRuntimeValue = { ...sessionRuntimeValue, pendingApprovals: (sessionRuntimeValue?.pendingApprovals || []).filter((approval) => approval.approvalId !== item.approvalId) };
        sessionRuntime.set(sessionRuntimeValue);
      } catch {
        // ignore malformed approval frames
      }
      return;
    }
    if (event.event === 'tool_event') {
      try {
        const item = JSON.parse(event.data);
        if (!eventBelongsToSession(id, item)) return;
        toolEvents.update((items) => [...items.filter((entry) => entry.toolCallId !== item.toolCallId), item].slice(-200));
        handleToolStatusEvent(item);
      } catch {
        // ignore malformed tool frames
      }
      return;
    }
    if (event.event === 'capability_event') {
      try {
        const item = JSON.parse(event.data);
        if (!eventBelongsToSession(id, item)) return;
        upsertSessionCapabilityEvent(item);
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

  function buildSubAgentSummary(agents = []) {
    const list = (agents || []).filter((item) => item?.id);
    const running = list.filter((item) => item.status === 'running' || item.status === 'ready').length;
    const failed = list.filter((item) => item.status === 'error' || item.status === 'failed').length;
    const done = list.filter((item) => item.status === 'done' || item.status === 'destroyed').length;
    return {
      visible: list.length > 0,
      count: list.length,
      running,
      failed,
      done,
      label: running > 0
        ? $t('chat.subagents.running', { count: running, total: list.length })
        : failed > 0
          ? $t('chat.subagents.failed', { count: failed, total: list.length })
          : $t('chat.subagents.done', { count: done || list.length, total: list.length })
    };
  }

  function subAgentStateClass(agent) {
    if (!agent) return 'done';
    if (agent.status === 'error' || agent.status === 'failed') return 'error';
    if (agent.status === 'running' || agent.status === 'ready') return 'running';
    return 'done';
  }

  function subAgentStatusLabel(status) {
    if (status === 'running' || status === 'ready') return $t('common.running');
    if (status === 'error' || status === 'failed') return $t('common.failed');
    if (status === 'destroyed') return $t('chat.subagents.destroyed');
    return $t('common.completed');
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
      const args = normalizeJSONValue(message.arguments);
      const plan = normalizePlan(message.plan || (message.toolName === 'plan' ? args : null));
      if (message.toolName === 'plan' && plan) {
        return {
          id: message.id,
          seq: message.seq,
          role: 'plan',
          agentId: message.agentId,
          toolCallId: message.toolCallId,
          toolName: message.toolName,
          plan
        };
      }
      return {
        id: message.id,
        seq: message.seq,
        role: 'toolCall',
        agentId: message.agentId,
        toolCallId: message.toolCallId,
        toolName: message.toolName || 'tool',
        arguments: args,
        invalidArguments: message.invalidArguments,
        callView: buildToolCallView(message.toolName || 'tool', args, message.invalidArguments)
      };
    }
    if (message.role === 'toolResult') {
      if (message.toolName === 'plan' && !message.isError) return null;
      return {
        id: message.id,
        seq: message.seq,
        role: 'toolResult',
        agentId: message.agentId,
        toolCallId: message.toolCallId,
        toolName: message.toolName || 'tool',
        summary: formatToolResultSummary(message.toolName || 'tool', message.summary || $t('chat.tool.result'), message.isError),
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
      agentId: message.agentId,
      content: message.content || textFromContents(message.contents),
      images
    };
  }

  function formatToolResultSummary(toolName, summary, isError = false) {
    if (isError) return summary;
    if (toolName === 'workflow_lint') {
      const parsed = parseWorkflowLintResult(summary);
      if (parsed) {
        if (parsed.valid) return `${$t('chat.tool.workflowLint.valid')} · ${parsed.status}`;
        return `${$t('chat.tool.workflowLint.invalid')} · ${parsed.error || parsed.status}`;
      }
    }
    return summary;
  }

  function normalizePlan(value) {
    value = normalizeJSONValue(value);
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
      bashResult: parseBashResult(content),
      browserResult: parseBrowserResult(content),
      subAgentResult: parseSubAgentResult(content),
      workflowLintResult: parseWorkflowLintResult(content)
    };
  }

  function buildToolCallView(toolName, args, invalidArguments = '') {
    const name = toolName || 'tool';
    const raw = normalizeJSONValue(args);
    const value = isPlainObject(raw) ? raw : {};
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
        raw,
        invalidArguments
      };
    }
    if (name === 'ls') {
      return {
        kind: 'ls',
        label: $t('chat.tool.ls.label'),
        target: value.path || '.',
        details: [],
        raw,
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
        raw,
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
        raw,
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
        raw,
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
        raw,
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
        raw,
        invalidArguments
      };
    }
    if (name === 'browser') {
      const details = [];
      const action = String(value.action || '').trim();
      if (value.selector) details.push($t('chat.tool.browser.selector', { selector: value.selector }));
      if (value.outputPath) details.push($t('chat.tool.browser.output', { path: value.outputPath }));
      if (value.fullPage) details.push($t('chat.tool.browser.fullPage'));
      if (value.interactive) details.push($t('chat.tool.browser.interactive'));
      if (value.width || value.height) details.push($t('chat.tool.browser.viewport', { width: value.width || '?', height: value.height || '?' }));
      if (value.viewportWidth || value.viewportHeight) details.push($t('chat.tool.browser.viewport', { width: value.viewportWidth || '?', height: value.viewportHeight || '?' }));
      if (value.format) details.push(String(value.format));
      return {
        kind: 'browser',
        label: $t('chat.tool.browser.label'),
        target: browserTarget(value) || $t('chat.tool.browser.missing'),
        details,
        action,
        selector: value.selector || '',
        url: value.url || '',
        value: value.value ?? value.text ?? value.key ?? '',
        expression: value.expression || '',
        raw,
        invalidArguments
      };
    }
    if (name === 'skill_ref') {
      return {
        kind: 'skill-ref',
        label: $t('chat.tool.skillRef.label'),
        target: value.skill && value.ref ? `${value.skill}/${value.ref}` : $t('chat.tool.skillRef.missing'),
        details: [
          value.skill ? $t('chat.tool.skillRef.skill', { skill: value.skill }) : '',
          value.ref ? $t('chat.tool.skillRef.ref', { ref: value.ref }) : ''
        ].filter(Boolean),
        skill: value.skill || '',
        ref: value.ref || '',
        raw,
        invalidArguments
      };
    }
    if (name === 'workflow_lint') {
      const source = typeof value.source === 'string' ? value.source : '';
      const firstLine = source.split('\n').map((line) => line.trim()).find(Boolean) || '';
      const lines = countTextLines(source);
      const chars = source.length;
      return {
        kind: 'workflow-lint',
        label: $t('chat.tool.workflowLint.label'),
        target: firstLine ? compactText(firstLine, 120) : $t('chat.tool.workflowLint.missing'),
        details: [
          $t('chat.tool.workflowLint.lines', { count: lines }),
          $t('chat.tool.workflowLint.chars', { count: chars })
        ],
        source,
        lines,
        chars,
        raw,
        invalidArguments
      };
    }
    if (name === 'delegate_subagent' || name === 'subagent_spawn') {
      const details = [];
      if (value.mode) details.push($t('chat.tool.subagent.mode', { mode: value.mode }));
      if (value.work_dir) details.push($t('chat.tool.subagent.workDir', { path: value.work_dir }));
      if (Array.isArray(value.tools) && value.tools.length > 0) details.push($t('chat.tool.subagent.tools', { tools: value.tools.join(', ') }));
      if (value.max_iterations) details.push($t('chat.tool.subagent.maxIterations', { count: value.max_iterations }));
      return {
        kind: 'subagent-task',
        label: name === 'delegate_subagent' ? $t('chat.tool.subagent.delegate') : $t('chat.tool.subagent.spawn'),
        target: compactText(value.task || $t('chat.tool.subagent.taskMissing'), 140),
        details,
        task: value.task || '',
        raw,
        invalidArguments
      };
    }
    if (name === 'subagent_status' || name === 'subagent_destroy' || name === 'subagent_send') {
      const details = [];
      if (name === 'subagent_send' && value.message) details.push(compactText(value.message, 120));
      return {
        kind: 'subagent-handle',
        label: name === 'subagent_status'
          ? $t('chat.tool.subagent.status')
          : name === 'subagent_destroy'
            ? $t('chat.tool.subagent.destroy')
            : $t('chat.tool.subagent.send'),
        target: value.handle || $t('chat.tool.subagent.handleMissing'),
        details,
        handle: value.handle || '',
        message: value.message || '',
        raw,
        invalidArguments
      };
    }
    return {
      kind: 'generic',
      label: name,
      target: '',
      details: [],
      raw,
      invalidArguments
    };
  }

  function isPlainObject(value) {
    return value && typeof value === 'object' && !Array.isArray(value);
  }

  function normalizeJSONValue(value) {
    if (typeof value !== 'string') return value;
    const trimmed = value.trim();
    if (!trimmed) return value;
    if (!['{', '[', '"'].includes(trimmed[0]) && !/^(true|false|null|-?\d)/.test(trimmed)) return value;
    try {
      return JSON.parse(trimmed);
    } catch {
      return value;
    }
  }

  function stringFrom(value) {
    if (value === undefined || value === null) return '';
    if (typeof value === 'string') return value;
    return String(value);
  }

  function countTextLines(text = '') {
    if (!text) return 0;
    return String(text).split('\n').length;
  }

  function compactText(text = '', limit = 120) {
    const normalized = String(text || '').replace(/\s+/g, ' ').trim();
    if (normalized.length <= limit) return normalized;
    return `${normalized.slice(0, Math.max(0, limit - 1))}...`;
  }

  function browserTarget(value = {}) {
    return value.url
      || value.selector
      || value.outputPath
      || value.text
      || value.value
      || value.key
      || value.attr
      || value.targetId
      || value.session
      || '';
  }

  function toolResultKind(toolName, content) {
    if (toolName === 'read' && parseReadResult(content).length > 0) return 'read';
    if (toolName === 'ls' && (parseLsResult(content).length > 0 || content === '(empty directory)')) return 'ls';
    if (toolName === 'grep' && (parseGrepResult(content).matches.length > 0 || content === '(no matches found)')) return 'grep';
    if (toolName === 'bash' && parseBashResult(content)) return 'bash';
    if (toolName === 'browser') return 'browser';
    if (toolName === 'skill_ref') return 'skill-ref';
    if (toolName === 'workflow_lint' && parseWorkflowLintResult(content)) return 'workflow-lint';
    if (isSubAgentTool(toolName)) return 'subagent';
    return 'text';
  }

  function isSubAgentTool(toolName) {
    return ['delegate_subagent', 'subagent_spawn', 'subagent_status', 'subagent_send', 'subagent_destroy'].includes(toolName);
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

  function parseBrowserResult(content = '') {
    const parsed = normalizeJSONValue(content);
    if (isPlainObject(parsed)) {
      return {
        status: stringFrom(parsed.status || parsed.message || parsed.result || parsed.action || 'browser result'),
        title: stringFrom(parsed.title || parsed.pageTitle || ''),
        url: stringFrom(parsed.url || parsed.href || parsed.currentURL || parsed.currentUrl || ''),
        content: JSON.stringify(parsed, null, 2)
      };
    }
    const text = String(content || '').trim();
    if (!text) return null;
    const lines = text.split('\n').map((line) => line.trim()).filter(Boolean);
    const first = lines[0] || '';
    const titleLine = lines.find((line) => line.toLowerCase().startsWith('title:'));
    const urlLine = lines.find((line) => line.toLowerCase().startsWith('url:'));
    return {
      status: first,
      title: titleLine ? titleLine.replace(/^title:\s*/i, '') : '',
      url: urlLine ? urlLine.replace(/^url:\s*/i, '') : '',
      content: text
    };
  }

  function parseSubAgentResult(content = '') {
    if (isPlainObject(content)) return content;
    const text = String(content || '').trim();
    if (!text) return null;
    try {
      const parsed = JSON.parse(text);
      if (parsed && typeof parsed === 'object') return parsed;
    } catch {
      // fall through to plain text
    }
    return { result: text };
  }

  function parseWorkflowLintResult(content = '') {
    const parsed = normalizeJSONValue(content);
    if (!isPlainObject(parsed) || !Object.prototype.hasOwnProperty.call(parsed, 'valid')) return null;
    return {
      valid: parsed.valid === true,
      status: stringFrom(parsed.status || (parsed.valid ? 'done' : 'error')),
      error: stringFrom(parsed.error || ''),
      tasks: Array.isArray(parsed.tasks) ? parsed.tasks.map(stringFrom).filter(Boolean) : [],
      results: Array.isArray(parsed.results) ? parsed.results.map(stringFrom).filter(Boolean) : [],
      raw: JSON.stringify(parsed, null, 2)
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
    if (item?.agentId) {
      applySubAgentToolStatus(item);
      scrollChatToBottom();
      return;
    }
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
      if (isSubAgentTool(item.tool)) {
        applySubAgentToolResultSummary({
          toolName: item.tool,
          summary: item.summary || '',
          isError: item.isError || item.status === 'failed'
        });
      }
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

  function upsertMessageInList(list, next) {
    if (!next) return list;
    if (next.id) {
      const existing = list.findIndex((m) => m.id === next.id);
      if (existing >= 0) {
        const copy = [...list];
        copy[existing] = { ...copy[existing], ...next };
        return copy;
      }
    }
    if (next.role === 'toolCall' || next.role === 'plan') {
      const toolCallId = next.toolCallId || '';
      const existing = toolCallId ? list.findIndex((m) => m.toolCallId === toolCallId && (m.role === 'toolCall' || m.role === 'plan')) : -1;
      if (existing >= 0) {
        const copy = [...list];
        copy[existing] = { ...copy[existing], ...next };
        return copy;
      }
    }
    if (next.role === 'toolResult') {
      const toolCallId = next.toolCallId || '';
      const existing = toolCallId ? list.findIndex((m) => m.role === 'toolResult' && m.toolCallId === toolCallId) : -1;
      if (existing >= 0) {
        const copy = [...list];
        copy[existing] = { ...copy[existing], ...next };
        return copy;
      }
      const callIdx = toolCallId ? list.findIndex((m) => m.toolCallId === toolCallId && (m.role === 'toolCall' || m.role === 'plan')) : -1;
      if (callIdx >= 0) {
        return [...list.slice(0, callIdx + 1), next, ...list.slice(callIdx + 1)];
      }
    }
    return [...list, next];
  }

  function applySubAgentTranscriptMessage(message, type = 'message') {
    const agentID = message?.agentId || '';
    if (!agentID) return false;
    ensureSubAgent(agentID, { status: 'running' });
    const current = subAgentTranscripts[agentID] || [];
    if (type === 'assistant_delta') {
      const delta = message.content || '';
      if (!delta) return true;
      const next = [...current];
      const last = next[next.length - 1];
      if (last?.role === 'assistant') {
        next[next.length - 1] = { ...last, content: `${last.content || ''}${delta}` };
      } else {
        next.push({ role: 'assistant', agentId: agentID, content: delta });
      }
      subAgentTranscripts = { ...subAgentTranscripts, [agentID]: next };
      ensureSubAgent(agentID, { status: 'running', messageCount: next.length });
    } else {
      const normalized = normalizeSessionMessage(message);
      if (normalized) {
        const next = upsertMessageInList(current, normalized);
        subAgentTranscripts = {
          ...subAgentTranscripts,
          [agentID]: next
        };
        ensureSubAgent(agentID, { status: 'running', messageCount: next.length });
      }
    }
    if (showSubAgentModal && selectedSubAgentID === agentID) {
      subAgentModalMessages = subAgentTranscripts[agentID] || [];
    }
    return true;
  }

  function applySubAgentToolStatus(item) {
    const agentID = item?.agentId || '';
    if (!agentID) return;
    ensureSubAgent(agentID, { status: subAgentStatusFromToolStatus(item.status, item.isError) });
    if (item.status === 'running') {
      applySubAgentTranscriptMessage({
        role: 'toolCall',
        agentId: agentID,
        toolCallId: item.toolCallId || '',
        toolName: item.tool,
        arguments: item.args
      });
    } else {
      applySubAgentTranscriptMessage({
        role: 'toolResult',
        agentId: agentID,
        toolCallId: item.toolCallId || '',
        toolName: item.tool,
        summary: item.summary || (item.status === 'failed' ? $t('chat.tool.failed') : $t('chat.tool.completed')),
        isError: item.isError || item.status === 'failed',
        hasDetail: false
      });
    }
  }

  function recordSubAgentStatus(agentID, status, summary = '') {
    if (!agentID) return;
    const state = subAgentStatusFromToolStatus(status, status === 'error' || status === 'failed');
    ensureSubAgent(agentID, {
      status: state,
      error: state === 'error' ? summary : ''
    });
    const current = subAgentTranscripts[agentID] || [];
    const next = upsertMessageInList(current, {
      id: `status:${agentID}:${state}:${summary}`,
      role: 'status',
      agentId: agentID,
      content: state,
      summary,
      isError: state === 'error'
    });
    subAgentTranscripts = { ...subAgentTranscripts, [agentID]: next };
    if (showSubAgentModal && selectedSubAgentID === agentID) {
      subAgentModalMessages = next;
    }
    scheduleSubAgentRefresh();
  }

  function subAgentStatusFromToolStatus(status, isError = false) {
    const s = String(status || '').toLowerCase();
    if (isError || s === 'error' || s === 'failed') return 'error';
    if (s === 'done' || s === 'completed' || s === 'complete') return 'done';
    if (s === 'destroyed') return 'destroyed';
    if (s === 'message_sent' || s === 'running' || s === 'ready') return 'running';
    return s || 'running';
  }

  function applySubAgentToolResultSummary(message) {
    if (!message || !isSubAgentTool(message.toolName)) return;
    const result = parseSubAgentResult(message.content || message.summary || '');
    if (!result) {
      scheduleSubAgentRefresh();
      return;
    }
    const handle = stringFrom(result.handle || result.id || result.agent_id || result.agentId);
    if (handle) {
      const status = subAgentStatusFromToolStatus(result.status, message.isError);
      const patch = {
        status,
        lastResponse: stringFrom(result.result || result.last_response || result.partial_result || ''),
        error: stringFrom(result.error || '')
      };
      const messageCount = Number(result.message_count || 0);
      if (Number.isFinite(messageCount) && messageCount > 0) patch.messageCount = messageCount;
      ensureSubAgent(handle, patch);
    }
    scheduleSubAgentRefresh();
  }

  function ensureSubAgent(agentID, patch = {}) {
    if (!agentID) return;
    const idx = subAgents.findIndex((item) => item.id === agentID);
    const now = new Date().toISOString();
    if (idx >= 0) {
      subAgents[idx] = { ...subAgents[idx], updatedAt: now, ...patch };
      subAgents = subAgents;
      return;
    }
    subAgents = [...subAgents, {
      id: agentID,
      status: patch.status || 'running',
      active: true,
      messageCount: 0,
      startedAt: now,
      updatedAt: now,
      ...patch
    }];
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
      agentId: item.agentId || '',
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
      agentId: item.agentId || '',
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

  function handleStreamEvent(boundSessionID, event) {
    if (event.data === '[DONE]') return;
    withSessionProjection(boundSessionID, () => handleProjectedCompletionEvent(boundSessionID, event));
  }

  function handleProjectedCompletionEvent(boundSessionID, event) {
    if (event.event === 'transcript') {
      try {
        const item = JSON.parse(event.data);
        if (!eventBelongsToSession(boundSessionID, item)) return;
        streamUsesTranscript = true;
        applyTranscriptStreamEvent(item, boundSessionID);
      } catch {
        // ignore malformed transcript frames
      }
      return;
    }
    if (event.event === 'approval_request') {
      try {
        const item = JSON.parse(event.data);
        if (!eventBelongsToSession(boundSessionID, item)) return;
        handleApprovalRequest(item, boundSessionID);
      } catch {
        // ignore malformed approval frames
      }
      return;
    }
    if (event.event === 'tool_status') {
      if (streamUsesTranscript) return;
      try {
        const item = JSON.parse(event.data);
        if (!eventBelongsToSession(boundSessionID, item)) return;
        handleToolStatusEvent(item);
      } catch {
        chatEvents = [...chatEvents.slice(-49), { type: 'tool', status: 'unknown', raw: event.data }];
        scrollChatToBottom();
      }
      return;
    }
    try {
      const chunk = JSON.parse(event.data);
      if (!eventBelongsToSession(boundSessionID, chunk)) return;
      const delta = chunk?.choices?.[0]?.delta?.content;
      if (delta && !streamUsesTranscript) {
        appendAssistantDelta(delta);
      }
    } catch {
      // ignore malformed frames
    }
  }

  function applyTranscriptStreamEvent(item, boundSessionID = $currentSession) {
    if (!eventBelongsToSession(boundSessionID, item)) return;
    const message = item?.message;
    if (!message) return;
    if (message.agentId) {
      if (item.type === 'subagent_status') {
        recordSubAgentStatus(message.agentId, message.content, message.summary || '');
        return;
      }
      applySubAgentTranscriptMessage(message, item.type);
      return;
    }
    if (item.type === 'assistant_delta') {
      appendAssistantDelta(message.content || '');
      return;
    }
    if (item.type === 'message') {
      if (message.role === 'toolResult') {
        applySubAgentToolResultSummary(message);
      } else if (message.role === 'toolCall' && isSubAgentTool(message.toolName)) {
        scheduleSubAgentRefresh();
      }
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
  {#if subAgentSummary.visible}
    <button type="button" class="subagent-strip" on:click={() => openSubAgentModal()}>
      <span class="dot {subAgentSummary.failed > 0 ? 'error' : subAgentSummary.running > 0 ? 'running' : 'done'}"></span>
      <strong>{$t('chat.subagents.title')}</strong>
      <span>{subAgentSummary.label}</span>
      <em>{$t('chat.subagents.open')}</em>
    </button>
  {/if}
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
                {:else if msg.callView?.kind === 'browser'}
                  <div class="browser-call">
                    <div class="find-row">
                      <span>{$t('chat.tool.browser.action')}</span>
                      <code>{msg.callView.action || $t('chat.tool.browser.missing')}</code>
                    </div>
                    {#if msg.callView.url}
                      <div class="find-row">
                        <span>{$t('chat.tool.browser.url')}</span>
                        <code>{msg.callView.url}</code>
                      </div>
                    {/if}
                    {#if msg.callView.selector}
                      <div class="find-row">
                        <span>{$t('chat.tool.browser.selectorLabel')}</span>
                        <code>{msg.callView.selector}</code>
                      </div>
                    {/if}
                    {#if msg.callView.value}
                      <div class="find-row">
                        <span>{$t('chat.tool.browser.value')}</span>
                        <code>{msg.callView.value}</code>
                      </div>
                    {/if}
                    {#if msg.callView.expression}
                      <div class="find-row">
                        <span>{$t('chat.tool.browser.expression')}</span>
                        <code>{msg.callView.expression}</code>
                      </div>
                    {/if}
                  </div>
                {:else if msg.callView?.kind === 'skill-ref'}
                  <div class="skill-ref-call">
                    <div class="find-row">
                      <span>{$t('chat.tool.skillRef.skillLabel')}</span>
                      <code>{msg.callView.skill || $t('chat.tool.skillRef.missing')}</code>
                    </div>
                    <div class="find-row">
                      <span>{$t('chat.tool.skillRef.refLabel')}</span>
                      <code>{msg.callView.ref || $t('chat.tool.skillRef.missing')}</code>
                    </div>
                  </div>
                {:else if msg.callView?.kind === 'workflow-lint'}
                  <div class="workflow-lint-call">
                    <div class="write-call-head">
                      <strong>{$t('chat.tool.workflowLint.source')}</strong>
                      <span>{$t('chat.tool.write.summary', { lines: msg.callView.lines, chars: msg.callView.chars })}</span>
                    </div>
                    <pre class:empty={msg.callView.source === ''}>{msg.callView.source || $t('chat.tool.workflowLint.missing')}</pre>
                  </div>
                {:else if msg.callView?.kind === 'subagent-task'}
                  <div class="subagent-call">
                    <span>{$t('chat.tool.subagent.task')}</span>
                    <p>{msg.callView.task || msg.callView.target}</p>
                  </div>
                {:else if msg.callView?.kind === 'subagent-handle'}
                  <div class="subagent-call compact">
                    <div class="find-row">
                      <span>{$t('chat.tool.subagent.handle')}</span>
                      <code>{msg.callView.handle || $t('chat.tool.subagent.handleMissing')}</code>
                    </div>
                    {#if msg.callView.message}
                      <div class="find-row">
                        <span>{$t('chat.tool.subagent.message')}</span>
                        <code>{msg.callView.message}</code>
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
                  {#if msg.detail?.kind === 'browser' && msg.detail.browserResult}
                    <div class="browser-result">
                      <div class="browser-result-head">
                        <strong>{msg.detail.browserResult.status}</strong>
                        {#if msg.detail.browserResult.title}<span>{msg.detail.browserResult.title}</span>{/if}
                      </div>
                      {#if msg.detail.browserResult.url}
                        <code>{msg.detail.browserResult.url}</code>
                      {/if}
                      {#if !msg.detail.browserResult.title && !msg.detail.browserResult.url && msg.detail.browserResult.content}
                        <pre>{msg.detail.browserResult.content}</pre>
                      {/if}
                    </div>
                  {:else if msg.detail?.kind === 'subagent' && msg.detail.subAgentResult}
                    <div class="subagent-result">
                      {#if msg.detail.subAgentResult.handle}
                        <div><span>{$t('chat.tool.subagent.handle')}</span><code>{msg.detail.subAgentResult.handle}</code></div>
                      {/if}
                      {#if msg.detail.subAgentResult.status}
                        <div><span>{$t('chat.tool.subagent.statusLabel')}</span><strong>{msg.detail.subAgentResult.status}</strong></div>
                      {/if}
                      {#if msg.detail.subAgentResult.duration}
                        <div><span>{$t('chat.tool.subagent.duration')}</span><code>{msg.detail.subAgentResult.duration}</code></div>
                      {/if}
                      {#if msg.detail.subAgentResult.tool_calls !== undefined}
                        <div><span>{$t('chat.tool.subagent.toolCalls')}</span><code>{msg.detail.subAgentResult.tool_calls}</code></div>
                      {/if}
                      {#if msg.detail.subAgentResult.error}
                        <p class="error-text">{msg.detail.subAgentResult.error}</p>
                      {/if}
                      {#if msg.detail.subAgentResult.result || msg.detail.subAgentResult.last_response || msg.detail.subAgentResult.partial_result}
                        <pre>{msg.detail.subAgentResult.result || msg.detail.subAgentResult.last_response || msg.detail.subAgentResult.partial_result}</pre>
                      {/if}
                    </div>
                  {:else if msg.detail?.kind === 'skill-ref' && msg.detail.content}
                    <div class="skill-ref-result">
                      <div class="markdown">{@html markdownToHTML(msg.detail.content)}</div>
                    </div>
                  {:else if msg.detail?.kind === 'workflow-lint' && msg.detail.workflowLintResult}
                    <div class="workflow-lint-result">
                      <div class="workflow-lint-head">
                        <strong class:failed={!msg.detail.workflowLintResult.valid}>
                          {msg.detail.workflowLintResult.valid ? $t('chat.tool.workflowLint.valid') : $t('chat.tool.workflowLint.invalid')}
                        </strong>
                        {#if msg.detail.workflowLintResult.status}
                          <span>{msg.detail.workflowLintResult.status}</span>
                        {/if}
                      </div>
                      {#if msg.detail.workflowLintResult.error}
                        <p class="error-text">{msg.detail.workflowLintResult.error}</p>
                      {/if}
                      {#if msg.detail.workflowLintResult.tasks.length}
                        <section>
                          <strong>{$t('chat.tool.workflowLint.tasks')}</strong>
                          <div class="workflow-chip-row">
                            {#each msg.detail.workflowLintResult.tasks as task}
                              <code>{task}</code>
                            {/each}
                          </div>
                        </section>
                      {/if}
                      {#if msg.detail.workflowLintResult.results.length}
                        <section>
                          <strong>{$t('chat.tool.workflowLint.results')}</strong>
                          <div class="workflow-chip-row">
                            {#each msg.detail.workflowLintResult.results as result}
                              <code>{result}</code>
                            {/each}
                          </div>
                        </section>
                      {/if}
                    </div>
                  {:else if msg.detail?.kind === 'bash' && msg.detail.bashResult}
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
        <div bind:this={runtimeControls} class="runtime-controls" aria-label="Session runtime controls">
          <button
            type="button"
            class:open={showRuntimePanel}
            class="runtime-toggle"
            aria-expanded={showRuntimePanel}
            aria-controls="session-runtime-panel"
            on:click={() => (showRuntimePanel = !showRuntimePanel)}
          >
            <span class="runtime-label">Mode</span>
            <strong>{runtimeMode}</strong>
            <span class="runtime-chevron" aria-hidden="true">⌄</span>
            {#if pendingApprovalCount}<span class="runtime-badge">{pendingApprovalCount}</span>{/if}
          </button>
          {#if showRuntimePanel}
            <section id="session-runtime-panel" class="runtime-panel">
              <header>
                <strong>Session runtime</strong>
                {#if runtimeActiveRun}<span class="dot running"></span><span>{runtimeActiveRun.status}</span>{/if}
              </header>
              <p class="runtime-hint">plan is read-only planning, agent requests approval for guarded actions, and yolo runs automatically.</p>
              <div class="mode-switcher" role="group" aria-label="Agent mode">
                {#each ['plan', 'agent', 'yolo'] as mode}
                  <button type="button" class:active={runtimeMode === mode} disabled={runtimeUpdating || busy} on:click={() => setMode(mode)}>{mode}</button>
                {/each}
              </div>
              {#if pendingApprovalCount}
                <div class="approval-summary"><strong>{pendingApprovalCount} pending approval{pendingApprovalCount === 1 ? '' : 's'}</strong><button type="button" class="ghost sm" on:click={() => (showApprovalCenter = true)}>Review approvals</button></div>
              {/if}
            </section>
          {/if}
        </div>
        <div class="tool-toggles" aria-label={$t('chat.tools')}>
          {#each availableToolToggles as item}
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


{#if showApprovalCenter}
  <div class="subagent-overlay" role="dialog" aria-modal="true" aria-label="Approval center">
    <div class="subagent-modal approval-center">
      <header>
        <div>
          <strong>Approval center</strong>
          <span>{pendingApprovalCount} pending · {approvalHistory.length} recorded for this session</span>
        </div>
        <button type="button" class="ghost sm" on:click={() => (showApprovalCenter = false)}>Close</button>
      </header>
      <div class="approval-list" aria-live="polite">
        {#if selectedApproval}
          <article class="approval-card" aria-labelledby="approval-title-{selectedApproval.approvalId}">
            <div class="approval-card-head">
              <div class="approval-title-group">
                <div class="approval-kicker"><span class="approval-risk {selectedApproval.risk || 'medium'}">{selectedApproval.risk || 'medium'} risk</span><span>{selectedApproval.mode || runtimeMode} mode</span></div>
                <strong id="approval-title-{selectedApproval.approvalId}">{selectedApproval.summary || selectedApproval.tool?.name}</strong>
                <p>{selectedApproval.reason || 'This action requires confirmation.'}</p>
              </div>
              {#if pendingApprovalCount > 1}
                <label class="approval-picker">Request
                  <select aria-label="Select pending approval" value={selectedApprovalID} on:change={(event) => { selectedApprovalID = event.currentTarget.value; activeApproval.set((sessionRuntimeValue?.pendingApprovals || []).find((approval) => approval.approvalId === selectedApprovalID) || null); }}>
                    {#each sessionRuntimeValue?.pendingApprovals || [] as approval}<option value={approval.approvalId}>{approval.summary || approval.tool?.name}</option>{/each}
                  </select>
                </label>
              {/if}
            </div>
            {#if selectedApproval.tool?.name === 'bash'}
              <div class="approval-bash tool-call-body embedded">
                <div class="tool-title">
                  <span class="dot running"></span>
                  <strong>Bash</strong>
                  {#if approvalBashWorkDir(selectedApproval)}<span class="tool-target">{approvalBashWorkDir(selectedApproval)}</span>{/if}
                </div>
                <div class="bash-block">
                  <span>command</span>
                  <pre>{approvalBashCommand(selectedApproval)}</pre>
                </div>
              </div>
            {:else}
              <div class="approval-tool tool-call-body embedded">
                <div class="tool-title">
                  <span class="dot running"></span>
                  <strong>{approvalToolViewValue.label || selectedApproval.tool?.label || selectedApproval.tool?.name}</strong>
                  {#if approvalToolViewValue.target}<span class="tool-target">{approvalToolViewValue.target}</span>{/if}
                </div>
                {#if approvalToolViewValue.details?.length}
                  <div class="tool-call-tags">
                    {#each approvalToolViewValue.details as detail}<span>{detail}</span>{/each}
                  </div>
                {/if}
                {#if approvalToolViewValue.kind === 'edit' && approvalToolViewValue.edits?.length}
                  <div class="edit-call">
                    {#each approvalToolViewValue.edits as edit}
                      <section class="edit-block">
                        <div class="edit-block-head"><strong>{$t('chat.tool.edit.editNumber', { number: edit.index })}</strong><span>{$t('chat.tool.edit.lineChange', { old: edit.oldLines, next: edit.newLines })}</span></div>
                        <div class="edit-columns"><div class="edit-pane old"><span>{$t('chat.tool.edit.oldText')}</span><pre class:empty={edit.oldText === ''}>{edit.oldText || $t('chat.tool.edit.empty')}</pre></div><div class="edit-pane new"><span>{$t('chat.tool.edit.newText')}</span><pre class:empty={edit.newText === ''}>{edit.newText || $t('chat.tool.edit.empty')}</pre></div></div>
                      </section>
                    {/each}
                  </div>
                {:else if approvalToolViewValue.kind === 'write'}
                  <div class="write-call"><div class="write-call-head"><strong>{$t('chat.tool.write.preview')}</strong><span>{$t('chat.tool.write.summary', { lines: approvalToolViewValue.lines, chars: approvalToolViewValue.chars })}</span></div><span>{$t('chat.tool.write.content')}</span><pre class:empty={approvalToolViewValue.content === ''}>{approvalToolViewValue.content || $t('chat.tool.edit.empty')}</pre></div>
                {:else if approvalToolViewValue.kind === 'find'}
                  <div class="find-call"><div class="find-row"><span>{$t('chat.tool.find.pattern')}</span><code>{approvalToolViewValue.pattern || $t('chat.tool.find.missing')}</code></div><div class="find-row"><span>{$t('chat.tool.find.searchPath')}</span><code>{approvalToolViewValue.path}</code></div></div>
                {:else if approvalToolViewValue.kind === 'browser'}
                  <div class="browser-call"><div class="find-row"><span>{$t('chat.tool.browser.action')}</span><code>{approvalToolViewValue.action || $t('chat.tool.browser.missing')}</code></div>{#if approvalToolViewValue.url}<div class="find-row"><span>{$t('chat.tool.browser.url')}</span><code>{approvalToolViewValue.url}</code></div>{/if}{#if approvalToolViewValue.selector}<div class="find-row"><span>{$t('chat.tool.browser.selectorLabel')}</span><code>{approvalToolViewValue.selector}</code></div>{/if}</div>
                {:else if approvalToolViewValue.kind === 'skill-ref'}
                  <div class="skill-ref-call"><div class="find-row"><span>{$t('chat.tool.skillRef.skillLabel')}</span><code>{approvalToolViewValue.skill || $t('chat.tool.skillRef.missing')}</code></div><div class="find-row"><span>{$t('chat.tool.skillRef.refLabel')}</span><code>{approvalToolViewValue.ref || $t('chat.tool.skillRef.missing')}</code></div></div>
                {:else}
                  <div class="approval-tool-summary"><span>{selectedApproval.tool?.details?.path || selectedApproval.context?.workDir || 'This action requires permission.'}</span></div>
                {/if}
              </div>
            {/if}
            <div class="approval-actions">
              <button class="primary" disabled={approvalSubmitting} on:click={() => respondApproval(selectedApproval, 'approve_once')}>Approve once</button>
              <button class="ghost approval-deny" disabled={approvalSubmitting} on:click={() => respondApproval(selectedApproval, 'deny_once')}>Deny</button>
              {#if selectedApproval.actions?.includes('remember_command')}<span class="approval-action-divider"></span><button class="ghost sm" disabled={approvalSubmitting} on:click={() => respondApproval(selectedApproval, 'remember_command')}>Always allow command</button><button class="ghost sm" disabled={approvalSubmitting} on:click={() => respondApproval(selectedApproval, 'remember_prefix')}>Always allow prefix</button>{/if}
              {#if selectedApproval.actions?.includes('allow_edit_path')}<button class="ghost sm" disabled={approvalSubmitting} on:click={() => respondApproval(selectedApproval, 'allow_edit_path')}>Allow this path</button>{/if}
            </div>
            <details class="approval-raw"><summary>Request JSON</summary><pre>{JSON.stringify(selectedApproval, null, 2)}</pre></details>
          </article>
        {:else}
          <div class="approval-empty"><strong>No pending approvals</strong><span>New approval requests will appear here.</span></div>
        {/if}
        {#if approvalHistory.length}
          <section class="approval-history" aria-label="Session approval history">
            <div class="approval-history-head"><h4>Session audit history</h4><span>{approvalHistory.length} decisions</span></div>
            <div class="approval-history-list">
              {#each approvalHistory as item}
                <article class="approval-history-item">
                  <strong>{item.action === 'deny_once' ? 'Denied' : 'Approved'}</strong>
                  <span>{item.message || item.action}</span>
                </article>
              {/each}
            </div>
          </section>
        {/if}
      </div>
    </div>
  </div>
{/if}

<DirBrowser bind:open={showBrowser} on:select={onDirSelect} on:close={() => (showBrowser = false)} />

{#if showSubAgentModal}
  <div class="subagent-overlay" role="dialog" aria-modal="true" aria-label={$t('chat.subagents.history')}>
    <div class="subagent-modal">
      <header>
        <div>
          <strong>{$t('chat.subagents.history')}</strong>
          <span>{$t('chat.subagents.subtitle', { count: subAgents.length })}</span>
        </div>
        <button type="button" class="ghost sm" on:click={closeSubAgentModal}>{$t('common.close')}</button>
      </header>
      <div class="subagent-modal-body">
        <aside class="subagent-list">
          {#each subAgents as agent}
            <button
              type="button"
              class:active={agent.id === selectedSubAgentID}
              on:click={() => selectSubAgent(agent.id)}
            >
              <span class="dot {subAgentStateClass(agent)}"></span>
              <strong>{shortID(agent.id)}</strong>
              <em>{subAgentStatusLabel(agent.status)}</em>
              {#if agent.messageCount}<small>{agent.messageCount}</small>{/if}
            </button>
          {/each}
        </aside>
        <section class="subagent-history">
          {#if subAgentModalLoading}
            <p class="pending-text">{$t('chat.subagents.loading')}</p>
          {:else if subAgentModalError}
            <p class="error-text">{subAgentModalError}</p>
          {:else if subAgentModalMessages.length === 0}
            <p class="pending-text">{$t('chat.subagents.empty')}</p>
          {:else}
            {#each subAgentModalMessages as item}
              <article class="subagent-msg {item.role}">
                <div class="meta">
                  <strong>{item.role === 'assistant' ? 'assistant' : item.role}</strong>
                  {#if item.toolName}<span>{item.toolName}</span>{/if}
                </div>
                {#if item.role === 'assistant'}
                  <div class="markdown">{@html markdownToHTML(item.content || '')}</div>
                {:else if item.role === 'user'}
                  <p>{item.content}</p>
                {:else if item.role === 'toolCall'}
                  <div class="tool-call-body embedded">
                    <div class="tool-title">
                      <span class="dot running"></span>
                      <strong>{item.callView?.label || item.toolName}</strong>
                      {#if item.callView?.target}<span class="tool-target">{item.callView.target}</span>{/if}
                    </div>
                    {#if item.callView?.details?.length}
                      <div class="tool-call-tags">
                        {#each item.callView.details as detail}
                          <span>{detail}</span>
                        {/each}
                      </div>
                    {/if}
                    {#if item.callView?.kind === 'browser'}
                      <div class="browser-call">
                        <div class="find-row">
                          <span>{$t('chat.tool.browser.action')}</span>
                          <code>{item.callView.action || $t('chat.tool.browser.missing')}</code>
                        </div>
                        {#if item.callView.url}
                          <div class="find-row">
                            <span>{$t('chat.tool.browser.url')}</span>
                            <code>{item.callView.url}</code>
                          </div>
                        {/if}
                        {#if item.callView.selector}
                          <div class="find-row">
                            <span>{$t('chat.tool.browser.selectorLabel')}</span>
                            <code>{item.callView.selector}</code>
                          </div>
                        {/if}
                      </div>
                    {:else if item.callView?.kind === 'skill-ref'}
                      <div class="skill-ref-call">
                        <div class="find-row">
                          <span>{$t('chat.tool.skillRef.skillLabel')}</span>
                          <code>{item.callView.skill || $t('chat.tool.skillRef.missing')}</code>
                        </div>
                        <div class="find-row">
                          <span>{$t('chat.tool.skillRef.refLabel')}</span>
                          <code>{item.callView.ref || $t('chat.tool.skillRef.missing')}</code>
                        </div>
                      </div>
                    {:else if item.callView?.kind === 'workflow-lint'}
                      <div class="workflow-lint-call">
                        <div class="write-call-head">
                          <strong>{$t('chat.tool.workflowLint.source')}</strong>
                          <span>{$t('chat.tool.write.summary', { lines: item.callView.lines, chars: item.callView.chars })}</span>
                        </div>
                        <pre class:empty={item.callView.source === ''}>{item.callView.source || $t('chat.tool.workflowLint.missing')}</pre>
                      </div>
                    {:else if item.callView?.kind === 'subagent-task'}
                      <div class="subagent-call">
                        <span>{$t('chat.tool.subagent.task')}</span>
                        <p>{item.callView.task || item.callView.target}</p>
                      </div>
                    {:else if item.callView?.kind === 'subagent-handle'}
                      <div class="subagent-call compact">
                        <div class="find-row">
                          <span>{$t('chat.tool.subagent.handle')}</span>
                          <code>{item.callView.handle || $t('chat.tool.subagent.handleMissing')}</code>
                        </div>
                      </div>
                    {/if}
                  </div>
                {:else if item.role === 'toolResult'}
                  <div class="tool-mini">
                    <span class="dot {item.isError ? 'error' : 'done'}"></span>
                    <strong>{item.toolName}</strong>
                    <span>{item.summary}</span>
                  </div>
                {:else if item.role === 'status'}
                  <div class="tool-mini">
                    <span class="dot {item.isError ? 'error' : 'done'}"></span>
                    <strong>{subAgentStatusLabel(item.content)}</strong>
                    {#if item.summary}<span>{item.summary}</span>{/if}
                  </div>
                {:else}
                  <pre>{item.content || formatArgs(item.arguments)}</pre>
                {/if}
              </article>
            {/each}
          {/if}
        </section>
      </div>
    </div>
  </div>
{/if}
