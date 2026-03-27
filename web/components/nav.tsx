"use client";

import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { useEffect, useRef, useState, type MouseEvent } from "react";
import { cn } from "@/lib/utils";
import { LayoutGrid, MessageSquare } from "lucide-react";
import { ThemeToggle } from "@/components/theme-toggle";
import { ReminderCenter } from "@/components/reminder-center";

const navItems = [
  { href: "/", label: "看板", icon: LayoutGrid },
  { href: "/agent", label: "Agent", icon: MessageSquare },
];

type ViewTransitionCapableDocument = Document & {
  startViewTransition?: (update: () => void | Promise<void>) => {
    finished: Promise<void>;
    ready: Promise<void>;
    updateCallbackDone: Promise<void>;
    skipTransition?: () => void;
  };
};

export function Nav() {
  const router = useRouter();
  const pathname = usePathname();
  const [activePath, setActivePath] = useState(pathname);
  const headerRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    setActivePath(pathname);
  }, [pathname]);

  useEffect(() => {
    const root = document.documentElement;
    const updateNavHeight = () => {
      const height = headerRef.current?.offsetHeight || 0;
      if (height > 0) {
        root.style.setProperty("--app-nav-height", `${height}px`);
      }
    };

    updateNavHeight();
    const rafID = window.requestAnimationFrame(updateNavHeight);
    window.addEventListener("resize", updateNavHeight);

    let observer: ResizeObserver | null = null;
    if (typeof ResizeObserver !== "undefined" && headerRef.current) {
      observer = new ResizeObserver(() => {
        updateNavHeight();
      });
      observer.observe(headerRef.current);
    }

    return () => {
      window.cancelAnimationFrame(rafID);
      window.removeEventListener("resize", updateNavHeight);
      observer?.disconnect();
    };
  }, []);

  const navigateWithTransition = (event: MouseEvent<HTMLAnchorElement>, href: string) => {
    if (
      event.defaultPrevented ||
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey ||
      href === pathname
    ) {
      return;
    }

    event.preventDefault();
    setActivePath(href);

    const viewTransitionDocument = document as ViewTransitionCapableDocument;
    if (typeof viewTransitionDocument.startViewTransition === "function") {
      viewTransitionDocument.startViewTransition(() => {
        router.push(href);
      });
      return;
    }

    router.push(href);
  };

  const activeNavIndex = navItems.findIndex((item) => {
    if (item.href === "/") {
      return activePath === "/";
    }
    return activePath.startsWith(item.href);
  });

  return (
    <header ref={headerRef} className="sticky top-0 z-50 px-4 pt-4 sm:px-6">
      <div className="mx-auto max-w-7xl">
        <div className="page-enter relative overflow-hidden rounded-[28px] border border-[var(--panel-border)] bg-card/72 px-4 py-4 shadow-[var(--panel-shadow)] backdrop-blur-xl sm:px-5">
          <div className="pointer-events-none absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-primary/45 to-transparent" />
          <div className="relative flex flex-col gap-4">
            <div className="flex items-start justify-between gap-3">
              <Link href="/" className="group flex min-w-0 items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-primary via-warning to-info text-[13px] font-black tracking-tight text-primary-foreground shadow-[var(--panel-shadow-strong)] transition-transform duration-500 ease-[var(--ease-fluid)] group-hover:-translate-y-0.5">
                  T2O
                </div>
                <div className="min-w-0">
                  <span className="block text-sm font-semibold tracking-[0.18em] text-foreground/85 uppercase">
                    Trace2Offer
                  </span>
                  <span className="hidden truncate text-xs text-muted-foreground sm:block">
                    从线索追踪到拿下 Offer
                  </span>
                </div>
              </Link>
              <div className="flex shrink-0 items-center gap-2">
                <ReminderCenter
                  mode="icon"
                  className="size-8 border border-border/70 bg-background/65 text-muted-foreground shadow-[var(--panel-shadow)] hover:bg-accent/75 hover:text-foreground"
                />
                <ThemeToggle />
              </div>
            </div>

            <nav className="relative isolate grid grid-cols-2 gap-1 rounded-full border border-border/70 bg-background/70 p-1 sm:inline-grid">
              {activeNavIndex >= 0 ? (
                <span
                  aria-hidden
                  className="pointer-events-none absolute inset-y-1 left-1 w-[calc((100%-0.5rem)/2)] rounded-full bg-foreground shadow-[var(--panel-shadow-strong)] transition-transform duration-300 ease-[var(--ease-fluid)]"
                  style={{ transform: `translateX(${activeNavIndex * 100}%)` }}
                />
              ) : null}
              {navItems.map((item) => {
                const isActive =
                  item.href === "/" ? activePath === "/" : activePath.startsWith(item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={(event) => navigateWithTransition(event, item.href)}
                    className={cn(
                      "relative z-10 inline-flex min-w-[112px] items-center justify-center gap-2 rounded-full px-4 py-2 text-sm font-medium transition-colors duration-300 ease-[var(--ease-fluid)]",
                      isActive
                        ? "text-background"
                        : "text-muted-foreground hover:text-foreground"
                    )}
                  >
                    <item.icon className="w-4 h-4" />
                    {item.label}
                  </Link>
                );
              })}
            </nav>
          </div>
        </div>
      </div>
    </header>
  );
}
