"use client";

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import { useMonitorHeartbeat } from '@/hooks/useMonitorHeartbeats';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useMemo, useState } from 'react';

const STATUS_COLORS = {
  0: { dot: 'bg-red-500', label: 'Down' },
  1: { dot: 'bg-green-500', label: 'Up' },
  2: { dot: 'bg-yellow-500', label: 'Pending' },
  3: { dot: 'bg-blue-500', label: 'Maintenance' },
};

function MonitorSidebarItem({ monitor }: { monitor: any }) {
  const heartbeat = useMonitorHeartbeat(monitor.id);
  const pathname = usePathname();
  const isActive = pathname === `/monitors/${monitor.id}`;

  // Use real-time status if available, otherwise use last_heartbeat from API, otherwise default to pending
  const status = heartbeat?.status ?? monitor.last_heartbeat?.status ?? 2;
  const ping = heartbeat?.ping ?? monitor.last_heartbeat?.ping;
  const statusStyle = STATUS_COLORS[status as keyof typeof STATUS_COLORS];

  return (
    <Link
      href={`/monitors/${monitor.id}`}
      className={`block px-3 py-2 rounded-md text-sm transition-colors ${isActive
        ? 'bg-primary text-white'
        : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
        } ${monitor.active ? '' : 'opacity-70'}`}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center min-w-0 flex-1">
          <div className={`h-2 w-2 rounded-full ${statusStyle.dot} mr-2 flex-shrink-0`} />
          <span className="truncate">{monitor.name}</span>
          {!monitor.active && (
            <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">⏸</span>
          )}
        </div>
        {ping !== undefined && (
          <span className={`ml-2 text-xs flex-shrink-0 ${isActive ? 'text-white' : 'text-gray-500 dark:text-gray-400'}`}>
            {ping}ms
          </span>
        )}
      </div>
    </Link>
  );
}

export function MonitorSidebar() {
  const { data: monitors = [], isLoading } = useQuery({
    queryKey: ['monitors'],
    queryFn: () => apiClient.getMonitors(),
    refetchInterval: 30000,
  });
  const [query, setQuery] = useState('');

  const sortedMonitors = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    const filtered = normalizedQuery.length
      ? monitors.filter((monitor) =>
          monitor.name.toLowerCase().includes(normalizedQuery)
        )
      : monitors;
    return [...filtered].sort((a, b) => a.name.localeCompare(b.name));
  }, [monitors, query]);

  return (
    <div className="w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 h-full overflow-y-auto">
      <div className="p-4">
        <div className="mb-4">
          <label className="sr-only" htmlFor="monitor-search">
            Search monitors
          </label>
          <input
            id="monitor-search"
            type="search"
            placeholder="Search monitors"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className="w-full rounded-md border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500"
          />
        </div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-gray-900 dark:text-white">Monitors</h2>
          <Link
            href="/monitors/new"
            className="text-primary hover:text-primary/80"
            title="Add Monitor"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
          </Link>
        </div>

        {isLoading ? (
          <div className="text-center py-4">
            <div className="inline-block h-6 w-6 animate-spin rounded-full border-2 border-solid border-primary border-r-transparent"></div>
          </div>
        ) : sortedMonitors.length > 0 ? (
          <div className="space-y-1">
            {sortedMonitors.map((monitor) => (
              <MonitorSidebarItem key={monitor.id} monitor={monitor} />
            ))}
          </div>
        ) : (
          <div className="text-center py-8">
            <p className="text-sm text-gray-500 dark:text-gray-400">No monitors yet</p>
            <Link
              href="/monitors/new"
              className="mt-2 inline-block text-sm text-primary hover:underline"
            >
              Add your first monitor
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
