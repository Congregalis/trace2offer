import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface WeakPointListProps {
  weakPoints: string[];
}

export function WeakPointList({ weakPoints }: WeakPointListProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">待改进重点</CardTitle>
      </CardHeader>
      <CardContent>
        {weakPoints.length === 0 ? (
          <p className="text-sm text-muted-foreground">当前没有明显弱项。</p>
        ) : (
          <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
            {weakPoints.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
