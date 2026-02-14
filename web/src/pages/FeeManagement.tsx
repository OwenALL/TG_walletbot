import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Table,
  Button,
  Modal,
  Form,
  InputNumber,
  App,
  Typography,
} from 'antd';
import { EditOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { SystemConfig } from '@/types/config.ts';
import { getConfigs, updateConfig } from '@/api/configs.ts';
import { formatTime } from '@/utils/format.ts';

const { Text } = Typography;

/** 费率相关的配置 key 集合 (仅展示手续费配置) */
const FEE_CONFIG_KEYS = new Set([
  'withdraw_usdt_fee',
  'withdraw_trx_fee',
]);

/** 配置 key → 友好名称 */
const FEE_LABELS: Record<string, string> = {
  withdraw_usdt_fee: 'USDT 提币手续费',
  withdraw_trx_fee: 'TRX 提币手续费',
};

/** 编辑表单字段 */
interface EditFormValues {
  config_value: number;
}

/**
 * 费率管理页面
 * 功能: 展示手续费相关系统配置，支持 Modal 编辑
 */
const FeeManagement: FC = () => {
  const [loading, setLoading] = useState(false);
  const [fees, setFees] = useState<SystemConfig[]>([]);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingFee, setEditingFee] = useState<SystemConfig | null>(null);
  const [form] = Form.useForm<EditFormValues>();
  const { message } = App.useApp();

  /** 加载费率配置 (从 system_configs 筛选) */
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const all = await getConfigs();
      const filtered = all.filter((c) => FEE_CONFIG_KEYS.has(c.config_key));
      setFees(filtered);
    } catch {
      message.error('加载费率配置失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  /** 打开编辑弹窗 */
  const handleEdit = (record: SystemConfig) => {
    setEditingFee(record);
    form.setFieldsValue({
      config_value: parseFloat(record.config_value),
    });
    setEditModalOpen(true);
  };

  /** 提交编辑 */
  const handleEditSubmit = async () => {
    if (!editingFee) return;
    try {
      const values = await form.validateFields();
      setLoading(true);
      await updateConfig({
        id: editingFee.id,
        config_value: values.config_value.toString(),
      });
      message.success('费率更新成功');
      setEditModalOpen(false);
      setEditingFee(null);
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
      title: '费率名称',
      dataIndex: 'config_key',
      key: 'config_key',
      width: 200,
      render: (val: string) => (
        <Text strong>{FEE_LABELS[val] ?? val}</Text>
      ),
    },
    {
      title: '费率值',
      dataIndex: 'config_value',
      key: 'config_value',
      width: 120,
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
        dataSource={fees}
        rowKey="id"
        loading={loading}
        pagination={false}
        scroll={{ x: 800 }}
        size="middle"
      />

      {/* 编辑费率弹窗 */}
      <Modal
        title={
          editingFee
            ? `编辑: ${FEE_LABELS[editingFee.config_key] ?? editingFee.config_key}`
            : '编辑费率'
        }
        open={editModalOpen}
        onOk={handleEditSubmit}
        onCancel={() => {
          setEditModalOpen(false);
          setEditingFee(null);
        }}
        okText="保存"
        cancelText="取消"
        confirmLoading={loading}
        destroyOnHidden
      >
        {editingFee && (
          <div style={{ marginBottom: 16 }}>
            <Text type="secondary">{editingFee.description}</Text>
          </div>
        )}
        <Form form={form} layout="vertical" style={{ marginTop: 8 }}>
          <Form.Item
            name="config_value"
            label="费率值"
            rules={[
              { required: true, message: '请输入费率值' },
              { type: 'number', min: 0, message: '费率值不能为负数' },
            ]}
          >
            <InputNumber
              style={{ width: '100%' }}
              precision={4}
              min={0}
              step={0.1}
              placeholder="请输入费率值"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default FeeManagement;
