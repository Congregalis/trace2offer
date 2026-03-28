import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PrepRetrievalPreview } from "@/lib/prep-types";
import { MatchedSourceCard } from "./matched-source-card";
import { PromptContextPreview } from "./prompt-context-preview";
import { RetrievalStageTimeline } from "./retrieval-stage-timeline";

interface RetrievalTracePanelProps {
  preview: PrepRetrievalPreview | null;
}

export function RetrievalTracePanel({ preview }: RetrievalTracePanelProps) {
  if (!preview) {
    return <p className="text-sm text-muted-foreground">先执行一次 retrieval preview，再看可解释链路。</p>;
  }

  return (
    <section className="space-y-4">
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Retrieval Trace</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <p className="text-sm">query: {preview.query || "-"}</p>
          <p className="text-sm text-muted-foreground">normalized: {preview.normalizedQuery || "-"}</p>
          <RetrievalStageTimeline trace={preview.trace} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">候选召回（初始）</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {preview.candidateChunks.length === 0 ? (
            <p className="text-sm text-muted-foreground">当前没有候选召回结果。</p>
          ) : (
            preview.candidateChunks.map((chunk) => <MatchedSourceCard key={`candidate-${chunk.id}`} chunk={chunk} />)
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">最终上下文（进入 Prompt）</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {preview.retrievedChunks.length === 0 ? (
            <p className="text-sm text-muted-foreground">当前检索结果为空。</p>
          ) : (
            preview.retrievedChunks.map((chunk) => <MatchedSourceCard key={`final-${chunk.id}`} chunk={chunk} />)
          )}
        </CardContent>
      </Card>

      <PromptContextPreview finalContext={preview.finalContext} />
    </section>
  );
}
