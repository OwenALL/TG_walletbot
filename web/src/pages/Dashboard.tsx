import { type FC, useState, useEffect } from 'react';
import {
  Row,
  Col,
  Card,
  Statistic,
  Table,
  Spin,
  App,
} from 'antd';
import {
  UserOutlined,
  TransactionOutlined,
  WalletOutlined,
  AuditOutlined,
  ArrowUpOutlined,
} from '@ant-design/icons';
import { Line } from '@ant-design/charts';
import type { ColumnsType } from 'antd/es/table';
import type { DashboardStats, TrendDataPoint } from '@/types/config.ts';
import type { Transaction } from '@/types/transaction.ts';
import { getDashboardStats, getDashboardTrend } from '@/api/configs.ts';
import { getTransactionList } from '@/api/transactions.ts';
import { formatAmount, formatTime, getTransactionTypeName, isIncomeType } from '@/utils/format.ts';

/**
 * 仪表盘页面
 * 包含: 4 个统计卡片 + 交易趋势折线图 + 最近 10 条交易列表
 */
const Dashboard: FC = () => {
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [trend, setTrend] = useState<TrendDataPoint[]>([]);
  const [recentTx, setRecentTx] = useState<Transaction[]>([]);
  const { message } = App.useApp();

  useEffect(() => {
    const loadData = async () => {
      try {
        const [statsData, trendData, txData] = await Promise.all([
          getDashboardStats(),
          getDashboardTrend(7),
          getTransactionList({ page: 1, size: 10 }),
        ]);
        setStats(statsData);
        setTrend(trendData);
        setRecentTx(txData.list);
      } catch {
        message.error('加载仪表盘数据失败');
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, [message]);

  /** 交易趋势图配置 */
  const trendConfig = {
    data: trend.map((item) => ({
      date: item.date,
      value: item.count,
    })),
    xField: 'date',
    yField: 'value',
    smooth: true,
    point: { size: 3 },
    label: false,
    style: {
      lineWidth: 2,
      stroke: '#1677ff',
    },
    axis: {
      x: { title: '日期' },
      y: { title: '交易笔数' },
    },
    height: 300,
  };

  /** 最近交易表格列定义 */
  const columns: ColumnsType<Transaction> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
      width: 120,
      render: (val: string) => val ? `@${val}` : '-',
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (val: string) => getTransactionTypeName(val),
    },
    {
      title: '金额',
      key: 'amount',
      width: 150,
      render: (_: unknown, record: Transaction) => {
        const income = isIncomeType(record.type);
        return (
          <span style={{ color: income ? '#52c41a' : '#ff4d4f' }}>
            {income ? '+' : '-'}{formatAmount(record.amount, record.currency)}
          </span>
        );
      },
    },
    {
      title: '交易后余额',
      dataIndex: 'balance_after',
      key: 'balance_after',
      width: 150,
      render: (val: string, record: Transaction) => formatAmount(val, record.currency),
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (val: string) => formatTime(val),
    },
  ];

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 400 }}>
        <Spin size="large" description="加载中..." />
      </div>
    );
  }

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="dashboard-stats-card">
            <Statistic
              title="总用户数"
              value={stats?.total_users ?? 0}
              prefix={<UserOutlined />}
              suffix={
                <span style={{ fontSize: 14, color: '#52c41a' }}>
                  <ArrowUpOutlined /> {stats?.today_new_users ?? 0} 今日
                </span>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="dashboard-stats-card">
            <Statistic
              title="今日交易"
              value={stats?.today_transactions ?? 0}
              prefix={<TransactionOutlined />}
              suffix="笔"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="dashboard-stats-card">
            <Statistic
              title="USDT 总余额"
              value={stats?.total_usdt_balance ?? '0'}
              prefix={<WalletOutlined />}
              precision={2}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="dashboard-stats-card">
            <Statistic
              title="待审核提币"
              value={stats?.pending_withdrawals ?? 0}
              prefix={<AuditOutlined />}
              styles={{
                content: {
                  color: (stats?.pending_withdrawals ?? 0) > 0 ? '#cf1322' : undefined,
                },
              }}
              suffix="笔"
            />
          </Card>
        </Col>
      </Row>

      {/* 交易趋势图 */}
      <Card title="近 7 日交易趋势" style={{ marginBottom: 24 }}>
        <Line {...trendConfig} />
      </Card>

      {/* 最近交易列表 */}
      <Card title="最近交易">
        <Table<Transaction>
          columns={columns}
          dataSource={recentTx}
          rowKey="id"
          pagination={false}
          size="middle"
          scroll={{ x: 800 }}
        />
      </Card>
    </div>
  );
};

export default Dashboard;
