"use client";

import { Monitor } from '@/lib/api';
import Link from 'next/link';
import { useMonitorHeartbeat } from '@/hooks/useMonitorHeartbeats';

interface MonitorCardProps {
  monitor: Monitor;
  latestPing?: number;
  uptime?: number;
}

const STATUS_COLORS = {
  0: { bg: 'bg-red-100 dark:bg-red-900/20', text: 'text-red-800 dark:text-red-200', label: 'Down' },
  1: { bg: 'bg-green-100 dark:bg-green-900/20', text: 'text-green-800 dark:text-green-200', label: 'Up' },
  2: { bg: 'bg-yellow-100 dark:bg-yellow-900/20', text: 'text-yellow-800 dark:text-yellow-200', label: 'Pending' },
  3: { bg: 'bg-blue-100 dark:bg-blue-900/20', text: 'text-blue-800 dark:text-blue-200', label: 'Maintenance' },
};

export default function MonitorCard({ monitor, latestPing, uptime }: MonitorCardProps) {
  // Get real-time heartbeat
  const heartbeat = useMonitorHeartbeat(monitor.id);

  // Use real-time status if available, otherwise default to pending
  const status = heartbeat?.status ?? 2;
  const ping = heartbeat?.ping ?? latestPing;
  const statusStyle = STATUS_COLORS[status as keyof typeof STATUS_COLORS];

  return (
    <Link href={`/monitors/${monitor.id}`}>
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 shadow-sm hover:shadow-md transition-shadow cursor-pointer">
        <div className="flex items-start justify-between">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white truncate">
                {monitor.name}
              </h3>
              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusStyle.bg} ${statusStyle.text}`}>
                {statusStyle.label}
              </span>
            </div>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400 truncate">
              {monitor.type.toUpperCase()} • {monitor.url}
            </p>
          </div>
          {!monitor.active && (
            <span className="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200">
              Paused
            </span>
          )}
        </div>

        <div className="mt-4 grid grid-cols-3 gap-4">
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400">Uptime</p>
            <p className="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
              {uptime !== undefined ? `${uptime.toFixed(2)}%` : '--'}
            </p>
          </div>
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400">Ping</p>
            <p className="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
              {ping !== undefined ? `${ping}ms` : '--'}
            </p>
          </div>
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400">Interval</p>
            <p className="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
              {monitor.interval}s
            </p>
          </div>
        </div>

        {/* Heartbeat bar (placeholder for now) */}
        <div className="mt-4 h-8 flex items-end gap-0.5">
          {Array.from({ length: 50 }).map((_, i) => (
            <div
              key={i}
              className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-sm"
              style={{ height: '100%' }}
            />
          ))}
        </div>
      </div>
    </Link>
  );
}
