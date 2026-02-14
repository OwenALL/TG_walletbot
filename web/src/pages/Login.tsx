import { type FC, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { Card, Form, Input, Button, App } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useAuthStore } from '@/stores/authStore.ts';

/** 登录表单字段 */
interface LoginFormValues {
  username: string;
  password: string;
}

/**
 * 登录页面
 * 居中卡片式布局，包含用户名和密码输入框
 */
const Login: FC = () => {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const login = useAuthStore((state) => state.login);
  const { message } = App.useApp();

  /** 表单提交处理 */
  const handleFinish = async (values: LoginFormValues) => {
    setLoading(true);
    try {
      await login(values);
      message.success('登录成功');
      /* 跳转到之前访问的页面或首页 */
      const from =
        (location.state as { from?: string } | null)?.from || '/';
      navigate(from, { replace: true });
    } catch (err) {
      const errorMsg =
        err instanceof Error ? err.message : '登录失败，请重试';
      message.error(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <Card className="login-card" variant="borderless">
        <div className="login-title">WalletBot 管理后台</div>
        <div className="login-subtitle">Telegram 钱包 Bot 管理系统</div>

        <Form<LoginFormValues>
          name="login"
          onFinish={handleFinish}
          autoComplete="off"
          size="large"
        >
          {/* 用户名 */}
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder="用户名"
              allowClear
            />
          </Form.Item>

          {/* 密码 */}
          <Form.Item
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="密码"
            />
          </Form.Item>

          {/* 登录按钮 */}
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              登录
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default Login;
