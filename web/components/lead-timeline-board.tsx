"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { CalendarDays, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { useLeadsStore } from "@/lib/leads-store";
import { useLeadTimelineStore } from "@/lib/lead-timeline-store";
import { Lead, LeadStatus, STATUS_CONFIG } from "@/lib/types";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

const DAY_MS = 24 * 60 * 60 * 1000;
const HOUR_MS = 60 * 60 * 1000;
const ROW_LANE_HEIGHT = 24;
const LEFT_COLUMN_WIDTH = 300;
const ZOOM_IN_FACTOR = 0.88;
const ZOOM_OUT_FACTOR = 1.12;

const STAGE_COLOR_CLASS: Record<LeadStatus, string> = {
  new: "bg-fuchsia-500/90",
  preparing: "bg-cyan-500/90",
  applied: "bg-amber-500/90",
  interviewing: "bg-indigo-500/90",
  offered: "bg-emerald-500/90",
  declined: "bg-zinc-500/85",
  rejected: "bg-rose-500/90",
  archived: "bg-slate-500/85",
};

interface StageSegment {
  stage: string;
  start: Date;
  end: Date;
}

interface VisibleStageSegment {
  stage: string;
  start: Date;
  end: Date;
  visibleStart: number;
  visibleEnd: number;
}

interface TimelineRow {
  lead: Lead;
  segments: StageSegment[];
}

interface TimeRange {
  minTime: number;
  maxTime: number;
  duration: number;
}

function parseDate(value: string): Date | null {
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return null;
  }
  return new Date(timestamp);
}

function getStageLabel(stage: string): string {
  const key = stage as LeadStatus;
  const config = STATUS_CONFIG[key];
  if (config) {
    return config.label;
  }
  return stage || "未知阶段";
}

function getStageColor(stage: string): string {
  const key = stage as LeadStatus;
  return STAGE_COLOR_CLASS[key] || "bg-slate-400";
}

function formatTickLabel(date: Date, stepMs: number): string {
  if (stepMs >= 28 * DAY_MS) {
    return new Intl.DateTimeFormat("zh-CN", { year: "numeric", month: "short" }).format(date);
  }
  return new Intl.DateTimeFormat("zh-CN", { month: "numeric", day: "numeric" }).format(date);
}

function formatShortDate(date: Date): string {
  return new Intl.DateTimeFormat("zh-CN", {
    month: "numeric",
    day: "numeric",
  }).format(date);
}

function formatRangeDate(date: Date): string {
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}

function buildSegments(
  lead: Lead,
  timelineStages: Array<{ stage: string; startedAt: string; endedAt: string }>
): StageSegment[] {
  const now = new Date();
  const segments: StageSegment[] = [];

  for (const stage of timelineStages) {
    const start = parseDate(stage.startedAt);
    if (!start) {
      continue;
    }

    const ended = parseDate(stage.endedAt);
    let end = ended ?? now;
    if (end.getTime() <= start.getTime()) {
      end = new Date(start.getTime() + 2 * HOUR_MS);
    }

    segments.push({ stage: stage.stage, start, end });
  }

  if (segments.length > 0) {
    return segments.sort((a, b) => a.start.getTime() - b.start.getTime());
  }

  const fallbackStart = parseDate(lead.createdAt) ?? parseDate(lead.updatedAt);
  const fallbackEnd = parseDate(lead.updatedAt) ?? now;
  if (!fallbackStart) {
    return [];
  }

  let normalizedFallbackEnd = fallbackEnd;
  if (normalizedFallbackEnd.getTime() <= fallbackStart.getTime()) {
    normalizedFallbackEnd = new Date(fallbackStart.getTime() + DAY_MS);
  }

  return [{ stage: lead.status, start: fallbackStart, end: normalizedFallbackEnd }];
}

function buildDataRange(rows: TimelineRow[]): TimeRange | null {
  const allSegments = rows.flatMap((row) => row.segments);
  if (allSegments.length === 0) {
    return null;
  }

  let minTime = allSegments[0].start.getTime();
  let maxTime = allSegments[0].end.getTime();
  for (const segment of allSegments) {
    minTime = Math.min(minTime, segment.start.getTime());
    maxTime = Math.max(maxTime, segment.end.getTime());
  }

  if (maxTime <= minTime) {
    maxTime = minTime + DAY_MS;
  }

  return {
    minTime,
    maxTime,
    duration: maxTime - minTime,
  };
}

function getMinZoomDuration(bounds: TimeRange): number {
  return Math.min(bounds.duration, Math.max(6 * HOUR_MS, bounds.duration / 250));
}

function clampViewByCenter(bounds: TimeRange, center: number, duration: number): TimeRange {
  const minDuration = getMinZoomDuration(bounds);
  const maxDuration = bounds.duration;
  const clampedDuration = Math.min(maxDuration, Math.max(minDuration, duration));

  let minTime = center - clampedDuration / 2;
  let maxTime = center + clampedDuration / 2;

  if (minTime < bounds.minTime) {
    maxTime += bounds.minTime - minTime;
    minTime = bounds.minTime;
  }
  if (maxTime > bounds.maxTime) {
    minTime -= maxTime - bounds.maxTime;
    maxTime = bounds.maxTime;
  }

  if (minTime < bounds.minTime) {
    minTime = bounds.minTime;
  }
  if (maxTime > bounds.maxTime) {
    maxTime = bounds.maxTime;
  }

  return {
    minTime,
    maxTime,
    duration: maxTime - minTime,
  };
}

function buildTicks(scale: TimeRange | null, stepMs: number): Date[] {
  if (!scale || stepMs <= 0) {
    return [];
  }

  const ticks: Date[] = [];
  const start = Math.floor(scale.minTime / stepMs) * stepMs;
  for (let timestamp = start; timestamp <= scale.maxTime; timestamp += stepMs) {
    ticks.push(new Date(timestamp));
  }
  return ticks;
}

function selectMajorTickStep(duration: number): number {
  if (duration <= 10 * DAY_MS) {
    return DAY_MS;
  }
  if (duration <= 45 * DAY_MS) {
    return 3 * DAY_MS;
  }
  if (duration <= 120 * DAY_MS) {
    return 7 * DAY_MS;
  }
  if (duration <= 365 * DAY_MS) {
    return 14 * DAY_MS;
  }
  return 30 * DAY_MS;
}

export function LeadTimelineBoard() {
  const { leads, isLoading: isLeadLoading, hasLoaded: hasLoadedLeads, fetchLeads } = useLeadsStore();
  const {
    timelineMap,
    isLoading: isTimelineLoading,
    hasLoaded: hasLoadedTimelines,
    fetchTimelines,
  } = useLeadTimelineStore();

  const [search, setSearch] = useState("");
  const [viewRange, setViewRange] = useState<TimeRange | null>(null);

  useEffect(() => {
    if (hasLoadedLeads) {
      return;
    }

    void fetchLeads().catch((error) => {
      const message = error instanceof Error && error.message ? error.message : "加载线索失败";
      toast.error(message);
    });
  }, [fetchLeads, hasLoadedLeads]);

  useEffect(() => {
    if (hasLoadedTimelines) {
      return;
    }

    void fetchTimelines().catch((error) => {
      const message = error instanceof Error && error.message ? error.message : "加载时间线失败";
      toast.error(message);
    });
  }, [fetchTimelines, hasLoadedTimelines]);

  const filteredLeads = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    if (!keyword) {
      return leads;
    }

    return leads.filter((lead) => {
      return (
        lead.company.toLowerCase().includes(keyword) ||
        lead.position.toLowerCase().includes(keyword) ||
        lead.status.toLowerCase().includes(keyword) ||
        lead.location.toLowerCase().includes(keyword)
      );
    });
  }, [leads, search]);

  const rows = useMemo(() => {
    return filteredLeads.map((lead) => {
      const timeline = timelineMap[lead.id];
      const segments = buildSegments(lead, timeline?.stages || []);
      return { lead, segments } satisfies TimelineRow;
    });
  }, [filteredLeads, timelineMap]);

  const dataRange = useMemo(() => buildDataRange(rows), [rows]);

  useEffect(() => {
    if (!dataRange) {
      setViewRange(null);
      return;
    }

    setViewRange((previous) => {
      if (!previous) {
        return dataRange;
      }

      const previousCenter = previous.minTime + previous.duration / 2;
      return clampViewByCenter(dataRange, previousCenter, previous.duration);
    });
  }, [dataRange]);

  const scale = viewRange;
  const majorStep = useMemo(() => selectMajorTickStep(scale?.duration ?? 0), [scale]);
  const minorStep = useMemo(() => Math.max(DAY_MS, Math.floor(majorStep / 2)), [majorStep]);
  const majorTicks = useMemo(() => buildTicks(scale, majorStep), [scale, majorStep]);
  const minorTicks = useMemo(() => buildTicks(scale, minorStep), [scale, minorStep]);

  const rangeLabel = useMemo(() => {
    if (!scale) {
      return "暂无阶段时间数据";
    }
    return `${formatRangeDate(new Date(scale.minTime))} - ${formatRangeDate(new Date(scale.maxTime))}`;
  }, [scale]);

  const handleWheelZoom = useCallback(
    (event: React.WheelEvent<HTMLDivElement>) => {
      if (!dataRange || !scale) {
        return;
      }

      if (Math.abs(event.deltaX) > Math.abs(event.deltaY)) {
        return;
      }

      event.preventDefault();

      const rect = event.currentTarget.getBoundingClientRect();
      const timelineWidth = Math.max(event.currentTarget.scrollWidth - LEFT_COLUMN_WIDTH, 1);
      const cursorX =
        event.currentTarget.scrollLeft + (event.clientX - rect.left) - LEFT_COLUMN_WIDTH;
      const clampedCursorX = Math.min(Math.max(cursorX, 0), timelineWidth);
      const ratio = clampedCursorX / timelineWidth;

      const pointerTime = scale.minTime + scale.duration * ratio;
      const nextDuration =
        event.deltaY > 0 ? scale.duration * ZOOM_OUT_FACTOR : scale.duration * ZOOM_IN_FACTOR;
      const nextCenter = pointerTime + (0.5 - ratio) * nextDuration;

      setViewRange(clampViewByCenter(dataRange, nextCenter, nextDuration));
    },
    [dataRange, scale]
  );

  const isInitialLoading = (isLeadLoading || isTimelineLoading) && rows.length === 0;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="搜索公司、职位、状态、地点..."
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className="pl-9 bg-secondary border-border"
          />
        </div>
        <div className="inline-flex items-center gap-2 rounded-full border border-border bg-secondary/30 px-3 py-1 text-xs text-muted-foreground">
          <CalendarDays className="h-3.5 w-3.5" />
          <span>{rangeLabel}</span>
          <span className="text-[10px] text-muted-foreground/80">滚轮缩放</span>
        </div>
      </div>

      <div
        className="rounded-xl border border-border/70 bg-gradient-to-b from-card to-card/85 shadow-sm overflow-x-auto"
        onWheel={handleWheelZoom}
      >
        <div className="min-w-[1400px]">
          <div className="grid grid-cols-[300px_1fr] border-b border-border/70 bg-secondary/30">
            <div className="sticky left-0 z-20 px-4 py-3 text-xs font-medium text-muted-foreground bg-secondary/40 backdrop-blur">
              线索 / 当前状态
            </div>
            <div className="px-4 py-2">
              <div className="relative h-8 text-[11px] text-muted-foreground">
                {majorTicks.map((tick, index) => {
                  const left = scale
                    ? ((tick.getTime() - scale.minTime) / scale.duration) * 100
                    : 0;
                  return (
                    <div
                      key={`${tick.toISOString()}_${index}`}
                      className="absolute top-0 -translate-x-1/2"
                      style={{ left: `${left}%` }}
                    >
                      <span className="whitespace-nowrap font-medium text-foreground/80">
                        {formatTickLabel(tick, majorStep)}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>

          {isInitialLoading ? (
            <div className="px-4 py-12 text-center text-sm text-muted-foreground">
              正在加载时间线...
            </div>
          ) : null}

          {!isInitialLoading && rows.length === 0 ? (
            <div className="px-4 py-12 text-center text-sm text-muted-foreground">
              暂无可展示的线索阶段数据。
            </div>
          ) : null}

          {rows.map((row) => {
            const visibleSegments: VisibleStageSegment[] = scale
              ? row.segments
                  .map((segment) => {
                    const visibleStart = Math.max(segment.start.getTime(), scale.minTime);
                    const visibleEnd = Math.min(segment.end.getTime(), scale.maxTime);
                    return {
                      ...segment,
                      visibleStart,
                      visibleEnd,
                    };
                  })
                  .filter((segment) => segment.visibleEnd > segment.visibleStart)
              : [];

            const laneCount = Math.max(visibleSegments.length, 1);
            const laneHeight = laneCount * ROW_LANE_HEIGHT + 10;

            return (
              <div
                key={row.lead.id}
                className="grid grid-cols-[300px_1fr] border-b border-border/60 last:border-b-0"
              >
                <div className="sticky left-0 z-10 px-4 py-3 bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/80">
                  <div className="text-sm font-semibold text-foreground">{row.lead.company}</div>
                  <div className="text-xs text-muted-foreground mt-0.5">{row.lead.position}</div>
                  <div className="mt-1 inline-flex rounded-full bg-secondary px-2 py-0.5 text-[11px] text-muted-foreground">
                    当前状态：{STATUS_CONFIG[row.lead.status].label}
                  </div>
                </div>

                <div className="px-4 py-3">
                  <div
                    className="relative rounded-lg border border-border/70 bg-secondary/15"
                    style={{ height: `${laneHeight}px` }}
                  >
                    {minorTicks.map((tick, index) => {
                      const left = scale
                        ? ((tick.getTime() - scale.minTime) / scale.duration) * 100
                        : 0;
                      return (
                        <div
                          key={`${row.lead.id}_minor_${tick.toISOString()}_${index}`}
                          className="absolute inset-y-0 w-px bg-border/35"
                          style={{ left: `${left}%` }}
                        />
                      );
                    })}

                    {majorTicks.map((tick, index) => {
                      const left = scale
                        ? ((tick.getTime() - scale.minTime) / scale.duration) * 100
                        : 0;
                      return (
                        <div
                          key={`${row.lead.id}_major_${tick.toISOString()}_${index}`}
                          className="absolute inset-y-0 w-px bg-border/70"
                          style={{ left: `${left}%` }}
                        />
                      );
                    })}

                    {visibleSegments.length === 0 ? (
                      <div className="absolute inset-0 flex items-center px-3 text-xs text-muted-foreground">
                        当前窗口无阶段条，滚轮缩小可查看更多。
                      </div>
                    ) : null}

                    {visibleSegments.map((segment, index) => {
                      const startPercent = scale
                        ? ((segment.visibleStart - scale.minTime) / scale.duration) * 100
                        : 0;
                      const endPercent = scale
                        ? ((segment.visibleEnd - scale.minTime) / scale.duration) * 100
                        : 0;
                      const widthPercent = Math.max(endPercent - startPercent, 1.2);

                      return (
                        <div
                          key={`${row.lead.id}_${segment.stage}_${segment.start.toISOString()}_${index}`}
                          className={cn(
                            "absolute h-5 rounded-md px-2 text-[11px] font-medium text-white shadow-sm ring-1 ring-white/15 flex items-center justify-between gap-2",
                            getStageColor(segment.stage)
                          )}
                          style={{
                            top: `${4 + index * ROW_LANE_HEIGHT}px`,
                            left: `${startPercent}%`,
                            width: `${widthPercent}%`,
                          }}
                          title={`${getStageLabel(segment.stage)}: ${segment.start.toLocaleString()} - ${segment.end.toLocaleString()}`}
                        >
                          <span className="truncate">{getStageLabel(segment.stage)}</span>
                          <span className="shrink-0 text-[10px] text-white/85">
                            {formatShortDate(segment.start)}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
