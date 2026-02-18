"use client";

import { useEffect, useState, useRef } from 'react';
import { wsClient, WebSocketMessage } from '@/lib/websocket';
import { apiClient } from '@/lib/api';

export function useWebSocket() {
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const token = apiClient.getToken();

    // Connect to WebSocket if not already connected
    if (!wsClient.isConnected()) {
      wsClient.connect(token || undefined);
    }

    // Setup event listeners
    const handleConnected = () => setConnected(true);
    const handleDisconnected = () => setConnected(false);

    wsClient.on('connected', handleConnected);
    wsClient.on('disconnected', handleDisconnected);

    // Update connected state if already connected
    if (wsClient.isConnected()) {
      setConnected(true);
    }

    // Cleanup on unmount - only remove listeners, keep connection alive
    return () => {
      wsClient.off('connected', handleConnected);
      wsClient.off('disconnected', handleDisconnected);
      // Don't disconnect - WebSocket should stay alive across navigation
    };
  }, []);

  return { connected, client: wsClient };
}

export function useWebSocketMessage(
  messageType: string,
  handler: (payload: any) => void
) {
  // Store handler in ref to keep subscription stable
  const handlerRef = useRef(handler);

  // Keep ref updated with latest handler
  useEffect(() => {
    handlerRef.current = handler;
  });

  useEffect(() => {
    const wrappedHandler = (message: WebSocketMessage) => {
      handlerRef.current(message.payload);
    };

    wsClient.on(messageType, wrappedHandler);

    return () => {
      wsClient.off(messageType, wrappedHandler);
    };
  }, [messageType]);
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
