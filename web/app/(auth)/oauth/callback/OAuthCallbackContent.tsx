"use client";

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';

export default function OAuthCallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get('code');
      const state = searchParams.get('state');
      const errorParam = searchParams.get('error');

      // Handle OAuth provider error
      if (errorParam) {
        setError(decodeURIComponent(errorParam));
        setTimeout(() => router.push('/login'), 3000);
        return;
      }

      // Validate required parameters
      if (!code || !state) {
        setError('Invalid OAuth callback - missing code or state');
        setTimeout(() => router.push('/login'), 3000);
        return;
      }

      try {
        // Call backend callback endpoint (use relative URL for Next.js proxy)
        const response = await fetch(
          `/api/auth/oauth/callback?code=${code}&state=${state}`
        );

        if (!response.ok) {
          throw new Error(`OAuth callback failed: ${response.statusText}`);
        }

        const data = await response.json();

        // Handle different response actions
        if (data.action === 'login' || data.action === 'register') {
          // Store token and redirect to monitors
          if (data.token) {
            localStorage.setItem('token', data.token);
            router.push('/monitors');
          } else {
            throw new Error('No token received from server');
          }
        } else if (data.action === 'link_required') {
          // Redirect to login with linking data
          const params = new URLSearchParams({
            action: 'link_required',
            linking_token: data.linking_token || '',
            email: data.email || '',
            message: data.message || 'Account linking required',
          });
          router.push(`/login?${params.toString()}`);
        } else if (data.action === 'error') {
          setError(data.message || 'OAuth authentication failed');
          setTimeout(() => router.push('/login'), 3000);
        } else {
          throw new Error(`Unknown response action: ${data.action}`);
        }
      } catch (err: any) {
        console.error('OAuth callback error:', err);
        setError(err.message || 'OAuth callback processing failed');
        setTimeout(() => router.push('/login'), 3000);
      }
    };

    handleCallback();
  }, [router, searchParams]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="text-center">
        {error ? (
          <div>
            <div className="text-red-600 dark:text-red-400 text-lg font-semibold mb-2">
              {error}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">
              Redirecting to login...
            </div>
          </div>
        ) : (
          <div>
            <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-primary border-r-transparent mb-4"></div>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Processing OAuth login...
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
