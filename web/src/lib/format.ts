export function formatRelativeTime(value: string | Date): string {
  const date = value instanceof Date ? value : new Date(value);

  if (Number.isNaN(date.getTime())) {
    return typeof value === 'string' ? value : '未知';
  }

  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffMins < 1) return '刚刚';
  if (diffMins < 60) return `${diffMins} 分钟前`;
  if (diffHours < 24) return `${diffHours} 小时前`;
  if (diffDays < 7) return `${diffDays} 天前`;

  return date.toLocaleDateString('zh-CN');
}

export function formatDateTime(value: string | Date): string {
  const date = value instanceof Date ? value : new Date(value);

  if (Number.isNaN(date.getTime())) {
    return typeof value === 'string' ? value : '未知';
  }

  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

export function formatCompactNumber(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value);
}

export type ActivityTone = 'success' | 'warning' | 'danger';

export function getActivityMeta(value: string | Date): {
  label: string;
  tone: ActivityTone;
} {
  const date = value instanceof Date ? value : new Date(value);
  const diffMs = Date.now() - date.getTime();

  if (diffMs <= 60 * 60 * 1000) {
    return { label: '在线', tone: 'success' };
  }

  if (diffMs <= 24 * 60 * 60 * 1000) {
    return { label: '今日活跃', tone: 'warning' };
  }

  return { label: '待关注', tone: 'danger' };
}
