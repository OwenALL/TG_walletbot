import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Table,
  Select,
  DatePicker,
  Space,
  Tag,
  Input,
  Button,
  App,
} from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { Dayjs } from 'dayjs';
import type { Transaction } from '@/types/transaction.ts';
import { getTransactionList } from '@/api/transactions.ts';
import {
  formatAmount,
  formatTime,
  getTransactionTypeName,
  isIncomeType,
} from '@/utils/format.ts';
import {
  CURRENCY_OPTIONS,
  TRANSACTION_TYPE_OPTIONS,
  DEFAULT_PAGE_SIZE,
} from '@/utils/constants.ts';

const { RangePicker } = DatePicker;

/**
 * 交易管理页面
 * 功能: 按类型/币种/时间范围筛选交易流水
 */
const TransactionList: FC = () => {
  const [loading, setLoading] = useState(false);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [typeFilter, setTypeFilter] = useState<string>('');
  const [currencyFilter, setCurrencyFilter] = useState<string>('');
  const [userKeyword, setUserKeyword] = useState('');
  const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null] | null>(null);
  const { message } = App.useApp();

  /** 加载交易列表 */
  const loadTransactions = useCallback(() => {
    setLoading(true);
    const loadData = async () => {
      try {
        const params: Record<string, unknown> = {
          page,
          size: pageSize,
          type: typeFilter || undefined,
          currency: currencyFilter || undefined,
          start_time: dateRange?.[0]?.format('YYYY-MM-DD') || undefined,
          end_time: dateRange?.[1]?.format('YYYY-MM-DD') || undefined,
        };
        /* userKeyword 为纯数字时作为 user_id 筛选 */
        const trimmed = userKeyword.trim();
        if (trimmed && /^\d+$/.test(trimmed)) {
          params.user_id = parseInt(trimmed, 10);
        }
        const resp = await getTransactionList(params as Parameters<typeof getTransactionList>[0]);
        setTransactions(resp.list);
        setTotal(resp.total);
      } catch {
        message.error('加载交易列表失败');
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, [page, pageSize, typeFilter, currencyFilter, userKeyword, dateRange, message]);

  useEffect(() => {
    loadTransactions();
  }, [loadTransactions]);

  /** 分页变更 */
  const handleTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current ?? 1);
    setPageSize(pagination.pageSize ?? DEFAULT_PAGE_SIZE);
  };

  /** 重置筛选条件 */
  const handleReset = () => {
    setTypeFilter('');
    setCurrencyFilter('');
    setUserKeyword('');
    setDateRange(null);
    setPage(1);
  };

  /** 表格列定义 */
  const columns: ColumnsType<Transaction> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '用户',
      key: 'user',
      width: 130,
      render: (_: unknown, record: Transaction) =>
        record.username ? `@${record.username}` : `ID:${record.user_id}`,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 110,
      render: (val: string) => {
        const income = isIncomeType(val);
        return (
          <Tag color={income ? 'green' : 'red'}>
            {getTransactionTypeName(val)}
          </Tag>
        );
      },
    },
    {
      title: '币种',
      dataIndex: 'currency',
      key: 'currency',
      width: 80,
      render: (val: string) => <Tag>{val}</Tag>,
    },
    {
      title: '金额',
      key: 'amount',
      width: 140,
      render: (_: unknown, record: Transaction) => {
        const income = isIncomeType(record.type);
        return (
          <span style={{ color: income ? '#52c41a' : '#ff4d4f', fontWeight: 500 }}>
            {income ? '+' : '-'}{formatAmount(record.amount)}
          </span>
        );
      },
    },
    {
      title: '手续费',
      dataIndex: 'fee',
      key: 'fee',
      width: 100,
      render: (val: string) => {
        const num = parseFloat(val);
        return num > 0 ? formatAmount(val) : '-';
      },
    },
    {
      title: '交易前余额',
      dataIndex: 'balance_before',
      key: 'balance_before',
      width: 130,
      render: (val: string) => formatAmount(val),
    },
    {
      title: '交易后余额',
      dataIndex: 'balance_after',
      key: 'balance_after',
      width: 130,
      render: (val: string) => formatAmount(val),
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (val: string) => formatTime(val),
    },
  ];

  return (
    <div>
      {/* 筛选区域 */}
      <div className="filter-form">
        <Space wrap>
          <Input
            placeholder="搜索用户 (ID / @username)"
            prefix={<SearchOutlined />}
            value={userKeyword}
            onChange={(e) => setUserKeyword(e.target.value)}
            onPressEnter={() => {
              setPage(1);
              loadTransactions();
            }}
            allowClear
            style={{ width: 220 }}
          />
          <Select
            value={typeFilter}
            onChange={(val) => {
              setTypeFilter(val);
              setPage(1);
            }}
            options={TRANSACTION_TYPE_OPTIONS.map((opt) => ({
              label: opt.label,
              value: opt.value,
            }))}
            placeholder="交易类型"
            style={{ width: 130 }}
          />
          <Select
            value={currencyFilter}
            onChange={(val) => {
              setCurrencyFilter(val);
              setPage(1);
            }}
            options={CURRENCY_OPTIONS.map((opt) => ({
              label: opt.label,
              value: opt.value,
            }))}
            placeholder="币种"
            style={{ width: 100 }}
          />
          <RangePicker
            value={dateRange}
            onChange={(dates) => {
              setDateRange(dates);
              setPage(1);
            }}
          />
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            重置
          </Button>
        </Space>
      </div>

      {/* 交易表格 */}
      <Table<Transaction>
        columns={columns}
        dataSource={transactions}
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
        scroll={{ x: 1200 }}
        size="middle"
      />
    </div>
  );
};

export default TransactionList;
