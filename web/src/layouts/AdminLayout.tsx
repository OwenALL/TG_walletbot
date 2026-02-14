import { useState, type FC } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  Layout,
  Menu,
  Button,
  Dropdown,
  theme,
  Modal,
  Form,
  Input,
  App,
  type MenuProps,
} from 'antd';
import {
  DashboardOutlined,
  UserOutlined,
  TransactionOutlined,
  AuditOutlined,
  SwapOutlined,
  SettingOutlined,
  LogoutOutlined,
  LockOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  DollarOutlined,
  SafetyCertificateOutlined,
  FileSearchOutlined,
  GiftOutlined,
  FundOutlined,
  ShopOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '@/stores/authStore.ts';
import { changePassword } from '@/api/auth.ts';

const { Header, Sider, Content } = Layout;

/** 修改密码表单字段 */
interface ChangePasswordFormValues {
  old_password: string;
  new_password: string;
  confirm_password: string;
}

/** 侧边栏菜单项配置 */
const menuItems: MenuProps['items'] = [
  {
    key: '/',
    icon: <DashboardOutlined />,
    label: '仪表盘',
  },
  {
    key: '/users',
    icon: <UserOutlined />,
    label: '用户管理',
  },
  {
    key: '/transactions',
    icon: <TransactionOutlined />,
    label: '交易管理',
  },
  {
    key: '/withdrawals',
    icon: <AuditOutlined />,
    label: '提币审核',
  },
  {
    key: '/exchange-rates',
    icon: <SwapOutlined />,
    label: '汇率管理',
  },
  {
    key: '/fee-management',
    icon: <DollarOutlined />,
    label: '费率管理',
  },
  {
    key: '/limit-management',
    icon: <SafetyCertificateOutlined />,
    label: '限额管理',
  },
  {
    key: '/system-config',
    icon: <SettingOutlined />,
    label: '系统配置',
  },
  {
    key: '/audit-logs',
    icon: <FileSearchOutlined />,
    label: '审计日志',
  },
  {
    key: '/red-packets',
    icon: <GiftOutlined />,
    label: '红包管理',
  },
  {
    key: '/finance',
    icon: <FundOutlined />,
    label: '余额宝管理',
  },
  {
    key: '/merchants',
    icon: <ShopOutlined />,
    label: '商户管理',
  },
];

/**
 * 管理后台布局组件
 * 包含: 可折叠侧边栏 + 顶栏 (管理员信息 + 退出) + 内容区域
 */
const AdminLayout: FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [passwordSubmitting, setPasswordSubmitting] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const { message } = App.useApp();
  const [form] = Form.useForm<ChangePasswordFormValues>();
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  /** 侧边栏菜单点击处理 */
  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    navigate(key);
  };

  /** 退出登录 */
  const handleLogout = () => {
    logout();
    navigate('/login', { replace: true });
  };

  /** 关闭修改密码弹窗 */
  const handleClosePasswordModal = () => {
    setPasswordModalOpen(false);
    form.resetFields();
  };

  /** 提交修改密码 */
  const handleChangePassword = async () => {
    try {
      const values = await form.validateFields();
      setPasswordSubmitting(true);
      await changePassword(values.old_password, values.new_password);
      message.success('密码修改成功');
      handleClosePasswordModal();
    } catch {
      /* form.validateFields 的校验失败不需要额外提示，API 错误由拦截器处理 */
    } finally {
      setPasswordSubmitting(false);
    }
  };

  /** 用户下拉菜单项 */
  const userDropdownItems: MenuProps['items'] = [
    {
      key: 'username',
      label: `${user?.username ?? '管理员'}`,
      disabled: true,
    },
    {
      key: 'role',
      label: `角色: ${user?.role === 'super_admin' ? '超级管理员' : user?.role === 'admin' ? '管理员' : '只读'}`,
      disabled: true,
    },
    { type: 'divider' },
    {
      key: 'change-password',
      icon: <LockOutlined />,
      label: '修改密码',
      onClick: () => setPasswordModalOpen(true),
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true,
      onClick: handleLogout,
    },
  ];

  /** 计算当前选中的菜单 key */
  const selectedKey = location.pathname === '/' ? '/' : location.pathname;

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* 侧边栏 */}
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        theme="dark"
        width={220}
        style={{
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          zIndex: 10,
        }}
      >
        {/* Logo 区域 */}
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            borderBottom: '1px solid rgba(255, 255, 255, 0.1)',
          }}
        >
          <span
            style={{
              color: '#fff',
              fontSize: collapsed ? 16 : 18,
              fontWeight: 700,
              whiteSpace: 'nowrap',
            }}
          >
            {collapsed ? 'WB' : 'WalletBot 管理后台'}
          </span>
        </div>

        {/* 菜单 */}
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ borderRight: 0 }}
        />
      </Sider>

      {/* 右侧内容区 */}
      <Layout style={{ marginLeft: collapsed ? 80 : 220, transition: 'margin-left 0.2s' }}>
        {/* 顶栏 */}
        <Header
          style={{
            padding: '0 24px',
            background: colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            position: 'sticky',
            top: 0,
            zIndex: 9,
            boxShadow: '0 1px 4px rgba(0, 0, 0, 0.08)',
          }}
        >
          {/* 折叠按钮 */}
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
            style={{ fontSize: 16, width: 48, height: 48 }}
          />

          {/* 用户信息 */}
          <Dropdown menu={{ items: userDropdownItems }} placement="bottomRight">
            <Button type="text" icon={<UserOutlined />}>
              {user?.username ?? '管理员'}
            </Button>
          </Dropdown>
        </Header>

        {/* 内容区 */}
        <Content
          style={{
            margin: 24,
            padding: 24,
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
            minHeight: 280,
            overflow: 'auto',
          }}
        >
          <Outlet />
        </Content>
      </Layout>

      {/* 修改密码弹窗 */}
      <Modal
        title="修改密码"
        open={passwordModalOpen}
        onOk={handleChangePassword}
        onCancel={handleClosePasswordModal}
        confirmLoading={passwordSubmitting}
        okText="确认修改"
        cancelText="取消"
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          autoComplete="off"
          style={{ marginTop: 16 }}
        >
          <Form.Item
            name="old_password"
            label="旧密码"
            rules={[{ required: true, message: '请输入旧密码' }]}
          >
            <Input.Password placeholder="请输入当前密码" />
          </Form.Item>

          <Form.Item
            name="new_password"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码长度不能少于 6 位' },
            ]}
          >
            <Input.Password placeholder="请输入新密码（至少 6 位）" />
          </Form.Item>

          <Form.Item
            name="confirm_password"
            label="确认新密码"
            dependencies={['new_password']}
            rules={[
              { required: true, message: '请再次输入新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password placeholder="请再次输入新密码" />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  );
};

export default AdminLayout;
