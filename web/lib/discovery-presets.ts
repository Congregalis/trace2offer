import type { DiscoveryRule, DiscoveryRuleMutationInput } from "./types";

export type DiscoveryPresetGroup = "priority" | "general";

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
    group: "priority",
    tags: ["Remote", "AI", "Agent"],
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
    group: "priority",
    tags: ["Remote", "SWE", "General"],
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
    group: "priority",
    tags: ["Remote", "Backend", "Platform"],
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
    group: "general",
    tags: ["Remote", "Backend"],
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
    group: "general",
    tags: ["Remote", "SWE", "Supplement"],
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
