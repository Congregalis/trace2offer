"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PrepKnowledgeDocument, PrepScope } from "@/lib/prep-types";
import { Plus } from "lucide-react";
import { KnowledgeDocumentList } from "./knowledge-document-list";
import { KnowledgeScopeSwitcher } from "./knowledge-scope-switcher";

export interface PrepKnowledgeSidebarProps {
  scopes: PrepScope[];
  activeScope: PrepScope;
  scopeID: string;
  documents: PrepKnowledgeDocument[];
  selectedFilename: string;
  onScopeChange: (scope: PrepScope) => void;
  onSelectDocument: (filename: string) => void;
  onDeleteDocument: (filename: string) => void;
  onCreateDocument: () => void;
  isLoading?: boolean;
  isMutating?: boolean;
}

export function PrepKnowledgeSidebar({
  scopes,
  activeScope,
  scopeID,
  documents,
  selectedFilename,
  onScopeChange,
  onSelectDocument,
  onDeleteDocument,
  onCreateDocument,
  isLoading = false,
  isMutating = false,
}: PrepKnowledgeSidebarProps) {
  return (
    <Card className="h-full">
      <CardHeader className="space-y-3">
        <div className="flex items-center justify-between gap-2">
          <CardTitle className="text-base">资料库</CardTitle>
          <Button type="button" size="sm" variant="outline" disabled={isLoading || isMutating} onClick={onCreateDocument}>
            <Plus className="size-4" />
            新建
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">当前对象：{scopeID || "-"}</p>
        <KnowledgeScopeSwitcher scopes={scopes} value={activeScope} onChange={onScopeChange} disabled={isLoading || isMutating} />
      </CardHeader>
      <CardContent>
        <KnowledgeDocumentList
          documents={documents}
          selectedFilename={selectedFilename}
          onSelect={onSelectDocument}
          onDelete={onDeleteDocument}
          isLoading={isLoading}
          isMutating={isMutating}
        />
      </CardContent>
    </Card>
  );
}
