import { type FC } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import AdminLayout from '@/layouts/AdminLayout.tsx';
import AuthGuard from '@/components/AuthGuard.tsx';
import Login from '@/pages/Login.tsx';
import Dashboard from '@/pages/Dashboard.tsx';
import UserList from '@/pages/UserList.tsx';
import TransactionList from '@/pages/TransactionList.tsx';
import WithdrawReview from '@/pages/WithdrawReview.tsx';
import ExchangeRates from '@/pages/ExchangeRates.tsx';
import FeeManagement from '@/pages/FeeManagement.tsx';
import LimitManagement from '@/pages/LimitManagement.tsx';
import SystemConfig from '@/pages/SystemConfig.tsx';
import AuditLogs from '@/pages/AuditLogs.tsx';
import RedPacketManagement from '@/pages/RedPacketManagement.tsx';
import FinanceManagement from '@/pages/FinanceManagement.tsx';
import MerchantReview from '@/pages/MerchantReview.tsx';

/**
 * 根组件 - 路由配置
 *
 * 路由结构:
 * /login             -> 登录页 (无布局)
 * /                  -> 仪表盘 (AdminLayout + AuthGuard)
 * /users             -> 用户管理 (AdminLayout + AuthGuard)
 * /transactions      -> 交易管理 (AdminLayout + AuthGuard)
 * /withdrawals       -> 提币审核 (AdminLayout + AuthGuard)
 * /exchange-rates    -> 汇率管理 (AdminLayout + AuthGuard)
 * /fee-management    -> 费率管理 (AdminLayout + AuthGuard)
 * /limit-management  -> 限额管理 (AdminLayout + AuthGuard)
 * /system-config     -> 系统配置 (AdminLayout + AuthGuard)
 * /audit-logs        -> 审计日志 (AdminLayout + AuthGuard)
 * /red-packets       -> 红包管理 (AdminLayout + AuthGuard)
 * /finance           -> 余额宝管理 (AdminLayout + AuthGuard)
 * /merchants         -> 商户管理 (AdminLayout + AuthGuard)
 */
const App: FC = () => {
  return (
    <BrowserRouter>
      <Routes>
        {/* 登录页 - 无需布局和认证 */}
        <Route path="/login" element={<Login />} />

        {/* 受保护的管理后台路由 */}
        <Route
          element={
            <AuthGuard>
              <AdminLayout />
            </AuthGuard>
          }
        >
          <Route path="/" element={<Dashboard />} />
          <Route path="/users" element={<UserList />} />
          <Route path="/transactions" element={<TransactionList />} />
          <Route path="/withdrawals" element={<WithdrawReview />} />
          <Route path="/exchange-rates" element={<ExchangeRates />} />
          <Route path="/fee-management" element={<FeeManagement />} />
          <Route path="/limit-management" element={<LimitManagement />} />
          <Route path="/system-config" element={<SystemConfig />} />
          <Route path="/audit-logs" element={<AuditLogs />} />
          <Route path="/red-packets" element={<RedPacketManagement />} />
          <Route path="/finance" element={<FinanceManagement />} />
          <Route path="/merchants" element={<MerchantReview />} />
        </Route>

        {/* 未匹配路由重定向到首页 */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
};

export default App;
