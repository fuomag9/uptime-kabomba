"use client";

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Skeleton } from '@/components/ui/skeleton';

export default function DashboardPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace('/monitors');
  }, [router]);

  return (
    <div className="flex items-center justify-center h-64">
      <div className="text-center space-y-4">
        <Skeleton className="h-8 w-8 rounded-full mx-auto" />
        <p className="text-sm text-muted-foreground">Redirecting...</p>
      </div>
    </div>
  );
}
