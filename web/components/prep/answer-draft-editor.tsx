"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Textarea } from "@/components/ui/textarea";
import { fetchPrepSession, savePrepDraftAnswers } from "@/lib/prep-api";
import { PrepAnswer, PrepQuestion } from "@/lib/prep-types";

type SaveStatus = "idle" | "unsaved" | "saving" | "saved" | "error";

export interface AnswerDraftEditorProps {
  sessionId: string;
  questions: PrepQuestion[];
  initialAnswers: PrepAnswer[];
}

function toAnswerMap(answers: PrepAnswer[]): Record<number, string> {
  const mapped: Record<number, string> = {};
  for (const answer of answers) {
    if (answer.questionId <= 0) {
      continue;
    }
    mapped[answer.questionId] = answer.answer || "";
  }
  return mapped;
}

type DebouncedFunction<TArgs extends unknown[]> = ((...args: TArgs) => void) & { cancel: () => void };

function debounce<TArgs extends unknown[]>(callback: (...args: TArgs) => void, wait: number): DebouncedFunction<TArgs> {
  let timer: number | undefined;
  const wrapped = (...args: TArgs) => {
    if (typeof timer === "number") {
      window.clearTimeout(timer);
    }
    timer = window.setTimeout(() => {
      timer = undefined;
      callback(...args);
    }, wait);
  };
  wrapped.cancel = () => {
    if (typeof timer === "number") {
      window.clearTimeout(timer);
      timer = undefined;
    }
  };
  return wrapped;
}

export function AnswerDraftEditor({ sessionId, questions, initialAnswers }: AnswerDraftEditorProps) {
  const [answerMap, setAnswerMap] = useState<Record<number, string>>(() => toAnswerMap(initialAnswers));
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saveStatus, setSaveStatus] = useState<SaveStatus>("idle");
  const [lastSavedAt, setLastSavedAt] = useState<string>("");

  const hydratedRef = useRef(false);

  const normalizedSessionID = sessionId.trim();

  const debouncedSave = useMemo(() => {
    return debounce((answers: PrepAnswer[]) => {
      setSaveStatus("saving");
      setError(null);

      void savePrepDraftAnswers(normalizedSessionID, answers)
        .then((result) => {
          setLastSavedAt(result.savedAt);
          setSaveStatus("saved");
        })
        .catch((saveError: unknown) => {
          const message = saveError instanceof Error && saveError.message ? saveError.message : "保存草稿失败";
          setError(message);
          setSaveStatus("error");
        });
    }, 1000);
  }, [normalizedSessionID]);

  useEffect(() => {
    return () => {
      debouncedSave.cancel();
    };
  }, [debouncedSave]);

  useEffect(() => {
    setAnswerMap(toAnswerMap(initialAnswers));
    setSaveStatus("idle");
  }, [initialAnswers]);

  useEffect(() => {
    if (!normalizedSessionID) {
      setAnswerMap({});
      setIsLoading(false);
      setError("session_id is required");
      return;
    }

    const controller = new AbortController();
    hydratedRef.current = false;
    setIsLoading(true);
    setError(null);

    void fetchPrepSession(normalizedSessionID, controller.signal)
      .then((session) => {
        setAnswerMap(toAnswerMap(session.answers));
        setLastSavedAt(session.updatedAt);
        setSaveStatus("idle");
        hydratedRef.current = true;
      })
      .catch((loadError: unknown) => {
        if (loadError instanceof Error && loadError.name === "AbortError") {
          return;
        }
        const message = loadError instanceof Error && loadError.message ? loadError.message : "加载会话失败";
        setError(message);
        setSaveStatus("error");
      })
      .finally(() => {
        setIsLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, [normalizedSessionID]);

  useEffect(() => {
    if (!hydratedRef.current) {
      return;
    }
    if (saveStatus !== "unsaved") {
      return;
    }

    debouncedSave(
      questions.map((question) => ({
        questionId: question.id,
        answer: answerMap[question.id] || "",
      })),
    );
  }, [answerMap, debouncedSave, questions, saveStatus]);

  function updateAnswer(questionID: number, value: string) {
    setAnswerMap((previous) => ({
      ...previous,
      [questionID]: value,
    }));
    setSaveStatus("unsaved");
  }

  function renderSaveHint() {
    if (saveStatus === "saving") {
      return "自动保存中...";
    }
    if (saveStatus === "saved") {
      return lastSavedAt ? `已保存 ${lastSavedAt}` : "已保存";
    }
    if (saveStatus === "unsaved") {
      return "有未保存草稿";
    }
    if (saveStatus === "error") {
      return "保存失败，请稍后重试";
    }
    return "";
  }

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">加载题目与草稿中...</p>;
  }

  return (
    <section className="space-y-4">
      {renderSaveHint() ? <p className="text-sm text-muted-foreground">{renderSaveHint()}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {questions.length === 0 ? (
        <p className="text-sm text-muted-foreground">当前会话暂无题目，无法编辑答案草稿。</p>
      ) : (
        <div className="space-y-4">
          {questions.map((question, index) => (
            <article key={question.id} className="space-y-2 rounded-xl border border-border/70 p-3">
              <h3 className="text-sm font-medium">
                Q{index + 1}. {question.content || "未命名题目"}
              </h3>
              <Textarea
                data-testid={`prep-answer-input-${question.id}`}
                value={answerMap[question.id] || ""}
                onChange={(event) => updateAnswer(question.id, event.target.value)}
                className="min-h-[140px]"
                placeholder="请输入你的答案草稿，停止输入 1 秒后自动保存"
              />
            </article>
          ))}
        </div>
      )}
    </section>
  );
}
