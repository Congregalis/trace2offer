export type PrepScope = "topics" | "companies" | "leads";

export interface PrepMeta {
  enabled: boolean;
  defaultQuestionCount: number;
  supportedScopes: PrepScope[];
}

export const DEFAULT_PREP_SCOPES: PrepScope[] = ["topics", "companies", "leads"];

export const DEFAULT_PREP_META: PrepMeta = {
  enabled: false,
  defaultQuestionCount: 8,
  supportedScopes: [...DEFAULT_PREP_SCOPES],
};
