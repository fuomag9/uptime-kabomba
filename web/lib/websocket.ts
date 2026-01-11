// WebSocket client for real-time updates

function getWebSocketURL(): string {
  if (typeof window === 'undefined') return '';

  // Use the same host as the frontend, but with ws/wss protocol
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  return `${protocol}//${host}/ws`;
}

export interface WebSocketMessage {
  type: string;
  payload: any;
}

export type MessageHandler = (message: WebSocketMessage) => void;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private listeners: Map<string, Set<MessageHandler>> = new Map();
  private reconnectTimer?: NodeJS.Timeout;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private connected = false;

  connect(token?: string) {
    if (typeof window === 'undefined') return; // Don't connect on server side

    const wsUrl = getWebSocketURL();
    const url = token ? `${wsUrl}?token=${token}` : wsUrl;

    try {
      this.ws = new WebSocket(url);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.connected = true;
        this.reconnectAttempts = 0;
        this.emit('connected', {});
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          this.emit(message.type, message.payload);
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this.emit('error', { error });
      };

      this.ws.onclose = () => {
        console.log('WebSocket disconnected');
        this.connected = false;
        this.emit('disconnected', {});
        this.attemptReconnect(token);
      };
    } catch (err) {
      console.error('Failed to create WebSocket:', err);
      this.attemptReconnect(token);
    }
  }

  private attemptReconnect(token?: string) {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

    this.reconnectTimer = setTimeout(() => {
      this.connect(token);
    }, delay);
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.connected = false;
  }

  on(event: string, handler: MessageHandler) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(handler);
  }

  off(event: string, handler: MessageHandler) {
    this.listeners.get(event)?.delete(handler);
  }

  private emit(event: string, payload: any) {
    const handlers = this.listeners.get(event);
    if (handlers) {
      handlers.forEach((handler) => {
        try {
          handler({ type: event, payload });
        } catch (err) {
          console.error('Error in WebSocket handler:', err);
        }
      });
    }
  }

  send(type: string, payload: any) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('WebSocket not connected, cannot send message');
      return;
    }

    const message: WebSocketMessage = { type, payload };
    this.ws.send(JSON.stringify(message));
  }

  isConnected(): boolean {
    return this.connected && this.ws?.readyState === WebSocket.OPEN;
  }

  // Helper methods for common operations
  subscribe(monitorId: number) {
    this.send('subscribe', { monitorId });
  }

  unsubscribe(monitorId: number) {
    this.send('unsubscribe', { monitorId });
  }

  ping() {
    this.send('ping', {});
  }
}

// Global WebSocket client instance
export const wsClient = new WebSocketClient();
