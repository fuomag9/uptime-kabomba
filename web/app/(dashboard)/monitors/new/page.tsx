"use client";

import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient, CreateMonitorRequest } from '@/lib/api';
import MonitorForm from '@/components/monitors/MonitorForm';

export default function NewMonitorPage() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: async (data: { monitor: CreateMonitorRequest, notificationIds: number[] }) => {
      const createdMonitor = await apiClient.createMonitor(data.monitor);
      // Update notifications if any are selected
      if (data.notificationIds.length > 0) {
        await apiClient.updateMonitorNotifications(createdMonitor.id, data.notificationIds);
      }
      return createdMonitor;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['monitors'] });
      router.push('/monitors');
    },
  });

  const handleSubmit = (data: { monitor: CreateMonitorRequest, notificationIds: number[] }) => {
    createMutation.mutate(data);
  };

  return (
    <div className="max-w-4xl">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
          Create New Monitor
        </h1>
        <p className="mt-2 text-sm text-gray-700 dark:text-gray-300">
          Configure a new monitor to track the status and performance of your services
        </p>
      </div>

      {createMutation.error && (
        <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
          <p className="text-sm text-red-800 dark:text-red-200">
            Failed to create monitor: {(createMutation.error as any).message}
          </p>
        </div>
      )}

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <MonitorForm
          onSubmit={handleSubmit}
          onCancel={() => router.push('/monitors')}
          isSubmitting={createMutation.isPending}
        />
      </div>
    </div>
  );
}
