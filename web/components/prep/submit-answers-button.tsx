"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { SubmitConfirmDialog } from "./submit-confirm-dialog";

interface SubmitAnswersButtonProps {
  disabled?: boolean;
  isSubmitted?: boolean;
  answeredCount: number;
  totalQuestions: number;
  onSubmit: () => Promise<void>;
}

export function SubmitAnswersButton({
  disabled = false,
  isSubmitted = false,
  answeredCount,
  totalQuestions,
  onSubmit,
}: SubmitAnswersButtonProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  async function handleConfirm() {
    setIsSubmitting(true);
    setSubmitError(null);
    try {
      await onSubmit();
      setIsOpen(false);
    } catch (error: unknown) {
      const message = error instanceof Error && error.message ? error.message : "提交失败";
      setSubmitError(message);
    } finally {
      setIsSubmitting(false);
    }
  }

  const isDisabled = disabled || isSubmitted;

  return (
    <>
      <Button type="button" onClick={() => setIsOpen(true)} disabled={isDisabled}>
        {isSubmitted ? "已提交" : "提交答案"}
      </Button>
      <SubmitConfirmDialog
        open={isOpen}
        onOpenChange={setIsOpen}
        answeredCount={answeredCount}
        totalQuestions={totalQuestions}
        isSubmitting={isSubmitting}
        error={submitError}
        onConfirm={handleConfirm}
      />
    </>
  );
}
