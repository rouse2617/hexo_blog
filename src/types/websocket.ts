/**
 * WebSocket message protocol types for OpsGenius Frontend
 * Requirements: 1.1, 2.1, 3.1, 5.1, 9.1
 */

import type { LogEntry, ConfirmationRequest, ChartData } from './models';

/**
 * Client to Server Messages
 */

/**
 * User message payload
 * Requirement 1.1: User message submission
 */
export interface UserMessagePayload {
  type: 'user_message';
  sessionId: string;
  content: string;
  timestamp: number;
}

/**
 * Confirmation response payload
 * Requirement 5.1: Confirmation response
 */
export interface ConfirmationResponsePayload {
  type: 'confirmation_response';
  requestId: string;
  action: 'confirm' | 'cancel';
  timestamp: number;
}

/**
 * Heartbeat payload
 * Requirement 9.1: Connection health monitoring
 */
export interface HeartbeatPayload {
  type: 'heartbeat';
  timestamp: number;
}

/**
 * Union type for all client messages
 */
export type ClientMessage = 
  | UserMessagePayload 
  | ConfirmationResponsePayload 
  | HeartbeatPayload;

/**
 * Server to Client Messages
 */

/**
 * Agent stream payload for streaming responses
 * Requirement 1.2: Streaming response display
 */
export interface AgentStreamPayload {
  type: 'agent_stream';
  sessionId: string;
  messageId: string;
  content: string;
  done: boolean;
  timestamp: number;
}

/**
 * Agent log payload
 * Requirement 3.1: Agent log real-time display
 */
export interface AgentLogPayload {
  type: 'agent_log';
  log: LogEntry;
}

/**
 * MCP status update payload
 * Requirement 2.2: MCP Server status updates
 */
export interface MCPStatusPayload {
  type: 'mcp_status';
  serverId: string;
  status: 'online' | 'offline' | 'error';
  timestamp: number;
}

/**
 * Confirmation request payload
 * Requirement 5.1: Confirmation request display
 */
export interface ConfirmationRequestPayload {
  type: 'confirmation_request';
  request: ConfirmationRequest;
}

/**
 * Chart data payload
 * Requirement 6.1: Chart data visualization
 */
export interface ChartDataPayload {
  type: 'chart_data';
  sessionId: string;
  messageId: string;
  chartType: 'line' | 'bar' | 'pie';
  data: ChartData;
}

/**
 * Error payload
 * Requirement 9.1: Error handling
 */
export interface ErrorPayload {
  type: 'error';
  code: string;
  message: string;
  timestamp: number;
}

/**
 * Union type for all server messages
 */
export type ServerMessage = 
  | AgentStreamPayload 
  | AgentLogPayload 
  | MCPStatusPayload 
  | ConfirmationRequestPayload
  | ChartDataPayload
  | ErrorPayload;

/**
 * WebSocket connection state
 * Requirement 9.1: Connection state management
 */
export type WebSocketState = 'connecting' | 'connected' | 'disconnected' | 'error';

/**
 * WebSocket event handlers
 */
export interface WebSocketHandlers {
  onMessage: (message: ServerMessage) => void;
  onConnect: () => void;
  onDisconnect: () => void;
  onError: (error: Error) => void;
}
