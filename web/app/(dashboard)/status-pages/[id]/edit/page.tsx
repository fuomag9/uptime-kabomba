'use client';

import { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { apiClient, Monitor, StatusPageWithMonitors } from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';

export default function EditStatusPagePage() {
  const router = useRouter();
  const params = useParams();
  const pageId = parseInt(params.id as string);

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
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, [pageId]);

  async function loadData() {
    try {
      setLoading(true);
      const [statusPageData, monitorList] = await Promise.all([
        apiClient.getStatusPage(pageId),
        apiClient.getMonitors(),
      ]);

      setSlug(statusPageData.slug);
      setTitle(statusPageData.title);
      setDescription(statusPageData.description || '');
      setPublished(statusPageData.published);
      setShowPoweredBy(statusPageData.show_powered_by);
      setTheme(statusPageData.theme || 'light');
      setCustomCss(statusPageData.custom_css || '');
      setSelectedMonitorIds(statusPageData.monitors?.map(m => m.id) || []);

      setMonitors(monitorList);
    } catch (err: any) {
      console.error('Failed to load status page:', err);
      setError(err.message || 'Failed to load status page');
      if (err.status === 404) {
        router.push('/status-pages');
      }
    } finally {
      setLoading(false);
    }
  }

  function toggleMonitor(monitorId: number) {
    if (selectedMonitorIds.includes(monitorId)) {
      setSelectedMonitorIds(selectedMonitorIds.filter((id) => id !== monitorId));
    } else {
      setSelectedMonitorIds([...selectedMonitorIds, monitorId]);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
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

      await apiClient.updateStatusPage(pageId, data);
      router.push('/status-pages');
    } catch (err: any) {
      console.error('Failed to update status page:', err);
      setError(err.message || 'Failed to update status page');
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto space-y-6">
        <div>
          <Skeleton className="h-8 w-48" />
          <Skeleton className="mt-2 h-4 w-72" />
        </div>
        <Skeleton className="h-[600px] w-full" />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Edit Status Page</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Update your public status page settings
        </p>
      </div>

      <Card>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <Label htmlFor="title">Title *</Label>
              <Input
                id="title"
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="slug">Slug *</Label>
              <div className="flex items-center">
                <span className="text-sm text-muted-foreground mr-2">/status/</span>
                <Input
                  id="slug"
                  type="text"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value)}
                  pattern="[a-z0-9-]+"
                  className="flex-1 font-mono"
                  required
                />
              </div>
              <p className="text-xs text-muted-foreground">
                Lowercase letters, numbers, and hyphens only
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
              />
            </div>

            <Separator />

            <div className="space-y-2">
              <Label>Monitors</Label>
              <div className="border border-input rounded-lg p-4 max-h-60 overflow-y-auto space-y-2">
                {monitors.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No monitors available</p>
                ) : (
                  monitors.map((monitor) => (
                    <div key={monitor.id} className="flex items-center gap-2">
                      <Checkbox
                        checked={selectedMonitorIds.includes(monitor.id)}
                        onCheckedChange={() => toggleMonitor(monitor.id)}
                      />
                      <Label className="cursor-pointer font-normal">
                        {monitor.name}
                        <span className="ml-2 text-xs text-muted-foreground">({monitor.type})</span>
                      </Label>
                    </div>
                  ))
                )}
              </div>
            </div>

            <Separator />

            <div className="space-y-2">
              <Label htmlFor="theme">Theme</Label>
              <select
                id="theme"
                value={theme}
                onChange={(e) => setTheme(e.target.value)}
                className="flex h-8 w-full rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 md:text-sm dark:bg-input/30"
              >
                <option value="light">Light</option>
                <option value="dark">Dark</option>
              </select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password Protection (optional)</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Leave empty to keep current password or no password"
              />
              <p className="text-xs text-muted-foreground">
                Set a password to restrict access to this status page
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="custom-css">Custom CSS (optional)</Label>
              <Textarea
                id="custom-css"
                value={customCss}
                onChange={(e) => setCustomCss(e.target.value)}
                rows={6}
                className="font-mono text-sm"
                placeholder=".custom-header { color: #333; }"
              />
            </div>

            <Separator />

            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Checkbox
                  checked={published}
                  onCheckedChange={(checked) => setPublished(checked as boolean)}
                />
                <Label className="cursor-pointer font-normal">Publish status page</Label>
              </div>

              <div className="flex items-center gap-2">
                <Checkbox
                  checked={showPoweredBy}
                  onCheckedChange={(checked) => setShowPoweredBy(checked as boolean)}
                />
                <Label className="cursor-pointer font-normal">Show &quot;Powered by Uptime Kabomba&quot; footer</Label>
              </div>
            </div>

            <Separator />

            <div className="flex justify-end gap-3">
              <Button
                type="button"
                variant="outline"
                onClick={() => router.push('/status-pages')}
                disabled={saving}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={saving}
              >
                {saving ? 'Saving...' : 'Save Changes'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
