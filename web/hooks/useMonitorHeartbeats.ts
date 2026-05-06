"use client";

import { useState, useEffect, useSyncExternalStore, useCallback } from 'react';
import { wsClient } from '@/lib/websocket';

interface HeartbeatState {
  monitorId: number;
  status: number;
  ping: number;
  message: string;
  time: string;
}

// Global state for all monitor heartbeats
const heartbeatState = new Map<number, HeartbeatState>();
const listeners = new Set<() => void>();
let wsHandlerRegistered = false;

function notifyListeners() {
  listeners.forEach(listener => listener());
}

function subscribe(listener: () => void) {
  listeners.add(listener);

  // Register the WebSocket handler once globally, not per component
  if (!wsHandlerRegistered) {
    wsHandlerRegistered = true;
    wsClient.on('heartbeat', (message) => {
      const payload = message.payload ?? message;
      const heartbeat: HeartbeatState = {
        monitorId: payload.monitor_id,
        status: payload.status,
        ping: payload.ping,
        message: payload.message,
        time: payload.time,
      };
      heartbeatState.set(heartbeat.monitorId, heartbeat);
      notifyListeners();
    });
  }

  return () => {
    listeners.delete(listener);
  };
}

function getSnapshot() {
  return heartbeatState;
}

// Hook to get a specific monitor's heartbeat (lightweight — no Map copy)
export function useMonitorHeartbeat(monitorId: number) {
  const [heartbeat, setHeartbeat] = useState<HeartbeatState | undefined>(
    () => heartbeatState.get(monitorId)
  );

  useEffect(() => {
    const listener = () => {
      const current = heartbeatState.get(monitorId);
      setHeartbeat(prev => {
        // Only re-render if this monitor's heartbeat actually changed
        if (prev === current) return prev;
        if (prev && current && prev.time === current.time && prev.status === current.status && prev.ping === current.ping) return prev;
        return current;
      });
    };

    return subscribe(listener);
  }, [monitorId]);

  return heartbeat;
}

// Hook to get all monitor heartbeats (only used if needed)
export function useMonitorHeartbeats() {
  const [snapshot, setSnapshot] = useState(() => new Map(heartbeatState));

  useEffect(() => {
    const listener = () => {
      setSnapshot(new Map(heartbeatState));
    };

    return subscribe(listener);
  }, []);

  return snapshot;
}
