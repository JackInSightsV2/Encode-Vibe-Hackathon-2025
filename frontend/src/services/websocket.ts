// WebSocket service for real-time communication with backend
export interface WSMessage {
  type: string;
  timestamp: string;
  data?: Record<string, any>;
  error?: string;
  id?: string;
}

// Message types constants
export const MessageTypes = {
  HEALTH_UPDATE: 'health_update',
  LOG_ENTRY: 'log_entry',
  CONFIG_CHANGE: 'config_change',
  KILL_SWITCH_UPDATE: 'kill_switch_update',
  PROVIDER_STATUS: 'provider_status',
  CLIENT_MESSAGE: 'client_message',
  ERROR: 'error',
  ACK: 'ack',
} as const;

export type MessageType = typeof MessageTypes[keyof typeof MessageTypes];

export enum ConnectionStatus {
  DISCONNECTED = 'disconnected',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  RECONNECTING = 'reconnecting',
  ERROR = 'error',
}

export interface WebSocketEventHandlers {
  onMessage?: (message: WSMessage) => void;
  onStatusChange?: (status: ConnectionStatus) => void;
  onError?: (error: Error) => void;
  onHealthUpdate?: (data: any) => void;
  onLogEntry?: (data: any) => void;
  onConfigChange?: (data: any) => void;
}

export class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectTimeout: number | null = null;
  private status: ConnectionStatus = ConnectionStatus.DISCONNECTED;
  private handlers: WebSocketEventHandlers = {};
  private url: string;
  private messageQueue: WSMessage[] = [];

  constructor(url?: string) {
    // Default to current host with ws protocol
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    this.url = url || `${protocol}//${host}/ws`;
  }

  // Register event handlers
  setHandlers(handlers: WebSocketEventHandlers): void {
    this.handlers = { ...this.handlers, ...handlers };
  }

  // Get current connection status
  getStatus(): ConnectionStatus {
    return this.status;
  }

  // Connect to WebSocket server
  async connect(): Promise<void> {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.setStatus(ConnectionStatus.CONNECTING);

    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
          console.log('WebSocket connected');
          this.setStatus(ConnectionStatus.CONNECTED);
          this.reconnectAttempts = 0;
          this.flushMessageQueue();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message: WSMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
            this.handlers.onError?.(new Error('Invalid message format'));
          }
        };

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected:', event.code, event.reason);
          this.setStatus(ConnectionStatus.DISCONNECTED);
          
          if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect();
          }
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          this.setStatus(ConnectionStatus.ERROR);
          this.handlers.onError?.(new Error('WebSocket connection error'));
          reject(new Error('WebSocket connection failed'));
        };

      } catch (error) {
        this.setStatus(ConnectionStatus.ERROR);
        reject(error);
      }
    });
  }

  // Disconnect from WebSocket server
  disconnect(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }

    this.setStatus(ConnectionStatus.DISCONNECTED);
  }

  // Send message to server
  sendMessage(message: Omit<WSMessage, 'timestamp'>): void {
    const fullMessage: WSMessage = {
      ...message,
      timestamp: new Date().toISOString(),
    };

    if (this.ws?.readyState === WebSocket.OPEN) {
      try {
        this.ws.send(JSON.stringify(fullMessage));
      } catch (error) {
        console.error('Failed to send WebSocket message:', error);
        this.handlers.onError?.(new Error('Failed to send message'));
      }
    } else {
      // Queue message for when connection is restored
      this.messageQueue.push(fullMessage);
      
      // Try to reconnect if not already trying
      if (this.status === ConnectionStatus.DISCONNECTED) {
        this.connect().catch(console.error);
      }
    }
  }

  // Send client message with automatic ID generation
  sendClientMessage(data: Record<string, any>, id?: string): void {
    this.sendMessage({
      type: MessageTypes.CLIENT_MESSAGE,
      data,
      id: id || this.generateMessageId(),
    });
  }

  // Private methods
  private setStatus(status: ConnectionStatus): void {
    if (this.status !== status) {
      this.status = status;
      this.handlers.onStatusChange?.(status);
    }
  }

  private handleMessage(message: WSMessage): void {
    // Call general message handler
    this.handlers.onMessage?.(message);

    // Route to specific handlers based on message type
    switch (message.type) {
      case MessageTypes.HEALTH_UPDATE:
        this.handlers.onHealthUpdate?.(message.data);
        break;
      case MessageTypes.LOG_ENTRY:
        this.handlers.onLogEntry?.(message.data);
        break;
      case MessageTypes.CONFIG_CHANGE:
        this.handlers.onConfigChange?.(message.data);
        break;
      case MessageTypes.ERROR:
        console.error('WebSocket error message:', message.error);
        this.handlers.onError?.(new Error(message.error || 'Unknown server error'));
        break;
      case MessageTypes.ACK:
        console.log('Message acknowledged:', message.id);
        break;
      default:
        console.log('Unhandled message type:', message.type, message);
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimeout) {
      return;
    }

    this.reconnectAttempts++;
    this.setStatus(ConnectionStatus.RECONNECTING);

    // Exponential backoff: 1s, 2s, 4s, 8s, 16s
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts - 1), 16000);
    
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

    this.reconnectTimeout = setTimeout(() => {
      this.reconnectTimeout = null;
      this.connect().catch((error) => {
        console.error('Reconnection failed:', error);
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
          this.setStatus(ConnectionStatus.ERROR);
          this.handlers.onError?.(new Error('Maximum reconnection attempts reached'));
        }
      });
    }, delay);
  }

  private flushMessageQueue(): void {
    if (this.messageQueue.length > 0) {
      console.log(`Sending ${this.messageQueue.length} queued messages`);
      const queue = [...this.messageQueue];
      this.messageQueue = [];
      
      queue.forEach(message => {
        if (this.ws?.readyState === WebSocket.OPEN) {
          this.ws.send(JSON.stringify(message));
        }
      });
    }
  }

  private generateMessageId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Singleton instance
export const webSocketService = new WebSocketService();