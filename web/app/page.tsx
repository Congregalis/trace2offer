"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { LeadsTable } from "@/components/leads-table";
import { CandidatesTable } from "@/components/candidates-table";
import { LeadTimelineBoard } from "@/components/lead-timeline-board";
import { StatsCards } from "@/components/stats-cards";
import { PrepLibraryWorkspace } from "@/components/prep/prep-library-workspace";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const MAIN_TABS = ["leads", "library", "dashboard"] as const;
type MainTabValue = (typeof MAIN_TABS)[number];
const LEAD_VIEW_TABS = ["candidates", "table", "timeline"] as const;
type LeadViewTabValue = (typeof LEAD_VIEW_TABS)[number];

export default function HomePage() {
  const [activeMainTab, setActiveMainTab] = useState<MainTabValue>("leads");
  const [activeLeadViewTab, setActiveLeadViewTab] = useState<LeadViewTabValue>("table");
  const activeMainTabIndex = MAIN_TABS.indexOf(activeMainTab);
  const leadViewTriggerRefs = useRef<Partial<Record<LeadViewTabValue, HTMLButtonElement | null>>>({});
  const [leadViewIndicator, setLeadViewIndicator] = useState({ x: 0, width: 0 });

  const updateLeadViewIndicator = useCallback(() => {
    const activeNode = leadViewTriggerRefs.current[activeLeadViewTab];
    if (!activeNode) {
      return;
    }
    setLeadViewIndicator({
      x: activeNode.offsetLeft,
      width: activeNode.offsetWidth,
    });
  }, [activeLeadViewTab]);

  useEffect(() => {
    updateLeadViewIndicator();
    const frameID = window.requestAnimationFrame(updateLeadViewIndicator);
    window.addEventListener("resize", updateLeadViewIndicator);
    return () => {
      window.cancelAnimationFrame(frameID);
      window.removeEventListener("resize", updateLeadViewIndicator);
    };
  }, [updateLeadViewIndicator]);

  return (
    <div>
      <main className="mx-auto max-w-7xl px-4 pb-12 pt-6 sm:px-6">
        <div className="space-y-6">
          <section className="page-enter rounded-[32px] border border-[var(--panel-border)] bg-card/72 p-4 shadow-[var(--panel-shadow)] backdrop-blur-xl sm:p-6">
            <Tabs
              value={activeMainTab}
              onValueChange={(value) => setActiveMainTab(value as MainTabValue)}
              className="space-y-4"
            >
              <TabsList className="relative !grid h-11 !w-full grid-cols-3 rounded-full border border-border/70 bg-background/70 p-1">
                <span
                  aria-hidden
                  className="pointer-events-none absolute inset-y-1 left-1 w-[calc((100%-0.5rem)/3)] rounded-full bg-background/95 shadow-sm transition-transform duration-300 ease-[var(--ease-fluid)]"
                  style={{ transform: `translateX(${activeMainTabIndex * 100}%)` }}
                />
                <TabsTrigger
                  value="leads"
                  className="relative z-10 rounded-full border-none bg-transparent px-4 text-muted-foreground transition-colors duration-300 ease-[var(--ease-fluid)] data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
                >
                  线索管理
                </TabsTrigger>
                <TabsTrigger
                  value="library"
                  className="relative z-10 rounded-full border-none bg-transparent px-4 text-muted-foreground transition-colors duration-300 ease-[var(--ease-fluid)] data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
                >
                  文档仓库
                </TabsTrigger>
                <TabsTrigger
                  value="dashboard"
                  className="relative z-10 rounded-full border-none bg-transparent px-4 text-muted-foreground transition-colors duration-300 ease-[var(--ease-fluid)] data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
                >
                  核心统计仪表盘
                </TabsTrigger>
              </TabsList>

              <TabsContent value="leads">
                <Tabs
                  value={activeLeadViewTab}
                  onValueChange={(value) => setActiveLeadViewTab(value as LeadViewTabValue)}
                  className="space-y-3"
                >
                  <TabsList className="relative h-9 rounded-full bg-transparent p-0">
                    <span
                      aria-hidden
                      className="pointer-events-none absolute top-0.5 left-0 h-8 rounded-full bg-background/90 shadow-sm transition-[transform,width] duration-300 ease-[var(--ease-fluid)] dark:bg-muted/50"
                      style={{
                        transform: `translateX(${leadViewIndicator.x}px)`,
                        width: `${leadViewIndicator.width}px`,
                      }}
                    />
                    <TabsTrigger
                      ref={(node) => {
                        leadViewTriggerRefs.current.candidates = node;
                      }}
                      value="candidates"
                      className="relative z-10 h-8 rounded-full px-3 text-xs font-normal text-muted-foreground transition-colors duration-300 ease-[var(--ease-fluid)] data-[state=active]:text-foreground"
                    >
                      候选池
                    </TabsTrigger>
                    <TabsTrigger
                      ref={(node) => {
                        leadViewTriggerRefs.current.table = node;
                      }}
                      value="table"
                      className="relative z-10 h-8 rounded-full px-3 text-xs font-normal text-muted-foreground transition-colors duration-300 ease-[var(--ease-fluid)] data-[state=active]:text-foreground"
                    >
                      表格管理
                    </TabsTrigger>
                    <TabsTrigger
                      ref={(node) => {
                        leadViewTriggerRefs.current.timeline = node;
                      }}
                      value="timeline"
                      className="relative z-10 h-8 rounded-full px-3 text-xs font-normal text-muted-foreground transition-colors duration-300 ease-[var(--ease-fluid)] data-[state=active]:text-foreground"
                    >
                      时间线
                    </TabsTrigger>
                  </TabsList>

                  <TabsContent value="candidates">
                    <CandidatesTable />
                  </TabsContent>

                  <TabsContent value="table">
                    <LeadsTable />
                  </TabsContent>

                  <TabsContent value="timeline">
                    <LeadTimelineBoard />
                  </TabsContent>
                </Tabs>
              </TabsContent>

              <TabsContent value="library">
                <PrepLibraryWorkspace />
              </TabsContent>

              <TabsContent value="dashboard">
                <StatsCards />
              </TabsContent>
            </Tabs>
          </section>
        </div>
      </main>
    </div>
  );
}
