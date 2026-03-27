"use client";

import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { PrepKnowledgeDocument } from "@/lib/prep-types";
import { FileText, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";

export interface KnowledgeDocumentListProps {
  documents: PrepKnowledgeDocument[];
  selectedFilename: string;
  onSelect: (filename: string) => void;
  onDelete: (filename: string) => void;
  isLoading?: boolean;
  isMutating?: boolean;
}

export function KnowledgeDocumentList({
  documents,
  selectedFilename,
  onSelect,
  onDelete,
  isLoading = false,
  isMutating = false,
}: KnowledgeDocumentListProps) {
  if (documents.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <FileText />
          </EmptyMedia>
          <EmptyTitle>还没有资料文档</EmptyTitle>
          <EmptyDescription>先创建一份 Markdown 文档，把你要背的内容塞进来。</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div className="space-y-2">
      {documents.map((document) => {
        const selected = selectedFilename === document.filename;
        return (
          <div
            key={document.filename}
            className={cn(
              "flex items-center gap-2 rounded-md border p-2",
              selected ? "border-primary/60 bg-primary/5" : "border-border",
            )}
          >
            <button
              type="button"
              className="flex min-w-0 flex-1 items-center gap-2 text-left"
              disabled={isLoading || isMutating}
              onClick={() => onSelect(document.filename)}
            >
              <FileText className="size-4 shrink-0 text-muted-foreground" />
              <span className="truncate text-sm">{document.filename}</span>
            </button>
            <Button
              variant="ghost"
              size="icon"
              type="button"
              disabled={isLoading || isMutating}
              onClick={() => onDelete(document.filename)}
              aria-label={`删除 ${document.filename}`}
            >
              <Trash2 className="size-4" />
            </Button>
          </div>
        );
      })}
    </div>
  );
}
