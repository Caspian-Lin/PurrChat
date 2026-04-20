# PurrChat 设计系统规范

> **设计理念：Soft Architecture（柔软建筑）**
> PurrChat 不是工具，是空间。一个安静、精致的空间，对话在其中自然流淌——无论对方是人还是 AI。

---

## 目录

1. [设计理念](#1-设计理念)
2. [色彩系统](#2-色彩系统)
3. [字体系统](#3-字体系统)
4. [圆角系统](#4-圆角系统)
5. [阴影系统](#5-阴影系统)
6. [间距系统](#6-间距系统)
7. [动效系统](#7-动效系统)
8. [组件规范](#8-组件规范)
9. [主题实现](#9-主题实现)
10. [实施指南](#10-实施指南)

---

## 1. 设计理念

### 1.1 品牌关键词

**Intimate · Refined · Alive**

| 关键词            | 含义                       | 设计表达                               |
| ----------------- | -------------------------- | -------------------------------------- |
| **Intimate** 亲密 | 人与人、人与 AI 的情感空间 | 柔软的形状、温暖的色彩、充裕的留白     |
| **Refined** 精致  | 每个细节都经过斟酌         | 统一的圆角系统、克制的阴影、精确的间距 |
| **Alive** 鲜活    | 有呼吸感的环境             | 微妙的动效、自然的色彩、有机的形状     |

### 1.2 设计原则

#### 原则一：Quiet Confidence（安静自信）

不做视觉上的大喊大叫。层级通过克制而非音量建立。

- 标题不需要巨大的字号来彰显重要性——它只需要比正文大得恰到好处
- 强调色只在真正需要引导注意力的地方出现——10% 的面积，100% 的存在感
- 留白不是"空"，而是"安静的力量"

#### 原则二：Living Geometry（生命几何）

统一的圆角系统创造视觉和谐。没有尖锐的直角，没有随机的混合。

- 所有交互元素共享同一套圆角语言
- 圆角值反映元素层级：越大越"容器"，越小越"内容"
- 形状传递情感：圆润 = 亲近，不是"可爱的圆润"，而是"从容的圆润"

#### 原则三：Substance Over Surface（实质胜于表面）

每一个设计元素都必须 earns its place。

- 不添加"装饰性"元素——如果它不服务于功能或愉悦感，删除它
- 每一行代码、每一个 CSS 属性都要问：这为什么存在？
- 最好的设计是用户不会注意到的设计

#### 原则四：Breathing Space（呼吸空间）

充裕的留白让内容有空间呼吸。

- 聊天应用特别容易拥挤——PurrChat 反其道而行
- 消息之间、面板之间、元素之间都有充裕的间距
- "chill" 不只是氛围，更是物理空间上的舒适感

#### 原则五：Material Honesty（材料诚实）

表面感觉像真实材料，而非数字抽象。

- 柔和的阴影暗示深度，不是悬浮在虚空中
- 色彩有自然质感，不是合成的荧光色
- 深色模式像黄昏的天空，不是纯黑的虚空

### 1.3 反模式（绝不使用的）

- ❌ 左侧色条（`border-left: 3px solid color`）
- ❌ 渐变文字（`background-clip: text`）
- ❌ 满屏毛玻璃效果（glassmorphism）
- ❌ 紫色到蓝色的渐变
- ❌ 深色背景 + 霓虹发光效果
- ❌ 通用圆角矩形 + 标准投影（"AI 设计"的标志）
- ❌ 同一页面内混合直角和圆角
- ❌ bounce/elastic 动效
- ❌ 每个区块都是卡片（不是所有内容都需要容器）

---

## 2. 色彩系统

### 2.1 设计哲学

PurrChat 的色彩像自然材料：亚麻布的温暖、石头的沉稳、黄昏天空的深邃。

- **中性色带有微妙的暖调**——像未漂白的纸张和天然石材，不是冷冰冰的数字灰
- **强调色低饱和、高品味**——每个颜色都像矿物颜料，不是荧光色
- **暗色模式使用蓝调深灰**——像暮色天空，不是纯黑的虚空

### 2.2 中性色板

#### 亮色模式

| Token                 | 色值      | 描述             | 用途                     |
| --------------------- | --------- | ---------------- | ------------------------ |
| `--background`        | `#F7F5F2` | 温暖的亚麻白     | 页面底色                 |
| `--surface`           | `#EEEAE5` | 浅灰褐           | 侧栏、面板底色           |
| `--surface-secondary` | `#F4F1EC` | 稍亮的暖灰       | 嵌套容器、列表背景       |
| `--surface-tertiary`  | `#E8E4DE` | 中灰褐           | hover 状态、分割区域     |
| `--surface-hover`     | `#E2DDD7` | hover 高亮       | 可交互元素 hover         |
| `--strong-surface`    | `#FFFFFF` | 纯白（唯一纯色） | 消息气泡、卡片、输入框底 |
| `--text`              | `#1C1917` | 深褐黑           | 主文本（stone-900 基调） |
| `--text-secondary`    | `#57534E` | 中褐灰           | 次要文本、说明文字       |
| `--text-tertiary`     | `#A8A29E` | 浅褐灰           | 占位符、禁用文字、时间戳 |
| `--border`            | `#D6D3CE` | 暖灰边框         | 分割线、边框             |
| `--border-subtle`     | `#E7E5E0` | 极淡边框         | 不需要强调的边界         |

#### 暗色模式

| Token                 | 色值      | 描述       | 用途                     |
| --------------------- | --------- | ---------- | ------------------------ |
| `--background`        | `#111116` | 深蓝炭     | 页面底色（蓝调而非纯灰） |
| `--surface`           | `#1A1A22` | 蓝灰深色   | 侧栏、面板底色           |
| `--surface-secondary` | `#161620` | 稍暗的蓝灰 | 嵌套容器                 |
| `--surface-tertiary`  | `#222230` | 中蓝灰     | hover 状态、分割区域     |
| `--surface-hover`     | `#282838` | hover 高亮 | 可交互元素 hover         |
| `--strong-surface`    | `#22222C` | 稍亮蓝灰   | 消息气泡、卡片底色       |
| `--text`              | `#ECEBE8` | 温暖的白   | 主文本                   |
| `--text-secondary`    | `#A1A0A8` | 冷灰       | 次要文本                 |
| `--text-tertiary`     | `#65656E` | 暗灰       | 占位符、时间戳           |
| `--border`            | `#2A2A36` | 蓝灰边框   | 分割线、边框             |
| `--border-subtle`     | `#1F1F2A` | 极淡边框   | 不需要强调的边界         |

**设计决策**：暗色模式的背景使用微妙的蓝调（`#111116` 而非 `#121212`），创造类似暮色天空的深邃感。这是让暗色模式有"灵魂"而非"关灯"的关键区别。

### 2.3 强调色方案

PurrChat 提供 8 种精心调配的强调色。每种颜色都经过低饱和处理，像矿物颜料一样自然。

| 名称                | Primary   | Secondary | 情感关键词       |
| ------------------- | --------- | --------- | ---------------- |
| **Sage** 🌿（默认） | `#5A8F4E` | `#E8F0E5` | 宁静、智慧、自然 |
| **Iris** 💜         | `#8B6FC0` | `#EDE8F5` | 想象、创造、灵感 |
| **Ocean** 🌊        | `#4A7FA8` | `#E5EEF5` | 深度、探索、沉思 |
| **Ember** 🔥        | `#BF7B3D` | `#F5ECE0` | 温暖、热情、活力 |
| **Rose** 🌸         | `#BF6B7E` | `#F5E5E9` | 亲密、温柔、情感 |
| **Slate** 🪨        | `#64748B` | `#E8EDF2` | 沉稳、从容、可靠 |
| **Clay** 🏺         | `#B86B50` | `#F5E5DE` | 大地、真实、踏实 |
| **Honey** 🍯        | `#A68B3D` | `#F5F0E0` | 优雅、珍贵、温暖 |

**使用规则**：

- `primary` 用于需要高对比度的场景（按钮文字、图标、active 状态）
- `secondary` 用于低对比度的背景色（选中项底色、标签底色、浅色装饰）
- primary 色值满足 WCAG AA（在亮色背景上 ≥ 4.5:1 对比度）
- 动态混色（消息气泡底色）使用 primary @ 8% 透明度（亮色）/ 15% 透明度（暗色）

### 2.4 语义色

| 状态    | 前景色    | 亮色底色  | 暗色底色  |
| ------- | --------- | --------- | --------- |
| Success | `#16A34A` | `#E8F5E9` | `#1A2E1A` |
| Warning | `#D97706` | `#FFF8E1` | `#2E2510` |
| Error   | `#DC2626` | `#FEE2E2` | `#2E1A1A` |
| Info    | `#2563EB` | `#E3F2FD` | `#1A2530` |

### 2.5 与当前系统的对比

| 方面       | 当前                 | 新设计                             |
| ---------- | -------------------- | ---------------------------------- |
| 亮色底色   | `#faf9f7` 纯暖灰     | `#F7F5F2` 更有质感的亚麻白         |
| 暗色底色   | `#121212` 纯灰       | `#111116` 蓝调深灰，有灵魂         |
| 默认强调色 | `#bf5eff` 高饱和紫   | `#5A8F4E` 低饱和鼠尾草绿           |
| 强调色数量 | 7 种                 | 8 种（新增 Slate）                 |
| 强调色风格 | 高饱和、科技感       | 低饱和、矿物感                     |
| 边框       | `#e0ddd8` 单一边框色 | 两级边框（border + border-subtle） |

---

## 3. 字体系统

### 3.1 设计哲学

字体是界面"说话的口音"。PurrChat 的口音是：清晰但不无聊，友好但不幼稚。

- **正文**：几何无衬线，略带有机感——精确但不机械
- **标题/品牌**：有独特性格的字体——让人记住"这是 PurrChat"
- **CJK 文字**：保持 Noto Sans SC——中文 web 字体的最优选择

### 3.2 字体选择

#### 正文：Onest

```
font-family: 'Onest', 'Noto Sans SC', -apple-system, BlinkMacSystemFont, sans-serif;
```

**为什么选 Onest**：

- 几何结构但有温暖的终端处理（g、a、y 等字母有微妙的曲线变化）
- 优秀的 x-height 保证聊天消息的长时间阅读舒适度
- 变量字体支持，加载高效
- 在"常见字体"和"怪异字体"之间找到完美平衡
- 支持 100–900 全部字重

**加载**：

```html
<link
  href="https://fonts.googleapis.com/css2?family=Onest:wght@400;500;600;700&display=swap"
  rel="stylesheet"
/>
```

#### 标题/品牌：Bricolage Grotesque

```
font-family: 'Bricolage Grotesque', 'Onest', sans-serif;
```

**为什么选 Bricolage Grotesque**：

- 有独特的"有机几何"品质——像手工制作但在数字世界出生
- 变量字体支持宽度和字重变化，适合从 logo 到标题的各种场景
- 在 Google Fonts 可用，加载便捷
- 让 PurrChat 的品牌视觉一眼可辨

**加载**：

```html
<link
  href="https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wght@12..96,400;12..96,600;12..96,700;12..96,800&display=swap"
  rel="stylesheet"
/>
```

#### 等宽字体（代码）：保持不变

```
font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
```

### 3.3 排版层级

使用固定 `rem` 尺寸（产品 UI 不使用 fluid type）：

| 层级      | 字号               | 字重 | 行高 | 用途                 |
| --------- | ------------------ | ---- | ---- | -------------------- |
| `display` | `2rem` (32px)      | 800  | 1.2  | 品牌标识、404 页     |
| `h1`      | `1.5rem` (24px)    | 700  | 1.25 | 页面标题             |
| `h2`      | `1.25rem` (20px)   | 600  | 1.3  | 模态框标题、区域标题 |
| `h3`      | `1.125rem` (18px)  | 600  | 1.35 | 子标题、设置项       |
| `body`    | `0.9375rem` (15px) | 400  | 1.55 | 聊天消息、正文       |
| `body-sm` | `0.875rem` (14px)  | 400  | 1.5  | 列表项、次要正文     |
| `caption` | `0.75rem` (12px)   | 500  | 1.4  | 时间戳、标签、徽标   |
| `micro`   | `0.6875rem` (11px) | 500  | 1.3  | 极小标注             |

**字号比例**：相邻层级之间的比例约为 1.12–1.25，创造清晰的视觉层级但不突兀。

**行高规则**：

- 正文（body）：1.55（聊天消息需要舒适的行距）
- 标题：1.2–1.35（标题行距紧凑）
- 暗色模式：在上述基础上 +0.05（浅色文字在深色背景上需要更多呼吸空间）

**最大行宽**：聊天消息区域 `max-width: 75ch`，确保长文本的可读性。

### 3.4 与当前系统的对比

| 方面         | 当前                       | 新设计                                |
| ------------ | -------------------------- | ------------------------------------- |
| 英文正文字体 | Inter（AI 设计最常用字体） | Onest（独特但不怪异）                 |
| 品牌字体     | 无（Inter 兼任）           | Bricolage Grotesque（独特的品牌标识） |
| 正文字号     | `1rem` (16px)              | `0.9375rem` (15px)（更精致）          |
| 行高         | `1.5`（全局固定）          | 分层级：1.2–1.55                      |
| 字重范围     | 400–700                    | 400–800（增加 display 层级）          |

---

## 4. 圆角系统

### 4.1 设计哲学

圆角是 PurrChat 最核心的视觉识别元素。统一的圆角系统让界面像"由同一双手设计"——没有尖锐的直角，没有随机的混合。

**核心决策**：PurrChat 取消所有直角元素（分割线/分隔条除外）。所有交互容器和元素都使用统一的圆角。

### 4.2 圆角尺度

| Token           | 值       | Tailwind 类      | 用途                         |
| --------------- | -------- | ---------------- | ---------------------------- |
| `--radius-xs`   | `4px`    | `rounded-[4px]`  | 徽标、标签、inline code      |
| `--radius-sm`   | `8px`    | `rounded-lg`     | 按钮、输入框、小型卡片       |
| `--radius-md`   | `12px`   | `rounded-xl`     | 下拉菜单、tooltip、头像      |
| `--radius-lg`   | `16px`   | `rounded-2xl`    | 消息气泡、对话列表项、模态框 |
| `--radius-xl`   | `20px`   | `rounded-[20px]` | 大型面板、登录卡片           |
| `--radius-full` | `9999px` | `rounded-full`   | 头像、药丸、状态点           |

### 4.3 应用规则

**容器层级（越大越"容器"）**：

- 登录/注册卡片：`radius-xl` (20px)
- 模态框容器：`radius-lg` (16px)
- 下拉菜单/tooltip：`radius-md` (12px)

**内容层级（越小越"内容"）**：

- 消息气泡：`radius-lg` (16px)
- 对话列表项 hover 背景：`radius-md` (12px)
- 按钮：`radius-sm` (8px)
- 输入框：`radius-sm` (8px)

**不使用圆角**：

- 1px 分割线（Splitter 组件）
- 分割线装饰

### 4.4 与当前系统的对比

| 方面     | 当前                    | 新设计                         |
| -------- | ----------------------- | ------------------------------ |
| 面板容器 | 无圆角（直角）          | `radius-md` ~ `radius-lg`      |
| 模态框   | `rounded-lg` (8px)      | `radius-lg` (16px)             |
| 按钮     | 8px（全局 reset）       | `radius-sm` (8px，但有意选择） |
| 消息气泡 | `rounded-2xl` (16px)    | `radius-lg` (16px，语义一致）  |
| 圆角规范 | 无统一规范，散落各处    | 6 级语义化 token               |
| 一致性   | 直角面板 + 圆角元素混搭 | 全局统一圆角                   |

---

## 5. 阴影系统

### 5.1 设计哲学

阴影不是"悬浮在虚空中"，而是"一层叠在另一层上"。柔和、带有环境色倾向的阴影创造深度感，而非发光效果。

**核心决策**：

- 使用单层阴影而非多层复合阴影（更干净）
- 阴影颜色跟随表面色（不是纯黑投影）
- 暗色模式大幅降低阴影强度（深色背景自带深度感）
- 更柔和、更大面积的阴影（premium 的标志）

### 5.2 阴影尺度

#### 亮色模式

| Token         | 值                                   | 用途                       |
| ------------- | ------------------------------------ | -------------------------- |
| `--shadow-xs` | `0 1px 2px rgba(28, 25, 23, 0.04)`   | 微妙的层级区分             |
| `--shadow-sm` | `0 2px 8px rgba(28, 25, 23, 0.06)`   | 列表项 hover、小型浮动元素 |
| `--shadow-md` | `0 4px 16px rgba(28, 25, 23, 0.08)`  | 下拉菜单、弹出层           |
| `--shadow-lg` | `0 8px 32px rgba(28, 25, 23, 0.10)`  | 模态框、大型浮动面板       |
| `--shadow-xl` | `0 16px 48px rgba(28, 25, 23, 0.12)` | 全屏覆盖层、最高层级       |

#### 暗色模式

| Token         | 值                                | 用途           |
| ------------- | --------------------------------- | -------------- |
| `--shadow-xs` | `0 1px 2px rgba(0, 0, 0, 0.15)`   | 微妙的层级区分 |
| `--shadow-sm` | `0 2px 8px rgba(0, 0, 0, 0.20)`   | 列表项 hover   |
| `--shadow-md` | `0 4px 16px rgba(0, 0, 0, 0.25)`  | 下拉菜单       |
| `--shadow-lg` | `0 8px 32px rgba(0, 0, 0, 0.30)`  | 模态框         |
| `--shadow-xl` | `0 16px 48px rgba(0, 0, 0, 0.35)` | 最高层级       |

### 5.3 使用规则

- 每个元素最多使用一级阴影——叠加阴影不如选择正确的级别
- 嵌套元素不叠加阴影（内层不影，外层影）
- hover 状态从无阴影升级到 `shadow-sm`（不是加大现有阴影）
- 阴影扩散面积（blur）大于偏移量（offset），创造"环境光"而非"定向光"

### 5.4 与当前系统的对比

| 方面     | 当前                         | 新设计                      |
| -------- | ---------------------------- | --------------------------- |
| 阴影层数 | 多层复合（3–4 层 per token） | 单层（更干净）              |
| 阴影颜色 | 纯黑 `rgba(0,0,0,...)`       | 褐色调 `rgba(28,25,23,...)` |
| 阴影级别 | 4 级 (sm/md/lg/xl)           | 5 级（+xs）                 |
| 暗色阴影 | 0.2–0.5 透明度               | 0.15–0.35（更克制）         |
| 阴影扩散 | 紧凑（偏移 > 扩散）          | 宽松（扩散 > 偏移）         |

---

## 6. 间距系统

### 6.1 设计哲学

间距是"chill"的物理基础。充裕的间距让界面有呼吸感，让用户感到从容不迫。

**核心决策**：

- 基于 4px 网格
- 比一般聊天应用更慷慨的间距
- 语义化 token 名称

### 6.2 间距尺度

| Token        | 值     | 用途                     |
| ------------ | ------ | ------------------------ |
| `--space-1`  | `4px`  | 图标与文字间距、紧凑分组 |
| `--space-2`  | `8px`  | 相关元素间最小间距       |
| `--space-3`  | `12px` | 列表项内部间距、小间距   |
| `--space-4`  | `16px` | 标准组件内边距           |
| `--space-5`  | `20px` | 区块内间距               |
| `--space-6`  | `24px` | 区块间间距、模态框内边距 |
| `--space-8`  | `32px` | 大区块间距、面板内边距   |
| `--space-10` | `40px` | 区域间距                 |
| `--space-12` | `48px` | 大区域分隔               |
| `--space-16` | `64px` | 页面级分隔               |

### 6.3 组件间距指南

| 组件           | 内边距      | 元素间距                                |
| -------------- | ----------- | --------------------------------------- |
| 面板 header    | `16px 20px` | —                                       |
| 列表项         | `12px 16px` | —                                       |
| 消息气泡       | `10px 14px` | —                                       |
| 按钮（默认）   | `8px 16px`  | —                                       |
| 按钮（small）  | `6px 12px`  | —                                       |
| 输入框         | `10px 14px` | —                                       |
| 模态框内容     | `24px`      | —                                       |
| 对话列表项间距 | —           | `4px`                                   |
| 消息间距       | —           | `8px`（同发送者）/ `16px`（不同发送者） |

---

## 7. 动效系统

### 7.1 设计哲学

动效传递状态变化，不传递"我很酷"。PurrChat 的动效应该像呼吸一样自然——存在，但不被注意。

**核心决策**：

- 使用 `ease-out-quart` 作为默认缓动（自然减速）
- 交互反馈在 150–250ms 内完成
- 结构变化在 300–500ms 内完成
- 绝不使用 bounce 或 elastic 缓动

### 7.2 缓动函数

| 名称                | 值                               | 用途                     |
| ------------------- | -------------------------------- | ------------------------ |
| `ease-out-quart`    | `cubic-bezier(0.25, 1, 0.5, 1)`  | 默认：hover、focus、展开 |
| `ease-out-quint`    | `cubic-bezier(0.22, 1, 0.36, 1)` | 入场动画                 |
| `ease-out-expo`     | `cubic-bezier(0.16, 1, 0.3, 1)`  | 退出动画（快出）         |
| `ease-in-out-quart` | `cubic-bezier(0.76, 0, 0.24, 1)` | 位置切换、状态变化       |

### 7.3 时长尺度

| Token                | 值      | 用途                   |
| -------------------- | ------- | ---------------------- |
| `--duration-instant` | `100ms` | 按钮按下、active 状态  |
| `--duration-fast`    | `200ms` | hover 变化、focus ring |
| `--duration-normal`  | `300ms` | 模态框出入、下拉展开   |
| `--duration-slow`    | `500ms` | 页面切换、大型结构变化 |

### 7.4 动画模式

| 场景         | 动画                     | 缓动           | 时长  |
| ------------ | ------------------------ | -------------- | ----- |
| 模态框出现   | 淡入 + 轻微缩放 (0.97→1) | ease-out-quart | 300ms |
| 模态框消失   | 淡出 + 轻微缩小 (1→0.97) | ease-out-expo  | 200ms |
| 下拉菜单出现 | 淡入 + 下移 (−8px→0)     | ease-out-quart | 200ms |
| 下拉菜单消失 | 淡出 + 上移 (0→−4px)     | ease-out-expo  | 150ms |
| 列表项 hover | 背景色变化               | ease-out-quart | 200ms |
| 按钮 hover   | 背景色/阴影变化          | ease-out-quart | 200ms |
| 通知出现     | 从右侧滑入               | ease-out-quart | 300ms |
| 通知消失     | 向右侧滑出               | ease-out-expo  | 200ms |
| 消息出现     | 淡入 + 轻微上移          | ease-out-quart | 250ms |

### 7.5 减少动画偏好

所有动画必须尊重 `prefers-reduced-motion: reduce`：

- 在此设置下，所有过渡时间缩短至 0ms
- 使用 `@media (prefers-reduced-motion: reduce)` 覆盖

---

## 8. 组件规范

### 8.1 导航栏（SideNavbar）

当前：60px 宽的垂直图标栏，nav buttons 使用 `rounded-xl`

新设计：

- 宽度保持 60px
- 导航按钮使用 `radius-md` (12px)
- Active 状态：`primary` 色背景 + 白色图标（保持）
- Hover 状态：`surface-hover` 背景（保持）
- Logo "Purr" 使用 Bricolage Grotesque 字体，"Chat" 使用 Onest
- 底部区域（连接状态 + 主题切换 + 头像）保持现有结构
- 整体背景色使用 `--surface`

### 8.2 面板布局

当前：面板容器无圆角，内部元素有各种圆角

新设计：

- 面板之间的分隔线（Splitter）保持 1px 细线，不使用圆角
- 面板内的可滚动区域容器添加 `radius-md` (12px) 的背景色区分
- 面板 header（搜索栏、标题栏）内边距调整为 `16px 20px`
- 列表项之间使用 `space-1` (4px) 间距

### 8.3 消息气泡

当前：`rounded-2xl` (16px)，`shadow-sm`

新设计：

- 圆角：`radius-lg` (16px)（保持）
- 阴影：取消（消息气泡不需要阴影，阴影用于浮层）
- 已发送消息：`primary @ 8%` 透明度底色
- 已接收消息：`--strong-surface` 底色
- 内边距：`10px 14px`
- 最大宽度：`75ch`
- hover 效果：无（消息不是交互元素，除非需要操作菜单）

### 8.4 对话列表项

当前：hover 使用 `bg-hover-bg`，选中使用 `bg-selected-background`

新设计：

- 圆角：`radius-md` (12px)（在列表容器内创建独立的交互区域）
- 选中状态：`primary @ 8%` 底色 + 左侧 3px `primary` 色条（这里使用色条是合理的——它指示"当前选中"，不是装饰）
  - **等一下**——设计原则禁止 `border-left > 1px`。改用：选中项底色加深 + 微弱的 `primary` 色左边框 1px
  - 实际上最好的方式是：选中项使用更深的 `primary @ 12%` 底色，不使用色条
- Hover：`surface-hover` 底色
- 未读徽标：`radius-full` 药丸形，`primary` 色底白字
- 列表项间距：`4px`

### 8.5 按钮

当前：`border-radius: 8px`（全局 reset）

新设计：

**Primary Button（主要按钮）**：

- 背景：`primary` 色
- 文字：白色
- 圆角：`radius-sm` (8px)
- 内边距：`8px 20px`
- hover：opacity 0.9 + `shadow-sm`
- active：轻微内缩（transform: scale(0.98)）+ `shadow-xs`

**Secondary Button（次要按钮）**：

- 背景：`surface-tertiary`
- 文字：`text`
- 圆角：`radius-sm` (8px)
- 内边距：`8px 20px`
- hover：`surface-hover` 底色

**Ghost Button（幽灵按钮）**：

- 背景：透明
- 文字：`text-secondary`
- 圆角：`radius-sm` (8px)
- 内边距：`8px 16px`
- hover：`surface-tertiary` 底色

**Icon Button（图标按钮）**：

- 背景：透明
- 尺寸：`36px × 36px`
- 圆角：`radius-sm` (8px)
- hover：`surface-hover` 底色

### 8.6 输入框

当前：`rounded-lg` (8px)，`--input-background` 底色

新设计：

- 圆角：`radius-sm` (8px)（保持）
- 底色：`--strong-surface`（白色/深灰，而非 surface）
- 边框：`1px solid var(--border)`
- focus：`border` 变为 `primary` 色 + `shadow-xs`（不使用 outline ring）
- 内边距：`10px 14px`
- placeholder：`text-tertiary`

### 8.7 模态框

当前：`rounded-lg` (8px)，`shadow-xl`

新设计：

- 容器圆角：`radius-lg` (16px)
- 阴影：`shadow-lg`
- 底色：`strong-surface`
- 内边距：`space-6` (24px)
- 出现动画：淡入 + 轻微缩放 (0.97→1)
- overlay：`rgba(0, 0, 0, 0.4)`（亮色）/ `rgba(0, 0, 0, 0.6)`（暗色）

### 8.8 自定义滚动条

当前：8px 宽，半透明主题色

新设计：

- 宽度：6px（更精致）
- 轨道：透明
- 滑块：`border-subtle` 色，hover 时加深
- 圆角：`radius-full`（两端圆角的药丸形）
- 只在滚动时显示（hover 区域后出现）

### 8.9 空状态

当前：简单的图标 + 文字提示

新设计：

- 使用 SVG 插图（保持现有的简洁风格）
- 标题使用 `h3` 层级
- 描述使用 `body-sm` + `text-tertiary`
- 整体居中，上下留 `space-16` (64px)

---

## 9. 主题实现

### 9.1 CSS 变量映射

所有设计 token 映射到 CSS 自定义属性，格式为 `--purr-{category}-{name}`：

```css
:root {
  /* 品牌色 */
  --purr-primary: #5a8f4e;
  --purr-primary-secondary: #e8f0e5;

  /* 表面色 */
  --purr-bg: #f7f5f2;
  --purr-surface: #eeeae5;
  --purr-surface-alt: #f4f1ec;
  --purr-surface-raised: #e8e4de;
  --purr-surface-hover: #e2ddd7;
  --purr-surface-strong: #ffffff;

  /* 文本色 */
  --purr-text: #1c1917;
  --purr-text-secondary: #57534e;
  --purr-text-tertiary: #a8a29e;

  /* 边框色 */
  --purr-border: #d6d3ce;
  --purr-border-subtle: #e7e5e0;

  /* 圆角 */
  --purr-radius-xs: 4px;
  --purr-radius-sm: 8px;
  --purr-radius-md: 12px;
  --purr-radius-lg: 16px;
  --purr-radius-xl: 20px;
  --purr-radius-full: 9999px;

  /* 阴影 */
  --purr-shadow-xs: 0 1px 2px rgba(28, 25, 23, 0.04);
  --purr-shadow-sm: 0 2px 8px rgba(28, 25, 23, 0.06);
  --purr-shadow-md: 0 4px 16px rgba(28, 25, 23, 0.08);
  --purr-shadow-lg: 0 8px 32px rgba(28, 25, 23, 0.1);
  --purr-shadow-xl: 0 16px 48px rgba(28, 25, 23, 0.12);

  /* 间距 */
  --purr-space-1: 4px;
  --purr-space-2: 8px;
  /* ... */
}

[data-theme='dark'] {
  --purr-bg: #111116;
  --purr-surface: #1a1a22;
  /* ... */
}
```

### 9.2 向后兼容过渡

为了最小化迁移成本，新的 `--purr-*` 变量将与现有的 `--theme-*` / `--background-color` 等变量并行存在：

1. **阶段一**：引入 `--purr-*` token，在关键组件上使用
2. **阶段二**：逐步将现有变量替换为 `--purr-*` 映射
3. **阶段三**：移除旧的变量名

### 9.3 Tailwind 配置更新

```javascript
// tailwind.config.js
theme: {
  extend: {
    colors: {
      'purr': {
        primary: 'var(--purr-primary)',
        'primary-alt': 'var(--purr-primary-secondary)',
        bg: 'var(--purr-bg)',
        surface: 'var(--purr-surface)',
        // ...
      },
    },
    borderRadius: {
      'purr-xs': 'var(--purr-radius-xs)',
      'purr-sm': 'var(--purr-radius-sm)',
      // ...
    },
    boxShadow: {
      'purr-xs': 'var(--purr-shadow-xs)',
      'purr-sm': 'var(--purr-shadow-sm)',
      // ...
    },
  },
}
```

---

## 10. 实施指南

### 10.1 迁移优先级

| 优先级 | 变更                                    | 影响范围         |
| ------ | --------------------------------------- | ---------------- |
| P0     | 全局 CSS 变量（色彩 + 圆角 + 阴影）     | 所有组件自动继承 |
| P0     | 字体替换（Onest + Bricolage Grotesque） | 全局             |
| P1     | SideNavbar 样式更新                     | 导航             |
| P1     | 消息气泡样式更新                        | 聊天核心体验     |
| P1     | 对话列表项样式更新                      | 聊天列表         |
| P2     | 模态框样式更新                          | 所有模态框       |
| P2     | 按钮/输入框全局 reset 更新              | 全部交互元素     |
| P2     | 滚动条样式更新                          | 所有可滚动区域   |
| P3     | 主题切换器 UI 更新                      | 主题配置         |
| P3     | 登录/注册页更新                         | 认证页面         |
| P3     | 404 页面更新                            | 错误页面         |

### 10.2 兼容性保障

- 所有功能性代码（WebSocket、状态管理、路由等）不做任何改动
- 仅修改 CSS/样式相关的代码
- 保持所有现有组件的 props 和 events 接口不变
- 主题 store 的数据结构保持兼容（`ThemeMode` 和 `ThemeColor` 类型不变）
- 8 种强调色向下兼容旧的 7 种（新增 Slate）

### 10.3 文件修改清单

| 文件                                        | 变更类型                                |
| ------------------------------------------- | --------------------------------------- |
| `index.html`                                | 更新 Google Fonts 链接                  |
| `src/style.css`                             | 重写 CSS 变量、全局 reset、滚动条、阴影 |
| `src/config/theme.ts`                       | 重写色彩配置（中性色 + 强调色）         |
| `src/stores/theme.ts`                       | 更新变量映射逻辑                        |
| `tailwind.config.js`                        | 新增 `--purr-*` token 映射              |
| `src/App.vue`                               | 更新字体声明                            |
| `src/components/home/SideNavbar.vue`        | 圆角 + 间距更新                         |
| `src/components/home/ChatWindow.vue`        | 消息气泡样式更新                        |
| `src/components/home/ConversationList.vue`  | 列表项样式更新                          |
| `src/components/common/BaseModal.vue`       | 模态框圆角 + 阴影更新                   |
| `src/components/common/BaseInput.vue`       | 输入框样式更新                          |
| `src/components/common/CustomScrollbar.vue` | 滚动条宽度更新                          |
| `src/views/LoginView.vue`                   | 登录页样式更新                          |
| `src/views/RegisterView.vue`                | 注册页样式更新                          |
| `src/views/NotFoundView.vue`                | 404 页样式更新                          |
| `src/components/ThemeSwitcher.vue`          | 色板 UI 更新                            |
| `src/components/DynamicBackground.vue`      | 动画背景颜色适配                        |

---

> **PurrChat Design System v1.0**
> 最后更新：2026-04-17
> 设计理念：Soft Architecture（柔软建筑）
> 品牌关键词：Intimate · Refined · Alive
