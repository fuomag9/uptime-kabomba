"use client";

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient, Heartbeat } from '@/lib/api';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useState } from 'react';
import { useMonitorHeartbeat } from '@/hooks/useMonitorHeartbeats';
import { useWebSocket } from '@/hooks/useWebSocket';
import PeriodSelector, { PeriodType } from '@/components/ui/PeriodSelector';
import HeartbeatChart from '@/components/monitors/HeartbeatChart';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from '@/components/ui/alert-dialog';
import { ChevronLeft } from 'lucide-react';

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

const STATUS_BADGE_VARIANT: Record<number, 'default' | 'destructive' | 'secondary'> = {
  0: 'destructive',
  1: 'default',
  2: 'secondary',
  3: 'secondary',
};

export default function MonitorDetailPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const monitorId = parseInt(params.id as string);

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<PeriodType>('1h');
  const [customRange, setCustomRange] = useState<{ start: Date; end: Date } | null>(null);

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
    queryKey: ['heartbeats', monitorId, selectedPeriod, customRange?.start?.toISOString(), customRange?.end?.toISOString()],
    queryFn: () => {
      if (selectedPeriod === 'custom' && customRange) {
        return apiClient.getHeartbeats(monitorId, {
          startTime: customRange.start.toISOString(),
          endTime: customRange.end.toISOString(),
        });
      }
      return apiClient.getHeartbeats(monitorId, {
        period: selectedPeriod as '1h' | '3h' | '6h' | '24h',
      });
    },
    // Only auto-refresh for short periods to avoid constant graph recalculation on large time ranges
    refetchInterval: selectedPeriod === '1h' ? 5000 : false,
  });

  const handlePeriodChange = (period: PeriodType, range?: { start: Date; end: Date }) => {
    setSelectedPeriod(period);
    if (period === 'custom' && range) {
      setCustomRange(range);
    }
  };

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
      <div className="space-y-6">
        <div className="flex items-center gap-2">
          <Skeleton className="h-5 w-5" />
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-5 w-12" />
        </div>
        <Skeleton className="h-4 w-64" />
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-3">
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
        <Skeleton className="h-64" />
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

  const badgeVariant = latestHeartbeat
    ? STATUS_BADGE_VARIANT[latestHeartbeat.status as keyof typeof STATUS_BADGE_VARIANT] ?? 'secondary'
    : 'secondary';

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
              <ChevronLeft className="h-5 w-5" />
            </Link>
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-white truncate">
              {monitor.name}
            </h1>
            <Badge
              variant={badgeVariant}
              className={`${statusStyle.bg} ${statusStyle.text}`}
            >
              {statusStyle.label}
            </Badge>
            {!monitor.active && (
              <span className="inline-flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
                <span className="text-sm leading-none">&#x23F8;</span>
                Paused
              </span>
            )}
          </div>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {monitor.type.toUpperCase()} &bull; {monitor.url}
          </p>
        </div>
        <div className="mt-4 flex flex-wrap gap-2 sm:ml-16 sm:mt-0">
          <Button
            variant="outline"
            onClick={() => toggleActiveMutation.mutate()}
            disabled={toggleActiveMutation.isPending}
          >
            {monitor.active ? 'Pause' : 'Resume'}
          </Button>
          <Button variant="outline" asChild>
            <Link href={`/monitors/${monitorId}/edit`}>
              Edit
            </Link>
          </Button>
          <AlertDialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
            <AlertDialogTrigger asChild>
              <Button variant="destructive">
                Delete
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Monitor</AlertDialogTitle>
                <AlertDialogDescription>
                  Are you sure you want to delete this monitor? This action cannot be undone and all heartbeat data will be lost.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel disabled={deleteMutation.isPending}>
                  Cancel
                </AlertDialogCancel>
                <AlertDialogAction
                  variant="destructive"
                  onClick={() => deleteMutation.mutate()}
                  disabled={deleteMutation.isPending}
                >
                  {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      {/* Stats */}
      <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-3">
        <Card className="transition-all hover:shadow-xl hover:scale-105">
          <CardContent>
            <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Uptime</dt>
            <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">
              {uptime.toFixed(2)}%
            </dd>
          </CardContent>
        </Card>
        <Card className="transition-all hover:shadow-xl hover:scale-105">
          <CardContent>
            <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Average Ping</dt>
            <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">
              {avgPing > 0 ? `${avgPing.toFixed(0)}ms` : '--'}
            </dd>
          </CardContent>
        </Card>
        <Card className="transition-all hover:shadow-xl hover:scale-105">
          <CardContent>
            <dt className="truncate text-sm font-medium text-gray-500 dark:text-gray-400">Check Interval</dt>
            <dd className="mt-1 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">
              {monitor.interval}s
            </dd>
          </CardContent>
        </Card>
      </div>

      {/* Heartbeat History */}
      <div className="mt-8">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-medium text-gray-900 dark:text-white">
            Response Time
            <span className="ml-2 text-sm font-normal text-gray-500 dark:text-gray-400">
              {heartbeats?.length || 0} checks
            </span>
          </h2>
          <PeriodSelector
            value={selectedPeriod}
            onChange={handlePeriodChange}
            customStart={customRange?.start}
            customEnd={customRange?.end}
          />
        </div>

        {/* Recharts Graph */}
        {heartbeats && heartbeats.length > 0 && (
          <Card className="mb-6">
            <CardContent className="p-6">
              <HeartbeatChart heartbeats={heartbeats} height={300} />
            </CardContent>
          </Card>
        )}

        {/* Heartbeat List */}
        <Card>
          <CardContent className="p-0">
            <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-800">
              <h3 className="text-base font-semibold text-gray-900 dark:text-white">
                Recent Checks
                <span className="ml-2 text-sm font-normal text-gray-500 dark:text-gray-400">
                  Last 20 heartbeats
                </span>
              </h3>
            </div>
            {isLoadingHeartbeats ? (
              <div className="p-6 space-y-3">
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
              </div>
            ) : heartbeats && heartbeats.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="px-6">Status</TableHead>
                    <TableHead className="px-6">Response Time</TableHead>
                    <TableHead className="px-6">Message</TableHead>
                    <TableHead className="px-6">Time</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {heartbeats.slice(0, 20).map((heartbeat: Heartbeat) => {
                    const status = STATUS_COLORS[heartbeat.status as keyof typeof STATUS_COLORS];
                    const hbBadgeVariant = STATUS_BADGE_VARIANT[heartbeat.status as keyof typeof STATUS_BADGE_VARIANT] ?? 'secondary';
                    const heartbeatDate = new Date(heartbeat.time);
                    return (
                      <TableRow
                        key={heartbeat.id}
                        className="cursor-pointer group"
                      >
                        <TableCell className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <div className={`w-2 h-2 rounded-full ${status.dot} group-hover:animate-pulse`}></div>
                            <Badge
                              variant={hbBadgeVariant}
                              className={`${status.bg} ${status.text}`}
                            >
                              {status.label}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-mono font-semibold text-gray-900 dark:text-white">
                              {heartbeat.ping}ms
                            </span>
                            {heartbeat.ping < 100 && (
                              <span className="text-xs text-green-500 dark:text-green-400">&#x25CF;</span>
                            )}
                            {heartbeat.ping >= 100 && heartbeat.ping < 500 && (
                              <span className="text-xs text-yellow-500 dark:text-yellow-400">&#x25CF;</span>
                            )}
                            {heartbeat.ping >= 500 && (
                              <span className="text-xs text-red-500 dark:text-red-400">&#x25CF;</span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 max-w-md truncate">
                          {heartbeat.message || <span className="text-gray-400 dark:text-gray-600 italic">No message</span>}
                        </TableCell>
                        <TableCell className="px-6 py-4">
                          <div className="flex flex-col">
                            <span className="text-sm text-gray-900 dark:text-white font-medium">
                              {formatRelativeTime(heartbeatDate)}
                            </span>
                            <span className="text-xs text-gray-500 dark:text-gray-500">
                              {heartbeatDate.toLocaleTimeString()}
                            </span>
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            ) : (
              <div className="text-center py-8">
                <p className="text-sm text-gray-500 dark:text-gray-400">No heartbeats yet</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
