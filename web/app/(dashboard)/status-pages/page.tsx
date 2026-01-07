'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient, StatusPage } from '@/lib/api';
import Link from 'next/link';

export default function StatusPagesPage() {
  const router = useRouter();
  const [statusPages, setStatusPages] = useState<StatusPage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadStatusPages();
  }, []);

  async function loadStatusPages() {
    try {
      setLoading(true);
      setError(null);
      const pages = await apiClient.getStatusPages();
      setStatusPages(pages || []);
    } catch (err: any) {
      console.error('Failed to load status pages:', err);
      setError(err.message || 'Failed to load status pages');
      if (err.status === 401) {
        router.push('/login');
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(id: number) {
    if (!confirm('Are you sure you want to delete this status page?')) {
      return;
    }

    try {
      await apiClient.deleteStatusPage(id);
      setStatusPages((statusPages || []).filter((p) => p.id !== id));
    } catch (err: any) {
      console.error('Failed to delete status page:', err);
      alert('Failed to delete status page: ' + (err.message || 'Unknown error'));
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading status pages...</div>
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
          <h1 className="text-2xl font-bold text-gray-900">Status Pages</h1>
          <p className="mt-1 text-sm text-gray-600">
            Create public status pages for your monitors
          </p>
        </div>
        <Link
          href="/status-pages/new"
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors"
        >
          Create Status Page
        </Link>
      </div>

      {(!statusPages || statusPages.length === 0) ? (
        <div className="bg-gray-50 border border-gray-200 rounded-lg p-8 text-center">
          <p className="text-gray-600">No status pages created yet.</p>
          <Link
            href="/status-pages/new"
            className="mt-4 inline-block text-blue-600 hover:text-blue-700 font-medium"
          >
            Create your first status page
          </Link>
        </div>
      ) : (
        <div className="grid gap-4">
          {(statusPages || []).map((page) => (
            <div
              key={page.id}
              className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-md transition-shadow"
            >
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <h3 className="text-lg font-semibold text-gray-900">
                      {page.title}
                    </h3>
                    {page.published ? (
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                        Published
                      </span>
                    ) : (
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                        Draft
                      </span>
                    )}
                  </div>
                  <p className="mt-1 text-sm text-gray-600">
                    {page.description || 'No description'}
                  </p>
                  <div className="mt-2 flex items-center gap-4 text-sm">
                    <span className="text-gray-500">
                      Slug: <span className="font-mono text-gray-700">{page.slug}</span>
                    </span>
                    {page.published && (
                      <a
                        href={`/status/${page.slug}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-blue-600 hover:text-blue-700"
                      >
                        View public page →
                      </a>
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <Link
                    href={`/status-pages/${page.id}/edit`}
                    className="text-gray-600 hover:text-gray-700 px-3 py-1 text-sm font-medium rounded border border-gray-300 hover:bg-gray-50 transition-colors"
                  >
                    Edit
                  </Link>
                  <button
                    onClick={() => handleDelete(page.id)}
                    className="text-red-600 hover:text-red-700 px-3 py-1 text-sm font-medium rounded border border-red-600 hover:bg-red-50 transition-colors"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
