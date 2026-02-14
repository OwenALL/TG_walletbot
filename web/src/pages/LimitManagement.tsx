import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  App,
  Typography,
} from 'antd';
import { EditOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { SystemConfig } from '@/types/config.ts';
import { getConfigs, updateConfig } from '@/api/configs.ts';
import { formatTime } from '@/utils/format.ts';
import { LIMIT_CONFIG_KEYS } from '@/utils/constants.ts';

const { Text } = Typography;

/** 配置 key → 友好名称 */
const LIMIT_LABELS: Record<string, string> = {
  withdraw_usdt_min: 'USDT 提币最低金额',
  withdraw_usdt_max: 'USDT 提币最高金额',
  withdraw_usdt_daily_max: 'USDT 提币每日上限',
  withdraw_usdt_auto_threshold: 'USDT 自动审核阈值',
  withdraw_trx_min: 'TRX 提币最低金额',
  withdraw_trx_max: 'TRX 提币最高金额',
  withdraw_cny_min: 'CNY 提现最低金额',
  withdraw_cny_daily_max: 'CNY 提现每日上限',
  transfer_usdt_min: 'USDT 转账最低金额',
  exchange_min_usdt: 'USDT 闪兑最低金额',
  exchange_min_cny: 'CNY 闪兑最低金额',
  exchange_min_trx: 'TRX 闪兑最低金额',
  finance_min: '余额宝买入最低金额',
  finance_max: '余额宝买入最高金额',
};

/** 编辑表单字段 */
interface EditFormValues {
  config_value: string;
}

/**
 * 限额管理页面
 * 功能: 展示限额相关系统配置，支持 Modal 编辑
 */
const LimitManagement: FC = () => {
  const [loading, setLoading] = useState(false);
  const [limits, setLimits] = useState<SystemConfig[]>([]);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingLimit, setEditingLimit] = useState<SystemConfig | null>(null);
  const [form] = Form.useForm<EditFormValues>();
  const { message } = App.useApp();

  /** 加载限额配置 (从 system_configs 筛选) */
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const all = await getConfigs();
      const filtered = all.filter((c) => LIMIT_CONFIG_KEYS.has(c.config_key));
      setLimits(filtered);
    } catch {
      message.error('加载限额配置失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  /**
   * Modal 打开且待编辑配置就绪后，再设置表单值
   * 避免 destroyOnHidden 导致 Form DOM 不存在时调用 setFieldsValue 的警告
   */
  useEffect(() => {
    if (editModalOpen && editingLimit) {
      form.setFieldsValue({
        config_value: editingLimit.config_value,
      });
    }
  }, [editModalOpen, editingLimit, form]);

  /** 打开编辑弹窗 */
  const handleEdit = (record: SystemConfig) => {
    setEditingLimit(record);
    setEditModalOpen(true);
  };

  /** 提交编辑 */
  const handleEditSubmit = async () => {
    if (!editingLimit) return;
    try {
      const values = await form.validateFields();
      setLoading(true);
      await updateConfig({
        id: editingLimit.id,
        config_value: values.config_value,
      });
      message.success('限额更新成功');
      setEditModalOpen(false);
      setEditingLimit(null);
      await loadData();
    } catch (e: unknown) {
      if (e && typeof e === 'object' && !('errorFields' in e)) {
        message.error('操作失败');
      }
    } finally {
      setLoading(false);
    }
  };

  /** 表格列定义 */
  const columns: ColumnsType<SystemConfig> = [
    {
      title: '限额名称',
      dataIndex: 'config_key',
      key: 'config_key',
      width: 220,
      render: (val: string) => (
        <Text strong>{LIMIT_LABELS[val] ?? val}</Text>
      ),
    },
    {
      title: '当前值',
      dataIndex: 'config_value',
      key: 'config_value',
      width: 150,
      render: (val: string) => (
        <Text strong style={{ color: '#1677ff' }}>
          {val}
        </Text>
      ),
    },
    {
      title: '说明',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 170,
      render: (val: string) => formatTime(val),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      fixed: 'right',
      render: (_: unknown, record: SystemConfig) => (
        <Button
          type="link"
          size="small"
          icon={<EditOutlined />}
          onClick={() => handleEdit(record)}
        >
          编辑
        </Button>
      ),
    },
  ];

  return (
    <div>
      <Table<SystemConfig>
        columns={columns}
        dataSource={limits}
        rowKey="id"
        loading={loading}
        pagination={false}
        scroll={{ x: 900 }}
        size="middle"
      />

      {/* 编辑限额弹窗 */}
      <Modal
        title={
          editingLimit
            ? `编辑: ${LIMIT_LABELS[editingLimit.config_key] ?? editingLimit.config_key}`
            : '编辑限额'
        }
        open={editModalOpen}
        onOk={handleEditSubmit}
        onCancel={() => {
          setEditModalOpen(false);
          setEditingLimit(null);
        }}
        okText="保存"
        cancelText="取消"
        confirmLoading={loading}
        destroyOnHidden
      >
        {editingLimit && (
          <div style={{ marginBottom: 16 }}>
            <Text type="secondary">{editingLimit.description}</Text>
          </div>
        )}
        <Form form={form} layout="vertical" style={{ marginTop: 8 }}>
          <Form.Item
            name="config_value"
            label="限额值"
            rules={[{ required: true, message: '请输入限额值' }]}
          >
            <Input placeholder="请输入限额值" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default LimitManagement;
