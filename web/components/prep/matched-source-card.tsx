import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PrepRetrievedChunk } from "@/lib/prep-types";

interface MatchedSourceCardProps {
  chunk: PrepRetrievedChunk;
}

export function MatchedSourceCard({ chunk }: MatchedSourceCardProps) {
  return (
    <Card>
      <CardHeader className="space-y-2 pb-3">
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle className="text-sm">{chunk.id || "-"}</CardTitle>
          <Badge variant="secondary">score: {chunk.score.toFixed(3)}</Badge>
          <Badge variant="outline">{chunk.source.scope}</Badge>
        </div>
        <p className="text-xs text-muted-foreground">
          {chunk.source.scopeId}/{chunk.source.documentTitle} · chunk #{chunk.source.chunkIndex}
        </p>
      </CardHeader>
      <CardContent className="space-y-2">
        <p className="whitespace-pre-wrap text-sm">{chunk.content || "-"}</p>
        <p className="text-xs text-muted-foreground">{chunk.whySelected || ""}</p>
      </CardContent>
    </Card>
  );
}
