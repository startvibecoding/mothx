import test from 'node:test';
import assert from 'node:assert/strict';
import { get } from 'svelte/store';
import {
  sessionRunStates,
  ensureSessionState,
  getSessionState,
  updateSessionState,
  registerCompletion,
  abortCompletion,
  clearCompletion,
  eventBelongsToSession
} from './session-runs.js';

test.beforeEach(() => sessionRunStates.set({}));

test('creates isolated session state', () => {
  ensureSessionState('a');
  ensureSessionState('b');
  updateSessionState('a', (state) => ({ ...state, messages: [{ role: 'user', content: 'A' }] }));
  assert.equal(getSessionState('a').messages.length, 1);
  assert.equal(getSessionState('b').messages.length, 0);
  assert.notEqual(getSessionState('a'), getSessionState('b'));
});

test('aborts only the bound session completion', () => {
  const calls = [];
  const a = { abort: () => calls.push('a') };
  const b = { abort: () => calls.push('b') };
  registerCompletion('a', a);
  registerCompletion('b', b);
  assert.equal(abortCompletion('a'), true);
  assert.deepEqual(calls, ['a']);
  assert.equal(getSessionState('a').completion.status, 'cancel_requested');
  assert.equal(getSessionState('b').completion.status, 'starting');
});

test('does not clear a replacement completion controller', () => {
  const first = { abort() {} };
  const second = { abort() {} };
  registerCompletion('a', first);
  updateSessionState('a', (state) => ({
    ...state,
    completion: { ...state.completion, controller: second }
  }));
  clearCompletion('a', first);
  assert.equal(getSessionState('a').completion.controller, second);
});

test('rejects payloads belonging to another session', () => {
  assert.equal(eventBelongsToSession('a', { sessionId: 'a' }), true);
  assert.equal(eventBelongsToSession('a', { x_session_id: 'a' }), true);
  assert.equal(eventBelongsToSession('a', {}), true);
  assert.equal(eventBelongsToSession('a', { sessionId: 'b' }), false);
});

test('keeps per-session cursors independent', () => {
  updateSessionState('a', (state) => ({ ...state, cursor: { ...state.cursor, entrySeq: 8 } }));
  updateSessionState('b', (state) => ({ ...state, cursor: { ...state.cursor, runSeq: 3 } }));
  const states = get(sessionRunStates);
  assert.equal(states.a.cursor.entrySeq, 8);
  assert.equal(states.a.cursor.runSeq, 0);
  assert.equal(states.b.cursor.entrySeq, 0);
  assert.equal(states.b.cursor.runSeq, 3);
});
