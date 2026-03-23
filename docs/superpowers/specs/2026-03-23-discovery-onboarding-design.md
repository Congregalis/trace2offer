# Discovery Onboarding Design

**Date:** 2026-03-23

**Goal:** 让新用户在不理解 RSS/Atom 细节的前提下，也能快速创建第一批发现规则并跑出候选职位。

## Background

当前发现规则已经具备：

- 规则管理
- 手动执行发现
- heartbeat 周期触发发现
- 候选池自动刷新

但对新用户来说，发现规则面板仍然过于原始：

- 表单字段缺少上下文
- 用户不知道去哪里找可用的 RSS/Atom 源
- 用户不知道应该填什么关键词
- 第一次进入时面对空表单，容易直接放弃

这个问题不是“功能不存在”，而是“首屏引导缺失”。如果不解决，新功能再完整也只是摆设。

## Design Goals

- 让新用户第一次进入候选池时就能直接开始
- 保持现有系统视觉语言，不引入新的主题色或独立 onboarding 风格
- 让示例规则与文档共用一套事实源，避免内容分叉
- 对老用户保留持续可见的补充入口
- 让用户可以“一键添加”示例规则，而不是先被迫理解所有字段

## Non-Goals

- 不新增独立页面
- 不做首次进入向导
- 不实现复杂个性化推荐逻辑
- 不在本次改动中扩展更多发现源抓取能力

## Approved Approach

采用“文档 + 产品内双入口引导 + 一键示例规则”的组合方案。

### 1. Candidate Pool Empty State

当用户尚未创建任何发现规则时，在候选池顶部展示 `快速开始` 区块。

区块内容：

- 简短说明：发现规则是什么、为什么先从示例开始
- `优先推荐` 规则组
- `通用补充` 规则组
- 每张规则卡片支持 `一键添加`
- 明显的 `不会填？看快速上手` 入口

该区块仅在 `rules.length === 0` 时显示，避免老用户被反复打扰。

### 2. Discovery Rules Dialog Persistent Help

在发现规则管理弹窗中，固定保留两块内容：

- `不会填？看快速上手`
- `推荐示例规则`

这部分不依赖现有规则数量，保证老用户在任何时候都能补加示例或查看说明。

### 3. One-Click Preset Rules

推荐示例规则以共享配置数据驱动，不在多个文件里手写多份。

每条示例规则包含：

- 名称
- 来源 URL
- source 标签
- 默认地区
- include keywords
- exclude keywords
- 方向标签
- 一句使用说明
- 分组信息（优先推荐 / 通用补充）

主交互为 `一键添加`。

若同名或同 feed 的示例规则已经存在，则按钮展示为 `已添加` 并禁用，避免重复创建。

### 4. Quickstart Documentation

新增正式文档：

`docs/discovery-rules-quickstart.md`

文档内容覆盖：

- 什么是发现规则
- RSS/Atom 是什么
- 去哪里找 feed
- 当前确认可用的 feed 示例
- 每个字段怎么填
- 推荐关键词模板
- 常见错误与排查

产品内帮助使用该文档的精简版本，而仓库文档保留完整说明。

## Preset Rule Set

首批内置 5 条示例规则。

### Priority Picks

1. RemoteYeah AI Engineer
2. Himalayas Remote SWE
3. RemoteYeah Backend

### General Supplements

4. We Work Remotely Backend
5. SmartRemoteJobs SWE

选择原则：

- 优先推荐更贴近 `Software Engineer / Agent / AI Infra`
- 通用补充用于覆盖更宽的软件工程远程职位
- 数量控制在 5 条，避免新用户面对“示例规则墙”

## Visual Language Constraints

本次改动必须保持当前系统视觉一致性：

- 复用现有 `Dialog`, `Button`, `Badge`, `Input`, `Textarea`, `Table`
- 保持 `bg-card/30`, `border-border`, `muted-foreground` 等现有语义类
- 不引入新的品牌主题色
- 不使用独立 onboarding 页面或营销式插画

用户应该感觉这是现有系统自然长出来的一部分，而不是后贴上的活动页。

## UX Copy Principles

- 说明文字必须短
- 卡片描述用“适合什么方向”而不是解释协议细节
- 文档中讲清楚事实，界面中只给最必要的信息
- 不把专业术语堆在首屏

## Data Flow

### Preset Add Flow

1. 用户点击某个示例规则的 `一键添加`
2. 前端将共享 preset 数据转换为 `DiscoveryRuleMutationInput`
3. 调用现有 `addRule`
4. 刷新规则列表
5. 如果用户处于候选池空状态，可继续点击 `立即发现`

### Quickstart Help Flow

1. 用户点击 `不会填？看快速上手`
2. 打开应用内帮助弹窗或抽屉
3. 查看字段解释、示例 feed、推荐关键词
4. 关闭后继续添加示例或手动创建规则

## Files Likely Affected

### Docs

- `docs/discovery-rules-quickstart.md`

### Shared Frontend Data / Types

- `web/lib/types.ts`
- `web/lib/discovery-store.ts`
- `web/lib/discovery-presets.ts` (new)

### UI

- `web/components/candidates-table.tsx`
- `web/components/discovery-rules-panel.tsx`
- `web/components/discovery-preset-cards.tsx` (new)
- `web/components/discovery-quickstart-dialog.tsx` (new)

## Risks

- 示例规则内容如果在 UI 和文档各维护一份，后续必然分叉
- 如果不做重复判断，一键添加会导致用户批量创建重复规则
- 如果快速开始区块始终显示，会干扰已有经验的用户
- 如果帮助内容太长，会把轻工具做成说明书

## Success Criteria

- 新用户不需要手填任何字段，也能创建第一条规则
- 新用户能在 1 分钟内完成“添加规则 -> 立即发现 -> 看到候选”
- 老用户仍能在规则管理弹窗中补加示例
- 文档与产品内示例保持一致
- 页面视觉与现有系统保持统一
