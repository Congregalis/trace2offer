"use client";

import { DiscoveryRule } from "@/lib/types";
import {
  DISCOVERY_PRESETS,
  DiscoveryPreset,
  DiscoveryPresetGroup,
  getDiscoveryPresetsByGroup,
  hasMatchingRule,
} from "@/lib/discovery-presets";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

const GROUP_LABELS: Record<DiscoveryPresetGroup, { title: string; description: string }> = {
  priority: {
    title: "优先推荐",
    description: "更贴近 Software Engineer / Agent / AI Infra，适合先加这几条。",
  },
  general: {
    title: "通用补充",
    description: "用于补充更宽的软件工程远程岗位，别一上来全加满。",
  },
};

interface DiscoveryPresetCardsProps {
  rules: DiscoveryRule[];
  onAddPreset: (preset: DiscoveryPreset) => void | Promise<void>;
  onOpenHelp?: () => void;
  isBusy?: boolean;
  groups?: DiscoveryPresetGroup[];
  className?: string;
  title?: string;
  description?: string;
}

function renderKeywordPreview(values: string[]): string {
  if (values.length === 0) {
    return "-";
  }
  return values.slice(0, 4).join(", ");
}

function PresetGroup({
  group,
  rules,
  onAddPreset,
  isBusy,
}: {
  group: DiscoveryPresetGroup;
  rules: DiscoveryRule[];
  onAddPreset: (preset: DiscoveryPreset) => void | Promise<void>;
  isBusy?: boolean;
}) {
  const presets = getDiscoveryPresetsByGroup(group);
  if (presets.length === 0) {
    return null;
  }

  const meta = GROUP_LABELS[group];

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">{meta.title}</div>
        <p className="text-xs text-muted-foreground">{meta.description}</p>
      </div>

      <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
        {presets.map((preset) => {
          const added = hasMatchingRule(preset, rules);

          return (
            <Card key={preset.id} className={cn("gap-4 border-border/80 bg-card/50 py-4 shadow-none")}>
              <CardHeader className="px-4">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div className="space-y-1">
                    <CardTitle className="text-sm font-semibold">{preset.name}</CardTitle>
                    <CardDescription className="text-xs">{preset.summary}</CardDescription>
                  </div>
                  <div className="flex flex-wrap gap-1">
                    {preset.tags.map((tag) => (
                      <Badge key={tag} variant="secondary" className="text-[11px]">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>
              </CardHeader>

              <CardContent className="space-y-3 px-4">
                <div className="space-y-1 text-xs text-muted-foreground">
                  <div>
                    <span className="font-medium text-foreground">来源：</span>
                    {preset.rule.source}
                  </div>
                  <div>
                    <span className="font-medium text-foreground">地区：</span>
                    {preset.rule.defaultLocation || "Remote"}
                  </div>
                  <div>
                    <span className="font-medium text-foreground">包含：</span>
                    {renderKeywordPreview(preset.rule.includeKeywords)}
                  </div>
                  <div>
                    <span className="font-medium text-foreground">排除：</span>
                    {renderKeywordPreview(preset.rule.excludeKeywords)}
                  </div>
                </div>

                <Button
                  className="w-full"
                  size="sm"
                  variant={added ? "secondary" : "default"}
                  disabled={added || isBusy}
                  onClick={() => void onAddPreset(preset)}
                >
                  {added ? "已添加" : "一键添加"}
                </Button>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}

export function DiscoveryPresetCards({
  rules,
  onAddPreset,
  onOpenHelp,
  isBusy = false,
  groups = ["priority", "general"],
  className,
  title = "推荐示例规则",
  description = "先加一条能跑起来的规则，比盯着空表单发呆强多了。",
}: DiscoveryPresetCardsProps) {
  if (DISCOVERY_PRESETS.length === 0) {
    return null;
  }

  return (
    <div className={cn("space-y-4 rounded-xl border border-border bg-card/30 p-4", className)}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          <div className="text-sm font-medium text-foreground">{title}</div>
          <p className="text-xs text-muted-foreground">{description}</p>
        </div>
        {onOpenHelp ? (
          <Button variant="link" size="sm" className="h-auto px-0 text-xs" onClick={onOpenHelp}>
            不会填？看快速上手
          </Button>
        ) : null}
      </div>

      {groups.includes("priority") ? <PresetGroup group="priority" rules={rules} onAddPreset={onAddPreset} isBusy={isBusy} /> : null}
      {groups.includes("general") ? <PresetGroup group="general" rules={rules} onAddPreset={onAddPreset} isBusy={isBusy} /> : null}
    </div>
  );
}
