import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { PrepGenerationTrace } from "@/lib/prep-types";

interface PrepRunTimelineProps {
  trace: PrepGenerationTrace | null;
  stageStatus?: Record<string, string>;
  stageOutput?: Record<string, string>;
  isStreaming?: boolean;
}

function statusText(status: string): string {
  if (status === "completed") {
    return "已完成";
  }
  if (status === "progress") {
    return "流式输出中";
  }
  if (status === "started") {
    return "进行中";
  }
  return "";
}

export function PrepRunTimeline({ trace, stageStatus, stageOutput, isStreaming = false }: PrepRunTimelineProps) {
  const stageState = (stage: string) => (stageStatus?.[stage] || "").trim();
  const hasStage = (stage: string) => stageState(stage).length > 0;

  const hasInputData = Boolean(
    trace &&
      ((trace.inputSnapshot.leadId || "").trim() || trace.inputSnapshot.topicKeys.length > 0 || trace.inputSnapshot.questionCount > 0),
  );
  const hasQueryData = Boolean(trace && ((trace.queryPlanning.finalQuery || "").trim() || (trace.retrievalQuery || "").trim()));
  const hasRetrievalData = Boolean(trace && ((trace.retrievalQuery || "").trim() || trace.retrievalResults.candidatesFound > 0));
  const hasPromptData = Boolean(trace && (trace.assembledPrompt || "").trim());
  const hasGenerationData = Boolean(trace && (trace.generationResult.model || "").trim());

  const showInput = hasInputData || hasStage("input_snapshot");
  const showQuery = hasQueryData || hasStage("query_planning");
  const showRetrieval = hasRetrievalData || hasStage("retrieval");
  const showPrompt = hasPromptData || hasStage("prompt_assembly");
  const showGeneration = hasGenerationData || hasStage("generation");

  if (!showInput && !showQuery && !showRetrieval && !showPrompt && !showGeneration) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">RAG 过程</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">生成题目后会展示检索与提示词阶段摘要。</CardContent>
      </Card>
    );
  }

  const promptByTitle = new Map((trace?.promptSections || []).map((section) => [section.title.toLowerCase(), section.content]));
  const taskSection = promptByTitle.get("task") || "";
  const planningOutput = (stageOutput?.query_planning || "").trim() || (trace?.queryPlanning.rawOutput || "").trim();
  const generationOutput = (stageOutput?.generation || "").trim();
  const assembledPrompt =
    trace?.assembledPrompt ||
    [
      promptByTitle.get("system") || "",
      promptByTitle.get("context") || "",
      promptByTitle.get("candidate_profile") || "",
      promptByTitle.get("job_description") || "",
      promptByTitle.get("task") || "",
      promptByTitle.get("requirements") || "",
      promptByTitle.get("output_format") || "",
    ]
      .filter((item) => item.trim().length > 0)
      .join("\n\n");

  const renderStageBadge = (stage: string) => {
    const status = stageState(stage);
    const text = statusText(status);
    if (!text) {
      return null;
    }
    return <Badge variant={status === "completed" ? "secondary" : "outline"}>{text}</Badge>;
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">RAG 过程时间线</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        {showInput ? (
          <div className="rounded-md border border-border p-3">
            <div className="flex items-center justify-between gap-2">
              <p className="font-medium">1) 输入配置快照</p>
              {renderStageBadge("input_snapshot")}
            </div>
            <p className="text-muted-foreground">
              lead_id={trace?.inputSnapshot.leadId || "-"}，topics={trace?.inputSnapshot.topicKeys.join(", ") || "-"}，question_count=
              {trace?.inputSnapshot.questionCount || 0}
            </p>
          </div>
        ) : null}

        {showQuery ? (
          <div className="rounded-md border border-border p-3">
            <div className="flex items-center justify-between gap-2">
              <p className="font-medium">2) Query 规划（Agent）</p>
              {renderStageBadge("query_planning")}
            </div>
            <p className="text-muted-foreground">
              strategy={trace?.queryPlanning.strategy || "-"}，model={trace?.queryPlanning.model || "-"}
            </p>
            <p className="text-muted-foreground">query: {trace?.queryPlanning.finalQuery || trace?.retrievalQuery || "-"}</p>
            {trace?.queryPlanning.resumeExcerpt ? <p className="text-muted-foreground">resume 摘要: {trace.queryPlanning.resumeExcerpt}</p> : null}
            {trace?.queryPlanning.jdExcerpt ? <p className="text-muted-foreground">JD 摘要: {trace.queryPlanning.jdExcerpt}</p> : null}
            {planningOutput ? (
              <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap rounded-md border border-border bg-secondary/30 p-2 text-xs">
                {planningOutput}
              </pre>
            ) : null}
          </div>
        ) : null}

        {showRetrieval ? (
          <div className="rounded-md border border-border p-3">
            <div className="flex items-center justify-between gap-2">
              <p className="font-medium">3) 检索执行</p>
              {renderStageBadge("retrieval")}
            </div>
            <p className="text-muted-foreground">query: {trace?.retrievalQuery || "-"}</p>
            <p className="text-muted-foreground">
              candidates={trace?.retrievalResults.candidatesFound || 0} → selected={trace?.retrievalResults.finalSelected || 0}
            </p>
          </div>
        ) : null}

        {showPrompt ? (
          <div className="rounded-md border border-border p-3">
            <div className="flex items-center justify-between gap-2">
              <p className="font-medium">4) Prompt 组装</p>
              {renderStageBadge("prompt_assembly")}
            </div>
            <p className="text-muted-foreground">{taskSection || "-"}</p>
            <Dialog>
              <DialogTrigger asChild>
                <Button type="button" variant="outline" size="sm" className="mt-2" disabled={!assembledPrompt}>
                  查看完整 Prompt
                </Button>
              </DialogTrigger>
              <DialogContent className="max-h-[80vh] overflow-hidden sm:max-w-3xl">
                <DialogHeader>
                  <DialogTitle>Prompt 详情</DialogTitle>
                  <DialogDescription>展示本次生成问题时组装出的完整 prompt。</DialogDescription>
                </DialogHeader>
                <pre className="max-h-[60vh] overflow-auto whitespace-pre-wrap rounded-md border border-border bg-secondary/30 p-3 text-xs">
                  {assembledPrompt || "-"}
                </pre>
              </DialogContent>
            </Dialog>
          </div>
        ) : null}

        {showGeneration ? (
          <div className="rounded-md border border-border p-3">
            <div className="flex items-center justify-between gap-2">
              <p className="font-medium">5) 生成结果</p>
              {renderStageBadge("generation")}
            </div>
            <p className="text-muted-foreground">
              model={trace?.generationResult.model || "-"}，questions={trace?.generationResult.questionsGenerated || 0}，time=
              {trace?.generationResult.generationTimeMs || 0}ms
            </p>
            {generationOutput ? (
              <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap rounded-md border border-border bg-secondary/30 p-2 text-xs">
                {generationOutput}
              </pre>
            ) : null}
          </div>
        ) : null}

        {!trace && isStreaming ? <p className="text-xs text-muted-foreground">正在等待首个阶段事件...</p> : null}
      </CardContent>
    </Card>
  );
}
