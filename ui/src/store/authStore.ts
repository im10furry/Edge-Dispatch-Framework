import { create } from 'zustand';
import type { User, RoleBinding } from '../types/api';

const AUTH_STORAGE_KEY = 'edf.auth.v1';

interface PersistedAuthState {
  token: string;
  refreshToken: string;
  user: User;
  roles: RoleBinding[];
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  roles: RoleBinding[];
  isAuthenticated: boolean;
  persistAuth: boolean;

  setAuth: (token: string, refreshToken: string, user: User, roles: RoleBinding[], remember?: boolean) => void;
  setTokens: (token: string, refreshToken: string) => void;
  setUser: (user: User) => void;
  logout: () => void;
  hasRole: (role: string) => boolean;
  hasTenantAccess: (tenantId: string) => boolean;
  hasProjectAccess: (tenantId: string, projectId: string) => boolean;
}

function loadStoredAuth(): PersistedAuthState | null {
  try {
    const raw = window.localStorage.getItem(AUTH_STORAGE_KEY);
    if (!raw) return null;

    const parsed = JSON.parse(raw) as Partial<PersistedAuthState>;
    if (!parsed.token || !parsed.refreshToken || !parsed.user || !Array.isArray(parsed.roles)) {
      return null;
    }

    return {
      token: parsed.token,
      refreshToken: parsed.refreshToken,
      user: parsed.user,
      roles: parsed.roles,
    };
  } catch {
    window.localStorage.removeItem(AUTH_STORAGE_KEY);
    return null;
  }
}

function storeAuth(state: PersistedAuthState) {
  window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(state));
}

function clearStoredAuth() {
  window.localStorage.removeItem(AUTH_STORAGE_KEY);
}

const storedAuth = loadStoredAuth();

export const useAuthStore = create<AuthState>((set, get) => ({
  token: storedAuth?.token ?? null,
  refreshToken: storedAuth?.refreshToken ?? null,
  user: storedAuth?.user ?? null,
  roles: storedAuth?.roles ?? [],
  isAuthenticated: !!storedAuth?.token,
  persistAuth: !!storedAuth?.token,

  setAuth: (token, refreshToken, user, roles, remember = false) => {
    if (remember) {
      storeAuth({ token, refreshToken, user, roles });
    } else {
      clearStoredAuth();
    }
    set({ token, refreshToken, user, roles, isAuthenticated: true, persistAuth: remember });
  },

  setTokens: (token, refreshToken) => {
    const { user, roles, persistAuth } = get();
    if (persistAuth && user) {
      storeAuth({ token, refreshToken, user, roles });
    }
    set({ token, refreshToken });
  },

  setUser: (user) => {
    const { token, refreshToken, roles, persistAuth } = get();
    if (persistAuth && token && refreshToken) {
      storeAuth({ token, refreshToken, user, roles });
    }
    set({ user });
  },

  logout: () => {
    clearStoredAuth();
    set({
      token: null,
      refreshToken: null,
      user: null,
      roles: [],
      isAuthenticated: false,
      persistAuth: false,
    });
  },

  hasRole: (role: string) => {
    return get().roles.some((r) => r.role === role);
  },

  hasTenantAccess: (tenantId: string) => {
    const { roles } = get();
    return roles.some((r) => r.tenant_id === tenantId || r.role === 'tenant_owner' || r.role === 'tenant_admin');
  },

  hasProjectAccess: (_tenantId: string, projectId: string) => {
    const { roles } = get();
    return roles.some(
      (r) =>
        r.project_id === projectId ||
        r.role === 'tenant_owner' ||
        r.role === 'tenant_admin',
    );
  },
}));
