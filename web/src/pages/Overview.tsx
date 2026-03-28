import { useEffect, useMemo, useState } from 'react';
import { Button } from '@heroui/react';
import { Link } from 'react-router-dom';

import { ApiError, fetchDevices } from '@/api/client';
import { AdminKeyInput } from '@/components/AdminKeyInput';
import { ErrorState, MetricStrip, PageHeader, Section, StatCard, StatusBadge } from '@/components/ui';
import { LoadingState } from '@/components/ui';
import { useAuth } from '@/contexts/auth';
import { formatCompactNumber, formatDateTime, formatRelativeTime, getActivityMeta } from '@/lib/format';
import type { DeviceRow } from '@/types';

export function Overview() {
  const { adminKey, clearAuth, isAuthenticated } = useAuth();
  const [devices, setDevices] = useState<DeviceRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [referenceTime, setReferenceTime] = useState(0);

  useEffect(() => {
    if (!isAuthenticated) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError('');

    fetchDevices(adminKey)
      .then((rows) => {
        setDevices(rows);
        setReferenceTime(Date.now());
      })
      .catch((err) => {
        if (err instanceof ApiError && err.status === 401) {
          setError('当前保存的访问密钥已失效。');
          return;
        }

        setError(err instanceof Error ? err.message : '设备数据暂时不可用');
      })
      .finally(() => setLoading(false));
  }, [adminKey, isAuthenticated]);

  const stats = useMemo(() => {
    const totalEvents = devices.reduce((sum, device) => sum + device.event_count, 0);
    const sortedByLatest = [...devices].sort(
      (a, b) => new Date(b.latest_event_time).getTime() - new Date(a.latest_event_time).getTime(),
    );
    const sortedByEvents = [...devices].sort((a, b) => b.event_count - a.event_count);

    const liveCount = devices.filter(
      (device) => referenceTime - new Date(device.latest_event_time).getTime() <= 60 * 60 * 1000,
    ).length;
    const activeToday = devices.filter(
      (device) => referenceTime - new Date(device.latest_event_time).getTime() <= 24 * 60 * 60 * 1000,
    ).length;
    const staleCount = devices.length - activeToday;
    const avgEventsPerDevice = devices.length > 0 ? Math.round(totalEvents / devices.length) : 0;

    return {
      activeToday,
      avgEventsPerDevice,
      liveCount,
      staleCount,
      sortedByEvents,
      sortedByLatest,
      totalEvents,
    };
  }, [devices, referenceTime]);

  if (!isAuthenticated) {
    return <AdminKeyInput />;
  }

  if (loading) {
    return <LoadingState message="正在准备总览" detail="正在汇总设备状态与最新动态。" />;
  }

  if (error) {
    return (
      <ErrorState
        action={
          <Button size="sm" variant="outline" onPress={clearAuth}>
            重新输入密钥
          </Button>
        }
        message={error}
        onRetry={() => window.location.reload()}
      />
    );
  }

  const newestDevice = stats.sortedByLatest[0];
  const topDevices = stats.sortedByEvents.slice(0, 5);
  const recentDevices = stats.sortedByLatest.slice(0, 6);

  return (
    <div className="space-y-6">
      <PageHeader
        title="总览"
        description="快速查看整体活跃度、最新上报情况，以及当前最值得关注的设备。"
        actions={
          <Link
            className="inline-flex items-center justify-center rounded-[--radius-md] border border-[--color-primary] bg-[--color-primary] px-4 py-2 text-sm font-medium text-white transition-all hover:-translate-y-0.5 hover:opacity-95"
            to="/devices"
          >
            打开设备列表
          </Link>
        }
        meta={
          <>
            <StatusBadge label={`${devices.length} 台设备`} tone="primary" />
            <StatusBadge label={`近 1 小时在线 ${stats.liveCount} 台`} tone="success" />
            <StatusBadge label={`待关注 ${stats.staleCount} 台`} tone={stats.staleCount > 0 ? 'warning' : 'default'} />
          </>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          accent="primary"
          footnote="当前已接入的设备数量"
          icon={<DeviceIcon />}
          label="设备总数"
          value={devices.length}
        />
        <StatCard
          accent="default"
          footnote={`平均每台设备累计 ${formatCompactNumber(stats.avgEventsPerDevice)} 条事件`}
          icon={<EventsIcon />}
          label="累计事件数"
          value={stats.totalEvents}
        />
        <StatCard
          accent="success"
          footnote="最近 24 小时内有动态的设备"
          icon={<LiveIcon />}
          label="今日活跃设备"
          value={stats.activeToday}
        />
        <StatCard
          accent={stats.staleCount > 0 ? 'warning' : 'success'}
          footnote={newestDevice ? `最近更新于 ${formatRelativeTime(newestDevice.latest_event_time)}` : '暂时没有新动态'}
          icon={<ClockIcon />}
          label="待关注设备"
          value={stats.staleCount}
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
        <Section
          title="设备状态速览"
          description="先看关键指标，再快速定位最近活跃或需要留意的设备。"
        >
          <MetricStrip
            items={[
              { label: '最近更新', value: newestDevice ? formatRelativeTime(newestDevice.latest_event_time) : '暂无数据' },
              { label: '最活跃设备', value: topDevices[0]?.device_id ?? '—' },
              { label: '最高事件量', value: topDevices[0] ? topDevices[0].event_count.toLocaleString('zh-CN') : '0' },
              { label: '待关注占比', value: `${devices.length === 0 ? 0 : Math.round((stats.staleCount / devices.length) * 100)}%` },
            ]}
          />

          <div className="mt-4 grid gap-3 lg:grid-cols-2">
            {recentDevices.map((device) => {
              const activity = getActivityMeta(device.latest_event_time);

              return (
                <Link
                  key={device.device_id}
                  className="rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,250,242,0.78)] p-4 transition-all hover:-translate-y-0.5 hover:border-[--color-border-subtle] hover:bg-[rgba(255,246,238,0.92)]"
                  to={`/devices/${encodeURIComponent(device.device_id)}`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="space-y-2">
                      <p className="font-data text-sm text-[--color-foreground]">{device.device_id}</p>
                      <div className="flex flex-wrap gap-2">
                        <StatusBadge label={activity.label} tone={activity.tone} />
                        <StatusBadge label={`${device.event_count.toLocaleString('zh-CN')} 条事件`} tone="default" />
                      </div>
                    </div>
                    <span className="text-xs uppercase tracking-[0.14em] text-[--color-foreground-dim]">
                      {formatRelativeTime(device.latest_event_time)}
                    </span>
                  </div>
                  <p className="mt-3 text-sm text-[--color-foreground-subtle]">
                    最近更新时间：{formatDateTime(device.latest_event_time)}
                  </p>
                </Link>
              );
            })}
          </div>
        </Section>

        <Section
          title="最活跃的设备"
          description="按事件量查看当前最忙的几台设备。"
          headerAside={
            <Link
              className="inline-flex items-center justify-center rounded-[--radius-md] border border-[--color-border] bg-[--color-background-elevated] px-3 py-2 text-sm text-[--color-foreground-subtle] transition-colors hover:border-[--color-border-subtle] hover:text-[--color-foreground]"
              to="/devices"
            >
              查看全部设备
            </Link>
          }
        >
          <div className="space-y-3">
            {topDevices.length === 0 ? (
                <div className="rounded-[--radius-md] border border-dashed border-[--color-border-subtle] bg-[--color-background-elevated] px-4 py-8 text-center text-sm text-[--color-foreground-subtle]">
                  暂时还没有设备数据。
                </div>
              ) : (
              topDevices.map((device, index) => {
                const share = stats.totalEvents === 0 ? 0 : Math.round((device.event_count / stats.totalEvents) * 100);

                return (
                  <div key={device.device_id} className="rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,250,242,0.74)] p-4">
                    <div className="flex items-start justify-between gap-4">
                      <div className="space-y-1">
                        <p className="text-xs uppercase tracking-[0.14em] text-[--color-foreground-dim]">活跃度第 {index + 1} 名</p>
                        <Link className="font-data text-sm text-[--color-foreground] transition-colors hover:text-[--color-primary]" to={`/devices/${encodeURIComponent(device.device_id)}`}>
                          {device.device_id}
                        </Link>
                      </div>
                      <div className="text-right">
                        <p className="font-data text-lg font-semibold text-[--color-foreground]">{device.event_count.toLocaleString('zh-CN')}</p>
                        <p className="text-xs text-[--color-foreground-subtle]">占全部事件量的 {share}%</p>
                      </div>
                    </div>
                    <div className="mt-3 h-2 rounded-full bg-[rgba(229,214,192,0.55)]">
                      <div
                        className="h-2 rounded-full bg-[--color-primary]"
                        style={{ width: `${Math.max(share, 6)}%` }}
                      />
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </Section>
      </div>
    </div>
  );
}

function DeviceIcon() {
  return (
    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
    </svg>
  );
}

function EventsIcon() {
  return (
    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 13.125C3 12.5037 3.50368 12 4.125 12H8.25c.62132 0 1.125.5037 1.125 1.125V19.5A1.5 1.5 0 017.875 21h-3.75A1.125 1.125 0 013 19.875v-6.75zm5.625-6.75C8.625 5.75368 9.12868 5.25 9.75 5.25h4.5c.6213 0 1.125.50368 1.125 1.125v13.5c0 .6213-.5037 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125V6.375zm7.5 3C16.125 8.75368 16.6287 8.25 17.25 8.25h2.625C20.4963 8.25 21 8.75368 21 9.375v10.5c0 .6213-.5037 1.125-1.125 1.125H17.25a1.125 1.125 0 01-1.125-1.125v-10.5z" />
    </svg>
  );
}

function LiveIcon() {
  return (
    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6v6l4 2.5m5-2.5a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function ClockIcon() {
  return (
    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}
