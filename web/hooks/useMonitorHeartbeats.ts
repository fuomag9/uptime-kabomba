"use client";

import { useState, useEffect } from 'react';
import { useWebSocketMessage } from './useWebSocket';
import { Heartbeat } from '@/lib/api';

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

function notifyListeners() {
  listeners.forEach(listener => listener());
}

// Hook to get all monitor heartbeats
export function useMonitorHeartbeats() {
  const [heartbeats, setHeartbeats] = useState<Map<number, HeartbeatState>>(new Map(heartbeatState));

  useEffect(() => {
    const listener = () => {
      setHeartbeats(new Map(heartbeatState));
    };

    listeners.add(listener);

    return () => {
      listeners.delete(listener);
    };
  }, []);

  // Listen for heartbeat messages
  useWebSocketMessage('heartbeat', (payload) => {
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

  return heartbeats;
}

// Hook to get a specific monitor's heartbeat
export function useMonitorHeartbeat(monitorId: number) {
  const heartbeats = useMonitorHeartbeats();
  return heartbeats.get(monitorId);
}
