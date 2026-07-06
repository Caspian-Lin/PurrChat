---
name: project_r2_china_access
description: R2 存储桶在中国大陆的访问连通性问题记录
type: project
originSessionId: 5d95008a-df14-4095-bb6a-21ea79cee9e3
---
## R2 r2.dev 域名在中国大陆无法访问

**事实**: Cloudflare R2 的 `pub-xxx.r2.dev` 公开访问子域名在中国大陆被 GFW 阻断，浏览器报 `ERR_CONNECTION_CLOSED`。即使挂梯子能直接在地址栏访问，`<img>` 标签加载仍可能失败。

**尝试过的方案**:
- `referrerpolicy="no-referrer"` — 未解决，确认是网络层面阻断而非防盗链
- 直接浏览器访问 URL — 需要梯子才能打开

**待办解决方案**（暂未实施）:
- 使用 R2 自定义域名（走 Cloudflare CDN 常规路径，可能改善但不保证）
- 存储服务代理模式（后端从 R2 拉取文件再返回前端，服务器端不受 GFW 影响）
- 切换到国内对象存储（如阿里云 OSS、腾讯 COS）

**Why**: 使用 r2.dev 子域名作为公开访问端点，国内用户无法直连
**How to apply**: 后续考虑部署时需要解决此问题，否则头像和文件功能对国内用户不可用
