import { useState, useCallback, type ReactNode } from 'react';

import { AuthContext } from '@/contexts/auth';

const STORAGE_KEY = 'linux-stalk-admin-key';

export function AuthProvider({ children }: { children: ReactNode }) {
  const [adminKey, setAdminKeyState] = useState<string>(() => {
    if (typeof window === 'undefined') return '';
    return localStorage.getItem(STORAGE_KEY) || '';
  });

  const setAdminKey = useCallback((key: string) => {
    setAdminKeyState(key);
    if (typeof window !== 'undefined') {
      if (key) {
        localStorage.setItem(STORAGE_KEY, key);
      } else {
        localStorage.removeItem(STORAGE_KEY);
      }
    }
  }, []);

  const clearAuth = useCallback(() => {
    setAdminKeyState('');
    if (typeof window !== 'undefined') {
      localStorage.removeItem(STORAGE_KEY);
    }
  }, []);

  return (
    <AuthContext.Provider
      value={{
        adminKey,
        setAdminKey,
        isAuthenticated: Boolean(adminKey),
        clearAuth,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
