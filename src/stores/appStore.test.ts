/**
 * Unit tests for appStore
 * Requirements: 1.1, 2.1, 3.1, 5.1, 8.1
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { useAppStore } from './appStore';
import type { MCPServer, LogEntry, ConfirmationRequest } from '../types/models';

describe('appStore', () => {
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

  describe('Session Management (Requirement 8.1)', () => {
    it('should create a new session', () => {
      const sessionId = useAppStore.getState().createSession();
      const state = useAppStore.getState();

      expect(sessionId).toBeTruthy();
      expect(state.currentSessionId).toBe(sessionId);
      expect(state.sessions.has(sessionId)).toBe(true);
      expect(state.messages.has(sessionId)).toBe(true);
    });

    it('should load an existing session', () => {
      const sessionId = useAppStore.getState().createSession();
      const newSessionId = useAppStore.getState().createSession();

      let state = useAppStore.getState();
      expect(state.currentSessionId).toBe(newSessionId);

      useAppStore.getState().loadSession(sessionId);
      state = useAppStore.getState();
      expect(state.currentSessionId).toBe(sessionId);
    });
  });

  describe('Message Management (Requirement 1.1)', () => {
    it('should send a user message', () => {
      const sessionId = useAppStore.getState().createSession();

      useAppStore.getState().sendMessage('Hello, Agent!');

      const state = useAppStore.getState();
      const messages = state.messages.get(sessionId);
      expect(messages).toHaveLength(1);
      expect(messages![0].role).toBe('user');
      expect(messages![0].content).toBe('Hello, Agent!');
    });

    it('should handle streaming messages', () => {
      const sessionId = useAppStore.getState().createSession();
      const messageId = 'test-message-id';

      useAppStore.getState().appendStreamingContent(messageId, 'Hello');
      useAppStore.getState().appendStreamingContent(messageId, ' World');

      let state = useAppStore.getState();
      expect(state.streamingMessages.get(messageId)).toBe('Hello World');

      useAppStore.getState().finalizeStreamingMessage(messageId);

      state = useAppStore.getState();
      const messages = state.messages.get(sessionId);
      expect(messages).toHaveLength(1);
      expect(messages![0].content).toBe('Hello World');
      expect(state.streamingMessages.has(messageId)).toBe(false);
    });
  });

  describe('MCP Server Management (Requirement 2.1)', () => {
    it('should set MCP servers', () => {
      const servers: MCPServer[] = [
        {
          id: 'server-1',
          name: 'Test Server',
          status: 'online',
          tools: [],
        },
      ];

      useAppStore.getState().setMCPServers(servers);

      const state = useAppStore.getState();
      expect(state.mcpServers).toHaveLength(1);
      expect(state.mcpServers[0].id).toBe('server-1');
    });

    it('should update MCP server status', () => {
      const servers: MCPServer[] = [
        {
          id: 'server-1',
          name: 'Test Server',
          status: 'online',
          tools: [],
        },
      ];

      useAppStore.getState().setMCPServers(servers);
      useAppStore.getState().updateMCPStatus('server-1', 'offline');

      const state = useAppStore.getState();
      expect(state.mcpServers[0].status).toBe('offline');
      expect(state.mcpServers[0].lastHeartbeat).toBeDefined();
    });
  });

  describe('Log Management (Requirement 3.1)', () => {
    it('should add log entries', () => {
      const log: LogEntry = {
        id: 'log-1',
        timestamp: Date.now(),
        level: 'info',
        message: 'Test log message',
      };

      useAppStore.getState().addLog(log);

      const state = useAppStore.getState();
      expect(state.logs).toHaveLength(1);
      expect(state.logs[0].message).toBe('Test log message');
    });

    it('should clear logs', () => {
      const log: LogEntry = {
        id: 'log-1',
        timestamp: Date.now(),
        level: 'info',
        message: 'Test log message',
      };

      useAppStore.getState().addLog(log);
      let state = useAppStore.getState();
      expect(state.logs).toHaveLength(1);

      useAppStore.getState().clearLogs();
      state = useAppStore.getState();
      expect(state.logs).toHaveLength(0);
    });
  });

  describe('Confirmation Management (Requirement 5.1)', () => {
    it('should add confirmation requests', () => {
      const request: ConfirmationRequest = {
        id: 'confirm-1',
        operation: 'Delete Database',
        description: 'This will delete all data',
        riskLevel: 'high',
        timeout: 30000,
      };

      useAppStore.getState().addConfirmation(request);

      const state = useAppStore.getState();
      expect(state.pendingConfirmations).toHaveLength(1);
      expect(state.pendingConfirmations[0].operation).toBe('Delete Database');
    });

    it('should remove confirmation requests', () => {
      const request: ConfirmationRequest = {
        id: 'confirm-1',
        operation: 'Delete Database',
        description: 'This will delete all data',
        riskLevel: 'high',
        timeout: 30000,
      };

      useAppStore.getState().addConfirmation(request);
      let state = useAppStore.getState();
      expect(state.pendingConfirmations).toHaveLength(1);

      useAppStore.getState().removeConfirmation('confirm-1');
      state = useAppStore.getState();
      expect(state.pendingConfirmations).toHaveLength(0);
    });
  });

  describe('WebSocket State Management (Requirement 9.1)', () => {
    it('should set WebSocket connection status', () => {
      useAppStore.getState().setWSConnected(true);
      let state = useAppStore.getState();
      expect(state.wsConnected).toBe(true);

      useAppStore.getState().setWSConnected(false);
      state = useAppStore.getState();
      expect(state.wsConnected).toBe(false);
    });

    it('should set WebSocket error', () => {
      useAppStore.getState().setWSError('Connection failed');
      let state = useAppStore.getState();
      expect(state.wsError).toBe('Connection failed');

      useAppStore.getState().setWSError(null);
      state = useAppStore.getState();
      expect(state.wsError).toBe(null);
    });
  });
});
