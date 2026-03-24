import Link from "next/link";
import { DiscoveryQuickstartContent } from "@/components/discovery-quickstart-content";
import { Button } from "@/components/ui/button";

export default function DiscoveryRulesDocsPage() {
  return (
    <div className="bg-background">
      <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
        <div className="space-y-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <h1 className="text-2xl font-semibold text-foreground">发现规则快速上手</h1>
              <p className="text-sm text-muted-foreground">
                给第一次用发现规则的人看的说明页。先跑起来，再慢慢调，不用一上来就把自己绕晕。
              </p>
            </div>
            <Button asChild variant="outline">
              <Link href="/">返回看板</Link>
            </Button>
          </div>

          <div className="rounded-2xl border border-border bg-card/30 p-5 sm:p-6">
            <DiscoveryQuickstartContent />
          </div>
        </div>
      </main>
    </div>
  );
}
