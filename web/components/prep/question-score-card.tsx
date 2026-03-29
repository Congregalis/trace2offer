import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PrepQuestion, PrepQuestionScore } from "@/lib/prep-types";

interface QuestionScoreCardProps {
  question: PrepQuestion;
  answer?: string;
  score?: PrepQuestionScore;
}

export function QuestionScoreCard({ question, answer, score }: QuestionScoreCardProps) {
  if (!score) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Q{question.id}</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">该题暂无评分数据。</CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between gap-2 text-base">
          <span>Q{question.id}</span>
          <Badge variant={score.answered ? "secondary" : "outline"}>{score.score.toFixed(1)} / 10</Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        <p className="font-medium">{question.content}</p>
        <div className="rounded-md border border-border bg-secondary/20 p-2">
          <p className="mb-1 text-xs text-muted-foreground">你的答案</p>
          <p className="whitespace-pre-wrap text-sm">{(answer || "").trim() || "（未作答）"}</p>
        </div>
        <p className="text-muted-foreground">{score.summary}</p>
        {score.strengths.length > 0 ? (
          <ul className="list-disc space-y-1 pl-5 text-muted-foreground">
            {score.strengths.map((item) => (
              <li key={`${question.id}_strength_${item}`}>优点：{item}</li>
            ))}
          </ul>
        ) : null}
        {score.improvements.length > 0 ? (
          <ul className="list-disc space-y-1 pl-5 text-muted-foreground">
            {score.improvements.map((item) => (
              <li key={`${question.id}_improvement_${item}`}>改进：{item}</li>
            ))}
          </ul>
        ) : null}
      </CardContent>
    </Card>
  );
}
