"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import type { DiscoveryPreset } from "@/lib/discovery-presets";
import { DiscoveryRule, DiscoveryRuleMutationInput, DiscoveryRunResult } from "@/lib/types";
import { DiscoveryPresetCards } from "@/components/discovery-preset-cards";
import { useDiscoveryStore } from "@/lib/discovery-store";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Globe2,
  Pencil,
  Play,
  Plus,
  Settings2,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

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

export function DiscoveryRulesPanel({
  onDiscoveryFinished,
}: {
  onDiscoveryFinished?: (result: DiscoveryRunResult) => Promise<void> | void;
}) {
  const {
    rules,
    lastRun,
    isSyncing,
    isRunning,
    hasLoaded,
    fetchRules,
    addRule,
    updateRule,
    deleteRule,
    runDiscoveryNow,
  } = useDiscoveryStore();

  const [isManageOpen, setIsManageOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<"rules" | "market">("rules");
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

  useEffect(() => {
    if (!isManageOpen) {
      return;
    }
    void fetchRules().catch((error) => {
      const message = error instanceof Error && error.message ? error.message : "加载发现规则失败";
      toast.error(message);
    });
  }, [fetchRules, isManageOpen]);

  const enabledCount = useMemo(() => rules.filter((item) => item.enabled).length, [rules]);

  const resetEditor = () => {
    setEditingRule(null);
    setRuleForm(EMPTY_RULE);
    setIncludeKeywordsInput("");
    setExcludeKeywordsInput("");
  };

  const handleManageOpenChange = (open: boolean) => {
    setIsManageOpen(open);
    if (open) {
      setActiveTab("rules");
      return;
    }
    resetEditor();
    setActiveTab("rules");
  };

  const beginEdit = (rule: DiscoveryRule) => {
    setEditingRule(rule);
    setRuleForm(toMutationInput(rule));
    setIncludeKeywordsInput(rule.includeKeywords.join(", "));
    setExcludeKeywordsInput(rule.excludeKeywords.join(", "));
    setActiveTab("rules");
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
      await fetchRules();
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
      await fetchRules();
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
      await fetchRules();
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

  const handleAddPreset = async (preset: DiscoveryPreset) => {
    try {
      await addRule(preset.rule);
      toast.success(`${preset.name} 已添加`);
      await fetchRules();
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "添加示例规则失败";
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
          <Button size="sm" variant="outline" onClick={() => handleManageOpenChange(true)} disabled={isSyncing || isRunning}>
            <Settings2 className="mr-1 h-3.5 w-3.5" />
            管理规则
          </Button>
          <Button size="sm" onClick={() => void handleRunNow()} disabled={isRunning || isSyncing}>
            <Play className="mr-1 h-3.5 w-3.5" />
            立即发现
          </Button>
        </div>
      </div>

      <Sheet open={isManageOpen} onOpenChange={handleManageOpenChange}>
        <SheetContent
          side="right"
          className="flex h-full w-[50vw] max-w-none flex-col overflow-hidden p-0 sm:max-w-none"
        >
          <div className="flex min-h-0 flex-1 flex-col">
            <SheetHeader className="border-b border-border px-6 py-5 text-left shrink-0">
              <SheetTitle>发现规则管理</SheetTitle>
              <SheetDescription>配置职位来源与关键词，并从规则市场挑选内置规则。</SheetDescription>
            </SheetHeader>

            <Tabs
              value={activeTab}
              onValueChange={(value) => setActiveTab(value as "rules" | "market")}
              className="flex min-h-0 flex-1 flex-col gap-0"
            >
              <div className="border-b border-border px-6 py-4">
                <TabsList className="h-10 bg-card/40">
                  <TabsTrigger value="rules" className="min-w-28">
                    我的规则
                  </TabsTrigger>
                  <TabsTrigger value="market" className="min-w-28">
                    规则市场
                  </TabsTrigger>
                </TabsList>
              </div>

              <TabsContent value="rules" className="min-h-0 flex-1 overflow-auto px-6 py-5">
                <div className="grid gap-4 xl:grid-cols-[minmax(0,400px)_minmax(0,1fr)]">
                  <div className="space-y-3 rounded-lg border border-border bg-card/20 p-4">
                    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                      <div className="text-sm font-medium">{editingRule ? "编辑规则" : "新建规则"}</div>
                      <div className="flex items-center gap-3">
                        {editingRule ? (
                          <Button size="sm" variant="ghost" onClick={resetEditor}>
                            取消编辑
                          </Button>
                        ) : null}
                        <Button asChild variant="link" size="sm" className="h-auto px-0 text-xs">
                          <Link href="/docs/discovery-rules" target="_blank" rel="noreferrer">
                            不会填？看快速上手
                          </Link>
                        </Button>
                      </div>
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

                  <div className="rounded-lg border border-border bg-card/20">
                    <div className="border-b border-border px-4 py-3">
                      <div className="text-sm font-medium text-foreground">已有规则</div>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {rules.length === 0 ? "还没有规则，去规则市场挑一条。" : "这里是当前已经生效的发现规则。"}
                      </p>
                    </div>
                    <div className={cn("overflow-auto", rules.length > 0 ? "max-h-[620px]" : "max-h-none")}>
                      <Table>
                        <TableHeader className="sticky top-0 z-10 bg-background">
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
                                <div className="space-y-3">
                                  <div>还没有规则，去规则市场挑一条。</div>
                                  <Button size="sm" variant="outline" onClick={() => setActiveTab("market")}>
                                    去规则市场
                                  </Button>
                                </div>
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
                </div>
              </TabsContent>

              <TabsContent value="market" className="min-h-0 flex-1 overflow-auto px-6 py-5">
                <div className="space-y-4">
                  <div className="flex items-center justify-between rounded-lg border border-border bg-card/20 px-4 py-3">
                    <div>
                      <div className="text-sm font-medium text-foreground">规则市场</div>
                      <p className="mt-1 text-xs text-muted-foreground">国外远程和国内社区规则都在这里，挑中后直接一键添加。</p>
                    </div>
                    <Button asChild variant="link" size="sm" className="h-auto px-0 text-xs">
                      <Link href="/docs/discovery-rules" target="_blank" rel="noreferrer">
                        不会填？看快速上手
                      </Link>
                    </Button>
                  </div>

                  <div className="rounded-lg border border-border bg-card/20 p-4">
                    <div className="flex items-start gap-3">
                      <div className="rounded-lg border border-border/80 bg-background/80 p-2">
                        <Globe2 className="h-4 w-4 text-foreground" />
                      </div>
                      <div className="space-y-1">
                        <div className="text-sm font-medium text-foreground">按来源分区</div>
                        <p className="text-xs text-muted-foreground">
                          国外远程通常更标准化，国内社区通常更灵活。你可以各挑几条，不必一口气全上。
                        </p>
                      </div>
                    </div>
                  </div>

                  <DiscoveryPresetCards
                    rules={rules}
                    onAddPreset={handleAddPreset}
                    isBusy={isSyncing || isRunning}
                    title="规则市场"
                    description="所有内置规则都在这里，按国外远程和国内社区两类来挑。"
                  />
                </div>
              </TabsContent>
            </Tabs>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}
