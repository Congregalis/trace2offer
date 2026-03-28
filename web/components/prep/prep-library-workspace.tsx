"use client";

import { useCallback, useEffect, useState, type ChangeEvent } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  createPrepKnowledgeDocument,
  createPrepTopic,
  fetchPrepIndexStatus,
  fetchPrepMeta,
  listPrepIndexChunks,
  listPrepIndexDocuments,
  listPrepKnowledgeDocuments,
  listPrepTopics,
  rebuildPrepIndex,
  updatePrepKnowledgeDocument,
} from "@/lib/prep-api";
import {
  DEFAULT_PREP_META,
  PrepIndexChunk,
  PrepIndexDocument,
  PrepIndexRunSummary,
  PrepIndexStatus,
  PrepKnowledgeDocument,
  PrepMeta,
  PrepTopic,
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

  const [topics, setTopics] = useState<PrepTopic[]>([]);
  const [isTopicsLoading, setIsTopicsLoading] = useState(false);
  const [isTopicCreating, setIsTopicCreating] = useState(false);
  const [topicError, setTopicError] = useState<string | null>(null);
  const [activeTopicKey, setActiveTopicKey] = useState("");
  const [newTopicKey, setNewTopicKey] = useState("");
  const [newTopicName, setNewTopicName] = useState("");

  const [knowledgeDocuments, setKnowledgeDocuments] = useState<PrepKnowledgeDocument[]>([]);
  const [isKnowledgeLoading, setIsKnowledgeLoading] = useState(false);
  const [isKnowledgeSaving, setIsKnowledgeSaving] = useState(false);
  const [knowledgeError, setKnowledgeError] = useState<string | null>(null);
  const [knowledgeMessage, setKnowledgeMessage] = useState<string | null>(null);
  const [knowledgeFilename, setKnowledgeFilename] = useState("");
  const [knowledgeContent, setKnowledgeContent] = useState("");

  const refreshIndexedData = useCallback((signal?: AbortSignal, preferredDocumentIDRaw = "") => {
    setIsIndexDataLoading(true);
    setIndexDataError(null);

    return listPrepIndexDocuments(signal)
      .then(async (documents) => {
        setIndexDocuments(documents);
        const preferredDocumentID = preferredDocumentIDRaw.trim();
        const nextDocumentID = preferredDocumentID && documents.some((item) => item.id === preferredDocumentID) ? preferredDocumentID : documents[0]?.id || "";
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
  }, [refreshIndexedData]);

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
    ]);

    return () => {
      controller.abort();
    };
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    setIsTopicsLoading(true);
    setTopicError(null);

    void listPrepTopics(controller.signal)
      .then((items) => {
        setTopics(items);
        setActiveTopicKey((previous) => {
          if (previous && items.some((item) => item.key === previous)) {
            return previous;
          }
          return items[0]?.key || "";
        });
      })
      .catch((error: unknown) => {
        if (isAbortError(error)) {
          return;
        }
        const message = error instanceof Error && error.message ? error.message : "加载 topic 失败";
        setTopicError(message);
      })
      .finally(() => {
        setIsTopicsLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, []);

  useEffect(() => {
    const topicKey = activeTopicKey.trim();
    if (!topicKey) {
      setKnowledgeDocuments([]);
      setIsKnowledgeLoading(false);
      return;
    }

    const controller = new AbortController();
    setIsKnowledgeLoading(true);
    setKnowledgeError(null);

    void listPrepKnowledgeDocuments("topics", topicKey, controller.signal)
      .then((documents) => {
        setKnowledgeDocuments(documents);
      })
      .catch((error: unknown) => {
        if (isAbortError(error)) {
          return;
        }
        const message = error instanceof Error && error.message ? error.message : "加载资料文档失败";
        setKnowledgeError(message);
      })
      .finally(() => {
        setIsKnowledgeLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, [activeTopicKey]);

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

  const handleCreateTopic = () => {
    const key = newTopicKey.trim();
    const name = newTopicName.trim();
    if (!key || !name) {
      setTopicError("请填写 topic key 和 topic 名称。");
      return;
    }

    setIsTopicCreating(true);
    setTopicError(null);

    void createPrepTopic({
      key,
      name,
      description: "",
    })
      .then(async (created) => {
        const items = await listPrepTopics();
        setTopics(items);
        setActiveTopicKey(created.key || items[0]?.key || "");
        setNewTopicKey("");
        setNewTopicName("");
      })
      .catch((error: unknown) => {
        const message = error instanceof Error && error.message ? error.message : "创建 topic 失败";
        setTopicError(message);
      })
      .finally(() => {
        setIsTopicCreating(false);
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
        setKnowledgeFilename(rawName);
        setKnowledgeContent(content);
        setKnowledgeMessage(`已读取文件：${selected.name}`);
      })
      .catch(() => {
        setKnowledgeError("读取上传文件失败，请重试。");
      })
      .finally(() => {
        event.target.value = "";
      });
  };

  const handleSelectKnowledgeDocument = (document: PrepKnowledgeDocument) => {
    setKnowledgeFilename(document.filename);
    setKnowledgeContent(document.content);
    setKnowledgeMessage(`已载入 ${document.filename}，可直接修改并保存。`);
    setKnowledgeError(null);
  };

  const handleSaveKnowledgeDocument = () => {
    const topicKey = activeTopicKey.trim();
    if (!topicKey) {
      setKnowledgeError("请先选择一个 topic。");
      return;
    }

    const filename = toMarkdownFilename(knowledgeFilename);
    const content = knowledgeContent;
    if (!filename) {
      setKnowledgeError("请填写文件名。");
      return;
    }
    if (!content.trim()) {
      setKnowledgeError("内容为空，至少输入一点文字。");
      return;
    }

    setIsKnowledgeSaving(true);
    setKnowledgeError(null);
    setKnowledgeMessage(null);

    const existing = knowledgeDocuments.find((item) => item.filename === filename);
    const savePromise = existing
      ? updatePrepKnowledgeDocument("topics", topicKey, filename, { content })
      : createPrepKnowledgeDocument("topics", topicKey, { filename, content });

    void savePromise
      .then(() => listPrepKnowledgeDocuments("topics", topicKey))
      .then((documents) => {
        setKnowledgeDocuments(documents);
        setKnowledgeFilename(filename);
        setKnowledgeMessage(existing ? "资料已覆盖更新。" : "资料已创建。");
      })
      .catch((error: unknown) => {
        const message = error instanceof Error && error.message ? error.message : "保存资料失败";
        setKnowledgeError(message);
      })
      .finally(() => {
        setIsKnowledgeSaving(false);
      });
  };

  return (
    <div className="space-y-4">
      <header className="space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-lg font-semibold tracking-tight">资料库</h2>
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
          <CardTitle className="text-base">Topic 资料库</CardTitle>
          <CardDescription>先创建 topic，再往 topic 下粘贴文本或上传 `.md`。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-2 sm:grid-cols-[minmax(0,160px)_minmax(0,1fr)_auto]">
            <Input value={newTopicKey} onChange={(event) => setNewTopicKey(event.target.value)} placeholder="topic key" disabled={isTopicCreating} />
            <Input value={newTopicName} onChange={(event) => setNewTopicName(event.target.value)} placeholder="topic 名称" disabled={isTopicCreating} />
            <Button type="button" onClick={handleCreateTopic} disabled={isTopicCreating}>
              {isTopicCreating ? "创建中..." : "创建 Topic"}
            </Button>
          </div>
          {topicError ? <p className="text-sm text-destructive">{topicError}</p> : null}

          <div className="grid gap-2">
            <p className="text-sm font-medium">选择 Topic</p>
            <div className="flex flex-wrap gap-2">
              {isTopicsLoading ? <p className="text-sm text-muted-foreground">加载 topics...</p> : null}
              {!isTopicsLoading && topics.length === 0 ? <p className="text-sm text-muted-foreground">还没有 topic，先创建一个。</p> : null}
              {!isTopicsLoading
                ? topics.map((topic) => (
                    <Button
                      key={topic.key}
                      type="button"
                      size="sm"
                      variant={topic.key === activeTopicKey ? "default" : "outline"}
                      onClick={() => setActiveTopicKey(topic.key)}
                      disabled={isTopicCreating}
                    >
                      {topic.key}
                    </Button>
                  ))
                : null}
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
            <Input
              value={knowledgeFilename}
              onChange={(event) => setKnowledgeFilename(event.target.value)}
              placeholder="文件名（例如：system-design-notes）"
              disabled={!activeTopicKey.trim() || isKnowledgeSaving}
            />
            <Input type="file" accept=".md,text/markdown,text/plain" onChange={handleUploadMarkdownFile} disabled={!activeTopicKey.trim() || isKnowledgeSaving} />
          </div>

          <Textarea
            value={knowledgeContent}
            onChange={(event) => setKnowledgeContent(event.target.value)}
            placeholder="直接粘贴资料文本，支持 Markdown。"
            className="min-h-[180px]"
            disabled={!activeTopicKey.trim() || isKnowledgeSaving}
          />

          <div className="flex items-center gap-3">
            <Button type="button" onClick={handleSaveKnowledgeDocument} disabled={!activeTopicKey.trim() || isKnowledgeSaving}>
              {isKnowledgeSaving ? "保存中..." : "保存资料"}
            </Button>
            {knowledgeMessage ? <p className="text-sm text-emerald-600">{knowledgeMessage}</p> : null}
            {knowledgeError ? <p className="text-sm text-destructive">{knowledgeError}</p> : null}
          </div>

          <div className="space-y-2">
            <p className="text-sm font-medium">已保存文档（{knowledgeDocuments.length}）</p>
            {isKnowledgeLoading ? <p className="text-sm text-muted-foreground">文档加载中...</p> : null}
            {!isKnowledgeLoading && knowledgeDocuments.length === 0 ? <p className="text-sm text-muted-foreground">当前 topic 还没有资料文档。</p> : null}
            {!isKnowledgeLoading ? (
              <div className="flex flex-wrap gap-2">
                {knowledgeDocuments.map((document) => (
                  <Button
                    key={document.filename}
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => handleSelectKnowledgeDocument(document)}
                    disabled={isKnowledgeSaving}
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
          <CardDescription>这里展示 sqlite 里的真实索引数据，方便你排查召回命中。</CardDescription>
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
