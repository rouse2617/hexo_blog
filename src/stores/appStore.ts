/**
 * Global Application Store using Zustand
 * Requirements: 1.1, 2.1, 3.1, 5.1, 8.1, 9.1
 * 
 * Manages all application state including sessions, messages, MCP servers,
 * logs, and confirmation requests.
 */

import { create } from 'zustand';
import type {
  Message,
  MCPServer,
  LogEntry,
  ConfirmationRequest,
  Session,
  ContextLog,
} from '../types/models';
import { WebSocketManager, createWebSocketManager } from '../utils/websocket';
import type { ServerMessage } from '../types/websocket';

// Global WebSocket manager instance
let wsManager: WebSocketManager | null = null;

/**
 * Application state interface
 */
export interface AppState {
  // Session state (Requirement 8.1)
  currentSessionId: string;
  sessions: Map<string, Session>;

  // Message state (Requirement 1.1)
  messages: Map<string, Message[]>;
  streamingMessages: Map<string, string>;

  // MCP state (Requirement 2.1)
  mcpServers: MCPServer[];

  // Log state (Requirement 3.1)
  logs: LogEntry[];

  // Context modal state (Requirement 4.1)
  selectedAgentId: string | null;
  contextLogs: ContextLog[];

  // Confirmation state (Requirement 5.1)
  pendingConfirmations: ConfirmationRequest[];

  // WebSocket state (Requirement 9.1)
  wsConnected: boolean;
  wsError: string | null;

  // Actions
  sendMessage: (content: string) => void;
  appendStreamingContent: (messageId: string, content: string) => void;
  finalizeStreamingMessage: (messageId: string) => void;
  createSession: () => string;
  loadSession: (sessionId: string) => void;
  updateMCPStatus: (serverId: string, status: 'online' | 'offline' | 'error') => void;
  setMCPServers: (servers: MCPServer[]) => void;
  addLog: (log: LogEntry) => void;
  clearLogs: () => void;
  setContextLogs: (agentId: string, logs: ContextLog[]) => void;
  clearContextLogs: () => void;
  addConfirmation: (request: ConfirmationRequest) => void;
  removeConfirmation: (requestId: string) => void;
  setWSConnected: (connected: boolean) => void;
  setWSError: (error: string | null) => void;
  addMessageToSession: (sessionId: string, message: Message) => void;
  updateSessionTimestamp: (sessionId: string) => void;
  connect: () => void;
  disconnect: () => void;
}

/**
 * Generate a unique ID
 */
function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
}

/**
 * Create the application store
 * Requirements: 1.1, 2.1, 3.1, 5.1, 8.1
 */
export const useAppStore = create<AppState>((set, get) => ({
  // Initial state
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

  /**
   * Send a user message
   * Requirement 1.1: User message submission
   */
  sendMessage: (content: string) => {
    const state = get();
    const sessionId = state.currentSessionId;

    if (!sessionId) {
      console.error('No active session');
      return;
    }

    const message: Message = {
      id: generateId(),
      role: 'user',
      content,
      timestamp: Date.now(),
    };

    // Add message to current session
    const sessionMessages = state.messages.get(sessionId) || [];
    const updatedMessages = new Map(state.messages);
    updatedMessages.set(sessionId, [...sessionMessages, message]);

    // Update session
    const session = state.sessions.get(sessionId);
    if (session) {
      const updatedSessions = new Map(state.sessions);
      updatedSessions.set(sessionId, {
        ...session,
        messages: [...sessionMessages, message],
        updatedAt: Date.now(),
      });

      set({
        messages: updatedMessages,
        sessions: updatedSessions,
      });
    } else {
      set({ messages: updatedMessages });
    }
  },

  /**
   * Append content to a streaming message
   * Requirement 1.2: Streaming response display
   */
  appendStreamingContent: (messageId: string, content: string) => {
    const state = get();
    const currentContent = state.streamingMessages.get(messageId) || '';
    const updatedStreamingMessages = new Map(state.streamingMessages);
    updatedStreamingMessages.set(messageId, currentContent + content);

    set({ streamingMessages: updatedStreamingMessages });
  },

  /**
   * Finalize a streaming message
   * Requirement 1.2: Streaming response completion
   */
  finalizeStreamingMessage: (messageId: string) => {
    const state = get();
    const sessionId = state.currentSessionId;
    const content = state.streamingMessages.get(messageId) || '';

    if (!sessionId) {
      console.error('No active session');
      return;
    }

    const message: Message = {
      id: messageId,
      role: 'agent',
      content,
      timestamp: Date.now(),
      streaming: false,
    };

    // Add finalized message to session
    const sessionMessages = state.messages.get(sessionId) || [];
    const updatedMessages = new Map(state.messages);
    updatedMessages.set(sessionId, [...sessionMessages, message]);

    // Remove from streaming messages
    const updatedStreamingMessages = new Map(state.streamingMessages);
    updatedStreamingMessages.delete(messageId);

    // Update session
    const session = state.sessions.get(sessionId);
    if (session) {
      const updatedSessions = new Map(state.sessions);
      updatedSessions.set(sessionId, {
        ...session,
        messages: [...sessionMessages, message],
        updatedAt: Date.now(),
      });

      set({
        messages: updatedMessages,
        streamingMessages: updatedStreamingMessages,
        sessions: updatedSessions,
      });
    } else {
      set({
        messages: updatedMessages,
        streamingMessages: updatedStreamingMessages,
      });
    }
  },

  /**
   * Create a new session
   * Requirement 8.3: New session creation
   */
  createSession: () => {
    const sessionId = generateId();
    const session: Session = {
      id: sessionId,
      title: `Session ${new Date().toLocaleString()}`,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      messages: [],
    };

    const updatedSessions = new Map(get().sessions);
    updatedSessions.set(sessionId, session);

    const updatedMessages = new Map(get().messages);
    updatedMessages.set(sessionId, []);

    set({
      currentSessionId: sessionId,
      sessions: updatedSessions,
      messages: updatedMessages,
    });

    return sessionId;
  },

  /**
   * Load an existing session
   * Requirement 8.5: Historical session loading
   */
  loadSession: (sessionId: string) => {
    const state = get();
    const session = state.sessions.get(sessionId);

    if (!session) {
      console.error(`Session ${sessionId} not found`);
      return;
    }

    set({ currentSessionId: sessionId });
  },

  /**
   * Update MCP server status
   * Requirement 2.2: MCP Server status updates
   */
  updateMCPStatus: (serverId: string, status: 'online' | 'offline' | 'error') => {
    const state = get();
    const updatedServers = state.mcpServers.map((server) =>
      server.id === serverId
        ? { ...server, status, lastHeartbeat: Date.now() }
        : server
    );

    set({ mcpServers: updatedServers });
  },

  /**
   * Set MCP servers
   * Requirement 2.1: MCP Server list initialization
   */
  setMCPServers: (servers: MCPServer[]) => {
    set({ mcpServers: servers });
  },

  /**
   * Add a log entry
   * Requirement 3.1: Agent log real-time display
   */
  addLog: (log: LogEntry) => {
    const state = get();
    set({ logs: [...state.logs, log] });
  },

  /**
   * Clear all logs
   * Requirement 3.1: Log management
   */
  clearLogs: () => {
    set({ logs: [] });
  },

  /**
   * Set context logs for an agent
   * Requirement 4.1: Context log detail viewing
   */
  setContextLogs: (agentId: string, logs: ContextLog[]) => {
    set({
      selectedAgentId: agentId,
      contextLogs: logs,
    });
  },

  /**
   * Clear context logs
   * Requirement 4.3: Context modal close
   */
  clearContextLogs: () => {
    set({
      selectedAgentId: null,
      contextLogs: [],
    });
  },

  /**
   * Add a confirmation request
   * Requirement 5.1: Confirmation request display
   */
  addConfirmation: (request: ConfirmationRequest) => {
    const state = get();
    set({ pendingConfirmations: [...state.pendingConfirmations, request] });
  },

  /**
   * Remove a confirmation request
   * Requirement 5.3, 5.4: Confirmation response handling
   */
  removeConfirmation: (requestId: string) => {
    const state = get();
    set({
      pendingConfirmations: state.pendingConfirmations.filter(
        (req) => req.id !== requestId
      ),
    });
  },

  /**
   * Set WebSocket connection status
   * Requirement 9.1: Connection state management
   */
  setWSConnected: (connected: boolean) => {
    set({ wsConnected: connected });
  },

  /**
   * Set WebSocket error
   * Requirement 9.1: Error state management
   */
  setWSError: (error: string | null) => {
    set({ wsError: error });
  },

  /**
   * Add a message to a specific session
   * Requirement 1.1: Message management
   */
  addMessageToSession: (sessionId: string, message: Message) => {
    const state = get();
    const sessionMessages = state.messages.get(sessionId) || [];
    const updatedMessages = new Map(state.messages);
    updatedMessages.set(sessionId, [...sessionMessages, message]);

    // Update session
    const session = state.sessions.get(sessionId);
    if (session) {
      const updatedSessions = new Map(state.sessions);
      updatedSessions.set(sessionId, {
        ...session,
        messages: [...sessionMessages, message],
        updatedAt: Date.now(),
      });

      set({
        messages: updatedMessages,
        sessions: updatedSessions,
      });
    } else {
      set({ messages: updatedMessages });
    }
  },

  /**
   * Update session timestamp
   * Requirement 8.1: Session update tracking
   */
  updateSessionTimestamp: (sessionId: string) => {
    const state = get();
    const session = state.sessions.get(sessionId);

    if (session) {
      const updatedSessions = new Map(state.sessions);
      updatedSessions.set(sessionId, {
        ...session,
        updatedAt: Date.now(),
      });

      set({ sessions: updatedSessions });
    }
  },

  /**
   * Connect to WebSocket server
   * Requirement 9.1: WebSocket connection management
   */
  connect: () => {
    if (wsManager && wsManager.isConnected()) {
      return;
    }

    const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws';

    wsManager = createWebSocketManager(
      { url: wsUrl },
      {
        onConnect: () => {
          get().setWSConnected(true);
          get().setWSError(null);
        },
        onDisconnect: () => {
          get().setWSConnected(false);
        },
        onMessage: (message: ServerMessage) => {
          const state = get();
          
          switch (message.type) {
            case 'agent_stream':
              if (message.content) {
                state.appendStreamingContent(message.messageId, message.content);
              }
              if (message.done) {
                state.finalizeStreamingMessage(message.messageId);
              }
              break;
            case 'mcp_status':
              state.updateMCPStatus(message.serverId, message.status);
              break;
            case 'confirmation_request':
              state.addConfirmation(message.request);
              break;
            case 'agent_log':
              state.addLog(message.log);
              break;
            case 'error':
              state.setWSError(message.message);
              break;
          }
        },
        onError: (error: Error) => {
          get().setWSError(error.message);
        },
      }
    );

    wsManager.connect();
  },

  /**
   * Disconnect from WebSocket server
   * Requirement 9.1: WebSocket disconnection
   */
  disconnect: () => {
    if (wsManager) {
      wsManager.disconnect();
      wsManager = null;
    }
    set({ wsConnected: false });
  },
}));
