import client from './client.ts';
import type { PaginatedResponse } from '@/types/common.ts';
import type {
  Withdrawal,
  WithdrawalListParams,
  ApproveWithdrawalReq,
  RejectWithdrawalReq,
  CompleteWithdrawalReq,
} from '@/types/transaction.ts';

/**
 * 获取提币记录列表
 * GET /withdrawals
 */
export const getWithdrawalList = (
  params?: WithdrawalListParams,
): Promise<PaginatedResponse<Withdrawal>> => {
  return client.get('/withdrawals', { params }) as unknown as Promise<
    PaginatedResponse<Withdrawal>
  >;
};

/**
 * 批准提币申请
 * PUT /withdrawals/:id/approve
 */
export const approveWithdrawal = (
  data: ApproveWithdrawalReq,
): Promise<void> => {
  return client.put(
    `/withdrawals/${data.id}/approve`,
  ) as unknown as Promise<void>;
};

/**
 * 拒绝提币申请
 * PUT /withdrawals/:id/reject
 */
export const rejectWithdrawal = (data: RejectWithdrawalReq): Promise<void> => {
  return client.put(`/withdrawals/${data.id}/reject`, {
    reason: data.reason,
  }) as unknown as Promise<void>;
};

/**
 * 手动完成提币 (管理员回填交易哈希)
 * PUT /withdrawals/:id/complete
 */
export const completeWithdrawal = (
  data: CompleteWithdrawalReq,
): Promise<void> => {
  return client.put(`/withdrawals/${data.id}/complete`, {
    tx_hash: data.tx_hash,
  }) as unknown as Promise<void>;
};
