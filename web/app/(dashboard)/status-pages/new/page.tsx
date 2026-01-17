'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient, Monitor } from '@/lib/api';

export default function NewStatusPagePage() {
  const router = useRouter();
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [slug, setSlug] = useState('');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [published, setPublished] = useState(false);
  const [showPoweredBy, setShowPoweredBy] = useState(true);
  const [theme, setTheme] = useState('light');
  const [customCss, setCustomCss] = useState('');
  const [password, setPassword] = useState('');
  const [selectedMonitorIds, setSelectedMonitorIds] = useState<number[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadMonitors();
  }, []);

  async function loadMonitors() {
    try {
      const monitorList = await apiClient.getMonitors();
      setMonitors(monitorList);
    } catch (err: any) {
      console.error('Failed to load monitors:', err);
    }
  }

  function toggleMonitor(monitorId: number) {
    if (selectedMonitorIds.includes(monitorId)) {
      setSelectedMonitorIds(selectedMonitorIds.filter((id) => id !== monitorId));
    } else {
      setSelectedMonitorIds([...selectedMonitorIds, monitorId]);
    }
  }

  function generateSlugFromTitle(title: string) {
    return title
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '');
  }

  function handleTitleChange(newTitle: string) {
    setTitle(newTitle);
    if (!slug || slug === generateSlugFromTitle(title)) {
      setSlug(generateSlugFromTitle(newTitle));
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const data = {
        slug,
        title,
        description,
        published,
        show_powered_by: showPoweredBy,
        theme,
        custom_css: customCss,
        password: password || undefined,
        monitor_ids: selectedMonitorIds,
      };

      await apiClient.createStatusPage(data);
      router.push('/status-pages');
    } catch (err: any) {
      console.error('Failed to create status page:', err);
      setError(err.message || 'Failed to create status page');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Create Status Page</h1>
        <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
          Create a public status page to display your monitor statuses
        </p>
      </div>

      <form onSubmit={handleSubmit} className="bg-white dark:bg-gray-800 shadow rounded-lg p-6 space-y-6">
        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-200 px-4 py-3 rounded">
            {error}
          </div>
        )}

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Title *
          </label>
          <input
            type="text"
            value={title}
            onChange={(e) => handleTitleChange(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Slug *
          </label>
          <div className="flex items-center">
            <span className="text-sm text-gray-500 dark:text-gray-400 mr-2">/status/</span>
            <input
              type="text"
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              pattern="[a-z0-9-]+"
              className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
              required
            />
          </div>
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Lowercase letters, numbers, and hyphens only
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Description
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={3}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Monitors
          </label>
          <div className="border border-gray-300 dark:border-gray-600 rounded-md p-4 max-h-60 overflow-y-auto space-y-2 bg-white dark:bg-gray-900">
            {monitors.length === 0 ? (
              <p className="text-sm text-gray-500 dark:text-gray-400">No monitors available</p>
            ) : (
              monitors.map((monitor) => (
                <label key={monitor.id} className="flex items-center">
                  <input
                    type="checkbox"
                    checked={selectedMonitorIds.includes(monitor.id)}
                    onChange={() => toggleMonitor(monitor.id)}
                    className="mr-2"
                  />
                  <span className="text-sm text-gray-700 dark:text-gray-200">{monitor.name}</span>
                  <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">({monitor.type})</span>
                </label>
              ))
            )}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Theme
          </label>
          <select
            value={theme}
            onChange={(e) => setTheme(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Password Protection (optional)
          </label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Leave empty for public access"
          />
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Set a password to restrict access to this status page
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Custom CSS (optional)
          </label>
          <textarea
            value={customCss}
            onChange={(e) => setCustomCss(e.target.value)}
            rows={6}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
            placeholder=".custom-header { color: #333; }"
          />
        </div>

        <div className="space-y-2">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={published}
              onChange={(e) => setPublished(e.target.checked)}
              className="mr-2"
            />
            <span className="text-sm text-gray-700 dark:text-gray-200">Publish status page</span>
          </label>

          <label className="flex items-center">
            <input
              type="checkbox"
              checked={showPoweredBy}
              onChange={(e) => setShowPoweredBy(e.target.checked)}
              className="mr-2"
            />
            <span className="text-sm text-gray-700 dark:text-gray-200">Show "Powered by Uptime Kabomba 💣" footer</span>
          </label>
        </div>

        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
          <button
            type="button"
            onClick={() => router.push('/status-pages')}
            className="px-4 py-2 text-gray-700 dark:text-gray-200 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
            disabled={loading}
          >
            Cancel
          </button>
          <button
            type="submit"
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
            disabled={loading}
          >
            {loading ? 'Creating...' : 'Create Status Page'}
          </button>
        </div>
      </form>
    </div>
  );
}
