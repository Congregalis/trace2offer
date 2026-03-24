import type { DiscoveryRule, DiscoveryRuleMutationInput } from "./types";

export type DiscoveryPresetGroup = "international" | "domestic";

export interface DiscoveryPreset {
  id: string;
  name: string;
  summary: string;
  group: DiscoveryPresetGroup;
  tags: string[];
  rule: DiscoveryRuleMutationInput;
}

export const DISCOVERY_PRESETS: DiscoveryPreset[] = [
  {
    id: "remoteyeah-ai-engineer",
    name: "RemoteYeah AI Engineer",
    summary: "偏 Agent / LLM / AI Infra，适合先拉高相关性。",
    group: "international",
    tags: ["国外远程", "AI", "Agent"],
    rule: {
      name: "RemoteYeah AI Engineer",
      feedUrl: "https://remoteyeah.com/remote-ai-engineer-jobs.xml",
      source: "remoteyeah:ai",
      defaultLocation: "Remote",
      includeKeywords: ["agent", "llm", "ai", "inference", "rag", "backend", "python", "golang"],
      excludeKeywords: ["intern", "recruiter", "sales", "marketing"],
      enabled: true,
    },
  },
  {
    id: "himalayas-remote-swe",
    name: "Himalayas Remote SWE",
    summary: "更宽的远程 Software Engineer 基础盘，适合作为补量入口。",
    group: "international",
    tags: ["国外远程", "SWE", "General"],
    rule: {
      name: "Himalayas Remote SWE",
      feedUrl: "https://himalayas.app/jobs/rss",
      source: "himalayas",
      defaultLocation: "Remote",
      includeKeywords: ["software engineer", "backend", "platform", "ai", "llm", "agent", "distributed systems"],
      excludeKeywords: ["intern", "frontend", "sales", "marketing"],
      enabled: true,
    },
  },
  {
    id: "remoteyeah-backend",
    name: "RemoteYeah Backend",
    summary: "补足后端和平台类岗位，把 AI 服务型职位一并兜住。",
    group: "international",
    tags: ["国外远程", "Backend", "Platform"],
    rule: {
      name: "RemoteYeah Backend",
      feedUrl: "https://remoteyeah.com/remote-back-end-jobs.xml",
      source: "remoteyeah:backend",
      defaultLocation: "Remote",
      includeKeywords: ["backend", "software engineer", "platform", "distributed systems", "golang", "python"],
      excludeKeywords: ["frontend", "ios", "android", "intern"],
      enabled: true,
    },
  },
  {
    id: "wwr-backend",
    name: "We Work Remotely Backend",
    summary: "经典远程后端盘子，适合补充更传统的软件工程岗位。",
    group: "international",
    tags: ["国外远程", "Backend"],
    rule: {
      name: "We Work Remotely Backend",
      feedUrl: "https://weworkremotely.com/categories/remote-back-end-programming-jobs.rss",
      source: "wwr:backend",
      defaultLocation: "Remote",
      includeKeywords: ["software engineer", "backend", "api", "platform", "ai", "llm"],
      excludeKeywords: ["frontend", "wordpress", "intern"],
      enabled: true,
    },
  },
  {
    id: "smartremotejobs-swe",
    name: "SmartRemoteJobs SWE",
    summary: "偏宽的远程软件开发补充源，适合后续扩池。",
    group: "international",
    tags: ["国外远程", "SWE", "Supplement"],
    rule: {
      name: "SmartRemoteJobs SWE",
      feedUrl: "https://www.smartremotejobs.com/feed/software-development-remote-jobs.rss",
      source: "smartremotejobs:swe",
      defaultLocation: "Remote",
      includeKeywords: ["software engineer", "backend", "platform", "ai", "llm", "agent"],
      excludeKeywords: ["intern", "frontend", "design", "marketing"],
      enabled: true,
    },
  },
  {
    id: "remotefirstjobs-ai",
    name: "RemoteFirstJobs AI",
    summary: "官方 AI 分类 feed，适合补更窄一点的 AI Engineer / ML / LLM 岗。",
    group: "international",
    tags: ["国外远程", "AI", "Supplement"],
    rule: {
      name: "RemoteFirstJobs AI",
      feedUrl: "https://remotefirstjobs.com/rss/jobs/ai.rss",
      source: "remotefirstjobs:ai",
      defaultLocation: "Remote",
      includeKeywords: ["agent", "llm", "ai", "machine learning", "ml", "inference", "python", "golang"],
      excludeKeywords: ["intern", "sales", "marketing", "recruiter"],
      enabled: true,
    },
  },
  {
    id: "remotefirstjobs-software-dev",
    name: "RemoteFirstJobs Software Dev",
    summary: "更宽的远程开发岗位盘子，适合补量，但排除词得带上。",
    group: "international",
    tags: ["国外远程", "SWE", "General"],
    rule: {
      name: "RemoteFirstJobs Software Dev",
      feedUrl: "https://remotefirstjobs.com/rss/jobs/software-development.rss",
      source: "remotefirstjobs:software-development",
      defaultLocation: "Remote",
      includeKeywords: ["software engineer", "backend", "platform", "distributed systems", "python", "golang"],
      excludeKeywords: ["intern", "frontend", "design", "marketing"],
      enabled: true,
    },
  },
  {
    id: "remotefirstjobs-golang",
    name: "RemoteFirstJobs Golang",
    summary: "更窄的 Go 方向远程源，适合后端和基础设施岗补洞。",
    group: "international",
    tags: ["国外远程", "Golang", "Backend"],
    rule: {
      name: "RemoteFirstJobs Golang",
      feedUrl: "https://remotefirstjobs.com/rss/jobs/golang.rss",
      source: "remotefirstjobs:golang",
      defaultLocation: "Remote",
      includeKeywords: ["golang", "go developer", "go engineer", "backend", "platform", "distributed systems"],
      excludeKeywords: ["intern", "frontend", "design"],
      enabled: true,
    },
  },
  {
    id: "realworkfromanywhere-backend",
    name: "Real Work From Anywhere Backend",
    summary: "真正全球 remote 的后端分类盘子，量不大，但地区语义更干净。",
    group: "international",
    tags: ["国外远程", "Backend", "Supplement"],
    rule: {
      name: "Real Work From Anywhere Backend",
      feedUrl: "https://www.realworkfromanywhere.com/remote-backend-jobs/rss.xml",
      source: "realworkfromanywhere:backend",
      defaultLocation: "Remote",
      includeKeywords: ["backend", "software engineer", "platform", "api", "distributed systems"],
      excludeKeywords: ["frontend", "sales", "marketing", "intern"],
      enabled: true,
    },
  },
  {
    id: "realworkfromanywhere-ai",
    name: "Real Work From Anywhere AI",
    summary: "AI/ML 分类更窄，适合拿来补真正偏模型与数据的远程岗。",
    group: "international",
    tags: ["国外远程", "AI", "Narrow"],
    rule: {
      name: "Real Work From Anywhere AI",
      feedUrl: "https://www.realworkfromanywhere.com/remote-ai-jobs/rss.xml",
      source: "realworkfromanywhere:ai",
      defaultLocation: "Remote",
      includeKeywords: ["ai", "machine learning", "ml", "llm", "agent", "python", "data"],
      excludeKeywords: ["intern", "sales", "marketing", "recruiter"],
      enabled: true,
    },
  },
  {
    id: "jobicy-dev-fulltime",
    name: "Jobicy Dev Full-Time",
    summary: "官方支持筛选参数的全职开发 feed，拿来补标准工程岗挺合适。",
    group: "international",
    tags: ["国外远程", "SWE", "Full-Time"],
    rule: {
      name: "Jobicy Dev Full-Time",
      feedUrl: "https://jobicy.com/feed/job_feed?job_categories=dev&job_types=full-time",
      source: "jobicy:dev:ft",
      defaultLocation: "Remote",
      includeKeywords: ["software engineer", "backend", "platform", "python", "golang", "api"],
      excludeKeywords: ["intern", "frontend", "sales", "marketing"],
      enabled: true,
    },
  },
  {
    id: "jobicy-data-science-fulltime",
    name: "Jobicy Data Science Full-Time",
    summary: "全职 Data / AI 方向补充源，适合补机器学习和数据工程岗位。",
    group: "international",
    tags: ["国外远程", "Data", "AI"],
    rule: {
      name: "Jobicy Data Science Full-Time",
      feedUrl: "https://jobicy.com/feed/job_feed?job_categories=data-science&job_types=full-time",
      source: "jobicy:data-science:ft",
      defaultLocation: "Remote",
      includeKeywords: ["ai", "machine learning", "ml", "data science", "data engineer", "llm", "python"],
      excludeKeywords: ["intern", "sales", "marketing"],
      enabled: true,
    },
  },
  {
    id: "remoteok-engineering",
    name: "Remote OK Engineering",
    summary: "老牌大盘子，覆盖广但噪音也大，关键词必须收紧。",
    group: "international",
    tags: ["国外远程", "General", "Wide"],
    rule: {
      name: "Remote OK Engineering",
      feedUrl: "https://remoteok.com/rss",
      source: "remoteok",
      defaultLocation: "Remote",
      includeKeywords: ["software engineer", "backend", "developer", "platform", "ai", "llm", "golang", "python"],
      excludeKeywords: ["intern", "sales", "marketing", "designer", "support"],
      enabled: true,
    },
  },
  {
    id: "smartremotejobs-devops",
    name: "SmartRemoteJobs DevOps",
    summary: "补 DevOps / SRE / Platform 这条线，适合基础设施方向。",
    group: "international",
    tags: ["国外远程", "DevOps", "Platform"],
    rule: {
      name: "SmartRemoteJobs DevOps",
      feedUrl: "https://www.smartremotejobs.com/feed/devops-remote-jobs.rss",
      source: "smartremotejobs:devops",
      defaultLocation: "Remote",
      includeKeywords: ["devops", "platform", "sre", "site reliability", "infrastructure", "kubernetes"],
      excludeKeywords: ["intern", "sales", "marketing"],
      enabled: true,
    },
  },
  {
    id: "smartremotejobs-data-science",
    name: "SmartRemoteJobs Data Science",
    summary: "Data / ML / Analytics 类远程岗位补充源，适合拉宽 AI 相关候选池。",
    group: "international",
    tags: ["国外远程", "Data", "AI"],
    rule: {
      name: "SmartRemoteJobs Data Science",
      feedUrl: "https://www.smartremotejobs.com/feed/data-science-remote-jobs.rss",
      source: "smartremotejobs:data-science",
      defaultLocation: "Remote",
      includeKeywords: ["ai", "machine learning", "ml", "data science", "data engineer", "python", "llm"],
      excludeKeywords: ["intern", "sales", "marketing"],
      enabled: true,
    },
  },
  {
    id: "v2ex-jobs",
    name: "V2EX 酷工作",
    summary: "国内技术社区里最像样的开发招聘源，适合先接入观察质量。",
    group: "domestic",
    tags: ["国内", "社区", "招聘"],
    rule: {
      name: "V2EX 酷工作",
      feedUrl: "https://www.v2ex.com/feed/jobs.xml",
      source: "v2ex:jobs",
      defaultLocation: "中国 / Remote",
      includeKeywords: ["agent", "ai", "llm", "backend", "后端", "software engineer", "golang", "python"],
      excludeKeywords: ["实习", "销售", "客服", "运营", "测试"],
      enabled: true,
    },
  },
  {
    id: "linuxdo-job-category",
    name: "LINUX DO 招聘分类",
    summary: "国内社区招聘分类 RSS，内容更杂，但确实是招聘板块，不是全站资讯。",
    group: "domestic",
    tags: ["国内", "社区", "招聘"],
    rule: {
      name: "LINUX DO 招聘分类",
      feedUrl: "https://linux.do/c/job/27.rss",
      source: "linuxdo:job",
      defaultLocation: "中国",
      includeKeywords: ["agent", "ai", "llm", "后端", "backend", "python", "golang"],
      excludeKeywords: ["客服", "销售", "运营", "实习"],
      enabled: true,
    },
  },
  {
    id: "linuxdo-job-tag",
    name: "LINUX DO 招聘标签",
    summary: "比整分类更窄一点的国内社区招聘源，可以作为补充观察。",
    group: "domestic",
    tags: ["国内", "社区", "招聘"],
    rule: {
      name: "LINUX DO 招聘标签",
      feedUrl: "https://linux.do/tag/2116-tag/2116.rss",
      source: "linuxdo:tag:job",
      defaultLocation: "中国",
      includeKeywords: ["agent", "ai", "llm", "后端", "backend", "python", "golang"],
      excludeKeywords: ["客服", "销售", "运营", "实习"],
      enabled: true,
    },
  },
  {
    id: "ruby-china-topics",
    name: "Ruby China Topics",
    summary: "全站主题 feed，噪音更大，但偶尔会有全栈/Agent 相关招聘帖。",
    group: "domestic",
    tags: ["国内", "社区", "杂讯"],
    rule: {
      name: "Ruby China Topics",
      feedUrl: "https://ruby-china.org/topics/feed",
      source: "rubychina:topics",
      defaultLocation: "中国",
      includeKeywords: ["招聘", "agent", "ai", "llm", "后端", "全栈", "golang", "python"],
      excludeKeywords: ["线下活动", "闲聊", "二手", "求助"],
      enabled: true,
    },
  },
];

function normalizeText(value: string): string {
  return value.trim().toLowerCase();
}

function normalizeFeedURL(value: string): string {
  return value.trim().toLowerCase();
}

export function hasMatchingRule(preset: DiscoveryPreset, rules: DiscoveryRule[]): boolean {
  const presetName = normalizeText(preset.rule.name);
  const presetFeedURL = normalizeFeedURL(preset.rule.feedUrl);

  return rules.some((rule) => {
    return normalizeText(rule.name) === presetName || normalizeFeedURL(rule.feedUrl) === presetFeedURL;
  });
}

export function getDiscoveryPresetsByGroup(group: DiscoveryPresetGroup): DiscoveryPreset[] {
  return DISCOVERY_PRESETS.filter((preset) => preset.group === group);
}
