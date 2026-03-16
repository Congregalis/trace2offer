"use client";

import { create } from "zustand";
import { LeadTimeline, LeadTimelineStage } from "./types";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");

interface APITimelineStage {
  stage?: string;
  started_at?: string;
  ended_at?: string;
}

interface APITimeline {
  lead_id: string;
  stages?: APITimelineStage[];
  updated_at?: string;
}

interface APIListPayload {
  data?: APITimeline[];
}

interface APIErrorPayload {
  message?: string;
  error?: string;
}

interface LeadTimelineState {
  timelines: LeadTimeline[];
  timelineMap: Record<string, LeadTimeline>;
  isLoading: boolean;
  hasLoaded: boolean;
  error: string | null;
  fetchTimelines: () => Promise<void>;
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

function normalizeDateTime(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }

  const timestamp = Date.parse(trimmed);
  if (Number.isNaN(timestamp)) {
    return "";
  }
  return new Date(timestamp).toISOString();
}

function normalizeStage(apiStage: APITimelineStage): LeadTimelineStage | null {
  const stage = (apiStage.stage || "").trim();
  const startedAt = normalizeDateTime(apiStage.started_at || "");
  if (!stage || !startedAt) {
    return null;
  }

  return {
    stage,
    startedAt,
    endedAt: normalizeDateTime(apiStage.ended_at || ""),
  };
}

function normalizeTimeline(apiTimeline: APITimeline): LeadTimeline {
  const stages = Array.isArray(apiTimeline.stages)
    ? apiTimeline.stages
        .map(normalizeStage)
        .filter((item): item is LeadTimelineStage => item !== null)
        .sort((a, b) => Date.parse(a.startedAt) - Date.parse(b.startedAt))
    : [];

  return {
    leadId: (apiTimeline.lead_id || "").trim(),
    stages,
    updatedAt: normalizeDateTime(apiTimeline.updated_at || ""),
  };
}

function parseTimestamp(value: string): number {
  if (!value) {
    return 0;
  }
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? 0 : ts;
}

function sortByUpdatedAtDesc(items: LeadTimeline[]): LeadTimeline[] {
  return [...items].sort((a, b) => parseTimestamp(b.updatedAt) - parseTimestamp(a.updatedAt));
}

function buildTimelineMap(items: LeadTimeline[]): Record<string, LeadTimeline> {
  const map: Record<string, LeadTimeline> = {};
  for (const item of items) {
    if (!item.leadId) {
      continue;
    }
    map[item.leadId] = item;
  }
  return map;
}

export const useLeadTimelineStore = create<LeadTimelineState>((set, get) => ({
  timelines: [],
  timelineMap: {},
  isLoading: false,
  hasLoaded: false,
  error: null,

  fetchTimelines: async () => {
    if (get().isLoading) {
      return;
    }

    set({ isLoading: true, error: null });
    try {
      const response = await fetch(getAPIURL("/api/lead-timelines"), {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) {
        throw await parseAPIError(response, "加载时间线失败");
      }

      const payload = (await response.json()) as APIListPayload;
      const rawList = Array.isArray(payload.data) ? payload.data : [];
      const timelines = sortByUpdatedAtDesc(rawList.map(normalizeTimeline).filter((item) => item.leadId));
      set({ timelines, timelineMap: buildTimelineMap(timelines), hasLoaded: true, error: null });
    } catch (error) {
      const message = toErrorMessage(error, "加载时间线失败");
      set({ error: message });
      throw new Error(message);
    } finally {
      set({ isLoading: false });
    }
  },
}));
