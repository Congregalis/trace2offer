"use client";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { PrepIndexStatus } from "@/lib/prep-types";

export interface IndexStatusCardProps {
  status: PrepIndexStatus | null;
  isLoading?: boolean;
  error?: string | null;
}

function statusLabel(status: string): string {
  const normalized = status.trim().toLowerCase();
  if (normalized === "completed") {
    return "已完成";
  }
  if (normalized === "running") {
    return "进行中";
  }
  if (normalized === "failed") {
    return "失败";
  }
  return normalized || "未知";
}

export function IndexStatusCard({ status, isLoading = false, error }: IndexStatusCardProps) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">索引状态</CardTitle>
          <CardDescription>正在加载索引状态...</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">索引状态</CardTitle>
          <CardDescription className="text-destructive">{error}</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  if (!status) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">索引状态</CardTitle>
          <CardDescription>暂无索引元数据。</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between gap-2">
          <CardTitle className="text-base">索引状态</CardTitle>
          <Badge variant={status.lastIndexStatus === "completed" ? "secondary" : "outline"}>{statusLabel(status.lastIndexStatus)}</Badge>
        </div>
        <CardDescription>
          provider: {status.embeddingProvider || "-"} · model: {status.embeddingModel || "-"}
        </CardDescription>
      </CardHeader>
      <CardContent className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-3">
        <Metric label="文档数" value={status.documentCount} />
        <Metric label="Chunk 数" value={status.chunkCount} />
        <Metric label="最近索引" value={status.lastIndexedAt || "-"} full />
      </CardContent>
    </Card>
  );
}

function Metric({ label, value, full = false }: { label: string; value: number | string; full?: boolean }) {
  return (
    <div className={`rounded-md border px-3 py-2 ${full ? "col-span-2 sm:col-span-1" : ""}`}>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="truncate text-base font-semibold leading-tight">{value}</p>
    </div>
  );
}
