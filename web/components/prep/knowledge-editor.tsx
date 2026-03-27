"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { PrepKnowledgeDocument } from "@/lib/prep-types";

export interface KnowledgeEditorProps {
  document: PrepKnowledgeDocument | null;
  draftFilename: string;
  draftContent: string;
  onFilenameChange: (value: string) => void;
  onContentChange: (value: string) => void;
  onSave: () => void | Promise<void>;
  onReset: () => void;
  isSaving?: boolean;
}

export function KnowledgeEditor({
  document,
  draftFilename,
  draftContent,
  onFilenameChange,
  onContentChange,
  onSave,
  onReset,
  isSaving = false,
}: KnowledgeEditorProps) {
  const isEditing = Boolean(document);

  return (
    <div className="space-y-3">
      <Input
        value={draftFilename}
        onChange={(event) => onFilenameChange(event.target.value)}
        placeholder="文件名（不带 .md 也行）"
        disabled={isSaving || isEditing}
      />
      <Textarea
        value={draftContent}
        onChange={(event) => onContentChange(event.target.value)}
        placeholder="写 Markdown 内容"
        className="min-h-[280px]"
        disabled={isSaving}
      />
      <div className="flex items-center justify-end gap-2">
        <Button type="button" variant="outline" disabled={isSaving} onClick={() => onReset()}>
          重置
        </Button>
        <Button type="button" disabled={isSaving} onClick={() => void onSave()}>
          {isEditing ? "保存修改" : "创建文档"}
        </Button>
      </div>
    </div>
  );
}
