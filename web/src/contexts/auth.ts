import { createContext, useContext } from 'react';

export interface AuthContextType {
  adminKey: string;
  setAdminKey: (key: string) => void;
  isAuthenticated: boolean;
  clearAuth: () => void;
}

export const AuthContext = createContext<AuthContextType | null>(null);

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
