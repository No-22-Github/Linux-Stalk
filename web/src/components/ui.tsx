import { Button, Chip, Spinner } from '@heroui/react';
import { Link } from 'react-router-dom';

type Tone = 'default' | 'primary' | 'success' | 'warning' | 'danger';

const toneClasses: Record<Tone, string> = {
  default: 'border-[--color-border] bg-[rgba(229,214,192,0.42)] text-[--color-foreground-subtle]',
  primary: 'border-[--color-primary-subtle] bg-[--color-primary-subtle] text-[--color-primary]',
  success: 'border-[--color-success-subtle] bg-[--color-success-subtle] text-[--color-success]',
  warning: 'border-[--color-warning-subtle] bg-[--color-warning-subtle] text-[--color-warning]',
  danger: 'border-[--color-danger-subtle] bg-[--color-danger-subtle] text-[--color-danger]',
};

interface LoadingStateProps {
  message?: string;
  detail?: string;
}

export function LoadingState({ message = '加载中...', detail }: LoadingStateProps) {
  return (
    <StateFrame>
      <div className="flex h-14 w-14 items-center justify-center rounded-2xl border border-[--color-border] bg-[--color-background-muted] shadow-[var(--shadow-sm)]">
        <Spinner size="lg" color="accent" />
      </div>
      <div className="space-y-1 text-center">
        <p className="text-sm font-medium text-[--color-foreground]">{message}</p>
        <p className="text-sm text-[--color-foreground-subtle]">{detail ?? '正在整理最新数据。'}</p>
      </div>
    </StateFrame>
  );
}

interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  action?: React.ReactNode;
}

export function EmptyState({ title, description, icon, action }: EmptyStateProps) {
  return (
    <StateFrame>
      {icon && (
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl border border-[--color-border] bg-[--color-background-muted] text-[--color-foreground-subtle]">
          {icon}
        </div>
      )}
      <div className="space-y-1 text-center">
        <p className="text-sm font-semibold text-[--color-foreground]">{title}</p>
        {description && <p className="text-sm text-[--color-foreground-subtle]">{description}</p>}
      </div>
      {action}
    </StateFrame>
  );
}

interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  action?: React.ReactNode;
}

export function ErrorState({ title = '暂时无法加载', message, onRetry, action }: ErrorStateProps) {
  return (
    <StateFrame>
      <div className="flex h-14 w-14 items-center justify-center rounded-2xl border border-[--color-danger-subtle] bg-[--color-danger-subtle] text-[--color-danger]">
        <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v3.75m0 3.75h.01M4.062 19.5h15.876c1.54 0 2.502-1.667 1.732-3L13.732 4.5c-.77-1.333-2.694-1.333-3.464 0L2.33 16.5c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <div className="space-y-1 text-center">
        <p className="text-sm font-semibold text-[--color-foreground]">{title}</p>
        <p className="max-w-md text-sm text-[--color-foreground-subtle]">{message}</p>
      </div>
      <div className="flex flex-wrap items-center justify-center gap-3">
        {onRetry && (
          <Button className="border-[--color-primary-subtle] text-[--color-primary]" size="sm" variant="outline" onPress={onRetry}>
            重试
          </Button>
        )}
        {action}
      </div>
    </StateFrame>
  );
}

interface PageHeaderProps {
  title: string;
  description?: string;
  breadcrumbs?: Array<{ label: string; href?: string }>;
  actions?: React.ReactNode;
  meta?: React.ReactNode;
}

export function PageHeader({ title, description, breadcrumbs, actions, meta }: PageHeaderProps) {
  return (
    <div className="space-y-4 animate-slide-up">
      {breadcrumbs && breadcrumbs.length > 0 && (
        <nav className="flex flex-wrap items-center gap-2 text-xs text-[--color-foreground-dim]">
          {breadcrumbs.map((crumb, index) => (
            <span key={`${crumb.label}-${index}`} className="flex items-center gap-2">
              {index > 0 && <span>/</span>}
              {crumb.href ? (
                <Link className="transition-colors hover:text-[--color-primary]" to={crumb.href}>
                  {crumb.label}
                </Link>
              ) : (
                <span className="text-[--color-foreground-subtle]">{crumb.label}</span>
              )}
            </span>
          ))}
        </nav>
      )}

      <div className="paper-panel relative flex flex-col gap-4 overflow-hidden rounded-[--radius-lg] p-5 lg:flex-row lg:items-end lg:justify-between">
        <div className="accent-orb -right-8 -top-8 h-28 w-28 bg-[rgba(181,102,59,0.16)]" />
        <div className="accent-orb right-20 top-8 h-20 w-20 bg-[rgba(97,127,157,0.14)]" />
        <div className="relative z-10 space-y-3">
          {meta && <div className="flex flex-wrap gap-2">{meta}</div>}
          <div className="space-y-1">
            <h1 className="text-2xl font-semibold tracking-tight text-[--color-foreground] lg:text-3xl">{title}</h1>
            {description && <p className="max-w-2xl text-sm text-[--color-foreground-subtle]">{description}</p>}
          </div>
        </div>
        {actions && <div className="relative z-10 flex flex-wrap items-center gap-2">{actions}</div>}
      </div>
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: string | number;
  icon?: React.ReactNode;
  accent?: Tone;
  footnote?: string;
}

export function StatCard({ label, value, icon, accent = 'default', footnote }: StatCardProps) {
  return (
    <div className="paper-panel animate-slide-up rounded-[--radius-lg] p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-[--color-foreground-dim]">{label}</p>
          <p className="font-data text-3xl font-semibold text-[--color-foreground]">
            {typeof value === 'number' ? value.toLocaleString('zh-CN') : value}
          </p>
          {footnote && <p className="text-xs text-[--color-foreground-subtle]">{footnote}</p>}
        </div>
        {icon && (
          <div className={`flex h-11 w-11 items-center justify-center rounded-2xl border ${toneClasses[accent]}`}>
            {icon}
          </div>
        )}
      </div>
    </div>
  );
}

interface SectionProps {
  title: string;
  description?: string;
  children: React.ReactNode;
  className?: string;
  headerAside?: React.ReactNode;
}

export function Section({ title, description, children, className = '', headerAside }: SectionProps) {
  return (
    <section className={`paper-panel rounded-[--radius-lg] ${className}`}>
      <div className="flex flex-col gap-3 border-b border-[--color-border] px-4 py-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-[--color-foreground]">{title}</h2>
          {description && <p className="text-sm text-[--color-foreground-subtle]">{description}</p>}
        </div>
        {headerAside}
      </div>
      <div className="p-4">{children}</div>
    </section>
  );
}

interface DataRowProps {
  label: string;
  value?: string | React.ReactNode;
  mono?: boolean;
}

export function DataRow({ label, value, mono }: DataRowProps) {
  return (
    <div className="flex flex-col gap-1 rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,250,242,0.72)] px-3 py-3 sm:flex-row sm:items-center sm:justify-between">
      <span className="text-xs uppercase tracking-[0.14em] text-[--color-foreground-dim]">{label}</span>
      {typeof value === 'string' ? (
        <span className={`text-sm text-[--color-foreground] ${mono ? 'font-data text-xs sm:text-sm' : ''}`}>{value || '—'}</span>
      ) : (
        value ?? <span className="text-sm text-[--color-foreground-subtle]">—</span>
      )}
    </div>
  );
}

interface StatusBadgeProps {
  label: string;
  tone?: Tone;
}

export function StatusBadge({ label, tone = 'default' }: StatusBadgeProps) {
  return (
    <Chip className={`border ${toneClasses[tone]} text-xs`} size="sm" variant="soft">
      {label}
    </Chip>
  );
}

interface MetricStripProps {
  items: Array<{ label: string; value: string; tone?: Tone }>;
}

export function MetricStrip({ items }: MetricStripProps) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {items.map((item) => (
        <div key={item.label} className="rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,250,242,0.72)] px-3 py-3">
          <div className="flex items-center justify-between gap-3">
            <span className="text-xs uppercase tracking-[0.14em] text-[--color-foreground-dim]">{item.label}</span>
            {item.tone && <StatusBadge label={item.value} tone={item.tone} />}
          </div>
          {!item.tone && <p className="mt-2 font-data text-lg font-semibold text-[--color-foreground]">{item.value}</p>}
        </div>
      ))}
    </div>
  );
}

function StateFrame({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-[60vh] items-center justify-center animate-fade-in">
      <div className="paper-panel flex w-full max-w-xl flex-col items-center gap-5 rounded-[--radius-lg] px-6 py-10 text-center">
        {children}
      </div>
    </div>
  );
}
