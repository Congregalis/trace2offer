import { Nav } from "@/components/nav";
import { LeadsTable } from "@/components/leads-table";
import { CandidatesTable } from "@/components/candidates-table";
import { LeadTimelineBoard } from "@/components/lead-timeline-board";
import { ReminderCenter } from "@/components/reminder-center";
import { StatsCards } from "@/components/stats-cards";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export default function HomePage() {
  return (
    <div className="min-h-screen">
      <Nav />
      <main className="mx-auto max-w-7xl px-4 pb-12 pt-6 sm:px-6">
        <div className="space-y-6">
          <section className="page-enter relative overflow-hidden rounded-[32px] border border-[var(--panel-border)] bg-card/74 px-6 py-6 shadow-[var(--panel-shadow)] backdrop-blur-xl sm:px-8 sm:py-8">
            <div className="pointer-events-none absolute -right-12 top-0 h-44 w-44 rounded-full bg-[var(--hero-glow)] blur-3xl" />
            <div className="pointer-events-none absolute -left-12 bottom-0 h-36 w-36 rounded-full bg-[var(--hero-glow-secondary)] blur-3xl" />
            <div className="relative flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
              <div className="max-w-2xl space-y-4">
                <span className="inline-flex items-center rounded-full border border-border/70 bg-background/75 px-3 py-1 text-[11px] font-semibold tracking-[0.28em] text-muted-foreground uppercase">
                  Job Signal Desk
                </span>
                <div className="space-y-3">
                  <h1 className="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
                    线索看板
                  </h1>
                  <p className="max-w-xl text-sm leading-6 text-muted-foreground sm:text-base">
                    白天用亮色把层次拉开，晚上自动收敛成暗色；候选池、线索、提醒和时间线还都老老实实待在一个台面上。
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  {["唯一事实源", "提醒闭环", "候选池联动", "时间线追踪"].map((item) => (
                    <span
                      key={item}
                      className="inline-flex items-center rounded-full border border-border/70 bg-background/70 px-3 py-1 text-xs font-medium text-foreground/85"
                    >
                      {item}
                    </span>
                  ))}
                </div>
              </div>

              <div className="flex w-full flex-col gap-3 lg:max-w-sm">
                <div className="flex items-start justify-between gap-3 rounded-[24px] border border-border/70 bg-background/70 p-4 shadow-[var(--panel-shadow)]">
                  <div className="space-y-1">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
                      Theme Shift
                    </p>
                    <p className="text-sm font-medium text-foreground">自动跟系统时间切换</p>
                    <p className="text-xs leading-5 text-muted-foreground">
                      白天亮、夜晚暗，手动切了也会记住你的选择。
                    </p>
                  </div>
                  <ReminderCenter
                    mode="icon"
                    className="h-11 w-11 rounded-2xl border border-border/70 bg-background/70 shadow-[var(--panel-shadow)]"
                  />
                </div>

                <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
                  <div className="rounded-[24px] border border-border/70 bg-background/65 p-4">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
                      Board
                    </p>
                    <p className="mt-1 text-base font-semibold text-foreground">候选、线索双视角</p>
                    <p className="mt-1 text-xs leading-5 text-muted-foreground">表格管理和时间线共用一套语义色，不会一页一个脾气。</p>
                  </div>
                  <div className="rounded-[24px] border border-border/70 bg-background/65 p-4">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
                      Agent
                    </p>
                    <p className="mt-1 text-base font-semibold text-foreground">对话页同步换肤</p>
                    <p className="mt-1 text-xs leading-5 text-muted-foreground">切到 Agent 也不是另一套皮，整体视觉和交互节奏统一。</p>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section className="page-enter rounded-[32px] border border-[var(--panel-border)] bg-card/72 p-4 shadow-[var(--panel-shadow)] backdrop-blur-xl sm:p-6">
            <Tabs defaultValue="leads" className="space-y-4">
              <TabsList className="h-11 rounded-full border border-border/70 bg-background/70 p-1">
                <TabsTrigger value="leads" className="rounded-full px-4">
                  线索管理
                </TabsTrigger>
                <TabsTrigger value="dashboard" className="rounded-full px-4">
                  核心统计仪表盘
                </TabsTrigger>
              </TabsList>

              <TabsContent value="leads">
                <Tabs defaultValue="table" className="space-y-3">
                  <TabsList className="h-9 rounded-full bg-transparent p-0">
                    <TabsTrigger
                      value="candidates"
                      className="h-8 rounded-full px-3 text-xs font-normal text-muted-foreground data-[state=active]:bg-background/90 data-[state=active]:text-foreground data-[state=active]:shadow-sm dark:data-[state=active]:bg-muted/50"
                    >
                      候选池
                    </TabsTrigger>
                    <TabsTrigger
                      value="table"
                      className="h-8 rounded-full px-3 text-xs font-normal text-muted-foreground data-[state=active]:bg-background/90 data-[state=active]:text-foreground data-[state=active]:shadow-sm dark:data-[state=active]:bg-muted/50"
                    >
                      表格管理
                    </TabsTrigger>
                    <TabsTrigger
                      value="timeline"
                      className="h-8 rounded-full px-3 text-xs font-normal text-muted-foreground data-[state=active]:bg-background/90 data-[state=active]:text-foreground data-[state=active]:shadow-sm dark:data-[state=active]:bg-muted/50"
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
