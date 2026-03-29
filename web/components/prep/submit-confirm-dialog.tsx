"use client";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";

interface SubmitConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  answeredCount: number;
  totalQuestions: number;
  isSubmitting: boolean;
  error: string | null;
  onConfirm: () => void;
}

export function SubmitConfirmDialog({
  open,
  onOpenChange,
  answeredCount,
  totalQuestions,
  isSubmitting,
  error,
  onConfirm,
}: SubmitConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>确认提交答案？</DialogTitle>
          <DialogDescription>
            提交后会话状态会从 `draft` 变为 `submitted`，并记录本次提交时间。当前已作答 {answeredCount}/{totalQuestions} 题。
          </DialogDescription>
        </DialogHeader>

        {error ? <p className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p> : null}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
            取消
          </Button>
          <Button type="button" onClick={onConfirm} disabled={isSubmitting}>
            {isSubmitting ? "提交中..." : "确认提交"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
