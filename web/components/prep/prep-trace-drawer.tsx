import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { PrepGenerationTrace } from "@/lib/prep-types";

export function PrepTraceDrawer({ trace }: { trace: PrepGenerationTrace | null }) {
  if (!trace) {
    return null;
  }

  const sectionContent = (title: string) =>
    trace.promptSections.find((item) => item.title.toLowerCase() === title.toLowerCase())?.content || "-";

  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          查看 Trace 详情
        </Button>
      </SheetTrigger>
      <SheetContent className="w-full sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle>生成 Trace 详情</SheetTitle>
          <SheetDescription>用于复盘本次 RAG 出题链路。</SheetDescription>
        </SheetHeader>

        <div className="mt-4 space-y-4 overflow-auto pr-1 text-sm">
          <section className="space-y-1">
            <h3 className="font-medium">检索命中来源</h3>
            <ul className="list-disc space-y-1 pl-5 text-muted-foreground">
              {trace.retrievalResults.sources.length === 0 ? <li>无</li> : trace.retrievalResults.sources.map((item) => <li key={item}>{item}</li>)}
            </ul>
          </section>

          <section className="space-y-1">
            <h3 className="font-medium">System Prompt</h3>
            <p className="whitespace-pre-wrap rounded-md border border-border bg-secondary/30 p-2 text-xs">{sectionContent("System")}</p>
          </section>

          <section className="space-y-1">
            <h3 className="font-medium">Context Section</h3>
            <p className="whitespace-pre-wrap rounded-md border border-border bg-secondary/30 p-2 text-xs">{sectionContent("Context")}</p>
          </section>

          <section className="space-y-1">
            <h3 className="font-medium">Task Instruction</h3>
            <p className="whitespace-pre-wrap rounded-md border border-border bg-secondary/30 p-2 text-xs">{sectionContent("Task")}</p>
          </section>
        </div>
      </SheetContent>
    </Sheet>
  );
}
