import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Table,
  Select,
  DatePicker,
  Space,
  Tag,
  Button,
  Typography,
  App,
} from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { Dayjs } from 'dayjs';
import { formatTime } from '@/utils/format.ts';
import { DEFAULT_PAGE_SIZE } from '@/utils/constants.ts';
import { getAuditLogs } from '@/api/auditLogs.ts';
import type { AuditLog, AuditLogListParams } from '@/types/auditLog.ts';

const { RangePicker } = DatePicker;
const { Paragraph } = Typography;

/** 操作类型选项列表 */
const ACTION_OPTIONS = [
  { label: '全部', value: '' },
  { label: '登录', value: 'login' },
  { label: '更新配置', value: 'update_config' },
  { label: '批准提币', value: 'approve_withdrawal' },
  { label: '拒绝提币', value: 'reject_withdrawal' },
  { label: '更新用户状态', value: 'update_user_status' },
  { label: '更新汇率', value: 'update_exchange_rate' },
] as const;

/** 目标类型 Tag 颜色映射 */
const TARGET_TYPE_COLORS: Record<string, string> = {
  user: 'blue',
  withdrawal: 'orange',
  config: 'purple',
  exchange_rate: 'cyan',
  admin: 'geekblue',
};

/**
 * 尝试格式化 JSON 详情字符串
 * 如果是有效 JSON 则美化输出，否则原样返回
 */
const formatDetail = (detail: string): string => {
  if (!detail) return '-';
  try {
    const parsed = JSON.parse(detail);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return detail;
  }
};

/**
 * 审计日志页面
 * 功能: 展示管理员操作日志，支持按操作类型和时间范围筛选
 */
const AuditLogs: FC = () => {
  const [loading, setLoading] = useState(false);
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [actionFilter, setActionFilter] = useState<string>('');
  const [dateRange, setDateRange] = useState<
    [Dayjs | null, Dayjs | null] | null
  >(null);
  const { message } = App.useApp();

  /** 加载审计日志列表 */
  const loadLogs = useCallback(async () => {
    setLoading(true);
    try {
      const params: AuditLogListParams = {
        page,
        size: pageSize,
        action: actionFilter || undefined,
        start_time: dateRange?.[0]?.format('YYYY-MM-DD') || undefined,
        end_time: dateRange?.[1]?.format('YYYY-MM-DD') || undefined,
      };
      const resp = await getAuditLogs(params);
      setLogs(resp.list ?? []);
      setTotal(resp.total ?? 0);
    } catch {
      message.error('加载审计日志失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, actionFilter, dateRange, message]);

  useEffect(() => {
    loadLogs();
  }, [loadLogs]);

  /** 分页变更 */
  const handleTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current ?? 1);
    setPageSize(pagination.pageSize ?? DEFAULT_PAGE_SIZE);
  };

  /** 重置筛选条件 */
  const handleReset = () => {
    setActionFilter('');
    setDateRange(null);
    setPage(1);
  };

  /** 表格列定义 */
  const columns: ColumnsType<AuditLog> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 70,
    },
    {
      title: '管理员',
      dataIndex: 'admin_username',
      key: 'admin_username',
      width: 120,
      render: (val: string) => val || '-',
    },
    {
      title: '操作类型',
      dataIndex: 'action',
      key: 'action',
      width: 150,
      render: (val: string) => {
        const opt = ACTION_OPTIONS.find((o) => o.value === val);
        return <Tag color="processing">{opt ? opt.label : val}</Tag>;
      },
    },
    {
      title: '目标类型',
      dataIndex: 'target_type',
      key: 'target_type',
      width: 120,
      render: (val: string) => {
        if (!val) return '-';
        const color = TARGET_TYPE_COLORS[val] ?? 'default';
        return <Tag color={color}>{val}</Tag>;
      },
    },
    {
      title: '目标 ID',
      dataIndex: 'target_id',
      key: 'target_id',
      width: 100,
      render: (val: number | null) => (val !== null && val !== undefined) ? val : '-',
    },
    {
      title: '详情',
      dataIndex: 'detail',
      key: 'detail',
      width: 260,
      render: (val: string) => {
        const formatted = formatDetail(val);
        if (formatted === '-') return '-';
        return (
          <Paragraph
            ellipsis={{ rows: 2, expandable: true, symbol: '展开' }}
            style={{ marginBottom: 0, fontSize: 12, whiteSpace: 'pre-wrap' }}
          >
            {formatted}
          </Paragraph>
        );
      },
    },
    {
      title: 'IP 地址',
      dataIndex: 'ip_address',
      key: 'ip_address',
      width: 140,
      render: (val: string) => val || '-',
    },
    {
      title: '操作时间',
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
          <Select
            value={actionFilter}
            onChange={(val) => {
              setActionFilter(val);
              setPage(1);
            }}
            options={ACTION_OPTIONS.map((opt) => ({
              label: opt.label,
              value: opt.value,
            }))}
            placeholder="操作类型"
            style={{ width: 160 }}
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

      {/* 审计日志表格 */}
      <Table<AuditLog>
        columns={columns}
        dataSource={logs}
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

export default AuditLogs;
