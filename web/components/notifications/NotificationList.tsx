import { Notification } from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

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
        <Card key={notification.id}>
          <CardContent>
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-3">
                  <h3 className="text-lg font-semibold">
                    {notification.name}
                  </h3>
                  <Badge variant="secondary">
                    {getProviderLabel(notification.type)}
                  </Badge>
                  {notification.is_default && (
                    <Badge className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300">
                      Default
                    </Badge>
                  )}
                  {!notification.active && (
                    <Badge variant="outline">
                      Inactive
                    </Badge>
                  )}
                </div>
                <p className="mt-1 text-sm text-muted-foreground">
                  Created {new Date(notification.created_at).toLocaleDateString()}
                </p>
              </div>

              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onTest(notification.id)}
                >
                  Test
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onEdit(notification)}
                >
                  Edit
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => onDelete(notification.id)}
                >
                  Delete
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
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
