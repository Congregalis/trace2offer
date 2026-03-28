import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { PrepLeadContextPreview } from "@/lib/prep-types";
import { ContextSourceList } from "./context-source-list";

interface PrepContextPreviewCardProps {
  preview: PrepLeadContextPreview | null;
  isLoading?: boolean;
  error?: string | null;
}

export function PrepContextPreviewCard({ preview, isLoading = false, error = null }: PrepContextPreviewCardProps) {
  return (
    <Card>
      <CardHeader className="space-y-3">
        <CardTitle className="text-base">上下文预览</CardTitle>
        {isLoading ? (
          <div className="grid gap-2 sm:grid-cols-2">
            <Skeleton className="h-5 w-full" />
            <Skeleton className="h-5 w-full" />
          </div>
        ) : preview ? (
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <Badge variant="secondary">{preview.company || "-"}</Badge>
            <Badge variant="outline">{preview.position || "-"}</Badge>
            <Badge variant={preview.hasResume ? "secondary" : "outline"}>{preview.hasResume ? "简历已就绪" : "无简历"}</Badge>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">选择线索后展示可用上下文。</p>
        )}
      </CardHeader>

      <CardContent className="space-y-6">
        {error ? <p className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p> : null}

        <section className="space-y-2">
          <h3 className="text-sm font-medium">来源清单</h3>
          {isLoading ? <Skeleton className="h-20 w-full" /> : <ContextSourceList sources={preview?.sources || []} />}
        </section>
      </CardContent>
    </Card>
  );
}
