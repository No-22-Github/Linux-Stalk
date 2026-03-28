import { useEffect, useState, type ComponentProps } from 'react';
import { Button, Chip } from '@heroui/react';

import { ApiError, checkHealth, fetchDevices } from '@/api/client';
import { useAuth } from '@/contexts/auth';

interface AdminKeyInputProps {
  onSuccess?: () => void;
}

export function AdminKeyInput({ onSuccess }: AdminKeyInputProps) {
  const { setAdminKey } = useAuth();
  const [key, setKey] = useState('');
  const [error, setError] = useState<string | undefined>();
  const [isLoading, setIsLoading] = useState(false);
  const [showKey, setShowKey] = useState(false);
  const [serverReachable, setServerReachable] = useState<boolean | null>(null);

  useEffect(() => {
    void checkHealth().then(setServerReachable);
  }, []);

  const handleSubmit = async (
    e: Parameters<NonNullable<ComponentProps<'form'>['onSubmit']>>[0],
  ) => {
    e.preventDefault();

    const trimmedKey = key.trim();
    if (!trimmedKey) {
      setError('请输入访问密钥。');
      return;
    }

    setIsLoading(true);
    setError(undefined);

    try {
      await fetchDevices(trimmedKey);
      setAdminKey(trimmedKey);
      onSuccess?.();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError('访问密钥无效，请重新确认。');
      } else {
        setError(err instanceof Error ? err.message : '当前无法连接服务，请稍后再试。');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-10">
      <div className="pointer-events-none absolute inset-0 hero-grid opacity-60" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(181,102,59,0.14),transparent_34%)]" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_bottom_right,rgba(83,138,99,0.12),transparent_24%)]" />

      <div className="relative grid w-full max-w-5xl gap-6 lg:grid-cols-[1.1fr_0.9fr]">
        <section className="paper-panel relative overflow-hidden rounded-[--radius-lg] p-6 sm:p-8">
          <div className="accent-orb -left-8 top-14 h-28 w-28 bg-[rgba(97,127,157,0.14)]" />
          <div className="accent-orb right-10 top-8 h-24 w-24 bg-[rgba(181,102,59,0.16)]" />
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl border border-[--color-border] bg-[--color-primary-subtle] text-[--color-primary]">
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
              </svg>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.24em] text-[--color-foreground-dim]">Linux Stalk Studio</p>
              <h1 className="text-3xl font-semibold tracking-tight text-[--color-foreground]">进入设备观察台</h1>
            </div>
          </div>

          <p className="mt-6 max-w-xl text-sm leading-6 text-[--color-foreground-subtle]">
            输入访问密钥后，即可查看设备状态、最近动态和事件记录。
            密钥只保存在当前浏览器中，不会同步到其他地方。
          </p>

          <div className="mt-8 grid gap-3 sm:grid-cols-3">
            <FeatureCard
              title="只读查看"
              description="聚焦状态查看与排查，不影响现有数据流。"
            />
            <FeatureCard
              title="本地记住密钥"
              description="无需额外登录流程，当前浏览器会保留你的访问密钥。"
            />
            <FeatureCard
              title="进入前校验"
              description="提交后会先验证密钥，确认可用后再进入界面。"
            />
          </div>
        </section>

        <section className="glass-panel rounded-[--radius-lg] p-6 sm:p-8">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-xs uppercase tracking-[0.18em] text-[--color-foreground-dim]">访问</p>
              <h2 className="mt-1 text-xl font-semibold text-[--color-foreground]">验证密钥</h2>
            </div>
            <Chip
              className={`border text-xs ${serverReachable === true
                ? 'border-[--color-success-subtle] bg-[--color-success-subtle] text-[--color-success]'
                : serverReachable === false
                  ? 'border-[--color-danger-subtle] bg-[--color-danger-subtle] text-[--color-danger]'
                  : 'border-[--color-border] bg-[--color-background-muted] text-[--color-foreground-subtle]'
              }`}
              size="sm"
              variant="soft"
            >
              {serverReachable === true ? '服务可用' : serverReachable === false ? '服务异常' : '正在检查服务'}
            </Chip>
          </div>

          <form onSubmit={handleSubmit} className="mt-8 space-y-5">
            <div className="space-y-2">
              <label className="block text-xs font-medium uppercase tracking-[0.16em] text-[--color-foreground-dim]">
                访问密钥
              </label>
              <div className="soft-input rounded-[--radius-md] px-3 py-3">
                <div className="flex items-center gap-3">
                  <input
                    autoFocus
                    className="min-w-0 flex-1 border-0 bg-transparent text-sm text-[--color-foreground] outline-none placeholder:text-[--color-foreground-dim]"
                    placeholder="请输入 sk-admin-••••••••"
                    type={showKey ? 'text' : 'password'}
                    value={key}
                    onChange={(e) => {
                      setKey(e.target.value);
                      setError(undefined);
                    }}
                  />
                  <button
                    type="button"
                    className="text-xs font-medium text-[--color-foreground-subtle] transition-colors hover:text-[--color-foreground]"
                    onClick={() => setShowKey((value) => !value)}
                  >
                    {showKey ? '隐藏' : '显示'}
                  </button>
                </div>
              </div>
              <p className="text-xs text-[--color-foreground-dim]">
                保存前会先进行一次可用性校验。
              </p>
              {error && <p className="text-xs text-[--color-danger]">{error}</p>}
            </div>

            <div className="grid gap-3 sm:grid-cols-[1fr_auto]">
              <div className="soft-input rounded-[--radius-md] px-3 py-3 text-xs leading-5 text-[--color-foreground-subtle]">
                这里需要使用服务端配置中的 <span className="font-data text-[--color-foreground]">admin_keys</span>。设备上报使用的密钥不能用于登录这里。
              </div>
              <Button
                className="h-auto min-h-12 bg-[--color-primary] px-5 font-medium text-white hover:opacity-90"
                isDisabled={isLoading}
                type="submit"
              >
                {isLoading ? '正在验证…' : '进入观察台'}
              </Button>
            </div>
          </form>
        </section>
      </div>
    </div>
  );
}

function FeatureCard({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-[--radius-md] border border-[--color-border] bg-[rgba(255,250,242,0.66)] p-4 backdrop-blur-sm">
      <p className="text-sm font-medium text-[--color-foreground]">{title}</p>
      <p className="mt-2 text-sm leading-6 text-[--color-foreground-subtle]">{description}</p>
    </div>
  );
}
