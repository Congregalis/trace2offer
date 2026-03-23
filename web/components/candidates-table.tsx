"use client";

import { useEffect, useMemo, useState } from "react";
import { useCandidatesStore } from "@/lib/candidates-store";
import { useLeadsStore } from "@/lib/leads-store";
import { Candidate, CandidateMutationInput, CandidateStatus } from "@/lib/types";
import { CANDIDATE_STATUS_CONFIG } from "@/lib/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
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
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { MoreHorizontal, Plus, Search, Sparkles, Trash2, ArrowRightCircle, Pencil } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

const EMPTY_CANDIDATE: CandidateMutationInput = {
  company: "",
  position: "",
  source: "",
  location: "",
  jdUrl: "",
  companyWebsiteUrl: "",
  status: "pending_review",
  matchScore: 0,
  matchReasons: [],
  recommendationNotes: "",
  notes: "",
};

const MANUAL_STATUS_OPTIONS: CandidateStatus[] = ["pending_review", "shortlisted", "dismissed"];

function toMutationInput(candidate: Candidate): CandidateMutationInput {
  return {
    company: candidate.company,
    position: candidate.position,
    source: candidate.source,
    location: candidate.location,
    jdUrl: candidate.jdUrl,
    companyWebsiteUrl: candidate.companyWebsiteUrl,
    status: candidate.status,
    matchScore: candidate.matchScore,
    matchReasons: [...candidate.matchReasons],
    recommendationNotes: candidate.recommendationNotes,
    notes: candidate.notes,
  };
}

function parseReasons(value: string): string[] {
  return value
    .split(/\r?\n|,|，|;|；/g)
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatDateTime(value: string): string {
  if (!value) {
    return "-";
  }
  const ts = Date.parse(value);
  if (Number.isNaN(ts)) {
    return value;
  }
  const date = new Date(ts);
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, "0")}:${String(
    date.getMinutes()
  ).padStart(2, "0")}`;
}

function CandidateStatusBadge({ status }: { status: CandidateStatus }) {
  const config = CANDIDATE_STATUS_CONFIG[status];
  return (
    <span className={cn("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium", config.color)}>
      {config.label}
    </span>
  );
}

export function CandidatesTable() {
  const { fetchLeads } = useLeadsStore();
  const { candidates, isLoading, isSyncing, hasLoaded, fetchCandidates, addCandidate, updateCandidate, deleteCandidate, promoteCandidate } =
    useCandidatesStore();

  const [search, setSearch] = useState("");
  const [isAddOpen, setIsAddOpen] = useState(false);
  const [newCandidate, setNewCandidate] = useState<CandidateMutationInput>(EMPTY_CANDIDATE);
  const [newReasonsInput, setNewReasonsInput] = useState("");
  const [editingCandidate, setEditingCandidate] = useState<Candidate | null>(null);
  const [editReasonsInput, setEditReasonsInput] = useState("");

  useEffect(() => {
    if (hasLoaded) {
      return;
    }
    void fetchCandidates().catch((error) => {
      const message = error instanceof Error && error.message ? error.message : "加载候选职位失败";
      toast.error(message);
    });
  }, [fetchCandidates, hasLoaded]);

  const filtered = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    if (!keyword) {
      return candidates;
    }
    return candidates.filter((candidate) => {
      return (
        candidate.company.toLowerCase().includes(keyword) ||
        candidate.position.toLowerCase().includes(keyword) ||
        candidate.source.toLowerCase().includes(keyword) ||
        candidate.location.toLowerCase().includes(keyword)
      );
    });
  }, [candidates, search]);

  const handleCreate = async () => {
    if (isSyncing) {
      return;
    }
    try {
      await addCandidate({
        ...newCandidate,
        matchReasons: parseReasons(newReasonsInput),
      });
      toast.success("候选职位已添加");
      setIsAddOpen(false);
      setNewCandidate(EMPTY_CANDIDATE);
      setNewReasonsInput("");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "添加候选职位失败";
      toast.error(message);
    }
  };

  const handleSaveEdit = async () => {
    if (!editingCandidate || isSyncing) {
      return;
    }
    try {
      await updateCandidate(editingCandidate.id, {
        ...toMutationInput(editingCandidate),
        matchReasons: parseReasons(editReasonsInput),
      });
      toast.success("候选职位已更新");
      setEditingCandidate(null);
      setEditReasonsInput("");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "更新候选职位失败";
      toast.error(message);
    }
  };

  const handleDelete = async (id: string) => {
    if (isSyncing) {
      return;
    }
    try {
      await deleteCandidate(id);
      toast.success("候选职位已删除");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "删除候选职位失败";
      toast.error(message);
    }
  };

  const handleStatusChange = async (candidate: Candidate, status: CandidateStatus) => {
    if (isSyncing) {
      return;
    }
    try {
      await updateCandidate(candidate.id, {
        ...toMutationInput(candidate),
        status,
      });
      toast.success("状态已更新");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "更新候选状态失败";
      toast.error(message);
    }
  };

  const handlePromote = async (candidate: Candidate) => {
    if (isSyncing) {
      return;
    }
    if (candidate.status === "promoted") {
      toast.info("该候选已转为线索");
      return;
    }
    try {
      await promoteCandidate(candidate.id, {
        status: "new",
        priority: 4,
        nextAction: "review candidate and apply",
      });
      await fetchLeads();
      toast.success("已转为线索");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "转为线索失败";
      toast.error(message);
    }
  };

  const syncingLabel = isSyncing ? "同步中..." : "";
  const loadingLabel = isLoading ? "加载中..." : "";

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative w-full sm:max-w-sm">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="搜索公司/职位/来源/地区"
            className="pl-9"
          />
        </div>
        <Button onClick={() => setIsAddOpen(true)} disabled={isSyncing}>
          <Plus className="mr-2 h-4 w-4" />
          添加候选
        </Button>
      </div>

      <div className="overflow-hidden rounded-xl border border-border bg-card/30">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>公司 / 职位</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>匹配度</TableHead>
              <TableHead>来源</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-sm text-muted-foreground">
                  暂无候选职位
                </TableCell>
              </TableRow>
            ) : (
              filtered.map((candidate) => (
                <TableRow key={candidate.id}>
                  <TableCell className="align-top">
                    <div className="font-medium">{candidate.company}</div>
                    <div className="text-sm text-muted-foreground">{candidate.position}</div>
                    {candidate.location ? <div className="text-xs text-muted-foreground/80">{candidate.location}</div> : null}
                  </TableCell>
                  <TableCell className="align-top">
                    <div className="space-y-2">
                      <CandidateStatusBadge status={candidate.status} />
                      {candidate.status === "promoted" ? (
                        <span className="text-xs text-muted-foreground">已通过“转线索”自动更新</span>
                      ) : (
                        <Select value={candidate.status} onValueChange={(value) => handleStatusChange(candidate, value as CandidateStatus)}>
                          <SelectTrigger className="h-8 w-[130px] text-xs">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {MANUAL_STATUS_OPTIONS.map((status) => (
                              <SelectItem key={status} value={status}>
                                {CANDIDATE_STATUS_CONFIG[status].label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="align-top">
                    <span className={cn("text-sm font-medium", candidate.matchScore >= 80 ? "text-success" : "text-foreground")}>
                      {candidate.matchScore}
                    </span>
                  </TableCell>
                  <TableCell className="align-top text-sm text-muted-foreground">{candidate.source || "-"}</TableCell>
                  <TableCell className="align-top text-sm text-muted-foreground">{formatDateTime(candidate.updatedAt)}</TableCell>
                  <TableCell className="align-top text-right">
                    <div className="flex items-center justify-end gap-2">
                      <Button
                        size="sm"
                        variant={candidate.status === "promoted" ? "secondary" : "default"}
                        disabled={isSyncing || candidate.status === "promoted"}
                        onClick={() => handlePromote(candidate)}
                      >
                        <ArrowRightCircle className="mr-1 h-3.5 w-3.5" />
                        转线索
                      </Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => {
                              setEditingCandidate(candidate);
                              setEditReasonsInput(candidate.matchReasons.join("\n"));
                            }}
                          >
                            <Pencil className="mr-2 h-4 w-4" />
                            编辑
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            onClick={() => void handleDelete(candidate.id)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            删除
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {(loadingLabel || syncingLabel) && (
        <div className="text-xs text-muted-foreground">
          {loadingLabel}
          {loadingLabel && syncingLabel ? " · " : ""}
          {syncingLabel}
        </div>
      )}

      <Dialog open={isAddOpen} onOpenChange={setIsAddOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Sparkles className="h-4 w-4" />
              新增候选职位
            </DialogTitle>
            <DialogDescription>手动录入候选职位，后续可直接转为正式线索。</DialogDescription>
          </DialogHeader>

          <FieldGroup>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Field>
                <FieldLabel>公司</FieldLabel>
                <Input value={newCandidate.company} onChange={(e) => setNewCandidate((prev) => ({ ...prev, company: e.target.value }))} />
              </Field>
              <Field>
                <FieldLabel>职位</FieldLabel>
                <Input value={newCandidate.position} onChange={(e) => setNewCandidate((prev) => ({ ...prev, position: e.target.value }))} />
              </Field>
              <Field>
                <FieldLabel>来源</FieldLabel>
                <Input value={newCandidate.source} onChange={(e) => setNewCandidate((prev) => ({ ...prev, source: e.target.value }))} />
              </Field>
              <Field>
                <FieldLabel>地区</FieldLabel>
                <Input value={newCandidate.location} onChange={(e) => setNewCandidate((prev) => ({ ...prev, location: e.target.value }))} />
              </Field>
              <Field>
                <FieldLabel>JD URL</FieldLabel>
                <Input value={newCandidate.jdUrl} onChange={(e) => setNewCandidate((prev) => ({ ...prev, jdUrl: e.target.value }))} />
              </Field>
              <Field>
                <FieldLabel>公司官网</FieldLabel>
                <Input
                  value={newCandidate.companyWebsiteUrl}
                  onChange={(e) => setNewCandidate((prev) => ({ ...prev, companyWebsiteUrl: e.target.value }))}
                />
              </Field>
              <Field>
                <FieldLabel>匹配度（0-100）</FieldLabel>
                <Input
                  type="number"
                  min={0}
                  max={100}
                  value={newCandidate.matchScore}
                  onChange={(e) => setNewCandidate((prev) => ({ ...prev, matchScore: Number(e.target.value) || 0 }))}
                />
              </Field>
              <Field>
                <FieldLabel>状态</FieldLabel>
                <Select
                  value={newCandidate.status}
                  onValueChange={(value) => setNewCandidate((prev) => ({ ...prev, status: value as CandidateStatus }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {MANUAL_STATUS_OPTIONS.map((status) => (
                      <SelectItem key={status} value={status}>
                        {CANDIDATE_STATUS_CONFIG[status].label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            </div>

            <Field>
              <FieldLabel>匹配理由（换行或逗号分隔）</FieldLabel>
              <Textarea value={newReasonsInput} onChange={(e) => setNewReasonsInput(e.target.value)} rows={3} />
            </Field>
            <Field>
              <FieldLabel>推荐说明</FieldLabel>
              <Textarea
                value={newCandidate.recommendationNotes}
                onChange={(e) => setNewCandidate((prev) => ({ ...prev, recommendationNotes: e.target.value }))}
                rows={3}
              />
            </Field>
            <Field>
              <FieldLabel>备注</FieldLabel>
              <Textarea value={newCandidate.notes} onChange={(e) => setNewCandidate((prev) => ({ ...prev, notes: e.target.value }))} rows={3} />
            </Field>
          </FieldGroup>

          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setIsAddOpen(false)} disabled={isSyncing}>
              取消
            </Button>
            <Button onClick={() => void handleCreate()} disabled={isSyncing}>
              创建候选
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog
        open={Boolean(editingCandidate)}
        onOpenChange={(open) => {
          if (!open) {
            setEditingCandidate(null);
            setEditReasonsInput("");
          }
        }}
      >
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>编辑候选职位</DialogTitle>
            <DialogDescription>调整候选职位信息和匹配结果。</DialogDescription>
          </DialogHeader>

          {editingCandidate ? (
            <FieldGroup>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <Field>
                  <FieldLabel>公司</FieldLabel>
                  <Input
                    value={editingCandidate.company}
                    onChange={(e) => setEditingCandidate((prev) => (prev ? { ...prev, company: e.target.value } : prev))}
                  />
                </Field>
                <Field>
                  <FieldLabel>职位</FieldLabel>
                  <Input
                    value={editingCandidate.position}
                    onChange={(e) => setEditingCandidate((prev) => (prev ? { ...prev, position: e.target.value } : prev))}
                  />
                </Field>
                <Field>
                  <FieldLabel>来源</FieldLabel>
                  <Input
                    value={editingCandidate.source}
                    onChange={(e) => setEditingCandidate((prev) => (prev ? { ...prev, source: e.target.value } : prev))}
                  />
                </Field>
                <Field>
                  <FieldLabel>地区</FieldLabel>
                  <Input
                    value={editingCandidate.location}
                    onChange={(e) => setEditingCandidate((prev) => (prev ? { ...prev, location: e.target.value } : prev))}
                  />
                </Field>
                <Field>
                  <FieldLabel>JD URL</FieldLabel>
                  <Input
                    value={editingCandidate.jdUrl}
                    onChange={(e) => setEditingCandidate((prev) => (prev ? { ...prev, jdUrl: e.target.value } : prev))}
                  />
                </Field>
                <Field>
                  <FieldLabel>公司官网</FieldLabel>
                  <Input
                    value={editingCandidate.companyWebsiteUrl}
                    onChange={(e) => setEditingCandidate((prev) => (prev ? { ...prev, companyWebsiteUrl: e.target.value } : prev))}
                  />
                </Field>
                <Field>
                  <FieldLabel>匹配度（0-100）</FieldLabel>
                  <Input
                    type="number"
                    min={0}
                    max={100}
                    value={editingCandidate.matchScore}
                    onChange={(e) =>
                      setEditingCandidate((prev) => (prev ? { ...prev, matchScore: Number(e.target.value) || 0 } : prev))
                    }
                  />
                </Field>
                <Field>
                  <FieldLabel>状态</FieldLabel>
                  {editingCandidate.status === "promoted" ? (
                    <div className="rounded-md border border-border px-3 py-2 text-sm text-muted-foreground">
                      已转为线索，状态不可手动修改
                    </div>
                  ) : (
                    <Select
                      value={editingCandidate.status}
                      onValueChange={(value) =>
                        setEditingCandidate((prev) => (prev ? { ...prev, status: value as CandidateStatus } : prev))
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {MANUAL_STATUS_OPTIONS.map((status) => (
                          <SelectItem key={status} value={status}>
                            {CANDIDATE_STATUS_CONFIG[status].label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </Field>
              </div>

              <Field>
                <FieldLabel>匹配理由（换行或逗号分隔）</FieldLabel>
                <Textarea value={editReasonsInput} onChange={(e) => setEditReasonsInput(e.target.value)} rows={3} />
              </Field>
              <Field>
                <FieldLabel>推荐说明</FieldLabel>
                <Textarea
                  value={editingCandidate.recommendationNotes}
                  onChange={(e) => setEditingCandidate((prev) => (prev ? { ...prev, recommendationNotes: e.target.value } : prev))}
                  rows={3}
                />
              </Field>
              <Field>
                <FieldLabel>备注</FieldLabel>
                <Textarea
                  value={editingCandidate.notes}
                  onChange={(e) => setEditingCandidate((prev) => (prev ? { ...prev, notes: e.target.value } : prev))}
                  rows={3}
                />
              </Field>
            </FieldGroup>
          ) : null}

          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setEditingCandidate(null)} disabled={isSyncing}>
              取消
            </Button>
            <Button onClick={() => void handleSaveEdit()} disabled={isSyncing || !editingCandidate}>
              保存
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
