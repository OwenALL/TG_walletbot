import client from './client.ts';
import type { PaginatedResponse } from '@/types/common.ts';
import type { Merchant, MerchantListParams } from '@/types/merchant.ts';

/**
 * 获取商户列表
 * GET /merchants
 */
export const getMerchantList = (
  params: MerchantListParams,
): Promise<PaginatedResponse<Merchant>> => {
  return client.get('/merchants', { params }) as unknown as Promise<
    PaginatedResponse<Merchant>
  >;
};

/**
 * 切换商户状态 (启用/禁用)
 * PUT /merchants/:id/toggle-status
 */
export const toggleMerchantStatus = (id: number): Promise<void> => {
  return client.put(`/merchants/${id}/toggle-status`) as unknown as Promise<void>;
};

/**
 * 修改商户费率
 * PUT /merchants/:id/fee-rate
 */
export const updateMerchantFeeRate = (id: number, feeRate: number): Promise<void> => {
  return client.put(`/merchants/${id}/fee-rate`, { fee_rate: feeRate }) as unknown as Promise<void>;
};
