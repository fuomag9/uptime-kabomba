'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient, Notification, NotificationProvider } from '@/lib/api';
import NotificationList from '@/components/notifications/NotificationList';
import NotificationForm from '@/components/notifications/NotificationForm';

export default function NotificationsPage() {
  const router = useRouter();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [providers, setProviders] = useState<NotificationProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingNotification, setEditingNotification] = useState<Notification | null>(null);

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
    if (!confirm('Are you sure you want to delete this notification?')) {
      return;
    }

    try {
      await apiClient.deleteNotification(id);
      setNotifications((notifications || []).filter((n) => n.id !== id));
    } catch (err: any) {
      console.error('Failed to delete notification:', err);
      alert('Failed to delete notification: ' + (err.message || 'Unknown error'));
    }
  }

  async function handleTest(id: number) {
    try {
      const result = await apiClient.testNotification(id);
      alert(result.message);
    } catch (err: any) {
      console.error('Failed to test notification:', err);
      alert('Failed to test notification: ' + (err.message || 'Unknown error'));
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
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading notifications...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
        {error}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Notifications</h1>
          <p className="mt-1 text-sm text-gray-600">
            Manage notification channels for monitor alerts
          </p>
        </div>
        <button
          onClick={handleCreate}
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors"
        >
          Add Notification
        </button>
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
        <div className="bg-gray-50 border border-gray-200 rounded-lg p-8 text-center">
          <p className="text-gray-600">No notifications configured yet.</p>
          <button
            onClick={handleCreate}
            className="mt-4 text-blue-600 hover:text-blue-700 font-medium"
          >
            Create your first notification
          </button>
        </div>
      ) : (
        <NotificationList
          notifications={notifications}
          onEdit={handleEdit}
          onDelete={handleDelete}
          onTest={handleTest}
        />
      )}
    </div>
  );
}
