import client from './client.ts';
import type { PaginatedResponse } from '@/types/common.ts';
import type {
  RedPacket,
  RedPacketClaim,
  RedPacketStats,
  RedPacketListParams,
} from '@/types/redPacket.ts';

/**
 * 获取红包列表
 * GET /red-packets
 */
export const getRedPacketList = (
  params: RedPacketListParams,
): Promise<PaginatedResponse<RedPacket>> => {
  return client.get('/red-packets', { params }) as unknown as Promise<
    PaginatedResponse<RedPacket>
  >;
};

/**
 * 获取红包详情 (含领取记录)
 * GET /red-packets/:id
 */
export const getRedPacketDetail = (
  id: number,
): Promise<{ red_packet: RedPacket; claims: RedPacketClaim[] }> => {
  return client.get(`/red-packets/${id}`) as unknown as Promise<{
    red_packet: RedPacket;
    claims: RedPacketClaim[];
  }>;
};

/**
 * 获取红包统计数据
 * GET /red-packets/stats
 */
export const getRedPacketStats = (): Promise<RedPacketStats> => {
  return client.get('/red-packets/stats') as unknown as Promise<RedPacketStats>;
};
