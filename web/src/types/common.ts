/** 通用 API 响应结构 */
export interface ApiResponse<T = unknown> {
  /** 业务状态码，0 表示成功 */
  code: number;
  /** 响应消息 */
  message: string;
  /** 响应数据 */
  data: T;
}

/** 分页请求参数 */
export interface PaginationParams {
  /** 当前页码，从 1 开始 */
  page: number;
  /** 每页条数 */
  size: number;
}

/** 分页响应数据 */
export interface PaginatedResponse<T> {
  /** 数据列表 */
  list: T[];
  /** 总条数 */
  total: number;
  /** 当前页码 */
  page: number;
  /** 每页条数 */
  size: number;
}

/** 通用 ID 参数 */
export interface IdParam {
  id: number;
}

/** 时间范围筛选参数 */
export interface TimeRangeParams {
  start_time?: string;
  end_time?: string;
}

/** 通用选项类型 */
export interface SelectOption<V = string> {
  label: string;
  value: V;
}
