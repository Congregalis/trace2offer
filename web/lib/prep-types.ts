export type PrepScope = "topics" | "companies" | "leads";

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

export const DEFAULT_PREP_SCOPES: PrepScope[] = ["topics", "companies", "leads"];

export const DEFAULT_PREP_META: PrepMeta = {
  enabled: false,
  defaultQuestionCount: 8,
  supportedScopes: [...DEFAULT_PREP_SCOPES],
};
