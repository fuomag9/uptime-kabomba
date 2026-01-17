"use client";

import { useState, useEffect } from 'react';
import { apiClient, User, OAuthConfig, UserSettings } from '@/lib/api';

export default function SettingsPage() {
  const [user, setUser] = useState<User | null>(null);
  const [oauthConfig, setOauthConfig] = useState<OAuthConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [retentionSettings, setRetentionSettings] = useState<UserSettings | null>(null);
  const [savingRetention, setSavingRetention] = useState(false);
  const [retentionMessage, setRetentionMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    const loadData = async () => {
      try {
        const [currentUser, config, settings] = await Promise.all([
          apiClient.getCurrentUser(),
          apiClient.getOAuthConfig(),
          apiClient.getUserSettings(),
        ]);
        setUser(currentUser);
        setOauthConfig(config);
        setRetentionSettings(settings);
      } catch (err) {
        console.error('Failed to load settings data:', err);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, []);

  const handleSaveRetention = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!retentionSettings) return;

    setSavingRetention(true);
    setRetentionMessage(null);

    try {
      const updated = await apiClient.updateUserSettings({
        heartbeat_retention_days: retentionSettings.heartbeat_retention_days,
        hourly_stat_retention_days: retentionSettings.hourly_stat_retention_days,
        daily_stat_retention_days: retentionSettings.daily_stat_retention_days,
      });
      setRetentionSettings(updated);
      setRetentionMessage({ type: 'success', text: 'Retention settings saved successfully!' });
      setTimeout(() => setRetentionMessage(null), 5000);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to save settings';
      setRetentionMessage({ type: 'error', text: message });
    } finally {
      setSavingRetention(false);
    }
  };

  const handleLinkOAuth = () => {
    // Redirect to OAuth authorization endpoint (use relative URL)
    window.location.href = '/api/auth/oauth/authorize';
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

        {/* Data Retention Settings */}
        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-white">
              Data Retention Settings
            </h3>
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              Configure how long your monitoring data is retained before automatic cleanup. Changes take effect on the next cleanup cycle.
            </p>

            {retentionSettings && (
              <form onSubmit={handleSaveRetention} className="mt-6 space-y-6">
                <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      Heartbeat Data
                    </label>
                    <div className="mt-1 flex items-center gap-2">
                      <input
                        type="number"
                        min="7"
                        max="365"
                        value={retentionSettings.heartbeat_retention_days}
                        onChange={(e) => setRetentionSettings({
                          ...retentionSettings,
                          heartbeat_retention_days: parseInt(e.target.value) || 90,
                        })}
                        className="block w-24 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white shadow-sm focus:border-blue-500 focus:ring-blue-500"
                      />
                      <span className="text-sm text-gray-500 dark:text-gray-400">days</span>
                    </div>
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      7-365 days
                    </p>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      Hourly Statistics
                    </label>
                    <div className="mt-1 flex items-center gap-2">
                      <input
                        type="number"
                        min="30"
                        max="730"
                        value={retentionSettings.hourly_stat_retention_days}
                        onChange={(e) => setRetentionSettings({
                          ...retentionSettings,
                          hourly_stat_retention_days: parseInt(e.target.value) || 365,
                        })}
                        className="block w-24 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white shadow-sm focus:border-blue-500 focus:ring-blue-500"
                      />
                      <span className="text-sm text-gray-500 dark:text-gray-400">days</span>
                    </div>
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      30-730 days
                    </p>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      Daily Statistics
                    </label>
                    <div className="mt-1 flex items-center gap-2">
                      <input
                        type="number"
                        min="90"
                        max="1825"
                        value={retentionSettings.daily_stat_retention_days}
                        onChange={(e) => setRetentionSettings({
                          ...retentionSettings,
                          daily_stat_retention_days: parseInt(e.target.value) || 730,
                        })}
                        className="block w-24 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-gray-900 dark:text-white shadow-sm focus:border-blue-500 focus:ring-blue-500"
                      />
                      <span className="text-sm text-gray-500 dark:text-gray-400">days</span>
                    </div>
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      90-1825 days (up to 5 years)
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-4">
                  <button
                    type="submit"
                    disabled={savingRetention}
                    className="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    {savingRetention ? (
                      <>
                        <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Saving...
                      </>
                    ) : (
                      'Save Retention Settings'
                    )}
                  </button>

                  {retentionMessage && (
                    <p className={`text-sm ${retentionMessage.type === 'success' ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
                      {retentionMessage.text}
                    </p>
                  )}
                </div>
              </form>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
