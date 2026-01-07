"use client";

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import Link from 'next/link';
import { useWebSocket } from '@/hooks/useWebSocket';

export default function MonitorsPage() {
  // Connect to WebSocket for real-time updates
  useWebSocket();
  const { data: monitors = [] } = useQuery({
    queryKey: ['monitors'],
    queryFn: () => apiClient.getMonitors(),
    refetchInterval: 30000,
  });

  // Calculate overall stats
  const totalMonitors = monitors.length;
  const activeMonitors = monitors.filter(m => m.active).length;
  const upMonitors = monitors.filter(m => m.last_heartbeat?.status === 1).length;
  const downMonitors = monitors.filter(m => m.last_heartbeat?.status === 0).length;
  const pausedMonitors = totalMonitors - activeMonitors;

  return (
    <div>
      <div className="sm:flex sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
            Dashboard
          </h1>
          <p className="mt-2 text-sm text-gray-700 dark:text-gray-300">
            Overview of all your monitors
          </p>
        </div>
        <div className="mt-4 sm:mt-0">
          <Link
            href="/monitors/new"
            className="inline-flex items-center rounded-md bg-primary px-3 py-2 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-primary/90"
          >
            <svg className="-ml-0.5 mr-1.5 h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" />
            </svg>
            Add Monitor
          </Link>
        </div>
      </div>

      {totalMonitors > 0 ? (
        <>
          {/* Stats Grid */}
          <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
            <div className="overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 py-5 shadow sm:p-6">
              <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Total Monitors</dt>
              <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">
                {totalMonitors}
              </dd>
            </div>
            <div className="overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 py-5 shadow sm:p-6">
              <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Up</dt>
              <dd className="mt-1 text-3xl font-semibold tracking-tight text-green-600 dark:text-green-400">
                {upMonitors}
              </dd>
            </div>
            <div className="overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 py-5 shadow sm:p-6">
              <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Down</dt>
              <dd className="mt-1 text-3xl font-semibold tracking-tight text-red-600 dark:text-red-400">
                {downMonitors}
              </dd>
            </div>
            <div className="overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 py-5 shadow sm:p-6">
              <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Paused</dt>
              <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-600 dark:text-gray-400">
                {pausedMonitors}
              </dd>
            </div>
          </div>

          {/* Getting Started */}
          <div className="mt-8 bg-white dark:bg-gray-800 shadow rounded-lg p-6">
            <h2 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
              Quick Start
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
              Select a monitor from the sidebar to view detailed information, heartbeat history, and manage settings.
            </p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Link
                href="/monitors/new"
                className="block p-4 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg hover:border-primary dark:hover:border-primary transition-colors"
              >
                <div className="flex items-center">
                  <svg className="h-6 w-6 text-gray-400 mr-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  <div>
                    <div className="text-sm font-medium text-gray-900 dark:text-white">Add Monitor</div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">Create a new monitor</div>
                  </div>
                </div>
              </Link>
              <Link
                href="/status-pages"
                className="block p-4 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg hover:border-primary dark:hover:border-primary transition-colors"
              >
                <div className="flex items-center">
                  <svg className="h-6 w-6 text-gray-400 mr-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                  <div>
                    <div className="text-sm font-medium text-gray-900 dark:text-white">Status Page</div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">Create a public status page</div>
                  </div>
                </div>
              </Link>
            </div>
          </div>
        </>
      ) : (
        <div className="mt-8 text-center py-12 bg-white dark:bg-gray-800 rounded-lg shadow">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
          <h3 className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
            No monitors
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Get started by creating your first monitor
          </p>
          <div className="mt-6">
            <Link
              href="/monitors/new"
              className="inline-flex items-center rounded-md bg-primary px-3 py-2 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-primary/90"
            >
              <svg className="-ml-0.5 mr-1.5 h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" />
              </svg>
              New Monitor
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
