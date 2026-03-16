"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { Message } from "./types";

const CHAT_STORAGE_KEY = "trace2offer-chat-sessions-v1";
const DEFAULT_SESSION_TITLE = "新会话";
const DEFAULT_GREETING =
  "你好！我是你的求职助手。我可以帮你管理求职线索、分析职位匹配度、优化投递策略。有什么可以帮助你的吗？";

export interface ChatSession {
  id: string;
  title: string;
  messages: Message[];
  updatedAt: string;
}

interface SessionSummaryInput {
  id: string;
  title?: string;
  updatedAt?: string;
}

interface CreateSessionInput {
  id: string;
  title?: string;
  messages?: Message[];
  updatedAt?: string;
  activate?: boolean;
}

interface ReplaceMessagesInput {
  id: string;
  messages: Message[];
  title?: string;
  updatedAt?: string;
  activate?: boolean;
}

interface ChatState {
  sessions: ChatSession[];
  activeSessionId: string;
  isLoading: boolean;
  hasHydrated: boolean;
  setLoading: (loading: boolean) => void;
  markHydrated: () => void;
  createSession: (input: CreateSessionInput) => void;
  mergeSessionSummaries: (summaries: SessionSummaryInput[]) => void;
  setActiveSessionId: (sessionId: string) => void;
  appendMessageToActiveSession: (message: Omit<Message, "id" | "createdAt">) => void;
  appendMessage: (sessionId: string, message: Omit<Message, "id" | "createdAt">, createdAt?: string) => void;
  replaceSessionMessages: (input: ReplaceMessagesInput) => void;
  renameSessionId: (fromSessionId: string, toSessionId: string) => void;
}

function nowISO(): string {
  return new Date().toISOString();
}

function randomID(prefix: string): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${prefix}${crypto.randomUUID()}`;
  }
  return `${prefix}${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

function parseTimestamp(value: string): number {
  if (!value) {
    return 0;
  }
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? 0 : ts;
}

function sortSessions(sessions: ChatSession[]): ChatSession[] {
  return [...sessions].sort((a, b) => {
    const diff = parseTimestamp(b.updatedAt) - parseTimestamp(a.updatedAt);
    if (diff !== 0) {
      return diff;
    }
    return a.id.localeCompare(b.id);
  });
}

function truncateText(value: string, maxRunes: number): string {
  const trimmed = value.trim();
  const runes = [...trimmed];
  if (runes.length <= maxRunes) {
    return trimmed;
  }
  if (maxRunes <= 1) {
    return runes.slice(0, maxRunes).join("");
  }
  return `${runes.slice(0, maxRunes - 1).join("")}…`;
}

function createGreetingMessage(createdAt = nowISO()): Message {
  return {
    id: randomID("msg_"),
    role: "assistant",
    content: DEFAULT_GREETING,
    createdAt,
  };
}

function normalizeMessages(messages: Message[] | undefined): Message[] {
  if (!messages || messages.length === 0) {
    return [];
  }

  return messages
    .map((item) => ({
      id: item.id?.trim() || randomID("msg_"),
      role: item.role === "assistant" ? ("assistant" as const) : ("user" as const),
      content: (item.content || "").trim(),
      createdAt: item.createdAt?.trim() || nowISO(),
    }))
    .filter((item) => item.content !== "");
}

function deriveTitle(messages: Message[], fallbackTitle?: string): string {
  for (const message of messages) {
    if (message.role !== "user") {
      continue;
    }
    const content = (message.content || "").trim();
    if (content === "") {
      continue;
    }
    return truncateText(content, 24);
  }

  const fallback = (fallbackTitle || "").trim();
  if (fallback !== "") {
    return truncateText(fallback, 24);
  }
  return DEFAULT_SESSION_TITLE;
}

function ensureActiveSessionId(sessions: ChatSession[], activeSessionId: string): string {
  const target = activeSessionId.trim();
  if (target !== "" && sessions.some((item) => item.id === target)) {
    return target;
  }
  return sessions[0]?.id || "";
}

function upsertSessionCollection(collection: ChatSession[], next: ChatSession): ChatSession[] {
  return sortSessions([...collection.filter((item) => item.id !== next.id), next]);
}

export const useChatStore = create<ChatState>()(
  persist(
    (set, get) => ({
      sessions: [],
      activeSessionId: "",
      isLoading: false,
      hasHydrated: false,

      setLoading: (loading) => set({ isLoading: loading }),

      markHydrated: () => set({ hasHydrated: true }),

      createSession: ({ id, title, messages, updatedAt, activate }) =>
        set((state) => {
          const sessionId = id.trim();
          if (sessionId === "") {
            return state;
          }

          const existing = state.sessions.find((item) => item.id === sessionId);
          const normalized = normalizeMessages(messages ?? existing?.messages);
          const nextMessages = normalized.length > 0 ? normalized : [createGreetingMessage(updatedAt || nowISO())];
          const nextUpdatedAt = (updatedAt || existing?.updatedAt || nowISO()).trim() || nowISO();

          const nextSession: ChatSession = {
            id: sessionId,
            title: deriveTitle(nextMessages, title || existing?.title),
            messages: nextMessages,
            updatedAt: nextUpdatedAt,
          };

          const sessions = upsertSessionCollection(state.sessions, nextSession);
          const shouldActivate =
            activate === true || state.activeSessionId.trim() === "" || state.activeSessionId === sessionId;

          return {
            sessions,
            activeSessionId: shouldActivate ? sessionId : ensureActiveSessionId(sessions, state.activeSessionId),
          };
        }),

      mergeSessionSummaries: (summaries) =>
        set((state) => {
          if (!Array.isArray(summaries) || summaries.length === 0) {
            return state;
          }

          let sessions = [...state.sessions];
          for (const item of summaries) {
            const sessionId = (item.id || "").trim();
            if (sessionId === "") {
              continue;
            }

            const existing = sessions.find((session) => session.id === sessionId);
            const nextMessages = normalizeMessages(existing?.messages);
            const safeMessages = nextMessages.length > 0 ? nextMessages : [createGreetingMessage(item.updatedAt || nowISO())];
            const nextSession: ChatSession = {
              id: sessionId,
              title: deriveTitle(safeMessages, item.title || existing?.title),
              messages: safeMessages,
              updatedAt: (item.updatedAt || existing?.updatedAt || nowISO()).trim() || nowISO(),
            };

            sessions = upsertSessionCollection(sessions, nextSession);
          }

          return {
            sessions,
            activeSessionId: ensureActiveSessionId(sessions, state.activeSessionId),
          };
        }),

      setActiveSessionId: (sessionId) =>
        set((state) => {
          const target = sessionId.trim();
          if (target === "" || !state.sessions.some((item) => item.id === target)) {
            return state;
          }
          return { activeSessionId: target };
        }),

      appendMessageToActiveSession: (message) => {
        const activeSessionId = get().activeSessionId;
        if (activeSessionId.trim() === "") {
          return;
        }
        get().appendMessage(activeSessionId, message);
      },

      appendMessage: (sessionId, message, createdAt) =>
        set((state) => {
          const targetId = sessionId.trim();
          if (targetId === "") {
            return state;
          }

          const existing = state.sessions.find((item) => item.id === targetId);
          if (!existing) {
            return state;
          }

          const content = (message.content || "").trim();
          if (content === "") {
            return state;
          }

          const safeCreatedAt = (createdAt || nowISO()).trim() || nowISO();
          const nextMessage: Message = {
            id: randomID("msg_"),
            role: message.role === "assistant" ? "assistant" : "user",
            content,
            createdAt: safeCreatedAt,
          };
          const messages = [...existing.messages, nextMessage];

          const nextSession: ChatSession = {
            ...existing,
            messages,
            updatedAt: safeCreatedAt,
            title: deriveTitle(messages, existing.title),
          };

          const sessions = upsertSessionCollection(state.sessions, nextSession);
          return {
            sessions,
            activeSessionId: ensureActiveSessionId(sessions, state.activeSessionId),
          };
        }),

      replaceSessionMessages: ({ id, messages, title, updatedAt, activate }) =>
        set((state) => {
          const sessionId = id.trim();
          if (sessionId === "") {
            return state;
          }

          const existing = state.sessions.find((item) => item.id === sessionId);
          const normalized = normalizeMessages(messages);
          const nextMessages = normalized.length > 0 ? normalized : [createGreetingMessage(updatedAt || nowISO())];
          const nextSession: ChatSession = {
            id: sessionId,
            messages: nextMessages,
            updatedAt: (updatedAt || existing?.updatedAt || nowISO()).trim() || nowISO(),
            title: deriveTitle(nextMessages, title || existing?.title),
          };

          const sessions = upsertSessionCollection(state.sessions, nextSession);
          const shouldActivate =
            activate === true || state.activeSessionId.trim() === "" || state.activeSessionId === sessionId;

          return {
            sessions,
            activeSessionId: shouldActivate ? sessionId : ensureActiveSessionId(sessions, state.activeSessionId),
          };
        }),

      renameSessionId: (fromSessionId, toSessionId) =>
        set((state) => {
          const fromID = fromSessionId.trim();
          const toID = toSessionId.trim();
          if (fromID === "" || toID === "" || fromID === toID) {
            return state;
          }

          const current = state.sessions.find((item) => item.id === fromID);
          if (!current) {
            return state;
          }

          const existingTarget = state.sessions.find((item) => item.id === toID);
          const mergedMessages = normalizeMessages(
            existingTarget ? [...existingTarget.messages, ...current.messages] : current.messages
          );
          const nextSession: ChatSession = {
            id: toID,
            messages: mergedMessages.length > 0 ? mergedMessages : [createGreetingMessage(current.updatedAt || nowISO())],
            updatedAt: (current.updatedAt || existingTarget?.updatedAt || nowISO()).trim() || nowISO(),
            title: deriveTitle(mergedMessages, existingTarget?.title || current.title),
          };

          const filtered = state.sessions.filter((item) => item.id !== fromID && item.id !== toID);
          const sessions = upsertSessionCollection(filtered, nextSession);
          const activeSessionId = state.activeSessionId === fromID ? toID : state.activeSessionId;

          return {
            sessions,
            activeSessionId: ensureActiveSessionId(sessions, activeSessionId),
          };
        }),
    }),
    {
      name: CHAT_STORAGE_KEY,
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        sessions: state.sessions,
        activeSessionId: state.activeSessionId,
      }),
      onRehydrateStorage: () => (state) => {
        state?.markHydrated();
      },
    }
  )
);
