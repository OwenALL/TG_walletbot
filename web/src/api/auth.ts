import client from './client.ts';
import type { LoginReq, LoginResp } from '@/types/config.ts';

/**
 * 管理员登录
 * POST /auth/login
 */
export const login = (data: LoginReq): Promise<LoginResp> => {
  return client.post('/auth/login', data) as unknown as Promise<LoginResp>;
};

/**
 * 管理员登出（前端清除本地状态即可，后端无对应接口）
 */
export const logout = (): Promise<void> => {
  localStorage.removeItem('admin_token');
  localStorage.removeItem('admin_user');
  return Promise.resolve();
};

/**
 * 获取当前管理员信息
 * GET /me
 */
export const getProfile = (): Promise<LoginResp['user']> => {
  return client.get('/me') as unknown as Promise<LoginResp['user']>;
};

/**
 * 修改密码
 * PUT /auth/change-password
 */
export const changePassword = (oldPassword: string, newPassword: string): Promise<void> => {
  return client.put('/auth/change-password', {
    old_password: oldPassword,
    new_password: newPassword,
  }) as unknown as Promise<void>;
};
