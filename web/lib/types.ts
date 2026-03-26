export type LeadStatus =
  | "new" // 新线索
  | "preparing" // 准备中
  | "applied" // 已投递
  | "interviewing" // 面试中
  | "offered" // 已获offer
  | "declined" // 已拒绝（主动拒绝）
  | "rejected" // 被拒绝（对方拒绝）
  | "archived"; // 已归档

export type ReminderMethod = "in_app" | "email" | "web_push";
export type CandidateStatus = "pending_review" | "shortlisted" | "dismissed" | "promoted";

export interface Lead {
  id: string;
  company: string;
  position: string;
  source: string;
  status: LeadStatus;
  priority: number;
  nextAction: string;
  nextActionAt: string;
  interviewAt: string;
  reminderMethods: ReminderMethod[];
  notes: string;
  companyWebsiteUrl: string;
  jdUrl: string;
  jdText: string;
  location: string;
  createdAt: string;
  updatedAt: string;
}

export interface LeadMutationInput {
  company: string;
  position: string;
  source: string;
  status: LeadStatus;
  priority: number;
  nextAction: string;
  nextActionAt: string;
  interviewAt: string;
  reminderMethods: ReminderMethod[];
  notes: string;
  companyWebsiteUrl: string;
  jdUrl: string;
  jdText: string;
  location: string;
}

export interface Candidate {
  id: string;
  company: string;
  position: string;
  source: string;
  location: string;
  jdUrl: string;
  companyWebsiteUrl: string;
  status: CandidateStatus;
  matchScore: number;
  matchReasons: string[];
  recommendationNotes: string;
  notes: string;
  promotedLeadId: string;
  createdAt: string;
  updatedAt: string;
}

export interface CandidateMutationInput {
  company: string;
  position: string;
  source: string;
  location: string;
  jdUrl: string;
  companyWebsiteUrl: string;
  status: CandidateStatus;
  matchScore: number;
  matchReasons: string[];
  recommendationNotes: string;
  notes: string;
}

export interface CandidatePromoteInput {
  source: string;
  status: LeadStatus;
  priority: number;
  nextAction: string;
  nextActionAt: string;
  interviewAt: string;
  reminderMethods: ReminderMethod[];
  notes: string;
}

export interface DiscoveryRule {
  id: string;
  name: string;
  feedUrl: string;
  source: string;
  defaultLocation: string;
  includeKeywords: string[];
  excludeKeywords: string[];
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface DiscoveryRuleMutationInput {
  name: string;
  feedUrl: string;
  source: string;
  defaultLocation: string;
  includeKeywords: string[];
  excludeKeywords: string[];
  enabled: boolean;
}

export interface DiscoveryRunResult {
  ranAt: string;
  rulesTotal: number;
  rulesExecuted: number;
  entriesFetched: number;
  candidatesCreated: number;
  candidatesUpdated: number;
  errors: string[];
}

export interface LeadTimeline {
  leadId: string;
  stages: LeadTimelineStage[];
  updatedAt: string;
}

export interface LeadTimelineStage {
  stage: string;
  startedAt: string;
  endedAt: string;
}

export interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  createdAt: string;
}

export const STATUS_CONFIG: Record<LeadStatus, { label: string; color: string }> = {
  new: { label: "新线索", color: "bg-warning/20 text-warning" },
  preparing: { label: "准备中", color: "bg-info/20 text-info" },
  applied: { label: "已投递", color: "bg-warning/20 text-warning" },
  interviewing: { label: "面试中", color: "bg-chart-2/20 text-chart-2" },
  offered: { label: "已获offer", color: "bg-success/20 text-success" },
  declined: { label: "已拒绝", color: "bg-chart-4/20 text-chart-4" },
  rejected: { label: "被拒绝", color: "bg-destructive/20 text-destructive" },
  archived: { label: "已归档", color: "bg-muted text-muted-foreground" },
};

export const CANDIDATE_STATUS_CONFIG: Record<CandidateStatus, { label: string; color: string }> = {
  pending_review: { label: "待评估", color: "bg-warning/20 text-warning" },
  shortlisted: { label: "已入围", color: "bg-chart-2/20 text-chart-2" },
  dismissed: { label: "已忽略", color: "bg-muted text-muted-foreground" },
  promoted: { label: "已转线索", color: "bg-success/20 text-success" },
};
