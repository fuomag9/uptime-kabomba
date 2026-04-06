"use client";

import { useState, useEffect } from 'react';
import { apiClient, User, OAuthConfig, UserSettings } from '@/lib/api';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { LinkIcon, Loader2 } from 'lucide-react';
import { toast } from 'sonner';

export default function SettingsPage() {
  const [user, setUser] = useState<User | null>(null);
  const [oauthConfig, setOauthConfig] = useState<OAuthConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [retentionSettings, setRetentionSettings] = useState<UserSettings | null>(null);
  const [savingRetention, setSavingRetention] = useState(false);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [savingPassword, setSavingPassword] = useState(false);

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

    try {
      const updated = await apiClient.updateUserSettings({
        heartbeat_retention_days: retentionSettings.heartbeat_retention_days,
        hourly_stat_retention_days: retentionSettings.hourly_stat_retention_days,
        daily_stat_retention_days: retentionSettings.daily_stat_retention_days,
      });
      setRetentionSettings(updated);
      toast.success('Retention settings saved successfully!');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to save settings';
      toast.error(message);
    } finally {
      setSavingRetention(false);
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      toast.error('New passwords do not match');
      return;
    }
    if (newPassword.length < 8) {
      toast.error('New password must be at least 8 characters');
      return;
    }
    setSavingPassword(true);
    try {
      await apiClient.changePassword(currentPassword, newPassword);
      toast.success('Password changed successfully!');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to change password';
      toast.error(message);
    } finally {
      setSavingPassword(false);
    }
  };

  const handleLinkOAuth = () => {
    window.location.href = '/api/auth/oauth/authorize';
  };

  if (loading) {
    return (
      <div className="max-w-4xl space-y-6">
        <Skeleton className="h-8 w-32" />
        <div className="space-y-6">
          <Skeleton className="h-48 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
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
      <h1 className="text-2xl font-semibold">
        Settings
      </h1>

      <div className="mt-6 space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Account Information</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <Label className="text-muted-foreground">Username</Label>
              <p className="mt-1 text-sm">{user?.username}</p>
            </div>
            {user?.email && (
              <div>
                <Label className="text-muted-foreground">Email</Label>
                <p className="mt-1 text-sm">{user.email}</p>
              </div>
            )}
            <div>
              <Label className="text-muted-foreground">Authentication Method</Label>
              <div className="mt-1 flex items-center gap-2">
                <span className="text-sm">
                  {getProviderLabel(user?.provider)}
                </span>
                {user?.provider && user.provider !== 'local' && (
                  <Badge variant="secondary">OAuth Enabled</Badge>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Change Password</CardTitle>
            <CardDescription>
              Update your account password.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleChangePassword} className="space-y-4 max-w-sm">
              <div className="space-y-2">
                <Label htmlFor="current-password">Current Password</Label>
                <Input
                  id="current-password"
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-password">New Password</Label>
                <Input
                  id="new-password"
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  required
                  minLength={8}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm-password">Confirm New Password</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  minLength={8}
                />
              </div>
              <Button type="submit" disabled={savingPassword}>
                {savingPassword ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Changing...
                  </>
                ) : (
                  'Change Password'
                )}
              </Button>
            </form>
          </CardContent>
        </Card>

        {oauthConfig?.enabled && user?.provider === 'local' && (
          <Card>
            <CardHeader>
              <CardTitle>Link OAuth Account</CardTitle>
              <CardDescription>
                Link your account with OAuth to enable single sign-on authentication.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button onClick={handleLinkOAuth}>
                <LinkIcon className="mr-2 h-4 w-4" />
                Link OAuth Account
              </Button>
            </CardContent>
          </Card>
        )}

        {oauthConfig?.enabled && user?.provider !== 'local' && (
          <Card>
            <CardHeader>
              <CardTitle>OAuth Configuration</CardTitle>
              <CardDescription>
                Your account is linked with OAuth. You can sign in using either your password or OAuth.
              </CardDescription>
            </CardHeader>
            {oauthConfig.issuer && (
              <CardContent>
                <Label className="text-muted-foreground">OAuth Provider</Label>
                <p className="mt-1 text-sm break-all">{oauthConfig.issuer}</p>
              </CardContent>
            )}
          </Card>
        )}

        <Separator />

        <Card>
          <CardHeader>
            <CardTitle>Data Retention Settings</CardTitle>
            <CardDescription>
              Configure how long your monitoring data is retained before automatic cleanup. Changes take effect on the next cleanup cycle.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {retentionSettings && (
              <form onSubmit={handleSaveRetention} className="space-y-6">
                <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
                  <div className="space-y-2">
                    <Label>Heartbeat Data</Label>
                    <div className="flex items-center gap-2">
                      <Input
                        type="number"
                        min="7"
                        max="365"
                        value={retentionSettings.heartbeat_retention_days}
                        onChange={(e) => setRetentionSettings({
                          ...retentionSettings,
                          heartbeat_retention_days: parseInt(e.target.value) || 90,
                        })}
                        className="w-24"
                      />
                      <span className="text-sm text-muted-foreground">days</span>
                    </div>
                    <p className="text-xs text-muted-foreground">7-365 days</p>
                  </div>

                  <div className="space-y-2">
                    <Label>Hourly Statistics</Label>
                    <div className="flex items-center gap-2">
                      <Input
                        type="number"
                        min="30"
                        max="730"
                        value={retentionSettings.hourly_stat_retention_days}
                        onChange={(e) => setRetentionSettings({
                          ...retentionSettings,
                          hourly_stat_retention_days: parseInt(e.target.value) || 365,
                        })}
                        className="w-24"
                      />
                      <span className="text-sm text-muted-foreground">days</span>
                    </div>
                    <p className="text-xs text-muted-foreground">30-730 days</p>
                  </div>

                  <div className="space-y-2">
                    <Label>Daily Statistics</Label>
                    <div className="flex items-center gap-2">
                      <Input
                        type="number"
                        min="90"
                        max="1825"
                        value={retentionSettings.daily_stat_retention_days}
                        onChange={(e) => setRetentionSettings({
                          ...retentionSettings,
                          daily_stat_retention_days: parseInt(e.target.value) || 730,
                        })}
                        className="w-24"
                      />
                      <span className="text-sm text-muted-foreground">days</span>
                    </div>
                    <p className="text-xs text-muted-foreground">90-1825 days (up to 5 years)</p>
                  </div>
                </div>

                <Button type="submit" disabled={savingRetention}>
                  {savingRetention ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Saving...
                    </>
                  ) : (
                    'Save Retention Settings'
                  )}
                </Button>
              </form>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
