/**
 * Core data models for OpsGenius Frontend
 * Requirements: 1.1, 2.1, 3.1, 5.1
 */

/**
 * Message in the chat interface
 * Requirement 1.1: Chat interface message model
 */
export interface Message {
  id: string;
  role: 'user' | 'agent';
  content: string;
  timestamp: number;
  streaming?: boolean;
  charts?: ChartData[];
}

/**
 * Chart data for visualization
 * Requirement 6.1, 6.2, 6.3: Chart rendering support
 */
export interface ChartData {
  type: 'line' | 'bar' | 'pie';
  title?: string;
  data: ChartDataPoint[];
  xAxisLabel?: string;
  yAxisLabel?: string;
}

export interface ChartDataPoint {
  x: string | number;
  y: number;
  label?: string;
}

/**
 * MCP Server model
 * Requirement 2.1: MCP Server status monitoring
 */
export interface MCPServer {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'error';
  tools: MCPTool[];
  lastHeartbeat?: number;
}

/**
 * MCP Tool definition
 * Requirement 2.3: MCP Server tool list display
 */
export interface MCPTool {
  name: string;
  description: string;
  readonly: boolean;
}

/**
 * Log entry in the log viewer
 * Requirement 3.1: Agent execution log viewing
 */
export interface LogEntry {
  id: string;
  timestamp: number;
  level: 'info' | 'warning' | 'error';
  message: string;
  agentRefs?: AgentReference[];
  metadata?: Record<string, unknown>;
}

/**
 * Agent reference in log entries
 * Requirement 3.2: Agent reference rendering
 */
export interface AgentReference {
  id: string;
  displayName: string;
  type: string;
}

/**
 * Confirmation request for high-risk operations
 * Requirement 5.1: Human confirmation mechanism
 */
export interface ConfirmationRequest {
  id: string;
  operation: string;
  description: string;
  riskLevel: 'low' | 'medium' | 'high';
  timeout: number;
}

/**
 * Context log for agent details
 * Requirement 4.1: Context log detail viewing
 */
export interface ContextLog {
  id: string;
  timestamp: number;
  message: string;
  level: string;
}

/**
 * Session model for chat history
 * Requirement 8.1: Session history management
 */
export interface Session {
  id: string;
  title: string;
  createdAt: number;
  updatedAt: number;
  messages: Message[];
}

/**
 * Stored session in local storage
 * Requirement 8.1, 8.2: Session persistence
 */
export interface StoredSession {
  id: string;
  title: string;
  createdAt: number;
  updatedAt: number;
  messages: Message[];
}

/**
 * MCP Server configuration
 * Requirement 2.1: MCP Server configuration
 */
export interface MCPServerConfig {
  id: string;
  name: string;
  endpoint: string;
  enabled: boolean;
}

/**
 * Local storage schema
 * Requirement 8.1: Session data persistence
 */
export interface LocalStorageSchema {
  sessions: StoredSession[];
  currentSessionId: string;
  mcpConfig: MCPServerConfig[];
}
