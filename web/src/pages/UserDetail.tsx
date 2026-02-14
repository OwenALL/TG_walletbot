import { type FC, useState, useEffect, useCallback } from 'react';
import {
  Drawer,
  Descriptions,
  Tag,
  Table,
  Spin,
  Space,
  App,
  Button,
  Popconfirm,
  Modal,
  Form,
  Select,
  InputNumber,
  Input,
  Divider,
} from 'antd';
import {
  KeyOutlined,
  DollarOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { UserDetail as UserDetailType } from '@/types/user.ts';
import type { Wallet } from '@/types/user.ts';
import { getUserDetail, resetUserPIN, adjustUserBalance } from '@/api/users.ts';
import type { AdjustBalanceReq } from '@/api/users.ts';
import { formatAmount, formatTime, getUserStatusTag } from '@/utils/format.ts';
import { CURRENCY_COLORS, CURRENCIES } from '@/utils/constants.ts';

/** UserDetailDrawer 属性 */
interface UserDetailDrawerProps {
  /** 是否打开抽屉 */
  open: boolean;
  /** 用户 ID，为 null 时不加载 */
  userId: number | null;
  /** 关闭抽屉回调 */
  onClose: () => void;
}

/** 调整余额表单字段 */
interface AdjustBalanceFormValues {
  currency: string;
  amount: number;
  reason: string;
}

/**
 * 用户详情抽屉组件
 * 展示用户基础信息和钱包余额，支持重置 PIN 和调整余额操作
 */
const UserDetailDrawer: FC<UserDetailDrawerProps> = ({
  open,
  userId,
  onClose,
}) => {
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<UserDetailType | null>(null);
  const [adjustModalOpen, setAdjustModalOpen] = useState(false);
  const [adjustLoading, setAdjustLoading] = useState(false);
  const [resetPinLoading, setResetPinLoading] = useState(false);
  const [adjustForm] = Form.useForm<AdjustBalanceFormValues>();
  const { message } = App.useApp();

  /** 加载用户详情 */
  const loadDetail = useCallback(async () => {
    if (userId === null) return;
    setLoading(true);
    try {
      const data = await getUserDetail(userId);
      setDetail(data);
    } catch {
      message.error('加载用户详情失败');
    } finally {
      setLoading(false);
    }
  }, [userId, message]);

  useEffect(() => {
    if (!open || userId === null) {
      setDetail(null);
      return;
    }
    loadDetail();
  }, [open, userId, loadDetail]);

  /** 重置用户支付密码 */
  const handleResetPIN = async () => {
    if (userId === null) return;
    setResetPinLoading(true);
    try {
      await resetUserPIN(userId);
      message.success('支付密码已重置');
      await loadDetail();
    } catch {
      message.error('重置支付密码失败');
    } finally {
      setResetPinLoading(false);
    }
  };

  /** 打开调整余额弹窗 */
  const handleOpenAdjust = () => {
    setAdjustModalOpen(true);
  };

  /**
   * 调整余额弹窗打开后，重置表单
   * 使用 useEffect 避免 destroyOnHidden 下 form 未挂载的问题
   */
  useEffect(() => {
    if (adjustModalOpen) {
      adjustForm.resetFields();
    }
  }, [adjustModalOpen, adjustForm]);

  /** 提交调整余额 */
  const handleAdjustSubmit = async () => {
    if (userId === null) return;
    try {
      const values = await adjustForm.validateFields();
      setAdjustLoading(true);
      const reqData: AdjustBalanceReq = {
        currency: values.currency,
        amount: values.amount,
        reason: values.reason,
      };
      await adjustUserBalance(userId, reqData);
      message.success('余额调整成功');
      setAdjustModalOpen(false);
      await loadDetail();
    } catch (e: unknown) {
      if (e && typeof e === 'object' && !('errorFields' in e)) {
        message.error('余额调整失败');
      }
    } finally {
      setAdjustLoading(false);
    }
  };

  /** 钱包表格列定义 */
  const walletColumns: ColumnsType<Wallet> = [
    {
      title: '币种',
      dataIndex: 'currency',
      key: 'currency',
      width: 80,
      render: (val: string) => (
        <Tag color={CURRENCY_COLORS[val] ?? 'default'}>{val}</Tag>
      ),
    },
    {
      title: '可用余额',
      dataIndex: 'balance',
      key: 'balance',
      render: (val: string, record: Wallet) => formatAmount(val, record.currency),
    },
    {
      title: '冻结余额',
      dataIndex: 'frozen_balance',
      key: 'frozen_balance',
      render: (val: string, record: Wallet) => {
        const num = parseFloat(val);
        return (
          <span style={{ color: num > 0 ? '#cf1322' : undefined }}>
            {formatAmount(val, record.currency)}
          </span>
        );
      },
    },
    {
      title: '充值地址',
      dataIndex: 'deposit_address',
      key: 'deposit_address',
      ellipsis: true,
      render: (val: string) => val || '-',
    },
  ];

  const statusTag = detail ? getUserStatusTag(detail.status) : null;

  return (
    <Drawer
      title="用户详情"
      open={open}
      onClose={onClose}
      styles={{ wrapper: { width: 640 } }}
      destroyOnHidden
    >
      {loading ? (
        <div
          style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            height: 200,
          }}
        >
          <Spin size="large" />
        </div>
      ) : detail ? (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          {/* 基础信息 */}
          <Descriptions
            title="基础信息"
            column={2}
            bordered
            size="small"
          >
            <Descriptions.Item label="用户 ID">{detail.id}</Descriptions.Item>
            <Descriptions.Item label="Telegram ID">
              {detail.telegram_id}
            </Descriptions.Item>
            <Descriptions.Item label="用户名">
              {detail.username ? `@${detail.username}` : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="昵称">
              {`${detail.first_name} ${detail.last_name}`.trim() || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              {statusTag && <Tag color={statusTag.color}>{statusTag.text}</Tag>}
            </Descriptions.Item>
            <Descriptions.Item label="Premium">
              {detail.is_premium ? (
                <Tag color="gold">Premium</Tag>
              ) : (
                <Tag>普通</Tag>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="支付密码">
              {detail.has_pin ? (
                <Tag color="green">已设置</Tag>
              ) : (
                <Tag color="orange">未设置</Tag>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="Google 验证器">
              {detail.totp_enabled ? (
                <Tag color="green">已绑定</Tag>
              ) : (
                <Tag>未绑定</Tag>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="PIN 失败次数">
              {detail.pin_fail_count}
            </Descriptions.Item>
            <Descriptions.Item label="PIN 锁定时间">
              {detail.pin_locked_until
                ? formatTime(detail.pin_locked_until)
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="语言">
              {detail.language_code}
            </Descriptions.Item>
            <Descriptions.Item label="邀请人 ID">
              {detail.referrer_id ?? '-'}
            </Descriptions.Item>
            <Descriptions.Item label="注册时间" span={2}>
              {formatTime(detail.created_at)}
            </Descriptions.Item>
          </Descriptions>

          {/* 钱包余额 */}
          <div>
            <h4 style={{ marginBottom: 12 }}>钱包余额</h4>
            <Table<Wallet>
              columns={walletColumns}
              dataSource={detail.wallets}
              rowKey="id"
              pagination={false}
              size="small"
            />
          </div>

          {/* 管理操作区域 */}
          <div>
            <Divider titlePlacement="left">
              管理操作
            </Divider>
            <Space wrap>
              {/* 重置支付密码 —— 仅当用户已设置 PIN 时显示 */}
              {detail.has_pin && (
                <Popconfirm
                  title="确认重置支付密码"
                  description="确认清空该用户的支付密码？用户需要重新设置。"
                  onConfirm={handleResetPIN}
                  okText="确认重置"
                  cancelText="取消"
                  okButtonProps={{ danger: true }}
                >
                  <Button
                    icon={<KeyOutlined />}
                    danger
                    loading={resetPinLoading}
                  >
                    重置支付密码
                  </Button>
                </Popconfirm>
              )}

              {/* 调整余额 */}
              <Button
                icon={<DollarOutlined />}
                type="primary"
                onClick={handleOpenAdjust}
              >
                调整余额
              </Button>
            </Space>
          </div>
        </Space>
      ) : (
        <div style={{ textAlign: 'center', color: '#999', padding: 40 }}>
          暂无数据
        </div>
      )}

      {/* 调整余额弹窗 */}
      <Modal
        title="调整用户余额"
        open={adjustModalOpen}
        onOk={handleAdjustSubmit}
        onCancel={() => setAdjustModalOpen(false)}
        okText="确认调整"
        cancelText="取消"
        confirmLoading={adjustLoading}
        destroyOnHidden
      >
        <Form
          form={adjustForm}
          layout="vertical"
          style={{ marginTop: 16 }}
        >
          <Form.Item
            name="currency"
            label="币种"
            rules={[{ required: true, message: '请选择币种' }]}
          >
            <Select
              placeholder="请选择币种"
              options={CURRENCIES.map((c) => ({ label: c, value: c }))}
            />
          </Form.Item>
          <Form.Item
            name="amount"
            label="调整金额"
            rules={[
              { required: true, message: '请输入调整金额' },
              {
                validator: (_, value: number | undefined) => {
                  if (value === undefined || value === null) {
                    return Promise.resolve();
                  }
                  if (value === 0) {
                    return Promise.reject(new Error('金额不能为 0'));
                  }
                  return Promise.resolve();
                },
              },
            ]}
            extra="正数为增加余额，负数为扣减余额"
          >
            <InputNumber
              placeholder="请输入金额"
              style={{ width: '100%' }}
              step={0.01}
              precision={8}
            />
          </Form.Item>
          <Form.Item
            name="reason"
            label="调整原因"
            rules={[
              { required: true, message: '请输入调整原因' },
              { min: 2, message: '原因至少 2 个字符' },
            ]}
          >
            <Input.TextArea
              placeholder="请输入调整原因，如：系统补偿、手动扣款等"
              rows={3}
              maxLength={200}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>
    </Drawer>
  );
};

export default UserDetailDrawer;
