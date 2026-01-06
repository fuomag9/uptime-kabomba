"use client";

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import Link from 'next/link';
import MonitorCard from '@/components/monitors/MonitorCard';
import { useWebSocket } from '@/hooks/useWebSocket';

export default function MonitorsPage() {
  // Connect to WebSocket for real-time updates
  useWebSocket();
  const { data: monitors, isLoading } = useQuery({
    queryKey: ['monitors'],
    queryFn: () => apiClient.getMonitors(),
    refetchInterval: 30000, // Refetch every 30 seconds
  });

  return (
    <div>
      <div className="sm:flex sm:items-center">
        <div className="sm:flex-auto">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
            Monitors
          </h1>
          <p className="mt-2 text-sm text-gray-700 dark:text-gray-300">
            Manage and monitor your services
          </p>
        </div>
        <div className="mt-4 sm:ml-16 sm:mt-0 sm:flex-none">
          <Link
            href="/monitors/new"
            className="block rounded-md bg-primary px-3 py-2 text-center text-sm font-semibold text-primary-foreground shadow-sm hover:bg-primary/90"
          >
            Add Monitor
          </Link>
        </div>
      </div>

      <div className="mt-8">
        {isLoading ? (
          <div className="text-center py-12">
            <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-primary border-r-transparent"></div>
            <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">Loading monitors...</p>
          </div>
        ) : monitors && monitors.length > 0 ? (
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {monitors.map((monitor) => (
              <MonitorCard key={monitor.id} monitor={monitor} />
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <svg
              className="mx-auto h-12 w-12 text-gray-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              aria-hidden="true"
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
              Get started by creating a new monitor
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
    </div>
  );
}
