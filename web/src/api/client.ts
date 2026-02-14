import axios from 'axios';
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import type { ApiResponse } from '@/types/common.ts';

/** Axios 实例，baseURL 指向后端 API 前缀 */
const client = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
});

/**
 * 请求拦截器：自动附加 JWT Token
 * 从 localStorage 读取 token 并添加到 Authorization 头
 */
client.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('admin_token');
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error: unknown) => {
    return Promise.reject(error);
  },
);

/**
 * 响应拦截器：统一错误处理和数据解包
 * - 自动解出 response.data (ApiResponse 层)
 * - 401 未授权时跳转登录页
 * - 403 无权限提示
 * - 其他错误抛出统一格式
 */
client.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const apiResp = response.data;
    /* 业务层错误 (code !== 0) */
    if (apiResp.code !== 0) {
      const error = new Error(apiResp.message || '请求失败') as Error & {
        code: number;
      };
      error.code = apiResp.code;
      return Promise.reject(error);
    }
    /* 成功：直接返回 data 字段，减少调用层解包步骤 */
    return apiResp.data as unknown as AxiosResponse;
  },
  (error: unknown) => {
    if (axios.isAxiosError(error)) {
      const status = error.response?.status;
      if (status === 401) {
        /* Token 失效，清除本地存储并跳转登录 */
        localStorage.removeItem('admin_token');
        localStorage.removeItem('admin_user');
        window.location.href = '/login';
      }
      /* 构造统一错误消息 */
      const apiMsg =
        (error.response?.data as ApiResponse | undefined)?.message;
      const msg = apiMsg || error.message || '网络请求失败';
      return Promise.reject(new Error(msg));
    }
    return Promise.reject(error);
  },
);

export default client;
