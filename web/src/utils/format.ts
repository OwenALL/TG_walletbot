import dayjs from 'dayjs';

/**
 * 格式化金额
 * @param amount - 金额 (字符串或数字)
 * @param currency - 币种
 * @returns 格式化后的金额字符串
 */
export const formatAmount = (
  amount: string | number,
  currency?: string,
): string => {
  const num = typeof amount === 'string' ? parseFloat(amount) : amount;
  if (isNaN(num)) return '0.00';

  /* CNY 保留 2 位，加密货币保留最多 8 位 (去除尾部 0) */
  let formatted: string;
  if (currency === 'CNY') {
    formatted = num.toFixed(2);
  } else {
    /* 先保留 8 位，然后去除尾部多余的 0 */
    formatted = parseFloat(num.toFixed(8)).toString();
    /* 确保至少 2 位小数 */
    if (!formatted.includes('.')) {
      formatted += '.00';
    } else {
      const decimals = formatted.split('.')[1];
      if (decimals && decimals.length < 2) {
        formatted += '0';
      }
    }
  }

  /* 添加千分位分隔符 */
  const parts = formatted.split('.');
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  const result = parts.join('.');

  return currency ? `${result} ${currency}` : result;
};

/**
 * 格式化时间
 * @param time - ISO 时间字符串
 * @param format - 输出格式，默认 'YYYY-MM-DD HH:mm:ss'
 * @returns 格式化后的时间字符串
 */
export const formatTime = (
  time: string | null | undefined,
  format: string = 'YYYY-MM-DD HH:mm:ss',
): string => {
  if (!time) return '-';
  const d = dayjs(time);
  return d.isValid() ? d.format(format) : '-';
};

/**
 * 格式化日期 (仅日期部分)
 * @param time - ISO 时间字符串
 * @returns 格式化后的日期字符串
 */
export const formatDate = (time: string | null | undefined): string => {
  return formatTime(time, 'YYYY-MM-DD');
};

/**
 * 缩写区块链地址
 * @param addr - 完整地址
 * @param prefixLen - 前缀保留长度，默认 6
 * @param suffixLen - 后缀保留长度，默认 6
 * @returns 缩写后的地址，如 T9yD82...Ae3Kf9
 */
export const shortenAddress = (
  addr: string | null | undefined,
  prefixLen: number = 6,
  suffixLen: number = 6,
): string => {
  if (!addr) return '-';
  if (addr.length <= prefixLen + suffixLen + 3) return addr;
  return `${addr.slice(0, prefixLen)}...${addr.slice(-suffixLen)}`;
};

/** 状态 Tag 信息 */
export interface StatusTagInfo {
  color: string;
  text: string;
}

/**
 * 获取用户状态 Tag 样式和文本
 * @param status - 用户状态 (1:正常 2:冻结 3:封禁)
 */
export const getUserStatusTag = (status: number): StatusTagInfo => {
  switch (status) {
    case 1:
      return { color: 'green', text: '正常' };
    case 2:
      return { color: 'orange', text: '冻结' };
    case 3:
      return { color: 'red', text: '封禁' };
    default:
      return { color: 'default', text: '未知' };
  }
};

/**
 * 获取提币状态 Tag 样式和文本
 * @param status - 提币状态 (0:待审核 1:处理中 2:已完成 3:已拒绝 4:失败)
 */
export const getWithdrawalStatusTag = (status: number): StatusTagInfo => {
  switch (status) {
    case 0:
      return { color: 'orange', text: '待审核' };
    case 1:
      return { color: 'blue', text: '处理中' };
    case 2:
      return { color: 'green', text: '已完成' };
    case 3:
      return { color: 'red', text: '已拒绝' };
    case 4:
      return { color: 'red', text: '失败' };
    default:
      return { color: 'default', text: '未知' };
  }
};

/**
 * 获取交易类型显示名称
 * @param type - 交易类型
 */
export const getTransactionTypeName = (type: string): string => {
  const map: Record<string, string> = {
    deposit: '充值',
    withdraw: '提币',
    transfer_in: '转入',
    transfer_out: '转出',
    exchange_in: '闪兑兑入',
    exchange_out: '闪兑兑出',
    redpacket_send: '红包发出',
    redpacket_receive: '红包领取',
    redpacket_refund: '红包退回',
    finance_in: '余额宝买入',
    finance_out: '余额宝取出',
    finance_profit: '余额宝收益',
  };
  return map[type] ?? type;
};

/**
 * 判断交易类型是否为收入
 */
export const isIncomeType = (type: string): boolean => {
  const incomeTypes = [
    'deposit',
    'transfer_in',
    'exchange_in',
    'redpacket_receive',
    'redpacket_refund',
    'finance_out',
    'finance_profit',
  ];
  return incomeTypes.includes(type);
};
