"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useLeadsStore } from "@/lib/leads-store";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart";
import { BarChart3, Loader2, RefreshCw, TrendingUp } from "lucide-react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");

interface StatusCount {
  status: string;
  label: string;
  count: number;
  percentage: number;
}

interface OverviewStats {
  total: number;
  active: number;
  offered: number;
  success_rate: number;
  this_week_new: number;
  status_counts: StatusCount[];
  last_updated: string;
}

interface FunnelStage {
  status: string;
  label: string;
  count: number;
  cumulative_count: number;
  percentage: number;
  conversion_from_prev: number;
  avg_days: number;
}

interface FunnelStats {
  stages: FunnelStage[];
  conversion: number;
  total_time_avg: number;
}

interface TrendPoint {
  date: string;
  label: string;
  new: number;
  moved: number;
  total: number;
}

interface TrendStats {
  period: "week" | "month" | string;
  points: TrendPoint[];
  growth_rate: number;
  is_growing: boolean;
}

interface DurationStatus {
  status: string;
  label: string;
  count: number;
  avg_days: number;
}

interface DurationStats {
  average_cycle_days: number;
  average_active_days: number;
  slowest_status: string;
  slowest_label: string;
  by_status: DurationStatus[];
}

interface DashboardStats {
  overview: OverviewStats;
  funnel: FunnelStats;
  weekly_trend: TrendStats;
  monthly_trend: TrendStats;
  duration: DurationStats;
  generated_at: string;
}

interface APIErrorPayload {
  message?: string;
  error?: string;
}

interface APIPayload {
  data?: DashboardStats;
}

const STATUS_ACCENT_CLASS: Record<string, string> = {
  new: "text-fuchsia-300",
  preparing: "text-cyan-300",
  applied: "text-warning",
  interviewing: "text-indigo-300",
  offered: "text-success",
  declined: "text-chart-4",
  rejected: "text-destructive",
  archived: "text-muted-foreground",
};

function formatGeneratedAt(raw: string): string {
  const ts = Date.parse(raw);
  if (Number.isNaN(ts)) {
    return "-";
  }
  const date = new Date(ts);
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, "0")}:${String(
    date.getMinutes()
  ).padStart(2, "0")}`;
}

function formatDays(days: number): string {
  if (!Number.isFinite(days)) {
    return "-";
  }
  return `${days.toFixed(1)} 天`;
}

function toErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return fallback;
}

async function parseAPIError(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as APIErrorPayload;
    const parts = [payload.message, payload.error].filter(Boolean);
    if (parts.length > 0) {
      return parts.join(": ");
    }
  } catch {
    // Ignore non-json body.
  }
  return `加载统计仪表板失败 (HTTP ${response.status})`;
}

export function StatsCards() {
  const { leads } = useLeadsStore();
  const [dashboard, setDashboard] = useState<DashboardStats | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [trendPeriod, setTrendPeriod] = useState<"week" | "month">("week");

  const leadsVersion = useMemo(() => {
    return leads.map((lead) => `${lead.id}:${lead.updatedAt}`).join("|");
  }, [leads]);

  const fetchDashboard = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE_URL}/api/stats`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }
      const payload = (await response.json()) as APIPayload;
      if (!payload.data) {
        throw new Error("加载统计仪表板失败: 响应缺少 data");
      }
      setDashboard(payload.data);
    } catch (error) {
      setError(toErrorMessage(error, "加载统计仪表板失败"));
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchDashboard();
  }, [fetchDashboard, leadsVersion]);

  const trend = useMemo(() => {
    if (!dashboard) {
      return null;
    }
    return trendPeriod === "week" ? dashboard.weekly_trend : dashboard.monthly_trend;
  }, [dashboard, trendPeriod]);

  if (!dashboard && isLoading) {
    return (
      <div className="rounded-lg border border-border bg-card px-4 py-6 text-sm text-muted-foreground">
        <span className="inline-flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          正在加载统计仪表板...
        </span>
      </div>
    );
  }

  if (!dashboard && error) {
    return (
      <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-6">
        <p className="text-sm text-destructive">{error}</p>
        <Button size="sm" variant="outline" className="mt-3" onClick={() => void fetchDashboard()}>
          <RefreshCw className="mr-1 h-4 w-4" />
          重试
        </Button>
      </div>
    );
  }

  if (!dashboard || !trend) {
    return null;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2">
        <p className="text-xs text-muted-foreground">
          最近更新：{formatGeneratedAt(dashboard.generated_at)} · 成功率 {dashboard.overview.success_rate.toFixed(1)}%
        </p>
        <Button size="sm" variant="ghost" onClick={() => void fetchDashboard()} disabled={isLoading}>
          {isLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
        </Button>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-8">
        {dashboard.overview.status_counts.map((item) => (
          <div key={item.status} className="rounded-lg border border-border bg-card px-4 py-3">
            <p className="text-xs text-muted-foreground">{item.label}</p>
            <p className={cn("mt-1 text-2xl font-semibold tabular-nums", STATUS_ACCENT_CLASS[item.status] || "text-foreground")}>
              {item.count}
            </p>
            <p className="text-[11px] text-muted-foreground">{item.percentage.toFixed(1)}%</p>
          </div>
        ))}
      </div>

      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="inline-flex items-center gap-2 text-base">
              <BarChart3 className="h-4 w-4" />
              转化率漏斗
            </CardTitle>
            <CardDescription>核心路径：new → preparing → applied → interviewing → offered</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {dashboard.funnel.stages.map((stage, index) => (
              <div key={stage.status} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="font-medium text-foreground">
                    {index + 1}. {stage.label}
                  </span>
                  <span className="text-muted-foreground">{stage.count} 个</span>
                </div>
                <div className="h-2 rounded-full bg-secondary">
                  <div
                    className="h-2 rounded-full bg-gradient-to-r from-chart-2/60 via-chart-2 to-chart-1"
                    style={{ width: `${Math.max(0, Math.min(100, stage.percentage))}%` }}
                  />
                </div>
                <div className="flex items-center justify-between text-[11px] text-muted-foreground">
                  <span>阶段保留率 {stage.conversion_from_prev.toFixed(1)}%</span>
                  <span>平均停留 {formatDays(stage.avg_days)}</span>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between gap-3">
              <div>
                <CardTitle className="inline-flex items-center gap-2 text-base">
                  <TrendingUp className="h-4 w-4" />
                  新增线索趋势
                </CardTitle>
                <CardDescription>
                  {trendPeriod === "week" ? "最近 7 天" : "最近 30 天"} · 增长率 {trend.growth_rate.toFixed(1)}%
                </CardDescription>
              </div>
              <div className="inline-flex rounded-md border border-border bg-background p-0.5">
                <button
                  className={cn(
                    "rounded px-2 py-1 text-xs transition-colors",
                    trendPeriod === "week" ? "bg-primary text-primary-foreground" : "text-muted-foreground"
                  )}
                  onClick={() => setTrendPeriod("week")}
                  type="button"
                >
                  周视图
                </button>
                <button
                  className={cn(
                    "rounded px-2 py-1 text-xs transition-colors",
                    trendPeriod === "month" ? "bg-primary text-primary-foreground" : "text-muted-foreground"
                  )}
                  onClick={() => setTrendPeriod("month")}
                  type="button"
                >
                  月视图
                </button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <ChartContainer
              config={{
                new: { label: "新增", color: "hsl(var(--chart-2))" },
                moved: { label: "变更", color: "hsl(var(--chart-5))" },
              }}
              className="h-[220px] w-full"
            >
              <LineChart data={trend.points}>
                <CartesianGrid vertical={false} />
                <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={12} />
                <YAxis allowDecimals={false} width={30} />
                <ChartTooltip cursor={false} content={<ChartTooltipContent />} />
                <Line dataKey="new" type="monotone" stroke="var(--color-new)" strokeWidth={2} dot={false} />
                <Line dataKey="moved" type="monotone" stroke="var(--color-moved)" strokeWidth={2} dot={false} />
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">平均停留时间分析</CardTitle>
          <CardDescription>识别当前卡点，优先清理停留过久的状态</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
            <div className="rounded-lg border border-border bg-background px-3 py-2">
              <p className="text-xs text-muted-foreground">平均周期（终态）</p>
              <p className="text-lg font-semibold tabular-nums">{formatDays(dashboard.duration.average_cycle_days)}</p>
            </div>
            <div className="rounded-lg border border-border bg-background px-3 py-2">
              <p className="text-xs text-muted-foreground">活跃线索平均停留</p>
              <p className="text-lg font-semibold tabular-nums">{formatDays(dashboard.duration.average_active_days)}</p>
            </div>
            <div className="rounded-lg border border-border bg-background px-3 py-2">
              <p className="text-xs text-muted-foreground">最慢阶段</p>
              <p className="text-lg font-semibold">{dashboard.duration.slowest_label || "-"}</p>
            </div>
          </div>

          <div className="space-y-2">
            {dashboard.duration.by_status.map((item) => (
              <div key={item.status} className="grid grid-cols-[minmax(0,1fr)_72px_72px] items-center gap-3 text-sm">
                <span className="truncate text-foreground">
                  {item.label} <span className="text-xs text-muted-foreground">({item.count})</span>
                </span>
                <span className="text-right tabular-nums text-muted-foreground">{item.avg_days.toFixed(1)} 天</span>
                <div className="h-2 rounded-full bg-secondary">
                  <div
                    className="h-2 rounded-full bg-chart-3"
                    style={{
                      width: `${Math.max(0, Math.min(100, item.avg_days * 4))}%`,
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </div>
  );
}
