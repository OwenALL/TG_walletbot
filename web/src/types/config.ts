/** 系统配置项 */
export interface SystemConfig {
  /** ID */
  id: number;
  /** 配置键名 */
  config_key: string;
  /** 配置值 */
  config_value: string;
  /** 配置说明 */
  description: string;
  /** 更新时间 */
  updated_at: string;
}

/** 更新系统配置请求 */
export interface UpdateConfigReq {
  /** 配置 ID */
  id: number;
  /** 配置值 */
  config_value: string;
}

/** 批量更新系统配置请求 */
export interface BatchUpdateConfigReq {
  configs: UpdateConfigReq[];
}

/** 仪表盘统计数据 */
export interface DashboardStats {
  /** 总用户数 */
  total_users: number;
  /** 今日新增用户 */
  today_new_users: number;
  /** 今日交易笔数 */
  today_transactions: number;
  /** 今日交易量 (USDT 等值) */
  today_volume: string;
  /** USDT 总余额 */
  total_usdt_balance: string;
  /** TRX 总余额 */
  total_trx_balance: string;
  /** CNY 总余额 */
  total_cny_balance: string;
  /** 待审核提币数 */
  pending_withdrawals: number;
}

/** 交易趋势数据点 */
export interface TrendDataPoint {
  /** 日期 (YYYY-MM-DD) */
  date: string;
  /** 交易笔数 */
  count: number;
  /** 交易量 */
  volume: string;
}

/** 登录请求 */
export interface LoginReq {
  username: string;
  password: string;
}

/** 登录响应 */
export interface LoginResp {
  token: string;
  user: {
    id: number;
    username: string;
    role: string;
    last_login_at: string | null;
  };
}
