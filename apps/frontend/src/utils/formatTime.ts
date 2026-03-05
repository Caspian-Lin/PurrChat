/**
 * 格式化时间（精确到分钟）
 * @param dateString - 日期字符串
 * @returns 格式化后的时间字符串
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
 * @returns 格式化后的时间字符串（包含秒）
 */
export const formatTimeWithSeconds = (dateString: string): string => {
  const date = new Date(dateString);

  if (isNaN(date.getTime())) {
    return '未知时间';
  }

  return formatDateTime(date, true);
};

/**
 * 格式化日期时间
 * @param date - 日期对象
 * @param includeSeconds - 是否包含秒
 * @returns 格式化后的日期时间字符串
 */
const formatDateTime = (date: Date, includeSeconds: boolean): string => {
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
};
