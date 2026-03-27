import { DEFAULT_PREP_META, DEFAULT_PREP_SCOPES, PrepMeta, PrepScope } from "./prep-types";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
const PREP_SCOPE_SET = new Set<PrepScope>(DEFAULT_PREP_SCOPES);

interface APIPrepMetaPayload {
  data?: {
    enabled?: boolean;
    default_question_count?: number;
    supported_scopes?: string[];
  };
}

interface APIErrorPayload {
  message?: string;
  error?: string;
}

function getAPIURL(path: string): string {
  return `${API_BASE_URL}${path}`;
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

function normalizePrepScopes(raw: string[] | undefined): PrepScope[] {
  if (!Array.isArray(raw) || raw.length === 0) {
    return [...DEFAULT_PREP_SCOPES];
  }

  const normalized: PrepScope[] = [];
  const seen = new Set<PrepScope>();
  for (const item of raw) {
    const scope = (item || "").trim() as PrepScope;
    if (!PREP_SCOPE_SET.has(scope) || seen.has(scope)) {
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
