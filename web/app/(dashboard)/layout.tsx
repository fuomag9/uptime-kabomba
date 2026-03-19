"use client";

import { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { apiClient } from '@/lib/api';
import { useWebSocket } from '@/hooks/useWebSocket';
import { ThemeToggle } from '@/components/ThemeToggle';
import { MonitorSidebar } from '@/components/MonitorSidebar';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { Menu, LogOut } from 'lucide-react';

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const { connected } = useWebSocket();

  const [sidebarOpen, setSidebarOpen] = useState(false);

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const token = apiClient.getToken();
        if (!token) {
          router.push('/login');
          return;
        }

        const currentUser = await apiClient.getCurrentUser();
        setUser(currentUser);
      } catch (err) {
        router.push('/login');
      } finally {
        setLoading(false);
      }
    };

    checkAuth();
  }, [router]);

  const handleLogout = async () => {
    try {
      await apiClient.logout();
      router.push('/login');
    } catch (err) {
      console.error('Logout failed:', err);
    }
  };

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <Skeleton className="h-8 w-8 rounded-full" />
          <Skeleton className="h-4 w-24" />
        </div>
      </div>
    );
  }

  const navLinks = [
    { href: '/notifications', label: 'Notifications', isActive: pathname === '/notifications' },
    { href: '/certificates', label: 'Certificates', isActive: pathname === '/certificates' },
    { href: '/status-pages', label: 'Status Pages', isActive: pathname === '/status-pages' || pathname.startsWith('/status-pages/') },
    { href: '/settings', label: 'Settings', isActive: pathname === '/settings' },
  ];

  return (
    <div className="flex h-screen flex-col bg-background">
      <header className="bg-card shadow z-10 flex-shrink-0">
        <div className="px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 justify-between items-center">
            <div className="flex items-center">
              <Button
                variant="ghost"
                size="icon"
                className="md:hidden -ml-2 mr-2"
                onClick={() => setSidebarOpen(true)}
              >
                <Menu className="h-6 w-6" />
                <span className="sr-only">Open sidebar</span>
              </Button>
              <h1 className="text-xl font-bold text-foreground">
                Uptime Kabomba
              </h1>
              {connected && (
                <div className="ml-4 items-center hidden sm:flex" title="Real-time updates active">
                  <div className="h-2 w-2 rounded-full bg-green-500" />
                </div>
              )}
            </div>
            <div className="flex items-center space-x-4">
              <nav className="hidden md:flex items-center space-x-4">
                {navLinks.map((link) => (
                  <Link
                    key={link.href}
                    href={link.href}
                    className={cn(
                      "text-sm font-medium transition-colors",
                      link.isActive
                        ? "text-primary"
                        : "text-muted-foreground hover:text-foreground"
                    )}
                  >
                    {link.label}
                  </Link>
                ))}
              </nav>
              <ThemeToggle />
              <div className="hidden md:flex items-center space-x-4">
                <span className="text-sm text-muted-foreground">
                  {user?.username}
                </span>
                <Button variant="secondary" size="sm" onClick={handleLogout}>
                  <LogOut className="h-4 w-4 mr-2" />
                  Logout
                </Button>
              </div>
              <Button
                variant="ghost"
                size="icon"
                className="md:hidden"
                onClick={handleLogout}
                title="Logout"
              >
                <LogOut className="h-5 w-5" />
              </Button>
            </div>
          </div>
        </div>
      </header>

      <div className="flex flex-1 overflow-hidden">
        <MonitorSidebar isOpen={sidebarOpen} onClose={() => setSidebarOpen(false)} />

        <main className="flex-1 overflow-y-auto w-full">
          <div className="px-4 sm:px-6 lg:px-8 py-8">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
