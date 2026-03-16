import { Nav } from "@/components/nav";
import { LeadsTable } from "@/components/leads-table";
import { ReminderCenter } from "@/components/reminder-center";
import { StatsCards } from "@/components/stats-cards";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export default function HomePage() {
  return (
    <div className="min-h-screen bg-background">
      <Nav />
      <main className="max-w-7xl mx-auto px-4 sm:px-6 py-8">
        <div className="space-y-6">
          <div className="flex items-start justify-between gap-4">
            <div>
              <h1 className="text-2xl font-semibold text-foreground">线索看板</h1>
              <p className="text-muted-foreground mt-1">
                集中管理你的求职线索和跟进状态
              </p>
            </div>
            <ReminderCenter mode="icon" />
          </div>

          <Tabs defaultValue="leads" className="space-y-4">
            <TabsList className="h-10">
              <TabsTrigger value="leads" className="px-4">
                线索管理
              </TabsTrigger>
              <TabsTrigger value="dashboard" className="px-4">
                核心统计仪表盘
              </TabsTrigger>
            </TabsList>

            <TabsContent value="leads">
              <LeadsTable />
            </TabsContent>

            <TabsContent value="dashboard">
              <StatsCards />
            </TabsContent>
          </Tabs>
        </div>
      </main>
    </div>
  );
}
