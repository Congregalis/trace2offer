"use client";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { PrepIndexRunSummary } from "@/lib/prep-types";

export interface IndexRunSummaryProps {
  summary: PrepIndexRunSummary | null;
}

function statusLabel(status: string): string {
  const normalized = status.trim().toLowerCase();
  if (normalized === "completed") {
    return "完成";
  }
  if (normalized === "failed") {
    return "失败";
  }
  if (normalized === "running") {
    return "进行中";
  }
  return normalized || "未知";
}

function modeLabel(mode: string): string {
  const normalized = mode.trim().toLowerCase();
  if (normalized === "full") {
    return "全量";
  }
  if (normalized === "incremental") {
    return "增量";
  }
  return normalized || "未知";
}

export function IndexRunSummary({ summary }: IndexRunSummaryProps) {
  if (!summary) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">索引摘要</CardTitle>
          <CardDescription>还没有索引记录，点击“重新索引”后这里会显示本次执行结果。</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between gap-2">
          <CardTitle className="text-base">索引摘要</CardTitle>
          <Badge variant={summary.status === "completed" ? "secondary" : "destructive"}>{statusLabel(summary.status)}</Badge>
        </div>
        <CardDescription>
          run_id: {summary.runId || "-"} · 模式 {modeLabel(summary.mode)} · 开始 {summary.startedAt || "-"} · 结束 {summary.completedAt || "-"}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
          <Metric label="扫描文档" value={summary.documentsScanned} />
          <Metric label="已索引文档" value={summary.documentsIndexed} />
          <Metric label="跳过文档" value={summary.documentsSkipped} />
          <Metric label="删除文档" value={summary.documentsDeleted} />
          <Metric label="新增 chunk" value={summary.chunksCreated} />
          <Metric label="更新 chunk" value={summary.chunksUpdated} />
          <Metric label="错误数" value={summary.errors.length} />
        </div>
        {summary.errors.length > 0 ? (
          <div className="space-y-1 rounded-md border border-destructive/40 bg-destructive/5 p-3 text-xs text-destructive">
            {summary.errors.map((item) => (
              <p key={`${summary.runId}-${item.source}-${item.message}`}>
                {(item.source || "unknown").trim()}: {(item.message || "unknown error").trim()}
              </p>
            ))}
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border px-3 py-2">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-base font-semibold leading-tight">{value}</p>
    </div>
  );
}
