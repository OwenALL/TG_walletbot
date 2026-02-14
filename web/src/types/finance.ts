/** 余额宝投资状态枚举 */
export const FinanceInvestmentStatus = {
  /** 持有中 */
  Active: 1,
  /** 已取出 */
  Withdrawn: 2,
} as const;

/** 余额宝投资状态类型 */
export type FinanceInvestmentStatus =
  (typeof FinanceInvestmentStatus)[keyof typeof FinanceInvestmentStatus];

/** 余额宝投资记录 */
export interface FinanceInvestment {
  /** 记录 ID */
  id: number;
  /** 用户 ID */
  user_id: number;
  /** 投资金额 */
  amount: string;
  /** 年化利率 (百分比值) */
  annual_rate: string;
  /** 状态: 1=持有中 2=已取出 */
  status: number;
  /** 开始计息时间 */
  interest_start: string;
  /** 累计收益 */
  total_profit: string;
  /** 创建时间 */
  created_at: string;
  /** 取出时间 */
  withdrawn_at?: string;
  /** 用户名 (后端 username 字段) */
  username?: string;
  /** 用户昵称 (后端 display_name 字段) */
  display_name?: string;
}

/** 余额宝统计数据 */
export interface FinanceStats {
  /** 总投资金额 (后端字段: total_invest_amount) */
  total_invest_amount: string;
  /** 活跃投资数 (后端字段: active_invest_count) */
  active_invest_count: number;
  /** 累计派息总额 */
  total_profit_paid: string;
  /** 当前年化利率 */
  current_annual_rate: string;
}

/** 余额宝投资列表查询参数 */
export interface FinanceListParams {
  page: number;
  size: number;
  /** 状态筛选 */
  status?: number;
}
