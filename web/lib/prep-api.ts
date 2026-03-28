import {
  DEFAULT_PREP_META,
  DEFAULT_PREP_SCOPES,
  PrepContextSource,
  PrepCreateSessionInput,
  PrepIndexChunk,
  PrepIndexDocument,
  PrepIndexStatus,
  PrepIndexRebuildInput,
  PrepIndexRunError,
  PrepIndexRunSummary,
  PrepKnowledgeDocument,
  PrepKnowledgeDocumentCreateInput,
  PrepKnowledgeDocumentUpdateInput,
  PrepDraftAnswersSaveResult,
  PrepAnswer,
  PrepGenerationProgressEvent,
  PrepGenerationTrace,
  PrepLeadContextPreview,
  PrepMeta,
  PrepQuestion,
  PrepRetrievalPreview,
  PrepRetrievalPreviewRequest,
  PrepScope,
  PrepSession,
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
  sources?: APIPrepContextSource[];
}

interface APIPrepIndexRunSummary {
  run_id?: string;
  mode?: string;
  started_at?: string;
  completed_at?: string;
  status?: string;
  documents_scanned?: number;
  documents_indexed?: number;
  documents_skipped?: number;
  documents_deleted?: number;
  chunks_created?: number;
  chunks_updated?: number;
  errors?: Array<{ source?: string; message?: string }>;
}

interface APIPrepIndexStatus {
  embedding_provider?: string;
  embedding_model?: string;
  document_count?: number;
  chunk_count?: number;
  last_indexed_at?: string;
  last_index_status?: string;
}

interface APIPrepIndexDocument {
  id?: string;
  scope?: string;
  scope_id?: string;
  kind?: string;
  title?: string;
  source_path?: string;
  content_hash?: string;
  updated_at?: string;
}

interface APIPrepIndexChunk {
  id?: string;
  document_id?: string;
  scope?: string;
  scope_id?: string;
  document_title?: string;
  chunk_index?: number;
  content?: string;
  token_count?: number;
  updated_at?: string;
}

interface APIRetrievedChunk {
  id?: string;
  content?: string;
  score?: number;
  why_selected?: string;
  source?: {
    scope?: string;
    scope_id?: string;
    document_title?: string;
    chunk_index?: number;
  };
}

interface APIRetrievalPreview {
  query?: string;
  normalized_query?: string;
  filters?: {
    scope?: string[];
  };
  trace?: {
    stage_query_normalization?: APITraceStage;
    stage_initial_retrieval?: APITraceStage;
    stage_deduplication?: APITraceStage;
    stage_reranking?: APITraceStage;
  };
  candidate_chunks?: APIRetrievedChunk[];
  retrieved_chunks?: APIRetrievedChunk[];
  final_context?: {
    total_tokens?: number;
    chunks_used?: number;
    context?: string;
  };
}

interface APITraceStage {
  input?: string;
  output?: string;
  method?: string;
  input_count?: number;
  output_count?: number;
  metadata?: Record<string, unknown>;
}

interface APIPrepQuestion {
  id?: number;
  type?: string;
  content?: string;
  expected_points?: string[];
  context_sources?: string[];
}

interface APIPrepAnswer {
  question_id?: number;
  answer?: string;
  submitted_at?: string;
}

interface APIPrepSession {
  id?: string;
  lead_id?: string;
  company?: string;
  position?: string;
  status?: string;
  config?: {
    question_count?: number;
    include_resume?: boolean;
    include_lead_docs?: boolean;
  };
  sources?: APIPrepContextSource[];
  questions?: APIPrepQuestion[];
  answers?: APIPrepAnswer[];
  evaluation?: unknown;
  reference_answers?: Record<string, unknown>;
  generation_trace?: {
    input_snapshot?: {
      lead_id?: string;
      question_count?: number;
    };
    query_planning?: {
      strategy?: string;
      model?: string;
      resume_excerpt?: string;
      jd_excerpt?: string;
      prompt?: string;
      raw_output?: string;
      final_query?: string;
    };
    retrieval_query?: string;
    retrieval_results?: {
      candidates_found?: number;
      final_selected?: number;
      sources?: string[];
    };
    prompt_sections?: Array<{
      title?: string;
      content?: string;
    }>;
    assembled_prompt?: string;
    generation_result?: {
      questions_generated?: number;
      generation_time_ms?: number;
      model?: string;
    };
  };
  created_at?: string;
  updated_at?: string;
}

interface APIPrepDraftAnswersSaveResult {
  session_id?: string;
  saved_at?: string;
  answers_count?: number;
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

interface APIPrepStreamStageEvent {
  stage?: string;
  status?: string;
  message?: string;
  delta?: string;
  trace?: APIPrepSession["generation_trace"];
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

  return {
    leadId: (raw?.lead_id || "").trim(),
    company: (raw?.company || "").trim(),
    position: (raw?.position || "").trim(),
    hasResume: Boolean(raw?.has_resume),
    sources,
  };
}

function normalizeIndexRunSummary(raw: APIPrepIndexRunSummary | undefined): PrepIndexRunSummary {
  const errors: PrepIndexRunError[] = Array.isArray(raw?.errors)
    ? raw.errors
        .map((item) => ({
          source: (item?.source || "").trim(),
          message: (item?.message || "").trim(),
        }))
        .filter((item) => item.source.length > 0 || item.message.length > 0)
    : [];

  return {
    runId: (raw?.run_id || "").trim(),
    mode: (raw?.mode || "").trim() || "incremental",
    startedAt: (raw?.started_at || "").trim(),
    completedAt: (raw?.completed_at || "").trim(),
    status: (raw?.status || "").trim(),
    documentsScanned: typeof raw?.documents_scanned === "number" ? raw.documents_scanned : 0,
    documentsIndexed: typeof raw?.documents_indexed === "number" ? raw.documents_indexed : 0,
    documentsSkipped: typeof raw?.documents_skipped === "number" ? raw.documents_skipped : 0,
    documentsDeleted: typeof raw?.documents_deleted === "number" ? raw.documents_deleted : 0,
    chunksCreated: typeof raw?.chunks_created === "number" ? raw.chunks_created : 0,
    chunksUpdated: typeof raw?.chunks_updated === "number" ? raw.chunks_updated : 0,
    errors,
  };
}

function normalizeIndexStatus(raw: APIPrepIndexStatus | undefined): PrepIndexStatus {
  return {
    embeddingProvider: (raw?.embedding_provider || "").trim(),
    embeddingModel: (raw?.embedding_model || "").trim(),
    documentCount: typeof raw?.document_count === "number" ? raw.document_count : 0,
    chunkCount: typeof raw?.chunk_count === "number" ? raw.chunk_count : 0,
    lastIndexedAt: (raw?.last_indexed_at || "").trim(),
    lastIndexStatus: (raw?.last_index_status || "").trim(),
  };
}

function normalizeIndexDocument(raw: APIPrepIndexDocument): PrepIndexDocument {
  return {
    id: (raw.id || "").trim(),
    scope: normalizeScope(raw.scope || ""),
    scopeId: (raw.scope_id || "").trim(),
    kind: (raw.kind || "").trim(),
    title: (raw.title || "").trim(),
    sourcePath: (raw.source_path || "").trim(),
    contentHash: (raw.content_hash || "").trim(),
    updatedAt: (raw.updated_at || "").trim(),
  };
}

function normalizeIndexChunk(raw: APIPrepIndexChunk): PrepIndexChunk {
  return {
    id: (raw.id || "").trim(),
    documentId: (raw.document_id || "").trim(),
    scope: normalizeScope(raw.scope || ""),
    scopeId: (raw.scope_id || "").trim(),
    documentTitle: (raw.document_title || "").trim(),
    chunkIndex: typeof raw.chunk_index === "number" && Number.isFinite(raw.chunk_index) ? Math.max(0, Math.floor(raw.chunk_index)) : 0,
    content: raw.content || "",
    tokenCount: typeof raw.token_count === "number" && Number.isFinite(raw.token_count) ? Math.max(0, Math.floor(raw.token_count)) : 0,
    updatedAt: (raw.updated_at || "").trim(),
  };
}

function normalizeRetrievalPreview(raw: APIRetrievalPreview | undefined): PrepRetrievalPreview {
  const normalizeRetrievedChunks = (chunks: APIRetrievedChunk[] | undefined) =>
    Array.isArray(chunks)
      ? chunks.map((chunk) => ({
          id: (chunk.id || "").trim(),
          content: chunk.content || "",
          score: typeof chunk.score === "number" && Number.isFinite(chunk.score) ? chunk.score : 0,
          source: {
            scope: normalizeScope(chunk.source?.scope || ""),
            scopeId: (chunk.source?.scope_id || "").trim(),
            documentTitle: (chunk.source?.document_title || "").trim(),
            chunkIndex:
              typeof chunk.source?.chunk_index === "number" && Number.isFinite(chunk.source.chunk_index)
                ? Math.max(0, Math.floor(chunk.source.chunk_index))
                : 0,
          },
          whySelected: (chunk.why_selected || "").trim(),
        }))
      : [];
  const candidateChunks = normalizeRetrievedChunks(raw?.candidate_chunks);
  const retrievedChunks = normalizeRetrievedChunks(raw?.retrieved_chunks);

  return {
    query: (raw?.query || "").trim(),
    normalizedQuery: (raw?.normalized_query || "").trim(),
    filters: {
      scope: normalizePrepScopes(raw?.filters?.scope),
    },
    trace: raw?.trace
      ? {
          stageQueryNormalization: normalizeTraceStage(raw.trace.stage_query_normalization),
          stageInitialRetrieval: normalizeTraceStage(raw.trace.stage_initial_retrieval),
          stageDeduplication: normalizeTraceStage(raw.trace.stage_deduplication),
          stageReranking: normalizeTraceStage(raw.trace.stage_reranking),
        }
      : undefined,
    candidateChunks,
    retrievedChunks,
    finalContext: {
      totalTokens:
        typeof raw?.final_context?.total_tokens === "number" && Number.isFinite(raw.final_context.total_tokens)
          ? Math.max(0, Math.floor(raw.final_context.total_tokens))
          : 0,
      chunksUsed:
        typeof raw?.final_context?.chunks_used === "number" && Number.isFinite(raw.final_context.chunks_used)
          ? Math.max(0, Math.floor(raw.final_context.chunks_used))
          : 0,
      context: raw?.final_context?.context || "",
    },
  };
}

function normalizeTraceStage(raw: APITraceStage | undefined) {
  return {
    input: typeof raw?.input === "string" ? raw.input.trim() : undefined,
    output: typeof raw?.output === "string" ? raw.output.trim() : undefined,
    method: (raw?.method || "").trim(),
    inputCount:
      typeof raw?.input_count === "number" && Number.isFinite(raw.input_count)
        ? Math.max(0, Math.floor(raw.input_count))
        : undefined,
    outputCount:
      typeof raw?.output_count === "number" && Number.isFinite(raw.output_count)
        ? Math.max(0, Math.floor(raw.output_count))
        : undefined,
    metadata: raw?.metadata && typeof raw.metadata === "object" ? raw.metadata : undefined,
  };
}

function normalizeQuestion(raw: APIPrepQuestion): PrepQuestion {
  const expectedPoints = Array.isArray(raw.expected_points)
    ? raw.expected_points.map((item) => (item || "").trim()).filter((item) => item.length > 0)
    : [];
  const contextSources = Array.isArray(raw.context_sources)
    ? raw.context_sources.map((item) => (item || "").trim()).filter((item) => item.length > 0)
    : [];

  return {
    id: typeof raw.id === "number" && Number.isFinite(raw.id) ? Math.max(0, Math.floor(raw.id)) : 0,
    type: (raw.type || "").trim(),
    content: (raw.content || "").trim(),
    expectedPoints,
    contextSources,
  };
}

function normalizeAnswer(raw: APIPrepAnswer): PrepAnswer {
  return {
    questionId: typeof raw.question_id === "number" && Number.isFinite(raw.question_id) ? Math.max(0, Math.floor(raw.question_id)) : 0,
    answer: raw.answer || "",
    submittedAt: (raw.submitted_at || "").trim() || undefined,
  };
}

function normalizeGenerationTrace(raw: APIPrepSession["generation_trace"] | undefined): PrepGenerationTrace | undefined {
  if (!raw) {
    return undefined;
  }

  const traceSourceTitles = Array.isArray(raw?.retrieval_results?.sources)
    ? raw.retrieval_results.sources.map((item) => (item || "").trim()).filter((item) => item.length > 0)
    : [];
  const promptSections = Array.isArray(raw?.prompt_sections)
    ? raw.prompt_sections
        .map((section) => ({
          title: (section?.title || "").trim(),
          content: (section?.content || "").trim(),
        }))
        .filter((section) => section.title.length > 0 || section.content.length > 0)
    : [];

  return {
    inputSnapshot: {
      leadId: (raw?.input_snapshot?.lead_id || "").trim(),
      questionCount:
        typeof raw?.input_snapshot?.question_count === "number" && Number.isFinite(raw.input_snapshot.question_count)
          ? Math.max(0, Math.floor(raw.input_snapshot.question_count))
          : 0,
    },
    queryPlanning: {
      strategy: (raw?.query_planning?.strategy || "").trim(),
      model: (raw?.query_planning?.model || "").trim(),
      resumeExcerpt: (raw?.query_planning?.resume_excerpt || "").trim(),
      jdExcerpt: (raw?.query_planning?.jd_excerpt || "").trim(),
      prompt: raw?.query_planning?.prompt || "",
      rawOutput: raw?.query_planning?.raw_output || "",
      finalQuery: (raw?.query_planning?.final_query || "").trim(),
    },
    retrievalQuery: (raw?.retrieval_query || "").trim(),
    retrievalResults: {
      candidatesFound:
        typeof raw?.retrieval_results?.candidates_found === "number" && Number.isFinite(raw.retrieval_results.candidates_found)
          ? Math.max(0, Math.floor(raw.retrieval_results.candidates_found))
          : 0,
      finalSelected:
        typeof raw?.retrieval_results?.final_selected === "number" && Number.isFinite(raw.retrieval_results.final_selected)
          ? Math.max(0, Math.floor(raw.retrieval_results.final_selected))
          : 0,
      sources: traceSourceTitles,
    },
    promptSections,
    assembledPrompt: raw?.assembled_prompt || "",
    generationResult: {
      questionsGenerated:
        typeof raw?.generation_result?.questions_generated === "number" && Number.isFinite(raw.generation_result.questions_generated)
          ? Math.max(0, Math.floor(raw.generation_result.questions_generated))
          : 0,
      generationTimeMs:
        typeof raw?.generation_result?.generation_time_ms === "number" && Number.isFinite(raw.generation_result.generation_time_ms)
          ? Math.max(0, Math.floor(raw.generation_result.generation_time_ms))
          : 0,
      model: (raw?.generation_result?.model || "").trim(),
    },
  };
}

function normalizeSession(raw: APIPrepSession | undefined): PrepSession {
  const questions = Array.isArray(raw?.questions)
    ? raw.questions.map(normalizeQuestion).filter((item) => item.id > 0)
    : [];
  const answers = Array.isArray(raw?.answers)
    ? raw.answers.map(normalizeAnswer).filter((item) => item.questionId > 0)
    : [];
  const sources = Array.isArray(raw?.sources) ? raw.sources.map(normalizeContextSource) : [];
  const generationTrace = normalizeGenerationTrace(raw?.generation_trace);

  return {
    id: (raw?.id || "").trim(),
    leadId: (raw?.lead_id || "").trim(),
    company: (raw?.company || "").trim(),
    position: (raw?.position || "").trim(),
    status: (raw?.status || "").trim(),
    config: {
      questionCount:
        typeof raw?.config?.question_count === "number" && Number.isFinite(raw.config.question_count)
          ? Math.max(0, Math.floor(raw.config.question_count))
          : 0,
      includeResume: Boolean(raw?.config?.include_resume),
      includeLeadDocs: Boolean(raw?.config?.include_lead_docs),
    },
    sources,
    questions,
    answers,
    evaluation: raw?.evaluation,
    referenceAnswers: raw?.reference_answers || {},
    generationTrace,
    createdAt: (raw?.created_at || "").trim(),
    updatedAt: (raw?.updated_at || "").trim(),
  };
}

function normalizeDraftAnswersSaveResult(raw: APIPrepDraftAnswersSaveResult | undefined): PrepDraftAnswersSaveResult {
  return {
    sessionId: (raw?.session_id || "").trim(),
    savedAt: (raw?.saved_at || "").trim(),
    answersCount: typeof raw?.answers_count === "number" && Number.isFinite(raw.answers_count) ? Math.max(0, Math.floor(raw.answers_count)) : 0,
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

export async function rebuildPrepIndex(input: PrepIndexRebuildInput): Promise<PrepIndexRunSummary> {
  const response = await fetch(getAPIURL("/api/prep/index/rebuild"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      scope: (input.scope || "*").toString().trim() || "*",
      scope_id: (input.scopeId || "").trim(),
      mode: (input.mode || "incremental").trim(),
    }),
  });
  if (!response.ok) {
    throw await parseAPIError(response, "重建索引失败");
  }
  const payload = (await response.json()) as APISinglePayload<APIPrepIndexRunSummary>;
  return normalizeIndexRunSummary(payload.data);
}

export async function fetchPrepIndexStatus(signal?: AbortSignal): Promise<PrepIndexStatus> {
  const response = await fetch(getAPIURL("/api/prep/index/status"), {
    method: "GET",
    headers: { "Content-Type": "application/json" },
    signal,
  });
  if (!response.ok) {
    throw await parseAPIError(response, "加载索引状态失败");
  }

  const payload = (await response.json()) as APISinglePayload<APIPrepIndexStatus>;
  return normalizeIndexStatus(payload.data);
}

export async function listPrepIndexDocuments(signal?: AbortSignal): Promise<PrepIndexDocument[]> {
  const response = await fetch(getAPIURL("/api/prep/index/documents"), {
    method: "GET",
    headers: { "Content-Type": "application/json" },
    signal,
  });
  if (!response.ok) {
    throw await parseAPIError(response, "加载索引文档失败");
  }
  const payload = (await response.json()) as APIListPayload<APIPrepIndexDocument>;
  const documents = Array.isArray(payload.data) ? payload.data : [];
  return documents.map(normalizeIndexDocument);
}

export async function listPrepIndexChunks(
  options: { documentId?: string; limit?: number } = {},
  signal?: AbortSignal,
): Promise<PrepIndexChunk[]> {
  const query = new URLSearchParams();
  if ((options.documentId || "").trim()) {
    query.set("document_id", (options.documentId || "").trim());
  }
  if (typeof options.limit === "number" && Number.isFinite(options.limit) && options.limit > 0) {
    query.set("limit", String(Math.floor(options.limit)));
  }
  const suffix = query.toString();
  const url = suffix ? `/api/prep/index/chunks?${suffix}` : "/api/prep/index/chunks";

  const response = await fetch(getAPIURL(url), {
    method: "GET",
    headers: { "Content-Type": "application/json" },
    signal,
  });
  if (!response.ok) {
    throw await parseAPIError(response, "加载索引 chunks 失败");
  }
  const payload = (await response.json()) as APIListPayload<APIPrepIndexChunk>;
  const chunks = Array.isArray(payload.data) ? payload.data : [];
  return chunks.map(normalizeIndexChunk);
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

export async function previewPrepRetrieval(input: PrepRetrievalPreviewRequest, signal?: AbortSignal): Promise<PrepRetrievalPreview> {
  const response = await fetch(getAPIURL("/api/prep/retrieval/preview"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      lead_id: (input.leadId || "").trim(),
      query: (input.query || "").trim(),
      top_k: typeof input.topK === "number" && Number.isFinite(input.topK) ? Math.max(1, Math.floor(input.topK)) : undefined,
      include_trace: typeof input.includeTrace === "boolean" ? input.includeTrace : true,
      include_resume: typeof input.includeResume === "boolean" ? input.includeResume : true,
      include_lead_docs: typeof input.includeLeadDocs === "boolean" ? input.includeLeadDocs : true,
    }),
    signal,
  });
  if (!response.ok) {
    throw await parseAPIError(response, "加载检索预览失败");
  }

  const payload = (await response.json()) as APISinglePayload<APIRetrievalPreview>;
  return normalizeRetrievalPreview(payload.data);
}

export async function fetchPrepSession(sessionId: string, signal?: AbortSignal): Promise<PrepSession> {
  const normalizedSessionID = (sessionId || "").trim();
  if (!normalizedSessionID) {
    throw new Error("session_id is required");
  }

  const response = await fetch(getAPIURL(`/api/prep/sessions/${encodeSegment(normalizedSessionID)}`), {
    method: "GET",
    headers: { "Content-Type": "application/json" },
    signal,
  });
  if (!response.ok) {
    throw await parseAPIError(response, "加载备面会话失败");
  }

  const payload = (await response.json()) as APISinglePayload<APIPrepSession>;
  return normalizeSession(payload.data);
}

export async function createPrepSession(input: PrepCreateSessionInput): Promise<PrepSession> {
  const normalizedLeadID = (input.leadId || "").trim();
  if (!normalizedLeadID) {
    throw new Error("lead_id is required");
  }

  const response = await fetch(getAPIURL("/api/prep/sessions"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      lead_id: normalizedLeadID,
      question_count:
        typeof input.questionCount === "number" && Number.isFinite(input.questionCount)
          ? Math.max(1, Math.floor(input.questionCount))
          : DEFAULT_PREP_META.defaultQuestionCount,
      include_resume: Boolean(input.includeResume),
      include_lead_docs: Boolean(input.includeLeadDocs),
    }),
  });
  if (!response.ok) {
    throw await parseAPIError(response, "生成备面题目失败");
  }

  const payload = (await response.json()) as APISinglePayload<APIPrepSession>;
  return normalizeSession(payload.data);
}

export async function createPrepSessionStream(
  input: PrepCreateSessionInput,
  handlers?: {
    onStage?: (event: PrepGenerationProgressEvent) => void;
  },
): Promise<PrepSession> {
  const normalizedLeadID = (input.leadId || "").trim();
  if (!normalizedLeadID) {
    throw new Error("lead_id is required");
  }

  const response = await fetch(getAPIURL("/api/prep/sessions/stream"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "text/event-stream",
    },
    body: JSON.stringify({
      lead_id: normalizedLeadID,
      question_count:
        typeof input.questionCount === "number" && Number.isFinite(input.questionCount)
          ? Math.max(1, Math.floor(input.questionCount))
          : DEFAULT_PREP_META.defaultQuestionCount,
      include_resume: Boolean(input.includeResume),
      include_lead_docs: Boolean(input.includeLeadDocs),
    }),
  });
  if (!response.ok) {
    throw await parseAPIError(response, "流式生成备面题目失败");
  }
  if (!response.body) {
    throw new Error("当前环境不支持流式响应");
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder("utf-8");
  let buffer = "";
  let completedSession: PrepSession | null = null;
  let streamError: Error | null = null;

  for (;;) {
    const { done, value } = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done });
    buffer = buffer.replace(/\r\n/g, "\n");

    let separatorIndex = buffer.indexOf("\n\n");
    while (separatorIndex >= 0) {
      const frame = buffer.slice(0, separatorIndex);
      buffer = buffer.slice(separatorIndex + 2);
      const parsed = parseSSEFrame(frame);
      if (parsed && parsed.data) {
        if (parsed.event === "stage") {
          const rawStage = JSON.parse(parsed.data) as APIPrepStreamStageEvent;
          handlers?.onStage?.({
            stage: (rawStage.stage || "").trim(),
            status: (rawStage.status || "").trim(),
            message: rawStage.message || "",
            delta: rawStage.delta || "",
            trace: normalizeGenerationTrace(rawStage.trace),
          });
        } else if (parsed.event === "completed") {
          const completedPayload = JSON.parse(parsed.data) as { session?: APIPrepSession };
          completedSession = normalizeSession(completedPayload.session);
        } else if (parsed.event === "error") {
          const errorPayload = JSON.parse(parsed.data) as { message?: string; error?: string };
          streamError = new Error((errorPayload.message || errorPayload.error || "流式生成失败").trim());
        }
      }
      separatorIndex = buffer.indexOf("\n\n");
    }

    if (done) {
      break;
    }
  }

  if (streamError) {
    throw streamError;
  }
  if (!completedSession) {
    throw new Error("流式生成结束但未收到会话结果");
  }
  return completedSession;
}

function parseSSEFrame(frame: string): { event: string; data: string } | null {
  const lines = frame.split("\n");
  let eventName = "message";
  const dataLines: string[] = [];
  for (const line of lines) {
    const trimmed = line.trimEnd();
    if (!trimmed || trimmed.startsWith(":")) {
      continue;
    }
    if (trimmed.startsWith("event:")) {
      eventName = trimmed.slice("event:".length).trim();
      continue;
    }
    if (trimmed.startsWith("data:")) {
      dataLines.push(trimmed.slice("data:".length).trimStart());
    }
  }
  if (dataLines.length === 0) {
    return null;
  }
  return {
    event: eventName,
    data: dataLines.join("\n"),
  };
}

export async function savePrepDraftAnswers(sessionId: string, answers: PrepAnswer[]): Promise<PrepDraftAnswersSaveResult> {
  const normalizedSessionID = (sessionId || "").trim();
  if (!normalizedSessionID) {
    throw new Error("session_id is required");
  }

  const normalizedAnswers = answers
    .map((item) => ({
      question_id: Number.isFinite(item.questionId) ? Math.floor(item.questionId) : 0,
      answer: item.answer || "",
    }))
    .filter((item) => item.question_id > 0);

  const response = await fetch(getAPIURL(`/api/prep/sessions/${encodeSegment(normalizedSessionID)}/draft-answers`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ answers: normalizedAnswers }),
  });
  if (!response.ok) {
    throw await parseAPIError(response, "保存答案草稿失败");
  }

  const payload = (await response.json()) as APISinglePayload<APIPrepDraftAnswersSaveResult>;
  return normalizeDraftAnswersSaveResult(payload.data);
}
