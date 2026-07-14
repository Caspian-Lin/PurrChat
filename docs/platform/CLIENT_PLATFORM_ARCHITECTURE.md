# PurrChat 多端客户端平台能力层

## 背景

PurrChat 前端同时运行在 Web、Tauri Desktop 和 Tauri Mobile。历史代码把 `isMobile`、窗口宽度、Tauri 运行时和浏览器能力混在一起判断，导致窄桌面窗口进入移动端布局，平板和触摸桌面又缺少清晰边界。

本层对应 GitHub issue #22，只建立平台边界和 Web 默认实现，不实现 SQLite、系统凭证库、原生附件缓存或桌面托盘业务。

## 核心文件

- `apps/frontend/src/platform/types.ts`：`PlatformCapabilities` 与 adapter interface。
- `apps/frontend/src/platform/detection.ts`：可测试的平台检测纯函数。
- `apps/frontend/src/platform/adapters.ts`：Web 默认 adapter 与不支持能力的显式降级。
- `apps/frontend/src/composables/usePlatform.ts`：Vue 响应式包装，保持旧组件使用的 `isMobile` / `isDesktop` 兼容出口。
- `apps/frontend/src-tau/capabilities/`：Tauri 2 capability 文件。
- `apps/frontend/src-tau/tauri.conf.json`：Tauri 安全配置和 CSP。

## 能力模型

`PlatformCapabilities` 按能力域拆分：

- `runtime`：`web` / `tauri`、client、env、是否原生。
- `os`：Windows、macOS、Linux、Android、iOS 或 unknown。
- `viewport`：宽高与 `compact` / `medium` / `expanded`。
- `input`：触摸、主指针、hover。
- `window`：`phone` / `tablet` / `desktop` 与 `mobile` / `tablet` / `desktop` layout。
- `files`：Web 文件选择/下载、拖拽、原生打开/保存/定位能力。
- `notifications`：Web/native 通知与权限状态。
- `tray`：桌面托盘能力。
- `clipboard`：读写剪贴板能力。
- `lifecycle`：visibility、online/offline、native resume、deep link。
- `haptics`：触觉反馈。

## 判断规则

平台判断不能只依赖 user agent：

- 运行时优先使用 `VITE_APP_CLIENT` 与 Tauri 注入对象判断。
- OS 使用 `navigator.userAgentData.platform` / `navigator.platform` / UA 作为组合信号。
- 输入能力使用 `maxTouchPoints`、`(pointer: coarse)`、`(pointer: fine)` 和 `(hover: hover)`。
- viewport 只描述窗口大小，不再等同于设备类型。

典型结果：

- 窄桌面窗口：`viewport.class = compact`，但 `window.deviceType = desktop`，`layoutMode = desktop`。
- 手机：`deviceType = phone`，`layoutMode = mobile`。
- 平板：`deviceType = tablet`，`layoutMode = tablet`。
- Tauri Desktop：`runtime.isNative = true`，`tray.supported = true`，原生文件能力可被后续 adapter 接入。

## Adapter 降级规则

当前提供这些接口：

- `PersistenceAdapter`
- `FileAdapter`
- `CredentialAdapter`
- `NotificationAdapter`
- `LifecycleAdapter`
- `ClipboardAdapter`
- `FeedbackAdapter`

Web 默认实现使用浏览器能力：

- persistence 使用 `localStorage`。
- file 使用 `<input type="file">`、Blob URL 和下载链接。
- credentials 在 Web 默认实现中不保存 secret，`setSecret` 会抛出 unsupported error。
- notification 使用浏览器 Notification API。
- lifecycle 使用 window/document 事件。
- clipboard 使用 Clipboard API，失败时 fallback 到临时 textarea。
- feedback 使用 `navigator.vibrate`，不支持时静默降级。

后续 issue 的接入位置：

- #23：替换 `CredentialAdapter` 为 Tauri 系统凭证库实现。
- #24：替换 Desktop 的 `PersistenceAdapter` 为加密 SQLite 实现。
- #25：替换 Desktop 的 `FileAdapter` 为 Tauri dialog/fs/opener 实现。
- #26：在 Tauri Rust 侧实现 tray、single instance、window state 等 shell 能力。
- #30：扩展 `NotificationAdapter` 与 `LifecycleAdapter` 的原生实现。

## Tauri 安全边界

Tauri 2 通过 capability 文件控制窗口可用权限。当前只启用基础 shell open、notification、clipboard 和 mobile haptics，不再使用 `fs.scope: ["**"]`。

`tauri.conf.json` 中 CSP 明确限制：

- API / WebSocket：本地开发源与 PurrChat API 预留域。
- 图片：self、asset、data、blob、本地服务、PurrChat 域和现有 R2 域。
- 字体：self、data。
- 脚本：self。
- object/base/frame：禁用或限制到 self。

生产部署如果使用新的 API、WebSocket、图片或字体域名，必须同步更新 CSP。
