import client from './client.ts';
import type { ExchangeRate, UpdateExchangeRateReq } from '@/types/exchange.ts';

/**
 * 获取所有汇率配置
 * GET /exchange-rates
 */
export const getExchangeRates = (): Promise<ExchangeRate[]> => {
  return client.get('/exchange-rates') as unknown as Promise<ExchangeRate[]>;
};

/**
 * 更新汇率配置（批量更新，后端 PUT /exchange-rates 无 :id）
 * PUT /exchange-rates
 */
export const updateExchangeRate = (
  data: UpdateExchangeRateReq,
): Promise<void> => {
  return client.put('/exchange-rates', data) as unknown as Promise<void>;
};
