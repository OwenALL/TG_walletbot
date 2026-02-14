import client from './client.ts';
import type { PaginatedResponse } from '@/types/common.ts';
import type {
  FinanceInvestment,
  FinanceStats,
  FinanceListParams,
} from '@/types/finance.ts';

/**
 * 获取余额宝投资列表
 * GET /finance/investments
 */
export const getFinanceInvestments = (
  params: FinanceListParams,
): Promise<PaginatedResponse<FinanceInvestment>> => {
  return client.get('/finance/investments', { params }) as unknown as Promise<
    PaginatedResponse<FinanceInvestment>
  >;
};

/**
 * 获取余额宝统计数据
 * GET /finance/stats
 */
export const getFinanceStats = (): Promise<FinanceStats> => {
  return client.get('/finance/stats') as unknown as Promise<FinanceStats>;
};

/**
 * 更新余额宝年化利率
 * PUT /finance/rate
 */
export const updateFinanceRate = (annualRate: number): Promise<void> => {
  return client.put('/finance/rate', {
    annual_rate: annualRate,
  }) as unknown as Promise<void>;
};
