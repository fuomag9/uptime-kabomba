'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient, StatusPage } from '@/lib/api';
import Link from 'next/link';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { toast } from 'sonner';

export default function StatusPagesPage() {
  const router = useRouter();
  const [statusPages, setStatusPages] = useState<StatusPage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);

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
    try {
      await apiClient.deleteStatusPage(id);
      setStatusPages((statusPages || []).filter((p) => p.id !== id));
    } catch (err: any) {
      console.error('Failed to delete status page:', err);
      toast.error('Failed to delete status page: ' + (err.message || 'Unknown error'));
    }
  }

  function handleDeleteRequest(id: number) {
    setDeleteId(id);
  }

  function handleDeleteConfirm() {
    if (deleteId !== null) {
      handleDelete(deleteId);
      setDeleteId(null);
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <Skeleton className="h-8 w-40" />
            <Skeleton className="mt-2 h-4 w-64" />
          </div>
          <Skeleton className="h-8 w-40" />
        </div>
        <div className="grid gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-28 w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">Status Pages</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Create public status pages for your monitors
          </p>
        </div>
        <Button asChild>
          <Link href="/status-pages/new">
            Create Status Page
          </Link>
        </Button>
      </div>

      {(!statusPages || statusPages.length === 0) ? (
        <Card>
          <CardContent className="py-8 text-center">
            <p className="text-muted-foreground">No status pages created yet.</p>
            <Button variant="link" className="mt-4" asChild>
              <Link href="/status-pages/new">
                Create your first status page
              </Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {(statusPages || []).map((page) => (
            <Card key={page.id}>
              <CardContent>
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                  <div className="flex-1">
                    <div className="flex flex-wrap items-center gap-2 sm:gap-3">
                      <h3 className="text-lg font-semibold">
                        {page.title}
                      </h3>
                      {page.published ? (
                        <Badge className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
                          Published
                        </Badge>
                      ) : (
                        <Badge variant="outline">
                          Draft
                        </Badge>
                      )}
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {page.description || 'No description'}
                    </p>
                    <div className="mt-2 flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 text-sm">
                      <span className="text-muted-foreground">
                        Slug: <span className="font-mono text-foreground text-xs break-all">{page.slug}</span>
                      </span>
                      {page.published && (
                        <a
                          href={`/status/${page.slug}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary hover:underline"
                        >
                          View public page &rarr;
                        </a>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-2 pt-2 sm:pt-0 border-t sm:border-t-0 border-border justify-end w-full sm:w-auto">
                    <Button variant="outline" size="sm" asChild>
                      <Link href={`/status-pages/${page.id}/edit`}>
                        Edit
                      </Link>
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => handleDeleteRequest(page.id)}
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <AlertDialog open={deleteId !== null} onOpenChange={(open) => { if (!open) setDeleteId(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this status page? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={handleDeleteConfirm}>
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
