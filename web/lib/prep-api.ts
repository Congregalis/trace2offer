import {
  DEFAULT_PREP_META,
  DEFAULT_PREP_SCOPES,
  PrepContextSource,
  PrepKnowledgeDocument,
  PrepKnowledgeDocumentCreateInput,
  PrepKnowledgeDocumentUpdateInput,
  PrepLeadContextPreview,
  PrepMeta,
  PrepScope,
  PrepTopic,
  PrepTopicCreateInput,
  PrepTopicPatchInput,
} from "./prep-types";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
const PREP_SCOPE_SET = new Set<PrepScope>(DEFAULT_PREP_SCOPES);

interface APIPrepMetaPayload {
  data?: {
    enabled?: boolean;
    default_question_count?: number;
    supported_scopes?: string[];
  };
}

interface APITopic {
  key?: string;
  name?: string;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

interface APIKnowledgeDocument {
  scope?: string;
  scope_id?: string;
  filename?: string;
  content?: string;
  updated_at?: string;
}

interface APIPrepContextSource {
  scope?: string;
  kind?: string;
  title?: string;
}

interface APIPrepLeadContextPreview {
  lead_id?: string;
  company?: string;
  position?: string;
  has_resume?: boolean;
  has_profile?: boolean;
  topic_keys?: string[];
  sources?: APIPrepContextSource[];
}

interface APIListPayload<T> {
  data?: T[];
}

interface APISinglePayload<T> {
  data?: T;
}

interface APIErrorPayload {
  message?: string;
  error?: string;
}

function getAPIURL(path: string): string {
  return `${API_BASE_URL}${path}`;
}

function normalizeScope(scope: string): PrepScope {
  const candidate = (scope || "").trim() as PrepScope;
  if (PREP_SCOPE_SET.has(candidate)) {
    return candidate;
  }
  return DEFAULT_PREP_SCOPES[0];
}

function normalizePrepScopes(raw: string[] | undefined): PrepScope[] {
  if (!Array.isArray(raw) || raw.length === 0) {
    return [...DEFAULT_PREP_SCOPES];
  }

  const normalized: PrepScope[] = [];
  const seen = new Set<PrepScope>();
  for (const item of raw) {
    const scope = normalizeScope(item || "");
    if (seen.has(scope)) {
      continue;
    }
    seen.add(scope);
    normalized.push(scope);
  }

  return normalized.length > 0 ? normalized : [...DEFAULT_PREP_SCOPES];
}

function normalizeQuestionCount(raw: number | undefined): number {
  if (typeof raw !== "number" || Number.isNaN(raw) || raw <= 0) {
    return DEFAULT_PREP_META.defaultQuestionCount;
  }
  return Math.floor(raw);
}

function normalizeTopic(raw: APITopic): PrepTopic {
  return {
    key: (raw.key || "").trim(),
    name: (raw.name || "").trim(),
    description: (raw.description || "").trim(),
    createdAt: (raw.created_at || "").trim(),
    updatedAt: (raw.updated_at || "").trim(),
  };
}

function normalizeDocument(raw: APIKnowledgeDocument): PrepKnowledgeDocument {
  return {
    scope: normalizeScope(raw.scope || ""),
    scopeId: (raw.scope_id || "").trim(),
    filename: (raw.filename || "").trim(),
    content: raw.content || "",
    updatedAt: (raw.updated_at || "").trim(),
  };
}

function normalizeContextSource(raw: APIPrepContextSource): PrepContextSource {
  return {
    scope: (raw.scope || "").trim(),
    kind: (raw.kind || "").trim(),
    title: (raw.title || "").trim(),
  };
}

function normalizeContextPreview(raw: APIPrepLeadContextPreview | undefined): PrepLeadContextPreview {
  const sources = Array.isArray(raw?.sources) ? raw.sources.map(normalizeContextSource) : [];
  const topicKeys = Array.isArray(raw?.topic_keys)
    ? raw.topic_keys
        .map((key) => (key || "").trim())
        .filter((key) => key.length > 0)
    : [];

  return {
    leadId: (raw?.lead_id || "").trim(),
    company: (raw?.company || "").trim(),
    position: (raw?.position || "").trim(),
    hasResume: Boolean(raw?.has_resume),
    hasProfile: Boolean(raw?.has_profile),
    topicKeys,
    sources,
  };
}

async function parseAPIError(response: Response, fallback: string): Promise<Error> {
  try {
    const payload = (await response.json()) as APIErrorPayload;
    const details = [payload.message, payload.error].filter(Boolean).join(": ");
    if (details) {
      return new Error(details);
    }
  } catch {
    // ignore non-json error payloads
  }
  return new Error(`${fallback} (HTTP ${response.status})`);
}

function encodeSegment(value: string): string {
  return encodeURIComponent((value || "").trim());
}

export async function fetchPrepMeta(signal?: AbortSignal): Promise<PrepMeta> {
  const response = await fetch(getAPIURL("/api/prep/meta"), {
    method: "GET",
    headers: { "Content-Type": "application/json" },
    signal,
  });
  if (!response.ok) {
    throw await parseAPIError(response, "加载备面元信息失败");
  }

  const payload = (await response.json()) as APIPrepMetaPayload;
  return {
    enabled: Boolean(payload.data?.enabled),
    defaultQuestionCount: normalizeQuestionCount(payload.data?.default_question_count),
    supportedScopes: normalizePrepScopes(payload.data?.supported_scopes),
  };
}

export async function listPrepTopics(signal?: AbortSignal): Promise<PrepTopic[]> {
  const response = await fetch(getAPIURL("/api/prep/topics"), {
    method: "GET",
    headers: { "Content-Type": "application/json" },
    signal,
  });
  if (!response.ok) {
    throw await parseAPIError(response, "加载备面主题失败");
  }
  const payload = (await response.json()) as APIListPayload<APITopic>;
  const topics = Array.isArray(payload.data) ? payload.data : [];
  return topics.map(normalizeTopic);
}

export async function fetchPrepLeadContextPreview(leadId: string, signal?: AbortSignal): Promise<PrepLeadContextPreview> {
  const normalizedLeadID = (leadId || "").trim();
  if (!normalizedLeadID) {
    throw new Error("lead_id is required");
  }

  const response = await fetch(getAPIURL(`/api/prep/leads/${encodeSegment(normalizedLeadID)}/context-preview`), {
    method: "GET",
    headers: { "Content-Type": "application/json" },
    signal,
  });
  if (!response.ok) {
    throw await parseAPIError(response, "加载备面上下文失败");
  }
  const payload = (await response.json()) as APISinglePayload<APIPrepLeadContextPreview>;
  return normalizeContextPreview(payload.data);
}

export async function createPrepTopic(input: PrepTopicCreateInput): Promise<PrepTopic> {
  const response = await fetch(getAPIURL("/api/prep/topics"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      key: (input.key || "").trim(),
      name: (input.name || "").trim(),
      description: (input.description || "").trim(),
    }),
  });
  if (!response.ok) {
    throw await parseAPIError(response, "创建备面主题失败");
  }
  const payload = (await response.json()) as APISinglePayload<APITopic>;
  if (!payload.data) {
    throw new Error("创建备面主题响应缺少 data");
  }
  return normalizeTopic(payload.data);
}

export async function updatePrepTopic(key: string, patch: PrepTopicPatchInput): Promise<PrepTopic> {
  const payload: Record<string, string> = {};
  if (typeof patch.name === "string") {
    payload.name = patch.name.trim();
  }
  if (typeof patch.description === "string") {
    payload.description = patch.description.trim();
  }

  const response = await fetch(getAPIURL(`/api/prep/topics/${encodeSegment(key)}`), {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    throw await parseAPIError(response, "更新备面主题失败");
  }
  const body = (await response.json()) as APISinglePayload<APITopic>;
  if (!body.data) {
    throw new Error("更新备面主题响应缺少 data");
  }
  return normalizeTopic(body.data);
}

export async function deletePrepTopic(key: string): Promise<void> {
  const response = await fetch(getAPIURL(`/api/prep/topics/${encodeSegment(key)}`), {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
  });
  if (!response.ok) {
    throw await parseAPIError(response, "删除备面主题失败");
  }
}

export async function listPrepKnowledgeDocuments(scope: PrepScope, scopeId: string, signal?: AbortSignal): Promise<PrepKnowledgeDocument[]> {
  const response = await fetch(getAPIURL(`/api/prep/knowledge/${encodeSegment(scope)}/${encodeSegment(scopeId)}/documents`), {
    method: "GET",
    headers: { "Content-Type": "application/json" },
    signal,
  });
  if (!response.ok) {
    throw await parseAPIError(response, "加载备面资料失败");
  }
  const payload = (await response.json()) as APIListPayload<APIKnowledgeDocument>;
  const documents = Array.isArray(payload.data) ? payload.data : [];
  return documents.map(normalizeDocument);
}

export async function createPrepKnowledgeDocument(
  scope: PrepScope,
  scopeId: string,
  input: PrepKnowledgeDocumentCreateInput,
): Promise<PrepKnowledgeDocument> {
  const response = await fetch(getAPIURL(`/api/prep/knowledge/${encodeSegment(scope)}/${encodeSegment(scopeId)}/documents`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      filename: (input.filename || "").trim(),
      content: input.content || "",
    }),
  });
  if (!response.ok) {
    throw await parseAPIError(response, "创建备面资料失败");
  }
  const payload = (await response.json()) as APISinglePayload<APIKnowledgeDocument>;
  if (!payload.data) {
    throw new Error("创建备面资料响应缺少 data");
  }
  return normalizeDocument(payload.data);
}

export async function updatePrepKnowledgeDocument(
  scope: PrepScope,
  scopeId: string,
  filename: string,
  input: PrepKnowledgeDocumentUpdateInput,
): Promise<PrepKnowledgeDocument> {
  const response = await fetch(
    getAPIURL(`/api/prep/knowledge/${encodeSegment(scope)}/${encodeSegment(scopeId)}/documents/${encodeSegment(filename)}`),
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        content: input.content || "",
      }),
    },
  );
  if (!response.ok) {
    throw await parseAPIError(response, "更新备面资料失败");
  }
  const payload = (await response.json()) as APISinglePayload<APIKnowledgeDocument>;
  if (!payload.data) {
    throw new Error("更新备面资料响应缺少 data");
  }
  return normalizeDocument(payload.data);
}

export async function deletePrepKnowledgeDocument(scope: PrepScope, scopeId: string, filename: string): Promise<void> {
  const response = await fetch(
    getAPIURL(`/api/prep/knowledge/${encodeSegment(scope)}/${encodeSegment(scopeId)}/documents/${encodeSegment(filename)}`),
    {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
    },
  );
  if (!response.ok) {
    throw await parseAPIError(response, "删除备面资料失败");
  }
}
