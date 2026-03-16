"use client";

import { create } from "zustand";
import { Lead, LeadMutationInput, LeadStatus } from "./types";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");

interface APILead {
  id: string;
  company: string;
  position: string;
  source?: string;
  status?: string;
  priority?: number;
  next_action?: string;
  notes?: string;
  company_website_url?: string;
  jd_url?: string;
  location?: string;
  created_at?: string;
  updated_at?: string;
}

interface APIErrorPayload {
  message?: string;
  error?: string;
}

interface APIListPayload {
  data?: APILead[];
}

interface APISinglePayload {
  data?: APILead;
}

interface LeadsState {
  leads: Lead[];
  isLoading: boolean;
  isSyncing: boolean;
  hasLoaded: boolean;
  error: string | null;
  fetchLeads: () => Promise<void>;
  addLead: (lead: LeadMutationInput) => Promise<void>;
  updateLead: (id: string, updates: Partial<LeadMutationInput>) => Promise<void>;
  deleteLead: (id: string) => Promise<void>;
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

const statusSet = new Set<LeadStatus>([
  "new",
  "preparing",
  "applied",
  "interviewing",
  "offered",
  "declined",
  "rejected",
  "archived",
]);

function normalizeStatus(raw: string | undefined): LeadStatus {
  const canonical = (raw || "").trim() as LeadStatus;
  if (statusSet.has(canonical)) {
    return canonical;
  }
  return "new";
}

function parsePriority(raw: number | undefined): number {
  if (typeof raw !== "number" || Number.isNaN(raw)) {
    return 0;
  }
  return Math.max(0, Math.floor(raw));
}

function normalizeLead(apiLead: APILead): Lead {
  return {
    id: apiLead.id,
    company: (apiLead.company || "").trim(),
    position: (apiLead.position || "").trim(),
    source: (apiLead.source || "").trim(),
    status: normalizeStatus(apiLead.status),
    priority: parsePriority(apiLead.priority),
    nextAction: (apiLead.next_action || "").trim(),
    notes: (apiLead.notes || "").trim(),
    companyWebsiteUrl: (apiLead.company_website_url || "").trim(),
    jdUrl: (apiLead.jd_url || "").trim(),
    location: (apiLead.location || "").trim(),
    createdAt: (apiLead.created_at || "").trim(),
    updatedAt: (apiLead.updated_at || "").trim(),
  };
}

function toMutationPayload(input: LeadMutationInput): Record<string, unknown> {
  return {
    company: input.company.trim(),
    position: input.position.trim(),
    source: input.source.trim(),
    status: input.status,
    priority: parsePriority(input.priority),
    next_action: input.nextAction.trim(),
    notes: input.notes.trim(),
    company_website_url: input.companyWebsiteUrl.trim(),
    jd_url: input.jdUrl.trim(),
    location: input.location.trim(),
  };
}

function toMutationInput(lead: Lead): LeadMutationInput {
  return {
    company: lead.company,
    position: lead.position,
    source: lead.source,
    status: lead.status,
    priority: lead.priority,
    nextAction: lead.nextAction,
    notes: lead.notes,
    companyWebsiteUrl: lead.companyWebsiteUrl,
    jdUrl: lead.jdUrl,
    location: lead.location,
  };
}

function parseTimestamp(value: string): number {
  if (!value) {
    return 0;
  }
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? 0 : ts;
}

function sortByUpdatedAtDesc(leads: Lead[]): Lead[] {
  return [...leads].sort((a, b) => parseTimestamp(b.updatedAt) - parseTimestamp(a.updatedAt));
}

export const useLeadsStore = create<LeadsState>((set, get) => ({
  leads: [],
  isLoading: false,
  isSyncing: false,
  hasLoaded: false,
  error: null,

  fetchLeads: async () => {
    if (get().isLoading) {
      return;
    }

    set({ isLoading: true, error: null });
    try {
      const response = await fetch(getAPIURL("/api/leads"), {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) {
        throw await parseAPIError(response, "加载线索失败");
      }

      const payload = (await response.json()) as APIListPayload;
      const rawList = Array.isArray(payload.data) ? payload.data : [];
      const leads = sortByUpdatedAtDesc(rawList.map(normalizeLead));
      set({ leads, hasLoaded: true, error: null });
    } catch (error) {
      const message = toErrorMessage(error, "加载线索失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isLoading: false });
    }
  },

  addLead: async (lead) => {
    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL("/api/leads"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(toMutationPayload(lead)),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "创建线索失败");
      }

      const payload = (await response.json()) as APISinglePayload;
      if (!payload.data) {
        throw new Error("创建线索失败: 响应缺少 data");
      }
      const created = normalizeLead(payload.data);
      set((state) => ({
        leads: sortByUpdatedAtDesc([created, ...state.leads.filter((item) => item.id !== created.id)]),
      }));
    } catch (error) {
      const message = toErrorMessage(error, "创建线索失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },

  updateLead: async (id, updates) => {
    const currentLead = get().leads.find((lead) => lead.id === id);
    if (!currentLead) {
      throw new Error("线索不存在，无法更新");
    }

    const merged: LeadMutationInput = {
      ...toMutationInput(currentLead),
      ...updates,
      status: normalizeStatus(updates.status ?? currentLead.status),
      priority: parsePriority(updates.priority ?? currentLead.priority),
    };

    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL(`/api/leads/${id}`), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(toMutationPayload(merged)),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "更新线索失败");
      }

      const payload = (await response.json()) as APISinglePayload;
      if (!payload.data) {
        throw new Error("更新线索失败: 响应缺少 data");
      }
      const updated = normalizeLead(payload.data);
      set((state) => ({
        leads: sortByUpdatedAtDesc(state.leads.map((item) => (item.id === id ? updated : item))),
      }));
    } catch (error) {
      const message = toErrorMessage(error, "更新线索失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },

  deleteLead: async (id) => {
    set({ isSyncing: true, error: null });
    try {
      const response = await fetch(getAPIURL(`/api/leads/${id}`), {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok && response.status !== 404) {
        throw await parseAPIError(response, "删除线索失败");
      }
      if (response.status === 404) {
        throw new Error("线索不存在或已被删除");
      }

      set((state) => ({
        leads: state.leads.filter((item) => item.id !== id),
      }));
    } catch (error) {
      const message = toErrorMessage(error, "删除线索失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isSyncing: false });
    }
  },
}));
