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
  location: string;
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
