import { useEffect, useMemo, useState } from 'react';
import { Button } from '@heroui/react';
import { Link } from 'react-router-dom';

import { ApiError, fetchDevices } from '@/api/client';
import { AdminKeyInput } from '@/components/AdminKeyInput';
import { EmptyState, ErrorState, MetricStrip, PageHeader, Section, StatusBadge } from '@/components/ui';
import { LoadingState } from '@/components/ui';
import { useAuth } from '@/contexts/auth';
import { formatDateTime, formatRelativeTime, getActivityMeta } from '@/lib/format';
import type { DeviceRow } from '@/types';

type SortField = 'device_id' | 'event_count' | 'latest_event_time';
type SortOrder = 'asc' | 'desc';
type ActivityFilter = 'all' | 'live' | 'recent' | 'stale';

const activityFilterLabels: Record<ActivityFilter, string> = {
  all: '全部活跃状态',
  live: '近 1 小时在线',
  recent: '今日活跃',
  stale: '待关注',
};

export function Devices() {
  const { adminKey, clearAuth, isAuthenticated } = useAuth();
  const [devices, setDevices] = useState<DeviceRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [referenceTime, setReferenceTime] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState<SortField>('latest_event_time');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
  const [activityFilter, setActivityFilter] = useState<ActivityFilter>('all');

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

        setError(err instanceof Error ? err.message : '设备列表暂时不可用');
      })
      .finally(() => setLoading(false));
  }, [adminKey, isAuthenticated]);

  const summary = useMemo(() => {
    return {
      live: devices.filter((device) => referenceTime - new Date(device.latest_event_time).getTime() <= 60 * 60 * 1000).length,
      recent: devices.filter((device) => referenceTime - new Date(device.latest_event_time).getTime() <= 24 * 60 * 60 * 1000).length,
      stale: devices.filter((device) => referenceTime - new Date(device.latest_event_time).getTime() > 24 * 60 * 60 * 1000).length,
      totalEvents: devices.reduce((sum, device) => sum + device.event_count, 0),
    };
  }, [devices, referenceTime]);

  const filteredAndSortedDevices = useMemo(() => {
    const filtered = devices.filter((device) => {
      const matchesSearch = searchQuery.trim().length === 0
        || device.device_id.toLowerCase().includes(searchQuery.trim().toLowerCase());

      const age = referenceTime - new Date(device.latest_event_time).getTime();
      const matchesActivity = activityFilter === 'all'
        || (activityFilter === 'live' && age <= 60 * 60 * 1000)
        || (activityFilter === 'recent' && age > 60 * 60 * 1000 && age <= 24 * 60 * 60 * 1000)
        || (activityFilter === 'stale' && age > 24 * 60 * 60 * 1000);

      return matchesSearch && matchesActivity;
    });

    filtered.sort((a, b) => {
      let comparison = 0;

      if (sortField === 'device_id') {
        comparison = a.device_id.localeCompare(b.device_id);
      }

      if (sortField === 'event_count') {
        comparison = a.event_count - b.event_count;
      }

      if (sortField === 'latest_event_time') {
        comparison = new Date(a.latest_event_time).getTime() - new Date(b.latest_event_time).getTime();
      }

      return sortOrder === 'asc' ? comparison : -comparison;
    });

    return filtered;
  }, [activityFilter, devices, referenceTime, searchQuery, sortField, sortOrder]);

  if (!isAuthenticated) {
    return <AdminKeyInput />;
  }

  if (loading) {
    return <LoadingState message="正在加载设备列表" detail="正在整理设备状态与最近动态。" />;
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

  return (
    <div className="space-y-6">
      <PageHeader
        title="设备"
        description="按搜索、活跃度和排序方式快速浏览全部设备。"
        meta={
          <>
            <StatusBadge label={`已登记 ${devices.length} 台`} tone="primary" />
            <StatusBadge label={`在线 ${summary.live} 台`} tone="success" />
            <StatusBadge label={`待关注 ${summary.stale} 台`} tone={summary.stale > 0 ? 'warning' : 'default'} />
          </>
        }
      />

      <MetricStrip
        items={[
          { label: '设备总数', value: devices.length.toLocaleString('zh-CN') },
          { label: '近 1 小时在线', value: summary.live.toLocaleString('zh-CN') },
          { label: '今日活跃', value: summary.recent.toLocaleString('zh-CN') },
          { label: '总事件数', value: summary.totalEvents.toLocaleString('zh-CN') },
        ]}
      />

      <Section title="设备列表" description="按你的关注方式筛选、搜索并查看设备。">
        <div className="space-y-4">
          <div className="grid gap-3 xl:grid-cols-[minmax(0,1.4fr)_repeat(3,minmax(0,0.45fr))]">
            <div className="soft-input flex min-h-12 items-center gap-3 rounded-[--radius-md] px-3">
              <SearchIcon />
              <input
                className="min-w-0 flex-1 border-0 bg-transparent text-sm text-[--color-foreground] outline-none placeholder:text-[--color-foreground-dim]"
                placeholder="搜索设备 ID"
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
              value={activityFilter}
              onChange={(e) => setActivityFilter(e.target.value as ActivityFilter)}
            >
              <option value="all">全部活跃状态</option>
              <option value="live">近 1 小时在线</option>
              <option value="recent">今日活跃</option>
              <option value="stale">待关注</option>
            </select>

            <select
              className="soft-input min-h-12 rounded-[--radius-md] px-3 text-sm text-[--color-foreground] outline-none"
              value={sortField}
              onChange={(e) => setSortField(e.target.value as SortField)}
            >
              <option value="latest_event_time">排序：最近活跃</option>
              <option value="event_count">排序：事件数量</option>
              <option value="device_id">排序：设备 ID</option>
            </select>

            <select
              className="soft-input min-h-12 rounded-[--radius-md] px-3 text-sm text-[--color-foreground] outline-none"
              value={sortOrder}
              onChange={(e) => setSortOrder(e.target.value as SortOrder)}
            >
              <option value="desc">降序</option>
              <option value="asc">升序</option>
            </select>
          </div>

          {(searchQuery || activityFilter !== 'all') && (
            <div className="flex flex-wrap items-center gap-2">
              <StatusBadge label={`共匹配 ${filteredAndSortedDevices.length} 台`} tone="default" />
              {searchQuery && <StatusBadge label={`搜索：${searchQuery}`} tone="primary" />}
              {activityFilter !== 'all' && <StatusBadge label={`筛选：${activityFilterLabels[activityFilter]}`} tone="warning" />}
              <Button
                variant="outline"
                onPress={() => {
                  setSearchQuery('');
                  setActivityFilter('all');
                }}
              >
                清空条件
              </Button>
            </div>
          )}

          {filteredAndSortedDevices.length === 0 ? (
            <EmptyState
              title={devices.length === 0 ? '还没有设备' : '没有符合条件的设备'}
              description={devices.length === 0
                  ? '设备开始上报后，这里会自动出现。'
                  : '可以试试更换关键词，或放宽筛选条件。'}
              icon={<DeviceTableIcon />}
            />
            ) : (
              <div className="overflow-hidden rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,250,242,0.68)]">
               <div className="grid grid-cols-[minmax(0,1.3fr)_minmax(0,0.5fr)_minmax(0,0.8fr)_auto] gap-3 border-b border-[--color-border] bg-[rgba(112,91,64,0.03)] px-4 py-3 text-[11px] uppercase tracking-[0.18em] text-[--color-foreground-dim]">
                 <span>设备</span>
                 <span>事件数</span>
                 <span>最近动态</span>
                 <span className="text-right">操作</span>
               </div>

                <div className="divide-y divide-[--color-border]">
                {filteredAndSortedDevices.map((device) => {
                  const activity = getActivityMeta(device.latest_event_time);

                  return (
                    <div key={device.device_id} className="grid grid-cols-1 gap-4 px-4 py-4 transition-colors hover:bg-[rgba(112,91,64,0.04)] md:grid-cols-[minmax(0,1.3fr)_minmax(0,0.5fr)_minmax(0,0.8fr)_auto] md:items-center">
                      <div className="space-y-2">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="font-data text-sm text-[--color-foreground]">{device.device_id}</p>
                          <StatusBadge label={activity.label} tone={activity.tone} />
                        </div>
                        <p className="text-sm text-[--color-foreground-subtle]">最近更新时间：{formatDateTime(device.latest_event_time)}</p>
                      </div>

                      <div>
                        <p className="font-data text-sm font-semibold text-[--color-foreground]">{device.event_count.toLocaleString('zh-CN')}</p>
                        <p className="text-xs text-[--color-foreground-dim]">累计事件</p>
                      </div>

                      <div>
                        <p className="text-sm text-[--color-foreground]">{formatRelativeTime(device.latest_event_time)}</p>
                        <p className="text-xs text-[--color-foreground-dim]">按最近更新时间计算</p>
                      </div>

                      <div className="flex items-center justify-end">
                        <Link
                          className="inline-flex items-center justify-center rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,248,240,0.85)] px-3 py-2 text-sm text-[--color-foreground-subtle] transition-all hover:-translate-y-0.5 hover:border-[--color-border-subtle] hover:text-[--color-foreground]"
                          to={`/devices/${encodeURIComponent(device.device_id)}`}
                        >
                          查看设备
                        </Link>
                      </div>
                    </div>
                  );
                })}
                </div>
              </div>
            )}
        </div>
      </Section>
    </div>
  );
}

function SearchIcon() {
  return (
    <svg className="h-4 w-4 text-[--color-foreground-subtle]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
    </svg>
  );
}

function DeviceTableIcon() {
  return (
    <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
    </svg>
  );
}
