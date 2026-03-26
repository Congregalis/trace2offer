"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useChatStore } from "@/lib/chat-store";
import { useLeadsStore } from "@/lib/leads-store";
import { Message } from "@/lib/types";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { Send, Bot, User, Sparkles, Settings2, ClipboardList, Plus } from "lucide-react";
import { toast } from "sonner";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
const DRAFT_SESSION_PREFIX = "draft_";
const LOCAL_SESSION_PREFIX = "local_";

interface AgentSettingsView {
  model: string;
  max_steps: number;
  system_prompt: string;
  openai_base_url: string;
  openai_timeout_seconds: number;
  has_openai_api_key: boolean;
}

interface AgentSettingsForm {
  model: string;
  maxSteps: number;
  systemPrompt: string;
  openaiBaseURL: string;
  openaiTimeoutSeconds: number;
  openaiAPIKey: string;
  hasOpenAIAPIKey: boolean;
}

interface UserProfileView {
  name: string;
  current_title: string;
  total_years: number;
  core_skills: string[];
  programming_languages: string[];
  project_evidence: string[];
  preferred_roles: string[];
  preferred_locations: string[];
  job_search_priorities: string[];
  strength_summary: string;
  updated_at?: string;
}

interface UserProfileForm {
  name: string;
  currentTitle: string;
  totalYears: string;
  coreSkills: string;
  programmingLanguages: string;
  projectEvidence: string;
  preferredRoles: string;
  preferredLocations: string;
  jobSearchPriorities: string;
  strengthSummary: string;
}

interface UserProfileImportView {
  profile: UserProfileView;
  extracted: UserProfileView;
  source_name: string;
  content_type: string;
  text_length: number;
  truncated: boolean;
  extract_model: string;
  resume_path?: string;
  resume_total_chars?: number;
  resume_truncated?: boolean;
}

interface APIErrorPayload {
  message?: string;
  error?: string;
}

interface AgentSessionMessageView {
  role: string;
  content: string;
  created_at: string;
}

interface AgentSessionView {
  id: string;
  messages: AgentSessionMessageView[];
  updated_at: string;
}

interface AgentSessionSummaryView {
  id: string;
  title: string;
  preview: string;
  message_count: number;
  updated_at: string;
}

const emptyUserProfileForm: UserProfileForm = {
  name: "",
  currentTitle: "",
  totalYears: "",
  coreSkills: "",
  programmingLanguages: "",
  projectEvidence: "",
  preferredRoles: "",
  preferredLocations: "",
  jobSearchPriorities: "",
  strengthSummary: "",
};

function getAPIURL(path: string): string {
  return `${API_BASE_URL}${path}`;
}

function createDraftSessionID(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${DRAFT_SESSION_PREFIX}${crypto.randomUUID()}`;
  }
  return `${DRAFT_SESSION_PREFIX}${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

function isTransientSessionID(sessionID: string): boolean {
  const safeID = sessionID.trim();
  return safeID.startsWith(DRAFT_SESSION_PREFIX) || safeID.startsWith(LOCAL_SESSION_PREFIX);
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
    // ignore JSON parsing errors for non-standard responses.
  }

  return new Error(`${fallback} (HTTP ${response.status})`);
}

function listToMultiline(values: string[] | undefined): string {
  if (!values || values.length === 0) {
    return "";
  }
  return values.join("\n");
}

function multilineToList(value: string): string[] {
  return value
    .split(/\r?\n|,|，|;|；/g)
    .map((item) => item.trim())
    .filter(Boolean);
}

function toProfileForm(profile: UserProfileView): UserProfileForm {
  return {
    name: profile.name || "",
    currentTitle: profile.current_title || "",
    totalYears: profile.total_years > 0 ? String(profile.total_years) : "",
    coreSkills: listToMultiline(profile.core_skills),
    programmingLanguages: listToMultiline(profile.programming_languages),
    projectEvidence: listToMultiline(profile.project_evidence),
    preferredRoles: listToMultiline(profile.preferred_roles),
    preferredLocations: listToMultiline(profile.preferred_locations),
    jobSearchPriorities: listToMultiline(profile.job_search_priorities),
    strengthSummary: profile.strength_summary || "",
  };
}

function toProfilePayload(form: UserProfileForm): UserProfileView {
  const parsedYears = Number(form.totalYears.trim() || "0");
  return {
    name: form.name.trim(),
    current_title: form.currentTitle.trim(),
    total_years: Number.isFinite(parsedYears) && parsedYears > 0 ? parsedYears : 0,
    core_skills: multilineToList(form.coreSkills),
    programming_languages: multilineToList(form.programmingLanguages),
    project_evidence: multilineToList(form.projectEvidence),
    preferred_roles: multilineToList(form.preferredRoles),
    preferred_locations: multilineToList(form.preferredLocations),
    job_search_priorities: multilineToList(form.jobSearchPriorities),
    strength_summary: form.strengthSummary.trim(),
    updated_at: "",
  };
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

function toClientMessages(messages: AgentSessionMessageView[] | undefined): Message[] {
  if (!Array.isArray(messages) || messages.length === 0) {
    return [];
  }

  return messages
    .filter((item) => item.role === "user" || item.role === "assistant")
    .map((item) => ({
      id: crypto.randomUUID(),
      role: item.role === "assistant" ? ("assistant" as const) : ("user" as const),
      content: (item.content || "").trim(),
      createdAt: (item.created_at || "").trim() || new Date().toISOString(),
    }))
    .filter((item) => item.content !== "");
}

export function Chat() {
  const {
    sessions,
    activeSessionId,
    isLoading,
    hasHydrated,
    setLoading,
    setActiveSessionId,
    createSession,
    mergeSessionSummaries,
    appendMessageToActiveSession,
    replaceSessionMessages,
    renameSessionId,
  } = useChatStore();
  const { leads, hasLoaded, fetchLeads } = useLeadsStore();

  const [input, setInput] = useState("");
  const [isSessionBootstrapping, setIsSessionBootstrapping] = useState(false);
  const [isSessionSwitching, setIsSessionSwitching] = useState(false);
  const [isSessionCreating, setIsSessionCreating] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isSettingsLoading, setIsSettingsLoading] = useState(false);
  const [isSettingsSaving, setIsSettingsSaving] = useState(false);
  const [isProfileOpen, setIsProfileOpen] = useState(false);
  const [isProfileLoading, setIsProfileLoading] = useState(false);
  const [isProfileSaving, setIsProfileSaving] = useState(false);
  const [isProfileImporting, setIsProfileImporting] = useState(false);
  const [resumeFile, setResumeFile] = useState<File | null>(null);
  const [profileUpdatedAt, setProfileUpdatedAt] = useState("");
  const [settings, setSettings] = useState<AgentSettingsForm>({
    model: "gpt-5-mini",
    maxSteps: 6,
    systemPrompt: "",
    openaiBaseURL: "https://api.openai.com/v1/responses",
    openaiTimeoutSeconds: 60,
    openaiAPIKey: "",
    hasOpenAIAPIKey: false,
  });
  const [profile, setProfile] = useState<UserProfileForm>(emptyUserProfileForm);

  const activeSession = useMemo(
    () => sessions.find((item) => item.id === activeSessionId) || null,
    [activeSessionId, sessions]
  );
  const messages = activeSession?.messages || [];
  const sessionId = activeSession?.id || "";

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  useEffect(() => {
    if (hasLoaded) {
      return;
    }

    void fetchLeads().catch(() => {
      // keep chat usable even if leads loading fails.
    });
  }, [fetchLeads, hasLoaded]);

  const syncSessionFromServer = useCallback(async (targetSessionId: string, activate = false) => {
    const safeSessionID = targetSessionId.trim();
    if (!safeSessionID) {
      return;
    }

    const response = await fetch(getAPIURL(`/api/agent/sessions/${encodeURIComponent(safeSessionID)}`), {
      method: "GET",
      headers: { "Content-Type": "application/json" },
    });
    if (!response.ok) {
      throw await parseAPIError(response, "加载会话失败");
    }

    const payload = (await response.json()) as { data?: AgentSessionView };
    if (!payload.data?.id) {
      throw new Error("加载会话失败: 响应缺少 data");
    }

    const messagesFromServer = toClientMessages(payload.data.messages);
    replaceSessionMessages({
      id: payload.data.id,
      messages: messagesFromServer,
      updatedAt: payload.data.updated_at,
      activate,
    });
  }, [replaceSessionMessages]);

  const ensureDraftSession = useCallback((forceCreate = false): string => {
    const state = useChatStore.getState();
    const active = state.sessions.find((item) => item.id === state.activeSessionId) || null;
    const activeHasUserMessages = Boolean(active?.messages.some((item) => item.role === "user"));

    if (!forceCreate && active && isTransientSessionID(active.id) && !activeHasUserMessages) {
      setActiveSessionId(active.id);
      return active.id;
    }

    const draftID = createDraftSessionID();
    createSession({
      id: draftID,
      title: "新会话",
      updatedAt: new Date().toISOString(),
      activate: true,
    });
    return draftID;
  }, [createSession, setActiveSessionId]);

  const handleCreateSession = useCallback(async () => {
    if (isSessionCreating || isLoading) {
      return;
    }

    setIsSessionCreating(true);
    try {
      ensureDraftSession(true);
      textareaRef.current?.focus();
    } finally {
      setIsSessionCreating(false);
    }
  }, [ensureDraftSession, isLoading, isSessionCreating]);

  const handleSwitchSession = useCallback(async (targetSessionId: string) => {
    const safeSessionID = targetSessionId.trim();
    if (!safeSessionID || safeSessionID === activeSessionId || isSessionSwitching) {
      return;
    }

    setActiveSessionId(safeSessionID);
    if (isTransientSessionID(safeSessionID)) {
      return;
    }

    setIsSessionSwitching(true);
    try {
      await syncSessionFromServer(safeSessionID, true);
    } catch (error) {
      toast.error(toErrorMessage(error, "切换会话失败"));
    } finally {
      setIsSessionSwitching(false);
    }
  }, [activeSessionId, isSessionSwitching, setActiveSessionId, syncSessionFromServer]);

  useEffect(() => {
    if (!hasHydrated) {
      return;
    }

    let cancelled = false;

    const bootstrap = async () => {
      setIsSessionBootstrapping(true);
      ensureDraftSession(false);
      try {
        const response = await fetch(getAPIURL("/api/agent/sessions"), {
          method: "GET",
          headers: { "Content-Type": "application/json" },
        });
        if (!response.ok) {
          throw await parseAPIError(response, "加载会话列表失败");
        }

        const payload = (await response.json()) as { data?: AgentSessionSummaryView[] };
        const summaries = Array.isArray(payload.data) ? payload.data : [];
        mergeSessionSummaries(
          summaries.map((item) => ({
            id: item.id,
            title: item.title,
            updatedAt: item.updated_at,
          }))
        );
      } catch (error) {
        if (!cancelled) {
          toast.error(toErrorMessage(error, "加载会话列表失败"));
        }
      } finally {
        if (!cancelled) {
          setIsSessionBootstrapping(false);
        }
      }
    };

    Promise.resolve().then(() => {
      if (!cancelled) {
        void bootstrap();
      }
    });

    return () => {
      cancelled = true;
      setIsSessionBootstrapping(false);
    };
  }, [ensureDraftSession, hasHydrated, mergeSessionSummaries]);

  const loadSettings = async () => {
    setIsSettingsLoading(true);
    try {
      const response = await fetch(getAPIURL("/api/agent/settings"), {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) {
        throw await parseAPIError(response, "加载 Agent 设置失败");
      }

      const payload = (await response.json()) as { data?: AgentSettingsView };
      if (!payload.data) {
        throw new Error("加载 Agent 设置失败: 响应缺少 data");
      }

      setSettings((prev) => ({
        ...prev,
        model: payload.data!.model,
        maxSteps: payload.data!.max_steps,
        systemPrompt: payload.data!.system_prompt,
        openaiBaseURL: payload.data!.openai_base_url,
        openaiTimeoutSeconds: payload.data!.openai_timeout_seconds,
        openaiAPIKey: "",
        hasOpenAIAPIKey: payload.data!.has_openai_api_key,
      }));
    } catch (error) {
      toast.error(toErrorMessage(error, "加载 Agent 设置失败"));
    } finally {
      setIsSettingsLoading(false);
    }
  };

  useEffect(() => {
    if (isSettingsOpen) {
      void loadSettings();
    }
  }, [isSettingsOpen]);

  const loadProfile = async () => {
    setIsProfileLoading(true);
    try {
      const response = await fetch(getAPIURL("/api/user/profile"), {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) {
        throw await parseAPIError(response, "加载能力画像失败");
      }

      const payload = (await response.json()) as { data?: UserProfileView };
      if (!payload.data) {
        throw new Error("加载能力画像失败: 响应缺少 data");
      }

      setProfile(toProfileForm(payload.data));
      setProfileUpdatedAt(payload.data.updated_at || "");
    } catch (error) {
      toast.error(toErrorMessage(error, "加载能力画像失败"));
    } finally {
      setIsProfileLoading(false);
    }
  };

  useEffect(() => {
    if (isProfileOpen) {
      void loadProfile();
    } else {
      setResumeFile(null);
    }
  }, [isProfileOpen]);

  const saveProfile = async () => {
    if (isProfileSaving) {
      return;
    }

    const payload = toProfilePayload(profile);
    setIsProfileSaving(true);
    try {
      const response = await fetch(getAPIURL("/api/user/profile"), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "保存能力画像失败");
      }

      const body = (await response.json()) as { data?: UserProfileView };
      if (!body.data) {
        throw new Error("保存能力画像失败: 响应缺少 data");
      }

      setProfile(toProfileForm(body.data));
      setProfileUpdatedAt(body.data.updated_at || "");
      toast.success("能力画像已保存，Agent 会自动使用最新画像");
      setIsProfileOpen(false);
    } catch (error) {
      toast.error(toErrorMessage(error, "保存能力画像失败"));
    } finally {
      setIsProfileSaving(false);
    }
  };

  const importProfileFromResume = async (selectedFile?: File | null) => {
    const file = selectedFile ?? resumeFile;
    if (!file) {
      toast.error("请先选择简历文件");
      return;
    }
    if (isProfileImporting) {
      return;
    }

    const formData = new FormData();
    formData.append("resume", file);

    setIsProfileImporting(true);
    try {
      const response = await fetch(getAPIURL("/api/user/profile/import"), {
        method: "POST",
        body: formData,
      });
      if (!response.ok) {
        throw await parseAPIError(response, "简历导入失败");
      }

      const body = (await response.json()) as { data?: UserProfileImportView };
      if (!body.data?.profile) {
        throw new Error("简历导入失败: 响应缺少 profile");
      }

      setProfile(toProfileForm(body.data.profile));
      setProfileUpdatedAt(body.data.profile.updated_at || "");
      const suffix = body.data.truncated ? "（画像抽取时已截断部分内容）" : "";
      toast.success(`简历导入完成，已更新能力画像与完整简历事实源${suffix}`);
    } catch (error) {
      toast.error(toErrorMessage(error, "简历导入失败"));
    } finally {
      setIsProfileImporting(false);
    }
  };

  const saveSettings = async () => {
    if (isSettingsSaving) {
      return;
    }

    const payload: Record<string, unknown> = {
      model: settings.model.trim(),
      max_steps: Number(settings.maxSteps),
      system_prompt: settings.systemPrompt,
      openai_base_url: settings.openaiBaseURL.trim(),
      openai_timeout_seconds: Number(settings.openaiTimeoutSeconds),
    };
    if (settings.openaiAPIKey.trim() !== "") {
      payload.openai_api_key = settings.openaiAPIKey.trim();
    }

    setIsSettingsSaving(true);
    try {
      const response = await fetch(getAPIURL("/api/agent/settings"), {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "保存 Agent 设置失败");
      }

      const body = (await response.json()) as { data?: AgentSettingsView };
      if (!body.data) {
        throw new Error("保存 Agent 设置失败: 响应缺少 data");
      }

      setSettings((prev) => ({
        ...prev,
        openaiAPIKey: "",
        hasOpenAIAPIKey: body.data!.has_openai_api_key,
      }));

      toast.success("Agent 设置已更新并生效");
      setIsSettingsOpen(false);
    } catch (error) {
      toast.error(toErrorMessage(error, "保存 Agent 设置失败"));
    } finally {
      setIsSettingsSaving(false);
    }
  };

  const updateProfileField = <K extends keyof UserProfileForm>(key: K, value: UserProfileForm[K]) => {
    setProfile((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!input.trim() || isLoading || isSessionSwitching || isSessionCreating) return;

    const userMessage = input.trim();
    setInput("");

    let currentSessionId = sessionId.trim();
    if (!currentSessionId) {
      currentSessionId = ensureDraftSession(true);
    }
    const requestSessionId = isTransientSessionID(currentSessionId) ? "" : currentSessionId;

    appendMessageToActiveSession({
      role: "user",
      content: userMessage,
    });

    setLoading(true);

    try {
      const response = await fetch(getAPIURL("/api/agent/chat"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          session_id: requestSessionId,
          message: userMessage,
          history: [],
        }),
      });
      if (!response.ok) {
        throw await parseAPIError(response, "Agent 请求失败");
      }

      const payload = (await response.json()) as {
        data?: {
          session_id?: string;
          reply?: string;
        };
      };

      const nextSessionID = payload.data?.session_id?.trim() || "";
      const reply = payload.data?.reply?.trim() || "";

      if (!reply) {
        throw new Error("Agent 返回了空回复");
      }
      if (nextSessionID && nextSessionID !== currentSessionId) {
        renameSessionId(currentSessionId, nextSessionID);
        currentSessionId = nextSessionID;
      }

      appendMessageToActiveSession({
        role: "assistant",
        content: reply,
      });
    } catch (error) {
      const message = toErrorMessage(error, "Agent 请求失败");
      appendMessageToActiveSession({
        role: "assistant",
        content: `请求失败：${message}`,
      });
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void handleSubmit();
    }
  };

  const quickActions = [
    { label: "分析我的投递策略", icon: Sparkles },
    { label: "哪些线索需要跟进", icon: Sparkles },
    { label: "帮我准备面试", icon: Sparkles },
  ];

  return (
    <div className="flex h-full min-h-0 w-full flex-1 flex-col overflow-hidden">
      <div className="border-b border-border/60 bg-background/38 px-4 pb-3 pt-4 backdrop-blur-sm">
        <div className="mx-auto flex w-full max-w-5xl flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-2 min-w-[220px]">
            <Select
              value={activeSessionId}
              onValueChange={(value) => {
                void handleSwitchSession(value);
              }}
              disabled={isSessionBootstrapping || isSessionSwitching || isSessionCreating || sessions.length === 0}
            >
              <SelectTrigger className="w-[240px]">
                <SelectValue placeholder="选择会话" />
              </SelectTrigger>
              <SelectContent>
                {sessions.map((session) => (
                  <SelectItem key={session.id} value={session.id}>
                    {truncateText(session.title || "新会话", 24)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              size="sm"
              onClick={() => void handleCreateSession()}
              disabled={isSessionBootstrapping || isSessionSwitching || isSessionCreating || isLoading}
            >
              <Plus className="w-4 h-4 mr-1" />
              新会话
            </Button>
          </div>

          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => setIsProfileOpen(true)}>
              <ClipboardList className="w-4 h-4 mr-1" />
              能力画像
            </Button>
            <Button variant="outline" size="sm" onClick={() => setIsSettingsOpen(true)}>
              <Settings2 className="w-4 h-4 mr-1" />
              Agent 设置
            </Button>
          </div>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
        <div className="mx-auto w-full max-w-5xl space-y-6">
          {messages.map((message) => (
            <div
              key={message.id}
              className={cn(
                "flex gap-3",
                message.role === "user" ? "justify-end" : "justify-start"
              )}
            >
              {message.role === "assistant" && (
                <div className="w-8 h-8 rounded-full border border-border/70 bg-background/80 shadow-sm flex items-center justify-center flex-shrink-0">
                  <Bot className="w-4 h-4 text-foreground" />
                </div>
              )}
              <div
                className={cn(
                  "max-w-[80%] rounded-[22px] px-4 py-3 shadow-sm",
                  message.role === "user"
                    ? "bg-primary text-primary-foreground shadow-[var(--panel-shadow)]"
                    : "border border-border/70 bg-background/72 text-foreground"
                )}
              >
                <p className="text-sm leading-relaxed whitespace-pre-wrap">{message.content}</p>
              </div>
              {message.role === "user" && (
                <div className="w-8 h-8 rounded-full bg-primary shadow-[var(--panel-shadow)] flex items-center justify-center flex-shrink-0">
                  <User className="w-4 h-4 text-primary-foreground" />
                </div>
              )}
            </div>
          ))}

          {isLoading && (
            <div className="flex gap-3">
              <div className="w-8 h-8 rounded-full border border-border/70 bg-background/80 shadow-sm flex items-center justify-center flex-shrink-0">
                <Bot className="w-4 h-4 text-foreground" />
              </div>
              <div className="rounded-[22px] border border-border/70 bg-background/72 px-4 py-3 shadow-sm">
                <div className="flex gap-1">
                  <span className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce" style={{ animationDelay: "0ms" }} />
                  <span className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce" style={{ animationDelay: "150ms" }} />
                  <span className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce" style={{ animationDelay: "300ms" }} />
                </div>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>
      </div>

      {messages.length === 1 && (
        <div className="px-4 pb-4">
          <div className="mx-auto w-full max-w-5xl">
            <div className="flex flex-wrap gap-2">
              {quickActions.map((action) => (
                <Button
                  key={action.label}
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setInput(action.label);
                    textareaRef.current?.focus();
                  }}
                  className="rounded-full border border-border/70 bg-background/72 text-muted-foreground hover:text-foreground"
                >
                  <action.icon className="w-3 h-3 mr-1" />
                  {action.label}
                </Button>
              ))}
            </div>
          </div>
        </div>
      )}

      <div className="border-t border-border/60 bg-background/42 p-4 backdrop-blur-sm">
        <form onSubmit={handleSubmit} className="mx-auto w-full max-w-5xl">
          <div className="relative">
            <Textarea
              ref={textareaRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="输入消息，让 Agent 帮你管理求职..."
              className="min-h-[52px] max-h-32 resize-none border-border/80 bg-background/74 pr-12 focus-visible:border-primary/70 focus-visible:ring-primary/45"
              rows={1}
              disabled={isSessionSwitching || isSessionCreating || !hasHydrated}
            />
            <Button
              type="submit"
              size="icon"
              disabled={
                !input.trim() ||
                isLoading ||
                isSessionSwitching ||
                isSessionCreating ||
                !hasHydrated
              }
              className="absolute right-2 bottom-2 h-8 w-8"
            >
              <Send className="w-4 h-4" />
            </Button>
          </div>
          <p className="text-xs text-muted-foreground mt-2 text-center">
            当前线索数: {leads.length}
            {sessionId ? ` · 会话ID: ${sessionId}` : ""}
            {isSessionBootstrapping ? " · 正在加载会话..." : ""}
            {isSessionSwitching ? " · 正在切换会话..." : ""}
          </p>
        </form>
      </div>

      <Dialog open={isProfileOpen} onOpenChange={setIsProfileOpen}>
        <DialogContent className="sm:max-w-4xl max-h-[92vh] overflow-hidden [&_input]:border-border/90 [&_input]:bg-background/70 [&_input]:focus-visible:border-primary/70 [&_input]:focus-visible:ring-primary/45 [&_textarea]:border-border/90 [&_textarea]:bg-background/70 [&_textarea]:focus-visible:border-primary/70 [&_textarea]:focus-visible:ring-primary/45">
          <DialogHeader>
            <DialogTitle>用户能力画像</DialogTitle>
            <DialogDescription>维护技能、年限、项目证据与求职偏好，让 Agent 更了解你。</DialogDescription>
          </DialogHeader>

          <div className="space-y-4 max-h-[68vh] overflow-y-auto pr-2">
            <div className="rounded-md border border-border/90 p-3 space-y-3">
              <div>
                <p className="text-sm font-medium">简历导入</p>
                <p className="text-xs text-muted-foreground">
                  支持 pdf / docx / txt / md，选择文件后自动导入，导入后你还能继续手动修。
                </p>
              </div>
              <Input
                type="file"
                accept=".pdf,.docx,.txt,.md,.markdown,.json,.yaml,.yml,.csv"
                onChange={(e) => {
                  const file = e.target.files?.[0] || null;
                  setResumeFile(file);
                  if (file) {
                    void importProfileFromResume(file);
                  }
                  e.currentTarget.value = "";
                }}
                disabled={isProfileLoading || isProfileSaving || isProfileImporting}
              />
              <p className="text-xs text-muted-foreground">
                {isProfileImporting ? "正在自动导入..." : ""}
                {isProfileImporting && (resumeFile || profileUpdatedAt) ? " · " : ""}
                {resumeFile ? `已选择: ${resumeFile.name}` : "未选择文件"}
                {profileUpdatedAt ? ` · 最近更新: ${profileUpdatedAt}` : ""}
              </p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="profile-name">姓名</Label>
                <Input
                  id="profile-name"
                  value={profile.name}
                  onChange={(e) => updateProfileField("name", e.target.value)}
                  disabled={isProfileLoading || isProfileSaving || isProfileImporting}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="profile-title">当前职级/职位</Label>
                <Input
                  id="profile-title"
                  value={profile.currentTitle}
                  onChange={(e) => updateProfileField("currentTitle", e.target.value)}
                  disabled={isProfileLoading || isProfileSaving || isProfileImporting}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="profile-years">总年限</Label>
                <Input
                  id="profile-years"
                  type="number"
                  min={0}
                  step={0.5}
                  value={profile.totalYears}
                  onChange={(e) => updateProfileField("totalYears", e.target.value)}
                  disabled={isProfileLoading || isProfileSaving || isProfileImporting}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="profile-core-skills">核心技能</Label>
                <Textarea
                  id="profile-core-skills"
                  value={profile.coreSkills}
                  onChange={(e) => updateProfileField("coreSkills", e.target.value)}
                  placeholder="每行一条，如：Go / 系统设计 / 分布式事务"
                  className="min-h-[96px]"
                  disabled={isProfileLoading || isProfileSaving || isProfileImporting}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="profile-programming">编程语言</Label>
                <Textarea
                  id="profile-programming"
                  value={profile.programmingLanguages}
                  onChange={(e) => updateProfileField("programmingLanguages", e.target.value)}
                  placeholder="每行一条"
                  className="min-h-[96px]"
                  disabled={isProfileLoading || isProfileSaving || isProfileImporting}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="profile-project-evidence">项目证据</Label>
              <Textarea
                id="profile-project-evidence"
                value={profile.projectEvidence}
                onChange={(e) => updateProfileField("projectEvidence", e.target.value)}
                placeholder="每行一条：项目/成果/量化指标"
                className="min-h-[110px]"
                disabled={isProfileLoading || isProfileSaving || isProfileImporting}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="profile-preferred-roles">目标岗位</Label>
                <Textarea
                  id="profile-preferred-roles"
                  value={profile.preferredRoles}
                  onChange={(e) => updateProfileField("preferredRoles", e.target.value)}
                  placeholder="每行一条"
                  className="min-h-[96px]"
                  disabled={isProfileLoading || isProfileSaving || isProfileImporting}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="profile-preferred-locations">期望城市/地区</Label>
                <Textarea
                  id="profile-preferred-locations"
                  value={profile.preferredLocations}
                  onChange={(e) => updateProfileField("preferredLocations", e.target.value)}
                  placeholder="每行一条"
                  className="min-h-[96px]"
                  disabled={isProfileLoading || isProfileSaving || isProfileImporting}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="profile-priorities">求职优先级</Label>
              <Textarea
                id="profile-priorities"
                value={profile.jobSearchPriorities}
                onChange={(e) => updateProfileField("jobSearchPriorities", e.target.value)}
                placeholder="成长 / 薪资 / WLB / 技术深度 ..."
                className="min-h-[96px]"
                disabled={isProfileLoading || isProfileSaving || isProfileImporting}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="profile-strength">能力总结</Label>
              <Textarea
                id="profile-strength"
                value={profile.strengthSummary}
                onChange={(e) => updateProfileField("strengthSummary", e.target.value)}
                placeholder="一句话总结你的竞争力"
                className="min-h-[90px]"
                disabled={isProfileLoading || isProfileSaving || isProfileImporting}
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsProfileOpen(false)} disabled={isProfileSaving || isProfileImporting}>
              取消
            </Button>
            <Button onClick={() => void saveProfile()} disabled={isProfileLoading || isProfileSaving || isProfileImporting}>
              {isProfileSaving ? "保存中..." : "保存画像"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isSettingsOpen} onOpenChange={setIsSettingsOpen}>
        <DialogContent className="sm:max-w-xl [&_input]:border-border/90 [&_input]:bg-background/70 [&_input]:focus-visible:border-primary/70 [&_input]:focus-visible:ring-primary/45 [&_textarea]:border-border/90 [&_textarea]:bg-background/70 [&_textarea]:focus-visible:border-primary/70 [&_textarea]:focus-visible:ring-primary/45">
          <DialogHeader>
            <DialogTitle>Agent 设置</DialogTitle>
            <DialogDescription>修改后端运行时配置，保存后立即生效。</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="agent-model">模型</Label>
              <Input
                id="agent-model"
                value={settings.model}
                onChange={(e) => setSettings((prev) => ({ ...prev, model: e.target.value }))}
                disabled={isSettingsLoading || isSettingsSaving}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="agent-max-steps">最大步骤</Label>
                <Input
                  id="agent-max-steps"
                  type="number"
                  min={1}
                  value={settings.maxSteps}
                  onChange={(e) =>
                    setSettings((prev) => ({
                      ...prev,
                      maxSteps: Number(e.target.value || 0),
                    }))
                  }
                  disabled={isSettingsLoading || isSettingsSaving}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="agent-timeout">OpenAI 超时(秒)</Label>
                <Input
                  id="agent-timeout"
                  type="number"
                  min={1}
                  value={settings.openaiTimeoutSeconds}
                  onChange={(e) =>
                    setSettings((prev) => ({
                      ...prev,
                      openaiTimeoutSeconds: Number(e.target.value || 0),
                    }))
                  }
                  disabled={isSettingsLoading || isSettingsSaving}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="agent-base-url">OpenAI Base URL</Label>
              <Input
                id="agent-base-url"
                value={settings.openaiBaseURL}
                onChange={(e) =>
                  setSettings((prev) => ({
                    ...prev,
                    openaiBaseURL: e.target.value,
                  }))
                }
                disabled={isSettingsLoading || isSettingsSaving}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="agent-api-key">OpenAI API Key</Label>
              <Input
                id="agent-api-key"
                type="password"
                value={settings.openaiAPIKey}
                onChange={(e) =>
                  setSettings((prev) => ({
                    ...prev,
                    openaiAPIKey: e.target.value,
                  }))
                }
                placeholder={settings.hasOpenAIAPIKey ? "已配置，留空则不修改" : "输入新的 API Key"}
                disabled={isSettingsLoading || isSettingsSaving}
              />
              <p className="text-xs text-muted-foreground">
                当前状态：{settings.hasOpenAIAPIKey ? "已配置" : "未配置"}
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="agent-system-prompt">System Prompt</Label>
              <Textarea
                id="agent-system-prompt"
                value={settings.systemPrompt}
                onChange={(e) =>
                  setSettings((prev) => ({
                    ...prev,
                    systemPrompt: e.target.value,
                  }))
                }
                className="min-h-[120px]"
                disabled={isSettingsLoading || isSettingsSaving}
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsSettingsOpen(false)} disabled={isSettingsSaving}>
              取消
            </Button>
            <Button onClick={() => void saveSettings()} disabled={isSettingsLoading || isSettingsSaving}>
              {isSettingsSaving ? "保存中..." : "保存并生效"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
