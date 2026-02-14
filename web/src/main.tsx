import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App as AntdApp, ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import App from './App.tsx';
import '@/styles/global.css';

/* 设置 dayjs 全局语言为中文 */
dayjs.locale('zh-cn');

/**
 * 应用入口
 * - StrictMode: React 严格模式
 * - ConfigProvider: Ant Design 全局配置 (中文语言包、主题)
 * - AntdApp: 提供 message / notification / modal 的上下文
 */
const rootEl = document.getElementById('root');
if (!rootEl) {
  throw new Error('找不到 #root 挂载节点');
}

createRoot(rootEl).render(
  <StrictMode>
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#1677ff',
          borderRadius: 6,
        },
      }}
    >
      <AntdApp>
        <App />
      </AntdApp>
    </ConfigProvider>
  </StrictMode>,
);
