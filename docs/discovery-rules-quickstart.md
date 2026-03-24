# Discovery Rules Quickstart

`Discovery Rules` 用来告诉 Trace2Offer 去哪里发现新岗位，以及哪些岗位值得进候选池。

如果你是第一次用，最简单的做法不是自己从零填表，而是先添加一条推荐示例规则，再点一次 `立即发现` 看结果。

## What Discovery Rules Do

一条发现规则本质上就是：

- 去哪个 `RSS/Atom` 源抓职位
- 抓到后给这些职位打上什么 `source`
- 没有明确地区时，用什么默认地区
- 哪些关键词命中后保留
- 哪些关键词命中后排除

系统会按规则定时抓取新职位，并把结果写进 `候选池`，而不是直接塞进正式 lead。

## Where To Find RSS/Atom Feeds

最稳的方式是找：

- 招聘站官方文档里的 RSS/Atom
- 明确提供 feed 的远程岗位聚合站
- 有公开订阅地址的分类职位源
- 支持按分类 / 职位类型筛选后输出 feed 的招聘站

不要把普通职位列表页或者公司 careers 页直接填进 `RSS/Atom URL`，那种大概率不是 feed。

当前已经确认可用的示例源：

- `https://himalayas.app/jobs/rss`
- `https://remoteyeah.com/remote-ai-engineer-jobs.xml`
- `https://remoteyeah.com/remote-back-end-jobs.xml`
- `https://weworkremotely.com/categories/remote-back-end-programming-jobs.rss`
- `https://www.smartremotejobs.com/feed/software-development-remote-jobs.rss`
- `https://remotefirstjobs.com/rss/jobs/software-development.rss`
- `https://remotefirstjobs.com/rss/jobs/ai.rss`
- `https://remotefirstjobs.com/rss/jobs/golang.rss`
- `https://www.realworkfromanywhere.com/rss.xml`
- `https://www.realworkfromanywhere.com/remote-backend-jobs/rss.xml`
- `https://www.realworkfromanywhere.com/remote-ai-jobs/rss.xml`
- `https://remoteok.com/rss`
- `https://jobicy.com/feed/job_feed?job_categories=dev&job_types=full-time`
- `https://jobicy.com/feed/job_feed?job_categories=data-science&job_types=full-time`
- `https://www.smartremotejobs.com/feed/devops-remote-jobs.rss`
- `https://www.smartremotejobs.com/feed/data-science-remote-jobs.rss`
- `https://www.v2ex.com/feed/jobs.xml`
- `https://linux.do/c/job/27.rss`
- `https://linux.do/tag/2116-tag/2116.rss`
- `https://ruby-china.org/topics/feed`

## Recommended Starter Feeds

建议先从 3 条开始，不要一口气加太多：

1. `RemoteYeah AI Engineer`
   适合偏 `Agent / LLM / AI Infra` 的岗位。

2. `Himalayas Remote SWE`
   适合拉一个更宽的远程 Software Engineer 基础盘。

3. `RemoteYeah Backend`
   适合把大量“后端 + AI 服务”混合型岗位兜住。

如果你还想补充通用工程岗，再加：

4. `We Work Remotely Backend`
5. `SmartRemoteJobs SWE`

如果你想把国外远程盘子再拉宽一点，可以继续加：

6. `RemoteFirstJobs Software Dev`
7. `RemoteFirstJobs AI`
8. `RemoteFirstJobs Golang`
9. `Real Work From Anywhere Backend`
10. `Jobicy Dev Full-Time`
11. `Remote OK Engineering`（更宽，噪音也更大）
12. `SmartRemoteJobs DevOps`
13. `SmartRemoteJobs Data Science`

如果你也想收一些国内技术社区招聘线索，可以加：

14. `V2EX 酷工作`
15. `LINUX DO 招聘分类`
16. `LINUX DO 招聘标签`
17. `Ruby China Topics`（噪音更大，放最后）

## How To Fill Each Field

### 规则名

给自己看的名字。写人话，别写成 `rule-1` 这种废名。

好例子：

- `RemoteYeah AI Engineer`
- `Himalayas Remote SWE`

### RSS/Atom URL

真正的 feed 地址。

好例子：

- `https://himalayas.app/jobs/rss`
- `https://remoteyeah.com/remote-ai-engineer-jobs.xml`

坏例子：

- `https://himalayas.app/jobs`
- `https://company.com/careers`

### 来源标签

最后会进候选池的 `source` 字段。建议统一写法，方便后面过滤和统计。

推荐格式：

- `himalayas`
- `remoteyeah:ai`
- `remoteyeah:backend`
- `wwr:backend`
- `smartremotejobs:swe`

### 默认地区

当 feed 本身没有明确地区时，用它兜底。

常见写法：

- `Remote`
- `US Remote`
- `Taipei / Remote`

### 包含关键词

只要标题或摘要里命中其中一个，就保留。

对于 `Software Engineer / Agent / AI Infra`，推荐起步关键词：

- `software engineer`
- `backend`
- `platform`
- `ai`
- `llm`
- `agent`
- `inference`
- `rag`
- `golang`
- `python`
- `distributed systems`

### 排除关键词

只要命中其中一个，就直接排除。

推荐起步排除词：

- `intern`
- `frontend`
- `ios`
- `android`
- `sales`
- `marketing`
- `recruiter`
- `wordpress`

## Recommended Keywords

如果你现在的求职方向是：

- `Software Engineer`
- `Agent 开发`
- `AI Infra`
- `LLM 应用后端`

可以直接拿这组：

### 包含关键词

`software engineer, backend, platform, ai, llm, agent, inference, rag, golang, python, distributed systems`

### 排除关键词

`intern, frontend, ios, android, sales, marketing, recruiter, wordpress`

如果你想让国内社区源也能勉强收得住，可以再补一组中文词：

### 国内源建议包含关键词

`招聘, agent, ai, llm, 后端, backend, 全栈, software engineer, golang, python`

### 国内源建议排除关键词

`实习, 销售, 客服, 运营, 测试, 线下活动, 闲聊`

## Common Mistakes

### 1. 把普通网页当 feed

最常见的坑。普通 jobs 页面不是 RSS/Atom，填进去系统抓不到结构化列表。

### 2. 关键词写太宽

如果你把 `engineer`、`developer` 这种过于宽泛的词堆太多，候选池很快会被噪音职位塞满。

### 3. 一上来加太多规则

规则不是越多越好。先跑 2 到 3 条，观察噪音，再调关键词。

### 4. 不写排除词

不加 `intern`、`frontend`、`sales` 这类排除词，候选池质量通常会直线下滑。

### 5. 重复添加同一来源

如果两个规则本质上盯的是同一个 feed，只是名字不同，候选池会产生大量重复感知。优先复用已有规则，再调整关键词。

### 6. 把社区 feed 当成标准 JD 源

像 `V2EX / LINUX DO / Ruby China` 这种源，本质上更像招聘帖子流，不是统一格式的职位平台。

这意味着：

- 质量波动会更大
- 岗位描述可能是自由文本
- 关键词过滤比国外聚合站更重要

## Suggested First Run

第一次上手建议按这个顺序：

1. 添加 `RemoteYeah AI Engineer`
2. 添加 `Himalayas Remote SWE`
3. 点 `立即发现`
4. 打开候选池看结果
5. 如果噪音太多，再缩小包含词或补充排除词

别一开始就想把规则体系设计成宇宙终极形态，那玩意儿一般叫折腾，不叫上手。
