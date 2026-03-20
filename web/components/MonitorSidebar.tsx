"use client";

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import { useMonitorHeartbeat } from '@/hooks/useMonitorHeartbeats';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useMemo, useState } from 'react';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { buttonVariants } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { Plus, Search } from 'lucide-react';

const STATUS_COLORS = {
  0: { dot: 'bg-red-500', label: 'Down' },
  1: { dot: 'bg-green-500', label: 'Up' },
  2: { dot: 'bg-yellow-500', label: 'Pending' },
  3: { dot: 'bg-blue-500', label: 'Maintenance' },
};

function MonitorSidebarItem({ monitor }: { monitor: any }) {
  const heartbeat = useMonitorHeartbeat(monitor.id);
  const pathname = usePathname();
  const isActive = pathname === `/monitors/${monitor.id}`;

  // Use real-time status if available, otherwise use last_heartbeat from API, otherwise default to pending
  const status = heartbeat?.status ?? monitor.last_heartbeat?.status ?? 2;
  const ping = heartbeat?.ping ?? monitor.last_heartbeat?.ping;
  const statusStyle = STATUS_COLORS[status as keyof typeof STATUS_COLORS];

  return (
    <Link
      href={`/monitors/${monitor.id}`}
      className={cn(
        buttonVariants({ variant: "ghost" }),
        "w-full justify-start px-3 py-2 h-auto font-normal",
        isActive && "bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground",
        !monitor.active && "opacity-70"
      )}
    >
      <div className="flex items-center justify-between w-full">
        <div className="flex items-center min-w-0 flex-1">
          <div className={cn("h-2 w-2 rounded-full mr-2 flex-shrink-0", statusStyle.dot)} />
          <span className="truncate">{monitor.name}</span>
          {!monitor.active && (
            <span className="ml-2 text-xs text-muted-foreground">&#9198;</span>
          )}
        </div>
        {ping !== undefined && (
          <span className={cn(
            "ml-2 text-xs flex-shrink-0",
            isActive ? "text-primary-foreground" : "text-muted-foreground"
          )}>
            {ping}ms
          </span>
        )}
      </div>
    </Link>
  );
}

function SidebarContent({
  query,
  setQuery,
  sortedMonitors,
  isLoading,
  onClose,
}: {
  query: string;
  setQuery: (q: string) => void;
  sortedMonitors: any[];
  isLoading: boolean;
  onClose?: () => void;
}) {
  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="p-4 space-y-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            id="monitor-search"
            type="search"
            placeholder="Search monitors"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className="pl-9"
          />
        </div>
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-foreground">Monitors</h2>
          <Link
            href="/monitors/new"
            title="Add Monitor"
            onClick={onClose}
            className={cn(
              buttonVariants({ variant: "ghost", size: "icon" }),
              "h-7 w-7"
            )}
          >
            <Plus className="h-4 w-4 text-primary" />
          </Link>
        </div>
      </div>

      <ScrollArea className="flex-1 px-4 pb-4">
        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-full" />
          </div>
        ) : sortedMonitors.length > 0 ? (
          <div className="space-y-1">
            {sortedMonitors.map((monitor) => (
              <div key={monitor.id} onClick={onClose}>
                <MonitorSidebarItem monitor={monitor} />
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8">
            <p className="text-sm text-muted-foreground">No monitors yet</p>
            <Link
              href="/monitors/new"
              className="mt-2 inline-block text-sm text-primary hover:underline"
              onClick={onClose}
            >
              Add your first monitor
            </Link>
          </div>
        )}
      </ScrollArea>
    </div>
  );
}

export function MonitorSidebar({
  isOpen,
  onClose
}: {
  isOpen?: boolean;
  onClose?: () => void;
}) {
  const { data: monitors = [], isLoading } = useQuery({
    queryKey: ['monitors'],
    queryFn: () => apiClient.getMonitors(),
    refetchInterval: 30000,
  });
  const [query, setQuery] = useState('');
  const pathname = usePathname();

  const sortedMonitors = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    const filtered = normalizedQuery.length
      ? monitors.filter((monitor) =>
          monitor.name.toLowerCase().includes(normalizedQuery)
        )
      : monitors;
    return [...filtered].sort((a, b) => a.name.localeCompare(b.name));
  }, [monitors, query]);

  const mobileNavLinks = [
    { href: '/notifications', label: 'Notifications', isActive: pathname === '/notifications' },
    { href: '/status-pages', label: 'Status Pages', isActive: pathname === '/status-pages' },
    { href: '/settings', label: 'Settings', isActive: pathname === '/settings' },
  ];

  return (
    <>
      {/* Mobile Sheet */}
      <Sheet open={isOpen} onOpenChange={(open) => { if (!open) onClose?.(); }}>
        <SheetContent side="left" className="w-64 p-0">
          <SheetHeader className="p-4 pb-0">
            <SheetTitle>Monitors</SheetTitle>
          </SheetHeader>
          <SidebarContent
            query={query}
            setQuery={setQuery}
            sortedMonitors={sortedMonitors}
            isLoading={isLoading}
            onClose={onClose}
          />
          <div className="p-4 border-t border-border">
            <div className="space-y-1">
              {mobileNavLinks.map((link) => (
                <Link
                  key={link.href}
                  href={link.href}
                  onClick={onClose}
                  className={cn(
                    buttonVariants({ variant: "ghost" }),
                    "w-full justify-start",
                    link.isActive && "bg-accent text-accent-foreground"
                  )}
                >
                  {link.label}
                </Link>
              ))}
            </div>
          </div>
        </SheetContent>
      </Sheet>

      {/* Desktop Sidebar */}
      <div className="hidden md:flex md:w-64 md:flex-col md:h-full bg-card border-r border-border">
        <SidebarContent
          query={query}
          setQuery={setQuery}
          sortedMonitors={sortedMonitors}
          isLoading={isLoading}
          onClose={onClose}
        />
      </div>
    </>
  );
}
