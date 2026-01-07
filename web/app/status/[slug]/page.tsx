'use client';

import { useState, useEffect } from 'react';
import { useParams } from 'next/navigation';
import { apiClient, PublicStatusPage } from '@/lib/api';
import { ThemeToggle } from '@/components/ThemeToggle';

export default function PublicStatusPageComponent() {
  const params = useParams();
  const slug = params.slug as string;

  const [data, setData] = useState<PublicStatusPage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [password, setPassword] = useState('');
  const [showPasswordPrompt, setShowPasswordPrompt] = useState(false);

  useEffect(() => {
    loadStatusPage();
  }, [slug]);

  async function loadStatusPage(pwd?: string) {
    try {
      setLoading(true);
      setError(null);
      const statusPage = await apiClient.getPublicStatusPage(slug, pwd);
      setData(statusPage);
      setShowPasswordPrompt(false);
    } catch (err: any) {
      console.error('Failed to load status page:', err);
      if (err.status === 401) {
        setShowPasswordPrompt(true);
        setError('This status page is password protected');
      } else {
        setError(err.message || 'Failed to load status page');
      }
    } finally {
      setLoading(false);
    }
  }

  function handlePasswordSubmit(e: React.FormEvent) {
    e.preventDefault();
    loadStatusPage(password);
  }

  function getStatusColor(status: number) {
    switch (status) {
      case 1: return 'bg-green-500';
      case 0: return 'bg-red-500';
      case 3: return 'bg-yellow-500';
      default: return 'bg-gray-500';
    }
  }

  function getStatusText(status: number) {
    switch (status) {
      case 1: return 'Operational';
      case 0: return 'Down';
      case 3: return 'Maintenance';
      default: return 'Unknown';
    }
  }

  function getOverallStatus() {
    if (!data?.monitors.length) return { text: 'No monitors', color: 'text-gray-600' };

    const hasDown = data.monitors.some(m => m.last_heartbeat?.status === 0);
    const hasMaintenance = data.monitors.some(m => m.last_heartbeat?.status === 3);

    if (hasDown) return { text: 'Partial Outage', color: 'text-red-600' };
    if (hasMaintenance) return { text: 'Under Maintenance', color: 'text-yellow-600' };
    return { text: 'All Systems Operational', color: 'text-green-600' };
  }

  function getIncidentStyle(style: string) {
    switch (style) {
      case 'info': return 'bg-blue-50 border-blue-200 text-blue-900';
      case 'warning': return 'bg-yellow-50 border-yellow-200 text-yellow-900';
      case 'danger': return 'bg-red-50 border-red-200 text-red-900';
      case 'success': return 'bg-green-50 border-green-200 text-green-900';
      default: return 'bg-gray-50 border-gray-200 text-gray-900';
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-gray-600">Loading status page...</div>
      </div>
    );
  }

  if (showPasswordPrompt) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="bg-white shadow rounded-lg p-8 max-w-md w-full">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Password Required</h2>
          <p className="text-gray-600 mb-6">This status page is password protected.</p>
          <form onSubmit={handlePasswordSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Password
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              />
            </div>
            {error && error !== 'This status page is password protected' && (
              <div className="text-red-600 text-sm">{error}</div>
            )}
            <button
              type="submit"
              className="w-full px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
            >
              Access Status Page
            </button>
          </form>
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error || 'Status page not found'}
        </div>
      </div>
    );
  }

  const isDark = data.page.theme === 'dark';
  const bgClass = isDark ? 'bg-gray-900' : 'bg-gray-50';
  const textClass = isDark ? 'text-white' : 'text-gray-900';
  const mutedTextClass = isDark ? 'text-gray-400' : 'text-gray-600';
  const cardBgClass = isDark ? 'bg-gray-800' : 'bg-white';
  const borderClass = isDark ? 'border-gray-700' : 'border-gray-200';

  const overallStatus = getOverallStatus();

  return (
    <div className={`min-h-screen ${bgClass} ${textClass}`}>
      <style dangerouslySetInnerHTML={{ __html: data.page.custom_css || '' }} />

      <div className="absolute top-4 right-4 z-10">
        <ThemeToggle />
      </div>

      <div className="max-w-4xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold mb-2">{data.page.title}</h1>
          {data.page.description && (
            <p className={`text-lg ${mutedTextClass}`}>{data.page.description}</p>
          )}
          <div className={`mt-4 text-2xl font-semibold ${overallStatus.color}`}>
            {overallStatus.text}
          </div>
        </div>

        {/* Incidents */}
        {data.incidents && data.incidents.length > 0 && (
          <div className="mb-8 space-y-4">
            <h2 className="text-xl font-semibold mb-4">Incidents</h2>
            {data.incidents.map((incident) => (
              <div
                key={incident.id}
                className={`border rounded-lg p-4 ${getIncidentStyle(incident.style)}`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <h3 className="font-semibold">{incident.title}</h3>
                    <p className="mt-2 text-sm whitespace-pre-wrap">{incident.content}</p>
                    <p className="mt-2 text-xs opacity-75">
                      {new Date(incident.created_at).toLocaleString()}
                    </p>
                  </div>
                  {incident.pin && (
                    <span className="ml-4 text-xs font-medium px-2 py-1 rounded bg-white bg-opacity-50">
                      Pinned
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Monitors */}
        <div className="space-y-4">
          <h2 className="text-xl font-semibold mb-4">Services</h2>
          {data.monitors.length === 0 ? (
            <div className={`${cardBgClass} border ${borderClass} rounded-lg p-8 text-center`}>
              <p className={mutedTextClass}>No monitors configured</p>
            </div>
          ) : (
            data.monitors.map((monitor) => {
              const status = monitor.last_heartbeat?.status ?? 2;
              const statusColor = getStatusColor(status);
              const statusText = getStatusText(status);

              return (
                <div
                  key={monitor.id}
                  className={`${cardBgClass} border ${borderClass} rounded-lg p-4 hover:shadow-md transition-shadow`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3 flex-1">
                      <div className={`w-3 h-3 rounded-full ${statusColor}`} />
                      <div>
                        <h3 className="font-semibold">{monitor.name}</h3>
                        <p className={`text-sm ${mutedTextClass}`}>{monitor.type}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="font-medium">{statusText}</div>
                      {monitor.last_heartbeat && (
                        <div className={`text-sm ${mutedTextClass}`}>
                          {monitor.last_heartbeat.ping}ms
                        </div>
                      )}
                    </div>
                  </div>
                  {monitor.last_heartbeat?.message && status === 0 && (
                    <div className="mt-2 text-sm text-red-600">
                      {monitor.last_heartbeat.message}
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>

        {/* Footer */}
        {data.page.show_powered_by && (
          <div className={`mt-12 text-center text-sm ${mutedTextClass}`}>
            Powered by <a href="https://github.com/fuomag9/uptime-kabomba" className="underline hover:no-underline">Uptime Kabomba</a>
          </div>
        )}

        <div className={`mt-4 text-center text-xs ${mutedTextClass}`}>
          Last updated: {new Date().toLocaleString()}
        </div>
      </div>
    </div>
  );
}
