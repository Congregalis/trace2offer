"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { LayoutGrid, MessageSquare } from "lucide-react";
import { ThemeToggle } from "@/components/theme-toggle";

const navItems = [
  { href: "/", label: "看板", icon: LayoutGrid },
  { href: "/agent", label: "Agent", icon: MessageSquare },
];

export function Nav() {
  const pathname = usePathname();

  return (
    <header className="sticky top-0 z-50 px-4 pt-4 sm:px-6">
      <div className="mx-auto max-w-7xl">
        <div className="page-enter relative overflow-hidden rounded-[28px] border border-[var(--panel-border)] bg-card/72 px-4 py-4 shadow-[var(--panel-shadow)] backdrop-blur-xl sm:px-5">
          <div className="pointer-events-none absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-primary/45 to-transparent" />
          <div className="relative flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:gap-8">
              <Link href="/" className="group flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-primary via-warning to-info text-sm font-black text-primary-foreground shadow-[var(--panel-shadow-strong)] transition-transform duration-500 ease-[var(--ease-fluid)] group-hover:-translate-y-0.5">
                  T2
                </div>
                <div className="min-w-0">
                  <span className="block text-sm font-semibold tracking-[0.18em] text-foreground/85 uppercase">
                    Trace2Offer
                  </span>
                  <span className="block truncate text-xs text-muted-foreground">
                    线索、候选、提醒和 Agent 收在一张桌上
                  </span>
                </div>
              </Link>

              <nav className="grid grid-cols-2 gap-1 rounded-full border border-border/70 bg-background/70 p-1 sm:inline-flex sm:w-auto">
              {navItems.map((item) => {
                const isActive = pathname === item.href;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={cn(
                      "inline-flex min-w-[112px] items-center justify-center gap-2 rounded-full px-4 py-2 text-sm font-medium transition-all duration-300 ease-[var(--ease-fluid)]",
                      isActive
                        ? "bg-foreground text-background shadow-[var(--panel-shadow-strong)]"
                        : "text-muted-foreground hover:bg-accent/75 hover:text-foreground"
                    )}
                  >
                    <item.icon className="w-4 h-4" />
                    {item.label}
                  </Link>
                );
              })}
              </nav>
            </div>

            <div className="flex items-center justify-end">
              <ThemeToggle />
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}
