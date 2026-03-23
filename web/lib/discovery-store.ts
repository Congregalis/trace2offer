"use client";

import { create } from "zustand";
import { DiscoveryRule, DiscoveryRuleMutationInput, DiscoveryRunResult } from "./types";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");

interface APIDiscoveryRule {
  id: string;
  name?: string;
  feed_url?: string;
  source?: string;
  default_location?: string;
  include_keywords?: string[];
  exclude_keywords?: string[];
  enabled?: boolean;
  created_at?: string;
  updated_at?: string;
}

interface APIDiscoveryRunResult {
  ran_at?: string;
  rules_total?: number;
  rules_executed?: number;
  entries_fetched?: number;
  candidates_created?: number;
  candidates_updated?: number;
  errors?: string[];
}

interface APIListPayload {
  data?: APIDiscoveryRule[];
}

interface APISinglePayload {
  data?: APIDiscoveryRule;
}

interface APIRunPayload {
  data?: APIDiscoveryRunResult;
}

interface APIErrorPayload {
  message?: string;
  error?: string;
}

interface DiscoveryState {
  rules: DiscoveryRule[];
  lastRun: DiscoveryRunResult | null;
  isLoading: boolean;
  isSyncing: boolean;
  isRunning: boolean;
  hasLoaded: boolean;
  error: string | null;
  fetchRules: () => Promise<void>;
  addRule: (input: DiscoveryRuleMutationInput) => Promise<void>;
  updateRule: (id: string, input: DiscoveryRuleMutationInput) => Promise<void>;
  deleteRule: (id: string) => Promise<void>;
  runDiscoveryNow: () => Promise<DiscoveryRunResult>;
}

function getAPIURL(path: string): string {
  return `${API_BASE_URL}${path}`;
}

function toErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return fallback;
}

async function parseAPIError(response: Response, fallback: string): Promise<Error> {
  try {
    const payload = (await response.json()) as APIErrorPayload;
    const parts = [payload.message, payload.error].filter(Boolean);
    if (parts.length > 0) {
      return new Error(parts.join(": "));
    }
  } catch {
    // ignore parse errors from non-standard payloads.
  }
  return new Error(`${fallback} (HTTP ${response.status})`);
}

function normalizeStringList(raw: string[] | undefined): string[] {
  if (!Array.isArray(raw) || raw.length === 0) {
    return [];
  }
  const result: string[] = [];
  const seen = new Set<string>();
  for (const item of raw) {
    const value = (item || "").trim();
    if (!value) {
      continue;
    }
    const key = value.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    result.push(value);
  }
  return result;
}

function parseTimestamp(value: string): number {
  if (!value) {
    return 0;
  }
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? 0 : ts;
}

function sortRulesByUpdatedAtDesc(rules: DiscoveryRule[]): DiscoveryRule[] {
  return [...rules].sort((a, b) => parseTimestamp(b.updatedAt) - parseTimestamp(a.updatedAt));
}

function normalizeRule(raw: APIDiscoveryRule): DiscoveryRule {
  return {
    id: raw.id,
    name: (raw.name || "").trim(),
    feedUrl: (raw.feed_url || "").trim(),
    source: (raw.source || "").trim(),
    defaultLocation: (raw.default_location || "").trim(),
    includeKeywords: normalizeStringList(raw.include_keywords),
    excludeKeywords: normalizeStringList(raw.exclude_keywords),
    enabled: Boolean(raw.enabled),
    createdAt: (raw.created_at || "").trim(),
    updatedAt: (raw.updated_at || "").trim(),
  };
}

function normalizeRunResult(raw: APIDiscoveryRunResult | undefined): DiscoveryRunResult {
  return {
    ranAt: (raw?.ran_at || "").trim(),
    rulesTotal: typeof raw?.rules_total === "number" ? raw.rules_total : 0,
    rulesExecuted: typeof raw?.rules_executed === "number" ? raw.rules_executed : 0,
    entriesFetched: typeof raw?.entries_fetched === "number" ? raw.entries_fetched : 0,
    candidatesCreated: typeof raw?.candidates_created === "number" ? raw.candidates_created : 0,
    candidatesUpdated: typeof raw?.candidates_updated === "number" ? raw.candidates_updated : 0,
    errors: normalizeStringList(raw?.errors),
  };
}

function toRulePayload(input: DiscoveryRuleMutationInput): Record<string, unknown> {
  return {
    name: input.name.trim(),
    feed_url: input.feedUrl.trim(),
    source: input.source.trim(),
    default_location: input.defaultLocation.trim(),
    include_keywords: normalizeStringList(input.includeKeywords),
    exclude_keywords: normalizeStringList(input.excludeKeywords),
    enabled: input.enabled,
  };
}

export const useDiscoveryStore = create<DiscoveryState>((set, get) => ({
  rules: [],
  lastRun: null,
  isLoading: false,
  isSyncing: false,
  isRunning: false,
  hasLoaded: false,
  error: null,

  fetchRules: async () => {
    if (get().isLoading) {
      return;
    }
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(getAPIURL("/api/discovery/rules"), {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) {
        throw await parseAPIError(response, "加载发现规则失败");
      }
      const payload = (await response.json()) as APIListPayload;
      const rawList = Array.isArray(payload.data) ? payload.data : [];
      set({
        rules: sortRulesByUpdatedAtDesc(rawList.map(normalizeRule)),
        hasLoaded: true,
        error: null,
      });
    } catch (error) {
      const message = toErrorMessage(error, "加载发现规则失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isLoading: false });
    }
  },

  addRule: async (input) => {
    if (get().isSyncing) {
      return;
    }
    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL("/api/discovery/rules"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(toRulePayload(input)),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "创建发现规则失败");
      }
      const payload = (await response.json()) as APISinglePayload;
      if (!payload.data) {
        throw new Error("创建发现规则响应缺少 data");
      }
      const created = normalizeRule(payload.data);
      set((state) => ({
        rules: sortRulesByUpdatedAtDesc([...state.rules.filter((item) => item.id !== created.id), created]),
        error: null,
      }));
    } catch (error) {
      const message = toErrorMessage(error, "创建发现规则失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },

  updateRule: async (id, input) => {
    if (get().isSyncing) {
      return;
    }
    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL(`/api/discovery/rules/${id}`), {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(toRulePayload(input)),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "更新发现规则失败");
      }
      const payload = (await response.json()) as APISinglePayload;
      if (!payload.data) {
        throw new Error("更新发现规则响应缺少 data");
      }
      const updated = normalizeRule(payload.data);
      set((state) => ({
        rules: sortRulesByUpdatedAtDesc([...state.rules.filter((item) => item.id !== updated.id), updated]),
        error: null,
      }));
    } catch (error) {
      const message = toErrorMessage(error, "更新发现规则失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },

  deleteRule: async (id) => {
    if (get().isSyncing) {
      return;
    }
    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL(`/api/discovery/rules/${id}`), {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok && response.status !== 204) {
        throw await parseAPIError(response, "删除发现规则失败");
      }
      set((state) => ({
        rules: state.rules.filter((item) => item.id !== id),
        error: null,
      }));
    } catch (error) {
      const message = toErrorMessage(error, "删除发现规则失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },

  runDiscoveryNow: async () => {
    if (get().isRunning) {
      throw new Error("发现任务正在运行，请稍后再试");
    }
    set({ isRunning: true, error: null });
    try {
      const response = await fetch(getAPIURL("/api/discovery/run"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) {
        throw await parseAPIError(response, "执行发现任务失败");
      }
      const payload = (await response.json()) as APIRunPayload;
      const result = normalizeRunResult(payload.data);
      set({ lastRun: result, error: null });
      return result;
    } catch (error) {
      const message = toErrorMessage(error, "执行发现任务失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isRunning: false });
    }
  },
}));
