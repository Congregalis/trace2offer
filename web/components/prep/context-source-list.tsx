import { Badge } from "@/components/ui/badge";
import { PrepContextSource } from "@/lib/prep-types";

interface ContextSourceListProps {
  sources: PrepContextSource[];
}

export function ContextSourceList({ sources }: ContextSourceListProps) {
  if (!sources.length) {
    return <p className="text-sm text-muted-foreground">当前暂无可用来源。</p>;
  }

  return (
    <ul className="space-y-2">
      {sources.map((source, index) => (
        <li
          key={`${source.scope}-${source.kind}-${source.title}-${index}`}
          className="flex items-center justify-between gap-3 rounded-md border border-border/70 bg-muted/20 px-3 py-2"
        >
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">{source.title || "-"}</p>
            <p className="text-xs text-muted-foreground">{source.kind || "-"}</p>
          </div>
          <Badge variant="secondary" className="shrink-0">
            {source.scope || "-"}
          </Badge>
        </li>
      ))}
    </ul>
  );
}
