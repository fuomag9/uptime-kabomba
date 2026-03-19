"use client";

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import Link from 'next/link';
import { useWebSocket } from '@/hooks/useWebSocket';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Plus, FileText } from 'lucide-react';

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
          <Button asChild>
            <Link href="/monitors/new">
              <Plus className="-ml-0.5 mr-1.5 h-5 w-5" />
              Add Monitor
            </Link>
          </Button>
        </div>
      </div>

      {totalMonitors > 0 ? (
        <>
          {/* Stats Grid */}
          <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardContent>
                <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Total Monitors</dt>
                <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">
                  {totalMonitors}
                </dd>
              </CardContent>
            </Card>
            <Card>
              <CardContent>
                <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Up</dt>
                <dd className="mt-1 text-3xl font-semibold tracking-tight text-green-600 dark:text-green-400">
                  {upMonitors}
                </dd>
              </CardContent>
            </Card>
            <Card>
              <CardContent>
                <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Down</dt>
                <dd className="mt-1 text-3xl font-semibold tracking-tight text-red-600 dark:text-red-400">
                  {downMonitors}
                </dd>
              </CardContent>
            </Card>
            <Card>
              <CardContent>
                <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Paused</dt>
                <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-600 dark:text-gray-400">
                  {pausedMonitors}
                </dd>
              </CardContent>
            </Card>
          </div>

          {/* Getting Started */}
          <Card className="mt-8">
            <CardContent>
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
                    <Plus className="h-6 w-6 text-gray-400 mr-3" />
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
                    <FileText className="h-6 w-6 text-gray-400 mr-3" />
                    <div>
                      <div className="text-sm font-medium text-gray-900 dark:text-white">Status Page</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">Create a public status page</div>
                    </div>
                  </div>
                </Link>
              </div>
            </CardContent>
          </Card>
        </>
      ) : (
        <Card className="mt-8 text-center py-12">
          <CardContent>
            <FileText className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
              No monitors
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Get started by creating your first monitor
            </p>
            <div className="mt-6">
              <Button asChild>
                <Link href="/monitors/new">
                  <Plus className="-ml-0.5 mr-1.5 h-5 w-5" />
                  New Monitor
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
