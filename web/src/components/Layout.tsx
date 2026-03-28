import type { ReactNode } from 'react';

import { Button, Chip } from '@heroui/react';
import { Link, useLocation } from 'react-router-dom';

import { useAuth } from '@/contexts/auth';

const navItems = [
  { path: '/', label: '总览', description: '当前状态与重点变化' },
  { path: '/devices', label: '设备', description: '全部设备与最近动态' },
];

export function Layout({ children }: { children: ReactNode }) {
  const location = useLocation();
  const { isAuthenticated, clearAuth } = useAuth();

  const activeItem = navItems.find((item) => {
    if (item.path === '/') {
      return location.pathname === '/';
    }

    return location.pathname.startsWith(item.path);
  });

  return (
    <div className="grain-shell min-h-screen text-[--color-foreground]">
      <div className="pointer-events-none fixed inset-0 hero-grid opacity-50" />
      <div className="pointer-events-none fixed inset-0 bg-[radial-gradient(circle_at_top_left,rgba(181,102,59,0.12),transparent_28%)]" />
      <div className="pointer-events-none fixed inset-0 bg-[radial-gradient(circle_at_right,rgba(83,138,99,0.08),transparent_22%)]" />

      <div className="relative mx-auto flex min-h-screen w-full max-w-7xl flex-col px-4 py-5 lg:px-8 lg:py-8">
        <header className="glass-panel sticky top-4 z-50 overflow-hidden rounded-[--radius-lg]">
          <div className="accent-orb left-8 top-3 h-24 w-24 bg-[rgba(181,102,59,0.2)]" />
          <div className="accent-orb right-14 top-6 h-20 w-20 bg-[rgba(83,138,99,0.18)]" />
          <div className="flex flex-col gap-4 px-5 py-5 lg:flex-row lg:items-center lg:justify-between lg:px-6">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:gap-8">
              <Link className="relative z-10 flex items-center gap-3" to="/">
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl border border-[--color-border] bg-[--color-primary-subtle] text-[--color-primary] shadow-[var(--shadow-sm)]">
                  <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                  </svg>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-[0.24em] text-[--color-foreground-dim]">Linux Stalk Studio</p>
                  <p className="text-sm font-semibold text-[--color-foreground]">设备观察台</p>
                </div>
              </Link>

              {isAuthenticated && (
                <nav className="relative z-10 flex flex-wrap gap-2">
                  {navItems.map((item) => {
                    const isActive = item.path === '/'
                      ? location.pathname === '/'
                      : location.pathname.startsWith(item.path);

                    return (
                      <NavPill key={item.path} active={isActive} description={item.description} href={item.path}>
                        {item.label}
                      </NavPill>
                    );
                  })}
                </nav>
              )}
            </div>

            {isAuthenticated && (
              <div className="relative z-10 flex flex-col gap-3 sm:flex-row sm:items-center">
                <div className="flex items-center gap-2">
                  <Chip className="border border-[--color-success-subtle] bg-[--color-success-subtle] text-[--color-success]" size="sm" variant="soft">
                    已接入
                  </Chip>
                  {activeItem && (
                    <span className="text-xs tracking-[0.08em] text-[--color-foreground-dim]">
                      {activeItem.description}
                    </span>
                  )}
                </div>
                <Button className="border border-[--color-border] bg-[--color-background-elevated] text-[--color-foreground-subtle] hover:text-[--color-danger]" variant="outline" onPress={clearAuth}>
                  退出访问
                </Button>
              </div>
            )}
          </div>
        </header>

        <main className="flex-1 py-6 lg:py-10">{children}</main>
      </div>
    </div>
  );
}

function NavPill({
  active,
  children,
  description,
  href,
}: {
  active: boolean;
  children: ReactNode;
  description: string;
  href: string;
}) {
  return (
    <Link
      className={`rounded-[--radius-md] border px-4 py-3 transition-all ${active
        ? 'border-[rgba(181,102,59,0.18)] bg-[rgba(181,102,59,0.12)] text-[--color-foreground] shadow-[0_10px_24px_rgba(181,102,59,0.12)]'
        : 'border-[--color-border] bg-[rgba(255,250,242,0.58)] text-[--color-foreground-subtle] hover:-translate-y-0.5 hover:border-[--color-border-subtle] hover:bg-[rgba(255,248,240,0.9)] hover:text-[--color-foreground]'
      }`}
      to={href}
    >
      <div className="text-sm font-medium">{children}</div>
      <div className="text-[11px] text-[--color-foreground-dim]">{description}</div>
    </Link>
  );
}
