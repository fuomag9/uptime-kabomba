"use client";

import { useState } from 'react';
import { MonitorWithStatus } from '@/lib/api';
import Link from 'next/link';
import { useMonitorHeartbeat } from '@/hooks/useMonitorHeartbeats';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import PeriodSelector from '@/components/ui/PeriodSelector';

interface MonitorCardProps {
  monitor: MonitorWithStatus;
  latestPing?: number;
  uptime?: number;
}

const STATUS_COLORS = {
  0: { bg: 'bg-red-100 dark:bg-red-900/20', text: 'text-red-800 dark:text-red-200', label: 'Down', bar: 'bg-red-500' },
  1: { bg: 'bg-green-100 dark:bg-green-900/20', text: 'text-green-800 dark:text-green-200', label: 'Up', bar: 'bg-green-500' },
  2: { bg: 'bg-yellow-100 dark:bg-yellow-900/20', text: 'text-yellow-800 dark:text-yellow-200', label: 'Pending', bar: 'bg-yellow-500' },
  3: { bg: 'bg-blue-100 dark:bg-blue-900/20', text: 'text-blue-800 dark:text-blue-200', label: 'Maintenance', bar: 'bg-blue-500' },
};

export default function MonitorCard({ monitor, latestPing, uptime }: MonitorCardProps) {
  const [period, setPeriod] = useState<'1h' | '24h' | '7d' | '30d' | '90d'>('7d');

  // Get real-time heartbeat
  const heartbeat = useMonitorHeartbeat(monitor.id);

  // Fetch recent heartbeats for the bar chart based on selected period
  const { data: heartbeats = [] } = useQuery({
    queryKey: ['heartbeats', monitor.id, period],
    queryFn: () => apiClient.getHeartbeats(monitor.id, { period, limit: 50 }),
    refetchInterval: 60000, // Refetch every minute
  });

  // Use real-time status if available, otherwise use last_heartbeat from API, otherwise default to pending
  const status = heartbeat?.status ?? monitor.last_heartbeat?.status ?? 2;
  const ping = heartbeat?.ping ?? monitor.last_heartbeat?.ping ?? latestPing;
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

        {/* Period selector and Heartbeat bar */}
        <div className="mt-4">
          <div
            className="mb-2"
            onClick={(e) => e.preventDefault()}
          >
            <PeriodSelector
              value={period}
              onChange={(p) => setPeriod(p as '1h' | '24h' | '7d' | '30d' | '90d')}
              compact
            />
          </div>
          <div className="h-8 flex items-end gap-0.5">
            {heartbeats.length > 0 ? (
              heartbeats.slice(0, 50).reverse().map((hb, i) => {
                const hbStatusStyle = STATUS_COLORS[hb.status as keyof typeof STATUS_COLORS];
                return (
                  <div
                    key={hb.id || i}
                    className={`flex-1 rounded-sm ${hbStatusStyle.bar}`}
                    style={{ height: '100%' }}
                    title={`${hbStatusStyle.label} - ${hb.ping}ms`}
                  />
                );
              })
            ) : (
              Array.from({ length: 50 }).map((_, i) => (
                <div
                  key={i}
                  className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-sm"
                  style={{ height: '100%' }}
                />
              ))
            )}
          </div>
        </div>
      </div>
    </Link>
  );
}
