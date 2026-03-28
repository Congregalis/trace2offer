import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PrepGenerationTrace } from "@/lib/prep-types";

interface PrepRunTimelineProps {
  trace: PrepGenerationTrace | null;
}

export function PrepRunTimeline({ trace }: PrepRunTimelineProps) {
  if (!trace) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">RAG 过程</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">生成题目后会展示检索与提示词阶段摘要。</CardContent>
      </Card>
    );
  }

  const taskSection = trace.promptSections.find((item) => item.title.toLowerCase() === "task");

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">RAG 过程时间线</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        <div className="rounded-md border border-border p-3">
          <p className="font-medium">1) 输入配置快照</p>
          <p className="text-muted-foreground">
            lead_id={trace.inputSnapshot.leadId || "-"}，topics={trace.inputSnapshot.topicKeys.join(", ") || "-"}，question_count=
            {trace.inputSnapshot.questionCount}
          </p>
        </div>
        <div className="rounded-md border border-border p-3">
          <p className="font-medium">2) 检索执行</p>
          <p className="text-muted-foreground">query: {trace.retrievalQuery || "-"}</p>
          <p className="text-muted-foreground">
            candidates={trace.retrievalResults.candidatesFound} → selected={trace.retrievalResults.finalSelected}
          </p>
        </div>
        <div className="rounded-md border border-border p-3">
          <p className="font-medium">3) Prompt 组装</p>
          <p className="text-muted-foreground">{taskSection?.content || "-"}</p>
        </div>
        <div className="rounded-md border border-border p-3">
          <p className="font-medium">4) 生成结果</p>
          <p className="text-muted-foreground">
            model={trace.generationResult.model || "-"}，questions={trace.generationResult.questionsGenerated}，time=
            {trace.generationResult.generationTimeMs}ms
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
