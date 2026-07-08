// Shared reactive stores. Views subscribe; a small refresh helper reloads
// everything after significant server-state changes.

import { writable, derived, get } from 'svelte/store';
import { request } from './api.js';

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
export const notice = writable('');
export const error = writable('');
export const currentSession = writable('');
export const selectedModel = writable('default');

const sessionToolStorageKey = 'mothx.webui.sessionTools';
const defaultSessionTools = {
  webSearch: false,
  browser: false,
  a2aMaster: false,
  delegate: false,
  multiAgent: false
};
export const sessionToolOptions = writable(loadSessionToolOptions());

export const features = derived(status, ($s) => ({
  api: $s?.features?.openaiAPI !== false,
  webUI: $s?.features?.webUI !== false,
  websocket: $s?.features?.websocket === true,
  cron: $s?.features?.cron === true,
  memory: $s?.features?.memory === true,
  multiAgent: $s?.features?.multiAgent === true,
  delegate: $s?.features?.delegate === true,
  webSearch: $s?.features?.webSearch === true,
  browser: $s?.features?.browser === true,
  a2aMaster: $s?.features?.a2aMaster === true
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
    await refreshModels();
  } catch (err) {
    setError(err);
  }
}

export async function refreshSessions() {
  const data = await request('/api/sessions');
  sessions.set(data?.sessions || []);
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

export async function refreshCron() {
  cronInfo.set(await request('/api/cron'));
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
    if (list.length > 0 && !list.some((m) => m.id === current)) {
      selectedModel.set(list[0].id);
    }
  } catch {
    models.set([]);
    selectedModel.set('default');
  }
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
    multiAgent: Boolean(value.multiAgent)
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
