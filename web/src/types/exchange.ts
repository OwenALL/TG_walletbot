/** 汇率配置 */
export interface ExchangeRate {
  /** ID */
  id: number;
  /** 兑出币种 */
  from_currency: string;
  /** 兑入币种 */
  to_currency: string;
  /** 基准汇率 */
  rate: string;
  /** 加点比例 (%) */
  spread: string;
  /** 最低兑换量 */
  min_amount: string;
  /** 最高兑换量 */
  max_amount: string;
  /** 是否启用 */
  enabled: boolean;
  /** 更新时间 */
  updated_at: string;
}

/** 更新汇率请求 */
export interface UpdateExchangeRateReq {
  /** 汇率配置 ID */
  id: number;
  /** 基准汇率 */
  rate: string;
  /** 加点比例 (%) */
  spread: string;
  /** 最低兑换量 */
  min_amount: string;
  /** 最高兑换量 */
  max_amount: string;
  /** 是否启用 */
  enabled: boolean;
}

/** 兑换记录 */
export interface Exchange {
  /** ID */
  id: number;
  /** 用户 ID */
  user_id: number;
  /** 兑出币种 */
  from_currency: string;
  /** 兑入币种 */
  to_currency: string;
  /** 兑出金额 */
  from_amount: string;
  /** 兑入金额 */
  to_amount: string;
  /** 汇率 */
  rate: string;
  /** 手续费 */
  fee: string;
  /** 状态 1:已完成 */
  status: number;
  /** 创建时间 */
  created_at: string;
  /** 关联用户名 (用于展示) */
  username?: string;
}

/** 兑换记录查询参数 */
export interface ExchangeListParams {
  page?: number;
  size?: number;
  user_id?: number;
  from_currency?: string;
  to_currency?: string;
  start_time?: string;
  end_time?: string;
}
