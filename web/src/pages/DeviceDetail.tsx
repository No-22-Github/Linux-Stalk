import { useEffect, useMemo, useState } from 'react';
import { Button, Chip } from '@heroui/react';
import { Link, useParams } from 'react-router-dom';

import { ApiError, fetchEvents, fetchLatestEvent } from '@/api/client';
import { AdminKeyInput } from '@/components/AdminKeyInput';
import { DataRow, EmptyState, ErrorState, PageHeader, Section, StatCard, StatusBadge } from '@/components/ui';
import { LoadingState } from '@/components/ui';
import { useAuth } from '@/contexts/auth';
import { formatDateTime, formatRelativeTime, getActivityMeta } from '@/lib/format';
import type { EventRow, IngestPayload } from '@/types';

type TriggerFilter = 'all' | 'focus' | 'system' | 'other';

export function DeviceDetail() {
  const { deviceId } = useParams<{ deviceId: string }>();
  const { adminKey, clearAuth, isAuthenticated } = useAuth();
  const [events, setEvents] = useState<EventRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [latestState, setLatestState] = useState<IngestPayload | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [triggerFilter, setTriggerFilter] = useState<TriggerFilter>('all');

  useEffect(() => {
    if (!isAuthenticated || !deviceId) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError('');

    Promise.all([
      fetchEvents(adminKey, deviceId, 100),
      fetchLatestEvent(adminKey, deviceId).catch(() => null),
    ])
      .then(([eventsList, latest]) => {
        setEvents(eventsList);
        setLatestState(latest?.payload ?? null);
      })
      .catch((err) => {
        if (err instanceof ApiError && err.status === 401) {
          setError('当前保存的访问密钥已失效。');
          return;
        }

        setError(err instanceof Error ? err.message : '设备详情暂时不可用');
      })
      .finally(() => setLoading(false));
  }, [adminKey, deviceId, isAuthenticated]);

  const derived = useMemo(() => {
    const triggerSummary = events.reduce<Record<string, number>>((acc, event) => {
      const trigger = event.payload.trigger;
      acc[trigger] = (acc[trigger] ?? 0) + 1;
      return acc;
    }, {});

    const topTriggers = Object.entries(triggerSummary)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 3);

    const filteredEvents = events.filter((event) => {
      const query = searchQuery.trim().toLowerCase();
      const trigger = event.payload.trigger.toLowerCase();
      const appName = event.payload.focused_app?.application.name?.toLowerCase() ?? '';
      const objectName = event.payload.focused_app?.object?.name?.toLowerCase() ?? '';

      const matchesQuery = query.length === 0
        || trigger.includes(query)
        || appName.includes(query)
        || objectName.includes(query);

      const triggerKind = classifyTrigger(event.payload.trigger);
      const matchesTrigger = triggerFilter === 'all' || triggerKind === triggerFilter;

      return matchesQuery && matchesTrigger;
    });

    return {
      filteredEvents,
      topTriggers,
    };
  }, [events, searchQuery, triggerFilter]);

  if (!isAuthenticated) {
    return <AdminKeyInput />;
  }

  if (!deviceId) {
    return <ErrorState message="缺少设备 ID，暂时无法查看详情。" />;
  }

  if (loading) {
    return <LoadingState message="正在加载设备详情" detail="正在整理最近状态与历史记录。" />;
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

  const latestActivity = latestState?.event_time ?? events[0]?.payload.event_time;
  const activityMeta = latestActivity ? getActivityMeta(latestActivity) : null;

  return (
    <div className="space-y-6">
      <PageHeader
        title={deviceId}
        description="查看这台设备的最近状态、系统信息和事件记录。"
        breadcrumbs={[{ label: '设备', href: '/devices' }, { label: deviceId }]}
        actions={
          <Link
            className="inline-flex items-center justify-center rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,248,240,0.85)] px-3 py-2 text-sm text-[--color-foreground-subtle] transition-all hover:-translate-y-0.5 hover:border-[--color-border-subtle] hover:text-[--color-foreground]"
            to="/devices"
          >
            返回列表
          </Link>
        }
        meta={
          <>
            {activityMeta && <StatusBadge label={activityMeta.label} tone={activityMeta.tone} />}
            <StatusBadge label={`已载入 ${events.length} 条事件`} tone="primary" />
            {latestState?.system.hostname && <StatusBadge label={latestState.system.hostname} tone="default" />}
          </>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          accent={activityMeta?.tone ?? 'default'}
          footnote={latestActivity ? formatDateTime(latestActivity) : '暂时没有最近记录'}
          icon={<PulseIcon />}
          label="最近活动"
          value={latestActivity ? formatRelativeTime(latestActivity) : '—'}
        />
        <StatCard
          accent="primary"
          footnote="当前窗口内已载入的事件数量"
          icon={<StackIcon />}
          label="历史窗口"
          value={events.length}
        />
        <StatCard
          accent="default"
          footnote="当前记录里出现最多的触发类型"
          icon={<TriggerIcon />}
          label="最高频触发器"
          value={derived.topTriggers[0]?.[0] ?? '—'}
        />
        <StatCard
          accent="warning"
          footnote="最近一次记录中的前台应用"
          icon={<AppIcon />}
          label="当前焦点应用"
          value={latestState?.focused_app?.application.name ?? '—'}
        />
      </div>

      {latestState ? (
        <div className="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
          <div className="space-y-4">
            <Section
              title="最近状态"
              description="最近一次记录中包含的系统与会话信息。"
              headerAside={<StatusBadge label={latestState.trigger} tone="primary" />}
            >
              <div className="grid gap-3 lg:grid-cols-2">
                <DataRow label="主机名" mono value={latestState.system.hostname} />
                <DataRow label="操作系统" value={latestState.system.pretty_os} />
                <DataRow label="内核" mono value={latestState.system.kernel} />
                <DataRow label="系统架构" value={latestState.system.architecture} />
                <DataRow label="桌面环境" value={latestState.system.desktop_session || latestState.system.current_desktop} />
                <DataRow label="会话类型" value={latestState.system.session_type} />
                <DataRow label="显示标识" mono value={latestState.system.display || latestState.system.wayland_display} />
                <DataRow label="采集时间" mono value={formatDateTime(latestState.captured_at)} />
              </div>
            </Section>

            <Section title="焦点应用" description="最近一次记录中的当前活动应用。">
              <div className="grid gap-3 lg:grid-cols-2">
                <DataRow label="应用名称" value={latestState.focused_app?.application.name || '—'} />
                <DataRow label="应用角色" value={latestState.focused_app?.application.role || '—'} />
                <DataRow label="对象名称" value={latestState.focused_app?.object?.name || '—'} />
                <DataRow label="对象角色" value={latestState.focused_app?.object?.role || '—'} />
              </div>
            </Section>

            {latestState.trigger_event && (
              <Section title="触发事件" description="生成这次记录的事件元信息。">
                <div className="grid gap-3 lg:grid-cols-2">
                  <DataRow label="信号" mono value={latestState.trigger_event.signal} />
                  <DataRow label="发送者" mono value={latestState.trigger_event.sender} />
                  <DataRow label="接口" value={latestState.trigger_event.interface} />
                  <DataRow label="成员" value={latestState.trigger_event.member} />
                </div>
              </Section>
            )}
          </div>

          <div className="space-y-4">
            <Section title="连接状态" description="查看网络、电源和相关会话状态。">
              <div className="space-y-3">
                <DataRow label="Wi-Fi" value={latestState.system.wifi_ssid || '未连接 Wi-Fi'} />
                <DataRow label="电源" value={latestState.system.power_summary || '暂无电源状态'} />
              </div>

              <div className="mt-4 space-y-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.16em] text-[--color-foreground-dim]">蓝牙设备</p>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {(latestState.system.bluetooth_devices?.length ?? 0) > 0 ? (
                      latestState.system.bluetooth_devices?.map((device) => (
                        <Chip key={device} className="border border-[--color-info-subtle] bg-[--color-info-subtle] text-[--color-info]" size="sm" variant="soft">
                          {device}
                        </Chip>
                      ))
                    ) : (
                      <p className="text-sm text-[--color-foreground-subtle]">当前没有已连接的蓝牙设备</p>
                    )}
                  </div>
                </div>

                <div>
                  <p className="text-xs uppercase tracking-[0.16em] text-[--color-foreground-dim]">媒体会话</p>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {(latestState.system.media_sessions?.length ?? 0) > 0 ? (
                      latestState.system.media_sessions?.map((session) => (
                        <Chip key={session} className="border border-[--color-warning-subtle] bg-[--color-warning-subtle] text-[--color-warning]" size="sm" variant="soft">
                          {session}
                        </Chip>
                      ))
                    ) : (
                      <p className="text-sm text-[--color-foreground-subtle]">当前没有活动中的媒体会话</p>
                    )}
                  </div>
                </div>
              </div>
            </Section>

            <Section title="触发器分布" description="当前记录里最常见的触发类型。">
              <div className="space-y-3">
                {derived.topTriggers.length === 0 ? (
                  <p className="text-sm text-[--color-foreground-subtle]">这台设备暂时没有可展示的历史记录。</p>
                ) : (
                  derived.topTriggers.map(([trigger, count]) => (
                    <div key={trigger} className="rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,250,242,0.72)] p-3">
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-sm text-[--color-foreground]">{trigger}</span>
                        <span className="font-data text-sm text-[--color-foreground-subtle]">{count.toLocaleString('zh-CN')}</span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </Section>
          </div>
        </div>
      ) : (
        <EmptyState
          title="还没有最近状态"
          description="这台设备暂时没有可展示的最近状态，下方仍会显示已载入的历史记录。"
          icon={<StackIcon />}
        />
      )}

      <Section
        title="事件历史"
        description={`当前命中 ${derived.filteredEvents.length} / ${events.length} 条已载入事件。`}
      >
        <div className="space-y-4">
          <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_240px]">
            <div className="soft-input flex min-h-12 items-center gap-3 rounded-[--radius-md] px-3">
              <SearchIcon />
              <input
                className="min-w-0 flex-1 border-0 bg-transparent text-sm text-[--color-foreground] outline-none placeholder:text-[--color-foreground-dim]"
                placeholder="搜索触发器或焦点应用"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
              {searchQuery && (
                <button
                  className="text-xs text-[--color-foreground-subtle] transition-colors hover:text-[--color-foreground]"
                  type="button"
                  onClick={() => setSearchQuery('')}
                >
                  清空
                </button>
              )}
            </div>

            <select
              className="soft-input min-h-12 rounded-[--radius-md] px-3 text-sm text-[--color-foreground] outline-none"
              value={triggerFilter}
              onChange={(e) => setTriggerFilter(e.target.value as TriggerFilter)}
            >
              <option value="all">全部触发器</option>
              <option value="focus">焦点相关</option>
              <option value="system">系统相关</option>
              <option value="other">其他</option>
            </select>
          </div>

          {derived.filteredEvents.length === 0 ? (
            <EmptyState
              title={events.length === 0 ? '还没有事件记录' : '没有符合条件的事件'}
              description={events.length === 0
                ? '设备开始上报后，这里会自动出现。'
                : '可以试试更换关键词，或切回全部触发器。'}
              icon={<TriggerIcon />}
            />
          ) : (
            <div className="space-y-3">
              {derived.filteredEvents.slice(0, 50).map((event, index) => {
                const triggerKind = classifyTrigger(event.payload.trigger);

                return (
                  <div key={`${event.payload.event_time}-${index}`} className="rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,250,242,0.72)] p-4">
                    <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                      <div className="space-y-2">
                        <div className="flex flex-wrap items-center gap-2">
                          <StatusBadge label={event.payload.trigger} tone="primary" />
                          <StatusBadge label={capitalize(triggerKind)} tone="default" />
                          {event.payload.focused_app?.application.name && (
                            <StatusBadge label={event.payload.focused_app.application.name} tone="warning" />
                          )}
                        </div>
                        <p className="text-sm text-[--color-foreground-subtle]">
                          {event.payload.focused_app?.object?.name || event.payload.focused_app?.application.role || '暂无焦点对象信息'}
                        </p>
                      </div>

                      <div className="text-left lg:text-right">
                        <p className="font-data text-sm text-[--color-foreground]">{formatDateTime(event.payload.event_time)}</p>
                        <p className="text-xs text-[--color-foreground-dim]">写入于 {formatRelativeTime(event.received_at)}</p>
                      </div>
                    </div>
                  </div>
                );
              })}

              {derived.filteredEvents.length > 50 && (
                <div className="rounded-[--radius-md] border border-dashed border-[--color-border-subtle] px-4 py-3 text-center text-sm text-[--color-foreground-subtle]">
                  当前最多展示前 50 条匹配结果。
                </div>
              )}
            </div>
          )}
        </div>
      </Section>
    </div>
  );
}

function classifyTrigger(trigger: string): TriggerFilter {
  const normalized = trigger.toLowerCase();

  if (normalized.includes('focus') || normalized.includes('window')) {
    return 'focus';
  }

  if (normalized.includes('system') || normalized.includes('power') || normalized.includes('wifi') || normalized.includes('bluetooth')) {
    return 'system';
  }

  return 'other';
}

function capitalize(value: string): string {
  const labels: Record<string, string> = {
    focus: '焦点相关',
    system: '系统相关',
    other: '其他',
    all: '全部',
  };

  return labels[value] ?? value;
}

function SearchIcon() {
  return (
    <svg className="h-4 w-4 text-[--color-foreground-subtle]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
    </svg>
  );
}

function PulseIcon() {
  return (
    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3.75 12h3l2.25-6 4.5 12 2.25-6h4.5" />
    </svg>
  );
}

function StackIcon() {
  return (
    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 3l9 4.5-9 4.5-9-4.5L12 3zm0 9l9 4.5-9 4.5-9-4.5L12 12z" />
    </svg>
  );
}

function TriggerIcon() {
  return (
    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13 10V3L4 14h7v7l9-11h-7z" />
    </svg>
  );
}

function AppIcon() {
  return (
    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3.75 5.25A2.25 2.25 0 016 3h12a2.25 2.25 0 012.25 2.25v13.5A2.25 2.25 0 0118 21H6a2.25 2.25 0 01-2.25-2.25V5.25zM9 7.5h6M9 12h6m-6 4.5h3" />
    </svg>
  );
}
