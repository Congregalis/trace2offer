"use client";

import { useLeadsStore } from "@/lib/leads-store";
import { cn } from "@/lib/utils";

export function StatsCards() {
  const { leads } = useLeadsStore();

  const stats = [
    {
      label: "总线索",
      value: leads.length,
      color: "text-foreground",
    },
    {
      label: "面试中",
      value: leads.filter((l) => l.status === "interviewing").length,
      color: "text-chart-2",
    },
    {
      label: "已投递",
      value: leads.filter((l) => l.status === "applied").length,
      color: "text-warning",
    },
    {
      label: "已获offer",
      value: leads.filter((l) => l.status === "offered").length,
      color: "text-success",
    },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
      {stats.map((stat) => (
        <div
          key={stat.label}
          className="bg-card border border-border rounded-lg p-4"
        >
          <p className="text-sm text-muted-foreground">{stat.label}</p>
          <p className={cn("text-2xl font-semibold mt-1", stat.color)}>
            {stat.value}
          </p>
        </div>
      ))}
    </div>
  );
}
