"use client";

import { useState } from 'react';
import { CreateMonitorRequest } from '@/lib/api';

interface MonitorFormProps {
  initialData?: Partial<CreateMonitorRequest>;
  onSubmit: (data: CreateMonitorRequest) => void;
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

export default function MonitorForm({ initialData, onSubmit, onCancel, isSubmitting }: MonitorFormProps) {
  const [formData, setFormData] = useState<CreateMonitorRequest>({
    name: initialData?.name || '',
    type: initialData?.type || 'http',
    url: initialData?.url || '',
    interval: initialData?.interval || 60,
    timeout: initialData?.timeout || 30,
    config: initialData?.config || {},
  });

  const [httpConfig, setHttpConfig] = useState({
    method: (initialData?.config?.method as string) || 'GET',
    headers: (initialData?.config?.headers as Record<string, string>) || {},
    body: (initialData?.config?.body as string) || '',
    acceptedStatusCodes: (initialData?.config?.accepted_status_codes as string) || '200-299',
    keyword: (initialData?.config?.keyword as string) || '',
    invertKeyword: (initialData?.config?.invert_keyword as boolean) || false,
    ignoreTLS: (initialData?.config?.ignore_tls as boolean) || false,
    maxRedirects: (initialData?.config?.max_redirects as number) || 10,
  });

  const [headerKey, setHeaderKey] = useState('');
  const [headerValue, setHeaderValue] = useState('');

  // TCP config
  const [tcpConfig, setTcpConfig] = useState({
    port: (initialData?.config?.port as number) || 80,
  });

  // Ping config
  const [pingConfig, setPingConfig] = useState({
    packet_count: (initialData?.config?.packet_count as number) || 4,
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
      if (httpConfig.acceptedStatusCodes !== '200-299') {
        config.accepted_status_codes = httpConfig.acceptedStatusCodes;
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
    } else if (formData.type === 'tcp') {
      config.port = tcpConfig.port;
    } else if (formData.type === 'ping') {
      if (pingConfig.packet_count !== 4) {
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
      ...formData,
      config,
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

        <div>
          <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Monitor Name
          </label>
          <input
            type="text"
            id="name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
            className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
            placeholder="My Website"
          />
        </div>

        <div>
          <label htmlFor="type" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Monitor Type
          </label>
          <select
            id="type"
            value={formData.type}
            onChange={(e) => setFormData({ ...formData, type: e.target.value })}
            required
            className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
          >
            {MONITOR_TYPES.map((type) => (
              <option key={type.value} value={type.value}>
                {type.label}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="url" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {MONITOR_TYPES.find(t => t.value === formData.type)?.urlLabel || 'URL'}
          </label>
          <input
            type="text"
            id="url"
            value={formData.url}
            onChange={(e) => setFormData({ ...formData, url: e.target.value })}
            required
            className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
            placeholder={MONITOR_TYPES.find(t => t.value === formData.type)?.urlPlaceholder || 'Enter URL'}
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label htmlFor="interval" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Check Interval (seconds)
            </label>
            <input
              type="number"
              id="interval"
              value={formData.interval}
              onChange={(e) => setFormData({ ...formData, interval: parseInt(e.target.value) })}
              min={10}
              required
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
            />
          </div>

          <div>
            <label htmlFor="timeout" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Timeout (seconds)
            </label>
            <input
              type="number"
              id="timeout"
              value={formData.timeout}
              onChange={(e) => setFormData({ ...formData, timeout: parseInt(e.target.value) })}
              min={1}
              required
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
            />
          </div>
        </div>
      </div>

      {/* HTTP-specific configuration */}
      {formData.type === 'http' && (
        <div className="space-y-4 border-t border-gray-200 dark:border-gray-700 pt-6">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white">HTTP Configuration</h3>

          <div>
            <label htmlFor="method" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              HTTP Method
            </label>
            <select
              id="method"
              value={httpConfig.method}
              onChange={(e) => setHttpConfig({ ...httpConfig, method: e.target.value })}
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
            >
              {HTTP_METHODS.map((method) => (
                <option key={method} value={method}>
                  {method}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label htmlFor="acceptedStatusCodes" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Accepted Status Codes
            </label>
            <input
              type="text"
              id="acceptedStatusCodes"
              value={httpConfig.acceptedStatusCodes}
              onChange={(e) => setHttpConfig({ ...httpConfig, acceptedStatusCodes: e.target.value })}
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
              placeholder="200-299,301,302"
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Comma-separated list of acceptable status codes or ranges (e.g., 200-299,301)
            </p>
          </div>

          {/* Headers */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Custom Headers
            </label>
            <div className="space-y-2">
              {Object.entries(httpConfig.headers).map(([key, value]) => (
                <div key={key} className="flex items-center gap-2">
                  <span className="flex-1 text-sm text-gray-700 dark:text-gray-300">
                    {key}: {value}
                  </span>
                  <button
                    type="button"
                    onClick={() => removeHeader(key)}
                    className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                  >
                    Remove
                  </button>
                </div>
              ))}
              <div className="flex gap-2">
                <input
                  type="text"
                  value={headerKey}
                  onChange={(e) => setHeaderKey(e.target.value)}
                  placeholder="Header name"
                  className="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
                />
                <input
                  type="text"
                  value={headerValue}
                  onChange={(e) => setHeaderValue(e.target.value)}
                  placeholder="Header value"
                  className="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
                />
                <button
                  type="button"
                  onClick={addHeader}
                  className="px-4 py-2 bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-md hover:bg-gray-300 dark:hover:bg-gray-500"
                >
                  Add
                </button>
              </div>
            </div>
          </div>

          {/* Request Body */}
          {(httpConfig.method === 'POST' || httpConfig.method === 'PUT' || httpConfig.method === 'PATCH') && (
            <div>
              <label htmlFor="body" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                Request Body
              </label>
              <textarea
                id="body"
                value={httpConfig.body}
                onChange={(e) => setHttpConfig({ ...httpConfig, body: e.target.value })}
                rows={4}
                className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
                placeholder="Optional request body (JSON, XML, etc.)"
              />
            </div>
          )}

          {/* Keyword Search */}
          <div>
            <label htmlFor="keyword" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Keyword (optional)
            </label>
            <input
              type="text"
              id="keyword"
              value={httpConfig.keyword}
              onChange={(e) => setHttpConfig({ ...httpConfig, keyword: e.target.value })}
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
              placeholder="Search for this keyword in response"
            />
            <div className="mt-2 flex items-center">
              <input
                type="checkbox"
                id="invertKeyword"
                checked={httpConfig.invertKeyword}
                onChange={(e) => setHttpConfig({ ...httpConfig, invertKeyword: e.target.checked })}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <label htmlFor="invertKeyword" className="ml-2 text-sm text-gray-700 dark:text-gray-300">
                Alert if keyword is NOT found
              </label>
            </div>
          </div>

          {/* Advanced Options */}
          <div className="space-y-2">
            <div className="flex items-center">
              <input
                type="checkbox"
                id="ignoreTLS"
                checked={httpConfig.ignoreTLS}
                onChange={(e) => setHttpConfig({ ...httpConfig, ignoreTLS: e.target.checked })}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <label htmlFor="ignoreTLS" className="ml-2 text-sm text-gray-700 dark:text-gray-300">
                Ignore TLS/SSL errors
              </label>
            </div>

            <div>
              <label htmlFor="maxRedirects" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                Maximum Redirects
              </label>
              <input
                type="number"
                id="maxRedirects"
                value={httpConfig.maxRedirects}
                onChange={(e) => setHttpConfig({ ...httpConfig, maxRedirects: parseInt(e.target.value) })}
                min={0}
                max={20}
                className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
              />
            </div>
          </div>
        </div>
      )}

      {/* TCP-specific configuration */}
      {formData.type === 'tcp' && (
        <div className="space-y-4 border-t border-gray-200 dark:border-gray-700 pt-6">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white">TCP Configuration</h3>

          <div>
            <label htmlFor="port" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Port
            </label>
            <input
              type="number"
              id="port"
              value={tcpConfig.port}
              onChange={(e) => setTcpConfig({ ...tcpConfig, port: parseInt(e.target.value) })}
              min={1}
              max={65535}
              required
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
              placeholder="80"
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Port number to check (1-65535)
            </p>
          </div>
        </div>
      )}

      {/* Ping-specific configuration */}
      {formData.type === 'ping' && (
        <div className="space-y-4 border-t border-gray-200 dark:border-gray-700 pt-6">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white">Ping Configuration</h3>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="packetCount" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                Packet Count
              </label>
              <input
                type="number"
                id="packetCount"
                value={pingConfig.packet_count}
                onChange={(e) => setPingConfig({ ...pingConfig, packet_count: parseInt(e.target.value) })}
                min={1}
                max={100}
                className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
              />
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                Number of packets to send (default: 4)
              </p>
            </div>

            <div>
              <label htmlFor="packetSize" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                Packet Size (bytes)
              </label>
              <input
                type="number"
                id="packetSize"
                value={pingConfig.packet_size}
                onChange={(e) => setPingConfig({ ...pingConfig, packet_size: parseInt(e.target.value) })}
                min={1}
                max={65500}
                className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
              />
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                Size of each packet (default: 56)
              </p>
            </div>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="privileged"
              checked={pingConfig.privileged}
              onChange={(e) => setPingConfig({ ...pingConfig, privileged: e.target.checked })}
              className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
            />
            <label htmlFor="privileged" className="ml-2 text-sm text-gray-700 dark:text-gray-300">
              Use privileged mode (requires root/admin, enables raw ICMP)
            </label>
          </div>
        </div>
      )}

      {/* DNS-specific configuration */}
      {formData.type === 'dns' && (
        <div className="space-y-4 border-t border-gray-200 dark:border-gray-700 pt-6">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white">DNS Configuration</h3>

          <div>
            <label htmlFor="queryType" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Query Type
            </label>
            <select
              id="queryType"
              value={dnsConfig.query_type}
              onChange={(e) => setDnsConfig({ ...dnsConfig, query_type: e.target.value })}
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
            >
              <option value="A">A (IPv4 Address)</option>
              <option value="AAAA">AAAA (IPv6 Address)</option>
              <option value="CNAME">CNAME (Canonical Name)</option>
              <option value="MX">MX (Mail Exchange)</option>
              <option value="NS">NS (Name Server)</option>
              <option value="TXT">TXT (Text Record)</option>
            </select>
          </div>

          <div>
            <label htmlFor="dnsServer" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              DNS Server (optional)
            </label>
            <input
              type="text"
              id="dnsServer"
              value={dnsConfig.dns_server}
              onChange={(e) => setDnsConfig({ ...dnsConfig, dns_server: e.target.value })}
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
              placeholder="8.8.8.8 or dns.google.com"
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Custom DNS server to use (leave empty for system default)
            </p>
          </div>

          <div>
            <label htmlFor="expectedResult" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Expected Result (optional)
            </label>
            <input
              type="text"
              id="expectedResult"
              value={dnsConfig.expected_result}
              onChange={(e) => setDnsConfig({ ...dnsConfig, expected_result: e.target.value })}
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
              placeholder="Expected value in DNS response"
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Alert if this value is not found in the response
            </p>
          </div>
        </div>
      )}

      {/* Docker-specific configuration */}
      {formData.type === 'docker' && (
        <div className="space-y-4 border-t border-gray-200 dark:border-gray-700 pt-6">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white">Docker Configuration</h3>

          <div>
            <label htmlFor="dockerHost" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Docker Host (optional)
            </label>
            <input
              type="text"
              id="dockerHost"
              value={dockerConfig.docker_host}
              onChange={(e) => setDockerConfig({ ...dockerConfig, docker_host: e.target.value })}
              className="mt-1 block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white focus:border-primary focus:ring-primary"
              placeholder="unix:///var/run/docker.sock or tcp://host:2375"
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Docker daemon socket (leave empty for default local socket)
            </p>
          </div>
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex justify-end gap-3 border-t border-gray-200 dark:border-gray-700 pt-6">
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={isSubmitting}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
          >
            Cancel
          </button>
        )}
        <button
          type="submit"
          disabled={isSubmitting}
          className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isSubmitting ? 'Saving...' : 'Save Monitor'}
        </button>
      </div>
    </form>
  );
}
