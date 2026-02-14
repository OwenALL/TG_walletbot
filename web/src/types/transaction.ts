/** 交易类型枚举 */
export type TransactionType =
  | 'deposit'
  | 'withdraw'
  | 'transfer_in'
  | 'transfer_out'
  | 'exchange_in'
  | 'exchange_out'
  | 'redpacket_send'
  | 'redpacket_receive'
  | 'redpacket_refund'
  | 'finance_in'
  | 'finance_out'
  | 'finance_profit';

/** 交易流水 */
export interface Transaction {
  /** 交易 ID */
  id: number;
  /** 用户 ID */
  user_id: number;
  /** 交易类型 */
  type: TransactionType;
  /** 币种 */
  currency: string;
  /** 交易金额 */
  amount: string;
  /** 手续费 */
  fee: string;
  /** 交易前余额 */
  balance_before: string;
  /** 交易后余额 */
  balance_after: string;
  /** 关联订单 ID */
  related_id: number | null;
  /** 关联订单类型 */
  related_type: string;
  /** 备注 */
  memo: string;
  /** 创建时间 */
  created_at: string;
  /** 关联用户名 (用于展示) */
  username?: string;
  /** 关联用户 Telegram ID */
  telegram_id?: number;
}

/** 交易列表查询参数 */
export interface TransactionListParams {
  page?: number;
  size?: number;
  /** 用户 ID 筛选 */
  user_id?: number;
  /** 交易类型筛选 */
  type?: TransactionType;
  /** 币种筛选 */
  currency?: string;
  /** 开始时间 */
  start_time?: string;
  /** 结束时间 */
  end_time?: string;
}

/** 充值记录状态常量 */
export const DepositStatus = {
  /** 待确认 */
  Pending: 0,
  /** 已确认 */
  Confirmed: 1,
  /** 已到账 */
  Credited: 2,
} as const;

/** 充值记录状态类型 */
export type DepositStatus = (typeof DepositStatus)[keyof typeof DepositStatus];

/** 充值记录 */
export interface Deposit {
  id: number;
  user_id: number;
  currency: string;
  amount: string;
  from_address: string;
  to_address: string;
  tx_hash: string;
  confirmations: number;
  status: DepositStatus;
  notified: boolean;
  created_at: string;
  confirmed_at: string | null;
  /** 关联用户名 (用于展示) */
  username?: string;
}

/** 提币记录状态常量 */
export const WithdrawalStatus = {
  /** 待审核 */
  Pending: 0,
  /** 处理中 */
  Processing: 1,
  /** 已完成 */
  Completed: 2,
  /** 已拒绝 */
  Rejected: 3,
  /** 失败 */
  Failed: 4,
} as const;

/** 提币记录状态类型 */
export type WithdrawalStatus = (typeof WithdrawalStatus)[keyof typeof WithdrawalStatus];

/** 提币记录 */
export interface Withdrawal {
  id: number;
  user_id: number;
  currency: string;
  amount: string;
  fee: string;
  actual_amount: string;
  to_address: string;
  tx_hash: string;
  status: WithdrawalStatus;
  reviewer_id: number | null;
  review_note: string;
  created_at: string;
  reviewed_at: string | null;
  completed_at: string | null;
  /** 关联用户名 (用于展示) */
  username?: string;
  /** 关联用户 Telegram ID */
  telegram_id?: number;
}

/** 提币列表查询参数 */
export interface WithdrawalListParams {
  page?: number;
  size?: number;
  /** 用户 ID 筛选 */
  user_id?: number;
  /** 状态筛选 */
  status?: WithdrawalStatus;
  /** 币种筛选 */
  currency?: string;
  /** 开始时间 */
  start_time?: string;
  /** 结束时间 */
  end_time?: string;
}

/** 提币审批请求 */
export interface ApproveWithdrawalReq {
  /** 提币记录 ID */
  id: number;
}

/** 提币拒绝请求 */
export interface RejectWithdrawalReq {
  /** 提币记录 ID */
  id: number;
  /** 拒绝原因 */
  reason: string;
}

/** 手动完成提币请求 */
export interface CompleteWithdrawalReq {
  /** 提币记录 ID */
  id: number;
  /** 链上交易哈希 */
  tx_hash: string;
}

/** 转账记录 */
export interface Transfer {
  id: number;
  from_user_id: number;
  to_user_id: number;
  currency: string;
  amount: string;
  /** 1:主动转账 2:收款请求 */
  type: number;
  /** 0:待确认 1:已完成 2:已拒绝 3:已取消 */
  status: number;
  memo: string;
  created_at: string;
  completed_at: string | null;
  /** 发起方用户名 */
  from_username?: string;
  /** 接收方用户名 */
  to_username?: string;
}
