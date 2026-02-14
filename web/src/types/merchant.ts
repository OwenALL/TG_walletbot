/** 商户状态枚举 */
export const MerchantStatus = {
  /** 已关闭 */
  Disabled: 0,
  /** 正常 */
  Active: 1,
  /** 待审核 (旧流程) */
  Pending: 2,
  /** 已拒绝 (旧流程) */
  Rejected: 3,
} as const;

/** 商户状态类型 */
export type MerchantStatus =
  (typeof MerchantStatus)[keyof typeof MerchantStatus];

/** 商户记录 */
export interface Merchant {
  /** 商户 ID */
  id: number;
  /** 用户 ID */
  user_id: number;
  /** 商户名称 */
  business_name: string;
  /** API Key */
  api_key: string;
  /** API Secret */
  api_secret: string;
  /** Logo URL */
  logo: string;
  /** 费率 (decimal 字符串，如 "0.01") */
  fee_rate: string;
  /** IP 白名单 */
  ip_whitelist: string;
  /** Webhook URL */
  webhook_url: string;
  /** 状态: 0=已关闭 1=正常 2=待审核(旧) 3=已拒绝(旧) */
  status: number;
  /** 商户描述 */
  description: string;
  /** 创建时间 */
  created_at: string;
  /** 更新时间 */
  updated_at: string;
  /** 用户名 (后端 username 字段) */
  username?: string;
  /** 用户昵称 (后端 display_name 字段) */
  display_name?: string;
}

/** 商户列表查询参数 */
export interface MerchantListParams {
  page: number;
  size: number;
  /** 状态筛选 */
  status?: number;
}
