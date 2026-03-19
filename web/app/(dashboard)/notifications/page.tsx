'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient, Notification, NotificationProvider } from '@/lib/api';
import NotificationList from '@/components/notifications/NotificationList';
import NotificationForm from '@/components/notifications/NotificationForm';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { toast } from 'sonner';

export default function NotificationsPage() {
  const router = useRouter();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [providers, setProviders] = useState<NotificationProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingNotification, setEditingNotification] = useState<Notification | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  async function loadData() {
    try {
      setLoading(true);
      setError(null);
      const [notificationsData, providersData] = await Promise.all([
        apiClient.getNotifications(),
        apiClient.getNotificationProviders(),
      ]);
      setNotifications(notificationsData || []);
      setProviders(providersData || []);
    } catch (err: any) {
      console.error('Failed to load notifications:', err);
      setError(err.message || 'Failed to load notifications');
      if (err.status === 401) {
        router.push('/login');
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(id: number) {
    try {
      await apiClient.deleteNotification(id);
      setNotifications((notifications || []).filter((n) => n.id !== id));
    } catch (err: any) {
      console.error('Failed to delete notification:', err);
      toast.error('Failed to delete notification: ' + (err.message || 'Unknown error'));
    }
  }

  function handleDeleteRequest(id: number) {
    setDeleteId(id);
  }

  function handleDeleteConfirm() {
    if (deleteId !== null) {
      handleDelete(deleteId);
      setDeleteId(null);
    }
  }

  async function handleTest(id: number) {
    try {
      const result = await apiClient.testNotification(id);
      toast.success(result.message);
    } catch (err: any) {
      console.error('Failed to test notification:', err);
      toast.error('Failed to test notification: ' + (err.message || 'Unknown error'));
    }
  }

  function handleEdit(notification: Notification) {
    setEditingNotification(notification);
    setShowForm(true);
  }

  function handleCreate() {
    setEditingNotification(null);
    setShowForm(true);
  }

  function handleFormClose() {
    setShowForm(false);
    setEditingNotification(null);
  }

  function handleFormSuccess() {
    setShowForm(false);
    setEditingNotification(null);
    loadData();
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <Skeleton className="h-8 w-40" />
            <Skeleton className="mt-2 h-4 w-64" />
          </div>
          <Skeleton className="h-8 w-36" />
        </div>
        <div className="grid gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">Notifications</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Manage notification channels for monitor alerts
          </p>
        </div>
        <Button onClick={handleCreate}>
          Add Notification
        </Button>
      </div>

      {showForm && (
        <NotificationForm
          notification={editingNotification}
          providers={providers}
          onClose={handleFormClose}
          onSuccess={handleFormSuccess}
        />
      )}

      {(!notifications || notifications.length === 0) ? (
        <Card>
          <CardContent className="py-8 text-center">
            <p className="text-muted-foreground">No notifications configured yet.</p>
            <Button
              variant="link"
              onClick={handleCreate}
              className="mt-4"
            >
              Create your first notification
            </Button>
          </CardContent>
        </Card>
      ) : (
        <NotificationList
          notifications={notifications}
          onEdit={handleEdit}
          onDelete={handleDeleteRequest}
          onTest={handleTest}
        />
      )}

      <AlertDialog open={deleteId !== null} onOpenChange={(open) => { if (!open) setDeleteId(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this notification? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={handleDeleteConfirm}>
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
