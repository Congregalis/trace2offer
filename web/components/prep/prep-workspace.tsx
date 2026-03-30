"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  createPrepSessionStream,
  fetchPrepLeadContextPreview,
  fetchPrepMeta,
  fetchPrepSession,
  retryPrepSessionEvaluation,
  submitPrepSession,
} from "@/lib/prep-api";
import { DEFAULT_PREP_META, PrepAnswer, PrepGenerationTrace, PrepLeadContextPreview, PrepMeta, PrepReferenceAnswer, PrepSession } from "@/lib/prep-types";
import { AnswerDraftEditor } from "./answer-draft-editor";
import { PrepConfigPanel, PrepGenerationConfig } from "./prep-config-panel";
import { PrepContextPreviewCard } from "./prep-context-preview-card";
import { ReferenceAnswerDrawer } from "./reference-answer-drawer";
import { PrepRunTimeline } from "./prep-run-timeline";
import { PrepTraceDrawer } from "./prep-trace-drawer";
import { QuestionScoreCard } from "./question-score-card";
import { QuestionList } from "./question-list";
import { ReviewSummaryCard } from "./review-summary-card";
import { SubmitAnswersButton } from "./submit-answers-button";
import { WeakPointList } from "./weak-point-list";

function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === "AbortError";
}

function deriveDraftAnswers(session: PrepSession | null): PrepAnswer[] {
  if (!session) {
    return [];
  }
  const answerMap = new Map<number, string>();
  for (const answer of session.answers || []) {
    if (answer.questionId > 0) {
      answerMap.set(answer.questionId, answer.answer || "");
    }
  }
  return (session.questions || []).map((question) => ({
    questionId: question.id,
    answer: answerMap.get(question.id) || "",
  }));
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
  const [liveTrace, setLiveTrace] = useState<PrepGenerationTrace | null>(null);
  const [liveStageStatus, setLiveStageStatus] = useState<Record<string, string>>({});
  const [liveStageOutput, setLiveStageOutput] = useState<Record<string, string>>({});
  const [draftAnswers, setDraftAnswers] = useState<PrepAnswer[]>([]);
  const [isRetryingEvaluation, setIsRetryingEvaluation] = useState(false);

  const [config, setConfig] = useState<PrepGenerationConfig>({
    questionCount: DEFAULT_PREP_META.defaultQuestionCount,
    includeResume: true,
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

  useEffect(() => {
    setDraftAnswers(deriveDraftAnswers(session));
  }, [session]);

  useEffect(() => {
    if (!session?.id) {
      return;
    }
    const status = (session.evaluation?.status || "").trim();
    if (status !== "pending" && status !== "running") {
      return;
    }

    const timer = window.setTimeout(() => {
      void fetchPrepSession(session.id)
        .then((loadedSession) => {
          setSession(loadedSession);
        })
        .catch(() => {
          // keep current state; retry on next poll tick
        });
    }, 2000);

    return () => {
      window.clearTimeout(timer);
    };
  }, [session?.evaluation?.status, session?.id]);

  const handleGenerateQuestions = () => {
    if (!leadID.trim()) {
      setPracticeError("lead_id 缺失，无法生成题目。请从线索表重新进入。");
      return;
    }
    setIsGenerating(true);
    setPracticeError(null);
    setLiveTrace(null);
    setLiveStageStatus({});
    setLiveStageOutput({});

    void createPrepSessionStream({
      leadId: leadID.trim(),
      questionCount: config.questionCount,
      includeResume: config.includeResume,
      includeLeadDocs: config.includeLeadDocs,
    }, {
      onStage: (event) => {
        if (event.trace) {
          setLiveTrace(event.trace);
        }
        if (event.stage) {
          setLiveStageStatus((previous) => ({
            ...previous,
            [event.stage]: event.status || previous[event.stage] || "",
          }));
        }
        if (event.stage && event.delta) {
          setLiveStageOutput((previous) => ({
            ...previous,
            [event.stage]: `${previous[event.stage] || ""}${event.delta}`,
          }));
        }
      },
    })
      .then((createdSession) => {
        setSession(createdSession);
        setLiveTrace(createdSession.generationTrace || null);

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

  const timelineTrace = isGenerating ? liveTrace : session?.generationTrace || liveTrace;
  const answeredCount = useMemo(() => draftAnswers.filter((item) => (item.answer || "").trim().length > 0).length, [draftAnswers]);
  const totalQuestions = session?.questions.length || 0;
  const isSubmitted = (session?.status || "").trim() === "submitted";
  const evaluationStatus = (session?.evaluation?.status || "").trim();
  const scoreByQuestionID = useMemo(() => {
    const mapped = new Map<number, NonNullable<PrepSession["evaluation"]>["scores"][number]>();
    for (const score of session?.evaluation?.scores || []) {
      if (score.questionId > 0) {
        mapped.set(score.questionId, score);
      }
    }
    return mapped;
  }, [session?.evaluation?.scores]);
  const answerByQuestionID = useMemo(() => {
    const mapped = new Map<number, string>();
    for (const answer of session?.answers || []) {
      if (answer.questionId > 0) {
        mapped.set(answer.questionId, answer.answer || "");
      }
    }
    return mapped;
  }, [session?.answers]);
  const referenceAnswerByQuestionID = useMemo(() => {
    const mapped = new Map<number, PrepReferenceAnswer>();
    const all = session?.referenceAnswers || {};
    for (const value of Object.values(all)) {
      if (!value || value.questionId <= 0) {
        continue;
      }
      mapped.set(value.questionId, value);
    }
    return mapped;
  }, [session?.referenceAnswers]);

  const handleSubmitAnswers = async () => {
    if (!session?.id) {
      throw new Error("session_id is required");
    }
    const submittedSession = await submitPrepSession(session.id);
    setSession(submittedSession);
  };

  const handleRetryEvaluation = async () => {
    if (!session?.id) {
      return;
    }
    setIsRetryingEvaluation(true);
    try {
      const updated = await retryPrepSessionEvaluation(session.id);
      setSession(updated);
    } catch (error: unknown) {
      const message = error instanceof Error && error.message ? error.message : "重试评分失败";
      setPracticeError(message);
    } finally {
      setIsRetryingEvaluation(false);
    }
  };

  const handleReferenceAnswerGenerated = (questionId: number, generated: PrepReferenceAnswer) => {
    setSession((previous) => {
      if (!previous) {
        return previous;
      }
      return {
        ...previous,
        referenceAnswers: {
          ...previous.referenceAnswers,
          [String(questionId)]: generated,
        },
      };
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
          <p className="text-sm text-muted-foreground">备面入口只负责练习和复盘，资料维护请去「文档仓库」Tab。</p>
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

              <PrepRunTimeline
                trace={timelineTrace || null}
                stageStatus={liveStageStatus}
                stageOutput={liveStageOutput}
                isStreaming={isGenerating}
              />
              <QuestionList questions={session?.questions || []} />
              {session ? (
                <div className="space-y-3">
                  {isSubmitted ? (
                    <p className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700 dark:text-emerald-300">
                      当前会话已提交，答案已锁定为正式提交状态。
                    </p>
                  ) : null}
                  <AnswerDraftEditor
                    sessionId={session.id}
                    questions={session.questions}
                    initialAnswers={session.answers}
                    readOnly={isSubmitted}
                    onDraftAnswersChange={setDraftAnswers}
                  />
                  <div className="flex justify-end">
                    <SubmitAnswersButton
                      disabled={isGenerating || totalQuestions === 0 || answeredCount === 0}
                      isSubmitted={isSubmitted}
                      answeredCount={answeredCount}
                      totalQuestions={totalQuestions}
                      onSubmit={handleSubmitAnswers}
                    />
                  </div>
                </div>
              ) : null}
            </div>
          </TabsContent>

          <TabsContent value="review">
            {!session ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">复盘</CardTitle>
                  <CardDescription>先生成并提交一轮答案，再查看评分。</CardDescription>
                </CardHeader>
              </Card>
            ) : !isSubmitted ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">复盘</CardTitle>
                  <CardDescription>当前会话仍是草稿状态，提交答案后会生成评分。</CardDescription>
                </CardHeader>
              </Card>
            ) : !session.evaluation ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">复盘</CardTitle>
                  <CardDescription>已提交，但暂时没有评分数据。</CardDescription>
                </CardHeader>
              </Card>
            ) : evaluationStatus === "pending" || evaluationStatus === "running" ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">复盘</CardTitle>
                  <CardDescription>评分任务正在后台执行，页面会自动刷新结果。</CardDescription>
                </CardHeader>
              </Card>
            ) : evaluationStatus === "failed" ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">复盘</CardTitle>
                  <CardDescription className="text-destructive">
                    评分失败：{session.evaluation.error || "未知错误"}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <Button type="button" onClick={handleRetryEvaluation} disabled={isRetryingEvaluation}>
                    {isRetryingEvaluation ? "重试中..." : "重试评分"}
                  </Button>
                </CardContent>
              </Card>
            ) : (
              <div className="space-y-4">
                <ReviewSummaryCard evaluation={session.evaluation} />
                <WeakPointList weakPoints={session.evaluation.overall.weakPoints} />
                <div className="space-y-3">
                  {session.questions.map((question) => (
                    <QuestionScoreCard
                      key={`score_${question.id}`}
                      question={question}
                      answer={answerByQuestionID.get(question.id) || ""}
                      score={scoreByQuestionID.get(question.id)}
                      referenceAction={
                        <ReferenceAnswerDrawer
                          sessionId={session.id}
                          question={question}
                          cachedReferenceAnswer={referenceAnswerByQuestionID.get(question.id)}
                          onGenerated={(generated) => handleReferenceAnswerGenerated(question.id, generated)}
                        />
                      }
                    />
                  ))}
                </div>
              </div>
            )}
          </TabsContent>
        </Tabs>
      </section>
    </main>
  );
}
