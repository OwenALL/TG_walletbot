/** 审计日志条目 */
export interface AuditLog {
  /** 日志 ID */
  id: number;
  /** 操作管理员 ID */
  admin_id: number;
  /** 操作管理员用户名 (后端 JOIN 返回) */
  admin_username: string;
  /** 操作类型 (如 login、update_config、approve_withdrawal 等) */
  action: string;
  /** 操作目标类型 (如 user、withdrawal、config 等) */
  target_type: string;
  /** 操作目标 ID (后端 *uint64，可为 null) */
  target_id: number | null;
  /** 操作详情 (JSON 字符串)，注意: 后端字段名为 detail (单数) */
  detail: string;
  /** 操作来源 IP */
  ip_address: string;
  /** 创建时间 */
  created_at: string;
}

/** 审计日志列表查询参数 */
export interface AuditLogListParams {
  /** 页码 */
  page?: number;
  /** 每页条数 */
  size?: number;
  /** 管理员 ID 筛选 */
  admin_id?: number;
  /** 操作类型筛选 */
  action?: string;
  /** 开始日期 (YYYY-MM-DD) */
  start_time?: string;
  /** 结束日期 (YYYY-MM-DD) */
  end_time?: string;
}
