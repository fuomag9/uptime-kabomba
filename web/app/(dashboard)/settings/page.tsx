"use client";

import { useState, useEffect } from 'react';
import { apiClient, User, OAuthConfig } from '@/lib/api';

export default function SettingsPage() {
  const [user, setUser] = useState<User | null>(null);
  const [oauthConfig, setOauthConfig] = useState<OAuthConfig | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadData = async () => {
      try {
        const [currentUser, config] = await Promise.all([
          apiClient.getCurrentUser(),
          apiClient.getOAuthConfig(),
        ]);
        setUser(currentUser);
        setOauthConfig(config);
      } catch (err) {
        console.error('Failed to load settings data:', err);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, []);

  const handleLinkOAuth = () => {
    // Redirect to OAuth authorization endpoint
    window.location.href = `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/auth/oauth/authorize`;
  };

  if (loading) {
    return (
      <div className="text-center py-12">
        <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-primary border-r-transparent"></div>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">Loading settings...</p>
      </div>
    );
  }

  const getProviderLabel = (provider?: string) => {
    if (!provider || provider === 'local') return 'Password';
    if (provider === 'oidc') return 'OAuth (OIDC)';
    return provider;
  };

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
        Settings
      </h1>

      <div className="mt-6 space-y-6">
        {/* Account Information */}
        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-white">
              Account Information
            </h3>
            <div className="mt-4 space-y-3">
              <div>
                <label className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Username
                </label>
                <p className="mt-1 text-sm text-gray-900 dark:text-white">
                  {user?.username}
                </p>
              </div>
              {user?.email && (
                <div>
                  <label className="text-sm font-medium text-gray-500 dark:text-gray-400">
                    Email
                  </label>
                  <p className="mt-1 text-sm text-gray-900 dark:text-white">
                    {user.email}
                  </p>
                </div>
              )}
              <div>
                <label className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Authentication Method
                </label>
                <div className="mt-1 flex items-center gap-2">
                  <span className="text-sm text-gray-900 dark:text-white">
                    {getProviderLabel(user?.provider)}
                  </span>
                  {user?.provider && user.provider !== 'local' && (
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">
                      OAuth Enabled
                    </span>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* OAuth Linking */}
        {oauthConfig?.enabled && user?.provider === 'local' && (
          <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
            <div className="px-4 py-5 sm:p-6">
              <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-white">
                Link OAuth Account
              </h3>
              <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                Link your account with OAuth to enable single sign-on authentication.
              </p>
              <div className="mt-5">
                <button
                  type="button"
                  onClick={handleLinkOAuth}
                  className="inline-flex items-center rounded-md bg-blue-600 dark:bg-blue-500 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 dark:hover:bg-blue-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
                >
                  <svg className="h-5 w-5 mr-2" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 0C5.373 0 0 5.373 0 12s5.373 12 12 12 12-5.373 12-12S18.627 0 12 0zm0 22C6.486 22 2 17.514 2 12S6.486 2 12 2s10 4.486 10 10-4.486 10-10 10z"/>
                  </svg>
                  Link OAuth Account
                </button>
              </div>
            </div>
          </div>
        )}

        {/* OAuth Info */}
        {oauthConfig?.enabled && user?.provider !== 'local' && (
          <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
            <div className="px-4 py-5 sm:p-6">
              <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-white">
                OAuth Configuration
              </h3>
              <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                Your account is linked with OAuth. You can sign in using either your password or OAuth.
              </p>
              {oauthConfig.issuer && (
                <div className="mt-4">
                  <label className="text-sm font-medium text-gray-500 dark:text-gray-400">
                    OAuth Provider
                  </label>
                  <p className="mt-1 text-sm text-gray-900 dark:text-white break-all">
                    {oauthConfig.issuer}
                  </p>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
