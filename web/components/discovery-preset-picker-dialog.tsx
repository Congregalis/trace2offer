"use client";

import { DiscoveryPreset, DiscoveryPresetGroup } from "@/lib/discovery-presets";
import { DiscoveryRule } from "@/lib/types";
import { DiscoveryPresetCards } from "@/components/discovery-preset-cards";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";

const PRESET_GROUP_COPY: Record<
  DiscoveryPresetGroup,
  {
    title: string;
    description: string;
    cardTitle: string;
    cardDescription: string;
  }
> = {
  priority: {
    title: "快速开始",
    description: "优先给第一次配置发现规则的人用，先把最相关的岗位源跑起来。",
    cardTitle: "优先推荐规则",
    cardDescription: "更贴近 Software Engineer / Agent / AI Infra，先从这里开始最稳。",
  },
  general: {
    title: "通用补充",
    description: "在基础规则跑顺后，再补这些更宽的软件工程远程岗位来源。",
    cardTitle: "通用补充规则",
    cardDescription: "作为候选池扩容用，别一上来全加满。",
  },
};

export function DiscoveryPresetPickerDialog({
  open,
  onOpenChange,
  group,
  rules,
  onAddPreset,
  isBusy = false,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  group: DiscoveryPresetGroup;
  rules: DiscoveryRule[];
  onAddPreset: (preset: DiscoveryPreset) => void | Promise<void>;
  isBusy?: boolean;
}) {
  const copy = PRESET_GROUP_COPY[group];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100vh-2rem)] max-w-5xl overflow-hidden p-0">
        <div className="max-h-[calc(100vh-2rem)] overflow-y-auto p-6">
          <DialogHeader>
            <DialogTitle>{copy.title}</DialogTitle>
            <DialogDescription>{copy.description}</DialogDescription>
          </DialogHeader>

          <div className="mt-4">
            <DiscoveryPresetCards
              rules={rules}
              onAddPreset={onAddPreset}
              isBusy={isBusy}
              groups={[group]}
              title={copy.cardTitle}
              description={copy.cardDescription}
            />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
