import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Table,
  Tabs,
  Tag,
  Button,
  Space,
  Popconfirm,
  Modal,
  Input,
  App,
} from 'antd';
import { CheckOutlined, CloseOutlined, SendOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { Withdrawal } from '@/types/transaction.ts';
import { WithdrawalStatus } from '@/types/transaction.ts';
import { getWithdrawalList, approveWithdrawal, rejectWithdrawal, completeWithdrawal } from '@/api/withdrawals.ts';
import {
  formatAmount,
  formatTime,
  shortenAddress,
  getWithdrawalStatusTag,
} from '@/utils/format.ts';
import { DEFAULT_PAGE_SIZE } from '@/utils/constants.ts';

/**
 * 提币审核页面
 * 功能: Tab 切换待审核/已处理，批准/拒绝操作
 */
const WithdrawReview: FC = () => {
  const [activeTab, setActiveTab] = useState<string>('pending');
  const [loading, setLoading] = useState(false);
  const [withdrawals, setWithdrawals] = useState<Withdrawal[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [rejectModalOpen, setRejectModalOpen] = useState(false);
  const [rejectingId, setRejectingId] = useState<number | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [completeModalOpen, setCompleteModalOpen] = useState(false);
  const [completingId, setCompletingId] = useState<number | null>(null);
  const [txHashInput, setTxHashInput] = useState('');
  const { message } = App.useApp();

  /** 加载提币列表 */
  const loadWithdrawals = useCallback(() => {
    setLoading(true);
    const loadData = async () => {
      try {
        const params = {
          page,
          size: pageSize,
          status: activeTab === 'pending' ? WithdrawalStatus.Pending : undefined,
        };
        const resp = await getWithdrawalList(params);
        setWithdrawals(resp.list);
        setTotal(resp.total);
      } catch {
        message.error('加载提币列表失败');
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, [activeTab, page, pageSize, message]);

  useEffect(() => {
    loadWithdrawals();
  }, [loadWithdrawals]);

  /** 批准提币 */
  const handleApprove = async (id: number) => {
    try {
      await approveWithdrawal({ id });
      message.success('已批准提币申请');
      loadWithdrawals();
    } catch {
      message.error('操作失败');
    }
  };

  /** 打开拒绝弹窗 */
  const handleOpenReject = (id: number) => {
    setRejectingId(id);
    setRejectReason('');
    setRejectModalOpen(true);
  };

  /** 确认拒绝 */
  const handleConfirmReject = async () => {
    if (!rejectReason.trim()) {
      message.warning('请输入拒绝原因');
      return;
    }
    if (rejectingId === null) return;

    try {
      await rejectWithdrawal({ id: rejectingId, reason: rejectReason });
      message.success('已拒绝提币申请');
      setRejectModalOpen(false);
      setRejectingId(null);
      setRejectReason('');
      loadWithdrawals();
    } catch {
      message.error('操作失败');
    }
  };

  /** 打开手动完成弹窗 */
  const handleOpenComplete = (id: number) => {
    setCompletingId(id);
    setTxHashInput('');
    setCompleteModalOpen(true);
  };

  /** 确认手动完成提币 */
  const handleConfirmComplete = async () => {
    if (!txHashInput.trim()) {
      message.warning('请输入交易哈希');
      return;
    }
    if (completingId === null) return;

    try {
      await completeWithdrawal({ id: completingId, tx_hash: txHashInput.trim() });
      message.success('提币已标记为完成');
      setCompleteModalOpen(false);
      setCompletingId(null);
      setTxHashInput('');
      loadWithdrawals();
    } catch {
      message.error('操作失败');
    }
  };

  /** Tab 切换 */
  const handleTabChange = (key: string) => {
    setActiveTab(key);
    setPage(1);
  };

  /** 分页变更 */
  const handleTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current ?? 1);
    setPageSize(pagination.pageSize ?? DEFAULT_PAGE_SIZE);
  };

  /** 表格列定义 */
  const columns: ColumnsType<Withdrawal> = [
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
      render: (_: unknown, record: Withdrawal) =>
        record.username ? `@${record.username}` : `ID:${record.user_id}`,
    },
    {
      title: '币种',
      dataIndex: 'currency',
      key: 'currency',
      width: 80,
      render: (val: string) => <Tag>{val}</Tag>,
    },
    {
      title: '提币金额',
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      render: (val: string) => formatAmount(val),
    },
    {
      title: '手续费',
      dataIndex: 'fee',
      key: 'fee',
      width: 80,
      render: (val: string) => formatAmount(val),
    },
    {
      title: '实际到账',
      dataIndex: 'actual_amount',
      key: 'actual_amount',
      width: 120,
      render: (val: string) => formatAmount(val),
    },
    {
      title: '提币地址',
      dataIndex: 'to_address',
      key: 'to_address',
      width: 160,
      render: (val: string) => (
        <span title={val} style={{ cursor: 'pointer' }}>
          {shortenAddress(val)}
        </span>
      ),
    },
    {
      title: '申请时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (val: string) => formatTime(val),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (val: number) => {
        const tag = getWithdrawalStatusTag(val);
        return <Tag color={tag.color}>{tag.text}</Tag>;
      },
    },
  ];

  /* 待审核状态添加操作列 */
  if (activeTab === 'pending') {
    columns.push({
      title: '操作',
      key: 'action',
      width: 150,
      fixed: 'right',
      render: (_: unknown, record: Withdrawal) => (
        <Space size="small">
          <Popconfirm
            title="确认批准该提币申请？"
            description={`${formatAmount(record.amount)} ${record.currency}`}
            onConfirm={() => handleApprove(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button
              type="link"
              size="small"
              icon={<CheckOutlined />}
              style={{ color: '#52c41a' }}
            >
              批准
            </Button>
          </Popconfirm>
          <Button
            type="link"
            size="small"
            danger
            icon={<CloseOutlined />}
            onClick={() => handleOpenReject(record.id)}
          >
            拒绝
          </Button>
        </Space>
      ),
    });
  } else {
    /* 已处理状态显示审核信息 */
    columns.push(
      {
        title: '审核时间',
        dataIndex: 'reviewed_at',
        key: 'reviewed_at',
        width: 170,
        render: (val: string | null) => formatTime(val),
      },
      {
        title: '备注',
        dataIndex: 'review_note',
        key: 'review_note',
        width: 160,
        ellipsis: true,
        render: (val: string) => val || '-',
      },
      {
        title: '操作',
        key: 'action',
        width: 120,
        fixed: 'right',
        render: (_: unknown, record: Withdrawal) => {
          /* 仅处理中 (Processing) 状态显示"完成"按钮 */
          if (record.status !== WithdrawalStatus.Processing) {
            return null;
          }
          return (
            <Button
              type="link"
              size="small"
              icon={<SendOutlined />}
              onClick={() => handleOpenComplete(record.id)}
            >
              完成
            </Button>
          );
        },
      },
    );
  }

  return (
    <div>
      <Tabs
        activeKey={activeTab}
        onChange={handleTabChange}
        items={[
          {
            key: 'pending',
            label: `待审核 (${activeTab === 'pending' ? total : '...'})`,
          },
          { key: 'processed', label: '已处理' },
        ]}
      />

      <Table<Withdrawal>
        columns={columns}
        dataSource={withdrawals}
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
        scroll={{ x: 1400 }}
        size="middle"
      />

      {/* 拒绝原因弹窗 */}
      <Modal
        title="拒绝提币"
        open={rejectModalOpen}
        onOk={handleConfirmReject}
        onCancel={() => {
          setRejectModalOpen(false);
          setRejectingId(null);
          setRejectReason('');
        }}
        okText="确认拒绝"
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <p style={{ marginBottom: 12, color: '#666' }}>
          请输入拒绝原因，该原因将通知用户:
        </p>
        <Input.TextArea
          rows={3}
          value={rejectReason}
          onChange={(e) => setRejectReason(e.target.value)}
          placeholder="请输入拒绝原因..."
          maxLength={200}
          showCount
        />
      </Modal>

      {/* 手动完成提币弹窗 */}
      <Modal
        title="手动完成提币"
        open={completeModalOpen}
        onOk={handleConfirmComplete}
        onCancel={() => {
          setCompleteModalOpen(false);
          setCompletingId(null);
          setTxHashInput('');
        }}
        okText="确认完成"
        cancelText="取消"
      >
        <p style={{ marginBottom: 12, color: '#666' }}>
          请输入链上交易哈希 (tx_hash)，确认该笔提币已在链上完成:
        </p>
        <Input
          value={txHashInput}
          onChange={(e) => setTxHashInput(e.target.value)}
          placeholder="请输入交易哈希..."
          maxLength={128}
          allowClear
        />
      </Modal>
    </div>
  );
};

export default WithdrawReview;
