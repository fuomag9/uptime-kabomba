"use client";

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient, Heartbeat } from '@/lib/api';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useState, useEffect } from 'react';
import { useMonitorHeartbeat } from '@/hooks/useMonitorHeartbeats';
import { useWebSocket } from '@/hooks/useWebSocket';

// Helper function to format relative time
function formatRelativeTime(date: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 60) return `${diffSec}s ago`;
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHour < 24) return `${diffHour}h ago`;
  return `${diffDay}d ago`;
}

const STATUS_COLORS = {
  0: { bg: 'bg-red-100 dark:bg-red-900/20', text: 'text-red-800 dark:text-red-200', label: 'Down', dot: 'bg-red-500' },
  1: { bg: 'bg-green-100 dark:bg-green-900/20', text: 'text-green-800 dark:text-green-200', label: 'Up', dot: 'bg-green-500' },
  2: { bg: 'bg-yellow-100 dark:bg-yellow-900/20', text: 'text-yellow-800 dark:text-yellow-200', label: 'Pending', dot: 'bg-yellow-500' },
  3: { bg: 'bg-blue-100 dark:bg-blue-900/20', text: 'text-blue-800 dark:text-blue-200', label: 'Maintenance', dot: 'bg-blue-500' },
};

export default function MonitorDetailPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const monitorId = parseInt(params.id as string);

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [hoveredHeartbeat, setHoveredHeartbeat] = useState<{heartbeat: Heartbeat, x: number, y: number} | null>(null);

  // Connect to WebSocket
  const { connected } = useWebSocket();

  // Get real-time heartbeat
  const realtimeHeartbeat = useMonitorHeartbeat(monitorId);

  const { data: monitor, isLoading: isLoadingMonitor } = useQuery({
    queryKey: ['monitor', monitorId],
    queryFn: () => apiClient.getMonitor(monitorId),
    refetchInterval: 10000, // Refetch every 10 seconds
  });

  const { data: heartbeats, isLoading: isLoadingHeartbeats } = useQuery({
    queryKey: ['heartbeats', monitorId],
    queryFn: () => apiClient.getHeartbeats(monitorId, 100),
    refetchInterval: 5000, // Refetch every 5 seconds
  });

  const deleteMutation = useMutation({
    mutationFn: () => apiClient.deleteMonitor(monitorId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['monitors'] });
      router.push('/monitors');
    },
  });

  const toggleActiveMutation = useMutation({
    mutationFn: () => apiClient.updateMonitor(monitorId, {
      name: monitor!.name,
      type: monitor!.type,
      url: monitor!.url,
      interval: monitor!.interval,
      timeout: monitor!.timeout,
      active: !monitor!.active,
      config: monitor!.config,
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['monitor', monitorId] });
      queryClient.invalidateQueries({ queryKey: ['monitors'] });
    },
  });

  if (isLoadingMonitor) {
    return (
      <div className="text-center py-12">
        <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-primary border-r-transparent"></div>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">Loading monitor...</p>
      </div>
    );
  }

  if (!monitor) {
    return (
      <div className="text-center py-12">
        <p className="text-sm text-gray-600 dark:text-gray-400">Monitor not found</p>
        <Link href="/monitors" className="mt-4 inline-block text-primary hover:underline">
          Back to monitors
        </Link>
      </div>
    );
  }

  // Use real-time heartbeat if available, otherwise use latest from history
  const latestHeartbeat = realtimeHeartbeat
    ? {
        id: 0,
        monitor_id: monitorId,
        status: realtimeHeartbeat.status,
        ping: realtimeHeartbeat.ping,
        important: false,
        message: realtimeHeartbeat.message,
        time: realtimeHeartbeat.time,
      }
    : (heartbeats && heartbeats.length > 0 ? heartbeats[0] : null);

  const statusStyle = latestHeartbeat
    ? STATUS_COLORS[latestHeartbeat.status as keyof typeof STATUS_COLORS]
    : STATUS_COLORS[2];

  // Calculate uptime
  let uptime = 0;
  if (heartbeats && heartbeats.length > 0) {
    const upCount = heartbeats.filter(h => h.status === 1).length;
    uptime = (upCount / heartbeats.length) * 100;
  }

  // Calculate average ping
  let avgPing = 0;
  if (heartbeats && heartbeats.length > 0) {
    const validPings = heartbeats.filter(h => h.ping > 0);
    if (validPings.length > 0) {
      const totalPing = validPings.reduce((sum, h) => sum + h.ping, 0);
      avgPing = totalPing / validPings.length;
    }
  }

  // Calculate dynamic scale for heartbeat visualization
  let minPing = 0;
  let maxPing = 100;
  if (heartbeats && heartbeats.length > 0) {
    const validPings = heartbeats.filter(h => h.ping > 0).map(h => h.ping);
    if (validPings.length > 0) {
      minPing = Math.min(...validPings);
      maxPing = Math.max(...validPings);
      // Add 10% padding to max for better visualization
      const padding = (maxPing - minPing) * 0.1;
      maxPing = maxPing + padding;
      minPing = Math.max(0, minPing - padding);
    }
  }

  return (
    <div>
      {/* Header */}
      <div className="sm:flex sm:items-center sm:justify-between">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <Link
              href="/monitors"
              className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </Link>
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-white truncate">
              {monitor.name}
            </h1>
            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusStyle.bg} ${statusStyle.text}`}>
              {statusStyle.label}
            </span>
            {!monitor.active && (
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200">
                Paused
              </span>
            )}
          </div>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {monitor.type.toUpperCase()} • {monitor.url}
          </p>
        </div>
        <div className="mt-4 flex gap-2 sm:ml-16 sm:mt-0">
          <button
            onClick={() => toggleActiveMutation.mutate()}
            disabled={toggleActiveMutation.isPending}
            className="inline-flex items-center rounded-md bg-white dark:bg-gray-700 px-3 py-2 text-sm font-semibold text-gray-900 dark:text-white shadow-sm ring-1 ring-inset ring-gray-300 dark:ring-gray-600 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
          >
            {monitor.active ? 'Pause' : 'Resume'}
          </button>
          <Link
            href={`/monitors/${monitorId}/edit`}
            className="inline-flex items-center rounded-md bg-white dark:bg-gray-700 px-3 py-2 text-sm font-semibold text-gray-900 dark:text-white shadow-sm ring-1 ring-inset ring-gray-300 dark:ring-gray-600 hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            Edit
          </Link>
          <button
            onClick={() => setShowDeleteConfirm(true)}
            className="inline-flex items-center rounded-md bg-red-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-red-500"
          >
            Delete
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-3">
        <div className="overflow-hidden rounded-lg bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 px-4 py-5 shadow-lg dark:shadow-2xl sm:p-6 transition-all hover:shadow-xl dark:hover:shadow-2xl hover:scale-105">
          <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Uptime</dt>
          <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">
            {uptime.toFixed(2)}%
          </dd>
        </div>
        <div className="overflow-hidden rounded-lg bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 px-4 py-5 shadow-lg dark:shadow-2xl sm:p-6 transition-all hover:shadow-xl dark:hover:shadow-2xl hover:scale-105">
          <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Average Ping</dt>
          <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">
            {avgPing > 0 ? `${avgPing.toFixed(0)}ms` : '--'}
          </dd>
        </div>
        <div className="overflow-hidden rounded-lg bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 px-4 py-5 shadow-lg dark:shadow-2xl sm:p-6 transition-all hover:shadow-xl dark:hover:shadow-2xl hover:scale-105">
          <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Check Interval</dt>
          <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">
            {monitor.interval}s
          </dd>
        </div>
      </div>

      {/* Heartbeat History */}
      <div className="mt-8">
        <h2 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
          Heartbeat History
          <span className="ml-2 text-sm font-normal text-gray-500 dark:text-gray-400">
            Last 100 checks
          </span>
        </h2>

        {/* Heartbeat Bar */}
        {heartbeats && heartbeats.length > 0 && (
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg shadow-lg dark:shadow-2xl p-6 mb-6 relative">
            <div className="flex gap-4">
              {/* Y-axis scale */}
              <div className="flex flex-col justify-between text-xs text-gray-500 dark:text-gray-400 font-mono w-12 text-right pr-2">
                <span className="text-green-500 dark:text-green-400 font-semibold">{Math.round(maxPing)}ms</span>
                <span>{Math.round((maxPing + minPing) / 2)}ms</span>
                <span className="text-gray-400 dark:text-gray-600">{Math.round(minPing)}ms</span>
              </div>

              {/* Chart area */}
              <div className="flex-1 relative">
                {/* Grid lines */}
                <div className="absolute inset-0 flex flex-col justify-between pointer-events-none">
                  <div className="border-t border-dashed border-gray-300 dark:border-gray-700"></div>
                  <div className="border-t border-dashed border-gray-300 dark:border-gray-700"></div>
                  <div className="border-t border-dashed border-gray-300 dark:border-gray-700"></div>
                </div>

                {/* Bars */}
                <div className="h-48 flex items-end gap-[2px] relative z-10">
                  {heartbeats.slice(0, 100).reverse().map((heartbeat: Heartbeat, i: number) => {
                    const status = STATUS_COLORS[heartbeat.status as keyof typeof STATUS_COLORS];
                    // Calculate height based on dynamic scale
                    let heightPercent = 5; // minimum height for visibility
                    if (heartbeat.ping > 0 && maxPing > minPing) {
                      heightPercent = Math.max(5, ((heartbeat.ping - minPing) / (maxPing - minPing)) * 100);
                    }

                    return (
                      <div
                        key={heartbeat.id}
                        className={`flex-1 ${status.dot} rounded-t-md hover:opacity-90 transition-all cursor-pointer hover:scale-y-105 hover:shadow-lg`}
                        style={{
                          height: `${heightPercent}%`,
                          minHeight: '6px'
                        }}
                        onMouseEnter={(e) => {
                          const rect = e.currentTarget.getBoundingClientRect();
                          setHoveredHeartbeat({
                            heartbeat,
                            x: rect.left + rect.width / 2,
                            y: rect.top
                          });
                        }}
                        onMouseLeave={() => setHoveredHeartbeat(null)}
                      />
                    );
                  })}
                </div>
              </div>
            </div>

            <div className="mt-4 flex justify-between items-center">
              <div className="flex gap-4 text-xs text-gray-500 dark:text-gray-400">
                <span>← Oldest</span>
                <span className="text-gray-400 dark:text-gray-600">|</span>
                <span>Latest →</span>
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400">
                <span className="font-medium">Range:</span> {Math.round(minPing)}ms - {Math.round(maxPing)}ms
              </div>
            </div>

            {/* Fancy Tooltip */}
            {hoveredHeartbeat && (
              <div
                className="fixed z-50 pointer-events-none"
                style={{
                  left: `${hoveredHeartbeat.x}px`,
                  top: `${hoveredHeartbeat.y - 10}px`,
                  transform: 'translate(-50%, -100%)'
                }}
              >
                <div className="bg-gray-900 dark:bg-black text-white rounded-lg shadow-2xl p-4 border border-gray-700 dark:border-gray-600 min-w-[250px]">
                  <div className="flex items-center gap-2 mb-2">
                    <div className={`w-3 h-3 rounded-full ${STATUS_COLORS[hoveredHeartbeat.heartbeat.status as keyof typeof STATUS_COLORS].dot}`}></div>
                    <span className="font-semibold text-lg">
                      {STATUS_COLORS[hoveredHeartbeat.heartbeat.status as keyof typeof STATUS_COLORS].label}
                    </span>
                  </div>
                  <div className="space-y-1.5 text-sm">
                    <div className="flex justify-between gap-4">
                      <span className="text-gray-400">Response Time:</span>
                      <span className="font-mono font-medium text-green-400">{hoveredHeartbeat.heartbeat.ping}ms</span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span className="text-gray-400">Time:</span>
                      <span className="font-medium">{formatRelativeTime(new Date(hoveredHeartbeat.heartbeat.time))}</span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span className="text-gray-400">Exact:</span>
                      <span className="text-xs">{new Date(hoveredHeartbeat.heartbeat.time).toLocaleString()}</span>
                    </div>
                    {hoveredHeartbeat.heartbeat.message && (
                      <div className="mt-2 pt-2 border-t border-gray-700">
                        <div className="text-gray-400 text-xs mb-1">Message:</div>
                        <div className="text-gray-200 break-words">{hoveredHeartbeat.heartbeat.message}</div>
                      </div>
                    )}
                  </div>
                  <div className="absolute bottom-0 left-1/2 transform -translate-x-1/2 translate-y-full">
                    <div className="w-0 h-0 border-l-8 border-r-8 border-t-8 border-transparent border-t-gray-900 dark:border-t-black"></div>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Heartbeat List */}
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg shadow-lg dark:shadow-2xl overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-900">
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">
              Recent Checks
              <span className="ml-2 text-sm font-normal text-gray-500 dark:text-gray-400">
                Last 20 heartbeats
              </span>
            </h3>
          </div>
          {isLoadingHeartbeats ? (
            <div className="text-center py-8">
              <div className="inline-block h-6 w-6 animate-spin rounded-full border-2 border-solid border-primary border-r-transparent"></div>
            </div>
          ) : heartbeats && heartbeats.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
                <thead className="bg-gray-50 dark:bg-black/50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Response Time
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Message
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Time
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-800">
                  {heartbeats.slice(0, 20).map((heartbeat: Heartbeat) => {
                    const status = STATUS_COLORS[heartbeat.status as keyof typeof STATUS_COLORS];
                    const heartbeatDate = new Date(heartbeat.time);
                    return (
                      <tr
                        key={heartbeat.id}
                        className="hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors cursor-pointer group"
                      >
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex items-center gap-2">
                            <div className={`w-2 h-2 rounded-full ${status.dot} group-hover:animate-pulse`}></div>
                            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${status.bg} ${status.text}`}>
                              {status.label}
                            </span>
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-mono font-semibold text-gray-900 dark:text-white">
                              {heartbeat.ping}ms
                            </span>
                            {heartbeat.ping < 100 && (
                              <span className="text-xs text-green-500 dark:text-green-400">●</span>
                            )}
                            {heartbeat.ping >= 100 && heartbeat.ping < 500 && (
                              <span className="text-xs text-yellow-500 dark:text-yellow-400">●</span>
                            )}
                            {heartbeat.ping >= 500 && (
                              <span className="text-xs text-red-500 dark:text-red-400">●</span>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 max-w-md truncate">
                          {heartbeat.message || <span className="text-gray-400 dark:text-gray-600 italic">No message</span>}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex flex-col">
                            <span className="text-sm text-gray-900 dark:text-white font-medium">
                              {formatRelativeTime(heartbeatDate)}
                            </span>
                            <span className="text-xs text-gray-500 dark:text-gray-500">
                              {heartbeatDate.toLocaleTimeString()}
                            </span>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="text-center py-8">
              <p className="text-sm text-gray-500 dark:text-gray-400">No heartbeats yet</p>
            </div>
          )}
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 bg-black/60 dark:bg-black/80 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg shadow-2xl p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
              Delete Monitor
            </h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
              Are you sure you want to delete this monitor? This action cannot be undone and all heartbeat data will be lost.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowDeleteConfirm(false)}
                disabled={deleteMutation.isPending}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-50 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => deleteMutation.mutate()}
                disabled={deleteMutation.isPending}
                className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-500 disabled:opacity-50 transition-colors"
              >
                {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
