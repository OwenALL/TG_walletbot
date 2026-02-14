import client from './client.ts';
import type { PaginatedResponse } from '@/types/common.ts';
import type { AuditLog, AuditLogListParams } from '@/types/auditLog.ts';

export type { AuditLog, AuditLogListParams };

/**
 * 获取审计日志列表
 * GET /audit-logs
 */
export const getAuditLogs = (
  params?: AuditLogListParams,
): Promise<PaginatedResponse<AuditLog>> => {
  return client.get('/audit-logs', { params }) as unknown as Promise<
    PaginatedResponse<AuditLog>
  >;
};
