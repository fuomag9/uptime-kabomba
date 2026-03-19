"use client";

import { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { apiClient } from '@/lib/api';
import { ThemeToggle } from '@/components/ThemeToggle';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Separator } from '@/components/ui/separator';
import { Globe, AlertCircle } from 'lucide-react';

export default function LoginContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [oauthEnabled, setOauthEnabled] = useState(false);
  const [showLinkingForm, setShowLinkingForm] = useState(false);
  const [linkingData, setLinkingData] = useState<{
    token: string;
    email: string;
  } | null>(null);

  useEffect(() => {
    // Check if OAuth is enabled
    apiClient.getOAuthConfig().then(config => {
      setOauthEnabled(config.enabled);
    }).catch(err => {
      console.error('Failed to check OAuth config:', err);
    });

    // Check for OAuth callback data in URL
    const action = searchParams.get('action');
    if (action === 'link_required') {
      const token = searchParams.get('linking_token');
      const email = searchParams.get('email');
      const message = searchParams.get('message');

      if (token && email) {
        setShowLinkingForm(true);
        setLinkingData({ token, email });
        if (message) {
          setError(decodeURIComponent(message));
        }
      }
    } else if (action === 'error') {
      const message = searchParams.get('message');
      if (message) {
        setError(decodeURIComponent(message));
      }
    }
  }, [searchParams]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await apiClient.login({ username, password });
      router.push('/monitors');
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  const handleOAuthLogin = () => {
    // Redirect to OAuth authorization endpoint
    // Use relative URL since Next.js rewrites proxy /api/* to backend
    window.location.href = '/api/auth/oauth/authorize';
  };

  const handleLinkAccount = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!linkingData) return;

    setError('');
    setLoading(true);

    try {
      await apiClient.linkOAuthAccount({
        linking_token: linkingData.token,
        password,
      });
      router.push('/monitors');
    } catch (err: any) {
      setError(err.message || 'Account linking failed');
    } finally {
      setLoading(false);
    }
  };

  if (showLinkingForm && linkingData) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4 py-12 sm:px-6 lg:px-8">
        <div className="absolute top-4 right-4">
          <ThemeToggle />
        </div>
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <CardTitle className="text-3xl font-bold tracking-tight">
              Link OAuth Account
            </CardTitle>
            <CardDescription>
              An account with email <strong>{linkingData.email}</strong> already exists.
              Please enter your password to link accounts.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form id="link-form" className="space-y-6" onSubmit={handleLinkAccount}>
              {error && (
                <Alert variant="destructive">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              <div className="space-y-2">
                <Label htmlFor="password" className="sr-only">
                  Password
                </Label>
                <Input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  placeholder="Password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </div>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? 'Linking account...' : 'Link Account'}
              </Button>
            </form>
          </CardContent>
          <CardFooter className="justify-center">
            <Button
              variant="link"
              onClick={() => {
                setShowLinkingForm(false);
                setLinkingData(null);
                router.push('/login');
              }}
            >
              Cancel
            </Button>
          </CardFooter>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 py-12 sm:px-6 lg:px-8">
      <div className="absolute top-4 right-4">
        <ThemeToggle />
      </div>
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-3xl font-bold tracking-tight text-foreground">
            Sign in to Uptime Kabomba 💣
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {oauthEnabled && (
            <div className="space-y-6">
              <Button
                type="button"
                variant="outline"
                className="w-full"
                onClick={handleOAuthLogin}
              >
                <Globe className="h-5 w-5 mr-2" />
                Sign in with OAuth
              </Button>

              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <Separator className="w-full" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">
                    Or continue with
                  </span>
                </div>
              </div>
            </div>
          )}

          <form className="space-y-6" onSubmit={handleSubmit}>
            {error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="username" className="sr-only">
                  Username or Email
                </Label>
                <Input
                  id="username"
                  name="username"
                  type="text"
                  autoComplete="username"
                  required
                  placeholder="Username or Email"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password" className="sr-only">
                  Password
                </Label>
                <Input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  placeholder="Password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </div>
            </div>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? 'Signing in...' : 'Sign in'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
