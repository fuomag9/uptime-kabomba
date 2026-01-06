import { useState, useEffect } from 'react';
import { apiClient, Notification, NotificationProvider } from '@/lib/api';

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
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4">
          <h2 className="text-xl font-bold text-gray-900">
            {notification ? 'Edit Notification' : 'Add Notification'}
          </h2>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Type
            </label>
            <select
              value={type}
              onChange={(e) => {
                setType(e.target.value);
                setConfig({});
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
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

          <div className="flex items-center gap-4">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={isDefault}
                onChange={(e) => setIsDefault(e.target.checked)}
                className="mr-2"
              />
              <span className="text-sm text-gray-700">Default notification</span>
            </label>

            <label className="flex items-center">
              <input
                type="checkbox"
                checked={active}
                onChange={(e) => setActive(e.target.checked)}
                className="mr-2"
              />
              <span className="text-sm text-gray-700">Active</span>
            </label>
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
              disabled={loading}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
              disabled={loading}
            >
              {loading ? 'Saving...' : notification ? 'Update' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
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
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              SMTP Host
            </label>
            <input
              type="text"
              value={config.smtp_host || ''}
              onChange={(e) => updateConfig('smtp_host', e.target.value)}
              placeholder="smtp.gmail.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              SMTP Port
            </label>
            <input
              type="number"
              value={config.smtp_port || 587}
              onChange={(e) => updateConfig('smtp_port', parseInt(e.target.value))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              From Email
            </label>
            <input
              type="email"
              value={config.from_email || ''}
              onChange={(e) => updateConfig('from_email', e.target.value)}
              placeholder="alerts@example.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              To Email
            </label>
            <input
              type="email"
              value={config.to_email || ''}
              onChange={(e) => updateConfig('to_email', e.target.value)}
              placeholder="you@example.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Username (optional)
            </label>
            <input
              type="text"
              value={config.smtp_username || ''}
              onChange={(e) => updateConfig('smtp_username', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Password (optional)
            </label>
            <input
              type="password"
              value={config.smtp_password || ''}
              onChange={(e) => updateConfig('smtp_password', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            />
          </div>
        </>
      );

    case 'webhook':
      return (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Webhook URL
            </label>
            <input
              type="url"
              value={config.url || ''}
              onChange={(e) => updateConfig('url', e.target.value)}
              placeholder="https://example.com/webhook"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              HTTP Method
            </label>
            <select
              value={config.method || 'POST'}
              onChange={(e) => updateConfig('method', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            >
              <option value="POST">POST</option>
              <option value="GET">GET</option>
              <option value="PUT">PUT</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Headers (JSON, optional)
            </label>
            <textarea
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
              className="w-full px-3 py-2 border border-gray-300 rounded-md font-mono text-sm"
              rows={3}
            />
          </div>
        </>
      );

    case 'discord':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Discord Webhook URL
          </label>
          <input
            type="url"
            value={config.webhook_url || ''}
            onChange={(e) => updateConfig('webhook_url', e.target.value)}
            placeholder="https://discord.com/api/webhooks/..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md"
            required
          />
        </div>
      );

    case 'slack':
      return (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Slack Webhook URL
            </label>
            <input
              type="url"
              value={config.webhook_url || ''}
              onChange={(e) => updateConfig('webhook_url', e.target.value)}
              placeholder="https://hooks.slack.com/services/..."
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Channel (optional, overrides webhook default)
            </label>
            <input
              type="text"
              value={config.channel || ''}
              onChange={(e) => updateConfig('channel', e.target.value)}
              placeholder="#alerts"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            />
          </div>
        </>
      );

    case 'telegram':
      return (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Bot Token
            </label>
            <input
              type="text"
              value={config.bot_token || ''}
              onChange={(e) => updateConfig('bot_token', e.target.value)}
              placeholder="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Chat ID
            </label>
            <input
              type="text"
              value={config.chat_id || ''}
              onChange={(e) => updateConfig('chat_id', e.target.value)}
              placeholder="-1001234567890"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
        </>
      );

    case 'teams':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Microsoft Teams Webhook URL
          </label>
          <input
            type="url"
            value={config.webhook_url || ''}
            onChange={(e) => updateConfig('webhook_url', e.target.value)}
            placeholder="https://outlook.office.com/webhook/..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md"
            required
          />
        </div>
      );

    case 'pagerduty':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Integration Key
          </label>
          <input
            type="text"
            value={config.integration_key || ''}
            onChange={(e) => updateConfig('integration_key', e.target.value)}
            placeholder="Integration key from PagerDuty"
            className="w-full px-3 py-2 border border-gray-300 rounded-md"
            required
          />
        </div>
      );

    case 'pushover':
      return (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              User Key
            </label>
            <input
              type="text"
              value={config.user_key || ''}
              onChange={(e) => updateConfig('user_key', e.target.value)}
              placeholder="Your Pushover user key"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              API Token
            </label>
            <input
              type="text"
              value={config.api_token || ''}
              onChange={(e) => updateConfig('api_token', e.target.value)}
              placeholder="Your Pushover API token"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
        </>
      );

    case 'gotify':
      return (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Server URL
            </label>
            <input
              type="url"
              value={config.server_url || ''}
              onChange={(e) => updateConfig('server_url', e.target.value)}
              placeholder="https://gotify.example.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              App Token
            </label>
            <input
              type="text"
              value={config.app_token || ''}
              onChange={(e) => updateConfig('app_token', e.target.value)}
              placeholder="Application token from Gotify"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
        </>
      );

    case 'ntfy':
      return (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Server URL (leave empty for ntfy.sh)
            </label>
            <input
              type="url"
              value={config.server_url || ''}
              onChange={(e) => updateConfig('server_url', e.target.value)}
              placeholder="https://ntfy.sh or your own server"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Topic
            </label>
            <input
              type="text"
              value={config.topic || ''}
              onChange={(e) => updateConfig('topic', e.target.value)}
              placeholder="my-uptime-alerts"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Username (optional, for authentication)
            </label>
            <input
              type="text"
              value={config.username || ''}
              onChange={(e) => updateConfig('username', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Password (optional, for authentication)
            </label>
            <input
              type="password"
              value={config.password || ''}
              onChange={(e) => updateConfig('password', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            />
          </div>
        </>
      );

    default:
      return (
        <div className="text-gray-600 text-sm">
          No additional configuration required for this provider.
        </div>
      );
  }
}
