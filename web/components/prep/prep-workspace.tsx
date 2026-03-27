"use client";

import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { fetchPrepLeadContextPreview, fetchPrepMeta } from "@/lib/prep-api";
import { DEFAULT_PREP_META, PrepLeadContextPreview, PrepMeta } from "@/lib/prep-types";
import { PrepContextPreviewCard } from "./prep-context-preview-card";

function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === "AbortError";
}

export function PrepWorkspace() {
  const searchParams = useSearchParams();
  const leadID = useMemo(() => (searchParams.get("lead_id") || "").trim(), [searchParams]);

  const [meta, setMeta] = useState<PrepMeta>(DEFAULT_PREP_META);
  const [metaError, setMetaError] = useState<string | null>(null);
  const [isMetaLoading, setIsMetaLoading] = useState(true);

  const [contextPreview, setContextPreview] = useState<PrepLeadContextPreview | null>(null);
  const [contextError, setContextError] = useState<string | null>(null);
  const [isContextLoading, setIsContextLoading] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    setIsMetaLoading(true);
    setMetaError(null);

    void fetchPrepMeta(controller.signal)
      .then((nextMeta) => {
        setMeta(nextMeta);
      })
      .catch((error: unknown) => {
        if (isAbortError(error)) {
          return;
        }
        const message = error instanceof Error && error.message ? error.message : "加载备面元信息失败";
        setMetaError(message);
      })
      .finally(() => {
        setIsMetaLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, []);

  useEffect(() => {
    const normalizedLeadID = leadID.trim();
    if (!normalizedLeadID) {
      setContextPreview(null);
      setContextError("请从线索表点击「备面」进入，或手动带上 lead_id 参数。");
      setIsContextLoading(false);
      return;
    }

    const controller = new AbortController();
    setIsContextLoading(true);
    setContextError(null);

    void fetchPrepLeadContextPreview(normalizedLeadID, controller.signal)
      .then((preview) => {
        setContextPreview(preview);
      })
      .catch((error: unknown) => {
        if (isAbortError(error)) {
          return;
        }
        setContextPreview(null);
        const message = error instanceof Error && error.message ? error.message : "加载上下文预览失败";
        setContextError(message);
      })
      .finally(() => {
        setIsContextLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, [leadID]);

  return (
    <main className="mx-auto w-full max-w-7xl px-4 pb-12 pt-6 sm:px-6">
      <section className="page-enter space-y-4 rounded-[32px] border border-[var(--panel-border)] bg-card/72 p-4 shadow-[var(--panel-shadow)] backdrop-blur-xl sm:p-6">
        <header className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-xl font-semibold tracking-tight">面试准备</h1>
            <Badge variant="outline">lead_id: {leadID || "-"}</Badge>
            {isMetaLoading ? <Badge variant="secondary">加载配置中</Badge> : null}
            {!isMetaLoading && meta.enabled ? <Badge variant="secondary">模块已启用</Badge> : null}
          </div>
          <p className="text-sm text-muted-foreground">先把资料上下文核对清楚，再开始练习和复盘，别上来就硬答。</p>
          {metaError ? <p className="text-sm text-destructive">{metaError}</p> : null}
          {!isMetaLoading && !meta.enabled ? (
            <p className="text-sm text-destructive">备面模块当前未启用（`T2O_PREP_ENABLED=false`）。</p>
          ) : null}
        </header>

        <Tabs defaultValue="materials" className="space-y-4">
          <TabsList>
            <TabsTrigger value="materials">资料</TabsTrigger>
            <TabsTrigger value="practice">练习</TabsTrigger>
            <TabsTrigger value="review">复盘</TabsTrigger>
          </TabsList>

          <TabsContent value="materials" className="space-y-4">
            <PrepContextPreviewCard preview={contextPreview} isLoading={isContextLoading} error={contextError} />
            <Card>
              <CardHeader>
                <CardTitle className="text-base">资料维护</CardTitle>
                <CardDescription>知识库编辑入口在 Sprint1 后续任务继续接线，这里先把页面壳子和上下文预览跑通。</CardDescription>
              </CardHeader>
            </Card>
          </TabsContent>

          <TabsContent value="practice">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">练习（占位）</CardTitle>
                <CardDescription>下一阶段会在这里放出题配置、题目列表和答案草稿编辑。</CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">当前只完成路由与 tab 壳子，练习流还没接入。</CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="review">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">复盘（占位）</CardTitle>
                <CardDescription>后续会接评分结果、改进建议和参考答案。</CardDescription>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">这块暂时是空壳，等提交评分链路完成后再填充。</CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </section>
    </main>
  );
}
