"use client";

import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient, CreateMonitorRequest } from '@/lib/api';
import MonitorForm from '@/components/monitors/MonitorForm';
import { Card, CardContent } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';

export default function NewMonitorPage() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: async (data: { monitor: CreateMonitorRequest, notificationIds: number[], useDefaultNotifications: boolean }) => {
      const createdMonitor = await apiClient.createMonitor(data.monitor);
      // Update notifications based on user preference
      if (data.useDefaultNotifications) {
        // Use default notifications - set the flag to false (will use defaults)
        await apiClient.updateMonitorNotifications(createdMonitor.id, [], true);
      } else {
        // Explicit notification configuration
        await apiClient.updateMonitorNotifications(createdMonitor.id, data.notificationIds, false);
      }
      return createdMonitor;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['monitors'] });
      router.push('/monitors');
    },
  });

  const handleSubmit = (data: { monitor: CreateMonitorRequest, notificationIds: number[], useDefaultNotifications: boolean }) => {
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
        <Alert variant="destructive" className="mb-6">
          <AlertDescription>
            Failed to create monitor: {(createMutation.error as any).message}
          </AlertDescription>
        </Alert>
      )}

      <Card>
        <CardContent className="p-6">
          <MonitorForm
            onSubmit={handleSubmit}
            onCancel={() => router.push('/monitors')}
            isSubmitting={createMutation.isPending}
          />
        </CardContent>
      </Card>
    </div>
  );
}
