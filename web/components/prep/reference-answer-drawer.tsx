"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { generatePrepReferenceAnswer } from "@/lib/prep-api";
import { PrepQuestion, PrepReferenceAnswer } from "@/lib/prep-types";
import { ReferenceSourceList } from "./reference-source-list";

interface ReferenceAnswerDrawerProps {
  sessionId: string;
  question: PrepQuestion;
  cachedReferenceAnswer?: PrepReferenceAnswer;
  disabled?: boolean;
  onGenerated?: (answer: PrepReferenceAnswer) => void;
}

export function ReferenceAnswerDrawer({
  sessionId,
  question,
  cachedReferenceAnswer,
  disabled = false,
  onGenerated,
}: ReferenceAnswerDrawerProps) {
  const [open, setOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [referenceAnswer, setReferenceAnswer] = useState<PrepReferenceAnswer | undefined>(cachedReferenceAnswer);

  const normalizedSessionID = useMemo(() => (sessionId || "").trim(), [sessionId]);

  useEffect(() => {
    setReferenceAnswer(cachedReferenceAnswer);
  }, [cachedReferenceAnswer]);

  const handleGenerate = useCallback(async () => {
    if (!normalizedSessionID) {
      setError("session_id is required");
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const generated = await generatePrepReferenceAnswer(normalizedSessionID, question.id);
      setReferenceAnswer(generated);
      onGenerated?.(generated);
    } catch (generationError: unknown) {
      const message = generationError instanceof Error && generationError.message ? generationError.message : "生成参考答案失败";
      setError(message);
    } finally {
      setIsLoading(false);
    }
  }, [normalizedSessionID, onGenerated, question.id]);

  useEffect(() => {
    if (!open) {
      return;
    }
    if (referenceAnswer?.referenceAnswer) {
      return;
    }
    void handleGenerate();
  }, [handleGenerate, open, referenceAnswer?.referenceAnswer]);

  return (
    <>
      <Button type="button" variant="outline" size="sm" onClick={() => setOpen(true)} disabled={disabled}>
        参考答案
      </Button>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="right" className="w-full sm:max-w-xl">
          <SheetHeader>
            <SheetTitle>Q{question.id} 参考答案</SheetTitle>
            <SheetDescription>基于当前题目和资料检索生成，可用于对照复盘。</SheetDescription>
          </SheetHeader>
          <div className="flex h-full flex-col gap-4 overflow-y-auto px-4 pb-6">
            <div className="rounded-md border border-border/70 bg-secondary/20 p-3">
              <p className="text-xs text-muted-foreground">题目</p>
              <p className="mt-1 whitespace-pre-wrap text-sm">{question.content}</p>
            </div>

            {error ? <p className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p> : null}

            {referenceAnswer?.referenceAnswer ? (
              <>
                <div className="rounded-md border border-border/70 p-3">
                  <p className="mb-1 text-xs text-muted-foreground">参考答案</p>
                  <p className="whitespace-pre-wrap text-sm">{referenceAnswer.referenceAnswer}</p>
                </div>
                <div className="space-y-2">
                  <p className="text-xs text-muted-foreground">参考来源</p>
                  <ReferenceSourceList sources={referenceAnswer.sources} />
                </div>
              </>
            ) : (
              <p className="text-sm text-muted-foreground">{isLoading ? "参考答案生成中..." : "暂无参考答案"}</p>
            )}

            <div className="mt-auto">
              <Button type="button" onClick={handleGenerate} disabled={isLoading}>
                {isLoading ? "生成中..." : referenceAnswer?.referenceAnswer ? "重新生成" : "生成参考答案"}
              </Button>
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}
