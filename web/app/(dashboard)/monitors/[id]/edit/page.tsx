"use client";

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient, UpdateMonitorRequest } from '@/lib/api';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import MonitorForm from '@/components/monitors/MonitorForm';

export default function EditMonitorPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const monitorId = parseInt(params.id as string);

  const { data: monitor, isLoading } = useQuery({
    queryKey: ['monitor', monitorId],
    queryFn: () => apiClient.getMonitor(monitorId),
  });

  const updateMutation = useMutation({
    mutationFn: (data: UpdateMonitorRequest) => apiClient.updateMonitor(monitorId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['monitor', monitorId] });
      queryClient.invalidateQueries({ queryKey: ['monitors'] });
      router.push(`/monitors/${monitorId}`);
    },
  });

  if (isLoading) {
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

  return (
    <div className="max-w-4xl">
      <div className="mb-6">
        <div className="flex items-center gap-2 mb-2">
          <Link
            href={`/monitors/${monitorId}`}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </Link>
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
            Edit Monitor
          </h1>
        </div>
        <p className="ml-7 text-sm text-gray-700 dark:text-gray-300">
          Update the configuration for {monitor.name}
        </p>
      </div>

      {updateMutation.error && (
        <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
          <p className="text-sm text-red-800 dark:text-red-200">
            Failed to update monitor: {(updateMutation.error as any).message}
          </p>
        </div>
      )}

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <MonitorForm
          initialData={{
            name: monitor.name,
            type: monitor.type,
            url: monitor.url,
            interval: monitor.interval,
            timeout: monitor.timeout,
            config: monitor.config,
          }}
          onSubmit={(data) => updateMutation.mutate({ ...data, active: monitor.active })}
          onCancel={() => router.push(`/monitors/${monitorId}`)}
          isSubmitting={updateMutation.isPending}
        />
      </div>
    </div>
  );
}
