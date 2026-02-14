import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Table,
  Tag,
  Button,
  Modal,
  Form,
  InputNumber,
  Switch,
  Space,
  App,
} from 'antd';
import { EditOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { ExchangeRate } from '@/types/exchange.ts';
import { getExchangeRates, updateExchangeRate } from '@/api/exchangeRates.ts';
import { formatAmount, formatTime } from '@/utils/format.ts';

/** 编辑表单字段 */
interface EditFormValues {
  rate: number;
  spread: number;
  min_amount: number;
  max_amount: number;
  enabled: boolean;
}

/**
 * 汇率管理页面
 * 功能: 展示所有兑换方向的汇率配置，支持编辑
 */
const ExchangeRates: FC = () => {
  const [loading, setLoading] = useState(false);
  const [rates, setRates] = useState<ExchangeRate[]>([]);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingRate, setEditingRate] = useState<ExchangeRate | null>(null);
  const [form] = Form.useForm<EditFormValues>();
  const { message } = App.useApp();

  /** 加载汇率列表 */
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getExchangeRates();
      setRates(data);
    } catch {
      message.error('加载汇率配置失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  /** 计算最终汇率 (基准汇率 + 加点) */
  const calcFinalRate = (rate: string, spread: string): string => {
    const r = parseFloat(rate);
    const s = parseFloat(spread);
    if (isNaN(r) || isNaN(s)) return '-';
    const final = r * (1 + s / 100);
    return final.toFixed(6);
  };

  /** 打开编辑弹窗 */
  const handleEdit = (record: ExchangeRate) => {
    setEditingRate(record);
    form.setFieldsValue({
      rate: parseFloat(record.rate),
      spread: parseFloat(record.spread),
      min_amount: parseFloat(record.min_amount),
      max_amount: parseFloat(record.max_amount),
      enabled: record.enabled,
    });
    setEditModalOpen(true);
  };

  /** 提交编辑 */
  const handleEditSubmit = async () => {
    if (!editingRate) return;
    try {
      const values = await form.validateFields();
      setLoading(true);
      await updateExchangeRate({
        id: editingRate.id,
        rate: values.rate.toString(),
        spread: values.spread.toString(),
        min_amount: values.min_amount.toString(),
        max_amount: values.max_amount.toString(),
        enabled: values.enabled,
      });
      message.success('汇率更新成功');
      setEditModalOpen(false);
      setEditingRate(null);
      await loadData();
    } catch (e: unknown) {
      /* 表单验证失败时 antd 会 reject，仅在非验证错误时提示 */
      if (e && typeof e === 'object' && !('errorFields' in e)) {
        message.error('操作失败');
      }
    } finally {
      setLoading(false);
    }
  };

  /** 表格列定义 */
  const columns: ColumnsType<ExchangeRate> = [
    {
      title: '兑换方向',
      key: 'direction',
      width: 160,
      render: (_: unknown, record: ExchangeRate) => (
        <span style={{ fontWeight: 500 }}>
          {record.from_currency} {'->'} {record.to_currency}
        </span>
      ),
    },
    {
      title: '基准汇率',
      dataIndex: 'rate',
      key: 'rate',
      width: 120,
      render: (val: string) => formatAmount(val),
    },
    {
      title: '加点比例 (%)',
      dataIndex: 'spread',
      key: 'spread',
      width: 110,
      render: (val: string) => `${val}%`,
    },
    {
      title: '最终汇率',
      key: 'final_rate',
      width: 130,
      render: (_: unknown, record: ExchangeRate) => (
        <span style={{ color: '#1677ff', fontWeight: 500 }}>
          {calcFinalRate(record.rate, record.spread)}
        </span>
      ),
    },
    {
      title: '最低限额',
      dataIndex: 'min_amount',
      key: 'min_amount',
      width: 120,
      render: (val: string, record: ExchangeRate) =>
        `${formatAmount(val)} ${record.from_currency}`,
    },
    {
      title: '最高限额',
      dataIndex: 'max_amount',
      key: 'max_amount',
      width: 140,
      render: (val: string, record: ExchangeRate) =>
        `${formatAmount(val)} ${record.from_currency}`,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (val: boolean) =>
        val ? (
          <Tag color="green">启用</Tag>
        ) : (
          <Tag color="default">停用</Tag>
        ),
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
      render: (_: unknown, record: ExchangeRate) => (
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
      <Table<ExchangeRate>
        columns={columns}
        dataSource={rates}
        rowKey="id"
        loading={loading}
        pagination={false}
        scroll={{ x: 1100 }}
        size="middle"
      />

      {/* 编辑汇率弹窗 */}
      <Modal
        title={
          editingRate
            ? `编辑汇率: ${editingRate.from_currency} -> ${editingRate.to_currency}`
            : '编辑汇率'
        }
        open={editModalOpen}
        onOk={handleEditSubmit}
        onCancel={() => {
          setEditModalOpen(false);
          setEditingRate(null);
        }}
        okText="保存"
        cancelText="取消"
        confirmLoading={loading}
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          style={{ marginTop: 16 }}
        >
          <Form.Item
            name="rate"
            label="基准汇率"
            rules={[
              { required: true, message: '请输入基准汇率' },
              {
                type: 'number',
                min: 0.000001,
                message: '汇率必须大于 0',
              },
            ]}
          >
            <InputNumber
              style={{ width: '100%' }}
              precision={8}
              min={0}
              step={0.01}
              placeholder="请输入基准汇率"
            />
          </Form.Item>
          <Form.Item
            name="spread"
            label="加点比例 (%)"
            rules={[
              { required: true, message: '请输入加点比例' },
              {
                type: 'number',
                min: 0,
                max: 50,
                message: '加点比例范围 0-50%',
              },
            ]}
          >
            <Space.Compact style={{ width: '100%' }}>
              <InputNumber
                style={{ width: '100%' }}
                precision={2}
                min={0}
                max={50}
                step={0.1}
                placeholder="请输入加点比例"
              />
              <Button disabled style={{ cursor: 'default' }}>%</Button>
            </Space.Compact>
          </Form.Item>
          <Form.Item
            name="min_amount"
            label="最低限额"
            rules={[
              { required: true, message: '请输入最低限额' },
              { type: 'number', min: 0, message: '最低限额不能为负数' },
            ]}
          >
            <InputNumber
              style={{ width: '100%' }}
              precision={2}
              min={0}
              placeholder="请输入最低限额"
            />
          </Form.Item>
          <Form.Item
            name="max_amount"
            label="最高限额"
            rules={[
              { required: true, message: '请输入最高限额' },
              { type: 'number', min: 1, message: '最高限额必须大于 0' },
            ]}
          >
            <InputNumber
              style={{ width: '100%' }}
              precision={2}
              min={1}
              placeholder="请输入最高限额"
            />
          </Form.Item>
          <Form.Item
            name="enabled"
            label="启用状态"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="停用" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ExchangeRates;
