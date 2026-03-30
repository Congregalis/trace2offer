import { PrepQuestionScoreSource } from "@/lib/prep-types";

interface ReferenceSourceListProps {
  sources: PrepQuestionScoreSource[];
}

export function ReferenceSourceList({ sources }: ReferenceSourceListProps) {
  if (sources.length === 0) {
    return <p className="text-sm text-muted-foreground">本题未命中可展示的参考来源。</p>;
  }

  return (
    <ul className="space-y-2 text-sm">
      {sources.map((source) => (
        <li key={`${source.title}_${source.score}`} className="rounded-md border border-border/70 px-3 py-2">
          <p className="font-medium">{source.title}</p>
          <p className="text-xs text-muted-foreground">相关度: {source.score.toFixed(2)}</p>
        </li>
      ))}
    </ul>
  );
}
