import { Badge } from "@/components/ui/badge";
import { PrepRetrievalTrace } from "@/lib/prep-types";

interface RetrievalStageTimelineProps {
  trace?: PrepRetrievalTrace;
}

export function RetrievalStageTimeline({ trace }: RetrievalStageTimelineProps) {
  if (!trace) {
    return <p className="text-sm text-muted-foreground">当前未返回检索链路 trace。</p>;
  }

  return (
    <ol className="space-y-2">
      <li className="rounded-md border border-border/70 bg-muted/20 p-3">
        <p className="text-sm font-medium">1) Query Normalization</p>
        <p className="text-xs text-muted-foreground">{trace.stageQueryNormalization.method}</p>
        <p className="mt-1 text-xs">{trace.stageQueryNormalization.output || "-"}</p>
      </li>
      <li className="rounded-md border border-border/70 bg-muted/20 p-3">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-medium">2) Initial Retrieval</p>
          <Badge variant="secondary">{trace.stageInitialRetrieval.outputCount ?? 0}</Badge>
        </div>
        <p className="text-xs text-muted-foreground">{trace.stageInitialRetrieval.method}</p>
      </li>
      <li className="rounded-md border border-border/70 bg-muted/20 p-3">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-medium">3) Deduplication</p>
          <Badge variant="outline">
            {trace.stageDeduplication.inputCount ?? 0} → {trace.stageDeduplication.outputCount ?? 0}
          </Badge>
        </div>
        <p className="text-xs text-muted-foreground">{trace.stageDeduplication.method}</p>
      </li>
      <li className="rounded-md border border-border/70 bg-muted/20 p-3">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-medium">4) Reranking</p>
          <Badge variant="outline">
            {trace.stageReranking.inputCount ?? 0} → {trace.stageReranking.outputCount ?? 0}
          </Badge>
        </div>
        <p className="text-xs text-muted-foreground">{trace.stageReranking.method}</p>
      </li>
    </ol>
  );
}
