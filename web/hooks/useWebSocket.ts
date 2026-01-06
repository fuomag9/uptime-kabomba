"use client";

import { useEffect, useState } from 'react';
import { wsClient, WebSocketMessage } from '@/lib/websocket';
import { apiClient } from '@/lib/api';

export function useWebSocket() {
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const token = apiClient.getToken();

    // Connect to WebSocket
    wsClient.connect(token || undefined);

    // Setup event listeners
    const handleConnected = () => setConnected(true);
    const handleDisconnected = () => setConnected(false);

    wsClient.on('connected', handleConnected);
    wsClient.on('disconnected', handleDisconnected);

    // Cleanup on unmount
    return () => {
      wsClient.off('connected', handleConnected);
      wsClient.off('disconnected', handleDisconnected);
      wsClient.disconnect();
    };
  }, []);

  return { connected, client: wsClient };
}

export function useWebSocketMessage(
  messageType: string,
  handler: (payload: any) => void
) {
  useEffect(() => {
    const wrappedHandler = (message: WebSocketMessage) => {
      handler(message.payload);
    };

    wsClient.on(messageType, wrappedHandler);

    return () => {
      wsClient.off(messageType, wrappedHandler);
    };
  }, [messageType, handler]);
}

export function useHeartbeat(monitorId: number) {
  const [heartbeat, setHeartbeat] = useState<any>(null);

  useWebSocketMessage('heartbeat', (payload) => {
    if (payload.monitorId === monitorId) {
      setHeartbeat(payload);
    }
  });

  useEffect(() => {
    wsClient.subscribe(monitorId);

    return () => {
      wsClient.unsubscribe(monitorId);
    };
  }, [monitorId]);

  return heartbeat;
}
