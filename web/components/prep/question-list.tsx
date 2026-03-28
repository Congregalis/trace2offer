import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PrepQuestion } from "@/lib/prep-types";

export function QuestionList({ questions }: { questions: PrepQuestion[] }) {
  if (questions.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">题目列表</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">还没有题目，先点击“生成题目”。</CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">题目列表（{questions.length}）</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {questions.map((question) => (
          <div key={question.id} className="rounded-lg border border-border p-4">
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <Badge variant="secondary">Q{question.id}</Badge>
              <Badge variant="outline">{question.type || "technical"}</Badge>
            </div>
            <p className="text-sm font-medium text-foreground">{question.content}</p>
            {question.expectedPoints.length > 0 ? (
              <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-muted-foreground">
                {question.expectedPoints.map((point) => (
                  <li key={`${question.id}_${point}`}>{point}</li>
                ))}
              </ul>
            ) : null}
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
