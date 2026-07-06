export function formatTime(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleTimeString();
}

export function formatDateTime(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString();
}

export function shortID(value) {
  if (!value) return 'default';
  if (value.length <= 18) return value;
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}

export function scheduleLabel(job) {
  if (job?.oneshot || !job?.schedule) return 'one-shot';
  return job.schedule;
}

export function toolStateClass(item) {
  if (item?.status === 'running') return 'running';
  if (item?.status === 'error' || item?.status === 'failed') return 'error';
  return 'done';
}

export function formatLogMessage(item) {
  if (!item) return '';
  if (item.message) return item.message;
  return JSON.stringify(item.status || item);
}

export function formatArgs(value) {
  if (!value) return '';
  return JSON.stringify(value, null, 2);
}
