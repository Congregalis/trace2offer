import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PrepRetrievalFinalContext } from "@/lib/prep-types";

interface PromptContextPreviewProps {
  finalContext: PrepRetrievalFinalContext;
}

export function PromptContextPreview({ finalContext }: PromptContextPreviewProps) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">Final Prompt Context</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <p className="text-xs text-muted-foreground">
          chunks: {finalContext.chunksUsed} · tokens: {finalContext.totalTokens}
        </p>
        <pre className="max-h-72 overflow-auto whitespace-pre-wrap rounded-md border border-border/70 bg-muted/20 p-3 text-xs">
          {finalContext.context || "(empty)"}
        </pre>
      </CardContent>
    </Card>
  );
}
