/** 红包类型枚举 */
export const RedPacketType = {
  /** 均分红包 */
  Equal: 1,
  /** 拼手气红包 */
  Random: 2,
} as const;

/** 红包类型 */
export type RedPacketType = (typeof RedPacketType)[keyof typeof RedPacketType];

/** 红包状态枚举 */
export const RedPacketStatus = {
  /** 待发送 */
  Pending: 0,
  /** 已发送 */
  Sent: 1,
  /** 已领完 */
  Claimed: 2,
  /** 已过期 */
  Expired: 3,
} as const;

/** 红包状态类型 */
export type RedPacketStatus = (typeof RedPacketStatus)[keyof typeof RedPacketStatus];

/** 红包记录 */
export interface RedPacket {
  /** 红包 ID */
  id: number;
  /** 发送用户 ID */
  user_id: number;
  /** 币种 */
  currency: string;
  /** 总金额 */
  total_amount: string;
  /** 红包个数 */
  total_count: number;
  /** 已领取个数 */
  claimed_count: number;
  /** 已领取金额 */
  claimed_amount: string;
  /** 红包类型: 1=均分 2=拼手气 */
  type: number;
  /** 状态: 0=待发送 1=已发送 2=已领完 3=已过期 */
  status: number;
  /** 过期时间 */
  expire_at: string;
  /** 创建时间 */
  created_at: string;
  /** 发送者用户名 (后端 sender_username 字段) */
  sender_username?: string;
  /** 发送者昵称 (后端 sender_name 字段) */
  sender_name?: string;
}

/** 红包领取记录 */
export interface RedPacketClaim {
  /** 记录 ID */
  id: number;
  /** 红包 ID */
  red_packet_id: number;
  /** 领取用户 ID */
  user_id: number;
  /** 领取金额 */
  amount: string;
  /** 领取时间 */
  created_at: string;
  /** 领取者用户名 (后端 claimer_username 字段) */
  claimer_username?: string;
  /** 领取者昵称 (后端 claimer_name 字段) */
  claimer_name?: string;
}

/** 红包统计数据 */
export interface RedPacketStats {
  /** 红包总数 */
  total_count: number;
  /** 红包总金额 */
  total_amount: string;
  /** 已发送数 */
  sent_count: number;
  /** 已领完数 */
  claimed_count: number;
  /** 已过期数 */
  expired_count: number;
  /** 待发送数 */
  pending_count: number;
}

/** 红包列表查询参数 */
export interface RedPacketListParams {
  page: number;
  size: number;
  /** 状态筛选 */
  status?: number;
}
