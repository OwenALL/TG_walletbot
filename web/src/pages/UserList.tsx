import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Table,
  Input,
  Tag,
  Button,
  Space,
  Popconfirm,
  Select,
  App,
} from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { User } from '@/types/user.ts';
import { UserStatus } from '@/types/user.ts';
import { getUserList, updateUserStatus } from '@/api/users.ts';
import { formatAmount, formatTime, getUserStatusTag } from '@/utils/format.ts';
import { USER_STATUS_OPTIONS, DEFAULT_PAGE_SIZE } from '@/utils/constants.ts';
import UserDetailDrawer from './UserDetail.tsx';

/**
 * 用户管理页面
 * 功能: 搜索、状态筛选、冻结/解冻、查看详情
 */
const UserList: FC = () => {
  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [keyword, setKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<number>(0);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailUserId, setDetailUserId] = useState<number | null>(null);
  const { message } = App.useApp();

  /** 加载用户列表 */
  const loadUsers = useCallback(() => {
    setLoading(true);
    const loadData = async () => {
      try {
        const resp = await getUserList({ page, size: pageSize, keyword, status: statusFilter > 0 ? statusFilter as 1 | 2 | 3 : undefined });
        setUsers(resp.list);
        setTotal(resp.total);
      } catch {
        message.error('加载用户列表失败');
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, [page, pageSize, keyword, statusFilter, message]);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  /** 冻结/解冻用户 */
  const handleToggleStatus = async (user: User) => {
    const newStatus =
      user.status === UserStatus.Active ? UserStatus.Frozen : UserStatus.Active;
    const actionText = newStatus === UserStatus.Frozen ? '冻结' : '解冻';

    try {
      await updateUserStatus({ id: user.id, status: newStatus });
      message.success(`${actionText}成功`);
      loadUsers();
    } catch {
      message.error(`${actionText}失败`);
    }
  };

  /** 打开用户详情 */
  const handleViewDetail = (userId: number) => {
    setDetailUserId(userId);
    setDetailOpen(true);
  };

  /** 分页变更 */
  const handleTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current ?? 1);
    setPageSize(pagination.pageSize ?? DEFAULT_PAGE_SIZE);
  };

  /** 表格列定义 */
  const columns: ColumnsType<User> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: '昵称',
      key: 'name',
      width: 120,
      render: (_: unknown, record: User) =>
        `${record.first_name} ${record.last_name}`.trim() || '-',
    },
    {
      title: '@username',
      dataIndex: 'username',
      key: 'username',
      width: 130,
      render: (val: string) => (val ? `@${val}` : '-'),
    },
    {
      title: 'Telegram ID',
      dataIndex: 'telegram_id',
      key: 'telegram_id',
      width: 130,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (val: number) => {
        const tag = getUserStatusTag(val);
        return <Tag color={tag.color}>{tag.text}</Tag>;
      },
    },
    {
      title: 'USDT 余额',
      key: 'usdt',
      width: 120,
      render: (_: unknown, record: User) => formatAmount(record.usdt_balance ?? '0'),
    },
    {
      title: 'TRX 余额',
      key: 'trx',
      width: 120,
      render: (_: unknown, record: User) => formatAmount(record.trx_balance ?? '0'),
    },
    {
      title: 'CNY 余额',
      key: 'cny',
      width: 120,
      render: (_: unknown, record: User) => formatAmount(record.cny_balance ?? '0', 'CNY'),
    },
    {
      title: '注册时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (val: string) => formatTime(val),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      fixed: 'right',
      render: (_: unknown, record: User) => {
        const isFrozen = record.status === UserStatus.Frozen;
        const isActive = record.status === UserStatus.Active;
        return (
          <Space size="small">
            <Button
              type="link"
              size="small"
              onClick={() => handleViewDetail(record.id)}
            >
              详情
            </Button>
            {(isActive || isFrozen) && (
              <Popconfirm
                title={`确认${isFrozen ? '解冻' : '冻结'}该用户？`}
                description={`用户: @${record.username || record.telegram_id}`}
                onConfirm={() => handleToggleStatus(record)}
                okText="确认"
                cancelText="取消"
              >
                <Button
                  type="link"
                  size="small"
                  danger={!isFrozen}
                >
                  {isFrozen ? '解冻' : '冻结'}
                </Button>
              </Popconfirm>
            )}
          </Space>
        );
      },
    },
  ];

  return (
    <div>
      {/* 搜索和筛选区域 */}
      <div className="filter-form">
        <Space wrap>
          <Input
            placeholder="搜索 Telegram ID / @username"
            prefix={<SearchOutlined />}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onPressEnter={() => {
              setPage(1);
              loadUsers();
            }}
            allowClear
            style={{ width: 280 }}
          />
          <Select
            value={statusFilter}
            onChange={(val) => {
              setStatusFilter(val);
              setPage(1);
            }}
            options={USER_STATUS_OPTIONS.map((opt) => ({
              label: opt.label,
              value: opt.value,
            }))}
            style={{ width: 120 }}
          />
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              setKeyword('');
              setStatusFilter(0);
              setPage(1);
            }}
          >
            重置
          </Button>
        </Space>
      </div>

      {/* 用户表格 */}
      <Table<User>
        columns={columns}
        dataSource={users}
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

      {/* 用户详情抽屉 */}
      <UserDetailDrawer
        open={detailOpen}
        userId={detailUserId}
        onClose={() => setDetailOpen(false)}
      />
    </div>
  );
};

export default UserList;
