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
  InputNumber,
  Button,
  App,
} from 'antd';
import {
  FundOutlined,
  DollarOutlined,
  RiseOutlined,
  PercentageOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { FinanceInvestment, FinanceStats, FinanceListParams } from '@/types/finance.ts';
import { FinanceInvestmentStatus } from '@/types/finance.ts';
import { getFinanceInvestments, getFinanceStats, updateFinanceRate } from '@/api/finance.ts';
import { formatAmount, formatTime } from '@/utils/format.ts';
import { DEFAULT_PAGE_SIZE } from '@/utils/constants.ts';
import { useAuthStore } from '@/stores/authStore.ts';

/** 投资状态筛选选项 */
const STATUS_OPTIONS = [
  { label: '全部', value: -1 },
  { label: '持有中', value: FinanceInvestmentStatus.Active },
  { label: '已取出', value: FinanceInvestmentStatus.Withdrawn },
];

/** 获取投资状态 Tag 颜色和文本 */
const getInvestmentStatusTag = (
  status: number,
): { color: string; text: string } => {
  switch (status) {
    case FinanceInvestmentStatus.Active:
      return { color: 'green', text: '持有中' };
    case FinanceInvestmentStatus.Withdrawn:
      return { color: 'default', text: '已取出' };
    default:
      return { color: 'default', text: '未知' };
  }
};

/**
 * 余额宝管理页面
 * 功能: 统计卡片 + 利率调整 (SuperAdmin) + 投资列表表格
 */
const FinanceManagement: FC = () => {
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(true);
  const [stats, setStats] = useState<FinanceStats | null>(null);
  const [investments, setInvestments] = useState<FinanceInvestment[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [statusFilter, setStatusFilter] = useState<number>(-1);
  const [newRate, setNewRate] = useState<number | null>(null);
  const [rateUpdating, setRateUpdating] = useState(false);
  const { message } = App.useApp();
  const user = useAuthStore((state) => state.user);

  /** 判断是否为超级管理员 */
  const isSuperAdmin = user?.role === 'super_admin';

  /** 加载统计数据 */
  const loadStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const data = await getFinanceStats();
      setStats(data);
      setNewRate(parseFloat(data.current_annual_rate));
    } catch {
      message.error('加载余额宝统计失败');
    } finally {
      setStatsLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadStats();
  }, [loadStats]);

  /** 加载投资列表 */
  const loadInvestments = useCallback(async () => {
    setLoading(true);
    try {
      const params: FinanceListParams = { page, size: pageSize };
      if (statusFilter !== -1) {
        params.status = statusFilter;
      }
      const resp = await getFinanceInvestments(params);
      setInvestments(resp.list);
      setTotal(resp.total);
    } catch {
      message.error('加载投资列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, statusFilter, message]);

  useEffect(() => {
    loadInvestments();
  }, [loadInvestments]);

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

  /** 更新利率 */
  const handleUpdateRate = async () => {
    if (newRate === null || newRate <= 0) {
      message.warning('请输入有效的年化利率');
      return;
    }
    setRateUpdating(true);
    try {
      await updateFinanceRate(newRate);
      message.success(`年化利率已更新为 ${newRate.toFixed(2)}%`);
      await loadStats();
    } catch {
      message.error('利率更新失败');
    } finally {
      setRateUpdating(false);
    }
  };

  /** 表格列定义 */
  const columns: ColumnsType<FinanceInvestment> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 70,
    },
    {
      title: '用户',
      key: 'user',
      width: 120,
      render: (_: unknown, record: FinanceInvestment) =>
        record.display_name || (record.username ? `@${record.username}` : '-'),
    },
    {
      title: '投资金额',
      dataIndex: 'amount',
      key: 'amount',
      width: 130,
      render: (val: string) => (
        <span style={{ fontWeight: 500 }}>{formatAmount(val)} USDT</span>
      ),
    },
    {
      title: '年化利率',
      dataIndex: 'annual_rate',
      key: 'annual_rate',
      width: 100,
      align: 'center',
      render: (val: string) => (
        <span style={{ color: '#1677ff' }}>{val}%</span>
      ),
    },
    {
      title: '累计收益',
      dataIndex: 'total_profit',
      key: 'total_profit',
      width: 130,
      render: (val: string) => (
        <span style={{ color: '#52c41a' }}>+{formatAmount(val)}</span>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (val: number) => {
        const tag = getInvestmentStatusTag(val);
        return <Tag color={tag.color}>{tag.text}</Tag>;
      },
    },
    {
      title: '开始计息',
      dataIndex: 'interest_start',
      key: 'interest_start',
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
              title="总投资金额"
              value={stats?.total_invest_amount ?? '0.00'}
              loading={statsLoading}
              prefix={<FundOutlined />}
              precision={2}
              suffix="USDT"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃投资数"
              value={stats?.active_invest_count ?? 0}
              loading={statsLoading}
              prefix={<DollarOutlined />}
              suffix="笔"
              styles={{ content: { color: '#1677ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="累计派息"
              value={stats?.total_profit_paid ?? '0.00'}
              loading={statsLoading}
              prefix={<RiseOutlined />}
              precision={2}
              suffix="USDT"
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="当前年化利率"
              value={stats?.current_annual_rate ?? '0.00'}
              loading={statsLoading}
              prefix={<PercentageOutlined />}
              precision={2}
              suffix="%"
              styles={{ content: { color: '#fa8c16' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 利率调整区域 - 仅 SuperAdmin 可见 */}
      {isSuperAdmin && (
        <Card
          title="利率调整"
          size="small"
          style={{ marginBottom: 24 }}
        >
          <Space>
            <span>年化利率:</span>
            <Space.Compact>
              <InputNumber
                value={newRate}
                onChange={(val) => setNewRate(val)}
                min={0}
                max={100}
                step={0.1}
                precision={2}
                style={{ width: 150 }}
                placeholder="请输入年化利率"
              />
              <Button disabled style={{ cursor: 'default' }}>%</Button>
            </Space.Compact>
            <Button
              type="primary"
              onClick={handleUpdateRate}
              loading={rateUpdating}
              disabled={newRate === null || newRate <= 0}
            >
              更新利率
            </Button>
          </Space>
        </Card>
      )}

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

      {/* 投资列表表格 */}
      <Table<FinanceInvestment>
        columns={columns}
        dataSource={investments}
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
        scroll={{ x: 1100 }}
        size="middle"
      />
    </div>
  );
};

export default FinanceManagement;
