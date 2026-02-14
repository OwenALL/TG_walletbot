import client from './client.ts';
import type { PaginatedResponse } from '@/types/common.ts';
import type { Transaction, Deposit, TransactionListParams } from '@/types/transaction.ts';

/**
 * 获取交易流水列表
 * GET /transactions
 */
export const getTransactionList = (
  params?: TransactionListParams,
): Promise<PaginatedResponse<Transaction>> => {
  return client.get('/transactions', { params }) as unknown as Promise<
    PaginatedResponse<Transaction>
  >;
};

/**
 * 获取充值记录列表
 * GET /transactions/deposits
 */
export const getDepositList = (
  params?: { page?: number; size?: number },
): Promise<PaginatedResponse<Deposit>> => {
  return client.get('/transactions/deposits', { params }) as unknown as Promise<
    PaginatedResponse<Deposit>
  >;
};
