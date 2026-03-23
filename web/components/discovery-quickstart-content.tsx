"use client";

import { getDiscoveryPresetsByGroup } from "@/lib/discovery-presets";
import { Badge } from "@/components/ui/badge";

export function DiscoveryQuickstartContent() {
  const priorityPresets = getDiscoveryPresetsByGroup("priority");
  const generalPresets = getDiscoveryPresetsByGroup("general");
  const starterKeywords = [
    "software engineer",
    "backend",
    "platform",
    "ai",
    "llm",
    "agent",
    "inference",
    "rag",
    "golang",
    "python",
    "distributed systems",
  ];
  const excludeKeywords = ["intern", "frontend", "ios", "android", "sales", "marketing", "recruiter", "wordpress"];

  return (
    <div className="space-y-8 text-sm">
      <section className="space-y-2">
        <h2 className="text-xl font-semibold text-foreground">发现规则快速上手</h2>
        <p className="text-muted-foreground">
          发现规则就是告诉系统去哪个 RSS/Atom 源抓岗位，再用包含词和排除词决定哪些岗位进候选池。
        </p>
      </section>

      <section className="space-y-3">
        <h3 className="text-base font-medium text-foreground">先这么开始就够了</h3>
        <ul className="list-disc space-y-2 pl-5 text-muted-foreground">
          <li>先添加 2 到 3 条推荐示例规则。</li>
          <li>点一次“立即发现”。</li>
          <li>看候选池质量，再调关键词。</li>
        </ul>
      </section>

      <section className="space-y-4">
        <h3 className="text-base font-medium text-foreground">推荐先用的规则</h3>
        <div className="grid gap-4 md:grid-cols-2">
          <div className="rounded-xl border border-border bg-card/30 p-4">
            <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">优先推荐</div>
            <div className="mt-3 space-y-3 text-muted-foreground">
              {priorityPresets.map((preset) => (
                <div key={preset.id}>
                  <span className="font-medium text-foreground">{preset.name}</span>
                  <span>：{preset.summary}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-xl border border-border bg-card/30 p-4">
            <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">通用补充</div>
            <div className="mt-3 space-y-3 text-muted-foreground">
              {generalPresets.map((preset) => (
                <div key={preset.id}>
                  <span className="font-medium text-foreground">{preset.name}</span>
                  <span>：{preset.summary}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="space-y-4">
        <h3 className="text-base font-medium text-foreground">每个字段怎么理解</h3>
        <div className="grid gap-3 md:grid-cols-2">
          <div className="rounded-xl border border-border bg-card/30 p-4">
            <div className="font-medium text-foreground">规则名</div>
            <p className="mt-1 text-muted-foreground">给自己看的名字，写人话，别写成 `rule-1`。</p>
          </div>
          <div className="rounded-xl border border-border bg-card/30 p-4">
            <div className="font-medium text-foreground">RSS/Atom URL</div>
            <p className="mt-1 text-muted-foreground">必须是真 feed，不是普通 jobs 页面。</p>
          </div>
          <div className="rounded-xl border border-border bg-card/30 p-4">
            <div className="font-medium text-foreground">来源标签</div>
            <p className="mt-1 text-muted-foreground">最终会进候选池 `source`，建议统一格式，例如 `remoteyeah:ai`。</p>
          </div>
          <div className="rounded-xl border border-border bg-card/30 p-4">
            <div className="font-medium text-foreground">默认地区</div>
            <p className="mt-1 text-muted-foreground">feed 没写地区时的兜底值，例如 `Remote`。</p>
          </div>
          <div className="rounded-xl border border-border bg-card/30 p-4">
            <div className="font-medium text-foreground">包含关键词</div>
            <p className="mt-1 text-muted-foreground">标题或摘要命中任意一个就保留。</p>
          </div>
          <div className="rounded-xl border border-border bg-card/30 p-4">
            <div className="font-medium text-foreground">排除关键词</div>
            <p className="mt-1 text-muted-foreground">标题或摘要命中任意一个就踢掉。</p>
          </div>
        </div>
      </section>

      <section className="space-y-4">
        <h3 className="text-base font-medium text-foreground">推荐关键词起步模板</h3>
        <div className="space-y-4 rounded-xl border border-border bg-card/30 p-4">
          <div>
            <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">包含关键词</div>
            <div className="mt-2 flex flex-wrap gap-2">
              {starterKeywords.map((keyword) => (
                <Badge key={keyword} variant="secondary">
                  {keyword}
                </Badge>
              ))}
            </div>
          </div>
          <div>
            <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">排除关键词</div>
            <div className="mt-2 flex flex-wrap gap-2">
              {excludeKeywords.map((keyword) => (
                <Badge key={keyword} variant="secondary">
                  {keyword}
                </Badge>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-base font-medium text-foreground">别踩这些坑</h3>
        <ul className="list-disc space-y-2 pl-5 text-muted-foreground">
          <li>普通职位列表页不是 feed，填进去大概率白给。</li>
          <li>关键词写太宽，候选池就会变垃圾桶。</li>
          <li>示例规则不用一次全加，先跑 2 到 3 条最合适。</li>
          <li>重复添加同一来源没有意义，只会制造噪音。</li>
        </ul>
      </section>
    </div>
  );
}
