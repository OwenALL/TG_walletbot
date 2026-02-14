/** 用户状态常量 */
export const UserStatus = {
  /** 正常 */
  Active: 1,
  /** 冻结 */
  Frozen: 2,
  /** 封禁 */
  Banned: 3,
} as const;

/** 用户状态类型 */
export type UserStatus = (typeof UserStatus)[keyof typeof UserStatus];

/** 用户信息 */
export interface User {
  /** 用户 ID */
  id: number;
  /** Telegram 用户 ID */
  telegram_id: number;
  /** Telegram @username */
  username: string;
  /** 名 */
  first_name: string;
  /** 姓 */
  last_name: string;
  /** 语言代码 */
  language_code: string;
  /** 是否 Telegram Premium */
  is_premium: boolean;
  /** 是否已设置 PIN */
  has_pin: boolean;
  /** 是否启用 2FA (Google 验证器) */
  totp_enabled: boolean;
  /** 用户状态 */
  status: UserStatus;
  /** PIN 连续失败次数 */
  pin_fail_count: number;
  /** PIN 锁定截止时间 */
  pin_locked_until: string | null;
  /** 邀请人 ID */
  referrer_id: number | null;
  /** 创建时间 */
  created_at: string;
  /** 更新时间 */
  updated_at: string;
  /** USDT 余额 (列表接口返回) */
  usdt_balance?: string;
  /** TRX 余额 (列表接口返回) */
  trx_balance?: string;
  /** CNY 余额 (列表接口返回) */
  cny_balance?: string;
}

/** 用户钱包信息 */
export interface Wallet {
  /** 钱包 ID */
  id: number;
  /** 用户 ID */
  user_id: number;
  /** 币种 USDT / TRX / CNY */
  currency: string;
  /** 可用余额 */
  balance: string;
  /** 冻结余额 */
  frozen_balance: string;
  /** 充值地址 */
  deposit_address: string;
  /** 创建时间 */
  created_at: string;
  /** 更新时间 */
  updated_at: string;
}

/** 用户详情 (含钱包) */
export interface UserDetail extends User {
  /** 用户所有钱包 */
  wallets: Wallet[];
}

/** 用户列表查询参数 */
export interface UserListParams {
  /** 当前页码 */
  page?: number;
  /** 每页条数 */
  size?: number;
  /** 搜索关键词 (Telegram ID / @username) */
  keyword?: string;
  /** 用户状态筛选 */
  status?: UserStatus;
}

/** 冻结/解冻用户请求 */
export interface UpdateUserStatusReq {
  /** 用户 ID */
  id: number;
  /** 目标状态 */
  status: UserStatus;
}

/** 管理员用户信息 */
export interface AdminUser {
  /** 管理员 ID */
  id: number;
  /** Telegram ID */
  telegram_id: number | null;
  /** 用户名 */
  username: string;
  /** 角色 */
  role: 'super_admin' | 'admin' | 'viewer';
  /** 状态 */
  status: number;
  /** 上次登录时间 */
  last_login_at: string | null;
  /** 创建时间 */
  created_at: string;
}
