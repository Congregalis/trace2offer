"use client";

import { useCallback, useEffect, useState, type ChangeEvent } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  createPrepDocument,
  fetchPrepIndexStatus,
  fetchPrepMeta,
  listPrepDocuments,
  listPrepIndexChunks,
  listPrepIndexDocuments,
  rebuildPrepIndex,
  updatePrepDocument,
} from "@/lib/prep-api";
import {
  DEFAULT_PREP_META,
  PrepIndexChunk,
  PrepIndexDocument,
  PrepIndexRunSummary,
  PrepIndexStatus,
  PrepKnowledgeDocument,
  PrepMeta,
} from "@/lib/prep-types";
import { IndexRunSummary } from "./index-run-summary";
import { IndexStatusCard } from "./index-status-card";

function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === "AbortError";
}

function toMarkdownFilename(value: string): string {
  const trimmed = (value || "").trim();
  if (!trimmed) {
    return "";
  }
  return trimmed.toLowerCase().endsWith(".md") ? trimmed : `${trimmed}.md`;
}

function toInlinePreview(value: string, maxLength = 180): string {
  const normalized = (value || "").replace(/\s+/g, " ").trim();
  if (normalized.length <= maxLength) {
    return normalized;
  }
  return `${normalized.slice(0, maxLength)}...`;
}

export function PrepLibraryWorkspace() {
  const [meta, setMeta] = useState<PrepMeta>(DEFAULT_PREP_META);
  const [metaError, setMetaError] = useState<string | null>(null);
  const [isMetaLoading, setIsMetaLoading] = useState(true);

  const [indexStatus, setIndexStatus] = useState<PrepIndexStatus | null>(null);
  const [indexSummary, setIndexSummary] = useState<PrepIndexRunSummary | null>(null);
  const [indexError, setIndexError] = useState<string | null>(null);
  const [isRebuildingIndex, setIsRebuildingIndex] = useState(false);
  const [indexDocuments, setIndexDocuments] = useState<PrepIndexDocument[]>([]);
  const [indexChunks, setIndexChunks] = useState<PrepIndexChunk[]>([]);
  const [selectedIndexedDocumentID, setSelectedIndexedDocumentID] = useState("");
  const [expandedChunkIDs, setExpandedChunkIDs] = useState<string[]>([]);
  const [isIndexDataLoading, setIsIndexDataLoading] = useState(false);
  const [indexDataError, setIndexDataError] = useState<string | null>(null);

  const [documents, setDocuments] = useState<PrepKnowledgeDocument[]>([]);
  const [isDocumentsLoading, setIsDocumentsLoading] = useState(false);
  const [isDocumentSaving, setIsDocumentSaving] = useState(false);
  const [documentError, setDocumentError] = useState<string | null>(null);
  const [documentMessage, setDocumentMessage] = useState<string | null>(null);
  const [documentFilename, setDocumentFilename] = useState("");
  const [documentContent, setDocumentContent] = useState("");

  const refreshDocuments = useCallback((signal?: AbortSignal) => {
    setIsDocumentsLoading(true);
    setDocumentError(null);
    return listPrepDocuments(signal)
      .then((items) => {
        setDocuments(items);
      })
      .catch((error: unknown) => {
        if (isAbortError(error)) {
          return;
        }
        const message = error instanceof Error && error.message ? error.message : "加载文档失败";
        setDocumentError(message);
      })
      .finally(() => {
        setIsDocumentsLoading(false);
      });
  }, []);

  const refreshIndexedData = useCallback((signal?: AbortSignal, preferredDocumentIDRaw = "") => {
    setIsIndexDataLoading(true);
    setIndexDataError(null);

    return listPrepIndexDocuments(signal)
      .then(async (indexed) => {
        setIndexDocuments(indexed);
        const preferredDocumentID = preferredDocumentIDRaw.trim();
        const nextDocumentID = preferredDocumentID && indexed.some((item) => item.id === preferredDocumentID) ? preferredDocumentID : indexed[0]?.id || "";
        setSelectedIndexedDocumentID(nextDocumentID);
        if (!nextDocumentID) {
          setIndexChunks([]);
          return;
        }
        const chunks = await listPrepIndexChunks({ documentId: nextDocumentID, limit: 120 }, signal);
        setIndexChunks(chunks);
        setExpandedChunkIDs([]);
      })
      .catch((error: unknown) => {
        if (isAbortError(error)) {
          return;
        }
        const message = error instanceof Error && error.message ? error.message : "加载索引数据失败";
        setIndexDataError(message);
      })
      .finally(() => {
        setIsIndexDataLoading(false);
      });
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    setIsMetaLoading(true);
    setMetaError(null);

    void fetchPrepMeta(controller.signal)
      .then((nextMeta) => {
        setMeta(nextMeta);
      })
      .catch((error: unknown) => {
        if (isAbortError(error)) {
          return;
        }
        const message = error instanceof Error && error.message ? error.message : "加载备面元信息失败";
        setMetaError(message);
      })
      .finally(() => {
        setIsMetaLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    setIndexError(null);

    void Promise.all([
      fetchPrepIndexStatus(controller.signal)
        .then((status) => {
          setIndexStatus(status);
        })
        .catch((error: unknown) => {
          if (isAbortError(error)) {
            return;
          }
          const message = error instanceof Error && error.message ? error.message : "加载索引状态失败";
          setIndexError(message);
        }),
      refreshIndexedData(controller.signal),
      refreshDocuments(controller.signal),
    ]);

    return () => {
      controller.abort();
    };
  }, [refreshDocuments, refreshIndexedData]);

  useEffect(() => {
    const documentID = selectedIndexedDocumentID.trim();
    if (!documentID) {
      setIndexChunks([]);
      setExpandedChunkIDs([]);
      return;
    }

    const controller = new AbortController();
    setIsIndexDataLoading(true);
    setIndexDataError(null);

    void listPrepIndexChunks({ documentId: documentID, limit: 120 }, controller.signal)
      .then((chunks) => {
        setIndexChunks(chunks);
        setExpandedChunkIDs([]);
      })
      .catch((error: unknown) => {
        if (isAbortError(error)) {
          return;
        }
        const message = error instanceof Error && error.message ? error.message : "加载索引 chunks 失败";
        setIndexDataError(message);
      })
      .finally(() => {
        setIsIndexDataLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, [selectedIndexedDocumentID]);

  const handleRebuildIndex = (mode: "incremental" | "full") => {
    setIsRebuildingIndex(true);
    setIndexError(null);

    void rebuildPrepIndex({ scope: "*", mode })
      .then(async (summary) => {
        setIndexSummary(summary);
        const status = await fetchPrepIndexStatus();
        setIndexStatus(status);
        await refreshIndexedData(undefined, selectedIndexedDocumentID);
      })
      .catch((error: unknown) => {
        const message = error instanceof Error && error.message ? error.message : "重建索引失败";
        setIndexError(message);
      })
      .finally(() => {
        setIsRebuildingIndex(false);
      });
  };

  const handleUploadMarkdownFile = (event: ChangeEvent<HTMLInputElement>) => {
    const selected = event.target.files?.[0];
    if (!selected) {
      return;
    }
    void selected
      .text()
      .then((content) => {
        const rawName = selected.name.replace(/\.md$/i, "");
        setDocumentFilename(rawName);
        setDocumentContent(content);
        setDocumentMessage(`已读取文件：${selected.name}`);
      })
      .catch(() => {
        setDocumentError("读取上传文件失败，请重试。");
      })
      .finally(() => {
        event.target.value = "";
      });
  };

  const handleSelectDocument = (document: PrepKnowledgeDocument) => {
    setDocumentFilename(document.filename);
    setDocumentContent(document.content);
    setDocumentMessage(`已载入 ${document.filename}，可直接修改并保存。`);
    setDocumentError(null);
  };

  const handleSaveDocument = () => {
    const filename = toMarkdownFilename(documentFilename);
    const content = documentContent;
    if (!filename) {
      setDocumentError("请填写文件名。");
      return;
    }
    if (!content.trim()) {
      setDocumentError("内容为空，至少输入一点文字。");
      return;
    }

    setIsDocumentSaving(true);
    setDocumentError(null);
    setDocumentMessage(null);

    const existing = documents.find((item) => item.filename === filename);
    const savePromise = existing
      ? updatePrepDocument(filename, { content })
      : createPrepDocument({ filename, content });

    void savePromise
      .then(() => refreshDocuments())
      .then(() => {
        setDocumentFilename(filename);
        setDocumentMessage(existing ? "文档已覆盖更新。" : "文档已创建。");
      })
      .catch((error: unknown) => {
        const message = error instanceof Error && error.message ? error.message : "保存文档失败";
        setDocumentError(message);
      })
      .finally(() => {
        setIsDocumentSaving(false);
      });
  };

  return (
    <div className="space-y-4">
      <header className="space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-lg font-semibold tracking-tight">文档仓库</h2>
          {isMetaLoading ? <Badge variant="secondary">加载配置中</Badge> : null}
          {!isMetaLoading && meta.enabled ? <Badge variant="secondary">模块已启用</Badge> : null}
        </div>
        {metaError ? <p className="text-sm text-destructive">{metaError}</p> : null}
        {!isMetaLoading && !meta.enabled ? (
          <p className="text-sm text-destructive">备面模块当前未启用（`T2O_PREP_ENABLED=false`）。</p>
        ) : null}
      </header>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">文档管理</CardTitle>
          <CardDescription>直接上传或粘贴文档，不需要预分类。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
            <Input
              value={documentFilename}
              onChange={(event) => setDocumentFilename(event.target.value)}
              placeholder="文件名（例如：system-design-notes）"
              disabled={isDocumentSaving}
            />
            <Input type="file" accept=".md,text/markdown,text/plain" onChange={handleUploadMarkdownFile} disabled={isDocumentSaving} />
          </div>

          <Textarea
            value={documentContent}
            onChange={(event) => setDocumentContent(event.target.value)}
            placeholder="直接粘贴文档内容，支持 Markdown。"
            className="min-h-[180px]"
            disabled={isDocumentSaving}
          />

          <div className="flex items-center gap-3">
            <Button type="button" onClick={handleSaveDocument} disabled={isDocumentSaving}>
              {isDocumentSaving ? "保存中..." : "保存文档"}
            </Button>
            {documentMessage ? <p className="text-sm text-emerald-600">{documentMessage}</p> : null}
            {documentError ? <p className="text-sm text-destructive">{documentError}</p> : null}
          </div>

          <div className="space-y-2">
            <p className="text-sm font-medium">已保存文档（{documents.length}）</p>
            {isDocumentsLoading ? <p className="text-sm text-muted-foreground">文档加载中...</p> : null}
            {!isDocumentsLoading && documents.length === 0 ? <p className="text-sm text-muted-foreground">当前还没有文档。</p> : null}
            {!isDocumentsLoading ? (
              <div className="flex flex-wrap gap-2">
                {documents.map((document) => (
                  <Button
                    key={`${document.scope}/${document.scopeId}/${document.filename}`}
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => handleSelectDocument(document)}
                    disabled={isDocumentSaving}
                  >
                    {document.filename}
                  </Button>
                ))}
              </div>
            ) : null}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">索引维护</CardTitle>
          <CardDescription>最近索引时间：{indexStatus?.lastIndexedAt || "暂无"}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <IndexStatusCard status={indexStatus} isLoading={false} error={indexError} />
          <div className="flex items-center gap-3">
            <Button type="button" onClick={() => handleRebuildIndex("incremental")} disabled={isRebuildingIndex || !meta.enabled}>
              {isRebuildingIndex ? "执行中..." : "增量重建"}
            </Button>
            <Button type="button" variant="outline" onClick={() => handleRebuildIndex("full")} disabled={isRebuildingIndex || !meta.enabled}>
              {isRebuildingIndex ? "执行中..." : "全量重建"}
            </Button>
            {indexError ? <p className="text-sm text-destructive">{indexError}</p> : null}
          </div>
          <IndexRunSummary summary={indexSummary} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">索引内容（Docs / Chunks）</CardTitle>
          <CardDescription>这里展示 sqlite 里的真实索引数据，方便排查召回命中。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-3">
            <Button type="button" variant="outline" onClick={() => void refreshIndexedData(undefined, selectedIndexedDocumentID)} disabled={isIndexDataLoading}>
              {isIndexDataLoading ? "加载中..." : "刷新索引数据"}
            </Button>
            {indexDataError ? <p className="text-sm text-destructive">{indexDataError}</p> : null}
          </div>

          <div className="space-y-2">
            <p className="text-sm font-medium">Documents（{indexDocuments.length}）</p>
            {indexDocuments.length === 0 ? <p className="text-sm text-muted-foreground">当前索引里还没有 documents。</p> : null}
            {indexDocuments.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {indexDocuments.map((document) => (
                  <Button
                    key={document.id}
                    type="button"
                    size="sm"
                    variant={document.id === selectedIndexedDocumentID ? "default" : "outline"}
                    onClick={() => setSelectedIndexedDocumentID(document.id)}
                  >
                    {document.scopeId}/{document.title}
                  </Button>
                ))}
              </div>
            ) : null}
          </div>

          <div className="space-y-2">
            <p className="text-sm font-medium">Chunks（{indexChunks.length}）</p>
            {indexChunks.length === 0 ? <p className="text-sm text-muted-foreground">当前 document 下没有 chunks。</p> : null}
            {indexChunks.length > 0 ? (
              <div className="space-y-2">
                {indexChunks.map((chunk) => (
                  <div key={chunk.id} className="rounded-md border border-border/70 p-3">
                    <p className="text-xs text-muted-foreground">
                      {chunk.scopeId}/{chunk.documentTitle} · chunk #{chunk.chunkIndex} · tokens {chunk.tokenCount}
                    </p>
                    <p className="mt-1 text-sm">
                      {expandedChunkIDs.includes(chunk.id) ? chunk.content : toInlinePreview(chunk.content)}
                    </p>
                    <div className="mt-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setExpandedChunkIDs((previous) =>
                            previous.includes(chunk.id) ? previous.filter((item) => item !== chunk.id) : [...previous, chunk.id],
                          );
                        }}
                      >
                        {expandedChunkIDs.includes(chunk.id) ? "收起" : "展开全文"}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : null}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
