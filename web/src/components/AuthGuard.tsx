import { type FC, type ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore.ts';

/** AuthGuard 属性 */
interface AuthGuardProps {
  children: ReactNode;
}

/**
 * 路由守卫组件
 * 检查用户是否已认证，未认证则重定向到登录页
 * 登录页不需要守卫
 */
const AuthGuard: FC<AuthGuardProps> = ({ children }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const location = useLocation();

  if (!isAuthenticated) {
    /* 保存当前路径到 state，登录成功后可以跳回 */
    return <Navigate to="/login" state={{ from: location.pathname }} replace />;
  }

  return <>{children}</>;
};

export default AuthGuard;
