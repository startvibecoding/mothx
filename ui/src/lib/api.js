// Thin HTTP + SSE helpers shared across views.
// Keeps fetch/JSON error handling in one place so views stay declarative.

const jsonHeaders = { 'Content-Type': 'application/json' };

export async function request(path, options = {}) {
  const res = await fetch(path, options);
  const text = await res.text();
  const data = text ? safeJSON(text) : null;
  if (!res.ok) {
    const msg =
      data?.error?.message ||
      data?.error ||
      data?.message ||
      `${res.status} ${res.statusText}`;
    throw new Error(msg);
  }
  return data;
}

export function jsonBody(value) {
  return { headers: jsonHeaders, body: JSON.stringify(value) };
}

export function putJSON(path, value) {
  return request(path, { method: 'PUT', ...jsonBody(value) });
}

export function postJSON(path, value) {
  return request(path, { method: 'POST', ...jsonBody(value) });
}

export function patchJSON(path, value) {
  return request(path, { method: 'PATCH', ...jsonBody(value) });
}

export function del(path) {
  return request(path, { method: 'DELETE' });
}

function safeJSON(text) {
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

// Consume an SSE body and emit parsed events. Callers own the abort controller.
export async function readSSE(body, onEvent) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  const flush = (final = false) => {
    buffer = buffer.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
    let idx = buffer.indexOf('\n\n');
    while (idx !== -1) {
      dispatch(buffer.slice(0, idx));
      buffer = buffer.slice(idx + 2);
      idx = buffer.indexOf('\n\n');
    }
    if (final && buffer.trim()) {
      dispatch(buffer);
      buffer = '';
    }
  };

  const dispatch = (raw) => {
    const event = parseSSEBlock(raw);
    if (!event || event.data === '') return;
    onEvent(event);
  };

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
