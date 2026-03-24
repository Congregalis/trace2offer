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
