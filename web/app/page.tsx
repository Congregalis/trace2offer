import { Nav } from "@/components/nav";
import { LeadsTable } from "@/components/leads-table";
import { StatsCards } from "@/components/stats-cards";

export default function HomePage() {
  return (
    <div className="min-h-screen bg-background">
      <Nav />
      <main className="max-w-7xl mx-auto px-4 sm:px-6 py-8">
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">线索看板</h1>
            <p className="text-muted-foreground mt-1">
              集中管理你的求职线索和跟进状态
            </p>
          </div>
          
          <StatsCards />
          
          <LeadsTable />
        </div>
      </main>
    </div>
  );
}
