import { useState, useEffect } from 'react';
import { apiClient, Notification, NotificationProvider } from '@/lib/api';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';

interface NotificationFormProps {
  notification: Notification | null;
  providers: NotificationProvider[];
  onClose: () => void;
  onSuccess: () => void;
}

export default function NotificationForm({
  notification,
  providers,
  onClose,
  onSuccess,
}: NotificationFormProps) {
  const [name, setName] = useState('');
  const [type, setType] = useState('smtp');
  const [isDefault, setIsDefault] = useState(false);
  const [active, setActive] = useState(true);
  const [config, setConfig] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (notification) {
      setName(notification.name);
      setType(notification.type);
      setIsDefault(notification.is_default);
      setActive(notification.active);
      try {
        const parsedConfig = JSON.parse(notification.config);
        setConfig(parsedConfig);
      } catch (e) {
        console.error('Failed to parse notification config:', e);
        setConfig({});
      }
    }
  }, [notification]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const data = {
        name,
        type,
        config,
        is_default: isDefault,
        active,
      };

      if (notification) {
        await apiClient.updateNotification(notification.id, data);
      } else {
        await apiClient.createNotification(data);
      }

      onSuccess();
    } catch (err: any) {
      console.error('Failed to save notification:', err);
      setError(err.message || 'Failed to save notification');
    } finally {
      setLoading(false);
    }
  }

  function updateConfig(key: string, value: any) {
    setConfig({ ...config, [key]: value });
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent className="sm:max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {notification ? 'Edit Notification' : 'Add Notification'}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="notification-name">Name</Label>
            <Input
              id="notification-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="notification-type">Type</Label>
            <select
              id="notification-type"
              value={type}
              onChange={(e) => {
                setType(e.target.value);
                setConfig({});
              }}
              className="flex h-8 w-full rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 md:text-sm dark:bg-input/30"
              required
            >
              {providers.map((provider) => (
                <option key={provider.name} value={provider.name}>
                  {provider.label}
                </option>
              ))}
            </select>
          </div>

          {renderProviderConfig(type, config, updateConfig)}

          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2">
              <Switch
                checked={isDefault}
                onCheckedChange={setIsDefault}
              />
              <Label className="cursor-pointer">Default notification</Label>
            </div>

            <div className="flex items-center gap-2">
              <Switch
                checked={active}
                onCheckedChange={setActive}
              />
              <Label className="cursor-pointer">Active</Label>
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t">
            <Button
              type="button"
              variant="outline"
              onClick={onClose}
              disabled={loading}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={loading}
            >
              {loading ? 'Saving...' : notification ? 'Update' : 'Create'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function renderProviderConfig(
  type: string,
  config: Record<string, any>,
  updateConfig: (key: string, value: any) => void
) {
  switch (type) {
    case 'smtp':
      return (
        <>
          <div className="space-y-2">
            <Label htmlFor="smtp-host">SMTP Host</Label>
            <Input
              id="smtp-host"
              type="text"
              value={config.smtp_host || ''}
              onChange={(e) => updateConfig('smtp_host', e.target.value)}
              placeholder="smtp.gmail.com"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="smtp-port">SMTP Port</Label>
            <Input
              id="smtp-port"
              type="number"
              value={config.smtp_port || 587}
              onChange={(e) => updateConfig('smtp_port', parseInt(e.target.value))}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="from-email">From Email</Label>
            <Input
              id="from-email"
              type="email"
              value={config.from_email || ''}
              onChange={(e) => updateConfig('from_email', e.target.value)}
              placeholder="alerts@example.com"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="to-email">To Email</Label>
            <Input
              id="to-email"
              type="email"
              value={config.to_email || ''}
              onChange={(e) => updateConfig('to_email', e.target.value)}
              placeholder="you@example.com"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="smtp-username">Username (optional)</Label>
            <Input
              id="smtp-username"
              type="text"
              value={config.smtp_username || ''}
              onChange={(e) => updateConfig('smtp_username', e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="smtp-password">Password (optional)</Label>
            <Input
              id="smtp-password"
              type="password"
              value={config.smtp_password || ''}
              onChange={(e) => updateConfig('smtp_password', e.target.value)}
            />
          </div>
        </>
      );

    case 'webhook':
      return (
        <>
          <div className="space-y-2">
            <Label htmlFor="webhook-url">Webhook URL</Label>
            <Input
              id="webhook-url"
              type="url"
              value={config.url || ''}
              onChange={(e) => updateConfig('url', e.target.value)}
              placeholder="https://example.com/webhook"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="webhook-method">HTTP Method</Label>
            <select
              id="webhook-method"
              value={config.method || 'POST'}
              onChange={(e) => updateConfig('method', e.target.value)}
              className="flex h-8 w-full rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 md:text-sm dark:bg-input/30"
            >
              <option value="POST">POST</option>
              <option value="GET">GET</option>
              <option value="PUT">PUT</option>
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="webhook-headers">Headers (JSON, optional)</Label>
            <textarea
              id="webhook-headers"
              value={config.headers ? JSON.stringify(config.headers, null, 2) : ''}
              onChange={(e) => {
                try {
                  const headers = e.target.value ? JSON.parse(e.target.value) : {};
                  updateConfig('headers', headers);
                } catch (err) {
                  // Invalid JSON, ignore
                }
              }}
              placeholder='{"Authorization": "Bearer token"}'
              className="flex field-sizing-content min-h-16 w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 md:text-sm dark:bg-input/30 font-mono"
              rows={3}
            />
          </div>
        </>
      );

    case 'discord':
      return (
        <div className="space-y-2">
          <Label htmlFor="discord-url">Discord Webhook URL</Label>
          <Input
            id="discord-url"
            type="url"
            value={config.webhook_url || ''}
            onChange={(e) => updateConfig('webhook_url', e.target.value)}
            placeholder="https://discord.com/api/webhooks/..."
            required
          />
        </div>
      );

    case 'slack':
      return (
        <>
          <div className="space-y-2">
            <Label htmlFor="slack-url">Slack Webhook URL</Label>
            <Input
              id="slack-url"
              type="url"
              value={config.webhook_url || ''}
              onChange={(e) => updateConfig('webhook_url', e.target.value)}
              placeholder="https://hooks.slack.com/services/..."
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="slack-channel">Channel (optional, overrides webhook default)</Label>
            <Input
              id="slack-channel"
              type="text"
              value={config.channel || ''}
              onChange={(e) => updateConfig('channel', e.target.value)}
              placeholder="#alerts"
            />
          </div>
        </>
      );

    case 'telegram':
      return (
        <>
          <div className="space-y-2">
            <Label htmlFor="telegram-token">Bot Token</Label>
            <Input
              id="telegram-token"
              type="text"
              value={config.bot_token || ''}
              onChange={(e) => updateConfig('bot_token', e.target.value)}
              placeholder="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="telegram-chat-id">Chat ID</Label>
            <Input
              id="telegram-chat-id"
              type="text"
              value={config.chat_id || ''}
              onChange={(e) => updateConfig('chat_id', e.target.value)}
              placeholder="-1001234567890"
              required
            />
          </div>
        </>
      );

    case 'teams':
      return (
        <div className="space-y-2">
          <Label htmlFor="teams-url">Microsoft Teams Webhook URL</Label>
          <Input
            id="teams-url"
            type="url"
            value={config.webhook_url || ''}
            onChange={(e) => updateConfig('webhook_url', e.target.value)}
            placeholder="https://outlook.office.com/webhook/..."
            required
          />
        </div>
      );

    case 'pagerduty':
      return (
        <div className="space-y-2">
          <Label htmlFor="pagerduty-key">Integration Key</Label>
          <Input
            id="pagerduty-key"
            type="text"
            value={config.integration_key || ''}
            onChange={(e) => updateConfig('integration_key', e.target.value)}
            placeholder="Integration key from PagerDuty"
            required
          />
        </div>
      );

    case 'pushover':
      return (
        <>
          <div className="space-y-2">
            <Label htmlFor="pushover-user-key">User Key</Label>
            <Input
              id="pushover-user-key"
              type="text"
              value={config.user_key || ''}
              onChange={(e) => updateConfig('user_key', e.target.value)}
              placeholder="Your Pushover user key"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="pushover-api-token">API Token</Label>
            <Input
              id="pushover-api-token"
              type="text"
              value={config.api_token || ''}
              onChange={(e) => updateConfig('api_token', e.target.value)}
              placeholder="Your Pushover API token"
              required
            />
          </div>
        </>
      );

    case 'gotify':
      return (
        <>
          <div className="space-y-2">
            <Label htmlFor="gotify-url">Server URL</Label>
            <Input
              id="gotify-url"
              type="url"
              value={config.server_url || ''}
              onChange={(e) => updateConfig('server_url', e.target.value)}
              placeholder="https://gotify.example.com"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="gotify-token">App Token</Label>
            <Input
              id="gotify-token"
              type="text"
              value={config.app_token || ''}
              onChange={(e) => updateConfig('app_token', e.target.value)}
              placeholder="Application token from Gotify"
              required
            />
          </div>
        </>
      );

    case 'ntfy':
      return (
        <>
          <div className="space-y-2">
            <Label htmlFor="ntfy-url">Server URL (leave empty for ntfy.sh)</Label>
            <Input
              id="ntfy-url"
              type="url"
              value={config.server_url || ''}
              onChange={(e) => updateConfig('server_url', e.target.value)}
              placeholder="https://ntfy.sh or your own server"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="ntfy-topic">Topic</Label>
            <Input
              id="ntfy-topic"
              type="text"
              value={config.topic || ''}
              onChange={(e) => updateConfig('topic', e.target.value)}
              placeholder="my-uptime-alerts"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="ntfy-username">Username (optional, for authentication)</Label>
            <Input
              id="ntfy-username"
              type="text"
              value={config.username || ''}
              onChange={(e) => updateConfig('username', e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="ntfy-password">Password (optional, for authentication)</Label>
            <Input
              id="ntfy-password"
              type="password"
              value={config.password || ''}
              onChange={(e) => updateConfig('password', e.target.value)}
            />
          </div>
        </>
      );

    default:
      return (
        <div className="text-muted-foreground text-sm">
          No additional configuration required for this provider.
        </div>
      );
  }
}
