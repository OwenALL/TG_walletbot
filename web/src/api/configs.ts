import client from './client.ts';
import type { SystemConfig, UpdateConfigReq } from '@/types/config.ts';
import type { DashboardStats, TrendDataPoint } from '@/types/config.ts';

/**
 * 获取所有系统配置
 * GET /configs
 */
export const getConfigs = (): Promise<SystemConfig[]> => {
  return client.get('/configs') as unknown as Promise<SystemConfig[]>;
};

/**
 * 更新系统配置（后端 PUT /configs 无 :id，整体更新）
 * PUT /configs
 */
export const updateConfig = (data: UpdateConfigReq): Promise<void> => {
  return client.put('/configs', data) as unknown as Promise<void>;
};

/**
 * 获取仪表盘统计数据
 * GET /dashboard/stats
 */
export const getDashboardStats = (): Promise<DashboardStats> => {
  return client.get('/dashboard/stats') as unknown as Promise<DashboardStats>;
};

/**
 * 获取交易趋势数据
 * GET /dashboard/trend
 */
export const getDashboardTrend = (
  days?: number,
): Promise<TrendDataPoint[]> => {
  return client.get('/dashboard/trend', {
    params: { days: days ?? 7 },
  }) as unknown as Promise<TrendDataPoint[]>;
};
