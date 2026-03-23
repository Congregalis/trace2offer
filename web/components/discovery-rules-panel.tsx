"use client";

import { useEffect, useMemo, useState } from "react";
import { DiscoveryRule, DiscoveryRuleMutationInput, DiscoveryRunResult } from "@/lib/types";
import { useDiscoveryStore } from "@/lib/discovery-store";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Badge } from "@/components/ui/badge";
import { RefreshCcw, Settings2, Plus, Trash2, Pencil, Play } from "lucide-react";
import { toast } from "sonner";

const EMPTY_RULE: DiscoveryRuleMutationInput = {
  name: "",
  feedUrl: "",
  source: "",
  defaultLocation: "",
  includeKeywords: [],
  excludeKeywords: [],
  enabled: true,
};

function toMutationInput(rule: DiscoveryRule): DiscoveryRuleMutationInput {
  return {
    name: rule.name,
    feedUrl: rule.feedUrl,
    source: rule.source,
    defaultLocation: rule.defaultLocation,
    includeKeywords: [...rule.includeKeywords],
    excludeKeywords: [...rule.excludeKeywords],
    enabled: rule.enabled,
  };
}

function parseKeywords(value: string): string[] {
  return value
    .split(/\r?\n|,|，|;|；/g)
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatRunSummary(lastRun: DiscoveryRunResult | null): string {
  if (!lastRun) {
    return "尚未执行发现任务";
  }
  return `规则 ${lastRun.rulesExecuted}/${lastRun.rulesTotal} · 抓取 ${lastRun.entriesFetched} · 新增 ${lastRun.candidatesCreated} · 更新 ${lastRun.candidatesUpdated} · 错误 ${lastRun.errors.length}`;
}

export function DiscoveryRulesPanel({ onDiscoveryFinished }: { onDiscoveryFinished?: (result: DiscoveryRunResult) => Promise<void> | void }) {
  const { rules, lastRun, isLoading, isSyncing, isRunning, hasLoaded, fetchRules, addRule, updateRule, deleteRule, runDiscoveryNow } =
    useDiscoveryStore();

  const [isManageOpen, setIsManageOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<DiscoveryRule | null>(null);
  const [ruleForm, setRuleForm] = useState<DiscoveryRuleMutationInput>(EMPTY_RULE);
  const [includeKeywordsInput, setIncludeKeywordsInput] = useState("");
  const [excludeKeywordsInput, setExcludeKeywordsInput] = useState("");

  useEffect(() => {
    if (hasLoaded) {
      return;
    }
    void fetchRules().catch((error) => {
      const message = error instanceof Error && error.message ? error.message : "加载发现规则失败";
      toast.error(message);
    });
  }, [fetchRules, hasLoaded]);

  const enabledCount = useMemo(() => rules.filter((item) => item.enabled).length, [rules]);

  const resetEditor = () => {
    setEditingRule(null);
    setRuleForm(EMPTY_RULE);
    setIncludeKeywordsInput("");
    setExcludeKeywordsInput("");
  };

  const beginEdit = (rule: DiscoveryRule) => {
    setEditingRule(rule);
    setRuleForm(toMutationInput(rule));
    setIncludeKeywordsInput(rule.includeKeywords.join(", "));
    setExcludeKeywordsInput(rule.excludeKeywords.join(", "));
  };

  const handleSubmitRule = async () => {
    try {
      const payload: DiscoveryRuleMutationInput = {
        ...ruleForm,
        includeKeywords: parseKeywords(includeKeywordsInput),
        excludeKeywords: parseKeywords(excludeKeywordsInput),
      };
      if (editingRule) {
        await updateRule(editingRule.id, payload);
        toast.success("发现规则已更新");
      } else {
        await addRule(payload);
        toast.success("发现规则已创建");
      }
      resetEditor();
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "保存发现规则失败";
      toast.error(message);
    }
  };

  const handleToggleRule = async (rule: DiscoveryRule) => {
    try {
      await updateRule(rule.id, {
        ...toMutationInput(rule),
        enabled: !rule.enabled,
      });
      toast.success(rule.enabled ? "规则已停用" : "规则已启用");
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "更新规则状态失败";
      toast.error(message);
    }
  };

  const handleDeleteRule = async (id: string) => {
    try {
      await deleteRule(id);
      toast.success("规则已删除");
      if (editingRule?.id === id) {
        resetEditor();
      }
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "删除规则失败";
      toast.error(message);
    }
  };

  const handleRunNow = async () => {
    try {
      const result = await runDiscoveryNow();
      toast.success("发现任务执行完成");
      if (onDiscoveryFinished) {
        await onDiscoveryFinished(result);
      }
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "执行发现任务失败";
      toast.error(message);
    }
  };

  return (
    <div className="rounded-xl border border-border bg-card/30 p-3 sm:p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">发现规则</span>
            <Badge variant="secondary">{enabledCount}/{rules.length} 已启用</Badge>
          </div>
          <p className="text-xs text-muted-foreground">{formatRunSummary(lastRun)}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => void fetchRules()} disabled={isLoading || isSyncing || isRunning}>
            <RefreshCcw className="mr-1 h-3.5 w-3.5" />
            刷新规则
          </Button>
          <Button size="sm" variant="outline" onClick={() => setIsManageOpen(true)} disabled={isSyncing || isRunning}>
            <Settings2 className="mr-1 h-3.5 w-3.5" />
            管理规则
          </Button>
          <Button size="sm" onClick={() => void handleRunNow()} disabled={isRunning || isSyncing}>
            <Play className="mr-1 h-3.5 w-3.5" />
            立即发现
          </Button>
        </div>
      </div>

      <Dialog open={isManageOpen} onOpenChange={setIsManageOpen}>
        <DialogContent className="max-w-5xl">
          <DialogHeader>
            <DialogTitle>发现规则管理</DialogTitle>
            <DialogDescription>配置职位来源与关键词，控制候选池自动发现行为。</DialogDescription>
          </DialogHeader>

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-[360px_minmax(0,1fr)]">
            <div className="space-y-3 rounded-lg border border-border p-3">
              <div className="flex items-center justify-between">
                <div className="text-sm font-medium">{editingRule ? "编辑规则" : "新建规则"}</div>
                {editingRule ? (
                  <Button size="sm" variant="ghost" onClick={resetEditor}>
                    取消编辑
                  </Button>
                ) : null}
              </div>

              <FieldGroup>
                <Field>
                  <FieldLabel>规则名</FieldLabel>
                  <Input value={ruleForm.name} onChange={(e) => setRuleForm((prev) => ({ ...prev, name: e.target.value }))} />
                </Field>
                <Field>
                  <FieldLabel>RSS/Atom URL</FieldLabel>
                  <Input value={ruleForm.feedUrl} onChange={(e) => setRuleForm((prev) => ({ ...prev, feedUrl: e.target.value }))} />
                </Field>
                <Field>
                  <FieldLabel>来源标签</FieldLabel>
                  <Input value={ruleForm.source} onChange={(e) => setRuleForm((prev) => ({ ...prev, source: e.target.value }))} />
                </Field>
                <Field>
                  <FieldLabel>默认地区</FieldLabel>
                  <Input
                    value={ruleForm.defaultLocation}
                    onChange={(e) => setRuleForm((prev) => ({ ...prev, defaultLocation: e.target.value }))}
                  />
                </Field>
                <Field>
                  <FieldLabel>包含关键词（逗号/换行分隔）</FieldLabel>
                  <Textarea value={includeKeywordsInput} onChange={(e) => setIncludeKeywordsInput(e.target.value)} rows={3} />
                </Field>
                <Field>
                  <FieldLabel>排除关键词（逗号/换行分隔）</FieldLabel>
                  <Textarea value={excludeKeywordsInput} onChange={(e) => setExcludeKeywordsInput(e.target.value)} rows={3} />
                </Field>
                <Field>
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={ruleForm.enabled}
                      onChange={(e) => setRuleForm((prev) => ({ ...prev, enabled: e.target.checked }))}
                    />
                    启用规则
                  </label>
                </Field>
              </FieldGroup>

              <Button onClick={() => void handleSubmitRule()} disabled={isSyncing || isRunning} className="w-full">
                <Plus className="mr-2 h-4 w-4" />
                {editingRule ? "保存规则" : "创建规则"}
              </Button>
            </div>

            <div className="overflow-hidden rounded-lg border border-border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>规则</TableHead>
                    <TableHead>来源</TableHead>
                    <TableHead>关键词</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rules.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="py-8 text-center text-sm text-muted-foreground">
                        暂无规则
                      </TableCell>
                    </TableRow>
                  ) : (
                    rules.map((rule) => (
                      <TableRow key={rule.id}>
                        <TableCell className="align-top">
                          <div className="font-medium">{rule.name}</div>
                          <div className="line-clamp-1 text-xs text-muted-foreground">{rule.feedUrl}</div>
                        </TableCell>
                        <TableCell className="align-top text-sm text-muted-foreground">{rule.source || "-"}</TableCell>
                        <TableCell className="align-top">
                          <div className="text-xs text-muted-foreground">
                            + {rule.includeKeywords.join(", ") || "-"}
                          </div>
                          <div className="text-xs text-muted-foreground">
                            - {rule.excludeKeywords.join(", ") || "-"}
                          </div>
                        </TableCell>
                        <TableCell className="align-top">
                          <Badge variant={rule.enabled ? "default" : "secondary"}>
                            {rule.enabled ? "启用" : "停用"}
                          </Badge>
                        </TableCell>
                        <TableCell className="align-top text-right">
                          <div className="flex items-center justify-end gap-2">
                            <Button size="sm" variant="outline" onClick={() => beginEdit(rule)} disabled={isSyncing || isRunning}>
                              <Pencil className="mr-1 h-3.5 w-3.5" />
                              编辑
                            </Button>
                            <Button size="sm" variant="outline" onClick={() => void handleToggleRule(rule)} disabled={isSyncing || isRunning}>
                              {rule.enabled ? "停用" : "启用"}
                            </Button>
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => void handleDeleteRule(rule.id)}
                              disabled={isSyncing || isRunning}
                            >
                              <Trash2 className="mr-1 h-3.5 w-3.5" />
                              删除
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
