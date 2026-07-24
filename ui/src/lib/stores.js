// Shared reactive stores. Views subscribe; a small refresh helper reloads
// everything after significant server-state changes.

import { writable, derived, readable, get } from 'svelte/store';
import { request, jsonBody } from './api.js';

// Reactive media-query store: true when viewport is mobile-width.
export const isMobile = readable(false, (set) => {
  if (typeof window === 'undefined') return;
  const mql = window.matchMedia('(max-width: 900px)');
  set(mql.matches);
  const handler = (e) => set(e.matches);
  mql.addEventListener('change', handler);
  return () => mql.removeEventListener('change', handler);
});

export const health = writable(null);
export const status = writable(null);
export const channels = writable([]);
export const sessions = writable([]);
export const models = writable([]);
export const cronInfo = writable(null);
export const serveConfig = writable('');
export const settings = writable('');
export const memoryInfo = writable(null);
export const memory = writable('');
export const logs = writable([]);
export const logsConnected = writable(false);
export const statsSummary = writable(null);
export const notice = writable('');
export const error = writable('');
export const currentSession = writable('');
export const sidebarOpen = writable(false);
export const selectedModel = writable('default');
export const sessionRuntime = writable(null);
export const pendingApprovals = derived(sessionRuntime, ($runtime) => $runtime?.pendingApprovals || []);
export const activeApproval = writable(null);
export const approvalHistory = writable([]);
export const toolEvents = writable([]);

const sessionToolStorageKey = 'mothx.webui.sessionTools';
const defaultSessionTools = {
  webSearch: false,
  browser: false,
  a2aMaster: false,
  delegate: false,
  multiAgent: false,
  workflows: false
};
export const sessionToolOptions = writable(loadSessionToolOptions());

export const features = derived(status, ($s) => ({
  api: $s?.features?.openaiAPI !== false,
  webUI: $s?.features?.webUI !== false,
  websocket: $s?.features?.websocket === true,
  cron: $s?.features?.cron !== false,
  memory: $s?.features?.memory !== false,
  multiAgent: $s?.features?.multiAgent === true,
  delegate: $s?.features?.delegate === true,
  webSearch: $s?.features?.webSearch === true,
  browser: $s?.features?.browser === true,
  a2aMaster: $s?.features?.a2aMaster === true,
  workflows: $s?.features?.workflows === true
}));

export const connectedChannels = derived(channels, ($c) =>
  $c.filter((ch) => ch.connected).length
);

export function setError(err) {
  error.set(err instanceof Error ? err.message : String(err || ''));
}

export function setNotice(msg) {
  notice.set(msg || '');
}

export function clearBanners() {
  error.set('');
  notice.set('');
}

let logsSocket = null;

export function connectLogs() {
  if (logsSocket) return;
  const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
  logsSocket = new WebSocket(`${scheme}://${window.location.host}/ws/logs`);
  logsSocket.onopen = () => logsConnected.set(true);
  logsSocket.onmessage = (event) => {
    try {
      const item = JSON.parse(event.data);
      if (item.type === 'heartbeat') return;
      logs.update((prev) => [...prev.slice(-199), item]);
      if (item.type === 'connected' && item.status) status.set(item.status);
    } catch {
      logs.update((prev) => [...prev.slice(-199), { type: 'log', message: event.data }]);
    }
  };
  logsSocket.onclose = () => {
    logsConnected.set(false);
    logsSocket = null;
  };
  logsSocket.onerror = () => logsConnected.set(false);
}

export function disconnectLogs() {
  if (logsSocket) logsSocket.close();
  logsSocket = null;
  logsConnected.set(false);
}

export async function refreshAll() {
  error.set('');
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
    health.set(h);
    status.set(st);
    channels.set(c || []);
    sessions.set(sess?.sessions || []);
    cronInfo.set(cron);
    serveConfig.set(JSON.stringify(sc, null, 2));
    settings.set(JSON.stringify(s, null, 2));
    memoryInfo.set(mem);
    memory.set(mem?.content || '');
    await Promise.all([refreshModels(), refreshStatsSummary()]);
  } catch (err) {
    setError(err);
  }
}

export async function refreshSessions() {
  const data = await request('/api/sessions');
  sessions.set(data?.sessions || []);
}

export function upsertSession(session) {
  if (!session?.id) return;
  sessions.update((items) => {
    const incoming = normalizeSessionListEntry(session);
    const idx = (items || []).findIndex((item) => item?.id === incoming.id);
    let next;
    if (idx >= 0) {
      next = items.slice();
      next[idx] = { ...next[idx], ...incoming };
    } else {
      next = [incoming, ...(items || [])];
    }
    return sortSessions(next);
  });
}

function normalizeSessionListEntry(session) {
  return {
    ...session,
    lastUsed: session.lastUsed || new Date().toISOString(),
    messageCount: Number(session.messageCount || 0)
  };
}

function sortSessions(items = []) {
  return [...items].sort((a, b) => {
    const left = Date.parse(a?.lastUsed || '') || 0;
    const right = Date.parse(b?.lastUsed || '') || 0;
    if (left === right) return String(a?.id || '').localeCompare(String(b?.id || ''));
    return right - left;
  });
}

export async function refreshStatsSummary() {
  try {
    statsSummary.set(await request('/api/stats/summary'));
  } catch {
    statsSummary.set(null);
  }
}

function statsQuery(params = {}) {
  const q = new URLSearchParams();
  for (const [key, value] of Object.entries(params || {})) {
    if (value !== undefined && value !== null && value !== '') q.set(key, value);
  }
  const query = q.toString();
  return query ? `?${query}` : '';
}

export async function getStatsSummary(params = {}) {
  return request(`/api/stats/summary${statsQuery(params)}`);
}

export async function getStatsTimeSeries(params = {}) {
  return request(`/api/stats/timeseries${statsQuery(params)}`);
}

export async function getStatsByProvider(params = {}) {
  return request(`/api/stats/by-provider${statsQuery(params)}`);
}

export async function getStatsByModel(params = {}) {
  return request(`/api/stats/by-model${statsQuery(params)}`);
}

export async function getStatsRecent(params = {}) {
  return request(`/api/stats/recent${statsQuery(params)}`);
}

export async function getSessionMessages(id) {
  if (!id) return []; // default session with no messages endpoint
  try {
    const data = await request(`/api/sessions/${encodeURIComponent(id)}/messages`);
    return data?.messages || [];
  } catch {
    return [];
  }
}

export async function getSessionToolResult(id, toolCallID) {
  if (!id || !toolCallID) return null;
  return request(
    `/api/sessions/${encodeURIComponent(id)}/tool-results/${encodeURIComponent(toolCallID)}`
  );
}

export async function getSessionSubAgents(id) {
  if (!id) return [];
  try {
    const data = await request(`/api/sessions/${encodeURIComponent(id)}/subagents`);
    return data?.subagents || [];
  } catch {
    return [];
  }
}

export async function getSessionSubAgentMessages(id, agentID) {
  if (!id || !agentID) return [];
  try {
    const data = await request(
      `/api/sessions/${encodeURIComponent(id)}/subagents/${encodeURIComponent(agentID)}/messages`
    );
    return data?.messages || [];
  } catch {
    return [];
  }
}

export async function getSessionRunEvents(id) {
  if (!id) return [];
  try {
    const data = await request(`/api/sessions/${encodeURIComponent(id)}/run-events`);
    return data?.events || [];
  } catch {
    return [];
  }
}

export async function getSessionCapabilityEvents(id) {
  if (!id) return [];
  try {
    const data = await request(`/api/sessions/${encodeURIComponent(id)}/capability-events`);
    return data?.events || [];
  } catch {
    return [];
  }
}

export async function getSessionRuntime(id) {
  if (!id) return null;
  return request(`/api/sessions/${encodeURIComponent(id)}/runtime`);
}

export async function patchSessionRuntime(id, patch) {
  if (!id) throw new Error('session ID is required');
  return request(`/api/sessions/${encodeURIComponent(id)}/runtime`, {
    method: 'PATCH',
    ...jsonBody(patch)
  });
}

export async function refreshCron(sessionId = '') {
  const query = sessionId ? `?sessionId=${encodeURIComponent(sessionId)}` : '';
  cronInfo.set(await request(`/api/cron${query}`));
}

export async function refreshModels() {
  const st = get(status);
  if (st?.features?.openaiAPI === false) {
    models.set([]);
    selectedModel.set('default');
    return;
  }
  try {
    const data = await request('/v1/models');
    const list = data?.data || [];
    models.set(list);
    const current = get(selectedModel);
    if (list.length > 0 && (!current || current === 'default' || !list.some((m) => m.id === current))) {
      selectedModel.set(defaultModelForList(list));
    }
  } catch {
    models.set([]);
    selectedModel.set('default');
  }
}

export function resetSelectedModelToDefault() {
  const list = get(models);
  selectedModel.set(defaultModelForList(list));
}

function defaultModelForList(list = []) {
  if (!Array.isArray(list) || list.length === 0) return 'default';
  const ids = new Set(list.map((m) => m?.id).filter(Boolean));
  const serve = parseJSONStore(serveConfig);
  const cfg = parseJSONStore(settings);
  const serveModel = stringValue(serve?.api?.model);
  if (serveModel && ids.has(serveModel)) return serveModel;
  const settingsModel = stringValue(cfg?.defaultModel);
  if (settingsModel && ids.has(settingsModel)) return settingsModel;
  return list[0]?.id || 'default';
}

function parseJSONStore(store) {
  try {
    const raw = get(store);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

function stringValue(value) {
  return typeof value === 'string' ? value.trim() : '';
}

function loadSessionToolOptions() {
  if (typeof window === 'undefined') return {};
  try {
    const parsed = JSON.parse(window.localStorage.getItem(sessionToolStorageKey) || '{}');
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {};
  } catch {
    return {};
  }
}

function saveSessionToolOptions(value) {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(sessionToolStorageKey, JSON.stringify(value || {}));
}

function normalizeSessionTools(value = {}) {
  return {
    webSearch: Boolean(value.webSearch),
    browser: Boolean(value.browser),
    a2aMaster: Boolean(value.a2aMaster),
    delegate: Boolean(value.delegate ?? value.delegateMode),
    multiAgent: Boolean(value.multiAgent),
    workflows: Boolean(value.workflows)
  };
}

export function sessionToolsFor(map, id, fallback = null) {
  const key = id || '__new__';
  const base = fallback ? normalizeSessionTools(fallback) : { ...defaultSessionTools };
  return normalizeSessionTools({ ...base, ...(map?.[key] || {}) });
}

export function setSessionTools(id, value) {
  const key = id || '__new__';
  const normalized = normalizeSessionTools(value);
  sessionToolOptions.update((prev) => {
    const next = { ...(prev || {}), [key]: normalized };
    saveSessionToolOptions(next);
    return next;
  });
}

export function moveSessionTools(fromID, toID) {
  const from = fromID || '__new__';
  const to = toID || '__new__';
  if (!from || !to || from === to) return;
  sessionToolOptions.update((prev) => {
    if (!prev?.[from]) return prev || {};
    const next = { ...(prev || {}), [to]: prev[from] };
    delete next[from];
    saveSessionToolOptions(next);
    return next;
  });
}
