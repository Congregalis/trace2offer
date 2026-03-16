# Trace2Offer

这是一个“求职线索台”：用一个极简 Web 看板集中管理职位/公司线索与跟进状态（唯一事实源），同时搭配配一个定制化 Agent 帮你管理这个看板，并基于你的偏好持续筛选与优化投递策略。

## WEB 端
web 端只包含两个页面：
- 简单的看板式 UI，有一个表格维护所有线索，用户可编辑
- Agent 对话页面


## 后端
基于 golang 的 web 服务，提供 API 供前端调用，API 包括：
- 线索表格都所有增删改查
- 与 Agent 对话

不需要数据库，所有存储维护在本地文件


## AGENT
基于 golang 实现现代的 AI Agent，实现要遵循以下特性：
- 超轻量级
- 结构清晰
- 可扩展

包含基础模块：
- sessions 管理
- memory 管理
- tool 管理 # 目前只给他内置线索表格的增删改查以及上网查询的 tool
- model provider 管理 # 目前只给他一个 OpenAI Response 的 provider
- AGENTS.md # Agent 行为指南
- HEARTBEAT.md # 周期性任务提示词 (每 30 分钟检查一次)
- IDENTITY.md # Agent 身份设定
- USER.md # 用户偏好

