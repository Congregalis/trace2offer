"use client";

import { create } from "zustand";
import {
  Candidate,
  CandidateMutationInput,
  CandidatePromoteInput,
  CandidateStatus,
  Lead,
  LeadStatus,
  ReminderMethod,
} from "./types";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");

interface APICandidate {
  id: string;
  company: string;
  position: string;
  source?: string;
  location?: string;
  jd_url?: string;
  company_website_url?: string;
  status?: string;
  match_score?: number;
  match_reasons?: string[];
  recommendation_notes?: string;
  notes?: string;
  promoted_lead_id?: string;
  created_at?: string;
  updated_at?: string;
}

interface APIPromotePayload {
  data?: {
    candidate?: APICandidate;
    lead?: APILead;
  };
}

interface APILead {
  id: string;
  company: string;
  position: string;
  source?: string;
  status?: string;
  priority?: number;
  next_action?: string;
  next_action_at?: string;
  interview_at?: string;
  reminder_methods?: string[];
  notes?: string;
  company_website_url?: string;
  jd_url?: string;
  jd_text?: string;
  location?: string;
  created_at?: string;
  updated_at?: string;
}

interface APIErrorPayload {
  message?: string;
  error?: string;
}

interface APIListPayload {
  data?: APICandidate[];
}

interface APISinglePayload {
  data?: APICandidate;
}

interface PromoteResult {
  candidate: Candidate;
  lead: Lead;
}

interface CandidatesState {
  candidates: Candidate[];
  isLoading: boolean;
  isSyncing: boolean;
  hasLoaded: boolean;
  error: string | null;
  fetchCandidates: () => Promise<void>;
  addCandidate: (candidate: CandidateMutationInput) => Promise<void>;
  updateCandidate: (id: string, updates: CandidateMutationInput) => Promise<void>;
  deleteCandidate: (id: string) => Promise<void>;
  promoteCandidate: (id: string, input: Partial<CandidatePromoteInput>) => Promise<PromoteResult>;
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

const statusSet = new Set<CandidateStatus>(["pending_review", "shortlisted", "dismissed", "promoted"]);
const leadStatusSet = new Set<LeadStatus>([
  "new",
  "preparing",
  "applied",
  "interviewing",
  "offered",
  "declined",
  "rejected",
  "archived",
]);
const reminderMethodSet = new Set<ReminderMethod>(["in_app", "email", "web_push"]);

function normalizeStatus(raw: string | undefined): CandidateStatus {
  const status = (raw || "").trim() as CandidateStatus;
  if (statusSet.has(status)) {
    return status;
  }
  return "pending_review";
}

function normalizeLeadStatus(raw: string | undefined): LeadStatus {
  const status = (raw || "").trim() as LeadStatus;
  if (leadStatusSet.has(status)) {
    return status;
  }
  return "new";
}

function parseScore(raw: number | undefined): number {
  if (typeof raw !== "number" || Number.isNaN(raw)) {
    return 0;
  }
  if (raw < 0) {
    return 0;
  }
  if (raw > 100) {
    return 100;
  }
  return Math.floor(raw);
}

function normalizeStringList(raw: string[] | undefined): string[] {
  if (!Array.isArray(raw) || raw.length === 0) {
    return [];
  }

  const normalized: string[] = [];
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
    normalized.push(value);
  }
  return normalized;
}

function normalizeReminderMethods(raw: string[] | undefined): ReminderMethod[] {
  if (!Array.isArray(raw) || raw.length === 0) {
    return ["in_app"];
  }

  const normalized: ReminderMethod[] = [];
  const seen = new Set<string>();
  for (const item of raw) {
    const method = (item || "").trim() as ReminderMethod;
    if (!reminderMethodSet.has(method) || seen.has(method)) {
      continue;
    }
    seen.add(method);
    normalized.push(method);
  }
  return normalized.length > 0 ? normalized : ["in_app"];
}

function normalizeDateTime(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }
  const ts = Date.parse(trimmed);
  if (Number.isNaN(ts)) {
    return "";
  }
  return new Date(ts).toISOString();
}

function parseTimestamp(value: string): number {
  if (!value) {
    return 0;
  }
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? 0 : ts;
}

function sortByUpdatedAtDesc(candidates: Candidate[]): Candidate[] {
  return [...candidates].sort((a, b) => parseTimestamp(b.updatedAt) - parseTimestamp(a.updatedAt));
}

function normalizeCandidate(raw: APICandidate): Candidate {
  return {
    id: raw.id,
    company: (raw.company || "").trim(),
    position: (raw.position || "").trim(),
    source: (raw.source || "").trim(),
    location: (raw.location || "").trim(),
    jdUrl: (raw.jd_url || "").trim(),
    companyWebsiteUrl: (raw.company_website_url || "").trim(),
    status: normalizeStatus(raw.status),
    matchScore: parseScore(raw.match_score),
    matchReasons: normalizeStringList(raw.match_reasons),
    recommendationNotes: (raw.recommendation_notes || "").trim(),
    notes: (raw.notes || "").trim(),
    promotedLeadId: (raw.promoted_lead_id || "").trim(),
    createdAt: (raw.created_at || "").trim(),
    updatedAt: (raw.updated_at || "").trim(),
  };
}

function normalizeLead(raw: APILead): Lead {
  return {
    id: raw.id,
    company: (raw.company || "").trim(),
    position: (raw.position || "").trim(),
    source: (raw.source || "").trim(),
    status: normalizeLeadStatus(raw.status),
    priority: typeof raw.priority === "number" && raw.priority > 0 ? Math.floor(raw.priority) : 0,
    nextAction: (raw.next_action || "").trim(),
    nextActionAt: normalizeDateTime(raw.next_action_at || ""),
    interviewAt: normalizeDateTime(raw.interview_at || ""),
    reminderMethods: normalizeReminderMethods(raw.reminder_methods),
    notes: (raw.notes || "").trim(),
    companyWebsiteUrl: (raw.company_website_url || "").trim(),
    jdUrl: (raw.jd_url || "").trim(),
    jdText: (raw.jd_text || "").trim(),
    location: (raw.location || "").trim(),
    createdAt: (raw.created_at || "").trim(),
    updatedAt: (raw.updated_at || "").trim(),
  };
}

function toMutationPayload(input: CandidateMutationInput): Record<string, unknown> {
  return {
    company: input.company.trim(),
    position: input.position.trim(),
    source: input.source.trim(),
    location: input.location.trim(),
    jd_url: input.jdUrl.trim(),
    company_website_url: input.companyWebsiteUrl.trim(),
    status: input.status,
    match_score: parseScore(input.matchScore),
    match_reasons: normalizeStringList(input.matchReasons),
    recommendation_notes: input.recommendationNotes.trim(),
    notes: input.notes.trim(),
  };
}

function toPromotePayload(input: Partial<CandidatePromoteInput>): Record<string, unknown> {
  const status = (input.status || "new") as LeadStatus;
  const safeStatus = leadStatusSet.has(status) ? status : "new";
  return {
    source: (input.source || "").trim(),
    status: safeStatus,
    priority: typeof input.priority === "number" ? Math.max(0, Math.floor(input.priority)) : 0,
    next_action: (input.nextAction || "").trim(),
    next_action_at: normalizeDateTime(input.nextActionAt || ""),
    interview_at: normalizeDateTime(input.interviewAt || ""),
    reminder_methods: normalizeReminderMethods(input.reminderMethods),
    notes: (input.notes || "").trim(),
  };
}

export const useCandidatesStore = create<CandidatesState>((set, get) => ({
  candidates: [],
  isLoading: false,
  isSyncing: false,
  hasLoaded: false,
  error: null,

  fetchCandidates: async () => {
    if (get().isLoading) {
      return;
    }
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(getAPIURL("/api/candidates"), {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) {
        throw await parseAPIError(response, "加载候选职位失败");
      }
      const payload = (await response.json()) as APIListPayload;
      const rawList = Array.isArray(payload.data) ? payload.data : [];
      const candidates = sortByUpdatedAtDesc(rawList.map(normalizeCandidate));
      set({ candidates, hasLoaded: true, error: null });
    } catch (error) {
      const message = toErrorMessage(error, "加载候选职位失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isLoading: false });
    }
  },

  addCandidate: async (candidateInput) => {
    if (get().isSyncing) {
      return;
    }
    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL("/api/candidates"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(toMutationPayload(candidateInput)),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "创建候选职位失败");
      }
      const payload = (await response.json()) as APISinglePayload;
      if (!payload.data) {
        throw new Error("创建候选职位响应缺少 data");
      }
      const created = normalizeCandidate(payload.data);
      set((state) => ({
        candidates: sortByUpdatedAtDesc([...state.candidates.filter((item) => item.id !== created.id), created]),
        error: null,
      }));
    } catch (error) {
      const message = toErrorMessage(error, "创建候选职位失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },

  updateCandidate: async (id, updates) => {
    if (get().isSyncing) {
      return;
    }
    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL(`/api/candidates/${id}`), {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(toMutationPayload(updates)),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "更新候选职位失败");
      }
      const payload = (await response.json()) as APISinglePayload;
      if (!payload.data) {
        throw new Error("更新候选职位响应缺少 data");
      }
      const updated = normalizeCandidate(payload.data);
      set((state) => ({
        candidates: sortByUpdatedAtDesc([...state.candidates.filter((item) => item.id !== updated.id), updated]),
        error: null,
      }));
    } catch (error) {
      const message = toErrorMessage(error, "更新候选职位失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },

  deleteCandidate: async (id) => {
    if (get().isSyncing) {
      return;
    }
    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL(`/api/candidates/${id}`), {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok && response.status !== 204) {
        throw await parseAPIError(response, "删除候选职位失败");
      }
      set((state) => ({
        candidates: state.candidates.filter((item) => item.id !== id),
        error: null,
      }));
    } catch (error) {
      const message = toErrorMessage(error, "删除候选职位失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },

  promoteCandidate: async (id, input) => {
    if (get().isSyncing) {
      throw new Error("正在同步数据，请稍后重试");
    }
    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL(`/api/candidates/${id}/promote`), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(toPromotePayload(input)),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "转为线索失败");
      }
      const payload = (await response.json()) as APIPromotePayload;
      const rawCandidate = payload.data?.candidate;
      const rawLead = payload.data?.lead;
      if (!rawCandidate || !rawLead) {
        throw new Error("转为线索响应缺少 candidate 或 lead");
      }
      const updatedCandidate = normalizeCandidate(rawCandidate);
      const createdLead = normalizeLead(rawLead);
      set((state) => ({
        candidates: sortByUpdatedAtDesc([
          ...state.candidates.filter((item) => item.id !== updatedCandidate.id),
          updatedCandidate,
        ]),
        error: null,
      }));
      return {
        candidate: updatedCandidate,
        lead: createdLead,
      };
    } catch (error) {
      const message = toErrorMessage(error, "转为线索失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },
}));
