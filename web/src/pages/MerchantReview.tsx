import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Table,
  Tag,
  Button,
  Space,
  Select,
  Popconfirm,
  Modal,
  InputNumber,
  Form,
  App,
  Typography,
} from 'antd';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { Merchant, MerchantListParams } from '@/types/merchant.ts';
import { MerchantStatus } from '@/types/merchant.ts';
import {
  getMerchantList,
  toggleMerchantStatus,
  updateMerchantFeeRate,
} from '@/api/merchants.ts';
import { formatTime } from '@/utils/format.ts';
import { DEFAULT_PAGE_SIZE } from '@/utils/constants.ts';

const { Text } = Typography;

/** 商户状态筛选选项 */
const STATUS_OPTIONS = [
  { label: '全部', value: -1 },
  { label: '正常', value: MerchantStatus.Active },
  { label: '已关闭', value: MerchantStatus.Disabled },
];

/** 获取商户状态 Tag 颜色和文本 */
const getMerchantStatusTag = (
  status: number,
): { color: string; text: string } => {
  switch (status) {
    case MerchantStatus.Active:
      return { color: 'green', text: '正常' };
    case MerchantStatus.Disabled:
      return { color: 'default', text: '已关闭' };
    case MerchantStatus.Pending:
      return { color: 'warning', text: '待审核' };
    case MerchantStatus.Rejected:
      return { color: 'error', text: '已拒绝' };
    default:
      return { color: 'default', text: '未知' };
  }
};

/** 费率表单字段 */
interface FeeRateFormValues {
  fee_rate: number;
}

/**
 * 商户管理页面
 * 功能: 状态筛选 + 商户列表表格 + 启用/禁用操作 + 修改费率
 */
const MerchantReview: FC = () => {
  const [loading, setLoading] = useState(false);
  const [merchants, setMerchants] = useState<Merchant[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [statusFilter, setStatusFilter] = useState<number>(-1);
  const [feeRateModalOpen, setFeeRateModalOpen] = useState(false);
  const [editingMerchant, setEditingMerchant] = useState<Merchant | null>(null);
  const [feeRateSubmitting, setFeeRateSubmitting] = useState(false);
  const [form] = Form.useForm<FeeRateFormValues>();
  const { message } = App.useApp();

  /** 加载商户列表 */
  const loadMerchants = useCallback(async () => {
    setLoading(true);
    try {
      const params: MerchantListParams = { page, size: pageSize };
      if (statusFilter !== -1) {
        params.status = statusFilter;
      }
      const resp = await getMerchantList(params);
      setMerchants(resp.list);
      setTotal(resp.total);
    } catch {
      message.error('加载商户列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, statusFilter, message]);

  useEffect(() => {
    loadMerchants();
  }, [loadMerchants]);

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

  /** 切换商户启用/禁用状态 */
  const handleToggleStatus = async (record: Merchant) => {
    setLoading(true);
    try {
      await toggleMerchantStatus(record.id);
      const actionText = record.status === MerchantStatus.Active ? '已禁用' : '已启用';
      message.success(`商户 ${record.business_name} ${actionText}`);
      await loadMerchants();
    } catch {
      message.error('操作失败');
    } finally {
      setLoading(false);
    }
  };

  /** 打开修改费率弹窗 */
  const handleOpenFeeRate = (record: Merchant) => {
    setEditingMerchant(record);
    /* 将后端 decimal 字符串转为百分比数值展示，如 "0.01" -> 1 */
    const currentRate = parseFloat(record.fee_rate) * 100;
    form.setFieldsValue({ fee_rate: isNaN(currentRate) ? 0 : currentRate });
    setFeeRateModalOpen(true);
  };

  /** 关闭修改费率弹窗 */
  const handleCloseFeeRate = () => {
    setFeeRateModalOpen(false);
    setEditingMerchant(null);
    form.resetFields();
  };

  /** 提交修改费率 */
  const handleSubmitFeeRate = async () => {
    try {
      const values = await form.validateFields();
      if (editingMerchant === null) return;

      setFeeRateSubmitting(true);
      /* 前端输入百分比，转为小数传给后端，如 1.5 -> 0.015 */
      const rateDecimal = values.fee_rate / 100;
      await updateMerchantFeeRate(editingMerchant.id, rateDecimal);
      message.success('费率修改成功');
      handleCloseFeeRate();
      await loadMerchants();
    } catch {
      /* form.validateFields 的校验失败不需要额外提示，API 调用失败由拦截器处理 */
    } finally {
      setFeeRateSubmitting(false);
    }
  };

  /** 脱敏显示 API Key：保留前 8 位和后 4 位 */
  const maskApiKey = (key: string): string => {
    if (!key) return '-';
    if (key.length <= 16) return key;
    return `${key.slice(0, 8)}****${key.slice(-4)}`;
  };

  /** 表格列定义 */
  const columns: ColumnsType<Merchant> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 70,
    },
    {
      title: '用户',
      key: 'user',
      width: 130,
      render: (_: unknown, record: Merchant) =>
        record.display_name || (record.username ? `@${record.username}` : '-'),
    },
    {
      title: '商户名称',
      dataIndex: 'business_name',
      key: 'business_name',
      width: 160,
      render: (val: string) => (
        <span style={{ fontWeight: 500 }}>{val}</span>
      ),
    },
    {
      title: 'API Key',
      dataIndex: 'api_key',
      key: 'api_key',
      width: 200,
      render: (val: string) => {
        if (!val) {
          return <Text type="secondary">-</Text>;
        }
        return (
          <Text copyable={{ text: val }} style={{ fontSize: 12 }}>
            {maskApiKey(val)}
          </Text>
        );
      },
    },
    {
      title: '回调地址',
      dataIndex: 'webhook_url',
      key: 'webhook_url',
      width: 220,
      ellipsis: true,
      render: (val: string) => val || '-',
    },
    {
      title: '费率',
      dataIndex: 'fee_rate',
      key: 'fee_rate',
      width: 90,
      render: (val: string) => {
        const rate = parseFloat(val);
        if (isNaN(rate)) return '-';
        /* 转为百分比显示，保留最多 2 位小数 */
        const percent = parseFloat((rate * 100).toFixed(2));
        return `${percent}%`;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (val: number) => {
        const tag = getMerchantStatusTag(val);
        return <Tag color={tag.color}>{tag.text}</Tag>;
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (val: string) => formatTime(val),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      fixed: 'right',
      render: (_: unknown, record: Merchant) => {
        const isActive = record.status === MerchantStatus.Active;
        const isDisabled = record.status === MerchantStatus.Disabled;

        return (
          <Space size="small">
            {/* 启用/禁用切换按钮，仅对正常和已关闭状态显示 */}
            {(isActive || isDisabled) && (
              <Popconfirm
                title={isActive ? '确认禁用该商户？' : '确认启用该商户？'}
                description={`商户: ${record.business_name}`}
                onConfirm={() => handleToggleStatus(record)}
                okText="确认"
                cancelText="取消"
              >
                <Button
                  type="link"
                  size="small"
                  danger={isActive}
                  style={isDisabled ? { color: '#52c41a' } : undefined}
                >
                  {isActive ? '禁用' : '启用'}
                </Button>
              </Popconfirm>
            )}
            {/* 修改费率按钮 */}
            <Button
              type="link"
              size="small"
              onClick={() => handleOpenFeeRate(record)}
            >
              修改费率
            </Button>
          </Space>
        );
      },
    },
  ];

  return (
    <div>
      {/* 页面标题 */}
      <Typography.Title level={4} style={{ marginTop: 0, marginBottom: 16 }}>
        商户管理
      </Typography.Title>

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

      {/* 商户列表表格 */}
      <Table<Merchant>
        columns={columns}
        dataSource={merchants}
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

      {/* 修改费率弹窗 */}
      <Modal
        title="修改商户费率"
        open={feeRateModalOpen}
        onOk={handleSubmitFeeRate}
        onCancel={handleCloseFeeRate}
        confirmLoading={feeRateSubmitting}
        okText="确认修改"
        cancelText="取消"
        destroyOnHidden
      >
        {editingMerchant && (
          <div style={{ marginBottom: 16 }}>
            <Text type="secondary">
              商户: {editingMerchant.business_name}
            </Text>
          </div>
        )}
        <Form
          form={form}
          layout="vertical"
          autoComplete="off"
        >
          <Form.Item
            name="fee_rate"
            label="费率 (%)"
            rules={[
              { required: true, message: '请输入费率' },
              {
                type: 'number',
                min: 0,
                max: 100,
                message: '费率范围为 0 ~ 100%',
              },
            ]}
          >
            <InputNumber
              style={{ width: '100%' }}
              placeholder="请输入费率百分比，如 1.5 表示 1.5%"
              min={0}
              max={100}
              step={0.1}
              precision={2}
              addonAfter="%"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default MerchantReview;
