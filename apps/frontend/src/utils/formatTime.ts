/**
 * 格式化时间（精确到分钟）
 * @param dateString - 日期字符串
 * @returns 格式化后的时间字符串（使用中国时区）
 */
export const formatTime = (dateString: string): string => {
  const date = new Date(dateString);

  if (isNaN(date.getTime())) {
    return '未知时间';
  }

  return formatDateTime(date, false);
};

/**
 * 格式化时间（精确到秒，用于鼠标悬停显示）
 * @param dateString - 日期字符串
 * @returns 格式化后的时间字符串（包含秒，使用中国时区）
 */
export const formatTimeWithSeconds = (dateString: string): string => {
  const date = new Date(dateString);

  if (isNaN(date.getTime())) {
    return '未知时间';
  }

  return formatDateTime(date, true);
};

/**
 * 格式化日期时间（使用中国时区）
 * @param date - 日期对象
 * @param includeSeconds - 是否包含秒
 * @returns 格式化后的日期时间字符串
 */
const formatDateTime = (date: Date, includeSeconds: boolean): string => {
  // 使用中国时区（Asia/Shanghai）格式化时间
  const options: Intl.DateTimeFormatOptions = {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Shanghai',
  };

  if (includeSeconds) {
    options.second = '2-digit';
  }

  try {
    // 使用toLocaleString确保使用中国时区
    const formatted = date.toLocaleString('zh-CN', options);
    // 将格式从 "2024/01/01 01:00:00" 转换为 "2024-01-01 01:00:00"
    return formatted.replace(/\//g, '-');
  } catch (error) {
    console.error('[formatTime] 格式化时间失败:', error);
    // 降级到本地时间
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');

    if (includeSeconds) {
      return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
    } else {
      return `${year}-${month}-${day} ${hours}:${minutes}`;
    }
  }
};

/**
 * 格式化时间用于会话列表（简短格式，使用中国时区）
 * @param dateString - 日期字符串
 * @returns 格式化后的时间字符串（如：今天 14:30、昨天 09:15、12-25 18:00）
 */
export const formatConversationTime = (dateString: string): string => {
  const date = new Date(dateString);

  if (isNaN(date.getTime())) {
    return '';
  }

  // 使用 Intl.DateTimeFormat 获取中国时区的日期时间
  const formatter = new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });

  // 获取当前时间在中国时区的日期
  const now = new Date();
  const nowParts = formatter.formatToParts(now);
  const nowYear = parseInt(nowParts.find((p) => p.type === 'year')?.value || '0');
  const nowMonth = parseInt(nowParts.find((p) => p.type === 'month')?.value || '0') - 1;
  const nowDay = parseInt(nowParts.find((p) => p.type === 'day')?.value || '0');

  const today = new Date(nowYear, nowMonth, nowDay);
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  // 获取消息时间在中国时区的日期时间
  const dateParts = formatter.formatToParts(date);
  const dateYear = parseInt(dateParts.find((p) => p.type === 'year')?.value || '0');
  const dateMonth = parseInt(dateParts.find((p) => p.type === 'month')?.value || '0') - 1;
  const dateDay = parseInt(dateParts.find((p) => p.type === 'day')?.value || '0');
  const dateHours = parseInt(dateParts.find((p) => p.type === 'hour')?.value || '0');
  const dateMinutes = parseInt(dateParts.find((p) => p.type === 'minute')?.value || '0');

  const dateInChina = new Date(dateYear, dateMonth, dateDay, dateHours, dateMinutes);

  // 格式化时间
  const hours = String(dateHours).padStart(2, '0');
  const minutes = String(dateMinutes).padStart(2, '0');
  const timeStr = `${hours}:${minutes}`;

  // 检查是否是今天
  if (dateInChina >= today) {
    return timeStr;
  }

  // 检查是否是昨天
  if (dateInChina >= yesterday) {
    return `昨天 ${timeStr}`;
  }

  // 检查是否是今年
  if (dateYear === nowYear) {
    const month = String(dateMonth + 1).padStart(2, '0');
    const day = String(dateDay).padStart(2, '0');
    return `${month}-${day} ${timeStr}`;
  }

  // 其他情况显示完整日期
  const year = dateYear;
  const month = String(dateMonth + 1).padStart(2, '0');
  const day = String(dateDay).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

/**
 * 将日期字符串转换为时间戳（毫秒）
 * @param dateString - 日期字符串
 * @returns 时间戳（毫秒）
 */
export const dateToTimestamp = (dateString: string): number => {
  const date = new Date(dateString);
  if (isNaN(date.getTime())) {
    return Date.now();
  }
  return date.getTime();
};

/**
 * 格式化消息分割线时间（用于消息之间的时间提示）
 * @param dateString - 日期字符串
 * @returns 格式化后的时间字符串（如：14:30、昨天 09:15、12-25 18:00）
 */
export const formatTimeDivider = (dateString: string): string => {
  return formatConversationTime(dateString);
};

/**
 * 计算哪些消息索引前需要显示时间分割线
 * 规则：
 * 1. 与上一条消息超过5分钟则显示时间
 * 2. 距离上一次时间提示超过30条消息且超过15分钟则显示时间
 * @param messages - 消息列表，每条消息需要有 created_at 字段
 * @returns Map<messageIndex, formattedTime>
 */
export const computeTimeDividers = (
  messages: Array<{ id: string; created_at?: string; createdAt?: string }>
): Map<number, string> => {
  const dividers = new Map<number, string>();
  if (messages.length === 0) return dividers;

  let lastDividerIndex = -1;

  for (let i = 1; i < messages.length; i++) {
    const prevCreatedAt = messages[i - 1]!.created_at || messages[i - 1]!.createdAt;
    const currCreatedAt = messages[i]!.created_at || messages[i]!.createdAt;
    const prevTime = new Date(prevCreatedAt!).getTime();
    const currTime = new Date(currCreatedAt!).getTime();
    const diffMinutes = (currTime - prevTime) / 60000;
    const messagesSinceDivider = i - lastDividerIndex - 1;

    if (diffMinutes > 5 || (messagesSinceDivider >= 30 && diffMinutes > 15)) {
      dividers.set(i, formatTimeDivider(currCreatedAt!));
      lastDividerIndex = i;
    }
  }

  return dividers;
};
