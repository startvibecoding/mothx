// Approval events may arrive before a normal chat/transcript SSE chunk carries
// x_session_id. Keep this decision independent from component state so the
// approval response always reaches the session that requested it.
export function approvalSessionID(approval, currentSessionID = '') {
  const requestedID = typeof approval?.sessionId === 'string' ? approval.sessionId.trim() : '';
  if (requestedID) return requestedID;
  return typeof currentSessionID === 'string' ? currentSessionID.trim() : '';
}

// An approval frame is sufficient to establish a just-created chat session.
// Do not switch an already-selected session based on an unrelated/replayed
// approval event; responses still target approvalSessionID() above.
export function shouldAdoptApprovalSession(currentSessionID, approval) {
  return !String(currentSessionID || '').trim() && Boolean(approvalSessionID(approval));
}

// Only the selected session may update the visible approval center. A stream
// from a session that the user has left can still deliver events while its
// request is winding down; rendering those events in the new session would
// incorrectly expose and count another session's pending approval.
export function approvalRequestOwnership(currentSessionID, approval) {
  const currentID = String(currentSessionID || '').trim();
  const sessionID = approvalSessionID(approval, currentID);
  const adopt = shouldAdoptApprovalSession(currentID, approval);
  return { sessionID, adopt, belongs: adopt || Boolean(currentID && sessionID === currentID) };
}

// Update a runtime snapshot only if this approval belongs to it. Keeping this
// pure makes session-switch behavior testable and prevents stale stream frames
// from leaking pending state into the newly selected session.
export function applyApprovalRequestToRuntime(runtime, currentSessionID, approval) {
  const ownership = approvalRequestOwnership(currentSessionID, approval);
  if (!ownership.belongs || !approval?.approvalId) return runtime;
  const pending = runtime?.pendingApprovals || [];
  return {
    ...runtime,
    pendingApprovals: [...pending.filter((item) => item?.approvalId !== approval.approvalId), approval]
  };
}

export function approvalBelongsToSession(currentSessionID, approval) {
  const currentID = String(currentSessionID || '').trim();
  return Boolean(currentID && approvalSessionID(approval, currentID) === currentID);
}

// Approval resolutions are persisted as approval_resolved run events. Derive
// the audit list from those records on every session load rather than retaining
// only decisions observed in this browser page.
export function approvalHistoryFromRunEvents(events = []) {
  return (events || [])
    .filter((event) => event?.eventType === 'approval_resolved')
    .map((event) => event.data?.resolution || event.data || {})
    .filter((resolution) => resolution?.approvalId)
    .sort((a, b) => String(b.timestamp || '').localeCompare(String(a.timestamp || '')));
}
