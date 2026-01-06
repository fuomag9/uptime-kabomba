import { Notification } from '@/lib/api';

interface NotificationListProps {
  notifications: Notification[];
  onEdit: (notification: Notification) => void;
  onDelete: (id: number) => void;
  onTest: (id: number) => void;
}

export default function NotificationList({
  notifications,
  onEdit,
  onDelete,
  onTest,
}: NotificationListProps) {
  return (
    <div className="grid gap-4">
      {notifications.map((notification) => (
        <div
          key={notification.id}
          className="bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow"
        >
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-3">
                <h3 className="text-lg font-semibold text-gray-900">
                  {notification.name}
                </h3>
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                  {getProviderLabel(notification.type)}
                </span>
                {notification.is_default && (
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                    Default
                  </span>
                )}
                {!notification.active && (
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                    Inactive
                  </span>
                )}
              </div>
              <p className="mt-1 text-sm text-gray-600">
                Created {new Date(notification.created_at).toLocaleDateString()}
              </p>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={() => onTest(notification.id)}
                className="text-blue-600 hover:text-blue-700 px-3 py-1 text-sm font-medium rounded border border-blue-600 hover:bg-blue-50 transition-colors"
              >
                Test
              </button>
              <button
                onClick={() => onEdit(notification)}
                className="text-gray-600 hover:text-gray-700 px-3 py-1 text-sm font-medium rounded border border-gray-300 hover:bg-gray-50 transition-colors"
              >
                Edit
              </button>
              <button
                onClick={() => onDelete(notification.id)}
                className="text-red-600 hover:text-red-700 px-3 py-1 text-sm font-medium rounded border border-red-600 hover:bg-red-50 transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function getProviderLabel(type: string): string {
  const labels: Record<string, string> = {
    smtp: 'Email (SMTP)',
    webhook: 'Webhook',
    discord: 'Discord',
    slack: 'Slack',
    telegram: 'Telegram',
    teams: 'Microsoft Teams',
    pagerduty: 'PagerDuty',
    pushover: 'Pushover',
    gotify: 'Gotify',
    ntfy: 'Ntfy',
  };
  return labels[type] || type;
}
