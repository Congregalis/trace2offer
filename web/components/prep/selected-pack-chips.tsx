import { Badge } from "@/components/ui/badge";

interface SelectedPackChipsProps {
  topicKeys: string[];
}

export function SelectedPackChips({ topicKeys }: SelectedPackChipsProps) {
  if (!topicKeys.length) {
    return <p className="text-sm text-muted-foreground">未选择 Topic Pack。</p>;
  }

  return (
    <div className="flex flex-wrap gap-2">
      {topicKeys.map((key) => (
        <Badge key={key} variant="outline">
          {key}
        </Badge>
      ))}
    </div>
  );
}
