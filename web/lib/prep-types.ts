export type PrepScope = "topics";

export interface PrepMeta {
  enabled: boolean;
  defaultQuestionCount: number;
  supportedScopes: PrepScope[];
}

export interface PrepTopic {
  key: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface PrepTopicCreateInput {
  key: string;
  name: string;
  description: string;
}

export interface PrepTopicPatchInput {
  name?: string;
  description?: string;
}

export interface PrepKnowledgeDocument {
  scope: PrepScope;
  scopeId: string;
  filename: string;
  content: string;
  updatedAt: string;
}

export interface PrepKnowledgeDocumentCreateInput {
  filename: string;
  content: string;
}

export interface PrepKnowledgeDocumentUpdateInput {
  content: string;
}

export interface PrepContextSource {
  scope: string;
  kind: string;
  title: string;
}

export interface PrepLeadContextPreview {
  leadId: string;
  company: string;
  position: string;
  hasResume: boolean;
  hasProfile: boolean;
  topicKeys: string[];
  sources: PrepContextSource[];
}

export interface PrepIndexRunError {
  source: string;
  message: string;
}

export interface PrepIndexRunSummary {
  runId: string;
  mode: "incremental" | "full" | string;
  startedAt: string;
  completedAt: string;
  status: string;
  documentsScanned: number;
  documentsIndexed: number;
  documentsSkipped: number;
  documentsDeleted: number;
  chunksCreated: number;
  chunksUpdated: number;
  errors: PrepIndexRunError[];
}

export interface PrepIndexStatus {
  embeddingProvider: string;
  embeddingModel: string;
  documentCount: number;
  chunkCount: number;
  lastIndexedAt: string;
  lastIndexStatus: string;
}

export interface PrepIndexDocument {
  id: string;
  scope: PrepScope;
  scopeId: string;
  kind: string;
  title: string;
  sourcePath: string;
  contentHash: string;
  updatedAt: string;
}

export interface PrepIndexChunk {
  id: string;
  documentId: string;
  scope: PrepScope;
  scopeId: string;
  documentTitle: string;
  chunkIndex: number;
  content: string;
  tokenCount: number;
  updatedAt: string;
}

export interface PrepIndexRebuildInput {
  scope: PrepScope | "*";
  scopeId?: string;
  mode?: "incremental" | "full";
}

export interface PrepRetrievalPreviewRequest {
  leadId?: string;
  query: string;
  topicKeys?: string[];
  topK?: number;
  includeTrace?: boolean;
  includeResume?: boolean;
  includeProfile?: boolean;
  includeLeadDocs?: boolean;
}

export interface PrepRetrievalFilters {
  scope: PrepScope[];
  topicKeys: string[];
}

export interface PrepRetrievalTrace {
  stageQueryNormalization: PrepTraceStage;
  stageInitialRetrieval: PrepTraceStage;
  stageDeduplication: PrepTraceStage;
  stageReranking: PrepTraceStage;
}

export interface PrepTraceStage {
  input?: string;
  output?: string;
  method: string;
  inputCount?: number;
  outputCount?: number;
  metadata?: Record<string, unknown>;
}

export interface PrepRetrievedChunk {
  id: string;
  content: string;
  score: number;
  source: {
    scope: PrepScope;
    scopeId: string;
    documentTitle: string;
    chunkIndex: number;
  };
  whySelected: string;
}

export interface PrepRetrievalFinalContext {
  totalTokens: number;
  chunksUsed: number;
  context: string;
}

export interface PrepRetrievalPreview {
  query: string;
  normalizedQuery: string;
  filters: PrepRetrievalFilters;
  trace?: PrepRetrievalTrace;
  candidateChunks: PrepRetrievedChunk[];
  retrievedChunks: PrepRetrievedChunk[];
  finalContext: PrepRetrievalFinalContext;
}

export interface PrepQuestion {
  id: number;
  type: string;
  content: string;
  expectedPoints: string[];
  contextSources: string[];
}

export interface PrepAnswer {
  questionId: number;
  answer: string;
  submittedAt?: string;
}

export interface PrepSessionConfig {
  topicKeys: string[];
  questionCount: number;
  includeResume: boolean;
  includeProfile: boolean;
  includeLeadDocs: boolean;
}

export interface PrepGenerationTrace {
  inputSnapshot: {
    leadId: string;
    topicKeys: string[];
    questionCount: number;
  };
  retrievalQuery: string;
  retrievalResults: {
    candidatesFound: number;
    finalSelected: number;
    sources: string[];
  };
  promptSections: Array<{
    title: string;
    content: string;
  }> & {
    systemPrompt?: string;
    contextSection?: string;
    taskInstruction?: string;
  };
  generationResult: {
    questionsGenerated: number;
    generationTimeMs: number;
    model: string;
  };
}

export interface PrepSession {
  id: string;
  leadId: string;
  company: string;
  position: string;
  status: string;
  config: PrepSessionConfig;
  sources: PrepContextSource[];
  questions: PrepQuestion[];
  answers: PrepAnswer[];
  evaluation?: unknown;
  referenceAnswers: Record<string, unknown>;
  generationTrace?: PrepGenerationTrace;
  createdAt: string;
  updatedAt: string;
}

export interface PrepCreateSessionInput {
  leadId: string;
  topicKeys: string[];
  questionCount: number;
  includeResume: boolean;
  includeProfile: boolean;
  includeLeadDocs: boolean;
}

export interface PrepDraftAnswersSaveResult {
  sessionId: string;
  savedAt: string;
  answersCount: number;
}

export const DEFAULT_PREP_SCOPES: PrepScope[] = ["topics"];

export const DEFAULT_PREP_META: PrepMeta = {
  enabled: false,
  defaultQuestionCount: 8,
  supportedScopes: [...DEFAULT_PREP_SCOPES],
};
