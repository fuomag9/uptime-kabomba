"use client";

import { useState, useEffect } from 'react';
import { CreateMonitorRequest, Notification, Certificate, apiClient } from '@/lib/api';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';

interface MonitorFormData {
  monitor: CreateMonitorRequest;
  notificationIds: number[];
  useDefaultNotifications: boolean;
}

interface MonitorFormProps {
  initialData?: Partial<CreateMonitorRequest>;
  monitorId?: number;
  notificationsConfigured?: boolean; // true if monitor has explicit config, false if using defaults
  onSubmit: (data: MonitorFormData) => void;
  onCancel?: () => void;
  isSubmitting?: boolean;
}

const MONITOR_TYPES = [
  { value: 'http', label: 'HTTP(s)', urlLabel: 'URL', urlPlaceholder: 'https://example.com' },
  { value: 'tcp', label: 'TCP Port', urlLabel: 'Host', urlPlaceholder: 'example.com or 192.168.1.1' },
  { value: 'ping', label: 'Ping (ICMP)', urlLabel: 'Host', urlPlaceholder: 'example.com or 192.168.1.1' },
  { value: 'dns', label: 'DNS', urlLabel: 'Hostname', urlPlaceholder: 'example.com' },
  { value: 'docker', label: 'Docker Container', urlLabel: 'Container Name/ID', urlPlaceholder: 'my-container' },
];

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];

function normalizeAcceptedStatusCodes(value: unknown): string {
  if (Array.isArray(value)) {
    const codes = value
      .filter((code) => typeof code === 'number' && Number.isFinite(code))
      .map((code) => String(code));
    if (codes.length > 0) {
      return codes.join(',');
    }
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }
  if (typeof value === 'string' && value.trim().length > 0) {
    return value.trim();
  }
  return '200-299';
}

export default function MonitorForm({ initialData, monitorId, notificationsConfigured, onSubmit, onCancel, isSubmitting }: MonitorFormProps) {
  const [formData, setFormData] = useState<CreateMonitorRequest>({
    name: initialData?.name || '',
    type: initialData?.type || 'http',
    url: initialData?.url || '',
    interval: initialData?.interval || 60,
    timeout: initialData?.timeout || 30,
    resend_interval: initialData?.resend_interval || 1,
    ip_version: initialData?.ip_version || 'auto',
    config: initialData?.config || {},
  });

  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [selectedNotificationIds, setSelectedNotificationIds] = useState<number[]>([]);
  const [loadingNotifications, setLoadingNotifications] = useState(true);
  const [certificates, setCertificates] = useState<Certificate[]>([]);
  // Default to using defaults for new monitors, for existing monitors check the flag
  const [useDefaultNotifications, setUseDefaultNotifications] = useState<boolean>(
    monitorId ? notificationsConfigured === false : true
  );

  const [httpConfig, setHttpConfig] = useState({
    method: (initialData?.config?.method as string) || 'GET',
    headers: (initialData?.config?.headers as Record<string, string>) || {},
    body: (initialData?.config?.body as string) || '',
    acceptedStatusCodes: normalizeAcceptedStatusCodes(initialData?.config?.accepted_status_codes),
    keyword: (initialData?.config?.keyword as string) || '',
    invertKeyword: (initialData?.config?.invert_keyword as boolean) || false,
    ignoreTLS: (initialData?.config?.ignore_tls as boolean) || false,
    maxRedirects: (initialData?.config?.max_redirects as number) || 10,
    certificateId: (initialData?.config?.certificate_id as number) || 0,
  });

  const [headerKey, setHeaderKey] = useState('');
  const [headerValue, setHeaderValue] = useState('');

  // TCP config
  const [tcpConfig, setTcpConfig] = useState({
    port: (initialData?.config?.port as number) || 80,
  });

  // Ping config
  const [pingConfig, setPingConfig] = useState({
    packet_count: (initialData?.config?.packet_count as number) || 1,
    packet_size: (initialData?.config?.packet_size as number) || 56,
    privileged: (initialData?.config?.privileged as boolean) || false,
  });

  // DNS config
  const [dnsConfig, setDnsConfig] = useState({
    query_type: (initialData?.config?.query_type as string) || 'A',
    dns_server: (initialData?.config?.dns_server as string) || '',
    expected_result: (initialData?.config?.expected_result as string) || '',
  });

  // Docker config
  const [dockerConfig, setDockerConfig] = useState({
    docker_host: (initialData?.config?.docker_host as string) || '',
  });

  // Load notifications on mount
  useEffect(() => {
    async function loadData() {
      try {
        const [notifs, certs] = await Promise.all([
          apiClient.getNotifications(),
          apiClient.getCertificates(),
        ]);
        setNotifications(notifs);
        setCertificates(certs);

        if (monitorId) {
          // Editing existing monitor - load its linked notifications
          const linked = await apiClient.getMonitorNotifications(monitorId);
          setSelectedNotificationIds(linked.map(n => n.id));
        } else {
          // Creating new monitor - auto-select default notifications
          const defaultIds = notifs.filter(n => n.is_default).map(n => n.id);
          setSelectedNotificationIds(defaultIds);
        }
      } catch (error) {
        console.error('Failed to load notifications:', error);
      } finally {
        setLoadingNotifications(false);
      }
    }
    loadData();
  }, [monitorId]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const config: Record<string, any> = {};

    if (formData.type === 'http') {
      config.method = httpConfig.method;
      if (Object.keys(httpConfig.headers).length > 0) {
        config.headers = httpConfig.headers;
      }
      if (httpConfig.body) {
        config.body = httpConfig.body;
      }
      const acceptedStatusCodes = httpConfig.acceptedStatusCodes.trim();
      if (acceptedStatusCodes.length > 0) {
        config.accepted_status_codes = acceptedStatusCodes;
      }
      if (httpConfig.keyword) {
        config.keyword = httpConfig.keyword;
        config.invert_keyword = httpConfig.invertKeyword;
      }
      if (httpConfig.ignoreTLS) {
        config.ignore_tls = true;
      }
      if (httpConfig.maxRedirects !== 10) {
        config.max_redirects = httpConfig.maxRedirects;
      }
      if (httpConfig.certificateId) {
        config.certificate_id = httpConfig.certificateId;
      }
    } else if (formData.type === 'tcp') {
      config.port = tcpConfig.port;
    } else if (formData.type === 'ping') {
      if (pingConfig.packet_count !== 1) {
        config.packet_count = pingConfig.packet_count;
      }
      if (pingConfig.packet_size !== 56) {
        config.packet_size = pingConfig.packet_size;
      }
      if (pingConfig.privileged) {
        config.privileged = true;
      }
    } else if (formData.type === 'dns') {
      config.query_type = dnsConfig.query_type;
      if (dnsConfig.dns_server) {
        config.dns_server = dnsConfig.dns_server;
      }
      if (dnsConfig.expected_result) {
        config.expected_result = dnsConfig.expected_result;
      }
    } else if (formData.type === 'docker') {
      if (dockerConfig.docker_host) {
        config.docker_host = dockerConfig.docker_host;
      }
    }

    onSubmit({
      monitor: {
        ...formData,
        config,
      },
      notificationIds: selectedNotificationIds,
      useDefaultNotifications,
    });
  };

  const addHeader = () => {
    if (headerKey && headerValue) {
      setHttpConfig({
        ...httpConfig,
        headers: { ...httpConfig.headers, [headerKey]: headerValue },
      });
      setHeaderKey('');
      setHeaderValue('');
    }
  };

  const removeHeader = (key: string) => {
    const { [key]: _, ...rest } = httpConfig.headers;
    setHttpConfig({ ...httpConfig, headers: rest });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Basic Information */}
      <div className="space-y-4">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white">Basic Information</h3>

        <div className="space-y-2">
          <Label htmlFor="name">
            Monitor Name
          </Label>
          <Input
            type="text"
            id="name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
            placeholder="My Website"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="type">
            Monitor Type
          </Label>
          <select
            id="type"
            value={formData.type}
            onChange={(e) => setFormData({ ...formData, type: e.target.value })}
            required
            className="flex h-8 w-full min-w-0 rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 md:text-sm dark:bg-input/30"
          >
            {MONITOR_TYPES.map((type) => (
              <option key={type.value} value={type.value}>
                {type.label}
              </option>
            ))}
          </select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="url">
            {MONITOR_TYPES.find(t => t.value === formData.type)?.urlLabel || 'URL'}
          </Label>
          <Input
            type="text"
            id="url"
            value={formData.url}
            onChange={(e) => setFormData({ ...formData, url: e.target.value })}
            required
            placeholder={MONITOR_TYPES.find(t => t.value === formData.type)?.urlPlaceholder || 'Enter URL'}
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="interval">
              Check Interval (seconds)
            </Label>
            <Input
              type="number"
              id="interval"
              value={formData.interval}
              onChange={(e) => setFormData({ ...formData, interval: parseInt(e.target.value) })}
              min={1}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="timeout">
              Timeout (seconds)
            </Label>
            <Input
              type="number"
              id="timeout"
              value={formData.timeout}
              onChange={(e) => setFormData({ ...formData, timeout: parseInt(e.target.value) })}
              min={1}
              required
            />
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="resend_interval">
            Resend Notification After X Consecutive Failures
          </Label>
          <Input
            type="number"
            id="resend_interval"
            value={formData.resend_interval}
            onChange={(e) => setFormData({ ...formData, resend_interval: parseInt(e.target.value) })}
            min={1}
            required
          />
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Notification will be sent after {formData.resend_interval || 1} consecutive failure{(formData.resend_interval || 1) > 1 ? 's' : ''},
            then resent every {formData.resend_interval || 1} failure{(formData.resend_interval || 1) > 1 ? 's' : ''} after that.
            Set to 1 to receive notification on every failure.
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="ip_version">
            IP Version
          </Label>
          <select
            id="ip_version"
            value={formData.ip_version}
            onChange={(e) => setFormData({ ...formData, ip_version: e.target.value })}
            className="flex h-8 w-full min-w-0 rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 md:text-sm dark:bg-input/30"
          >
            <option value="auto">Auto (IPv4/IPv6)</option>
            <option value="ipv4">IPv4 Only</option>
            <option value="ipv6">IPv6 Only</option>
          </select>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Choose which IP protocol to use for network connections. Auto will try both IPv4 and IPv6.
          </p>
        </div>
      </div>

      {/* Notification Configuration */}
      <Separator />
      <div className="space-y-4">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white">
          Notifications
        </h3>

        {loadingNotifications ? (
          <div className="text-sm text-gray-500 dark:text-gray-400">Loading notifications...</div>
        ) : (
          <div className="space-y-4">
            {/* Use Default Notifications Toggle */}
            <div className="flex items-start gap-3 p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
              <Checkbox
                id="useDefaults"
                checked={useDefaultNotifications}
                onCheckedChange={(checked) => setUseDefaultNotifications(checked === true)}
                className="mt-1"
              />
              <div className="flex-1">
                <Label htmlFor="useDefaults" className="text-sm font-medium text-blue-900 dark:text-blue-100 cursor-pointer">
                  Use default notifications
                </Label>
                <p className="text-sm text-blue-700 dark:text-blue-300 mt-1">
                  {useDefaultNotifications
                    ? "This monitor will automatically receive all notifications marked as 'Default'. New default notifications will be included automatically."
                    : "Uncheck to manually select specific notifications or to disable notifications completely."}
                </p>
              </div>
            </div>

            {/* Show notification list only when not using defaults */}
            {!useDefaultNotifications && (
              <>
                <div className="flex items-center justify-between">
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Select which notifications to use for this monitor. Unselect all to disable notifications completely.
                  </p>
                  {notifications.length > 0 && (
                    <div className="flex gap-2">
                      <Button
                        type="button"
                        variant="ghost"
                        size="xs"
                        onClick={() => setSelectedNotificationIds(notifications.map(n => n.id))}
                        className="text-blue-600 dark:text-blue-400"
                      >
                        Select all
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="xs"
                        onClick={() => setSelectedNotificationIds([])}
                      >
                        Unselect all
                      </Button>
                    </div>
                  )}
                </div>

                <div className="space-y-2">
                  {notifications.map((notif) => (
                    <label
                      key={notif.id}
                      className="flex items-center p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer transition-colors"
                    >
                      <Checkbox
                        checked={selectedNotificationIds.includes(notif.id)}
                        onCheckedChange={(checked) => {
                          setSelectedNotificationIds(
                            checked
                              ? [...selectedNotificationIds, notif.id]
                              : selectedNotificationIds.filter(id => id !== notif.id)
                          );
                        }}
                      />
                      <div className="ml-3 flex-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-sm font-medium text-gray-900 dark:text-white">
                            {notif.name}
                          </span>
                          <Badge variant="secondary" className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">
                            {getProviderLabel(notif.type)}
                          </Badge>
                          {notif.is_default && (
                            <Badge variant="secondary" className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">
                              Default
                            </Badge>
                          )}
                          {!notif.active && (
                            <Badge variant="secondary">
                              Inactive
                            </Badge>
                          )}
                        </div>
                      </div>
                    </label>
                  ))}

                  {notifications.length === 0 && (
                    <div className="text-sm text-gray-500 dark:text-gray-400 p-4 border border-dashed border-gray-300 dark:border-gray-700 rounded-lg text-center">
                      No notifications configured. Visit the Notifications tab to add one.
                    </div>
                  )}
                </div>
              </>
            )}

            {/* Show current defaults when using defaults mode */}
            {useDefaultNotifications && notifications.filter(n => n.is_default).length > 0 && (
              <div className="space-y-2">
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  Current default notifications that will be used:
                </p>
                <div className="flex flex-wrap gap-2">
                  {notifications.filter(n => n.is_default).map((notif) => (
                    <Badge
                      key={notif.id}
                      variant="secondary"
                      className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 px-3 py-1 h-auto"
                    >
                      {notif.name}
                      <span className="text-xs opacity-70 ml-1">({getProviderLabel(notif.type)})</span>
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {useDefaultNotifications && notifications.filter(n => n.is_default).length === 0 && (
              <div className="text-sm text-amber-700 dark:text-amber-300 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
                No default notifications are configured. This monitor will not receive any notifications until you add default notifications or select specific ones.
              </div>
            )}
          </div>
        )}
      </div>

      {/* HTTP-specific configuration */}
      {formData.type === 'http' && (
        <>
          <Separator />
          <div className="space-y-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">HTTP Configuration</h3>

            <div className="space-y-2">
              <Label htmlFor="method">
                HTTP Method
              </Label>
              <select
                id="method"
                value={httpConfig.method}
                onChange={(e) => setHttpConfig({ ...httpConfig, method: e.target.value })}
                className="flex h-8 w-full min-w-0 rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 md:text-sm dark:bg-input/30"
              >
                {HTTP_METHODS.map((method) => (
                  <option key={method} value={method}>
                    {method}
                  </option>
                ))}
              </select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="acceptedStatusCodes">
                Accepted Status Codes
              </Label>
              <Input
                type="text"
                id="acceptedStatusCodes"
                value={httpConfig.acceptedStatusCodes}
                onChange={(e) => setHttpConfig({ ...httpConfig, acceptedStatusCodes: e.target.value })}
                placeholder="200-299,301,302"
              />
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Comma-separated list of acceptable status codes or ranges (e.g., 200-299,301)
              </p>
            </div>

            {/* Headers */}
            <div className="space-y-2">
              <Label>
                Custom Headers
              </Label>
              <div className="space-y-2">
                {Object.entries(httpConfig.headers).map(([key, value]) => (
                  <div key={key} className="flex items-center gap-2">
                    <span className="flex-1 text-sm text-gray-700 dark:text-gray-300">
                      {key}: {value}
                    </span>
                    <Button
                      type="button"
                      variant="ghost"
                      size="xs"
                      onClick={() => removeHeader(key)}
                      className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                    >
                      Remove
                    </Button>
                  </div>
                ))}
                <div className="flex gap-2">
                  <Input
                    type="text"
                    value={headerKey}
                    onChange={(e) => setHeaderKey(e.target.value)}
                    placeholder="Header name"
                    className="flex-1"
                  />
                  <Input
                    type="text"
                    value={headerValue}
                    onChange={(e) => setHeaderValue(e.target.value)}
                    placeholder="Header value"
                    className="flex-1"
                  />
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={addHeader}
                  >
                    Add
                  </Button>
                </div>
              </div>
            </div>

            {/* Request Body */}
            {(httpConfig.method === 'POST' || httpConfig.method === 'PUT' || httpConfig.method === 'PATCH') && (
              <div className="space-y-2">
                <Label htmlFor="body">
                  Request Body
                </Label>
                <Textarea
                  id="body"
                  value={httpConfig.body}
                  onChange={(e) => setHttpConfig({ ...httpConfig, body: e.target.value })}
                  rows={4}
                  placeholder="Optional request body (JSON, XML, etc.)"
                />
              </div>
            )}

            {/* Keyword Search */}
            <div className="space-y-2">
              <Label htmlFor="keyword">
                Keyword (optional)
              </Label>
              <Input
                type="text"
                id="keyword"
                value={httpConfig.keyword}
                onChange={(e) => setHttpConfig({ ...httpConfig, keyword: e.target.value })}
                placeholder="Search for this keyword in response"
              />
              <div className="mt-2 flex items-center gap-2">
                <Checkbox
                  id="invertKeyword"
                  checked={httpConfig.invertKeyword}
                  onCheckedChange={(checked) => setHttpConfig({ ...httpConfig, invertKeyword: checked === true })}
                />
                <Label htmlFor="invertKeyword" className="font-normal">
                  Alert if keyword is NOT found
                </Label>
              </div>
            </div>

            {/* Advanced Options */}
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Checkbox
                  id="ignoreTLS"
                  checked={httpConfig.ignoreTLS}
                  onCheckedChange={(checked) => setHttpConfig({ ...httpConfig, ignoreTLS: checked === true })}
                />
                <Label htmlFor="ignoreTLS" className="font-normal">
                  Ignore TLS/SSL errors
                </Label>
              </div>

              <div className="space-y-2">
                <Label htmlFor="maxRedirects">
                  Maximum Redirects
                </Label>
                <Input
                  type="number"
                  id="maxRedirects"
                  value={httpConfig.maxRedirects}
                  onChange={(e) => setHttpConfig({ ...httpConfig, maxRedirects: parseInt(e.target.value) })}
                  min={0}
                  max={20}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="certificateId">
                Client Certificate (mTLS)
              </Label>
              <select
                id="certificateId"
                value={httpConfig.certificateId || ''}
                onChange={(e) => setHttpConfig({ ...httpConfig, certificateId: e.target.value ? Number(e.target.value) : 0 })}
                className="flex h-8 w-full min-w-0 rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 md:text-sm dark:bg-input/30"
              >
                <option value="">None</option>
                {certificates.map((c) => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Optional: select a client certificate for mTLS authentication
              </p>
            </div>
          </div>
        </>
      )}

      {/* TCP-specific configuration */}
      {formData.type === 'tcp' && (
        <>
          <Separator />
          <div className="space-y-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">TCP Configuration</h3>

            <div className="space-y-2">
              <Label htmlFor="port">
                Port
              </Label>
              <Input
                type="number"
                id="port"
                value={tcpConfig.port}
                onChange={(e) => setTcpConfig({ ...tcpConfig, port: parseInt(e.target.value) })}
                min={1}
                max={65535}
                required
                placeholder="80"
              />
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Port number to check (1-65535)
              </p>
            </div>
          </div>
        </>
      )}

      {/* Ping-specific configuration */}
      {formData.type === 'ping' && (
        <>
          <Separator />
          <div className="space-y-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">Ping Configuration</h3>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="packetCount">
                  Packet Count
                </Label>
                <Input
                  type="number"
                  id="packetCount"
                  value={pingConfig.packet_count}
                  onChange={(e) => setPingConfig({ ...pingConfig, packet_count: parseInt(e.target.value) })}
                  min={1}
                  max={100}
                />
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Number of packets to send (default: 4)
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="packetSize">
                  Packet Size (bytes)
                </Label>
                <Input
                  type="number"
                  id="packetSize"
                  value={pingConfig.packet_size}
                  onChange={(e) => setPingConfig({ ...pingConfig, packet_size: parseInt(e.target.value) })}
                  min={1}
                  max={65500}
                />
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Size of each packet (default: 56)
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <Checkbox
                id="privileged"
                checked={pingConfig.privileged}
                onCheckedChange={(checked) => setPingConfig({ ...pingConfig, privileged: checked === true })}
              />
              <Label htmlFor="privileged" className="font-normal">
                Use privileged mode (requires root/admin, enables raw ICMP)
              </Label>
            </div>
          </div>
        </>
      )}

      {/* DNS-specific configuration */}
      {formData.type === 'dns' && (
        <>
          <Separator />
          <div className="space-y-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">DNS Configuration</h3>

            <div className="space-y-2">
              <Label htmlFor="queryType">
                Query Type
              </Label>
              <select
                id="queryType"
                value={dnsConfig.query_type}
                onChange={(e) => setDnsConfig({ ...dnsConfig, query_type: e.target.value })}
                className="flex h-8 w-full min-w-0 rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 md:text-sm dark:bg-input/30"
              >
                <option value="A">A (IPv4 Address)</option>
                <option value="AAAA">AAAA (IPv6 Address)</option>
                <option value="CNAME">CNAME (Canonical Name)</option>
                <option value="MX">MX (Mail Exchange)</option>
                <option value="NS">NS (Name Server)</option>
                <option value="TXT">TXT (Text Record)</option>
              </select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="dnsServer">
                DNS Server (optional)
              </Label>
              <Input
                type="text"
                id="dnsServer"
                value={dnsConfig.dns_server}
                onChange={(e) => setDnsConfig({ ...dnsConfig, dns_server: e.target.value })}
                placeholder="8.8.8.8 or dns.google.com"
              />
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Custom DNS server to use (leave empty for system default)
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="expectedResult">
                Expected Result (optional)
              </Label>
              <Input
                type="text"
                id="expectedResult"
                value={dnsConfig.expected_result}
                onChange={(e) => setDnsConfig({ ...dnsConfig, expected_result: e.target.value })}
                placeholder="Expected value in DNS response"
              />
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Alert if this value is not found in the response
              </p>
            </div>
          </div>
        </>
      )}

      {/* Docker-specific configuration */}
      {formData.type === 'docker' && (
        <>
          <Separator />
          <div className="space-y-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">Docker Configuration</h3>

            <div className="space-y-2">
              <Label htmlFor="dockerHost">
                Docker Host (optional)
              </Label>
              <Input
                type="text"
                id="dockerHost"
                value={dockerConfig.docker_host}
                onChange={(e) => setDockerConfig({ ...dockerConfig, docker_host: e.target.value })}
                placeholder="unix:///var/run/docker.sock or tcp://host:2375"
              />
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Docker daemon socket (leave empty for default local socket)
              </p>
            </div>
          </div>
        </>
      )}

      {/* Action Buttons */}
      <Separator />
      <div className="flex justify-end gap-3">
        {onCancel && (
          <Button
            type="button"
            variant="outline"
            onClick={onCancel}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
        )}
        <Button
          type="submit"
          disabled={isSubmitting}
        >
          {isSubmitting ? 'Saving...' : 'Save Monitor'}
        </Button>
      </div>
    </form>
  );
}

function getProviderLabel(type: string): string {
  const labels: Record<string, string> = {
    smtp: 'Email',
    webhook: 'Webhook',
    discord: 'Discord',
    slack: 'Slack',
    telegram: 'Telegram',
    teams: 'Teams',
    pagerduty: 'PagerDuty',
    pushover: 'Pushover',
    gotify: 'Gotify',
    ntfy: 'Ntfy',
  };
  return labels[type] || type.toUpperCase();
}
