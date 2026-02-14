import client from './client.ts';
import type { PaginatedResponse } from '@/types/common.ts';
import type { User, UserDetail, UserListParams, UpdateUserStatusReq } from '@/types/user.ts';

/**
 * 获取用户列表
 * GET /users
 */
export const getUserList = (
  params?: UserListParams,
): Promise<PaginatedResponse<User>> => {
  return client.get('/users', { params }) as unknown as Promise<
    PaginatedResponse<User>
  >;
};

/**
 * 获取用户详情 (含钱包信息)
 * GET /users/:id
 */
export const getUserDetail = (id: number): Promise<UserDetail> => {
  return client.get(`/users/${id}`) as unknown as Promise<UserDetail>;
};

/**
 * 更新用户状态 (冻结/解冻/封禁)
 * PUT /users/:id/status
 */
export const updateUserStatus = (data: UpdateUserStatusReq): Promise<void> => {
  return client.put(`/users/${data.id}/status`, {
    status: data.status,
  }) as unknown as Promise<void>;
};

/**
 * 重置用户支付密码 (清空 PIN)
 * PUT /users/:id/reset-pin
 */
export const resetUserPIN = (id: number): Promise<void> => {
  return client.put(`/users/${id}/reset-pin`) as unknown as Promise<void>;
};

/** 调整余额请求参数 */
export interface AdjustBalanceReq {
  /** 币种 */
  currency: string;
  /** 金额 (正数增加，负数扣减) */
  amount: number;
  /** 调整原因 */
  reason: string;
}

/**
 * 调整用户余额
 * PUT /users/:id/adjust-balance
 */
export const adjustUserBalance = (
  id: number,
  data: AdjustBalanceReq,
): Promise<void> => {
  return client.put(`/users/${id}/adjust-balance`, data) as unknown as Promise<void>;
};
