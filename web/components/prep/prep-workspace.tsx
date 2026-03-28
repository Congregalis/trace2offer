"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { createPrepSession, fetchPrepLeadContextPreview, fetchPrepMeta, fetchPrepSession, previewPrepRetrieval } from "@/lib/prep-api";
import { DEFAULT_PREP_META, PrepLeadContextPreview, PrepMeta, PrepRetrievalPreview, PrepSession } from "@/lib/prep-types";
import { AnswerDraftEditor } from "./answer-draft-editor";
import { PrepConfigPanel, PrepGenerationConfig } from "./prep-config-panel";
import { PrepContextPreviewCard } from "./prep-context-preview-card";
import { PrepRunTimeline } from "./prep-run-timeline";
import { PrepTraceDrawer } from "./prep-trace-drawer";
import { QuestionList } from "./question-list";
import { RetrievalTracePanel } from "./retrieval-trace-panel";

function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === "AbortError";
}

export function PrepWorkspace() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const leadID = useMemo(() => (searchParams.get("lead_id") || "").trim(), [searchParams]);
  const sessionID = useMemo(() => (searchParams.get("session_id") || "").trim(), [searchParams]);

  const [meta, setMeta] = useState<PrepMeta>(DEFAULT_PREP_META);
  const [metaError, setMetaError] = useState<string | null>(null);
  const [isMetaLoading, setIsMetaLoading] = useState(true);

  const [contextPreview, setContextPreview] = useState<PrepLeadContextPreview | null>(null);
  const [contextError, setContextError] = useState<string | null>(null);
  const [isContextLoading, setIsContextLoading] = useState(false);

  const [session, setSession] = useState<PrepSession | null>(null);
  const [practiceError, setPracticeError] = useState<string | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);

  const [retrievalQuery, setRetrievalQuery] = useState("");
  const [retrievalPreview, setRetrievalPreview] = useState<PrepRetrievalPreview | null>(null);
  const [retrievalError, setRetrievalError] = useState<string | null>(null);
  const [isRetrievalLoading, setIsRetrievalLoading] = useState(false);

  const [config, setConfig] = useState<PrepGenerationConfig>({
    topicKeys: [],
    questionCount: DEFAULT_PREP_META.defaultQuestionCount,
    includeResume: true,
    includeProfile: true,
    includeLeadDocs: true,
  });

  useEffect(() => {
    const controller = new AbortController();
    setIsMetaLoading(true);
    setMetaError(null);

    void fetchPrepMeta(controller.signal)
      .then((nextMeta) => {
        setMeta(nextMeta);
        setConfig((previous) => ({
          ...previous,
          questionCount: nextMeta.defaultQuestionCount || previous.questionCount,
        }));
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
        setRetrievalQuery((previous) => {
          if (previous.trim()) {
            return previous;
          }
          const query = `${preview.company || ""} ${preview.position || ""} 面试问题`.trim();
          return query || "面试准备";
        });
        setConfig((previous) => ({
          ...previous,
          topicKeys: previous.topicKeys.length > 0 ? previous.topicKeys : preview.topicKeys,
        }));
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

  useEffect(() => {
    const normalizedSessionID = sessionID.trim();
    if (!normalizedSessionID) {
      return;
    }

    const controller = new AbortController();
    setPracticeError(null);

    void fetchPrepSession(normalizedSessionID, controller.signal)
      .then((loadedSession) => {
        setSession(loadedSession);
      })
      .catch((error: unknown) => {
        if (isAbortError(error)) {
          return;
        }
        const message = error instanceof Error && error.message ? error.message : "加载备面会话失败";
        setPracticeError(message);
      });

    return () => {
      controller.abort();
    };
  }, [sessionID]);

  const handleGenerateQuestions = () => {
    if (!leadID.trim()) {
      setPracticeError("lead_id 缺失，无法生成题目。请从线索表重新进入。");
      return;
    }
    setIsGenerating(true);
    setPracticeError(null);

    void createPrepSession({
      leadId: leadID.trim(),
      topicKeys: config.topicKeys,
      questionCount: config.questionCount,
      includeResume: config.includeResume,
      includeProfile: config.includeProfile,
      includeLeadDocs: config.includeLeadDocs,
    })
      .then((createdSession) => {
        setSession(createdSession);

        const nextQuery = new URLSearchParams(searchParams.toString());
        nextQuery.set("lead_id", leadID.trim());
        nextQuery.set("session_id", createdSession.id);
        router.replace(`/prep?${nextQuery.toString()}`);
      })
      .catch((error: unknown) => {
        const message = error instanceof Error && error.message ? error.message : "生成题目失败";
        setPracticeError(message);
      })
      .finally(() => {
        setIsGenerating(false);
      });
  };

  const handlePreviewRetrieval = () => {
    if (!leadID.trim()) {
      setRetrievalError("lead_id 缺失，无法预览检索链路。");
      return;
    }
    if (!retrievalQuery.trim()) {
      setRetrievalError("请输入检索 query。");
      return;
    }

    setIsRetrievalLoading(true);
    setRetrievalError(null);

    void previewPrepRetrieval({
      leadId: leadID.trim(),
      query: retrievalQuery.trim(),
      topicKeys: config.topicKeys,
      topK: config.questionCount,
      includeTrace: true,
      includeResume: config.includeResume,
      includeProfile: config.includeProfile,
      includeLeadDocs: config.includeLeadDocs,
    })
      .then((preview) => {
        setRetrievalPreview(preview);
      })
      .catch((error: unknown) => {
        const message = error instanceof Error && error.message ? error.message : "加载检索预览失败";
        setRetrievalError(message);
      })
      .finally(() => {
        setIsRetrievalLoading(false);
      });
  };

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
          <p className="text-sm text-muted-foreground">备面入口只负责练习和复盘，资料维护请去「资料库」Tab。</p>
          {metaError ? <p className="text-sm text-destructive">{metaError}</p> : null}
          {!isMetaLoading && !meta.enabled ? (
            <p className="text-sm text-destructive">备面模块当前未启用（`T2O_PREP_ENABLED=false`）。</p>
          ) : null}
        </header>

        <Tabs defaultValue="practice" className="space-y-4">
          <TabsList>
            <TabsTrigger value="practice">练习</TabsTrigger>
            <TabsTrigger value="review">复盘</TabsTrigger>
          </TabsList>

          <TabsContent value="practice">
            <div className="space-y-4">
              <PrepContextPreviewCard preview={contextPreview} isLoading={isContextLoading} error={contextError} />

              <PrepConfigPanel
                availableTopicKeys={contextPreview?.topicKeys || []}
                config={config}
                onChange={setConfig}
                onGenerate={handleGenerateQuestions}
                isGenerating={isGenerating}
                disabled={!meta.enabled || !leadID.trim()}
              />

              {practiceError ? <p className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">{practiceError}</p> : null}

              <div className="flex justify-end">
                <PrepTraceDrawer trace={session?.generationTrace || null} />
              </div>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">检索预览</CardTitle>
                  <CardDescription>先看候选召回和最终上下文，再决定是否生成题目。</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Input
                      value={retrievalQuery}
                      onChange={(event) => setRetrievalQuery(event.target.value)}
                      placeholder="输入检索 query，例如：RAG 常见面试问题"
                      disabled={!meta.enabled || !leadID.trim() || isRetrievalLoading}
                    />
                    <Button type="button" onClick={handlePreviewRetrieval} disabled={!meta.enabled || !leadID.trim() || isRetrievalLoading}>
                      {isRetrievalLoading ? "预览中..." : "预览检索链路"}
                    </Button>
                  </div>
                  {retrievalError ? <p className="text-sm text-destructive">{retrievalError}</p> : null}
                </CardContent>
              </Card>

              <RetrievalTracePanel preview={retrievalPreview} />
              <PrepRunTimeline trace={session?.generationTrace || null} />
              <QuestionList questions={session?.questions || []} />
              {session ? <AnswerDraftEditor sessionId={session.id} questions={session.questions} initialAnswers={session.answers} /> : null}
            </div>
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
