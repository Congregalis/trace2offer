"use client";

import { type ComponentType, useEffect, useMemo, useState } from "react";
import { useLeadsStore } from "@/lib/leads-store";
import { Lead, LeadMutationInput, LeadStatus, ReminderMethod, STATUS_CONFIG } from "@/lib/types";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { FieldGroup, Field, FieldLabel } from "@/components/ui/field";
import {
  Archive,
  Ban,
  CircleX,
  ExternalLink,
  MessageCircle,
  MoreHorizontal,
  Pencil,
  Plus,
  Search,
  Send,
  Sparkles,
  Trash2,
  Trophy,
  Wrench,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

const EMPTY_NEW_LEAD: LeadMutationInput = {
  company: "",
  position: "",
  status: "new",
  source: "",
  priority: 0,
  nextAction: "",
  nextActionAt: "",
  interviewAt: "",
  reminderMethods: ["in_app"],
  notes: "",
  companyWebsiteUrl: "",
  jdUrl: "",
  location: "",
};

const REMINDER_METHOD_OPTIONS: Array<{ value: ReminderMethod; label: string; description: string }> = [
  { value: "in_app", label: "页面内提醒", description: "在页面内提示待跟进事项" },
  { value: "email", label: "邮件提醒", description: "保留邮件提醒渠道（后续可接入 SMTP）" },
  { value: "web_push", label: "Web Push/系统通知", description: "通过浏览器系统通知弹窗提醒" },
];

function toMutationInput(lead: Lead): LeadMutationInput {
  return {
    company: lead.company,
    position: lead.position,
    source: lead.source,
    status: lead.status,
    priority: lead.priority,
    nextAction: lead.nextAction,
    nextActionAt: lead.nextActionAt,
    interviewAt: lead.interviewAt,
    reminderMethods: [...lead.reminderMethods],
    notes: lead.notes,
    companyWebsiteUrl: lead.companyWebsiteUrl,
    jdUrl: lead.jdUrl,
    location: lead.location,
  };
}

function toDateTimeLocalValue(value: string): string {
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return "";
  }
  const date = new Date(timestamp);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hour = String(date.getHours()).padStart(2, "0");
  const minute = String(date.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hour}:${minute}`;
}

function fromDateTimeLocalValue(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }
  const parsed = new Date(trimmed);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }
  return parsed.toISOString();
}

function toggleReminderMethod(methods: ReminderMethod[], method: ReminderMethod): ReminderMethod[] {
  const hasMethod = methods.includes(method);
  if (hasMethod) {
    const next = methods.filter((item) => item !== method);
    return next.length > 0 ? next : ["in_app"];
  }
  return [...methods, method];
}

function formatDate(dateString: string) {
  if (!dateString) {
    return "-";
  }

  const parsed = Date.parse(dateString);
  if (!Number.isNaN(parsed)) {
    const date = new Date(parsed);
    return `${date.getMonth() + 1}月${date.getDate()}日`;
  }

  const parts = dateString.split("-");
  if (parts.length === 3) {
    const month = parseInt(parts[1], 10);
    const day = parseInt(parts[2], 10);
    if (!Number.isNaN(month) && !Number.isNaN(day)) {
      return `${month}月${day}日`;
    }
  }

  return dateString;
}

const STATUS_VISUAL_CONFIG: Record<
  LeadStatus,
  {
    className: string;
    icon: ComponentType<{ className?: string }>;
    label?: string;
    dotPingClassName: string;
    dotClassName: string;
  }
> = {
  new: {
    className:
      "bg-gradient-to-r from-fuchsia-500/18 via-pink-500/10 to-fuchsia-500/18 text-fuchsia-700 ring-1 ring-inset ring-fuchsia-500/18 shadow-sm shadow-fuchsia-500/15 dark:from-fuchsia-500/40 dark:via-pink-500/15 dark:to-fuchsia-500/40 dark:text-fuchsia-300 dark:ring-fuchsia-400/10 dark:shadow-fuchsia-500/30",
    icon: Sparkles,
    label: "NEW",
    dotPingClassName: "bg-fuchsia-400/80",
    dotClassName: "bg-fuchsia-300",
  },
  preparing: {
    className:
      "bg-gradient-to-r from-cyan-500/18 via-sky-500/10 to-cyan-500/18 text-cyan-700 ring-1 ring-inset ring-cyan-500/18 shadow-sm shadow-cyan-500/15 dark:from-cyan-500/40 dark:via-sky-500/15 dark:to-cyan-500/40 dark:text-cyan-300 dark:ring-cyan-400/10 dark:shadow-cyan-500/30",
    icon: Wrench,
    dotPingClassName: "bg-cyan-400/80",
    dotClassName: "bg-cyan-300",
  },
  applied: {
    className:
      "bg-gradient-to-r from-warning/28 via-chart-5/12 to-warning/28 text-amber-700 ring-1 ring-inset ring-warning/20 shadow-sm shadow-warning/15 dark:from-warning/45 dark:via-chart-5/20 dark:to-warning/45 dark:text-warning dark:ring-warning/10 dark:shadow-warning/30",
    icon: Send,
    dotPingClassName: "bg-warning/70",
    dotClassName: "bg-warning",
  },
  interviewing: {
    className:
      "bg-gradient-to-r from-indigo-500/18 via-blue-500/10 to-indigo-500/18 text-indigo-700 ring-1 ring-inset ring-indigo-500/18 shadow-sm shadow-indigo-500/15 dark:from-indigo-500/40 dark:via-blue-500/15 dark:to-indigo-500/40 dark:text-indigo-300 dark:ring-indigo-400/10 dark:shadow-indigo-500/30",
    icon: MessageCircle,
    dotPingClassName: "bg-indigo-400/80",
    dotClassName: "bg-indigo-300",
  },
  offered: {
    className:
      "bg-gradient-to-r from-success/30 via-warning/14 to-success/30 text-emerald-700 ring-1 ring-inset ring-success/20 shadow-sm shadow-success/20 dark:from-success/55 dark:via-warning/20 dark:to-success/55 dark:text-success dark:ring-success/10 dark:shadow-success/45",
    icon: Trophy,
    label: "OFFER",
    dotPingClassName: "bg-success/80",
    dotClassName: "bg-success",
  },
  declined: {
    className:
      "bg-gradient-to-r from-chart-4/18 via-chart-4/10 to-chart-4/18 text-violet-700 ring-1 ring-inset ring-chart-4/16 shadow-sm shadow-chart-4/14 dark:from-chart-4/35 dark:via-chart-4/15 dark:to-chart-4/35 dark:text-chart-4 dark:ring-chart-4/10 dark:shadow-chart-4/20",
    icon: Ban,
    dotPingClassName: "bg-chart-4/70",
    dotClassName: "bg-chart-4",
  },
  rejected: {
    className:
      "bg-gradient-to-r from-destructive/18 via-destructive/10 to-destructive/18 text-rose-700 ring-1 ring-inset ring-destructive/16 shadow-sm shadow-destructive/14 dark:from-destructive/35 dark:via-destructive/15 dark:to-destructive/35 dark:text-destructive dark:ring-destructive/10 dark:shadow-destructive/20",
    icon: CircleX,
    dotPingClassName: "bg-destructive/70",
    dotClassName: "bg-destructive",
  },
  archived: {
    className:
      "bg-gradient-to-r from-muted/70 via-muted/40 to-muted/70 text-muted-foreground ring-1 ring-inset ring-border",
    icon: Archive,
    dotPingClassName: "bg-muted-foreground/40",
    dotClassName: "bg-muted-foreground",
  },
};

function StatusBadge({ status, className }: { status: LeadStatus; className?: string }) {
  const config = STATUS_CONFIG[status];
  const visual = STATUS_VISUAL_CONFIG[status];
  const Icon = visual.icon;
  const label = visual.label ?? config.label;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold tracking-[0.08em]",
        visual.className,
        className
      )}
    >
      <Icon className={cn("h-3 w-3 shrink-0", status === "new" || status === "offered" ? "animate-pulse" : "")} />
      <span>{label}</span>
      <span className="relative flex h-1.5 w-1.5">
        <span className={cn("absolute inline-flex h-full w-full rounded-full animate-ping", visual.dotPingClassName)} />
        <span className={cn("relative inline-flex h-1.5 w-1.5 rounded-full", visual.dotClassName)} />
      </span>
    </span>
  );
}

export function LeadsTable() {
  const { leads, isLoading, isSyncing, hasLoaded, fetchLeads, updateLead, deleteLead, addLead } = useLeadsStore();
  const [search, setSearch] = useState("");
  const [editingLead, setEditingLead] = useState<Lead | null>(null);
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [newLead, setNewLead] = useState<LeadMutationInput>(EMPTY_NEW_LEAD);

  useEffect(() => {
    if (hasLoaded) {
      return;
    }

    void fetchLeads().catch((error) => {
      const message = error instanceof Error && error.message ? error.message : "加载线索失败";
      toast.error(message);
    });
  }, [fetchLeads, hasLoaded]);

  const filteredLeads = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    if (!keyword) {
      return leads;
    }

    return leads.filter((lead) => {
      return (
        lead.company.toLowerCase().includes(keyword) ||
        lead.position.toLowerCase().includes(keyword) ||
        lead.source.toLowerCase().includes(keyword) ||
        lead.location.toLowerCase().includes(keyword)
      );
    });
  }, [leads, search]);

  const handleStatusChange = async (id: string, status: LeadStatus) => {
    if (isSyncing) {
      return;
    }
    const current = leads.find((item) => item.id === id);
    if (!current) {
      toast.error("线索不存在，无法更新状态");
      return;
    }

    try {
      await updateLead(id, {
        ...toMutationInput(current),
        status,
      });
      toast.success("状态已更新");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "状态更新失败";
      toast.error(message);
    }
  };

  const handleDelete = async (id: string) => {
    if (isSyncing) {
      return;
    }

    try {
      await deleteLead(id);
      toast.success("线索已删除");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "删除线索失败";
      toast.error(message);
    }
  };

  const handleSaveEdit = async () => {
    if (!editingLead || isSyncing) {
      return;
    }

    if (!editingLead.company.trim() || !editingLead.position.trim()) {
      toast.error("请填写公司和职位");
      return;
    }

    try {
      await updateLead(editingLead.id, toMutationInput(editingLead));
      setEditingLead(null);
      toast.success("线索已更新");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "更新线索失败";
      toast.error(message);
    }
  };

  const handleAddLead = async () => {
    if (isSyncing) {
      return;
    }

    if (!newLead.company.trim() || !newLead.position.trim()) {
      toast.error("请填写公司和职位");
      return;
    }

    try {
      await addLead(newLead);
      setNewLead(EMPTY_NEW_LEAD);
      setIsAddDialogOpen(false);
      toast.success("线索已添加");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "添加线索失败";
      toast.error(message);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="搜索公司、职位、来源、地点..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 bg-secondary border-border"
          />
        </div>
        <Button onClick={() => setIsAddDialogOpen(true)} size="sm" disabled={isSyncing}>
          <Plus className="w-4 h-4 mr-1" />
          添加线索
        </Button>
      </div>

      <div className="border border-border rounded-lg overflow-hidden bg-card">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent border-border">
              <TableHead className="text-muted-foreground">公司</TableHead>
              <TableHead className="text-muted-foreground">职位</TableHead>
              <TableHead className="text-muted-foreground">状态</TableHead>
              <TableHead className="text-muted-foreground">来源</TableHead>
              <TableHead className="text-muted-foreground">工作地点</TableHead>
              <TableHead className="text-muted-foreground">链接</TableHead>
              <TableHead className="text-muted-foreground">更新时间</TableHead>
              <TableHead className="text-muted-foreground w-12"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && filteredLeads.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="text-center text-muted-foreground py-8">
                  正在加载线索...
                </TableCell>
              </TableRow>
            ) : null}

            {filteredLeads.map((lead) => (
              <TableRow key={lead.id} className="border-border">
                <TableCell className="font-medium">{lead.company}</TableCell>
                <TableCell>{lead.position}</TableCell>
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button className="focus:outline-none" disabled={isSyncing}>
                        <StatusBadge
                          status={lead.status}
                          className="cursor-pointer transition-opacity hover:opacity-80"
                        />
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="start">
                      {Object.entries(STATUS_CONFIG).map(([status]) => (
                        <DropdownMenuItem
                          key={status}
                          onClick={() => handleStatusChange(lead.id, status as LeadStatus)}
                        >
                          <StatusBadge status={status as LeadStatus} className="mr-2" />
                        </DropdownMenuItem>
                      ))}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
                <TableCell className="text-muted-foreground">{lead.source || "-"}</TableCell>
                <TableCell className="text-muted-foreground">{lead.location || "-"}</TableCell>
                <TableCell className="text-muted-foreground">
                  <div className="flex items-center gap-3">
                    {lead.companyWebsiteUrl ? (
                      <a
                        href={lead.companyWebsiteUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 hover:text-foreground"
                      >
                        官网
                        <ExternalLink className="w-3 h-3" />
                      </a>
                    ) : null}
                    {lead.jdUrl ? (
                      <a
                        href={lead.jdUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 hover:text-foreground"
                      >
                        JD
                        <ExternalLink className="w-3 h-3" />
                      </a>
                    ) : null}
                    {!lead.companyWebsiteUrl && !lead.jdUrl ? "-" : null}
                  </div>
                </TableCell>
                <TableCell className="text-muted-foreground">{formatDate(lead.updatedAt)}</TableCell>
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-8 w-8" disabled={isSyncing}>
                        <MoreHorizontal className="w-4 h-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => setEditingLead(lead)}>
                        <Pencil className="w-4 h-4 mr-2" />
                        编辑
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        onClick={() => handleDelete(lead.id)}
                        className="text-destructive focus:text-destructive"
                      >
                        <Trash2 className="w-4 h-4 mr-2" />
                        删除
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            ))}
            {!isLoading && filteredLeads.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="text-center text-muted-foreground py-8">
                  暂无线索，点击「添加线索」开始追踪
                </TableCell>
              </TableRow>
            ) : null}
          </TableBody>
        </Table>
      </div>

      <Dialog
        open={!!editingLead}
        onOpenChange={(open) => {
          if (!open) {
            setEditingLead(null);
          }
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>编辑线索</DialogTitle>
            <DialogDescription>修改职位线索的详细信息</DialogDescription>
          </DialogHeader>
          {editingLead ? (
            <FieldGroup>
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel>公司</FieldLabel>
                  <Input
                    value={editingLead.company}
                    onChange={(e) => setEditingLead({ ...editingLead, company: e.target.value })}
                  />
                </Field>
                <Field>
                  <FieldLabel>职位</FieldLabel>
                  <Input
                    value={editingLead.position}
                    onChange={(e) => setEditingLead({ ...editingLead, position: e.target.value })}
                  />
                </Field>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel>状态</FieldLabel>
                  <Select
                    value={editingLead.status}
                    onValueChange={(value) => setEditingLead({ ...editingLead, status: value as LeadStatus })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {Object.entries(STATUS_CONFIG).map(([status, config]) => (
                        <SelectItem key={status} value={status}>
                          {config.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
                <Field>
                  <FieldLabel>来源</FieldLabel>
                  <Input
                    value={editingLead.source}
                    onChange={(e) => setEditingLead({ ...editingLead, source: e.target.value })}
                  />
                </Field>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel>工作地点</FieldLabel>
                  <Input
                    value={editingLead.location}
                    onChange={(e) => setEditingLead({ ...editingLead, location: e.target.value })}
                    placeholder="如: 北京 / 远程"
                  />
                </Field>
                <Field>
                  <FieldLabel>公司官网链接</FieldLabel>
                  <Input
                    value={editingLead.companyWebsiteUrl}
                    onChange={(e) =>
                      setEditingLead({
                        ...editingLead,
                        companyWebsiteUrl: e.target.value,
                      })
                    }
                    placeholder="https://company.com"
                  />
                </Field>
              </div>
              <Field>
                <FieldLabel>JD链接</FieldLabel>
                <Input
                  value={editingLead.jdUrl}
                  onChange={(e) => setEditingLead({ ...editingLead, jdUrl: e.target.value })}
                  placeholder="https://..."
                />
              </Field>
              <div className="grid grid-cols-3 gap-4">
                <Field>
                  <FieldLabel>下一步动作</FieldLabel>
                  <Input
                    value={editingLead.nextAction}
                    onChange={(e) => setEditingLead({ ...editingLead, nextAction: e.target.value })}
                    placeholder="如：周三前发跟进邮件"
                  />
                </Field>
                <Field>
                  <FieldLabel>下一步动作时间</FieldLabel>
                  <Input
                    type="datetime-local"
                    value={toDateTimeLocalValue(editingLead.nextActionAt)}
                    onChange={(e) =>
                      setEditingLead({
                        ...editingLead,
                        nextActionAt: fromDateTimeLocalValue(e.target.value),
                      })
                    }
                  />
                </Field>
                <Field>
                  <FieldLabel>面试时间</FieldLabel>
                  <Input
                    type="datetime-local"
                    value={toDateTimeLocalValue(editingLead.interviewAt)}
                    onChange={(e) =>
                      setEditingLead({
                        ...editingLead,
                        interviewAt: fromDateTimeLocalValue(e.target.value),
                      })
                    }
                  />
                </Field>
              </div>
              <Field>
                <FieldLabel>提醒方式</FieldLabel>
                <div className="space-y-2 rounded-md border border-border bg-secondary/30 p-3">
                  {REMINDER_METHOD_OPTIONS.map((option) => (
                    <label key={option.value} className="flex items-start gap-2 text-sm">
                      <Checkbox
                        checked={editingLead.reminderMethods.includes(option.value)}
                        onCheckedChange={() =>
                          setEditingLead({
                            ...editingLead,
                            reminderMethods: toggleReminderMethod(editingLead.reminderMethods, option.value),
                          })
                        }
                      />
                      <span className="space-y-0.5">
                        <span className="block text-foreground">{option.label}</span>
                        <span className="block text-xs text-muted-foreground">{option.description}</span>
                      </span>
                    </label>
                  ))}
                </div>
              </Field>
              <Field>
                <FieldLabel>备注</FieldLabel>
                <Textarea
                  value={editingLead.notes}
                  onChange={(e) => setEditingLead({ ...editingLead, notes: e.target.value })}
                  rows={3}
                />
              </Field>
              <div className="flex justify-end gap-2 pt-4">
                <Button variant="outline" onClick={() => setEditingLead(null)} disabled={isSyncing}>
                  取消
                </Button>
                <Button onClick={handleSaveEdit} disabled={isSyncing}>
                  保存
                </Button>
              </div>
            </FieldGroup>
          ) : null}
        </DialogContent>
      </Dialog>

      <Dialog open={isAddDialogOpen} onOpenChange={setIsAddDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>添加新线索</DialogTitle>
            <DialogDescription>填写新职位线索的基本信息</DialogDescription>
          </DialogHeader>
          <FieldGroup>
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>公司 *</FieldLabel>
                <Input
                  value={newLead.company}
                  onChange={(e) => setNewLead({ ...newLead, company: e.target.value })}
                  placeholder="公司名称"
                />
              </Field>
              <Field>
                <FieldLabel>职位 *</FieldLabel>
                <Input
                  value={newLead.position}
                  onChange={(e) => setNewLead({ ...newLead, position: e.target.value })}
                  placeholder="职位名称"
                />
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>状态</FieldLabel>
                <Select
                  value={newLead.status}
                  onValueChange={(value) => setNewLead({ ...newLead, status: value as LeadStatus })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.entries(STATUS_CONFIG).map(([status, config]) => (
                      <SelectItem key={status} value={status}>
                        {config.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
              <Field>
                <FieldLabel>来源</FieldLabel>
                <Input
                  value={newLead.source}
                  onChange={(e) => setNewLead({ ...newLead, source: e.target.value })}
                  placeholder="如: Boss直聘 / 官网"
                />
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>工作地点</FieldLabel>
                <Input
                  value={newLead.location}
                  onChange={(e) => setNewLead({ ...newLead, location: e.target.value })}
                  placeholder="如: 北京 / 远程"
                />
              </Field>
              <Field>
                <FieldLabel>公司官网链接</FieldLabel>
                <Input
                  value={newLead.companyWebsiteUrl}
                  onChange={(e) => setNewLead({ ...newLead, companyWebsiteUrl: e.target.value })}
                  placeholder="https://company.com"
                />
              </Field>
            </div>
            <Field>
              <FieldLabel>JD链接</FieldLabel>
              <Input
                value={newLead.jdUrl}
                onChange={(e) => setNewLead({ ...newLead, jdUrl: e.target.value })}
                placeholder="https://..."
              />
            </Field>
            <div className="grid grid-cols-3 gap-4">
              <Field>
                <FieldLabel>下一步动作</FieldLabel>
                <Input
                  value={newLead.nextAction}
                  onChange={(e) => setNewLead({ ...newLead, nextAction: e.target.value })}
                  placeholder="如：周三前发跟进邮件"
                />
              </Field>
              <Field>
                <FieldLabel>下一步动作时间</FieldLabel>
                <Input
                  type="datetime-local"
                  value={toDateTimeLocalValue(newLead.nextActionAt)}
                  onChange={(e) => setNewLead({ ...newLead, nextActionAt: fromDateTimeLocalValue(e.target.value) })}
                />
              </Field>
              <Field>
                <FieldLabel>面试时间</FieldLabel>
                <Input
                  type="datetime-local"
                  value={toDateTimeLocalValue(newLead.interviewAt)}
                  onChange={(e) => setNewLead({ ...newLead, interviewAt: fromDateTimeLocalValue(e.target.value) })}
                />
              </Field>
            </div>
            <Field>
              <FieldLabel>提醒方式</FieldLabel>
              <div className="space-y-2 rounded-md border border-border bg-secondary/30 p-3">
                {REMINDER_METHOD_OPTIONS.map((option) => (
                  <label key={option.value} className="flex items-start gap-2 text-sm">
                    <Checkbox
                      checked={newLead.reminderMethods.includes(option.value)}
                      onCheckedChange={() =>
                        setNewLead({
                          ...newLead,
                          reminderMethods: toggleReminderMethod(newLead.reminderMethods, option.value),
                        })
                      }
                    />
                    <span className="space-y-0.5">
                      <span className="block text-foreground">{option.label}</span>
                      <span className="block text-xs text-muted-foreground">{option.description}</span>
                    </span>
                  </label>
                ))}
              </div>
            </Field>
            <Field>
              <FieldLabel>备注</FieldLabel>
              <Textarea
                value={newLead.notes}
                onChange={(e) => setNewLead({ ...newLead, notes: e.target.value })}
                rows={3}
                placeholder="添加备注..."
              />
            </Field>
            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={() => setIsAddDialogOpen(false)} disabled={isSyncing}>
                取消
              </Button>
              <Button onClick={handleAddLead} disabled={isSyncing}>
                添加
              </Button>
            </div>
          </FieldGroup>
        </DialogContent>
      </Dialog>
    </div>
  );
}
