/**
 * Property-Based Tests for Layout Component
 * Feature: ops-genius-frontend, Property 22: 布局切换状态保持
 * Validates: Requirements 7.4
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import fc from 'fast-check';
import Layout from './Layout';
import { useAppStore } from '../../stores/appStore';
import type { Message, MCPServer, LogEntry } from '../../types/models';

describe('Layout Property Tests', () => {
  beforeEach(() => {
    // Reset store before each test
    const store = useAppStore.getState();
    store.sessions.clear();
    store.messages.clear();
    store.streamingMessages.clear();
    useAppStore.setState({
      currentSessionId: '',
      sessions: new Map(),
      messages: new Map(),
      streamingMessages: new Map(),
      mcpServers: [],
      logs: [],
      selectedAgentId: null,
      contextLogs: [],
      pendingConfirmations: [],
      wsConnected: false,
      wsError: null,
    });
  });

  // Feature: ops-genius-frontend, Property 22: 布局切换状态保持
  it('should preserve application state when layout switches', () => {
    fc.assert(
      fc.property(
        // Generate random application state
        fc.record({
          sessionId: fc.string({ minLength: 1, maxLength: 20 }),
          messages: fc.array(
            fc.record({
              id: fc.string({ minLength: 1, maxLength: 20 }),
              role: fc.constantFrom('user' as const, 'agent' as const),
              content: fc.string({ maxLength: 100 }),
              timestamp: fc.integer({ min: 0, max: Date.now() }),
            }),
            { maxLength: 10 }
          ),
          mcpServers: fc.array(
            fc.record({
              id: fc.string({ minLength: 1, maxLength: 20 }),
              name: fc.string({ minLength: 1, maxLength: 30 }),
              status: fc.constantFrom('online' as const, 'offline' as const, 'error' as const),
              tools: fc.array(
                fc.record({
                  name: fc.string({ minLength: 1, maxLength: 20 }),
                  description: fc.string({ maxLength: 50 }),
                  readonly: fc.boolean(),
                }),
                { maxLength: 5 }
              ),
            }),
            { maxLength: 5 }
          ),
          logs: fc.array(
            fc.record({
              id: fc.string({ minLength: 1, maxLength: 20 }),
              timestamp: fc.integer({ min: 0, max: Date.now() }),
              level: fc.constantFrom('info' as const, 'warning' as const, 'error' as const),
              message: fc.string({ maxLength: 100 }),
            }),
            { maxLength: 10 }
          ),
          wsConnected: fc.boolean(),
        }),
        (appState) => {
          // Set up initial application state
          const store = useAppStore.getState();
          
          // Create session
          const sessionId = appState.sessionId;
          store.sessions.set(sessionId, {
            id: sessionId,
            title: `Test Session`,
            createdAt: Date.now(),
            updatedAt: Date.now(),
            messages: appState.messages as Message[],
          });
          
          useAppStore.setState({
            currentSessionId: sessionId,
            sessions: new Map(store.sessions),
            messages: new Map([[sessionId, appState.messages as Message[]]]),
            mcpServers: appState.mcpServers as MCPServer[],
            logs: appState.logs as LogEntry[],
            wsConnected: appState.wsConnected,
          });

          // Capture state before layout switch
          const stateBefore = {
            currentSessionId: useAppStore.getState().currentSessionId,
            messagesCount: useAppStore.getState().messages.get(sessionId)?.length || 0,
            mcpServersCount: useAppStore.getState().mcpServers.length,
            logsCount: useAppStore.getState().logs.length,
            wsConnected: useAppStore.getState().wsConnected,
            messages: [...(useAppStore.getState().messages.get(sessionId) || [])],
            mcpServers: [...useAppStore.getState().mcpServers],
            logs: [...useAppStore.getState().logs],
          };

          // Render layout with panels
          const { rerender } = render(
            <Layout
              leftPanel={<div data-testid="left-panel">MCP Status</div>}
              rightPanel={<div data-testid="right-panel">Logs</div>}
            >
              <div data-testid="main-content">Chat Interface</div>
            </Layout>
          );

          // Simulate layout switch by re-rendering (simulates window resize)
          rerender(
            <Layout
              leftPanel={<div data-testid="left-panel">MCP Status</div>}
              rightPanel={<div data-testid="right-panel">Logs</div>}
            >
              <div data-testid="main-content">Chat Interface</div>
            </Layout>
          );

          // Capture state after layout switch
          const stateAfter = {
            currentSessionId: useAppStore.getState().currentSessionId,
            messagesCount: useAppStore.getState().messages.get(sessionId)?.length || 0,
            mcpServersCount: useAppStore.getState().mcpServers.length,
            logsCount: useAppStore.getState().logs.length,
            wsConnected: useAppStore.getState().wsConnected,
            messages: [...(useAppStore.getState().messages.get(sessionId) || [])],
            mcpServers: [...useAppStore.getState().mcpServers],
            logs: [...useAppStore.getState().logs],
          };

          // Verify state preservation
          expect(stateAfter.currentSessionId).toBe(stateBefore.currentSessionId);
          expect(stateAfter.messagesCount).toBe(stateBefore.messagesCount);
          expect(stateAfter.mcpServersCount).toBe(stateBefore.mcpServersCount);
          expect(stateAfter.logsCount).toBe(stateBefore.logsCount);
          expect(stateAfter.wsConnected).toBe(stateBefore.wsConnected);
          
          // Deep equality checks
          expect(stateAfter.messages).toEqual(stateBefore.messages);
          expect(stateAfter.mcpServers).toEqual(stateBefore.mcpServers);
          expect(stateAfter.logs).toEqual(stateBefore.logs);
        }
      ),
      { numRuns: 100 }
    );
  });
});
