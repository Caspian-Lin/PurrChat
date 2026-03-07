import { describe, it, expect } from 'vitest';
import {
  formatTime,
  formatTimeWithSeconds,
  formatConversationTime,
  dateToTimestamp,
} from '../utils/formatTime';

describe('formatTime', () => {
  it('应该正确格式化UTC时间为中国时区', () => {
    // UTC时间 2024-01-01T01:00:00Z
    // 中国时区（UTC+8）应该是 2024-01-01 09:00
    const result = formatTime('2024-01-01T01:00:00Z');
    expect(result).toBe('2024-01-01 09:00');
  });

  it('应该正确格式化带时区的时间', () => {
    // 已经是中国时区的时间
    const result = formatTime('2024-01-01T09:00:00+08:00');
    expect(result).toBe('2024-01-01 09:00');
  });

  it('应该处理无效日期', () => {
    const result = formatTime('invalid-date');
    expect(result).toBe('未知时间');
  });

  it('应该正确处理跨日时间', () => {
    // UTC时间 2024-01-01T23:00:00Z
    // 中国时区（UTC+8）应该是 2024-01-02 07:00
    const result = formatTime('2024-01-01T23:00:00Z');
    expect(result).toBe('2024-01-02 07:00');
  });

  it('应该正确处理午夜时间', () => {
    // UTC时间 2024-01-01T16:00:00Z
    // 中国时区（UTC+8）应该是 2024-01-02 00:00
    const result = formatTime('2024-01-01T16:00:00Z');
    expect(result).toBe('2024-01-02 00:00');
  });
});

describe('formatTimeWithSeconds', () => {
  it('应该正确格式化包含秒的时间', () => {
    const result = formatTimeWithSeconds('2024-01-01T01:00:30Z');
    expect(result).toBe('2024-01-01 09:00:30');
  });

  it('应该处理无效日期', () => {
    const result = formatTimeWithSeconds('invalid-date');
    expect(result).toBe('未知时间');
  });

  it('应该正确格式化带时区的时间', () => {
    const result = formatTimeWithSeconds('2024-01-01T09:00:30+08:00');
    expect(result).toBe('2024-01-01 09:00:30');
  });
});

describe('formatConversationTime', () => {
  // 注意：这些测试依赖于当前时间，所以结果可能会有变化
  // 在实际测试中，应该使用固定的时间或mock

  it('应该正确格式化今天的消息', () => {
    // 获取当前时间（中国时区）
    const now = new Date();
    const formatter = new Intl.DateTimeFormat('zh-CN', {
      timeZone: 'Asia/Shanghai',
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    });
    const nowParts = formatter.formatToParts(now);
    const nowYear = parseInt(nowParts.find((p) => p.type === 'year')?.value || '0');
    const nowMonth = parseInt(nowParts.find((p) => p.type === 'month')?.value || '0') - 1;
    const nowDay = parseInt(nowParts.find((p) => p.type === 'day')?.value || '0');
    const nowHours = parseInt(nowParts.find((p) => p.type === 'hour')?.value || '0');
    const nowMinutes = parseInt(nowParts.find((p) => p.type === 'minute')?.value || '0');

    // 创建一个今天的中国时区时间（UTC）
    const todayTime = new Date(
      Date.UTC(
        nowYear,
        nowMonth,
        nowDay,
        nowHours - 8, // 转换为UTC
        nowMinutes
      )
    );

    const result = formatConversationTime(todayTime.toISOString());
    // 今天的时间应该只显示时间部分
    expect(result).toMatch(/^\d{2}:\d{2}$/);
  });

  it('应该正确格式化昨天的消息', () => {
    // 获取当前时间（中国时区）
    const now = new Date();
    const formatter = new Intl.DateTimeFormat('zh-CN', {
      timeZone: 'Asia/Shanghai',
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    });
    const nowParts = formatter.formatToParts(now);
    const nowYear = parseInt(nowParts.find((p) => p.type === 'year')?.value || '0');
    const nowMonth = parseInt(nowParts.find((p) => p.type === 'month')?.value || '0') - 1;
    const nowDay = parseInt(nowParts.find((p) => p.type === 'day')?.value || '0');
    const nowHours = parseInt(nowParts.find((p) => p.type === 'hour')?.value || '0');
    const nowMinutes = parseInt(nowParts.find((p) => p.type === 'minute')?.value || '0');

    // 创建一个昨天的中国时区时间（UTC）
    const yesterdayTime = new Date(
      Date.UTC(
        nowYear,
        nowMonth,
        nowDay - 1,
        nowHours - 8, // 转换为UTC
        nowMinutes
      )
    );

    const result = formatConversationTime(yesterdayTime.toISOString());
    // 昨天的时间应该显示"昨天 HH:MM"
    expect(result).toMatch(/^昨天 \d{2}:\d{2}$/);
  });

  it('应该正确格式化今年的消息', () => {
    // 获取当前时间（中国时区）
    const now = new Date();
    const formatter = new Intl.DateTimeFormat('zh-CN', {
      timeZone: 'Asia/Shanghai',
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    });
    const nowParts = formatter.formatToParts(now);
    const nowYear = parseInt(nowParts.find((p) => p.type === 'year')?.value || '0');
    const nowHours = parseInt(nowParts.find((p) => p.type === 'hour')?.value || '0');
    const nowMinutes = parseInt(nowParts.find((p) => p.type === 'minute')?.value || '0');

    // 创建一个今年的较早时间（中国时区，UTC）
    const earlierThisYear = new Date(
      Date.UTC(
        nowYear,
        0, // 1月
        1,
        nowHours - 8, // 转换为UTC
        nowMinutes
      )
    );

    const result = formatConversationTime(earlierThisYear.toISOString());
    // 今年的消息应该显示"MM-DD HH:MM"
    expect(result).match(/^\d{2}-\d{2} \d{2}:\d{2}$/);
  });

  it('应该正确格式化往年的消息', () => {
    // 创建一个往年的时间（中国时区）
    const pastYear = new Date(
      Date.UTC(2023, 0, 1, 1, 0) // 2023-01-01 01:00 UTC = 2023-01-01 09:00 中国时区
    );

    const result = formatConversationTime(pastYear.toISOString());
    // 往年的消息应该显示"YYYY-MM-DD"
    expect(result).toBe('2023-01-01');
  });

  it('应该处理无效日期', () => {
    const result = formatConversationTime('invalid-date');
    expect(result).toBe('');
  });

  it('应该正确处理UTC时间转换为中国时区', () => {
    // 获取当前时间（中国时区）
    const now = new Date();
    const formatter = new Intl.DateTimeFormat('zh-CN', {
      timeZone: 'Asia/Shanghai',
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    });
    const nowParts = formatter.formatToParts(now);
    const nowYear = parseInt(nowParts.find((p) => p.type === 'year')?.value || '0');
    const nowMonth = parseInt(nowParts.find((p) => p.type === 'month')?.value || '0') - 1;
    const nowDay = parseInt(nowParts.find((p) => p.type === 'day')?.value || '0');
    const nowHours = parseInt(nowParts.find((p) => p.type === 'hour')?.value || '0');
    const nowMinutes = parseInt(nowParts.find((p) => p.type === 'minute')?.value || '0');

    // 创建一个今天的中国时区时间（UTC）
    const todayTime = new Date(
      Date.UTC(
        nowYear,
        nowMonth,
        nowDay,
        nowHours - 8, // 转换为UTC
        nowMinutes
      )
    );

    const result = formatConversationTime(todayTime.toISOString());
    // 今天的时间应该只显示时间部分
    expect(result).toMatch(/^\d{2}:\d{2}$/);
  });
});

describe('dateToTimestamp', () => {
  it('应该正确转换日期字符串为时间戳', () => {
    const timestamp = dateToTimestamp('2024-01-01T00:00:00Z');
    expect(timestamp).toBe(1704067200000);
  });

  it('应该处理无效日期', () => {
    const timestamp = dateToTimestamp('invalid-date');
    // 应该返回当前时间戳
    expect(timestamp).toBeGreaterThan(0);
    expect(timestamp).toBeLessThanOrEqual(Date.now());
  });
});

describe('中国时区一致性测试', () => {
  it('所有时间格式化函数应该使用相同的中国时区', () => {
    // 获取当前时间（中国时区）
    const now = new Date();
    const formatter = new Intl.DateTimeFormat('zh-CN', {
      timeZone: 'Asia/Shanghai',
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    });
    const nowParts = formatter.formatToParts(now);
    const nowYear = parseInt(nowParts.find((p) => p.type === 'year')?.value || '0');
    const nowMonth = parseInt(nowParts.find((p) => p.type === 'month')?.value || '0') - 1;
    const nowDay = parseInt(nowParts.find((p) => p.type === 'day')?.value || '0');
    const nowHours = parseInt(nowParts.find((p) => p.type === 'hour')?.value || '0');
    const nowMinutes = parseInt(nowParts.find((p) => p.type === 'minute')?.value || '0');

    // 创建一个今天的中国时区时间（UTC）
    const todayTime = new Date(
      Date.UTC(
        nowYear,
        nowMonth,
        nowDay,
        nowHours - 8, // 转换为UTC
        nowMinutes
      )
    );

    const expectedChinaTime = `${String(nowHours).padStart(2, '0')}:${String(nowMinutes).padStart(2, '0')}`;

    const formatTimeResult = formatTime(todayTime.toISOString());
    const formatTimeWithSecondsResult = formatTimeWithSeconds(todayTime.toISOString());
    const formatConversationTimeResult = formatConversationTime(todayTime.toISOString());

    // 所有结果都应该包含中国时区的时间
    expect(formatTimeResult).toContain(expectedChinaTime);
    expect(formatTimeWithSecondsResult).toContain(expectedChinaTime);
    expect(formatConversationTimeResult).toContain(expectedChinaTime);
  });

  it('应该正确处理跨年UTC时间', () => {
    // UTC时间 2023-12-31T20:00:00Z
    // 中国时区（UTC+8）应该是 2024-01-01 04:00
    const result = formatTime('2023-12-31T20:00:00Z');
    expect(result).toBe('2024-01-01 04:00');
  });

  it('应该正确处理夏令时边界（中国不使用夏令时）', () => {
    // 中国不使用夏令时，所以UTC+8是固定的
    // 测试几个不同的UTC时间，确保都正确转换为中国时区
    const testCases = [
      { utc: '2024-01-01T00:00:00Z', expected: '08:00' },
      { utc: '2024-06-01T00:00:00Z', expected: '08:00' },
      { utc: '2024-12-01T00:00:00Z', expected: '08:00' },
    ];

    testCases.forEach(({ utc, expected }) => {
      const result = formatTime(utc);
      expect(result).toContain(expected);
    });
  });
});
