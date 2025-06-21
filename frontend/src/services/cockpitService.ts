import { EventEmitter } from 'events';

// Event types matching backend
export enum EventType {
  TRACE = 'trace',
  SPAN = 'span',
  THREAT = 'threat',
  MODERATION = 'moderation',
  PROVIDER = 'provider',
  METRIC = 'metric',
  ALERT = 'alert'
}

// Event structure
export interface SafetyEvent {
  id: string;
  type: EventType;
  timestamp: Date;
  trace_id?: string;
  span_id?: string;
  data: Record<string, any>;
  severity?: string;
  tags?: string[];
  timestamp_unix?: number;
}

// WebSocket message types
export interface EventWebSocketMessage {
  type: 'event' | 'historical' | 'aggregated' | 'metrics';
  event?: SafetyEvent;
  events?: SafetyEvent[];
  command?: string;
  filter?: EventFilter;
  error?: string;
}

// Event filter
export interface EventFilter {
  types?: EventType[];
  severities?: string[];
  tags?: string[];
  trace_ids?: string[];
}

// Aggregated data structures
export interface AggregatedData {
  timestamp: Date;
  window: string;
  event_counts: Record<EventType, number>;
  threat_counts: Record<string, number>;
  provider_stats: Record<string, ProviderStats>;
  metrics: Record<string, MetricSummary>;
}

export interface ProviderStats {
  provider: string;
  request_count: number;
  success_count: number;
  error_count: number;
  avg_response_time: number;
  p95_response_time: number;
  p99_response_time: number;
}

export interface MetricSummary {
  name: string;
  count: number;
  average: number;
  min: number;
  max: number;
  current: number;
}

// Connection states
export enum ConnectionState {
  DISCONNECTED = 'disconnected',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  ERROR = 'error'
}

// CockpitService configuration
export interface CockpitServiceConfig {
  url?: string;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  heartbeatInterval?: number;
  bufferSize?: number;
}

// Main CockpitService class
export class CockpitService extends EventEmitter {
  private ws: WebSocket | null = null;
  private config: Required<CockpitServiceConfig>;
  private state: ConnectionState = ConnectionState.DISCONNECTED;
  private reconnectAttempts = 0;
  private reconnectTimer?: NodeJS.Timeout;
  private heartbeatTimer?: NodeJS.Timeout;
  private eventBuffer: SafetyEvent[] = [];
  private filter: EventFilter = {};

  constructor(config: CockpitServiceConfig = {}) {
    super();
    this.config = {
      url: config.url || this.getDefaultWebSocketUrl(),
      reconnectInterval: config.reconnectInterval || 5000,
      maxReconnectAttempts: config.maxReconnectAttempts || 10,
      heartbeatInterval: config.heartbeatInterval || 30000,
      bufferSize: config.bufferSize || 1000
    };
  }

  // Get default WebSocket URL based on current location
  private getDefaultWebSocketUrl(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${protocol}//${host}/api/safety-cockpit/stream`;
  }

  // Connect to WebSocket
  public connect(): void {
    if (this.state === ConnectionState.CONNECTING || this.state === ConnectionState.CONNECTED) {
      return;
    }

    this.setState(ConnectionState.CONNECTING);
    this.reconnectAttempts++;

    try {
      this.ws = new WebSocket(this.config.url);
      this.setupWebSocketHandlers();
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      this.handleConnectionError();
    }
  }

  // Disconnect from WebSocket
  public disconnect(): void {
    this.clearTimers();
    this.reconnectAttempts = 0;

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.setState(ConnectionState.DISCONNECTED);
  }

  // Set event filter
  public setFilter(filter: EventFilter): void {
    this.filter = filter;
    if (this.isConnected()) {
      this.sendMessage({
        command: 'filter',
        filter
      });
    }
  }

  // Request aggregated data
  public requestAggregatedData(): void {
    if (this.isConnected()) {
      this.sendMessage({
        command: 'getAggregated'
      });
    }
  }

  // Request metrics
  public requestMetrics(): void {
    if (this.isConnected()) {
      this.sendMessage({
        command: 'getMetrics'
      });
    }
  }

  // Get buffered events
  public getBufferedEvents(): SafetyEvent[] {
    return [...this.eventBuffer];
  }

  // Get connection state
  public getState(): ConnectionState {
    return this.state;
  }

  // Check if connected
  public isConnected(): boolean {
    return this.state === ConnectionState.CONNECTED && this.ws?.readyState === WebSocket.OPEN;
  }

  // Setup WebSocket event handlers
  private setupWebSocketHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      console.log('Safety Cockpit WebSocket connected');
      this.setState(ConnectionState.CONNECTED);
      this.reconnectAttempts = 0;
      this.startHeartbeat();

      // Apply filter if set
      if (Object.keys(this.filter).length > 0) {
        this.setFilter(this.filter);
      }
    };

    this.ws.onmessage = (event) => {
      try {
        const message: EventWebSocketMessage = JSON.parse(event.data);
        this.handleMessage(message);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    this.ws.onclose = (event) => {
      console.log('Safety Cockpit WebSocket closed:', event.code, event.reason);
      this.handleConnectionClose();
    };

    this.ws.onerror = (error) => {
      console.error('Safety Cockpit WebSocket error:', error);
      this.setState(ConnectionState.ERROR);
    };
  }

  // Handle incoming messages
  private handleMessage(message: EventWebSocketMessage): void {
    switch (message.type) {
      case 'event':
        if (message.event) {
          this.handleEvent(message.event);
        }
        break;

      case 'historical':
        if (message.events) {
          this.handleHistoricalEvents(message.events);
        }
        break;

      case 'aggregated':
        if (message.event?.data) {
          this.emit('aggregated', message.event.data);
        }
        break;

      case 'metrics':
        if (message.event?.data) {
          this.emit('metrics', message.event.data);
        }
        break;

      default:
        if (message.error) {
          this.emit('error', new Error(message.error));
        }
    }
  }

  // Handle single event
  private handleEvent(event: SafetyEvent): void {
    // Convert timestamp if needed
    if (event.timestamp_unix) {
      event.timestamp = new Date(event.timestamp_unix * 1000);
    } else if (typeof event.timestamp === 'string') {
      event.timestamp = new Date(event.timestamp);
    }

    // Add to buffer
    this.addToBuffer(event);

    // Emit typed event
    this.emit(event.type, event);

    // Emit general event
    this.emit('event', event);

    // Emit specific events based on severity or tags
    if (event.severity) {
      this.emit(`severity:${event.severity}`, event);
    }

    if (event.tags) {
      event.tags.forEach(tag => {
        this.emit(`tag:${tag}`, event);
      });
    }
  }

  // Handle historical events
  private handleHistoricalEvents(events: SafetyEvent[]): void {
    // Process all events
    events.forEach(event => {
      // Convert timestamps
      if (event.timestamp_unix) {
        event.timestamp = new Date(event.timestamp_unix * 1000);
      } else if (typeof event.timestamp === 'string') {
        event.timestamp = new Date(event.timestamp);
      }
    });

    // Replace buffer with historical events
    this.eventBuffer = events.slice(-this.config.bufferSize);

    // Emit historical event
    this.emit('historical', events);
  }

  // Add event to buffer
  private addToBuffer(event: SafetyEvent): void {
    this.eventBuffer.push(event);
    
    // Maintain buffer size
    if (this.eventBuffer.length > this.config.bufferSize) {
      this.eventBuffer.shift();
    }
  }

  // Send message to server
  private sendMessage(message: Partial<EventWebSocketMessage>): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  // Handle connection close
  private handleConnectionClose(): void {
    this.clearTimers();
    this.setState(ConnectionState.DISCONNECTED);

    // Attempt reconnection if not at max attempts
    if (this.reconnectAttempts < this.config.maxReconnectAttempts) {
      this.scheduleReconnect();
    } else {
      this.emit('maxReconnectAttemptsReached');
    }
  }

  // Handle connection error
  private handleConnectionError(): void {
    this.setState(ConnectionState.ERROR);
    this.scheduleReconnect();
  }

  // Schedule reconnection
  private scheduleReconnect(): void {
    if (this.reconnectTimer) return;

    const delay = Math.min(
      this.config.reconnectInterval * Math.pow(1.5, this.reconnectAttempts - 1),
      30000
    );

    console.log(`Scheduling reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = undefined;
      this.connect();
    }, delay);
  }

  // Start heartbeat
  private startHeartbeat(): void {
    this.clearHeartbeat();

    this.heartbeatTimer = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        // Server sends pings, we just need to handle connection state
        if (this.ws.readyState !== WebSocket.OPEN) {
          this.handleConnectionClose();
        }
      }
    }, this.config.heartbeatInterval);
  }

  // Clear timers
  private clearTimers(): void {
    this.clearHeartbeat();

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = undefined;
    }
  }

  // Clear heartbeat
  private clearHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = undefined;
    }
  }

  // Set connection state
  private setState(state: ConnectionState): void {
    if (this.state !== state) {
      const previousState = this.state;
      this.state = state;
      this.emit('stateChange', { previousState, currentState: state });
    }
  }
}

// Singleton instance
let cockpitServiceInstance: CockpitService | null = null;

// Get or create cockpit service instance
export function getCockpitService(config?: CockpitServiceConfig): CockpitService {
  if (!cockpitServiceInstance) {
    cockpitServiceInstance = new CockpitService(config);
  }
  return cockpitServiceInstance;
}

// Export default instance
export default getCockpitService();