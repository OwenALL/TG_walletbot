/** 支持的币种列表 */
export const CURRENCIES = ['USDT', 'TRX', 'CNY'] as const;

/** 币种类型 */
export type CurrencyType = (typeof CURRENCIES)[number];

/** 币种颜色映射 */
export const CURRENCY_COLORS: Record<string, string> = {
  USDT: '#26a17b',
  TRX: '#ef0027',
  CNY: '#cf1322',
};

/** 交易类型选项 */
export const TRANSACTION_TYPE_OPTIONS = [
  { label: '全部', value: '' },
  { label: '充值', value: 'deposit' },
  { label: '提币', value: 'withdraw' },
  { label: '转入', value: 'transfer_in' },
  { label: '转出', value: 'transfer_out' },
  { label: '闪兑兑入', value: 'exchange_in' },
  { label: '闪兑兑出', value: 'exchange_out' },
  { label: '红包发出', value: 'redpacket_send' },
  { label: '红包领取', value: 'redpacket_receive' },
  { label: '红包退回', value: 'redpacket_refund' },
] as const;

/** 币种选项 */
export const CURRENCY_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'USDT', value: 'USDT' },
  { label: 'TRX', value: 'TRX' },
  { label: 'CNY', value: 'CNY' },
] as const;

/** 提币状态选项 */
export const WITHDRAWAL_STATUS_OPTIONS = [
  { label: '全部', value: -1 },
  { label: '待审核', value: 0 },
  { label: '处理中', value: 1 },
  { label: '已完成', value: 2 },
  { label: '已拒绝', value: 3 },
  { label: '失败', value: 4 },
] as const;

/** 用户状态选项 */
export const USER_STATUS_OPTIONS = [
  { label: '全部', value: 0 },
  { label: '正常', value: 1 },
  { label: '冻结', value: 2 },
  { label: '封禁', value: 3 },
] as const;

/** 分页默认配置 */
export const DEFAULT_PAGE_SIZE = 20;

/** 限额相关的配置 key 集合，用于 SystemConfig 与 LimitManagement 页面去重 */
export const LIMIT_CONFIG_KEYS = new Set([
  'withdraw_usdt_min',
  'withdraw_usdt_max',
  'withdraw_usdt_daily_max',
  'withdraw_usdt_auto_threshold',
  'withdraw_trx_min',
  'withdraw_trx_max',
  'withdraw_cny_min',
  'withdraw_cny_daily_max',
  'transfer_usdt_min',
  'exchange_min_usdt',
  'exchange_min_cny',
  'exchange_min_trx',
  'finance_min',
  'finance_max',
]);

/** 侧边栏菜单 key 定义 */
export const MENU_KEYS = {
  DASHBOARD: '/',
  USERS: '/users',
  TRANSACTIONS: '/transactions',
  WITHDRAWALS: '/withdrawals',
  EXCHANGE_RATES: '/exchange-rates',
  FEE_MANAGEMENT: '/fee-management',
  LIMIT_MANAGEMENT: '/limit-management',
  SYSTEM_CONFIG: '/system-config',
  AUDIT_LOGS: '/audit-logs',
  RED_PACKETS: '/red-packets',
  FINANCE: '/finance',
  MERCHANTS: '/merchants',
} as const;
