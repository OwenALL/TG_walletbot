import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Table,
  Tag,
  Card,
  Statistic,
  Row,
  Col,
  Select,
  Space,
  App,
  Spin,
} from 'antd';
import {
  GiftOutlined,
  DollarOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { RedPacket, RedPacketClaim, RedPacketStats, RedPacketListParams } from '@/types/redPacket.ts';
import { RedPacketStatus, RedPacketType } from '@/types/redPacket.ts';
import { getRedPacketList, getRedPacketStats, getRedPacketDetail } from '@/api/redPackets.ts';
import { formatAmount, formatTime } from '@/utils/format.ts';
import { DEFAULT_PAGE_SIZE, CURRENCY_COLORS } from '@/utils/constants.ts';

/** 红包状态筛选选项 */
const STATUS_OPTIONS = [
  { label: '全部', value: -1 },
  { label: '待发送', value: RedPacketStatus.Pending },
  { label: '已发送', value: RedPacketStatus.Sent },
  { label: '已领完', value: RedPacketStatus.Claimed },
  { label: '已过期', value: RedPacketStatus.Expired },
];

/** 获取红包状态 Tag 颜色和文本 */
const getRedPacketStatusTag = (
  status: number,
): { color: string; text: string } => {
  switch (status) {
    case RedPacketStatus.Pending:
      return { color: 'blue', text: '待发送' };
    case RedPacketStatus.Sent:
      return { color: 'processing', text: '已发送' };
    case RedPacketStatus.Claimed:
      return { color: 'success', text: '已领完' };
    case RedPacketStatus.Expired:
      return { color: 'default', text: '已过期' };
    default:
      return { color: 'default', text: '未知' };
  }
};

/** 获取红包类型显示文本 */
const getRedPacketTypeName = (type: number): string => {
  switch (type) {
    case RedPacketType.Equal:
      return '均分';
    case RedPacketType.Random:
      return '拼手气';
    default:
      return '未知';
  }
};

/**
 * 红包管理页面
 * 功能: 统计卡片 + 状态筛选 + 红包列表表格 + 展开领取记录
 */
const RedPacketManagement: FC = () => {
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(true);
  const [stats, setStats] = useState<RedPacketStats | null>(null);
  const [redPackets, setRedPackets] = useState<RedPacket[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [statusFilter, setStatusFilter] = useState<number>(-1);
  const [expandedClaims, setExpandedClaims] = useState<
    Record<number, RedPacketClaim[]>
  >({});
  const { message } = App.useApp();

  /** 加载统计数据 */
  useEffect(() => {
    const loadStats = async () => {
      setStatsLoading(true);
      try {
        const data = await getRedPacketStats();
        setStats(data);
      } catch {
        message.error('加载红包统计失败');
      } finally {
        setStatsLoading(false);
      }
    };
    loadStats();
  }, [message]);

  /** 加载红包列表 */
  const loadRedPackets = useCallback(async () => {
    setLoading(true);
    try {
      const params: RedPacketListParams = { page, size: pageSize };
      if (statusFilter !== -1) {
        params.status = statusFilter;
      }
      const resp = await getRedPacketList(params);
      setRedPackets(resp.list);
      setTotal(resp.total);
    } catch {
      message.error('加载红包列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, statusFilter, message]);

  useEffect(() => {
    loadRedPackets();
  }, [loadRedPackets]);

  /** 状态筛选变更 */
  const handleStatusChange = (value: number) => {
    setStatusFilter(value);
    setPage(1);
  };

  /** 分页变更 */
  const handleTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current ?? 1);
    setPageSize(pagination.pageSize ?? DEFAULT_PAGE_SIZE);
  };

  /** 展开行加载领取记录 */
  const handleExpand = (expanded: boolean, record: RedPacket) => {
    if (expanded && !expandedClaims[record.id]) {
      const loadDetail = async () => {
        try {
          const detail = await getRedPacketDetail(record.id);
          setExpandedClaims((prev) => ({
            ...prev,
            [record.id]: detail.claims,
          }));
        } catch {
          message.error('加载领取记录失败');
        }
      };
      loadDetail();
    }
  };

  /** 领取记录子表格列定义 */
  const claimColumns: ColumnsType<RedPacketClaim> = [
    {
      title: '领取者',
      key: 'claimer',
      width: 150,
      render: (_: unknown, record: RedPacketClaim) =>
        record.claimer_name || record.claimer_username || '-',
    },
    {
      title: '领取金额',
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      render: (val: string) => formatAmount(val),
    },
    {
      title: '领取时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (val: string) => formatTime(val),
    },
  ];

  /** 展开行渲染 - 领取记录子表格 */
  const expandedRowRender = (record: RedPacket) => {
    const claims = expandedClaims[record.id];

    if (!claims) {
      return (
        <div style={{ padding: 24, textAlign: 'center' }}>
          <Spin size="small" />
          <span style={{ marginLeft: 8 }}>加载领取记录中...</span>
        </div>
      );
    }

    if (claims.length === 0) {
      return (
        <div style={{ padding: 16, textAlign: 'center', color: '#999' }}>
          暂无领取记录
        </div>
      );
    }

    return (
      <Table<RedPacketClaim>
        columns={claimColumns}
        dataSource={claims}
        rowKey="id"
        pagination={false}
        size="small"
      />
    );
  };

  /** 主表格列定义 */
  const columns: ColumnsType<RedPacket> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 70,
    },
    {
      title: '发送者',
      key: 'sender',
      width: 120,
      render: (_: unknown, record: RedPacket) =>
        record.sender_name || record.sender_username || '-',
    },
    {
      title: '币种',
      dataIndex: 'currency',
      key: 'currency',
      width: 80,
      render: (val: string) => (
        <Tag color={CURRENCY_COLORS[val] ?? 'default'}>{val}</Tag>
      ),
    },
    {
      title: '总金额',
      dataIndex: 'total_amount',
      key: 'total_amount',
      width: 120,
      render: (val: string) => formatAmount(val),
    },
    {
      title: '个数',
      dataIndex: 'total_count',
      key: 'total_count',
      width: 70,
      align: 'center',
    },
    {
      title: '已领',
      key: 'claimed_progress',
      width: 80,
      align: 'center',
      render: (_: unknown, record: RedPacket) => (
        <span>
          {record.claimed_count}/{record.total_count}
        </span>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 80,
      render: (val: number) => {
        const isRandom = val === RedPacketType.Random;
        return (
          <Tag color={isRandom ? 'volcano' : 'geekblue'}>
            {getRedPacketTypeName(val)}
          </Tag>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (val: number) => {
        const tag = getRedPacketStatusTag(val);
        return <Tag color={tag.color}>{tag.text}</Tag>;
      },
    },
    {
      title: '过期时间',
      dataIndex: 'expire_at',
      key: 'expire_at',
      width: 170,
      render: (val: string) => formatTime(val),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (val: string) => formatTime(val),
    },
  ];

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="红包总数"
              value={stats?.total_count ?? 0}
              loading={statsLoading}
              prefix={<GiftOutlined />}
              suffix="个"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="红包总金额"
              value={stats?.total_amount ?? '0.00'}
              loading={statsLoading}
              prefix={<DollarOutlined />}
              precision={2}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="已领完"
              value={stats?.claimed_count ?? 0}
              loading={statsLoading}
              prefix={<CheckCircleOutlined />}
              suffix="个"
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="已过期"
              value={stats?.expired_count ?? 0}
              loading={statsLoading}
              prefix={<ClockCircleOutlined />}
              suffix="个"
              styles={{ content: { color: '#999' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选栏 */}
      <Space style={{ marginBottom: 16 }}>
        <span>状态筛选:</span>
        <Select
          value={statusFilter}
          onChange={handleStatusChange}
          options={STATUS_OPTIONS}
          style={{ width: 140 }}
        />
      </Space>

      {/* 红包列表表格 */}
      <Table<RedPacket>
        columns={columns}
        dataSource={redPackets}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize: pageSize,
          total: total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        onChange={handleTableChange}
        expandable={{
          expandedRowRender,
          onExpand: handleExpand,
          rowExpandable: (record) => record.claimed_count > 0,
        }}
        scroll={{ x: 1200 }}
        size="middle"
      />
    </div>
  );
};

export default RedPacketManagement;
