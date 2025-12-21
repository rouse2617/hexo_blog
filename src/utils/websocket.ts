/**
 * WebSocket Manager for OpsGenius Frontend
 * Requirements: 9.1, 9.5
 * 
 * Handles WebSocket connection, reconnection with exponential backoff,
 * and message sending/receiving logic.
 */

import type {
  ClientMessage,
  ServerMessage,
  WebSocketHandlers,
  WebSocketState,
} from '../types/websocket';

/**
 * WebSocket Manager configuration
 */
export interface WebSocketConfig {
  url: string;
  reconnectMaxDelay?: number;
  heartbeatInterval?: number;
  maxReconnectAttempts?: number;
}

/**
 * WebSocket Manager class
 * Requirement 9.1: WebSocket connection management
 * Requirement 9.5: Exponential backoff reconnection strategy
 */
export class WebSocketManager {
  private ws: WebSocket | null = null;
  private config: Required<WebSocketConfig>;
  private handlers: WebSocketHandlers;
  private reconnectAttempts = 0;
  private reconnectTimer: number | null = null;
  private heartbeatTimer: number | null = null;
  private state: WebSocketState = 'disconnected';
  private messageQueue: ClientMessage[] = [];

  constructor(config: WebSocketConfig, handlers: WebSocketHandlers) {
    this.config = {
      url: config.url,
      reconnectMaxDelay: config.reconnectMaxDelay ?? 30000,
      heartbeatInterval: config.heartbeatInterval ?? 30000,
      maxReconnectAttempts: config.maxReconnectAttempts ?? Infinity,
    };
    this.handlers = handlers;
  }

  /**
   * Get current connection state
   */
  public getState(): WebSocketState {
    return this.state;
  }

  /**
   * Check if WebSocket is connected
   */
  public isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  /**
   * Connect to WebSocket server
   * Requirement 9.1: Connection establishment
   */
  public connect(): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      console.warn('WebSocket already connected');
      return;
    }

    this.setState('connecting');

    try {
      this.ws = new WebSocket(this.config.url);
      this.setupEventHandlers();
    } catch (error) {
      this.handleError(error as Error);
      this.scheduleReconnect();
    }
  }

  /**
   * Disconnect from WebSocket server
   */
  public disconnect(): void {
    this.clearTimers();
    
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.setState('disconnected');
    this.reconnectAttempts = 0;
    this.messageQueue = [];
  }

  /**
   * Send message to server
   * Requirement 9.1: Message sending
   */
  public send(message: ClientMessage): void {
    if (!this.isConnected()) {
      // Queue message for later sending
      this.messageQueue.push(message);
      console.warn('WebSocket not connected, message queued');
      return;
    }

    try {
      this.ws!.send(JSON.stringify(message));
    } catch (error) {
      console.error('Failed to send message:', error);
      this.messageQueue.push(message);
      this.handleError(error as Error);
    }
  }

  /**
   * Setup WebSocket event handlers
   */
  private setupEventHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      this.handleOpen();
    };

    this.ws.onmessage = (event: MessageEvent) => {
      this.handleMessage(event);
    };

    this.ws.onclose = (event: CloseEvent) => {
      this.handleClose(event);
    };

    this.ws.onerror = () => {
      this.handleError(new Error('WebSocket error occurred'));
    };
  }

  /**
   * Handle WebSocket open event
   * Requirement 9.1: Connection success handling
   */
  private handleOpen(): void {
    console.log('WebSocket connected');
    this.setState('connected');
    this.reconnectAttempts = 0;

    // Start heartbeat
    this.startHeartbeat();

    // Send queued messages
    this.flushMessageQueue();

    // Notify handlers
    this.handlers.onConnect();
  }

  /**
   * Handle incoming WebSocket message
   * Requirement 9.1: Message receiving
   */
  private handleMessage(event: MessageEvent): void {
    try {
      const message: ServerMessage = JSON.parse(event.data);
      this.handlers.onMessage(message);
    } catch (error) {
      console.error('Failed to parse message:', error);
      this.handleError(new Error('Invalid message format'));
    }
  }

  /**
   * Handle WebSocket close event
   * Requirement 9.1: Connection close handling
   */
  private handleClose(event: CloseEvent): void {
    console.log('WebSocket closed:', event.code, event.reason);
    this.clearTimers();
    this.setState('disconnected');
    this.handlers.onDisconnect();

    // Attempt reconnection if not a normal closure
    if (event.code !== 1000) {
      this.scheduleReconnect();
    }
  }

  /**
   * Handle WebSocket error
   * Requirement 9.1: Error handling
   */
  private handleError(error: Error): void {
    console.error('WebSocket error:', error);
    this.setState('error');
    this.handlers.onError(error);
  }

  /**
   * Schedule reconnection with exponential backoff
   * Requirement 9.5: Exponential backoff reconnection strategy
   */
  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      this.handleError(new Error('Max reconnection attempts reached'));
      return;
    }

    // Clear existing timer
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
    }

    // Calculate delay with exponential backoff: 1s, 2s, 4s, 8s, ..., max 30s
    const delay = Math.min(
      1000 * Math.pow(2, this.reconnectAttempts),
      this.config.reconnectMaxDelay
    );

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts + 1})`);

    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectAttempts++;
      this.connect();
    }, delay);
  }

  /**
   * Start heartbeat timer
   * Requirement 9.1: Connection health monitoring
   */
  private startHeartbeat(): void {
    this.clearHeartbeat();

    this.heartbeatTimer = window.setInterval(() => {
      if (this.isConnected()) {
        this.send({
          type: 'heartbeat',
          timestamp: Date.now(),
        });
      }
    }, this.config.heartbeatInterval);
  }

  /**
   * Clear heartbeat timer
   */
  private clearHeartbeat(): void {
    if (this.heartbeatTimer !== null) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  /**
   * Clear all timers
   */
  private clearTimers(): void {
    this.clearHeartbeat();

    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  /**
   * Flush queued messages
   */
  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0 && this.isConnected()) {
      const message = this.messageQueue.shift();
      if (message) {
        this.send(message);
      }
    }
  }

  /**
   * Set connection state
   */
  private setState(state: WebSocketState): void {
    this.state = state;
  }
}

/**
 * Create a WebSocket manager instance
 * Requirement 9.1: WebSocket manager factory
 */
export function createWebSocketManager(
  config: WebSocketConfig,
  handlers: WebSocketHandlers
): WebSocketManager {
  return new WebSocketManager(config, handlers);
}
