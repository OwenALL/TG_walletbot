import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import * as authApi from '@/api/auth.ts';
import type { LoginReq } from '@/types/config.ts';

/** 管理员用户信息 */
interface AdminUserInfo {
  id: number;
  username: string;
  role: string;
  last_login_at: string | null;
}

/** 认证状态 */
interface AuthState {
  /** JWT Token */
  token: string | null;
  /** 当前管理员信息 */
  user: AdminUserInfo | null;
  /** 是否已认证 */
  isAuthenticated: boolean;
  /** 登录 */
  login: (data: LoginReq) => Promise<void>;
  /** 登出 */
  logout: () => void;
  /** 设置 Token */
  setToken: (token: string) => void;
  /** 设置用户信息 */
  setUser: (user: AdminUserInfo) => void;
}

/**
 * 认证状态管理 Store
 * 使用 zustand persist 中间件将 token 和 user 持久化到 localStorage
 */
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      isAuthenticated: false,

      login: async (data: LoginReq) => {
        const resp = await authApi.login(data);
        set({
          token: resp.token,
          user: resp.user,
          isAuthenticated: true,
        });
        /* 同步写入 localStorage 供 axios 拦截器直接读取 */
        localStorage.setItem('admin_token', resp.token);
      },

      logout: () => {
        set({
          token: null,
          user: null,
          isAuthenticated: false,
        });
        localStorage.removeItem('admin_token');
      },

      setToken: (token: string) => {
        set({ token, isAuthenticated: true });
        localStorage.setItem('admin_token', token);
      },

      setUser: (user: AdminUserInfo) => {
        set({ user });
      },
    }),
    {
      name: 'admin-auth-storage',
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    },
  ),
);
