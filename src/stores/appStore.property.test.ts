/**
 * Property-based tests for appStore
 * Feature: ops-genius-frontend, Property 24: 新会话创建隔离性
 * Validates: Requirements 8.3
 */

import { describe, it, beforeEach, expect } from 'vitest';
import fc from 'fast-check';
import { useAppStore } from './appStore';
import type { Message } from '../types/models';

describe('AppStore Property Tests', () => {
  beforeEach(() => {
    // Reset store state before each test
    const store = useAppStore.getState();
    store.sessions.clear();
    store.messages.clear();
    store.streamingMessages.clear();
    store.mcpServers = [];
    store.logs = [];
    store.pendingConfirmations = [];
    store.currentSessionId = '';
    store.selectedAgentId = null;
    store.contextLogs = [];
    store.wsConnected = false;
    store.wsError = null;
  });

  // Feature: ops-genius-frontend, Property 24: 新会话创建隔离性
  it('new session should have different ID and empty message list', () => {
    fc.assert(
      fc.property(
        // Generate arbitrary messages for the current session
        fc.array(
          fc.record({
            id: fc.string({ minLength: 1 }),
            role: fc.constantFrom('user' as const, 'agent' as const),
            content: fc.string(),
            timestamp: fc.integer({ min: 0, max: Date.now() }),
            streaming: fc.option(fc.boolean(), { nil: undefined }),
          }),
          { minLength: 0, maxLength: 20 }
        ),
        (messages: Message[]) => {
          // Reset store state for each iteration
          const store = useAppStore.getState();
          store.sessions.clear();
          store.messages.clear();
          store.currentSessionId = '';

          // Create initial session
          const firstSessionId = store.createSession();
          
          // Add messages to the first session
          messages.forEach((message) => {
            store.addMessageToSession(firstSessionId, message);
          });

          // Get state after adding messages
          const stateBeforeNewSession = useAppStore.getState();
          const firstSessionMessages = stateBeforeNewSession.messages.get(firstSessionId) || [];

          // Create a new session
          const secondSessionId = store.createSession();

          // Get state after creating new session
          const stateAfterNewSession = useAppStore.getState();
          const secondSessionMessages = stateAfterNewSession.messages.get(secondSessionId) || [];

          // Property 24: New session should have different ID
          expect(secondSessionId).not.toBe(firstSessionId);

          // Property 24: New session should have empty message list
          expect(secondSessionMessages).toHaveLength(0);

          // Property 24: Current session ID should be the new session
          expect(stateAfterNewSession.currentSessionId).toBe(secondSessionId);

          // Property 24: First session messages should remain unchanged
          const firstSessionMessagesAfter = stateAfterNewSession.messages.get(firstSessionId) || [];
          expect(firstSessionMessagesAfter).toHaveLength(firstSessionMessages.length);
        }
      ),
      { numRuns: 100 }
    );
  });
});
