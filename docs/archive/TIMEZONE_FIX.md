# 时区问题修复文档

## 问题描述

用户报告了一个时区问题：
- 新发出的消息时间戳是正确的（例如：02:05:47）
- 但刷新页面后，时间戳变成8小时之后的时间（例如：10:05）

## 问题根源

经过分析，发现了两个主要问题：

### 1. 前端时间格式化问题

在 `formatConversationTime` 函数中，代码使用了 `date.toLocaleString('en-US', { timeZone: 'Asia/Shanghai' })` 来获取中国时间，然后又用 `new Date()` 来解析这个字符串。这导致了时区的双重转换。

**错误代码示例：**
```typescript
const nowInChina = new Date(
  now.toLocaleString('en-US', { timeZone: 'Asia/Shanghai' })
);
```

**问题：**
- `toLocaleString` 返回的是格式化后的字符串（例如："3/8/2026, 10:05:47 AM"）
- `new Date()` 会把这个字符串当作本地时间解析
- 如果本地时区是 Asia/Shanghai，那么它会被当作中国时间
- 但实际上它已经是转换后的中国时间了，所以会再次被当作 UTC 时间转换，导致多加了8小时

**修复方法：**
使用 `Intl.DateTimeFormat.formatToParts()` 来获取时区转换后的各个时间部分，然后直接使用这些部分创建新的 Date 对象，避免字符串解析。

**修复后的代码：**
```typescript
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
const nowYear = parseInt(nowParts.find(p => p.type === 'year')?.value || '0');
const nowMonth = parseInt(nowParts.find(p => p.type === 'month')?.value || '0') - 1;
const nowDay = parseInt(nowParts.find(p => p.type === 'day')?.value || '0');
const nowHours = parseInt(nowParts.find(p => p.type === 'hour')?.value || '0');
const nowMinutes = parseInt(nowParts.find(p => p.type === 'minute')?.value || '0');
```

### 2. 后端数据库时区问题

数据库连接字符串没有设置时区参数，导致数据库使用本地时间（中国时间）存储时间戳。

**问题：**
- 数据库使用 `CURRENT_TIMESTAMP`，这会使用数据库服务器的本地时区（可能是中国时区）
- Go 代码设置了 `time.Local = time.UTC`，但这只影响 Go 程序的本地时间，不影响数据库
- 当从数据库读取时间时，Go 的 `time.Time` 会假设时间戳是 UTC 时间，然后序列化为 JSON 时添加 'Z' 后缀
- 但实际上，数据库存储的是本地时间（中国时间），所以序列化后的时间戳是错误的

**示例：**
- 数据库存储：`2026-03-08 02:05:47`（中国时间）
- Go 序列化：`2026-03-08T02:05:47.792969Z`（错误地标记为 UTC）
- 前端解析：认为这是 UTC 时间，转换为中国时间后变成 `10:05:47`（多加了8小时）

**修复方法：**
在数据库连接字符串中添加 `timezone=UTC` 参数，确保数据库使用 UTC 时间存储时间戳。

**修复后的代码：**
```go
func GetDSN(cfg *DBConfig) string {
    // 添加时区参数，确保数据库使用 UTC 时间存储时间戳
    // 这可以避免时区转换问题，确保前端显示的时间戳一致
    return "postgres://" + cfg.User + ":" + cfg.Password + "@" + cfg.Host + ":" + cfg.Port + "/" + cfg.Name + "?timezone=UTC"
}
```

## 修复内容

### 前端修复

1. **修复 `formatConversationTime` 函数**（`apps/frontend/src/utils/formatTime.ts`）
   - 使用 `Intl.DateTimeFormat.formatToParts()` 代替 `toLocaleString()` + `new Date()`
   - 避免时区双重转换

2. **修复测试代码**（`apps/frontend/src/tests/formatTime.test.ts`）
   - 更新测试代码，使用相同的时区处理方法
   - 确保测试的一致性

### 后端修复

1. **修复数据库连接字符串**（`apps/backend/pkg/config/config.go`）
   - 添加 `timezone=UTC` 参数
   - 确保数据库使用 UTC 时间存储时间戳

## 测试

运行以下命令测试修复：

```bash
# 前端测试
cd apps/frontend && npm test -- formatTime.test.ts

# 后端需要重新编译和重启
cd apps/backend
go build -o server cmd/server/main.go
./server
```

## 预期结果

修复后：
1. 新发出的消息时间戳应该正确显示
2. 刷新页面后，时间戳应该保持不变
3. 从缓存加载的消息时间戳应该正确显示
4. 所有时间戳都应该使用中国时区（UTC+8）显示

## 注意事项

1. **数据库迁移**：如果数据库中已经存在使用本地时区存储的时间戳，这些时间戳在修复后可能会显示不正确。需要考虑数据迁移或重新生成时间戳。

2. **时区一致性**：确保所有服务（前端、后端、数据库）都使用相同的时区处理策略（使用 UTC 存储，使用中国时区显示）。

3. **WebSocket 消息**：WebSocket 发送的消息也应该使用相同的时区处理方式。

## 相关文件

- `apps/frontend/src/utils/formatTime.ts` - 前端时间格式化工具
- `apps/frontend/src/tests/formatTime.test.ts` - 前端时间格式化测试
- `apps/backend/pkg/config/config.go` - 后端配置文件
- `apps/backend/cmd/server/main.go` - 后端主程序（已设置 `time.Local = time.UTC`）

## 参考资料

- [Intl.DateTimeFormat - MDN](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat)
- [PostgreSQL Time Zones](https://www.postgresql.org/docs/current/datatype-datetime.html#DATATYPE-TIMEZONES)
- [Go time.Time](https://pkg.go.dev/time#Time)
