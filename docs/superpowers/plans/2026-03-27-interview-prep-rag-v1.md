# Interview Prep RAG V1 Implementation Plan

> **For agentic workers:** 推荐按 sprint 顺序执行，不要并行硬啃所有子系统。每完成一个任务就更新状态，并先跑该任务要求的最小验证，再进入下一个任务。

**Date:** 2026-03-27

**Goal:** 为 `Trace2Offer` 增加一个围绕 `lead` 的 `Interview Prep` 核心模块，支持知识库维护、RAG 出题、提交答案并评分、单题参考答案生成，以及最小历史沉淀能力。

**Scope Boundary:** 一期只做确定性工作流 RAG，不做 agentic RAG，不做语音答题，不做题目图谱，不做复杂面试状态机。

**Why This Fits This Repo**

- 当前产品已经有 `lead / JD / resume / user profile / agent` 这些备面所需事实源。
- 当前技术栈是 `Go + Gin + 本地文件存储 + Next.js`，适合先做轻量本地知识库与工作流式 RAG。
- 当前 agent runtime 更适合消费稳定工具，而不是自己主导多跳检索编排。

---

## Architecture Summary

### Product Shape

- 新增独立页面：`/prep`
- 从 `lead` 列表进入，入口参数是 `lead_id`
- 页面内部包含三块：
  - `资料`
  - `练习`
  - `复盘`

### Technical Approach

- 业务事实源继续使用本地文件
- 向量索引单独使用 SQLite：`prep_index.sqlite`
- RAG 采用固定工作流：
  - 组装上下文
  - 检索知识块
  - 拼 prompt
  - 生成结构化结果
- Agent 集成只做工具包装，不接管底层检索逻辑

### Non-Goals

- 不让 agent 自己决定查几轮、查哪些源
- 不接入外部向量数据库
- 不做自动上网抓公司资料
- 不做多用户 SaaS 级权限体系

---

## Storage Design

### New Files And Directories

- Create: `backend/data/prep/topic_catalog.json`
- Create: `backend/data/prep/knowledge/topics/<topic_key>/*.md`
- Create: `backend/data/prep/knowledge/companies/<company_slug>/*.md`
- Create: `backend/data/prep/knowledge/leads/<lead_id>/*.md`
- Create: `backend/data/prep/sessions/<session_id>.json`
- Create: `backend/data/prep/lead_memory.json`
- Create: `backend/data/prep/prep_index.sqlite`

### `topic_catalog.json`

```json
{
  "topics": [
    {
      "key": "rag",
      "name": "RAG",
      "description": "检索增强生成",
      "created_at": "2026-03-27T10:00:00Z",
      "updated_at": "2026-03-27T10:00:00Z"
    }
  ]
}
```

### `sessions/<session_id>.json`

```json
{
  "id": "prep_01",
  "lead_id": "lead_123",
  "company": "OpenAI",
  "position": "Agent Engineer",
  "status": "draft",
  "config": {
    "topic_keys": ["rag", "system_design"],
    "question_count": 8,
    "include_resume": true,
    "include_profile": true,
    "include_lead_docs": true
  },
  "sources": [],
  "questions": [],
  "answers": [],
  "evaluation": null,
  "reference_answers": {},
  "created_at": "2026-03-27T10:00:00Z",
  "updated_at": "2026-03-27T10:00:00Z"
}
```

### `lead_memory.json`

```json
{
  "leads": {
    "lead_123": {
      "last_session_id": "prep_01",
      "last_avg_score": 6.8,
      "weak_points": [
        {
          "point": "RAG 分块策略说不清",
          "times_seen": 2,
          "last_seen": "2026-03-27T10:00:00Z"
        }
      ],
      "recent_questions": [
        "为什么 RAG 需要 chunk overlap？"
      ],
      "updated_at": "2026-03-27T10:00:00Z"
    }
  }
}
```

### `prep_index.sqlite`

```sql
CREATE TABLE documents (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL,
  scope_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  source_path TEXT,
  content_hash TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE chunks (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL,
  chunk_index INTEGER NOT NULL,
  content TEXT NOT NULL,
  token_count INTEGER,
  embedding BLOB NOT NULL,
  updated_at TEXT NOT NULL
);
```

Notes:

- 业务事实源仍以文件为准，`prep_index.sqlite` 是检索加速层，不是最终事实源。
- `resume`、`user profile`、`lead.JDText` 在索引时转成 synthetic documents，不额外写回新的业务事实文件。

---

## API Surface

### Planned Public APIs

- `GET /api/prep/meta`
- `GET /api/prep/topics`
- `POST /api/prep/topics`
- `PATCH /api/prep/topics/:key`
- `DELETE /api/prep/topics/:key`
- `GET /api/prep/knowledge/:scope/:scope_id/documents`
- `POST /api/prep/knowledge/:scope/:scope_id/documents`
- `PUT /api/prep/knowledge/:scope/:scope_id/documents/:filename`
- `DELETE /api/prep/knowledge/:scope/:scope_id/documents/:filename`
- `GET /api/prep/leads/:lead_id/context-preview`
- `GET /api/prep/index/status`
- `POST /api/prep/index/rebuild`
- `POST /api/prep/retrieval/preview`
- `POST /api/prep/sessions`
- `GET /api/prep/sessions/:session_id`
- `PUT /api/prep/sessions/:session_id/draft-answers`
- `POST /api/prep/sessions/:session_id/submit`
- `POST /api/prep/sessions/:session_id/questions/:question_id/reference-answer`
- `GET /api/prep/leads/:lead_id/summary`
- `GET /api/prep/leads/:lead_id/sessions`

### Planned Internal Agent Tools

- `prep_get_context`
- `prep_generate_questions`
- `prep_get_session_review`
- `prep_reference_answer`

---

## Frontend Surface

### New Route

- Create: `web/app/prep/page.tsx`

### New Component Group

- Create: `web/components/prep/prep-workspace.tsx`
- Create: `web/components/prep/prep-knowledge-sidebar.tsx`
- Create: `web/components/prep/prep-context-preview-card.tsx`
- Create: `web/components/prep/prep-config-panel.tsx`
- Create: `web/components/prep/question-list.tsx`
- Create: `web/components/prep/answer-draft-editor.tsx`
- Create: `web/components/prep/prep-review-panel.tsx`
- Create: `web/components/prep/reference-answer-drawer.tsx`
- Create: `web/components/prep/prep-history-panel.tsx`
- Create: `web/components/prep/lead-prep-summary-card.tsx`

### Existing Components To Modify

- Modify: `web/components/leads-table.tsx`
- Reuse: `web/components/ui/tabs.tsx`
- Reuse: existing `Dialog`, `Drawer/Sheet`, `Button`, `Badge`, `Textarea`, `Card`

### New Client Libraries

- Create: `web/lib/prep-api.ts`
- Create: `web/lib/prep-types.ts`
- Create: `web/lib/prep-store.ts`

---

## Verification Strategy

### Backend

- `cd backend && go test ./...`
- 为 `prep` 模块新增 store/service/router tests
- 扩展后端 API regression tests
- Sprint 4 扩展 `make smoke`

### Frontend

- `cd web && pnpm lint`
- 当前仓库未显式配置前端单测 runner，不在一期里临时引入 `vitest/jest`
- 前端验证以 Playwright e2e 和手工 smoke 为主

### E2E

- Create: `web/playwright.config.ts`
- Create: `web/e2e/prep-navigation.spec.ts`
- Create: `web/e2e/prep-full-flow.spec.ts`

---

## Sprint Plan

一期拆成 4 个 sprint。每个 sprint 按 5 到 7 个工作日估算。

---

## Sprint 1: 底座、入口、知识库维护

**Sprint Goal:** 用户可以从 `lead` 进入备面页，查看本次备面的上下文预览，并维护 topic/company/lead 三类知识文件。

### Task 1.1: `prep` 模块骨架与配置

**Function**

- 建立 `backend/internal/prep` 基础包
- 增加配置读取
- 注册 `/api/prep` 路由骨架

**Files**

- Create: `backend/internal/prep/model.go`
- Create: `backend/internal/prep/config.go`
- Create: `backend/internal/prep/service.go`
- Modify: `backend/internal/api/router.go`
- Modify: `backend/cmd/server/main.go`

**Storage**

- Initialize: `backend/data/prep/`
- Initialize: `backend/data/prep/topic_catalog.json`
- Initialize: `backend/data/prep/knowledge/`
- Initialize: `backend/data/prep/sessions/`

**API**

- Add: `GET /api/prep/meta`

Response example:

```json
{
  "data": {
    "enabled": true,
    "default_question_count": 8,
    "supported_scopes": ["topics", "companies", "leads"]
  }
}
```

**Frontend**

- Create: `web/lib/prep-api.ts`
- Create: `web/lib/prep-types.ts`

**Tests**

- Add: `backend/internal/prep/config_test.go`
- Add: router-level test for `/api/prep/meta`
- Run: `cd backend && go test ./...`

### Task 1.2: Topic Catalog 与知识文档文件存储

**Function**

- 管理 topic 元信息
- 管理三类 scope 下的 Markdown 知识文档

**Files**

- Create: `backend/internal/prep/topic_store.go`
- Create: `backend/internal/prep/knowledge_store.go`
- Create: `backend/internal/prep/topic_store_test.go`
- Create: `backend/internal/prep/knowledge_store_test.go`

**Storage**

- `backend/data/prep/topic_catalog.json`
- `backend/data/prep/knowledge/topics/<topic_key>/*.md`
- `backend/data/prep/knowledge/companies/<company_slug>/*.md`
- `backend/data/prep/knowledge/leads/<lead_id>/*.md`

**API**

- Add: `GET /api/prep/topics`
- Add: `POST /api/prep/topics`
- Add: `PATCH /api/prep/topics/:key`
- Add: `DELETE /api/prep/topics/:key`
- Add: `GET /api/prep/knowledge/:scope/:scope_id/documents`
- Add: `POST /api/prep/knowledge/:scope/:scope_id/documents`
- Add: `PUT /api/prep/knowledge/:scope/:scope_id/documents/:filename`
- Add: `DELETE /api/prep/knowledge/:scope/:scope_id/documents/:filename`

**Frontend**

- Create: `web/components/prep/prep-knowledge-sidebar.tsx`
- Create: `web/components/prep/knowledge-scope-switcher.tsx`
- Create: `web/components/prep/knowledge-document-list.tsx`
- Create: `web/components/prep/knowledge-editor.tsx`

**Tests**

- 文件存储 CRUD roundtrip tests
- API tests for topic/document create-update-delete
- Run: `cd backend && go test ./...`
- Run: `cd web && pnpm lint`

### Task 1.3: Lead Context Resolver

**Function**

- 聚合 `lead.JDText`、简历、用户画像、topic packs、company docs、lead docs
- 返回本次备面可用的上下文概览，不做检索

**Files**

- Create: `backend/internal/prep/context_resolver.go`
- Create: `backend/internal/prep/context_resolver_test.go`

**Storage**

- No new persisted storage

**API**

- Add: `GET /api/prep/leads/:lead_id/context-preview`

Response example:

```json
{
  "data": {
    "lead_id": "lead_123",
    "company": "OpenAI",
    "position": "Agent Engineer",
    "has_resume": true,
    "has_profile": true,
    "topic_keys": ["rag"],
    "sources": [
      {"scope": "lead", "kind": "jd_text", "title": "JD 原文"},
      {"scope": "company", "kind": "markdown", "title": "openai/README.md"}
    ]
  }
}
```

**Frontend**

- Create: `web/components/prep/prep-context-preview-card.tsx`
- Create: `web/components/prep/context-source-list.tsx`
- Create: `web/components/prep/selected-pack-chips.tsx`

**Tests**

- 覆盖“无 JD”“无简历”“无画像”“无匹配 company docs”“topic 为空”
- Run: `cd backend && go test ./...`

### Task 1.4: 备面入口与页面壳子

**Function**

- 在 lead 表格中加入 `备面` 按钮
- 新增 `/prep` 页面与三 tab 空壳

**Files**

- Modify: `web/components/leads-table.tsx`
- Create: `web/app/prep/page.tsx`
- Create: `web/components/prep/prep-workspace.tsx`

**Storage**

- No new storage

**API**

- Reuse: `/api/prep/meta`
- Reuse: `/api/prep/leads/:lead_id/context-preview`

**Frontend**

- Create: `PrepEntryButton`
- Create: `PrepWorkspace`
- Add tabs:
  - `资料`
  - `练习`
  - `复盘`

**Tests**

- Manual smoke:
  1. 打开线索表
  2. 点击任意一条 lead 的 `备面`
  3. 进入 `/prep?lead_id=...`
  4. 页面成功加载 `资料` tab
- Run: `cd web && pnpm lint`

### Task 1.5: Playwright 冒烟基建

**Function**

- 建立前端 e2e 最小能力

**Files**

- Create: `web/playwright.config.ts`
- Create: `web/e2e/prep-navigation.spec.ts`

**Storage**

- No new storage

**API**

- Reuse existing prep APIs

**Frontend**

- No new user-facing component

**Tests**

- Verify:
  1. lead 页面存在 `备面` 按钮
  2. 点击后跳转到 `/prep`
  3. context preview 成功渲染

**Sprint 1 Exit Criteria**

- 能从 `lead` 进入备面页面
- 能查看上下文预览
- 能创建、编辑、删除三类知识文档

---

## Sprint 2: Hugging Face Embedding、可解释检索、出题

**Sprint Goal:** 完成基于 Hugging Face embedding 模型的索引与检索底座，打通“资料入库 -> 切块 -> 向量化 -> 召回 -> 组 prompt -> 生成题目”的可视化链路。用户不仅能生成一轮题并保存答案草稿，还能看到整个 RAG 过程到底用了哪些资料、命中了哪些 chunk、为什么这些内容被带进 prompt。

### Task 2.1: Embedding Provider 抽象、Hugging Face 接入、索引元数据

**Function**

- 抽象 `EmbeddingProvider` interface，不把实现绑死在 OpenAI-compatible 协议上
- 首版接入 Hugging Face embedding 模型，默认使用 `BAAI/bge-m3`
- 默认支持 Hugging Face Inference API，并为后续接自建 Hugging Face TEI endpoint 预留配置口
- 初始化 `prep_index.sqlite`，记录当前索引使用的 embedding model、维度、最近一次 build 状态

**Files**

- Create: `backend/internal/prep/embed_provider.go`
- Create: `backend/internal/prep/embed_huggingface.go`
- Create: `backend/internal/prep/index_store.go`
- Create: `backend/internal/prep/index_store_test.go`
- Create: `backend/internal/prep/embed_huggingface_test.go`
- Modify: `backend/internal/prep/config.go`
- Modify: `backend/cmd/server/main.go`

**Storage**

- Create: `backend/data/prep/prep_index.sqlite`
- Create tables:
  - `documents`
  - `chunks`
  - `index_runs`

**API**

- Add: `GET /api/prep/index/status`

Response should include at least:

- 当前 embedding provider
- 当前 embedding model
- indexed documents / chunks 数
- 最近一次索引时间
- 最近一次索引状态

**Frontend**

- Create: `web/components/prep/index-status-card.tsx`
- Create: `web/components/prep/index-run-summary.tsx`

**Tests**

- Fake HTTP server tests for Hugging Face embedding request/response
- config tests for provider/model/base URL parsing
- SQLite schema creation tests
- Run: `cd backend && go test ./...`

### Task 2.2: 文档 ingestion、切块、自动索引可视化

**Function**

- 对知识文档、`JDText`、简历、画像统一建索引
- 文档变更后可触发局部重建
- 每次重建产出结构化 `index run summary`，让用户看到本次到底索引了多少文档、切了多少 chunk、跳过了哪些未变化文件

**Files**

- Create: `backend/internal/prep/chunker.go`
- Create: `backend/internal/prep/ingestion.go`
- Create: `backend/internal/prep/chunker_test.go`
- Create: `backend/internal/prep/ingestion_test.go`

**Storage**

- Upsert into:
  - `documents`
  - `chunks`
  - `index_runs`

**API**

- Add: `POST /api/prep/index/rebuild`

Request example:

```json
{
  "scope": "topics",
  "scope_id": "rag"
}
```

**Frontend**

- Add `重新索引` button to `资料` tab
- Add `最近索引时间` text
- Add `索引摘要` area，展示：
  - 本次扫描文档数
  - 新增/更新 chunk 数
  - 命中 hash 跳过数
  - 失败文件与原因

**Tests**

- chunk 数与 hash 变化 tests
- 局部重建 tests
- `index run summary` 序列化 tests
- Run: `cd backend && go test ./...`

### Task 2.3: 检索预览升级为可解释 Trace

**Function**

- 输入 query，返回命中的 top-k chunks 与来源
- 除了最终结果，还返回完整检索 trace，至少包含：
  - query 标准化结果
  - scope/topic filter
  - 初始召回候选
  - 去重/截断后的最终上下文
- 每个命中 chunk 需要带上 `score`、`source`、`scope`、`why_selected`
- UI 上明确区分“召回候选”和“最终进入 prompt 的上下文”，别整成一坨列表糊弄人

**Files**

- Create: `backend/internal/prep/retrieval.go`
- Create: `backend/internal/prep/retrieval_test.go`

**Storage**

- Read only from `prep_index.sqlite`

**API**

- Add: `POST /api/prep/retrieval/preview`

Request example:

```json
{
  "lead_id": "lead_123",
  "query": "RAG 常见面试问题",
  "topic_keys": ["rag"],
  "top_k": 5,
  "include_trace": true
}
```

**Frontend**

- Create: `web/components/prep/retrieval-trace-panel.tsx`
- Create: `web/components/prep/retrieval-stage-timeline.tsx`
- Create: `web/components/prep/matched-source-card.tsx`
- Create: `web/components/prep/prompt-context-preview.tsx`

**Tests**

- top-k 排序
- 去重
- scope 过滤
- 空结果
- trace payload contract tests
- Run: `cd backend && go test ./...`
- Run: `cd web && pnpm lint`

### Task 2.4: 生成题目工作流与 RAG 过程展示

**Function**

- 组装上下文
- 执行检索
- 拼 prompt
- 调模型生成结构化题目
- 创建 prep session
- 为本次生成返回 `generation trace`，至少展示：
  - 输入配置快照
  - 检索 query
  - 命中的来源与最终上下文
  - prompt sections 摘要
  - 生成出的题目结果
- 首版只做“阶段级可视化”，不做 token 级 streaming，别一上来把复杂度拉爆

**Files**

- Create: `backend/internal/prep/question_generator.go`
- Create: `backend/internal/prep/prompt.go`
- Create: `backend/internal/prep/question_generator_test.go`
- Create: `backend/internal/prep/session_store.go`
- Create: `backend/internal/prep/session_store_test.go`

**Storage**

- Create session file: `backend/data/prep/sessions/<session_id>.json`
- Persist:
  - `config`
  - `sources`
  - `questions`
  - `status = draft`
  - `trace`

**API**

- Add: `POST /api/prep/sessions`
- Add: `GET /api/prep/sessions/:session_id`

Request example:

```json
{
  "lead_id": "lead_123",
  "topic_keys": ["rag", "system_design"],
  "question_count": 8,
  "include_resume": true,
  "include_profile": true,
  "include_lead_docs": true
}
```

**Frontend**

- Create: `web/components/prep/prep-config-panel.tsx`
- Create: `web/components/prep/question-list.tsx`
- Create: `web/components/prep/prep-run-timeline.tsx`
- Create: `web/components/prep/prep-trace-drawer.tsx`
- Create: `GenerateQuestionsButton`

**Tests**

- parser 测试：模型 JSON 非法时回退报错
- session 创建/读取 API tests
- generation trace contract tests
- Manual smoke:
  1. 进入 `/prep`
  2. 选择 topic
  3. 点击生成题目
  4. 看到阶段时间线与命中来源
  5. 成功看到问题列表

### Task 2.5: 答案草稿保存

**Function**

- 用户输入答案后可保存草稿
- 刷新后仍能恢复

**Files**

- Modify: `backend/internal/prep/session_store.go`
- Create: `web/components/prep/answer-draft-editor.tsx`

**Storage**

- Update `sessions/<session_id>.json.answers`

**API**

- Add: `PUT /api/prep/sessions/:session_id/draft-answers`

Request example:

```json
{
  "answers": [
    {"question_id": 1, "answer": "..." }
  ]
}
```

**Frontend**

- Add auto-save hint
- Add draft answer textarea collection

**Tests**

- session roundtrip tests
- API patch tests
- Playwright:
  1. 输入答案
  2. 保存草稿
  3. 刷新页面
  4. 答案仍存在

**Sprint 2 Exit Criteria**

- 使用 Hugging Face embedding 模型完成索引与检索
- 用户可以看到索引状态、最近一次重建摘要
- 用户可以看到检索 trace，包括候选召回与最终上下文
- 用户可以生成一轮题，并看到本次生成的 RAG 过程摘要
- 用户可以保存答案草稿

---

## Sprint 3: 提交答案、评分、参考答案

**Sprint Goal:** 用户可以完整做题、提交、得到结构化评分，并查看单题参考答案。

### Task 3.1: 提交答案工作流

**Function**

- 将草稿答案转成正式提交
- session 状态从 `draft` 变为 `submitted`

**Files**

- Modify: `backend/internal/prep/service.go`
- Modify: `backend/internal/prep/session_store.go`

**Storage**

- Update `sessions/<session_id>.json.status`
- Add `answers[].submitted_at`

**API**

- Add: `POST /api/prep/sessions/:session_id/submit`

**Frontend**

- Create: `SubmitAnswersButton`
- Create: `SubmitConfirmDialog`

**Tests**

- 重复提交 tests
- 空答案提交 tests
- Run: `cd backend && go test ./...`

### Task 3.2: 逐题检索并评分

**Function**

- 每题单独检索上下文
- 输出结构化评分和 overall

**Files**

- Create: `backend/internal/prep/scoring.go`
- Create: `backend/internal/prep/parser.go`
- Create: `backend/internal/prep/scoring_test.go`
- Create: `backend/internal/prep/parser_test.go`

**Storage**

- Update:
  - `sessions/<session_id>.json.evaluation.scores`
  - `sessions/<session_id>.json.evaluation.overall`

**API**

- Reuse: `POST /api/prep/sessions/:session_id/submit`
- Reuse: `GET /api/prep/sessions/:session_id`

**Frontend**

- Create: `web/components/prep/review-summary-card.tsx`
- Create: `web/components/prep/question-score-card.tsx`
- Create: `web/components/prep/weak-point-list.tsx`

**Tests**

- 部分题未作答
- JSON 解析失败 fallback
- 评分字段缺失 fallback
- Run: `cd backend && go test ./...`

### Task 3.3: 单题参考答案生成

**Function**

- 针对某一道题重新检索上下文
- 生成参考答案并缓存

**Files**

- Create: `backend/internal/prep/reference_answer.go`
- Create: `backend/internal/prep/reference_answer_test.go`

**Storage**

- Update: `sessions/<session_id>.json.reference_answers`

**API**

- Add: `POST /api/prep/sessions/:session_id/questions/:question_id/reference-answer`

Response example:

```json
{
  "data": {
    "question_id": 1,
    "reference_answer": "....",
    "sources": [
      {"title": "JD 原文", "score": 0.82}
    ]
  }
}
```

**Frontend**

- Create: `web/components/prep/reference-answer-drawer.tsx`
- Create: `web/components/prep/reference-source-list.tsx`

**Tests**

- 缓存命中
- 非法 question_id
- 空检索结果
- Run: `cd backend && go test ./...`

### Task 3.4: 复盘页成型

**Function**

- 展示 overall
- 展示逐题评分
- 展示改进建议和参考答案入口

**Files**

- Create: `web/components/prep/prep-review-panel.tsx`
- Create: `web/components/prep/score-badge.tsx`
- Create: `web/components/prep/improvement-checklist.tsx`

**Storage**

- No new storage

**API**

- Reuse session detail API

**Frontend**

- `复盘` tab 完整落地

**Tests**

- Playwright:
  1. 提交后自动进入复盘
  2. 能看到平均分
  3. 能展开某题参考答案

### Task 3.5: 全流程 e2e

**Function**

- 验证 happy path

**Files**

- Create: `web/e2e/prep-full-flow.spec.ts`

**Storage**

- Reuse all prep storage

**API**

- Reuse all prep APIs

**Frontend**

- No new component

**Tests**

- Run:
  - `cd backend && go test ./...`
  - `cd web && pnpm lint`
  - `cd web && pnpm exec playwright test`

**Sprint 3 Exit Criteria**

- 用户能完整完成一轮备面
- 能看到结构化评分
- 能生成单题参考答案

---

## Sprint 4: 历史沉淀、个性化、Agent 接入

**Sprint Goal:** 系统开始记住某条 lead 的历史练习结果，并将稳定能力包装给 agent 使用。

### Task 4.1: Lead 级最小记忆

**Function**

- 保存该 lead 最近 session、平均分、弱点、最近题目

**Files**

- Create: `backend/internal/prep/lead_memory_store.go`
- Create: `backend/internal/prep/lead_memory_store_test.go`

**Storage**

- Create: `backend/data/prep/lead_memory.json`

**API**

- Add: `GET /api/prep/leads/:lead_id/summary`

**Frontend**

- Create: `web/components/prep/lead-prep-summary-card.tsx`

**Tests**

- 弱点累加
- 最近问题去重
- 平均分更新

### Task 4.2: 历史 session 列表与回看

**Function**

- 查看同一条 lead 的历史练习记录

**Files**

- Modify: `backend/internal/prep/session_store.go`
- Create: `web/components/prep/prep-history-panel.tsx`
- Create: `web/components/prep/session-history-list.tsx`

**Storage**

- Reuse `sessions/`

**API**

- Add: `GET /api/prep/leads/:lead_id/sessions`
- Reuse: `GET /api/prep/sessions/:session_id`

**Tests**

- 按时间倒序
- 空列表
- lead_id 不存在

### Task 4.3: 个性化出题 v1

**Function**

- 出题时注入历史弱点与最近题目
- 实现“避重复 + 优先薄弱点”

**Files**

- Modify: `backend/internal/prep/question_generator.go`
- Modify: `backend/internal/prep/prompt.go`

**Storage**

- Read from `lead_memory.json`

**API**

- No new public API

**Frontend**

- Add `本轮优先补强点` banner

**Tests**

- 历史题目不重复
- 弱点优先命中 prompt
- Run: `cd backend && go test ./...`

### Task 4.4: Agent Tool 包装

**Function**

- 将 prep 稳定工作流包装为 agent tool

**Files**

- Create: `backend/agent/tool/prep.go`
- Create: `backend/agent/tool/prep_test.go`
- Modify: `backend/agent/bootstrap.go`

**Storage**

- Reuse prep service and session files

**API**

- No new public API

**Agent Tools**

- `prep_get_context`
- `prep_generate_questions`
- `prep_get_session_review`
- `prep_reference_answer`

**Tests**

- fake prep service tool tests
- existing agent runtime integration tests

### Task 4.5: 稳态打磨与 smoke 扩展

**Function**

- 打磨 loading / empty / error 状态
- 将 prep 流程纳入 smoke

**Files**

- Modify: `backend/Makefile`
- Modify: relevant prep UI components
- Create if needed:
  - `web/components/prep/prep-empty-state.tsx`
  - `web/components/prep/prep-error-state.tsx`
  - `web/components/prep/index-warning-banner.tsx`

**Storage**

- No new storage

**API**

- No new public API

**Tests**

- Extend smoke:
  1. 创建测试 lead
  2. 写入最小知识文档
  3. 生成题目
  4. 提交答案
  5. 拉取 session 详情
- Run:
  - `cd backend && make smoke`
  - `cd web && pnpm exec playwright test`

**Sprint 4 Exit Criteria**

- 同一条 lead 的第二轮练习能体现历史上下文
- agent 能调用 prep 相关工具
- prep 流程进入项目稳定能力集合

---

## Delivery Sequence

实现顺序必须严格遵守：

1. 先做知识库维护与上下文预览
2. 再做索引、检索与出题
3. 再做评分与参考答案
4. 最后做历史沉淀与 agent 接入

不要先做：

- 复杂个性化
- agent 多跳检索
- 自动抓外部资料

那会把一期直接搞成烂尾工程。

---

## Final Acceptance Criteria

- 用户能从任意 `lead` 进入 `/prep`
- 用户能管理 topic/company/lead 三类知识文档
- 用户能基于 `JD + resume + profile + knowledge docs` 生成一轮题
- 用户能提交答案并获得结构化评分
- 用户能查看单题参考答案
- 系统能沉淀 lead 级最小记忆
- agent 可以调用 prep 能力，但不主导底层 RAG

---

## Recommended Commit Strategy

- Sprint 1 完成后单独提交
- Sprint 2 完成后单独提交
- Sprint 3 完成后单独提交
- Sprint 4 完成后单独提交

每个 sprint 内部也可以按任务拆 commit，但不要把跨 sprint 的内容混到一个提交里。
