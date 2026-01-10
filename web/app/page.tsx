'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from "next/link";
import { apiClient } from '@/lib/api';

export default function Home() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [setupComplete, setSetupComplete] = useState(false);

  useEffect(() => {
    const checkSetupStatus = async () => {
      try {
        const status = await apiClient.getSetupStatus();

        if (!status.setupComplete) {
          // Setup not completed, redirect to setup
          router.push('/setup');
          return;
        }

        // Setup is complete, check if user is already logged in
        const token = apiClient.getToken();
        if (token) {
          try {
            await apiClient.getCurrentUser();
            // User is logged in, redirect to monitors
            router.push('/monitors');
            return;
          } catch (err) {
            // Token is invalid, continue to show login page
          }
        }

        // Setup is complete but user is not logged in, redirect to login
        router.push('/login');
      } catch (err) {
        console.error('Error checking setup status:', err);
        setSetupComplete(false);
      } finally {
        setLoading(false);
      }
    };

    checkSetupStatus();
  }, [router]);

  if (loading) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-24">
        <div className="text-center">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent align-[-0.125em] motion-reduce:animate-[spin_1.5s_linear_infinite]" role="status">
            <span className="!absolute !-m-px !h-px !w-px !overflow-hidden !whitespace-nowrap !border-0 !p-0 ![clip:rect(0,0,0,0)]">Loading...</span>
          </div>
        </div>
      </div>
    );
  }

  // This fallback should rarely be seen as we redirect immediately
  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-24">
      <div className="max-w-2xl text-center">
        <h1 className="text-4xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-6xl">
          Uptime Kabomba
        </h1>
        <div className="mt-10 flex items-center justify-center gap-x-6">
          {!setupComplete ? (
            <Link
              href="/setup"
              className="rounded-md bg-primary px-3.5 py-2.5 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-primary/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
            >
              Get Started
            </Link>
          ) : (
            <Link
              href="/login"
              className="rounded-md bg-primary px-3.5 py-2.5 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-primary/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
            >
              Login
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}
