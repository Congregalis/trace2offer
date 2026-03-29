import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PrepEvaluation } from "@/lib/prep-types";

interface ReviewSummaryCardProps {
  evaluation: PrepEvaluation;
}

export function ReviewSummaryCard({ evaluation }: ReviewSummaryCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">复盘总览</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        <p>
          平均分：<span className="font-semibold">{evaluation.overall.averageScore.toFixed(1)}</span> / 10
        </p>
        <p>
          作答题数：{evaluation.overall.answeredCount} / {evaluation.overall.totalQuestions}
        </p>
        <p className="text-muted-foreground">{evaluation.overall.summary || evaluation.summary || "暂无总体总结。"}</p>
      </CardContent>
    </Card>
  );
}
